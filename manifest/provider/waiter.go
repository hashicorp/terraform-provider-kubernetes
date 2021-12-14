package provider

import (
	"context"
	"fmt"
	"math/big"
	"regexp"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/payload"
	"github.com/zclconf/go-cty/cty"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/kubectl/pkg/polymorphichelpers"
)

const waiterSleepTime = 1 * time.Second

func (s *RawProviderServer) waitForCompletion(ctx context.Context, waitForBlock tftypes.Value, rs dynamic.ResourceInterface, rname string, rtype tftypes.Type, th map[string]string) error {
	if waitForBlock.IsNull() || !waitForBlock.IsKnown() {
		return nil
	}

	waiter, err := NewResourceWaiter(rs, rname, rtype, th, waitForBlock, s.logger)
	if err != nil {
		return err
	}
	return waiter.Wait(ctx)
}

// Waiter is a simple interface to implement a blocking wait operation
type Waiter interface {
	Wait(context.Context) error
}

// NewResourceWaiter constructs an appropriate Waiter using the supplied waitForBlock configuration
func NewResourceWaiter(resource dynamic.ResourceInterface, resourceName string, resourceType tftypes.Type, th map[string]string, waitForBlock tftypes.Value, hl hclog.Logger) (Waiter, error) {
	var waitForBlockVal map[string]tftypes.Value
	err := waitForBlock.As(&waitForBlockVal)
	if err != nil {
		return nil, err
	}

	if v, ok := waitForBlockVal["rollout"]; ok {
		var rollout bool
		v.As(&rollout)
		if rollout {
			return &RolloutWaiter{
				resource,
				resourceName,
				hl,
			}, nil
		}
	}

	fields, ok := waitForBlockVal["fields"]
	if !ok || fields.IsNull() || !fields.IsKnown() {
		return &NoopWaiter{}, nil
	}

	if !fields.Type().Is(tftypes.Map{}) {
		return nil, fmt.Errorf(`"fields" should be a map of strings`)
	}

	var vm map[string]tftypes.Value
	fields.As(&vm)
	var matchers []FieldMatcher

	for k, v := range vm {
		var expr string
		v.As(&expr)
		var re *regexp.Regexp
		if expr == "*" {
			// NOTE this is just a shorthand so the user doesn't have to
			// type the expression below all the time
			re = regexp.MustCompile("(.*)?")
		} else {
			var err error
			re, err = regexp.Compile(expr)
			if err != nil {
				return nil, fmt.Errorf("invalid regular expression: %q", expr)
			}
		}

		p, err := FieldPathToTftypesPath(k)
		if err != nil {
			return nil, err
		}
		matchers = append(matchers, FieldMatcher{p, re})
	}

	return &FieldWaiter{
		resource,
		resourceName,
		resourceType,
		th,
		matchers,
		hl,
	}, nil

}

// FieldMatcher contains a tftypes.AttributePath to a field and a regexp to match on it
type FieldMatcher struct {
	path         *tftypes.AttributePath
	valueMatcher *regexp.Regexp
}

// FieldWaiter will wait for a set of fields to be set,
// or have a particular value
type FieldWaiter struct {
	resource      dynamic.ResourceInterface
	resourceName  string
	resourceType  tftypes.Type
	typeHints     map[string]string
	fieldMatchers []FieldMatcher
	logger        hclog.Logger
}

// Wait blocks until all of the FieldMatchers configured evaluate to true
func (w *FieldWaiter) Wait(ctx context.Context) error {
	w.logger.Info("[ApplyResourceChange][Wait] Waiting until ready...\n")
	for {
		if deadline, ok := ctx.Deadline(); ok {
			if time.Now().After(deadline) {
				return context.DeadlineExceeded
			}
		}

		// NOTE The typed API resource is actually returned in the
		// event object but I haven't yet figured out how to convert it
		// to a cty.Value.
		res, err := w.resource.Get(ctx, w.resourceName, v1.GetOptions{})
		if err != nil {
			return err
		}
		if errors.IsGone(err) {
			return fmt.Errorf("resource was deleted")
		}
		resObj := res.Object
		meta := resObj["metadata"].(map[string]interface{})
		delete(meta, "managedFields")

		w.logger.Trace("[ApplyResourceChange][Wait]", "API Response", resObj)

		obj, err := payload.ToTFValue(resObj, w.resourceType, w.typeHints, tftypes.NewAttributePath())
		if err != nil {
			return err
		}

		done, err := func(obj tftypes.Value) (bool, error) {
			for _, m := range w.fieldMatchers {
				vi, rp, err := tftypes.WalkAttributePath(obj, m.path)
				if err != nil {
					return false, err
				}
				if len(rp.Steps()) > 0 {
					return false, fmt.Errorf("attribute not present at path '%s'", m.path.String())
				}

				var s string
				v := vi.(tftypes.Value)
				switch {
				case v.Type().Is(tftypes.String):
					v.As(&s)
				case v.Type().Is(tftypes.Bool):
					var vb bool
					v.As(&vb)
					s = fmt.Sprintf("%t", vb)
				case v.Type().Is(tftypes.Number):
					var f big.Float
					v.As(&f)
					if f.IsInt() {
						i, _ := f.Int64()
						s = fmt.Sprintf("%d", i)
					} else {
						i, _ := f.Float64()
						s = fmt.Sprintf("%f", i)
					}
				default:
					return true, fmt.Errorf("wait_for: cannot match on type %q", v.Type().String())
				}

				if !m.valueMatcher.Match([]byte(s)) {
					return false, nil
				}
			}

			return true, nil
		}(obj)

		if done {
			w.logger.Info("[ApplyResourceChange][Wait] Done waiting.\n")
			return err
		}

		// TODO: implement with exponential back-off.
		time.Sleep(waiterSleepTime) // lintignore:R018
	}
}

// NoopWaiter is a placeholder for when there is nothing to wait on
type NoopWaiter struct{}

// Wait returns immediately
func (w *NoopWaiter) Wait(_ context.Context) error {
	return nil
}

// FieldPathToTftypesPath takes a string representation of
// a path to a field in dot/square bracket notation
// and returns a tftypes.AttributePath
func FieldPathToTftypesPath(fieldPath string) (*tftypes.AttributePath, error) {
	t, d := hclsyntax.ParseTraversalAbs([]byte(fieldPath), "", hcl.Pos{Line: 1, Column: 1})
	if d.HasErrors() {
		return tftypes.NewAttributePath(), fmt.Errorf("invalid field path %q: %s", fieldPath, d.Error())
	}

	path := tftypes.NewAttributePath()
	for _, p := range t {
		switch p.(type) {
		case hcl.TraverseRoot:
			path = path.WithAttributeName(p.(hcl.TraverseRoot).Name)
		case hcl.TraverseIndex:
			indexKey := p.(hcl.TraverseIndex).Key
			indexKeyType := indexKey.Type()
			if indexKeyType.Equals(cty.String) {
				path = path.WithElementKeyString(indexKey.AsString())
			} else if indexKeyType.Equals(cty.Number) {
				f := indexKey.AsBigFloat()
				if f.IsInt() {
					i, _ := f.Int64()
					path = path.WithElementKeyInt(int(i))
				} else {
					return tftypes.NewAttributePath(), fmt.Errorf("index in field path must be an integer")
				}
			} else {
				return tftypes.NewAttributePath(), fmt.Errorf("unsupported type in field path: %s", indexKeyType.FriendlyName())
			}
		case hcl.TraverseAttr:
			path = path.WithAttributeName(p.(hcl.TraverseAttr).Name)
		case hcl.TraverseSplat:
			return tftypes.NewAttributePath(), fmt.Errorf("splat is not supported")
		}
	}

	return path, nil
}

// RolloutWaiter will wait for a resource that has a StatusViewer to
// finish rolling out
type RolloutWaiter struct {
	resource     dynamic.ResourceInterface
	resourceName string
	logger       hclog.Logger
}

// Wait uses StatusViewer to determine if the rollout is done
func (w *RolloutWaiter) Wait(ctx context.Context) error {
	w.logger.Info("[ApplyResourceChange][Wait] Waiting until rollout complete...\n")
	for {
		if deadline, ok := ctx.Deadline(); ok {
			if time.Now().After(deadline) {
				return context.DeadlineExceeded
			}
		}

		res, err := w.resource.Get(ctx, w.resourceName, v1.GetOptions{})
		if err != nil {
			return err
		}
		if errors.IsGone(err) {
			return fmt.Errorf("resource was deleted")
		}

		gk := res.GetObjectKind().GroupVersionKind().GroupKind()
		statusViewer, err := polymorphichelpers.StatusViewerFor(gk)
		if err != nil {
			return fmt.Errorf("error getting resource status: %v", err)
		}

		_, done, err := statusViewer.Status(res, 0)
		if err != nil {
			return fmt.Errorf("error getting resource status: %v", err)
		}

		if done {
			break
		}

		time.Sleep(waiterSleepTime) // lintignore:R018
	}

	w.logger.Info("[ApplyResourceChange][Wait] Rollout complete\n")
	return nil
}
