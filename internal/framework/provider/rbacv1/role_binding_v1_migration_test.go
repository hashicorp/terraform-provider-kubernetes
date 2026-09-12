// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1_test

// Migration tests for kubernetes_role_binding_v1.
//
// These tests verify MoveState — the mechanism that allows users to migrate
// from the deprecated kubernetes_role_binding (SDKv2) resource to
// kubernetes_role_binding_v1 (Framework) using a `moved` block:
//
//	moved {
//	  from = kubernetes_role_binding.example
//	  to   = kubernetes_role_binding_v1.example
//	}
//
// No UpgradeState handler is needed because the Framework schema uses
// ListNestedBlock for metadata and other blocks, which produces an identical
// JSON state shape to the SDKv2 TypeList{MaxItems:1} — schema_version stays
// at 0 and Terraform reads the existing state directly without any upgrade step.

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	rbacv1 "github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/rbacv1"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// sdkv2RawJSON produces raw JSON bytes that mirror what the SDKv2 provider
// writes into terraform.tfstate for kubernetes_role_binding.
func sdkv2RawJSON(
	id, name, generateName, namespace string,
	annotations, labels map[string]string,
	resourceVersion, uid string,
	generation int,
	roleRefAPIGroup, roleRefKind, roleRefName string,
	subjects []map[string]string,
) []byte {
	meta := map[string]interface{}{
		"name":             name,
		"generate_name":    generateName,
		"namespace":        namespace,
		"resource_version": resourceVersion,
		"uid":              uid,
		"generation":       generation,
		"annotations":      annotations,
		"labels":           labels,
	}

	roleRef := []interface{}{
		map[string]interface{}{
			"api_group": roleRefAPIGroup,
			"kind":      roleRefKind,
			"name":      roleRefName,
		},
	}

	subjectList := make([]interface{}, 0, len(subjects))
	for _, s := range subjects {
		subjectList = append(subjectList, map[string]interface{}{
			"api_group": s["api_group"],
			"kind":      s["kind"],
			"name":      s["name"],
			"namespace": s["namespace"],
		})
	}

	state := map[string]interface{}{
		"id":       id,
		"metadata": []interface{}{meta},
		"role_ref": roleRef,
		"subject":  subjectList,
	}
	raw, _ := json.Marshal(state)
	return raw
}

// runMoveState calls the MoveState handler with the given source type and raw JSON.
func runMoveState(t *testing.T, sourceTypeName string, rawJSON []byte) *resource.MoveStateResponse {
	t.Helper()
	r := rbacv1.NewRoleBindingV1()
	movers := r.(interface {
		MoveState(context.Context) []resource.StateMover
	}).MoveState(context.Background())

	if len(movers) == 0 {
		t.Fatal("expected at least 1 StateMover")
	}

	req := resource.MoveStateRequest{
		SourceTypeName: sourceTypeName,
		SourceRawState: &tfprotov6.RawState{JSON: rawJSON},
	}
	resp := &resource.MoveStateResponse{
		TargetState: tfsdk.State{Schema: rbacv1.RoleBindingV1Schema()},
	}
	movers[0].StateMover(context.Background(), req, resp)
	return resp
}

// readMovedModel reads the moved RoleBindingModel out of resp.TargetState.
func readMovedModel(t *testing.T, resp *resource.MoveStateResponse) rbacv1.RoleBindingModel {
	t.Helper()
	if resp.Diagnostics.HasError() {
		t.Fatalf("move state produced errors: %s", resp.Diagnostics)
	}
	var m rbacv1.RoleBindingModel
	resp.Diagnostics.Append(resp.TargetState.Get(context.Background(), &m)...)
	if resp.Diagnostics.HasError() {
		t.Fatalf("reading moved state: %s", resp.Diagnostics)
	}
	return m
}

// ── MoveState tests ───────────────────────────────────────────────────────────

// TestMigration_MoveState_handlersRegistered verifies the StateMover is wired
// up — a compile-time regression guard.
func TestMigration_MoveState_handlersRegistered(t *testing.T) {
	t.Parallel()
	r := rbacv1.NewRoleBindingV1()
	movers := r.(interface {
		MoveState(context.Context) []resource.StateMover
	}).MoveState(context.Background())

	if len(movers) == 0 {
		t.Error("expected at least 1 StateMover registered")
	}
}

// TestMigration_MoveState_basic verifies the core translation from
// kubernetes_role_binding raw JSON state to RoleBindingModel.
func TestMigration_MoveState_basic(t *testing.T) {
	t.Parallel()

	raw := sdkv2RawJSON(
		"default/my-binding",
		"my-binding", "", "default",
		map[string]string{"example.com/note": "test"},
		map[string]string{"managed-by": "terraform"},
		"567273", "a6da86ec-b80d-44c0-9007-aafa4d982d4a", 1,
		"rbac.authorization.k8s.io", "Role", "pod-reader",
		[]map[string]string{
			{"api_group": "rbac.authorization.k8s.io", "kind": "User", "name": "alice", "namespace": "default"},
		},
	)

	resp := runMoveState(t, "kubernetes_role_binding", raw)
	got := readMovedModel(t, resp)

	if got.ID.ValueString() != "default/my-binding" {
		t.Errorf("id: got %q, want default/my-binding", got.ID.ValueString())
	}
	if len(got.Metadata) != 1 {
		t.Fatalf("metadata: got %d elements, want 1", len(got.Metadata))
	}
	if got.Metadata[0].Name.ValueString() != "my-binding" {
		t.Errorf("name: got %q, want my-binding", got.Metadata[0].Name.ValueString())
	}
	if got.Metadata[0].Namespace.ValueString() != "default" {
		t.Errorf("namespace: got %q, want default", got.Metadata[0].Namespace.ValueString())
	}
	if len(got.RoleRef) != 1 {
		t.Fatalf("role_ref: got %d elements, want 1", len(got.RoleRef))
	}
	if got.RoleRef[0].Kind.ValueString() != "Role" {
		t.Errorf("role_ref.kind: got %q, want Role", got.RoleRef[0].Kind.ValueString())
	}
	if got.RoleRef[0].Name.ValueString() != "pod-reader" {
		t.Errorf("role_ref.name: got %q, want pod-reader", got.RoleRef[0].Name.ValueString())
	}
	if len(got.Subject) != 1 {
		t.Fatalf("subject: got %d elements, want 1", len(got.Subject))
	}
	if got.Subject[0].Name.ValueString() != "alice" {
		t.Errorf("subject[0].name: got %q, want alice", got.Subject[0].Name.ValueString())
	}
	if got.Metadata[0].Annotations["example.com/note"].ValueString() != "test" {
		t.Errorf("annotation: got %q, want test",
			got.Metadata[0].Annotations["example.com/note"].ValueString())
	}
	if got.Metadata[0].Labels["managed-by"].ValueString() != "terraform" {
		t.Errorf("label: got %q, want terraform",
			got.Metadata[0].Labels["managed-by"].ValueString())
	}
}

// TestMigration_MoveState_emptyGenerateName verifies that an empty string
// generate_name from SDKv2 is normalised to null to prevent plan drift.
func TestMigration_MoveState_emptyGenerateName(t *testing.T) {
	t.Parallel()

	raw := sdkv2RawJSON(
		"default/my-binding", "my-binding", "" /* empty generate_name */, "default",
		nil, nil, "1", "uid-1", 0,
		"rbac.authorization.k8s.io", "Role", "admin",
		[]map[string]string{
			{"api_group": "rbac.authorization.k8s.io", "kind": "User", "name": "bob", "namespace": "default"},
		},
	)

	resp := runMoveState(t, "kubernetes_role_binding", raw)
	got := readMovedModel(t, resp)

	if !got.Metadata[0].GenerateName.IsNull() {
		t.Errorf("generate_name: expected null for empty string, got %q",
			got.Metadata[0].GenerateName.ValueString())
	}
}

// TestMigration_MoveState_nonEmptyGenerateName verifies that a set
// generate_name is preserved after the move.
func TestMigration_MoveState_nonEmptyGenerateName(t *testing.T) {
	t.Parallel()

	raw := sdkv2RawJSON(
		"default/rb-gen-xk9p2", "rb-gen-xk9p2", "rb-gen-", "default",
		nil, nil, "2", "uid-2", 0,
		"rbac.authorization.k8s.io", "Role", "admin",
		[]map[string]string{
			{"api_group": "rbac.authorization.k8s.io", "kind": "User", "name": "carol", "namespace": "default"},
		},
	)

	resp := runMoveState(t, "kubernetes_role_binding", raw)
	got := readMovedModel(t, resp)

	if got.Metadata[0].GenerateName.IsNull() {
		t.Error("generate_name: expected non-null for 'rb-gen-', got null")
	}
	if got.Metadata[0].GenerateName.ValueString() != "rb-gen-" {
		t.Errorf("generate_name: got %q, want rb-gen-",
			got.Metadata[0].GenerateName.ValueString())
	}
}

// TestMigration_MoveState_emptyMapsAreNil verifies that empty annotation and
// label maps from SDKv2 become nil in the Framework model, preventing a
// perpetual plan diff against configs that omit annotations/labels entirely.
func TestMigration_MoveState_emptyMapsAreNil(t *testing.T) {
	t.Parallel()

	raw := sdkv2RawJSON(
		"default/my-binding", "my-binding", "", "default",
		map[string]string{}, map[string]string{}, // empty maps
		"1", "uid-3", 0,
		"rbac.authorization.k8s.io", "Role", "admin",
		[]map[string]string{
			{"api_group": "rbac.authorization.k8s.io", "kind": "User", "name": "dave", "namespace": "default"},
		},
	)

	resp := runMoveState(t, "kubernetes_role_binding", raw)
	got := readMovedModel(t, resp)

	if got.Metadata[0].Annotations != nil {
		t.Errorf("annotations: expected nil for empty map, got %v",
			got.Metadata[0].Annotations)
	}
	if got.Metadata[0].Labels != nil {
		t.Errorf("labels: expected nil for empty map, got %v",
			got.Metadata[0].Labels)
	}
}

// TestMigration_MoveState_multipleSubjects verifies that all three subject
// types (User, ServiceAccount, Group) are preserved correctly after the move.
func TestMigration_MoveState_multipleSubjects(t *testing.T) {
	t.Parallel()

	raw := sdkv2RawJSON(
		"default/my-binding", "my-binding", "", "default",
		nil, nil, "1", "uid-4", 0,
		"rbac.authorization.k8s.io", "Role", "pod-reader",
		[]map[string]string{
			{"api_group": "rbac.authorization.k8s.io", "kind": "User", "name": "alice", "namespace": "default"},
			{"api_group": "", "kind": "ServiceAccount", "name": "default", "namespace": "kube-system"},
			{"api_group": "rbac.authorization.k8s.io", "kind": "Group", "name": "dev-team", "namespace": "default"},
		},
	)

	resp := runMoveState(t, "kubernetes_role_binding", raw)
	got := readMovedModel(t, resp)

	if len(got.Subject) != 3 {
		t.Fatalf("subject: got %d elements, want 3", len(got.Subject))
	}
	if got.Subject[0].Kind.ValueString() != "User" {
		t.Errorf("subject[0].kind: got %q, want User", got.Subject[0].Kind.ValueString())
	}
	if got.Subject[1].Kind.ValueString() != "ServiceAccount" {
		t.Errorf("subject[1].kind: got %q, want ServiceAccount", got.Subject[1].Kind.ValueString())
	}
	if got.Subject[1].Namespace.ValueString() != "kube-system" {
		t.Errorf("subject[1].namespace: got %q, want kube-system", got.Subject[1].Namespace.ValueString())
	}
	if got.Subject[2].Kind.ValueString() != "Group" {
		t.Errorf("subject[2].kind: got %q, want Group", got.Subject[2].Kind.ValueString())
	}
}

// TestMigration_MoveState_clusterRoleRef verifies that a ClusterRole reference
// in role_ref is preserved correctly.
func TestMigration_MoveState_clusterRoleRef(t *testing.T) {
	t.Parallel()

	raw := sdkv2RawJSON(
		"default/my-binding", "my-binding", "", "default",
		nil, nil, "1", "uid-5", 0,
		"rbac.authorization.k8s.io", "ClusterRole", "cluster-admin",
		[]map[string]string{
			{"api_group": "rbac.authorization.k8s.io", "kind": "User", "name": "alice", "namespace": "default"},
		},
	)

	resp := runMoveState(t, "kubernetes_role_binding", raw)
	got := readMovedModel(t, resp)

	if got.RoleRef[0].Kind.ValueString() != "ClusterRole" {
		t.Errorf("role_ref.kind: got %q, want ClusterRole", got.RoleRef[0].Kind.ValueString())
	}
	if got.RoleRef[0].Name.ValueString() != "cluster-admin" {
		t.Errorf("role_ref.name: got %q, want cluster-admin", got.RoleRef[0].Name.ValueString())
	}
}

// TestMigration_MoveState_wrongSourceTypeIsIgnored verifies that the handler
// returns early without error or writing any state when SourceTypeName does
// not match "kubernetes_role_binding". This ensures the handler is safe to
// call for any moved block in the configuration.
func TestMigration_MoveState_wrongSourceTypeIsIgnored(t *testing.T) {
	t.Parallel()

	raw := sdkv2RawJSON(
		"default/some-other", "some-other", "", "default",
		nil, nil, "1", "uid-6", 0,
		"rbac.authorization.k8s.io", "Role", "admin",
		[]map[string]string{
			{"api_group": "rbac.authorization.k8s.io", "kind": "User", "name": "alice", "namespace": "default"},
		},
	)

	resp := runMoveState(t, "kubernetes_some_other_resource", raw)

	// No errors expected — handler must silently return
	if resp.Diagnostics.HasError() {
		t.Errorf("expected no errors for unrecognised source type, got: %s",
			resp.Diagnostics)
	}

	// TargetState must be empty — handler must not have written anything
	var m rbacv1.RoleBindingModel
	diags := resp.TargetState.Get(context.Background(), &m)
	if !diags.HasError() && m.ID.ValueString() != "" {
		t.Errorf("expected empty target state for unrecognised source type, got id=%q",
			m.ID.ValueString())
	}
}
