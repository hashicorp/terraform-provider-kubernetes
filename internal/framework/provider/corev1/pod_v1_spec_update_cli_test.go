// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

type podSpecCanonicalQuantityTestResource struct {
	podSpecNativeTestResource
}

func (podSpecCanonicalQuantityTestResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "podquantity_canonical"
}

func (r podSpecCanonicalQuantityTestResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.podSpecNativeTestResource.Create(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx,
		path.Root("spec").AtListIndex(0).AtName("container").AtListIndex(0).
			AtName("resources").AtListIndex(0).AtName("limits").AtMapKey("cpu"),
		types.StringValue("500m"))...)
}

func TestPodV1SpecCoreRejectsCreateQuantityCanonicalization(t *testing.T) {
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"podquantity": providerserver.NewProtocol6WithError(podSpecQuantityTestProvider{}),
		},
		Steps: []testresource.TestStep{{
			Config: `
resource "podquantity_canonical" "test" {
  spec {
    container {
      name      = "app"
      image     = "image:1"
      resources = [{ limits = { cpu = "0.5" } }]
    }
  }
}`,
			ExpectError: regexp.MustCompile(`(?s)Provider produced inconsistent result.*500m`),
		}},
	})
}

func TestPodV1SpecMixedQuantityChangesCore(t *testing.T) {
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"podquantity": providerserver.NewProtocol6WithError(podSpecQuantityTestProvider{}),
		},
		Steps: []testresource.TestStep{
			{Config: `resource "podquantity_test" "test" {
  limits = { cpu = "500m", memory = "128Mi" }
}`},
			{
				Config: `resource "podquantity_test" "test" {
  limits = { cpu = "0.5", memory = "512Mi", ephemeral-storage = "512Mi" }
}`,
				ConfigPlanChecks: testresource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction("podquantity_test.test", plancheck.ResourceActionDestroyBeforeCreate),
				}},
			},
		},
	})
}

func TestPodV1SpecProviderOnlyUpdateCore(t *testing.T) {
	for _, test := range []struct {
		name, resources, limits, requests string
	}{
		{"omitted resources", "", "1", "1"},
		{"omitted requests", `resources = [{ limits = { cpu = "0.5" } }]`, "0.5", "500m"},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := func(target string) string {
				return fmt.Sprintf(`
resource "podquantity_native" "test" {
  %s
  spec {
    container {
      name  = "app"
      image = "image:1"
      %s
    }
  }
}`, target, test.resources)
			}
			check := testresource.ComposeTestCheckFunc(
				testresource.TestCheckResourceAttr("podquantity_native.test", "spec.0.container.0.resources.0.limits.cpu", test.limits),
				testresource.TestCheckResourceAttr("podquantity_native.test", "spec.0.container.0.resources.0.requests.cpu", test.requests),
				testresource.TestCheckResourceAttr("podquantity_native.test", "spec.0.image_pull_secrets.0.name", "injected"),
				testresource.TestCheckResourceAttr("podquantity_native.test", "spec.0.readiness_gate.0.condition_type", "example.com/Ready"),
			)
			updated := config(`target_state = ["Pending"]`)
			testresource.UnitTest(t, testresource.TestCase{
				ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
					"podquantity": providerserver.NewProtocol6WithError(podSpecQuantityTestProvider{}),
				},
				Steps: []testresource.TestStep{
					{Config: config(""), Check: check},
					{
						Config: updated,
						ConfigPlanChecks: testresource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction("podquantity_native.test", plancheck.ResourceActionUpdate),
						}},
						Check: check,
					},
					{Config: updated, PlanOnly: true},
				},
			})
		})
	}
}
