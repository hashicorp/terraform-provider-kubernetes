// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/corev1"
)

// The gating tests below are the important ones and they run offline.
//
// A StateMover declines a move by returning with TargetState untouched, which
// lets the framework try the next mover; it claims the move by setting state.
// There is no third signal. So "did it correctly refuse?" is answered by
// comparing TargetState against the null value of the target type, exactly as
// fwserver/server_moveresourcestate.go does.

// sourceStateDeprecatedNamespace is the state SDKv2 writes for
// kubernetes_namespace — the same shape as kubernetes_namespace_v1, since both
// names are served by the same constructor.
const sourceStateDeprecatedNamespace = `{
  "id": "tf-moved-ns",
  "metadata": [
    {
      "annotations": null,
      "generate_name": "",
      "generation": 0,
      "labels": null,
      "name": "tf-moved-ns",
      "resource_version": "4242",
      "uid": "1c1a2b3c-4d5e-6f70-8192-a3b4c5d6e7f8"
    }
  ],
  "timeouts": null,
  "wait_for_default_service_account": false
}`

type moveResult struct {
	claimed bool
	state   corev1.NamespaceV1Model
	diags   string
}

func runMover(t *testing.T, req fwresource.MoveStateRequest) moveResult {
	t.Helper()
	ctx := context.Background()

	r := corev1.NewNamespaceV1()

	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("building the schema: %v", schemaResp.Diagnostics)
	}
	targetType := schemaResp.Schema.Type().TerraformType(ctx)
	nullState := tftypes.NewValue(targetType, nil)

	movers := r.(fwresource.ResourceWithMoveState).MoveState(ctx)
	if len(movers) == 0 {
		t.Fatal("no state movers registered; a moved block from kubernetes_namespace cannot work")
	}

	resp := fwresource.MoveStateResponse{
		TargetState: tfsdk.State{Schema: schemaResp.Schema, Raw: nullState},
	}
	for _, m := range movers {
		m.StateMover(ctx, req, &resp)
		if resp.Diagnostics.HasError() || !resp.TargetState.Raw.Equal(nullState) {
			break
		}
	}

	var errs []string
	for _, d := range resp.Diagnostics.Errors() {
		errs = append(errs, d.Summary()+": "+d.Detail())
	}
	out := moveResult{
		claimed: !resp.TargetState.Raw.Equal(nullState),
		diags:   strings.Join(errs, "; "),
	}
	if out.claimed {
		if d := resp.TargetState.Get(ctx, &out.state); d.HasError() {
			t.Fatalf("reading moved state: %v", d)
		}
	}
	return out
}

func validMoveRequest() fwresource.MoveStateRequest {
	return fwresource.MoveStateRequest{
		SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes",
		SourceTypeName:        "kubernetes_namespace",
		SourceSchemaVersion:   0,
		SourceRawState:        &tfprotov6.RawState{JSON: []byte(sourceStateDeprecatedNamespace)},
	}
}

func TestNamespaceV1MoveState_movesFromDeprecatedAlias(t *testing.T) {
	got := runMover(t, validMoveRequest())

	if !got.claimed {
		t.Fatalf("the mover declined a valid kubernetes_namespace source; diags: %s", got.diags)
	}
	if got.diags != "" {
		t.Fatalf("unexpected diagnostics: %s", got.diags)
	}

	if got.state.ID.ValueString() != "tf-moved-ns" {
		t.Errorf("id = %q, want %q", got.state.ID.ValueString(), "tf-moved-ns")
	}
	meta := got.state.Metadata[0]
	if meta.Name.ValueString() != "tf-moved-ns" {
		t.Errorf("metadata.name = %q, want %q", meta.Name.ValueString(), "tf-moved-ns")
	}
	if meta.UID.ValueString() != "1c1a2b3c-4d5e-6f70-8192-a3b4c5d6e7f8" {
		t.Errorf("metadata.uid = %q, want it carried across", meta.UID.ValueString())
	}
	if meta.ResourceVersion.ValueString() != "4242" {
		t.Errorf("metadata.resource_version = %q, want it carried across", meta.ResourceVersion.ValueString())
	}

	// The move must apply the same normalisation as the v0 upgrader. A moved
	// state lands at the target's CURRENT schema version, so no upgrader runs
	// after it — leaving generate_name as "" here would give the practitioner a
	// diff on the very first plan after the move.
	if !meta.GenerateName.IsNull() {
		t.Errorf("metadata.generate_name = %v, want null", meta.GenerateName)
	}
	if !meta.Annotations.IsNull() {
		t.Errorf("metadata.annotations = %v, want null", meta.Annotations)
	}
	if !meta.Labels.IsNull() {
		t.Errorf("metadata.labels = %v, want null", meta.Labels)
	}
	if got.state.WaitForDefaultServiceAccount.ValueBool() {
		t.Error("wait_for_default_service_account = true, want false")
	}
	if !got.state.Timeouts.Object.IsNull() {
		t.Errorf("timeouts = %v, want null", got.state.Timeouts.Object)
	}
}

// TestNamespaceV1MoveState_declines is the rule that keeps this mover honest.
// Each case must return NO state and NO diagnostics, so the framework can try
// another mover and ultimately report an accurate error rather than this
// resource claiming a move it cannot correctly perform.
func TestNamespaceV1MoveState_declines(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*fwresource.MoveStateRequest)
	}{
		{
			name: "a different provider that happens to use the same type name",
			mutate: func(r *fwresource.MoveStateRequest) {
				r.SourceProviderAddress = "registry.terraform.io/acme/kubernetes"
			},
		},
		{
			name:   "an unrelated source resource type",
			mutate: func(r *fwresource.MoveStateRequest) { r.SourceTypeName = "kubernetes_config_map" },
		},
		{
			name:   "a source schema version this decode does not understand",
			mutate: func(r *fwresource.MoveStateRequest) { r.SourceSchemaVersion = 1 },
		},
		{
			name:   "no source state at all",
			mutate: func(r *fwresource.MoveStateRequest) { r.SourceRawState = nil },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := validMoveRequest()
			tc.mutate(&req)

			got := runMover(t, req)
			if got.claimed {
				t.Error("the mover claimed a move it should have declined")
			}
			if got.diags != "" {
				t.Errorf("declining must be silent so the framework can try the next mover, but got: %s", got.diags)
			}
		})
	}
}

// A mirror image of the hostname case above: the same namespace served through a
// network mirror or private registry must still move. Matching the provider
// address by full string rather than by suffix is a common mistake that breaks
// exactly these practitioners.
func TestNamespaceV1MoveState_acceptsMirroredProviderAddress(t *testing.T) {
	for _, addr := range []string{
		"registry.terraform.io/hashicorp/kubernetes",
		"terraform.example.com/hashicorp/kubernetes",
		"hashicorp/kubernetes",
	} {
		t.Run(addr, func(t *testing.T) {
			req := validMoveRequest()
			req.SourceProviderAddress = addr

			if got := runMover(t, req); !got.claimed {
				t.Errorf("the mover declined source provider address %q; diags: %s", addr, got.diags)
			}
		})
	}
}

func TestNamespaceV1MoveState_reportsUnreadableSourceState(t *testing.T) {
	req := validMoveRequest()
	req.SourceRawState = &tfprotov6.RawState{JSON: []byte(`{"metadata": []}`)}

	got := runMover(t, req)
	if got.claimed {
		t.Error("the mover claimed a move from state with no metadata block")
	}
	if got.diags == "" {
		t.Error("a source state that gated in but cannot be used must report a diagnostic, not fail silently")
	}
}

// TestAccKubernetesNamespaceV1_movedFromDeprecatedAlias is the end-to-end proof
// that the migration delivers what it was done for: a practitioner retires
// kubernetes_namespace with a moved block, and the namespace is NOT recreated.
//
// The factory must be the mux (rule K8S-MIGRATE-019). The `from` side of the
// moved block is kubernetes_namespace, which is still served by SDKv2, so a
// Framework-only server could not even init this configuration.
func TestAccKubernetesNamespaceV1_movedFromDeprecatedAlias(t *testing.T) {
	nsName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	var uid string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		// Cross-type moved support requires Terraform 1.8 (rule K8S-MIGRATE-018).
		// Without this the suite goes red on older CLIs for a reason that has
		// nothing to do with the provider, which is how an ExpectEmptyPlan
		// assertion ends up being deleted by someone chasing a failure.
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		CheckDestroy: testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				// Create under the DEPRECATED alias, on the SDKv2 server.
				Config: testAccKubernetesNamespaceConfig_deprecatedAlias(nsName),
				Check:  captureNamespaceUID(nsName, &uid),
			},
			{
				// Move it to the Framework-served _v1 name. The plan must be a
				// no-op: a moved block that proposes changes, or worse a
				// replacement, would destroy every object inside the namespace.
				Config: testAccKubernetesNamespaceV1Config_movedFromAlias(nsName),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					assertNamespaceUIDUnchanged(nsName, &uid),
					resource.TestCheckResourceAttr(resourceName, "id", nsName),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", nsName),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					// The move carries identity across, so the moved resource is
					// importable immediately rather than only after a refresh.
					statecheck.ExpectIdentity(resourceName, map[string]knownvalue.Check{
						"name":        knownvalue.StringExact(nsName),
						"api_version": knownvalue.StringExact("v1"),
						"kind":        knownvalue.StringExact("Namespace"),
					}),
					// Normalisation must have been applied by the MOVER; no
					// upgrader runs after a move.
					metadataNull("generate_name"),
					metadataNull("annotations"),
					metadataNull("labels"),
				},
			},
			{
				// And it settles: the plan after the move is still empty.
				Config: testAccKubernetesNamespaceV1Config_movedFromAlias(nsName),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func testAccKubernetesNamespaceConfig_deprecatedAlias(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace" "test" {
  metadata {
    name = "%s"
  }
}
`, nsName)
}

func testAccKubernetesNamespaceV1Config_movedFromAlias(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = "%s"
  }
}

moved {
  from = kubernetes_namespace.test
  to   = kubernetes_namespace_v1.test
}
`, nsName)
}
