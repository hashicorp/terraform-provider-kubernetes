// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/corev1"
)

// These run offline. The state upgrader is the hinge of the whole migration —
// if it does not convert SDKv2's zero-filled empties, every existing namespace
// plans a diff on the first plan after upgrade — and it is pure state
// transformation, so it can be proven without a cluster. The acceptance tests in
// this package need a real API server; these do not, and should stay that way.

// sdkv2StateV0 is the state SDKv2 writes for a namespace whose configuration
// gave only a name. annotations and labels are `{}` and generate_name is `""`
// because SDKv2's flatteners always write those keys and d.Set zero-fills the
// rest of the block.
const sdkv2StateV0 = `{
  "id": "tf-acc-test-ns",
  "metadata": [
    {
      "annotations": {},
      "generate_name": "",
      "generation": 0,
      "labels": {},
      "name": "tf-acc-test-ns",
      "resource_version": "12345",
      "uid": "8f3a6a7c-1b2c-4d5e-8f90-0123456789ab"
    }
  ],
  "timeouts": null,
  "wait_for_default_service_account": false
}`

// sdkv2StateV0Populated is the same resource with real metadata and a
// configured delete timeout: nothing here may be touched by the upgrade.
const sdkv2StateV0Populated = `{
  "id": "tf-acc-test-ns",
  "metadata": [
    {
      "annotations": {"TestAnnotationOne": "one"},
      "generate_name": "tf-acc-test-gen-",
      "generation": 3,
      "labels": {"TestLabelOne": "one"},
      "name": "tf-acc-test-ns",
      "resource_version": "12345",
      "uid": "8f3a6a7c-1b2c-4d5e-8f90-0123456789ab"
    }
  ],
  "timeouts": {"delete": "30m"},
  "wait_for_default_service_account": true
}`

func upgradeV0(t *testing.T, rawJSON string) corev1.NamespaceV1Model {
	t.Helper()
	ctx := context.Background()

	r := corev1.NewNamespaceV1()

	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("building the schema: %v", schemaResp.Diagnostics)
	}

	upgraders := r.(fwresource.ResourceWithUpgradeState).UpgradeState(ctx)
	upgrader, ok := upgraders[0]
	if !ok {
		t.Fatal("no upgrader registered for stored schema version 0; every namespace written by an SDKv2 release would fail to upgrade")
	}
	if upgrader.PriorSchema == nil {
		t.Fatal("the version 0 upgrader has no PriorSchema, so req.State is never populated")
	}

	priorValue, err := tfprotov6.RawState{JSON: []byte(rawJSON)}.UnmarshalWithOpts(
		upgrader.PriorSchema.Type().TerraformType(ctx),
		tfprotov6.UnmarshalOpts{
			ValueFromJSONOpts: tftypes.ValueFromJSONOpts{IgnoreUndefinedAttributes: true},
		},
	)
	if err != nil {
		t.Fatalf("decoding SDKv2 state against the prior schema: %s", err)
	}

	req := fwresource.UpgradeStateRequest{
		RawState: &tfprotov6.RawState{JSON: []byte(rawJSON)},
		State:    &tfsdk.State{Raw: priorValue, Schema: *upgrader.PriorSchema},
	}
	resp := fwresource.UpgradeStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	upgrader.StateUpgrader(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("upgrading state: %v", resp.Diagnostics)
	}

	var got corev1.NamespaceV1Model
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("reading upgraded state: %v", diags)
	}
	return got
}

func TestNamespaceV1UpgradeState_nullsSDKv2ZeroValues(t *testing.T) {
	got := upgradeV0(t, sdkv2StateV0)

	if len(got.Metadata) != 1 {
		t.Fatalf("metadata block count = %d, want 1", len(got.Metadata))
	}
	meta := got.Metadata[0]

	if !meta.Annotations.IsNull() {
		t.Errorf("annotations = %v, want null; SDKv2's {} left in place makes every existing namespace plan `annotations: {} -> null`", meta.Annotations)
	}
	if !meta.Labels.IsNull() {
		t.Errorf("labels = %v, want null", meta.Labels)
	}
	if !meta.GenerateName.IsNull() {
		t.Errorf("generate_name = %v, want null", meta.GenerateName)
	}

	// Everything else must survive untouched.
	if got.ID.ValueString() != "tf-acc-test-ns" {
		t.Errorf("id = %q, want %q", got.ID.ValueString(), "tf-acc-test-ns")
	}
	if meta.Name.ValueString() != "tf-acc-test-ns" {
		t.Errorf("name = %q, want %q", meta.Name.ValueString(), "tf-acc-test-ns")
	}
	if meta.UID.ValueString() != "8f3a6a7c-1b2c-4d5e-8f90-0123456789ab" {
		t.Errorf("uid = %q, want it preserved", meta.UID.ValueString())
	}
	if meta.ResourceVersion.ValueString() != "12345" {
		t.Errorf("resource_version = %q, want it preserved", meta.ResourceVersion.ValueString())
	}
	if meta.Generation.ValueInt64() != 0 {
		t.Errorf("generation = %d, want 0", meta.Generation.ValueInt64())
	}
	if got.WaitForDefaultServiceAccount.ValueBool() {
		t.Error("wait_for_default_service_account = true, want false")
	}
	if !got.Timeouts.Object.IsNull() {
		t.Errorf("timeouts = %v, want null", got.Timeouts.Object)
	}
}

func TestNamespaceV1UpgradeState_preservesConfiguredValues(t *testing.T) {
	got := upgradeV0(t, sdkv2StateV0Populated)
	meta := got.Metadata[0]

	if meta.Annotations.IsNull() {
		t.Error("annotations were nulled; a configured annotation must survive the upgrade")
	}
	if n := len(meta.Annotations.Elements()); n != 1 {
		t.Errorf("annotations has %d entries, want 1", n)
	}
	if meta.Labels.IsNull() {
		t.Error("labels were nulled; a configured label must survive the upgrade")
	}
	if meta.GenerateName.ValueString() != "tf-acc-test-gen-" {
		t.Errorf("generate_name = %q, want it preserved", meta.GenerateName.ValueString())
	}
	if meta.Generation.ValueInt64() != 3 {
		t.Errorf("generation = %d, want 3", meta.Generation.ValueInt64())
	}
	if !got.WaitForDefaultServiceAccount.ValueBool() {
		t.Error("wait_for_default_service_account = false, want true")
	}
	if got.Timeouts.Object.IsNull() {
		t.Error("timeouts were dropped; a configured delete timeout must survive the upgrade")
	}
}

// TestNamespaceV1UpgradeState_toleratesUnknownStoredAttributes guards the decode
// path rather than the transformation: state written by a future or patched
// release may carry keys this schema does not define, and the upgrade must not
// fail on them.
func TestNamespaceV1UpgradeState_toleratesUnknownStoredAttributes(t *testing.T) {
	const withExtra = `{
  "id": "tf-acc-test-ns",
  "metadata": [
    {
      "annotations": {},
      "generate_name": "",
      "generation": 0,
      "labels": {},
      "name": "tf-acc-test-ns",
      "resource_version": "12345",
      "uid": "8f3a6a7c-1b2c-4d5e-8f90-0123456789ab"
    }
  ],
  "timeouts": null,
  "wait_for_default_service_account": false,
  "some_future_attribute": "ignored"
}`

	got := upgradeV0(t, withExtra)
	if !got.Metadata[0].Annotations.IsNull() {
		t.Error("annotations were not normalised when the stored state carried an unknown attribute")
	}
}

// TestNamespaceV1SchemaVersion pins the version bump itself. UpgradeState is
// only consulted when the STORED version is lower than the current one, so a
// schema version of 0 would mean the conversion above never runs at all and the
// passthrough branch would hand SDKv2's `{}` straight through.
func TestNamespaceV1SchemaVersion(t *testing.T) {
	ctx := context.Background()

	var schemaResp fwresource.SchemaResponse
	corev1.NewNamespaceV1().Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)

	if got := schemaResp.Schema.Version; got != 1 {
		t.Errorf("schema version = %d, want 1; at version 0 the v0 upgrader is unreachable", got)
	}
}
