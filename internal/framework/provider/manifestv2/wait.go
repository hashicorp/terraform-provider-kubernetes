// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestv2

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/util/jsonpath"
	"k8s.io/kubectl/pkg/polymorphichelpers"
)

type waitModel struct {
	Rollout    types.Bool       `tfsdk:"rollout"`
	Fields     types.Map        `tfsdk:"fields"`
	Conditions []conditionModel `tfsdk:"condition"`
}

type conditionModel struct {
	Type   types.String `tfsdk:"type"`
	Status types.String `tfsdk:"status"`
}

// expectation is one thing the applied object must show before the apply
// counts as done.
type expectation struct {
	description string
	met         func(*unstructured.Unstructured) (bool, error)
}

// expectations reads the wait argument. id may be incomplete while the
// manifest is not known.
func expectations(ctx context.Context, config types.Object, id identity) ([]expectation, error) {
	if config.IsNull() || config.IsUnknown() {
		return nil, nil
	}
	var w waitModel
	if diags := config.As(ctx, &w, basetypes.ObjectAsOptions{UnhandledUnknownAsEmpty: true}); diags.HasError() {
		return nil, fmt.Errorf("%s", diags[0].Detail())
	}
	var expected []expectation
	if w.Rollout.ValueBool() {
		viewer, err := polymorphichelpers.StatusViewerFor(schema.GroupKind{Group: id.group(), Kind: id.Kind})
		if err != nil && id.Kind != "" {
			return nil, fmt.Errorf("rollout supports Deployment, StatefulSet and DaemonSet, not %s", id.Kind)
		}
		expected = append(expected, expectation{"the rollout to complete", func(obj *unstructured.Unstructured) (bool, error) {
			_, done, err := viewer.Status(obj, 0)
			return done, err
		}})
	}
	for _, condition := range w.Conditions {
		kind, status := condition.Type.ValueString(), condition.Status.ValueString()
		expected = append(expected, expectation{fmt.Sprintf("condition %s to be %s", kind, status), func(obj *unstructured.Unstructured) (bool, error) {
			conditions, _, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
			for _, condition := range conditions {
				if c, ok := condition.(map[string]any); ok && c["type"] == kind && fmt.Sprint(c["status"]) == status {
					return true, nil
				}
			}
			return false, nil
		}})
	}
	for _, field := range sortedKeys(w.Fields) {
		pattern := w.Fields.Elements()[field].(types.String).ValueString()
		// kubernetes_manifest writes a map key in a path as ["key"]. JSONPath
		// takes it as a field name with its dots escaped.
		expression := mapKeys.ReplaceAllStringFunc(field, func(key string) string {
			return "." + strings.ReplaceAll(key[2:len(key)-2], ".", `\.`)
		})
		if !strings.HasPrefix(expression, "{") {
			expression = "{." + strings.TrimPrefix(expression, ".") + "}"
		}
		path := jsonpath.New(field).AllowMissingKeys(true)
		if err := path.Parse(expression); err != nil {
			return nil, fmt.Errorf("field path %q: %w", field, err)
		}
		var matcher *regexp.Regexp
		if pattern != "*" {
			var err error
			if matcher, err = regexp.Compile(pattern); err != nil {
				return nil, fmt.Errorf("pattern for field %q: %w", field, err)
			}
		}
		expected = append(expected, expectation{fmt.Sprintf("field %s to match %q", field, pattern), func(obj *unstructured.Unstructured) (bool, error) {
			results, err := path.FindResults(obj.Object)
			if err != nil || len(results) == 0 || len(results[0]) == 0 {
				return false, nil
			}
			text := fmt.Sprint(results[0][0].Interface())
			if matcher == nil {
				return text != "", nil
			}
			return matcher.MatchString(text), nil
		}})
	}
	return expected, nil
}

func sortedKeys(m types.Map) []string {
	keys := make([]string, 0, len(m.Elements()))
	for key, value := range m.Elements() {
		if !value.IsNull() && !value.IsUnknown() {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

// await polls the object until every expectation is met or ctx ends.
func await(ctx context.Context, client dynamic.ResourceInterface, name string, expected []expectation) error {
	if len(expected) == 0 {
		return nil
	}
	pending := ""
	err := wait.PollUntilContextCancel(ctx, 2*time.Second, true, func(ctx context.Context) (bool, error) {
		obj, err := client.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, e := range expected {
			met, err := e.met(obj)
			if err != nil || !met {
				pending = e.description
				return false, err
			}
		}
		return true, nil
	})
	if err != nil && ctx.Err() != nil {
		return fmt.Errorf("timed out waiting for %s", pending)
	}
	return err
}
