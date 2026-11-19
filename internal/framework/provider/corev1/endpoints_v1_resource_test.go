// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/corev1"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const endpointsResourceName = "kubernetes_endpoints_v1.test"

type endpointsUpdateClientsets struct {
	kubernetes.KubeClientsets
	client *k8sclient.Clientset
}

func (clients endpointsUpdateClientsets) MainClientset() (*k8sclient.Clientset, error) {
	return clients.client, nil
}

func (endpointsUpdateClientsets) GetIgnoreAnnotations() []string { return nil }
func (endpointsUpdateClientsets) GetIgnoreLabels() []string      { return nil }

func TestEndpointsUpdateNoPatch(t *testing.T) {
	ctx := context.Background()
	for _, testCase := range []struct {
		name       string
		labels     types.Map
		subset     []corev1.EndpointsSubsetModel
		wantMethod string
		getFails   bool
	}{
		{
			name:       "no change",
			labels:     types.MapNull(types.StringType),
			subset:     nil,
			wantMethod: http.MethodGet,
			getFails:   false,
		},
		{
			name:       "metadata change",
			labels:     types.MapValueMust(types.StringType, map[string]attr.Value{"env": types.StringValue("test")}),
			subset:     nil,
			wantMethod: http.MethodPatch,
			getFails:   false,
		},
		{
			name:   "subset change",
			labels: types.MapNull(types.StringType),
			subset: []corev1.EndpointsSubsetModel{
				{
					Address: []corev1.EndpointsAddressModel{
						{
							IP:       types.StringValue("10.0.0.1"),
							Hostname: types.StringNull(),
							NodeName: types.StringNull(),
						},
					},
					NotReadyAddress: nil,
					Port: []corev1.EndpointsPortModel{
						{
							Name:     types.StringNull(),
							Port:     types.Int64Value(80),
							Protocol: types.StringValue("TCP"),
						},
					},
				},
			},
			wantMethod: http.MethodPatch,
			getFails:   false,
		},
		{
			name:       "GET error preserves state",
			labels:     types.MapNull(types.StringType),
			subset:     nil,
			wantMethod: http.MethodGet,
			getFails:   true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			requests := make(chan string, 4)
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				requests <- request.Method
				if request.URL.Path != "/api/v1/namespaces/default/endpoints/test" {
					t.Errorf("unexpected request path: %s", request.URL.Path)
				}
				if request.Method != testCase.wantMethod || testCase.getFails {
					http.Error(writer, "request forbidden", http.StatusForbidden)
					return
				}
				writer.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(writer).Encode(&api.Endpoints{
					TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Endpoints"},
					ObjectMeta: metav1.ObjectMeta{
						Name: "test", Namespace: "default", UID: "uid-1", ResourceVersion: "20", Generation: 2,
					},
				}); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()

			client, err := k8sclient.NewForConfig(&rest.Config{Host: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			endpoints := &corev1.EndpointsV1{SDKv2Meta: func() any {
				return endpointsUpdateClientsets{client: client}
			}}

			var schemaResponse fwresource.SchemaResponse
			endpoints.Schema(ctx, fwresource.SchemaRequest{}, &schemaResponse)
			if schemaResponse.Diagnostics.HasError() {
				t.Fatal(schemaResponse.Diagnostics)
			}

			raw := tfprotov6.RawState{JSON: []byte(`{
				"id":"default/test",
				"metadata":[{"annotations":null,"labels":null,"generate_name":null,
				"name":"test","namespace":"default","generation":1,"resource_version":"10","uid":"uid-1"}],
				"subset":null
			}`)}
			stateValue, err := raw.Unmarshal(schemaResponse.Schema.Type().TerraformType(ctx))
			if err != nil {
				t.Fatal(err)
			}
			state := tfsdk.State{Schema: schemaResponse.Schema, Raw: stateValue}

			var planModel corev1.EndpointsV1Model
			if diags := state.Get(ctx, &planModel); diags.HasError() {
				t.Fatal(diags)
			}
			planModel.Metadata[0].Labels = testCase.labels
			planModel.Metadata[0].Generation = types.Int64Unknown()
			planModel.Metadata[0].ResourceVersion = types.StringUnknown()
			planModel.Subset = testCase.subset

			planState := tfsdk.State{Schema: schemaResponse.Schema}
			if diags := planState.Set(ctx, &planModel); diags.HasError() {
				t.Fatal(diags)
			}

			identity := &tfsdk.ResourceIdentity{Schema: common.NamespacedIdentitySchema()}
			wantIdentity := common.NamespacedResourceIdentity{
				ResourceIdentity: common.ResourceIdentity{
					APIVersion: types.StringValue("v1"),
					Kind:       types.StringValue("Endpoints"),
					Name:       types.StringValue("test"),
				},
				Namespace: types.StringValue("default"),
			}
			if diags := identity.Set(ctx, wantIdentity); diags.HasError() {
				t.Fatal(diags)
			}

			response := fwresource.UpdateResponse{State: state, Identity: identity}
			endpoints.Update(ctx, fwresource.UpdateRequest{
				State: state,
				Plan:  tfsdk.Plan{Schema: schemaResponse.Schema, Raw: planState.Raw},
			}, &response)

			if len(requests) != 1 {
				t.Fatalf("made %d requests, want 1", len(requests))
			}
			if method := <-requests; method != testCase.wantMethod {
				t.Errorf("request method = %s, want %s", method, testCase.wantMethod)
			}
			if testCase.getFails {
				if !response.Diagnostics.HasError() || !response.State.Raw.Equal(state.Raw) {
					t.Fatalf("GET failure must preserve state and return diagnostics: %v", response.Diagnostics)
				}
				return
			}
			if response.Diagnostics.HasError() {
				t.Fatal(response.Diagnostics)
			}
			planModel.Metadata[0].Generation = types.Int64Value(2)
			planModel.Metadata[0].ResourceVersion = types.StringValue("20")
			if diags := planState.Set(ctx, &planModel); diags.HasError() {
				t.Fatal(diags)
			}
			if !response.State.Raw.Equal(planState.Raw) {
				t.Errorf("state = %s, want planned values with resolved computed fields: %s", response.State.Raw, planState.Raw)
			}
			var gotIdentity common.NamespacedResourceIdentity
			if diags := response.Identity.Get(ctx, &gotIdentity); diags.HasError() {
				t.Fatal(diags)
			}
			if gotIdentity != wantIdentity {
				t.Errorf("identity = %v, want %v", gotIdentity, wantIdentity)
			}
		})
	}
}

func TestAccKubernetesEndpointsV1_basic(t *testing.T) {
	var conf api.Endpoints
	resourceName := "kubernetes_endpoints_v1.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesEndpointV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesEndpointsV1_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesEndpointV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "subset.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.address.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.address.0.ip", "10.0.0.4"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.port.0.name", "httptransport"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.port.0.port", "80"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.port.0.protocol", "TCP"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesEndpointsV1_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesEndpointV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "subset.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.address.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.address.0.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.address.0.ip", "10.0.0.5"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.port.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.port.0.name", "httptransport"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.port.0.port", "82"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.port.0.protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.port.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.port.1.name", "httpstransport"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.port.1.port", "443"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.port.1.protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.port.2.name", "httpstransport2"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.port.2.port", "444"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.port.2.protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.port.0.name", "aaaa"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.port.0.port", "442"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.port.0.protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.address.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.address.0.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.address.0.ip", "10.0.0.7"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.address.1.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.address.1.ip", "10.0.0.6"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.address.1.hostname", "test-hostname"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.not_ready_address.0.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.not_ready_address.0.ip", "10.0.0.10"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.not_ready_address.1.ip", "10.0.0.11"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.not_ready_address.1.%", "3"),
				),
			},
			{
				Config: testAccKubernetesEndpointsV1_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesEndpointV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "subset.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.address.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.address.0.ip", "10.0.0.4"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.port.0.name", "httptransport"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.port.0.port", "80"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.port.0.protocol", "TCP"),
				),
			},
		},
	})
}

func TestAccKubernetesEndpointsV1_generatedName(t *testing.T) {
	var conf api.Endpoints
	prefix := "tf-acc-test-gen-"
	resourceName := "kubernetes_endpoints_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesEndpointV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesEndpointsV1_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesEndpointV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr(resourceName, "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
		},
	})
}

func testAccCheckKubernetesEndpointV1Destroy(s *terraform.State) error {
	client, err := mainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_endpoints_v1" && rs.Type != "kubernetes_endpoints" {
			continue
		}

		namespace, name, err := kubernetes.IdParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := client.CoreV1().Endpoints(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Endpoint still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesEndpointV1Exists(n string, obj *api.Endpoints) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		client, err := mainClientset()
		if err != nil {
			return err
		}
		ctx := context.TODO()

		namespace, name, err := kubernetes.IdParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		out, err := client.CoreV1().Endpoints(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesEndpointsV1_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_endpoints_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }
    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }
    name = "%s"
  }
  subset {
    address {
      ip = "10.0.0.4"
    }
    port {
      name     = "httptransport"
      port     = 80
      protocol = "TCP"
    }
  }
}
`, name)
}

func testAccKubernetesEndpointsV1_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_endpoints_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      Different         = "4424"
    }
    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
    }
    name = "%s"
  }
  subset {
    address {
      ip = "10.0.0.5"
    }
    port {
      name     = "httptransport"
      port     = 82
      protocol = "TCP"
    }
  }
  subset {
    address {
      ip        = "10.0.0.6"
      hostname  = "test-hostname"
      node_name = "test-nodename"
    }
    address {
      ip = "10.0.0.7"
    }
    not_ready_address {
      ip = "10.0.0.10"
    }
    not_ready_address {
      ip = "10.0.0.11"
    }
    port {
      name     = "httpstransport"
      port     = 443
      protocol = "TCP"
    }
    port {
      name     = "httpstransport2"
      port     = 444
      protocol = "TCP"
    }
    port {
      name     = "aaaa"
      port     = 442
      protocol = "TCP"
    }
  }
}
`, name)
}

func testAccKubernetesEndpointsV1_generatedName(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_endpoints_v1" "test" {
  metadata {
    generate_name = "%s"
  }

  subset {
    address {
      ip = "10.0.0.4"
    }

    port {
      name     = "transport"
      port     = 80
      protocol = "TCP"
    }
  }
}
`, prefix)
}
