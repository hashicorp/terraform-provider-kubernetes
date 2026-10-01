// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	api "k8s.io/api/core/v1"
	kquantity "k8s.io/apimachinery/pkg/api/resource"
)

// This in-memory resource tests Terraform Core's plan consistency, not Kubernetes
// lifecycle behavior. It never configures a Kubernetes client or creates a Pod.
func TestPodV1SpecQuantityPlanCore(t *testing.T) {
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"podquantity": providerserver.NewProtocol6WithError(podSpecQuantityTestProvider{}),
		},
		Steps: []testresource.TestStep{
			{Config: `resource "podquantity_test" "test" {
  limits  = { cpu = "1000m" }
  divisor = "1000m"
}`},
			{
				Config: `resource "podquantity_test" "test" {
  limits  = { cpu = "1" }
  divisor = "1"
}`,
				PlanOnly: true,
			},
			{
				Config: `resource "podquantity_test" "test" {
  limits  = { cpu = "2" }
  divisor = "3"
}`,
				ConfigPlanChecks: testresource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction("podquantity_test.test", plancheck.ResourceActionDestroyBeforeCreate),
				}},
			},
		},
	})
}

type podSpecQuantityTestProvider struct{}

func (podSpecQuantityTestProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "podquantity"
}
func (podSpecQuantityTestProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = providerschema.Schema{}
}
func (podSpecQuantityTestProvider) Configure(context.Context, provider.ConfigureRequest, *provider.ConfigureResponse) {
}
func (podSpecQuantityTestProvider) DataSources(context.Context) []func() datasource.DataSource {
	return nil
}
func (podSpecQuantityTestProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		func() resource.Resource { return podSpecQuantityTestResource{} },
		func() resource.Resource { return podSpecNativeTestResource{} },
		func() resource.Resource { return podSpecNativeTestResource{omitResources: true} },
		func() resource.Resource { return podSpecNativeTestResource{plainReferences: true} },
		func() resource.Resource { return podSpecNativeTestResource{nestedReferences: true} },
		func() resource.Resource { return podSpecCanonicalQuantityTestResource{} },
	}
}

type podSpecQuantityTestResource struct{}

func (podSpecQuantityTestResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "podquantity_test"
}
func (podSpecQuantityTestResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"id":      schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"limits":  podQuantityMap(false, true),
		"divisor": podQuantityString(false, true, "1"),
	}}
}
func (podSpecQuantityTestResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	resp.State.Raw = req.Plan.Raw
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue("test"))...)
}
func (podSpecQuantityTestResource) Read(context.Context, resource.ReadRequest, *resource.ReadResponse) {
}
func (podSpecQuantityTestResource) Update(_ context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.State.Raw = req.Plan.Raw
}
func (podSpecQuantityTestResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
}

func TestPodV1SpecNativeBlocksCore(t *testing.T) {
	for name, config := range map[string]string{
		"omitted computed collections": `
resource "podquantity_native" "test" {
  spec {
    container {
      name  = "app"
      image = "image:1"
    }
  }
}`,
		"assigned references and computed resource child": `
resource "podquantity_native" "test" {
  spec {
    container {
      name      = "app"
      image     = "image:1"
      resources = [{ limits = { cpu = "1" } }]
    }
    image_pull_secrets = [{ name = "registry" }]
    readiness_gate     = [{ condition_type = "example.com/Ready" }]
  }
}`,
		"resources object with omitted maps": `
resource "podquantity_native" "test" {
  spec {
    container {
      name      = "app"
      image     = "image:1"
      resources = [{}]
    }
  }
}`,
	} {
		t.Run(name, func(t *testing.T) {
			testresource.UnitTest(t, testresource.TestCase{
				ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
					"podquantity": providerserver.NewProtocol6WithError(podSpecQuantityTestProvider{}),
				},
				Steps: []testresource.TestStep{
					{
						Config: config,
						Check: testresource.ComposeTestCheckFunc(
							testresource.TestCheckResourceAttr("podquantity_native.test", "spec.0.container.0.resources.0.requests.cpu", "1"),
							testresource.TestCheckResourceAttr("podquantity_native.test", "spec.0.image_pull_secrets.#", "1"),
							testresource.TestCheckResourceAttr("podquantity_native.test", "spec.0.readiness_gate.#", "1"),
						),
					},
					{Config: config, PlanOnly: true},
				},
			})
		})
	}
}

func TestPodV1SpecResourceCardinalityCore(t *testing.T) {
	for _, block := range []string{"container", "init_container"} {
		for _, test := range []struct {
			name       string
			assignment string
			input      string
			wantError  *regexp.Regexp
		}{
			{"omitted", "", "", nil},
			{"null", "resources = null", "", nil},
			{"conditional omission", "resources = false ? [{}] : null", "", nil},
			{"unknown conditional", `resources = podquantity_test.input.id == "test" ? null : [{}]`, `resource "podquantity_test" "input" {}`, nil},
			{"empty object", "resources = [{}]", "", nil},
			{"empty list", "resources = []", "", regexp.MustCompile("Invalid Attribute Value")},
			{"multiple objects", "resources = [{}, {}]", "", regexp.MustCompile("Invalid Attribute Value")},
		} {
			t.Run(block+"/"+test.name, func(t *testing.T) {
				body := fmt.Sprintf(`
    %s {
      name = "app"
      image = "image:1"
      %s
    }`, block, test.assignment)
				if block == "init_container" {
					body += `
    container {
      name = "main"
      image = "image:1"
    }`
				}
				config := fmt.Sprintf(`
%s
resource "podquantity_native" "test" {
  spec {
    %s
  }
}`, test.input, body)
				step := testresource.TestStep{Config: config, ExpectError: test.wantError}
				steps := []testresource.TestStep{step}
				if test.wantError == nil {
					prefix := "spec.0." + block + ".0.resources"
					steps[0].Check = testresource.ComposeTestCheckFunc(
						testresource.TestCheckResourceAttr("podquantity_native.test", prefix+".#", "1"),
						testresource.TestCheckResourceAttr("podquantity_native.test", prefix+".0.limits.cpu", "1"),
						testresource.TestCheckResourceAttr("podquantity_native.test", prefix+".0.requests.cpu", "1"),
					)
					steps = append(steps, testresource.TestStep{Config: config, PlanOnly: true})
				}
				testresource.UnitTest(t, testresource.TestCase{
					ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
						"podquantity": providerserver.NewProtocol6WithError(podSpecQuantityTestProvider{}),
					},
					Steps: steps,
				})
			})
		}
	}
}

func TestPodV1SpecReferenceAssignmentsCore(t *testing.T) {
	for _, reference := range []struct{ name, child, value, injected string }{
		{"image_pull_secrets", "name", "registry", "injected"},
		{"readiness_gate", "condition_type", "example.com/Ready", "example.com/Ready"},
	} {
		singleton := fmt.Sprintf(`[{ %s = %q }]`, reference.child, reference.value)
		for _, test := range []struct {
			name      string
			value     string
			input     string
			wantCount string
			wantChild string
			wantError *regexp.Regexp
		}{
			{"omitted", "", "", "1", reference.injected, nil},
			{"null", "null", "", "1", reference.injected, nil},
			{"conditional omission", "false ? " + singleton + " : null", "", "1", reference.injected, nil},
			{"unknown parent", `podquantity_test.input.id == "test" ? null : ` + singleton, `resource "podquantity_test" "input" {}`, "1", reference.injected, nil},
			{"unknown child", `[{ ` + reference.child + ` = podquantity_test.input.id }]`, `resource "podquantity_test" "input" {}`, "1", "test", nil},
			{"configured", singleton, "", "1", reference.value, nil},
			{"multiple", fmt.Sprintf(`[{ %s = %q }, { %s = "second" }]`, reference.child, reference.value, reference.child), "", "2", reference.value, nil},
			{"missing child", "[{}]", "", "", "", regexp.MustCompile(`"` + reference.child + `" is required`)},
			{"null child", `[{ ` + reference.child + ` = null }]`, "", "", "", regexp.MustCompile("Missing Required Argument")},
			{"null object", "[null]", "", "", "", regexp.MustCompile("Missing Required Argument")},
			// Reject configured empty lists before admission can populate them.
			{"empty configured", "[]", "", "", "", regexp.MustCompile("list must contain at least 1 elements")},
		} {
			t.Run(reference.name+"/"+test.name, func(t *testing.T) {
				assignment := ""
				if test.value != "" {
					assignment = reference.name + " = " + test.value
				}
				config := fmt.Sprintf(`
%s
resource "podquantity_native" "test" {
  spec {
    container {
      name  = "app"
      image = "image:1"
    }
    %s
  }
}`, test.input, assignment)
				steps := []testresource.TestStep{{Config: config, ExpectError: test.wantError}}
				if test.wantError == nil {
					prefix := "spec.0." + reference.name
					steps[0].Check = testresource.ComposeTestCheckFunc(
						testresource.TestCheckResourceAttr("podquantity_native.test", prefix+".#", test.wantCount),
						testresource.TestCheckResourceAttr("podquantity_native.test", prefix+".0."+reference.child, test.wantChild),
					)
					steps = append(steps, testresource.TestStep{Config: config, PlanOnly: true})
				}
				testresource.UnitTest(t, testresource.TestCase{
					ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
						"podquantity": providerserver.NewProtocol6WithError(podSpecQuantityTestProvider{}),
					},
					Steps: steps,
				})
			})
		}
	}
}

func TestPodV1SpecReferenceReplacementCore(t *testing.T) {
	for name, child := range map[string]string{"image_pull_secrets": "name", "readiness_gate": "condition_type"} {
		t.Run(name, func(t *testing.T) {
			config := func(value string) string {
				template := `
resource "podquantity_native" "test" {
  spec {
    container {
      name  = "app"
      image = "image:1"
    }
    reference_collection = [{ reference_field = reference_value }]
  }
}`
				return strings.NewReplacer(
					"reference_collection", name,
					"reference_field", child,
					"reference_value", fmt.Sprintf("%q", value),
				).Replace(template)
			}
			testresource.UnitTest(t, testresource.TestCase{
				ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
					"podquantity": providerserver.NewProtocol6WithError(podSpecQuantityTestProvider{}),
				},
				Steps: []testresource.TestStep{
					{Config: config("original")},
					{Config: config("original"), PlanOnly: true},
					{
						Config: config("changed"),
						ConfigPlanChecks: testresource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction("podquantity_native.test", plancheck.ResourceActionDestroyBeforeCreate),
						}},
					},
				},
			})
		})
	}
}

// This deliberately unguarded alternative demonstrates Core's interpretation of
// required nested children in an omitted, admission-populated collection:
// https://github.com/hashicorp/terraform/blob/v1.15.7/internal/plans/objchange/objchange.go
// optionalValueNotComputable then drops an omitted parent from the proposed plan.
func TestPodV1SpecUnguardedNestedReferenceCoreBoundary(t *testing.T) {
	config := `
resource "podquantity_nestedrefs" "test" {
  spec {
    container {
      name  = "app"
      image = "image:1"
    }
  }
}`
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"podquantity": providerserver.NewProtocol6WithError(podSpecQuantityTestProvider{}),
		},
		Steps: []testresource.TestStep{{
			Config: config, ExpectNonEmptyPlan: true,
			ConfigPlanChecks: testresource.ConfigPlanChecks{PostApplyPreRefresh: []plancheck.PlanCheck{
				plancheck.ExpectResourceAction("podquantity_nestedrefs.test", plancheck.ResourceActionDestroyBeforeCreate),
			}},
		}},
	})
}

// Preserve the pre-conversion proof: plain object-list references accept blocks
// only when resources' NestedType is also removed. Core's recursive skipFixup
// disables compatibility decoding for the whole resource, not just that subtree.
// https://github.com/hashicorp/terraform/blob/v1.15.7/internal/lang/blocktoattr/fixup.go
func TestPodV1SpecLegacyReferencesCoreBoundary(t *testing.T) {
	config := `
resource "podquantity_native" "test" {
  spec {
    container {
      name  = "app"
      image = "image:1"
    }
    image_pull_secrets { name = "registry" }
    readiness_gate { condition_type = "example.com/Ready" }
  }
}`
	for _, test := range []struct {
		name      string
		config    string
		wantError *regexp.Regexp
	}{
		{"complete schema with plain references", strings.ReplaceAll(config, "podquantity_native", "podquantity_plainrefs"), regexp.MustCompile("Unsupported block type")},
		{"no nested attributes", strings.ReplaceAll(config, "podquantity_native", "podquantity_legacyrefs"), nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			testresource.UnitTest(t, testresource.TestCase{
				ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
					"podquantity": providerserver.NewProtocol6WithError(podSpecQuantityTestProvider{}),
				},
				Steps: []testresource.TestStep{{
					Config: test.config, PlanOnly: true, ExpectNonEmptyPlan: test.wantError == nil, ExpectError: test.wantError,
				}},
			})
		})
	}
}

type podSpecNativeTestResource struct {
	omitResources    bool
	plainReferences  bool
	nestedReferences bool
}

func (r podSpecNativeTestResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "podquantity_native"
	if r.omitResources {
		resp.TypeName = "podquantity_legacyrefs"
	} else if r.plainReferences {
		resp.TypeName = "podquantity_plainrefs"
	} else if r.nestedReferences {
		resp.TypeName = "podquantity_nestedrefs"
	}
}
func (r podSpecNativeTestResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	var podSchema resource.SchemaResponse
	(&PodV1{}).Schema(ctx, resource.SchemaRequest{}, &podSchema)
	object := podSpecObject()
	if r.nestedReferences {
		for name, child := range map[string]string{"image_pull_secrets": "name", "readiness_gate": "condition_type"} {
			object.Attributes[name] = schema.ListNestedAttribute{
				Optional: true, Computed: true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					child: schema.StringAttribute{Required: true},
				}},
				PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()},
			}
		}
	}
	if r.omitResources || r.plainReferences {
		for name, child := range map[string]string{"image_pull_secrets": "name", "readiness_gate": "condition_type"} {
			object.Attributes[name] = schema.ListAttribute{
				Optional: true, Computed: true,
				ElementType: types.ObjectType{AttrTypes: map[string]attr.Type{child: types.StringType}},
			}
		}
	}
	if r.omitResources {
		for _, name := range []string{"container", "init_container"} {
			container := object.Blocks[name].(schema.ListNestedBlock)
			delete(container.NestedObject.Attributes, "resources")
			object.Blocks[name] = container
		}
	}
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":           schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"target_state": podSchema.Schema.Attributes["target_state"],
		},
		Blocks: map[string]schema.Block{"spec": podBlock(object, 1, 1, false)},
	}
}
func (podSpecNativeTestResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var planned types.List
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec"), &planned)...)
	spec, diagnostics := expandPodV1Spec(ctx, planned)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Simulate admission defaults without using a real or fake clientset.
	if len(spec.ImagePullSecrets) == 0 {
		spec.ImagePullSecrets = []api.LocalObjectReference{{Name: "injected"}}
	}
	if len(spec.ReadinessGates) == 0 {
		spec.ReadinessGates = []api.PodReadinessGate{{ConditionType: "example.com/Ready"}}
	}
	for _, containers := range [][]api.Container{spec.Containers, spec.InitContainers} {
		for i := range containers {
			container := &containers[i]
			container.ImagePullPolicy = api.PullIfNotPresent
			container.TerminationMessagePolicy = api.TerminationMessageReadFile
			if len(container.Resources.Limits) == 0 {
				container.Resources.Limits = api.ResourceList{api.ResourceCPU: kquantity.MustParse("1")}
			}
			if len(container.Resources.Requests) == 0 {
				container.Resources.Requests = container.Resources.Limits.DeepCopy()
			}
		}
	}
	value, diagnostics := flattenPodV1Spec(ctx, *spec, planned)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.State.Raw = req.Plan.Raw
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue("test"))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec"), value)...)
}
func (podSpecNativeTestResource) Read(context.Context, resource.ReadRequest, *resource.ReadResponse) {
}
func (podSpecNativeTestResource) Update(_ context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.State.Raw = req.Plan.Raw
}
func (podSpecNativeTestResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
}

func TestPodV1SpecTargetStateCore(t *testing.T) {
	for _, test := range []struct {
		name       string
		assignment string
		wantCount  string
		wantError  *regexp.Regexp
	}{
		{"omitted", "", "0", nil},
		{"null", "target_state = null", "0", nil},
		{"configured", `target_state = ["Succeeded", "Failed"]`, "2", nil},
		{"empty configured", "target_state = []", "", regexp.MustCompile("Invalid Attribute Value")},
		{"invalid phase", `target_state = ["Invalid"]`, "", regexp.MustCompile("Invalid Attribute Value")},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := fmt.Sprintf(`
resource "podquantity_native" "test" {
  %s
  spec {
    container {
      name  = "app"
      image = "image:1"
    }

  }
}`, test.assignment)
			step := testresource.TestStep{Config: config, ExpectError: test.wantError}
			steps := []testresource.TestStep{step}
			if test.wantError == nil {
				steps[0].Check = testresource.TestCheckResourceAttr("podquantity_native.test", "target_state.#", test.wantCount)
				steps = append(steps, testresource.TestStep{Config: config, PlanOnly: true})
			}
			testresource.UnitTest(t, testresource.TestCase{
				ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
					"podquantity": providerserver.NewProtocol6WithError(podSpecQuantityTestProvider{}),
				},
				Steps: steps,
			})
		})
	}
}
