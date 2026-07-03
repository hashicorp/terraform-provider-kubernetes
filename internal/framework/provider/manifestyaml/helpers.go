// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestyaml

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/kubemanifest"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// This file is the thin delegation layer over the shared, model-agnostic engine in
// package kubemanifest (RFC-011 §6 / RFC-012 §4.8). The real logic lives there so it
// can be reused by kubernetes_manifest_patch; these wrappers keep the resource's
// internal call sites and tests unchanged.

func decodeYAML(y string) (*unstructured.Unstructured, error) { return kubemanifest.DecodeYAML(y) }

func countYAMLDocuments(y string) (int, error) { return kubemanifest.CountYAMLDocuments(y) }

func resolveGVR(clients kubernetes.KubeClientsets, obj *unstructured.Unstructured) (k8sschema.GroupVersionResource, bool, error) {
	return kubemanifest.ResolveGVR(clients, obj)
}

func resourceInterface(dyn dynamic.Interface, gvr k8sschema.GroupVersionResource, namespaced bool, ns string) dynamic.ResourceInterface {
	return kubemanifest.ResourceInterface(dyn, gvr, namespaced, ns)
}

func buildID(obj *unstructured.Unstructured) string { return kubemanifest.BuildID(obj) }

func parseImportID(id string) (kubemanifest.ImportIdentity, error) {
	return kubemanifest.ParseImportID(id)
}

func unstructuredForImport(id kubemanifest.ImportIdentity) *unstructured.Unstructured {
	return kubemanifest.UnstructuredForImport(id)
}

func projectOwned(obj *unstructured.Unstructured, fieldManager string, ignore []string) (string, error) {
	return kubemanifest.ProjectOwned(obj, fieldManager, ignore)
}

func removeDotPath(m map[string]interface{}, path string) { kubemanifest.RemoveDotPath(m, path) }

func valueAtDotPath(m map[string]interface{}, dotted string) (interface{}, bool) {
	return kubemanifest.ValueAtDotPath(m, dotted)
}

func pathChanged(a, b map[string]interface{}, dotted string) bool {
	return kubemanifest.PathChanged(a, b, dotted)
}

func normalizeYAML(y string) (string, error) { return kubemanifest.NormalizeYAML(y) }

func applyErrDiag(err error) (string, string) {
	return kubemanifest.ApplyErrDiag("kubernetes_manifest_yaml", err)
}

func isImmutableErr(err error) bool { return kubemanifest.IsImmutableErr(err) }

func waitForDeleted(ctx context.Context, ri dynamic.ResourceInterface, name string) error {
	return kubemanifest.WaitForDeleted(ctx, ri, name)
}

// opTimeout adapts the shared OpTimeout to this resource's wait{} block: the effective
// operation timeout is never shorter than the readiness wait.
func opTimeout(ctx context.Context, tv timeouts.Value, kind string, w *waitModel) (time.Duration, diag.Diagnostics) {
	var floor time.Duration
	if w != nil && w.hasAny() {
		floor = waitTimeout(w) + time.Minute
	}
	return kubemanifest.OpTimeout(ctx, tv, kind, floor)
}
