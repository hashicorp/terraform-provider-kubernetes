// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
)

func podSpecProtocolBlock(t *testing.T, block *tfprotov6.SchemaBlock, name string) *tfprotov6.SchemaBlock {
	t.Helper()
	for _, nested := range block.BlockTypes {
		if nested.TypeName == name {
			return nested.Block
		}
	}
	t.Fatalf("protocol schema has no %s block", name)
	return nil
}

func podSpecProtocolAttribute(t *testing.T, attributes []*tfprotov6.SchemaAttribute, name string) *tfprotov6.SchemaAttribute {
	t.Helper()
	for _, attribute := range attributes {
		if attribute.Name == name {
			return attribute
		}
	}
	t.Fatalf("protocol schema has no %s attribute", name)
	return nil
}

func TestPodV1SpecNativeGetSchemaReferences(t *testing.T) {
	ctx := context.Background()
	fullMux, err := mux.MuxServer(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	for name, server := range map[string]tfprotov6.ProviderServer{
		"framework": providerserver.NewProtocol6(provider.New("test", nil))(),
		"full mux":  fullMux,
	} {
		t.Run(name, func(t *testing.T) {
			response, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
			if err != nil {
				t.Fatal(err)
			}
			for _, diagnostic := range response.Diagnostics {
				if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
					t.Fatalf("%s: %s", diagnostic.Summary, diagnostic.Detail)
				}
			}
			pod := response.ResourceSchemas["kubernetes_pod_v1"]
			if pod == nil || pod.Version != 1 {
				t.Fatal("expected native Pod schema version 1")
			}
			spec := podSpecProtocolBlock(t, pod.Block, "spec")
			for _, reference := range []struct{ name, child string }{
				{"readiness_gate", "condition_type"},
				{"image_pull_secrets", "name"},
			} {
				attribute := podSpecProtocolAttribute(t, spec.Attributes, reference.name)
				want := tftypes.List{ElementType: tftypes.Object{
					AttributeTypes: map[string]tftypes.Type{reference.child: tftypes.String},
				}}
				if attribute.NestedType != nil || attribute.Type == nil || !attribute.Type.Equal(want) ||
					!attribute.Optional || !attribute.Computed || attribute.Required {
					t.Fatalf("%s must be a plain optional-computed list(object), got %#v", reference.name, attribute)
				}
			}
			for _, container := range []string{"container", "init_container"} {
				block := podSpecProtocolBlock(t, spec, container)
				resources := podSpecProtocolAttribute(t, block.Attributes, "resources")
				if resources.NestedType == nil || resources.Type != nil ||
					resources.NestedType.Nesting != tfprotov6.SchemaObjectNestingModeList ||
					!resources.Optional || !resources.Computed {
					t.Fatalf("%s.resources must expose a computed nested list, got %#v", container, resources)
				}
				for _, child := range []string{"limits", "requests"} {
					attribute := podSpecProtocolAttribute(t, resources.NestedType.Attributes, child)
					if !attribute.Optional || !attribute.Computed {
						t.Fatalf("%s.resources.%s must be optional-computed", container, child)
					}
				}
			}
		})
	}
}

// Original blocks still pass for the SDK alias, but the authorized native schema
// requires list assignments. Both before/after configurations are retained here.
func TestPodV1SpecFullMuxLegacyReferenceBlockBoundary(t *testing.T) {
	for _, reference := range []struct{ name, config string }{
		{"readiness gate", `readiness_gate { condition_type = "example.com/Ready" }`},
		{"image pull secrets", `image_pull_secrets { name = "registry" }`},
		{"both", `readiness_gate { condition_type = "example.com/Ready" }
    image_pull_secrets { name = "registry" }`},
	} {
		for _, resourceType := range []string{"kubernetes_pod", "kubernetes_pod_v1"} {
			t.Run(reference.name+"/"+resourceType, func(t *testing.T) {
				config := fmt.Sprintf(`
provider "kubernetes" {
  host = "http://127.0.0.1:1"
}
resource "%s" "test" {
  metadata {
    name = "offline-schema-probe"
    namespace = "default"
  }
  spec {
    container {
      name = "app"
      image = "image:1"
    }
    %s
  }
}`, resourceType, reference.config)
				step := testresource.TestStep{Config: config, PlanOnly: true}
				if strings.HasSuffix(resourceType, "_v1") {
					step.ExpectError = regexp.MustCompile("Unsupported block type")
				} else {
					step.ExpectNonEmptyPlan = true
				}
				testresource.UnitTest(t, testresource.TestCase{
					ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
						"kubernetes": func() (tfprotov6.ProviderServer, error) {
							return mux.MuxServer(context.Background(), "test")
						},
					},
					Steps: []testresource.TestStep{step},
				})
			})
		}
	}
}

func TestPodV1SpecFullMuxReferenceAssignments(t *testing.T) {
	for _, reference := range []struct{ name, config string }{
		{"readiness gate", `readiness_gate = [{ condition_type = "example.com/Ready" }]`},
		{"image pull secrets", `image_pull_secrets = [{ name = "registry" }]`},
		{"both", `readiness_gate = [{ condition_type = "example.com/Ready" }]
	    image_pull_secrets = [{ name = "registry" }]`},
		{"multiple", `readiness_gate = [{ condition_type = "example.com/Ready" }, { condition_type = "example.com/Healthy" }]
	    image_pull_secrets = [{ name = "registry" }, { name = "second" }]`},
	} {
		t.Run(reference.name, func(t *testing.T) {
			config := fmt.Sprintf(`
provider "kubernetes" {
  host = "http://127.0.0.1:1"
}
resource "kubernetes_pod_v1" "test" {
  metadata {
    name      = "offline-schema-probe"
    namespace = "default"
  }
  spec {
    container {
      name  = "app"
      image = "image:1"
    }
	    %s
  }
}`, reference.config)
			testresource.UnitTest(t, testresource.TestCase{
				ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
					"kubernetes": func() (tfprotov6.ProviderServer, error) {
						return mux.MuxServer(context.Background(), "test")
					},
				},
				Steps: []testresource.TestStep{{Config: config, PlanOnly: true, ExpectNonEmptyPlan: true}},
			})
		})
	}
}
