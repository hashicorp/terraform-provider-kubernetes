// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestpatch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/kubemanifest"
)

// fieldManagerOf returns the effective SSA field manager (default "terraform-patch").
func fieldManagerOf(m *manifestPatchModel) string {
	if m.FieldManager.IsNull() || m.FieldManager.IsUnknown() || m.FieldManager.ValueString() == "" {
		return defaultFieldManager
	}
	return m.FieldManager.ValueString()
}

// destroyBehaviorOf returns the effective destroy behavior (default "relinquish").
func destroyBehaviorOf(m *manifestPatchModel) string {
	if m.DestroyBehavior.IsNull() || m.DestroyBehavior.IsUnknown() || m.DestroyBehavior.ValueString() == "" {
		return destroyRelinquish
	}
	return m.DestroyBehavior.ValueString()
}

// ignoreList extracts ignore_fields as []string.
func ignoreList(ctx context.Context, m *manifestPatchModel) []string {
	if m.IgnoreFields.IsNull() || m.IgnoreFields.IsUnknown() {
		return nil
	}
	var out []string
	m.IgnoreFields.ElementsAs(ctx, &out, false)
	return out
}

// targetObject builds the minimal identity object (apiVersion/kind/metadata) used for
// GVR resolution and as the base of an SSA apply body.
func targetObject(m *manifestPatchModel) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetAPIVersion(m.APIVersion.ValueString())
	u.SetKind(m.Kind.ValueString())
	u.SetName(m.Name.ValueString())
	if ns := m.Namespace.ValueString(); ns != "" {
		u.SetNamespace(ns)
	}
	return u
}

// resolveTarget resolves the dynamic client interface for the target object.
func (r *ManifestPatch) resolveTarget(m *manifestPatchModel) (dynamic.ResourceInterface, *unstructured.Unstructured, error) {
	obj := targetObject(m)
	gvr, namespaced, err := kubemanifest.ResolveGVR(r.clients(), obj)
	if err != nil {
		return nil, nil, err
	}
	dyn, err := r.clients().DynamicClient()
	if err != nil {
		return nil, nil, err
	}
	return kubemanifest.ResourceInterface(dyn, gvr, namespaced, obj.GetNamespace()), obj, nil
}

// buildID returns apiVersion=..,kind=..,namespace=..,name=..,fieldManager=..
func buildID(m *manifestPatchModel) string {
	return fmt.Sprintf("apiVersion=%s,kind=%s,namespace=%s,name=%s,fieldManager=%s",
		m.APIVersion.ValueString(), m.Kind.ValueString(), m.Namespace.ValueString(),
		m.Name.ValueString(), fieldManagerOf(m))
}

// patchIdentity is the parsed import ID.
type patchIdentity struct {
	APIVersion, Kind, Namespace, Name, FieldManager string
}

// parsePatchID parses "apiVersion=..,kind=..,namespace=..,name=..,fieldManager=..".
func parsePatchID(id string) (patchIdentity, error) {
	var out patchIdentity
	for _, part := range strings.Split(id, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return out, fmt.Errorf("expected key=value pairs, got %q", part)
		}
		key, val := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
		switch key {
		case "apiVersion":
			out.APIVersion = val
		case "kind":
			out.Kind = val
		case "namespace":
			out.Namespace = val
		case "name":
			out.Name = val
		case "fieldManager":
			out.FieldManager = val
		default:
			return out, fmt.Errorf("unknown import key %q (allowed: apiVersion, kind, namespace, name, fieldManager)", key)
		}
	}
	if out.APIVersion == "" || out.Kind == "" || out.Name == "" {
		return out, fmt.Errorf("import ID must include apiVersion, kind and name")
	}
	return out, nil
}

// ssaBody builds the Server-Side Apply JSON body: the target identity deep-merged with
// the user's patch object (null leaves preserved for field removal).
func ssaBody(ctx context.Context, m *manifestPatchModel) ([]byte, error) {
	patchGo, err := kubemanifest.AttrToGo(ctx, m.Patch.UnderlyingValue())
	if err != nil {
		return nil, err
	}
	patchMap, ok := patchGo.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("patch must be an object (got %T)", patchGo)
	}
	// A null leaf means "remove this field": prune it so the SSA body omits it and the
	// API server prunes the field this manager owned (portable across field types).
	kubemanifest.PruneNulls(patchMap)
	root := map[string]interface{}{
		"apiVersion": m.APIVersion.ValueString(),
		"kind":       m.Kind.ValueString(),
		"metadata":   map[string]interface{}{"name": m.Name.ValueString()},
	}
	if ns := m.Namespace.ValueString(); ns != "" {
		root["metadata"].(map[string]interface{})["namespace"] = ns
	}
	merged := kubemanifest.DeepMerge(root, patchMap)
	return json.Marshal(merged)
}

// jsonPatchType maps the configured patch_type to the Kubernetes wire PatchType.
func jsonPatchType(pt string) (apitypes.PatchType, error) {
	switch pt {
	case patchTypeJSON:
		return apitypes.JSONPatchType, nil
	case patchTypeMerge:
		return apitypes.MergePatchType, nil
	case patchTypeStrategic:
		return apitypes.StrategicMergePatchType, nil
	default:
		return "", fmt.Errorf("unsupported patch_type %q for patch_json", pt)
	}
}

// apply mutates the target: Server-Side Apply for the `patch` object, or the configured
// patch_json strategy. When preview is true it is a server dry-run. Returns the live
// object as returned by the API.
func (r *ManifestPatch) apply(ctx context.Context, ri dynamic.ResourceInterface, m *manifestPatchModel, preview bool) (*unstructured.Unstructured, error) {
	name := m.Name.ValueString()

	if m.usesJSONPatch() {
		pt, err := jsonPatchType(m.PatchType.ValueString())
		if err != nil {
			return nil, err
		}
		opts := metav1.PatchOptions{FieldManager: fieldManagerOf(m)}
		if preview {
			opts.DryRun = []string{metav1.DryRunAll}
		}
		return ri.Patch(ctx, name, pt, []byte(m.PatchJSON.ValueString()), opts)
	}

	body, err := ssaBody(ctx, m)
	if err != nil {
		return nil, err
	}
	force := m.ForceConflicts.ValueBool() || preview // dry-run forces so drift isn't masked by conflicts
	opts := metav1.PatchOptions{FieldManager: fieldManagerOf(m), Force: &force}
	if preview {
		opts.DryRun = []string{metav1.DryRunAll}
	}
	return ri.Patch(ctx, name, apitypes.ApplyPatchType, body, opts)
}

// setComputed writes id/object_exists/owned_manifest/field_manager/destroy_behavior.
func setComputed(ctx context.Context, m *manifestPatchModel, live *unstructured.Unstructured) error {
	m.ID = types.StringValue(buildID(m))
	m.FieldManager = types.StringValue(fieldManagerOf(m))
	m.DestroyBehavior = types.StringValue(destroyBehaviorOf(m))
	m.ObjectExists = types.BoolValue(live != nil)

	// patch_json has no SSA ownership → no owned-field projection.
	if m.usesJSONPatch() || live == nil {
		m.OwnedManifest = types.StringValue("{}")
		return nil
	}
	proj, err := kubemanifest.ProjectOwned(live, fieldManagerOf(m), ignoreList(ctx, m))
	if err != nil {
		return err
	}
	m.OwnedManifest = types.StringValue(proj)
	return nil
}
