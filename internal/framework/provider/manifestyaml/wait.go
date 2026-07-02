// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestyaml

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/util/jsonpath"
)

const defaultWaitTimeout = 5 * time.Minute

// waitTimeout returns the configured wait{} readiness timeout, or the default.
func waitTimeout(w *waitModel) time.Duration {
	if w != nil && !w.Timeout.IsNull() && w.Timeout.ValueString() != "" {
		if d, err := time.ParseDuration(w.Timeout.ValueString()); err == nil {
			return d
		}
	}
	return defaultWaitTimeout
}

// waitForReady blocks until the object satisfies the wait{} block, or the timeout
// elapses. On failure/timeout it appends pod-level diagnostics (CrashLoopBackOff,
// ImagePullBackOff, FailedScheduling, …) so the user learns *why* it didn't become ready.
func (r *ManifestYAML) waitForReady(ctx context.Context, obj *unstructured.Unstructured, w *waitModel) error {
	if w == nil || !w.hasAny() {
		return nil
	}

	timeout := waitTimeout(w)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	gvr, namespaced, err := resolveGVR(r.clients(), obj)
	if err != nil {
		return err
	}
	dyn, err := r.clients().DynamicClient()
	if err != nil {
		return err
	}
	ri := resourceInterface(dyn, gvr, namespaced, obj.GetNamespace())

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		live, getErr := ri.Get(ctx, obj.GetName(), metav1.GetOptions{})
		if getErr == nil {
			ready, rErr := checkReady(ctx, live, w)
			if rErr != nil {
				return r.withPodErrors(live, namespaced, rErr)
			}
			if ready {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return r.withPodErrors(obj, namespaced,
				fmt.Errorf("timed out after %s waiting for readiness", timeout))
		case <-ticker.C:
		}
	}
}

// checkReady evaluates all configured wait conditions; all must be satisfied.
func checkReady(ctx context.Context, u *unstructured.Unstructured, w *waitModel) (bool, error) {
	if w.Rollout.ValueBool() {
		ok, err := rolloutComplete(u)
		if err != nil || !ok {
			return false, err
		}
	}
	if !w.Condition.IsNull() && w.Condition.ValueString() != "" {
		ok, err := conditionMet(u, w.Condition.ValueString())
		if err != nil || !ok {
			return false, err
		}
	}
	if !w.Fields.IsNull() && !w.Fields.IsUnknown() {
		var fields map[string]string
		w.Fields.ElementsAs(ctx, &fields, false)
		for jp, want := range fields {
			got, err := evalJSONPath(u.Object, jp)
			if err != nil {
				return false, err
			}
			if got != want {
				return false, nil
			}
		}
	}
	return true, nil
}

// rolloutComplete implements kubectl-rollout-status-style readiness for the common
// workload kinds, without an external dependency.
func rolloutComplete(u *unstructured.Unstructured) (bool, error) {
	gen, _, _ := unstructured.NestedInt64(u.Object, "metadata", "generation")
	obs, _, _ := unstructured.NestedInt64(u.Object, "status", "observedGeneration")
	if obs < gen {
		return false, nil // controller hasn't observed the latest spec yet
	}

	switch u.GetKind() {
	case "Deployment":
		if msg := deploymentFailure(u); msg != "" {
			return false, fmt.Errorf("deployment rollout failed: %s", msg)
		}
		want := specReplicas(u)
		updated, _, _ := unstructured.NestedInt64(u.Object, "status", "updatedReplicas")
		ready, _, _ := unstructured.NestedInt64(u.Object, "status", "readyReplicas")
		avail, _, _ := unstructured.NestedInt64(u.Object, "status", "availableReplicas")
		return updated == want && ready == want && avail == want, nil

	case "StatefulSet":
		want := specReplicas(u)
		ready, _, _ := unstructured.NestedInt64(u.Object, "status", "readyReplicas")
		updated, _, _ := unstructured.NestedInt64(u.Object, "status", "updatedReplicas")
		cur, _, _ := unstructured.NestedString(u.Object, "status", "currentRevision")
		upd, _, _ := unstructured.NestedString(u.Object, "status", "updateRevision")
		return ready == want && updated == want && cur == upd && cur != "", nil

	case "DaemonSet":
		desired, _, _ := unstructured.NestedInt64(u.Object, "status", "desiredNumberScheduled")
		ready, _, _ := unstructured.NestedInt64(u.Object, "status", "numberReady")
		updated, _, _ := unstructured.NestedInt64(u.Object, "status", "updatedNumberScheduled")
		return desired > 0 && ready == desired && updated == desired, nil

	default:
		// Generic: satisfied if a Ready/Available condition is True; otherwise, we
		// cannot determine rollout for this kind, so treat as ready.
		if met, found := genericReadyCondition(u); found {
			return met, nil
		}
		return true, nil
	}
}

func specReplicas(u *unstructured.Unstructured) int64 {
	r, found, _ := unstructured.NestedInt64(u.Object, "spec", "replicas")
	if !found {
		return 1 // Kubernetes default
	}
	return r
}

// deploymentFailure returns a message if the Deployment has a hard rollout failure.
func deploymentFailure(u *unstructured.Unstructured) string {
	conds, _, _ := unstructured.NestedSlice(u.Object, "status", "conditions")
	for _, c := range conds {
		m, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		if m["type"] == "Progressing" && m["status"] == "False" && m["reason"] == "ProgressDeadlineExceeded" {
			if msg, ok := m["message"].(string); ok {
				return msg
			}
			return "ProgressDeadlineExceeded"
		}
	}
	return ""
}

func genericReadyCondition(u *unstructured.Unstructured) (met bool, found bool) {
	conds, _, _ := unstructured.NestedSlice(u.Object, "status", "conditions")
	for _, c := range conds {
		m, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		t, _ := m["type"].(string)
		if t == "Ready" || t == "Available" {
			return m["status"] == "True", true
		}
	}
	return false, false
}

// conditionMet parses "Type" or "Type=Status" (default Status=True) and checks status.conditions.
func conditionMet(u *unstructured.Unstructured, expr string) (bool, error) {
	ctype := expr
	want := "True"
	if i := strings.Index(expr, "="); i != -1 {
		ctype, want = expr[:i], expr[i+1:]
	}
	conds, _, _ := unstructured.NestedSlice(u.Object, "status", "conditions")
	for _, c := range conds {
		m, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		if m["type"] == ctype {
			return m["status"] == want, nil
		}
	}
	return false, nil
}

// evalJSONPath evaluates a jsonpath expression (e.g. "{.status.readyReplicas}") against the object.
func evalJSONPath(obj map[string]interface{}, expr string) (string, error) {
	jp := jsonpath.New("wait").AllowMissingKeys(true)
	if err := jp.Parse(expr); err != nil {
		return "", fmt.Errorf("invalid jsonpath %q: %w", expr, err)
	}
	var buf bytes.Buffer
	if err := jp.Execute(&buf, obj); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}

var badWaitingReasons = map[string]bool{
	"CrashLoopBackOff":           true,
	"ImagePullBackOff":           true,
	"ErrImagePull":               true,
	"InvalidImageName":           true,
	"CreateContainerConfigError": true,
	"CreateContainerError":       true,
	"RunContainerError":          true,
}

// withPodErrors augments a readiness error with pod/container failure reasons for
// workloads, so users see the root cause instead of a bare timeout. It uses a fresh
// short-lived context because the caller's context is often already cancelled (the
// readiness timeout is the common failure path), which would otherwise prevent us
// from fetching diagnostics.
func (r *ManifestYAML) withPodErrors(workload *unstructured.Unstructured, namespaced bool, base error) error {
	if !namespaced {
		return base
	}
	sel, found, _ := unstructured.NestedStringMap(workload.Object, "spec", "selector", "matchLabels")
	if !found || len(sel) == 0 {
		return base
	}
	cs, err := r.clients().MainClientset()
	if err != nil {
		return base
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pods, err := cs.CoreV1().Pods(workload.GetNamespace()).List(ctx, metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(sel).String(),
	})
	if err != nil {
		return base
	}

	var msgs []string
	for _, p := range pods.Items {
		for _, cst := range p.Status.ContainerStatuses {
			if wtng := cst.State.Waiting; wtng != nil && badWaitingReasons[wtng.Reason] {
				msgs = append(msgs, fmt.Sprintf("pod %s/%s: %s: %s",
					p.Name, cst.Name, wtng.Reason, strings.TrimSpace(wtng.Message)))
			}
		}
		for _, cond := range p.Status.Conditions {
			if cond.Type == corev1.PodScheduled && cond.Status == corev1.ConditionFalse {
				msgs = append(msgs, fmt.Sprintf("pod %s: %s: %s", p.Name, cond.Reason, strings.TrimSpace(cond.Message)))
			}
		}
	}
	if len(msgs) == 0 {
		return base
	}
	return fmt.Errorf("%w; pod diagnostics: %s", base, strings.Join(msgs, "; "))
}
