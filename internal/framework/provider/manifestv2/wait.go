// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestv2

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/fluxcd/cli-utils/pkg/kstatus/status"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/util/jsonpath"
	"k8s.io/kubectl/pkg/polymorphichelpers"
)

type waitModel struct {
	Rollout    types.Bool       `tfsdk:"rollout"`
	Ready      types.Bool       `tfsdk:"ready"`
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
			if err != nil && done {
				// kubectl has no rollout status for the OnDelete strategy,
				// which updates pods only as they are deleted.
				return podsReady(obj), nil
			}
			return done, err
		}})
	}
	if w.Ready.ValueBool() {
		// The readiness rules of Helm and of tools built on kstatus: per kind
		// for built-in kinds, from the status conditions for any other.
		var first time.Time
		expected = append(expected, expectation{"the object to be ready", func(obj *unstructured.Unstructured) (bool, error) {
			if first.IsZero() {
				first = time.Now()
			}
			result, err := status.Compute(obj)
			switch {
			case err != nil:
				return false, nil
			case result.Status == status.FailedStatus:
				return false, fmt.Errorf("the object failed: %s", result.Message)
			case result.Status != status.CurrentStatus:
				return false, nil
			case status.GetLegacyConditionsFn(obj) != nil:
				return true, nil
			}
			return currentStatus(obj, time.Since(first)), nil
		}})
	}
	for _, condition := range w.Conditions {
		kind, status := condition.Type.ValueString(), condition.Status.ValueString()
		if status == "" {
			status = string(metav1.ConditionTrue)
		}
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
			if err != nil || len(results) == 0 || len(results[0]) == 0 || results[0][0].Interface() == nil {
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

var mapKeys = regexp.MustCompile(`\["([^"]*)"\]`)

// settle is how long a status that does not tell which generation it describes
// is given to change, so that the controller can act on the apply first.
const settle = 10 * time.Second

// currentStatus adds to kstatus for kinds without rules of their own, which it
// reads as ready without a status: a Ready condition must be True, no
// condition may describe an older generation, and a status that names no
// generation counts only after the controller had time to act.
func currentStatus(obj *unstructured.Unstructured, waited time.Duration) bool {
	_, generational, _ := unstructured.NestedInt64(obj.Object, "status", "observedGeneration")
	conditions, _, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	for _, item := range conditions {
		condition, _ := item.(map[string]any)
		if observed, found, _ := unstructured.NestedInt64(condition, "observedGeneration"); found {
			if observed < obj.GetGeneration() {
				return false
			}
			generational = true
		}
		if condition["type"] == "Ready" && condition["status"] != string(metav1.ConditionTrue) {
			return false
		}
	}
	return generational || waited >= settle
}

// podsReady reports whether a StatefulSet or DaemonSet has seen its current
// spec and all its pods are ready.
func podsReady(obj *unstructured.Unstructured) bool {
	observed, _, _ := unstructured.NestedInt64(obj.Object, "status", "observedGeneration")
	if observed < obj.GetGeneration() {
		return false
	}
	if obj.GetKind() == "DaemonSet" {
		desired, _, _ := unstructured.NestedInt64(obj.Object, "status", "desiredNumberScheduled")
		ready, _, _ := unstructured.NestedInt64(obj.Object, "status", "numberReady")
		return ready >= desired
	}
	replicas, found, _ := unstructured.NestedInt64(obj.Object, "spec", "replicas")
	if !found {
		replicas = 1
	}
	ready, _, _ := unstructured.NestedInt64(obj.Object, "status", "readyReplicas")
	return ready >= replicas
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

// unmet is an expectation that failed, such as a rollout past its progress
// deadline: waiting longer does not help.
type unmet struct{ error }

var errDeleted = errors.New("deleted")

const watchSeconds = 30

// await waits until the object meets every expectation or ctx ends, and
// returns the object as it was then. It
// watches the object, so it sees every change, including the last state of an
// object that is deleted, such as a Job that removes itself when it finishes.
// Where the credentials do not allow a watch, it polls. Errors that waiting
// may resolve, such as an API server restart, do not end the wait.
func await(ctx context.Context, client dynamic.ResourceInterface, name string, expected []expectation) (*unstructured.Unstructured, error) {
	if len(expected) == 0 {
		return nil, nil
	}
	var ready *unstructured.Unstructured
	pending := expected[0].description
	met := func(obj *unstructured.Unstructured) (bool, error) {
		for _, e := range expected {
			if ok, err := e.met(obj); err != nil || !ok {
				pending = e.description
				if err != nil {
					return false, unmet{err}
				}
				return false, nil
			}
		}
		ready = obj
		return true, nil
	}
	watching := true
	var last error
	// resourceVersion is where the next watch starts. A watch that ends
	// continues from there, so no change falls between two of them; after an
	// error the object is read again.
	resourceVersion := ""
	var latest *unstructured.Unstructured
	for {
		var done bool
		var err error
		if resourceVersion == "" || !watching {
			if latest, err = client.Get(ctx, name, metav1.GetOptions{}); err == nil {
				resourceVersion = latest.GetResourceVersion()
				done, err = met(latest)
			}
		}
		began := time.Now()
		if err == nil && !done && watching {
			done, resourceVersion, latest, err = watchUntil(ctx, client, name, resourceVersion, latest, met)
			if apierrors.IsForbidden(err) || apierrors.IsMethodNotSupported(err) {
				watching, err = false, nil
			}
		}
		var failed unmet
		switch {
		case done:
			return ready, nil
		case ctx.Err() != nil:
			return nil, stopped(ctx, pending, last)
		case errors.As(err, &failed):
			return nil, failed.error
		case errors.Is(err, errDeleted) || apierrors.IsNotFound(err):
			return nil, fmt.Errorf("the object was deleted while waiting for %s", pending)
		case err != nil && !transient(err):
			return nil, err
		case err != nil:
			last, resourceVersion = err, ""
		case watching && time.Since(began) > time.Second:
			continue
		}
		select {
		case <-ctx.Done():
			return nil, stopped(ctx, pending, last)
		case <-time.After(2 * time.Second):
		}
	}
}

// watchUntil follows the object's changes after resourceVersion until it meets
// the expectations, is deleted, or the watch ends, and returns the resource
// version it got to and the object as last seen. The object is also looked at
// again every few seconds, for expectations that depend on time. A watch lasts
// half a minute, so that a proxy that holds back the events until a response
// ends delays the wait by no more than that.
func watchUntil(ctx context.Context, client dynamic.ResourceInterface, name, resourceVersion string, latest *unstructured.Unstructured, met func(*unstructured.Unstructured) (bool, error)) (bool, string, *unstructured.Unstructured, error) {
	ctx, cancel := context.WithTimeout(ctx, watchSeconds*time.Second+15*time.Second)
	defer cancel()
	seconds := int64(watchSeconds)
	w, err := client.Watch(ctx, metav1.ListOptions{
		FieldSelector:       fields.OneTermEqualSelector("metadata.name", name).String(),
		ResourceVersion:     resourceVersion,
		TimeoutSeconds:      &seconds,
		AllowWatchBookmarks: true,
	})
	if err != nil {
		return false, resourceVersion, latest, err
	}
	defer w.Stop()
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		var event watch.Event
		select {
		case <-tick.C:
			if latest != nil {
				if done, err := met(latest); done || err != nil {
					return done, resourceVersion, latest, err
				}
			}
			continue
		case received, open := <-w.ResultChan():
			if !open {
				return false, resourceVersion, latest, nil
			}
			event = received
		}
		if event.Type == watch.Error {
			return false, resourceVersion, latest, apierrors.FromObject(event.Object)
		}
		obj, ok := event.Object.(*unstructured.Unstructured)
		if !ok {
			continue
		}
		resourceVersion = obj.GetResourceVersion()
		if event.Type == watch.Bookmark {
			continue
		}
		latest = obj
		if done, err := met(obj); done || err != nil {
			return done, resourceVersion, latest, err
		}
		if event.Type == watch.Deleted {
			return false, resourceVersion, latest, errDeleted
		}
	}
}

// stopped explains why a wait ended before its condition held.
func stopped(ctx context.Context, pending string, last error) error {
	reason := "timed out"
	if errors.Is(ctx.Err(), context.Canceled) {
		reason = "interrupted"
	}
	if last != nil {
		return fmt.Errorf("%s waiting for %s; the last error was: %w", reason, pending, last)
	}
	return fmt.Errorf("%s waiting for %s", reason, pending)
}
