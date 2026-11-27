// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

// These tests exercise the deprecated core/v1 Endpoints API; SA1019 is expected here.
//
//nolint:staticcheck
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
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/corev1"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	api "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
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

// TestEndpointsReadMissingAndErrors covers Read paths acceptance tests do not reach.
// State upgraded from SDKv2 and Terraform before 1.12 carry no identity, so removing a
// missing Endpoints must set identity first; other API errors must keep state and identity.
func TestEndpointsReadMissingAndErrors(t *testing.T) {
	ctx := context.Background()
	for _, status := range []int{http.StatusNotFound, http.StatusForbidden, http.StatusInternalServerError} {
		for _, storedIdentity := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/identity=%t", status, storedIdentity), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					if req.Method != http.MethodGet || req.URL.Path != "/api/v1/namespaces/team/endpoints/web" {
						t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(status)
					if err := json.NewEncoder(w).Encode(metav1.Status{
						TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Status"},
						Status:   metav1.StatusFailure, Code: int32(status), Message: http.StatusText(status),
					}); err != nil {
						t.Error(err)
					}
				}))
				defer server.Close()
				conn, err := k8sclient.NewForConfig(&rest.Config{Host: server.URL})
				if err != nil {
					t.Fatal(err)
				}
				r := &corev1.EndpointsV1{SDKv2Meta: func() any { return endpointsUpdateClientsets{client: conn} }}
				var schemaResponse fwresource.SchemaResponse
				r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResponse)
				raw := tfprotov6.RawState{JSON: []byte(`{
					"id":"team/web",
					"metadata":[{"name":"web","namespace":"team","uid":"uid-1","resource_version":"10",
					"generation":1,"labels":null,"annotations":null,"generate_name":null}],
					"subset":[{"address":[{"ip":"10.0.0.4","hostname":null,"node_name":null}],
					"not_ready_address":null,"port":[{"name":null,"port":80,"protocol":"TCP"}]}]
				}`)}
				value, err := raw.Unmarshal(schemaResponse.Schema.Type().TerraformType(ctx))
				if err != nil {
					t.Fatal(err)
				}
				state := tfsdk.State{Schema: schemaResponse.Schema, Raw: value}
				identitySchema := common.NamespacedIdentitySchema()
				identity := tfsdk.ResourceIdentity{
					Schema: identitySchema, Raw: tftypes.NewValue(identitySchema.Type().TerraformType(ctx), nil),
				}
				wantIdentity := common.NamespacedResourceIdentity{
					ResourceIdentity: common.ResourceIdentity{
						Name: types.StringValue("web"), Kind: types.StringValue("Endpoints"),
						APIVersion: types.StringValue("v1"),
					},
					Namespace: types.StringValue("team"),
				}
				if storedIdentity {
					if status == http.StatusNotFound {
						// A distinct value proves an existing identity is not rebuilt.
						wantIdentity.Name = types.StringValue("existing-web")
					}
					if diags := identity.Set(ctx, wantIdentity); diags.HasError() {
						t.Fatal(diags)
					}
				}
				responseIdentity := identity
				response := fwresource.ReadResponse{State: state, Identity: &responseIdentity}
				r.Read(ctx, fwresource.ReadRequest{State: state, Identity: &identity}, &response)
				if status == http.StatusNotFound {
					if response.Diagnostics.HasError() || !response.State.Raw.IsNull() {
						t.Fatalf("missing Endpoints must be removed: %v", response.Diagnostics)
					}
					var got common.NamespacedResourceIdentity
					if diags := response.Identity.Get(ctx, &got); diags.HasError() {
						t.Fatal(diags)
					}
					if got != wantIdentity {
						t.Fatalf("identity = %#v, want %#v", got, wantIdentity)
					}
					return
				}
				if !response.Diagnostics.HasError() || !response.State.Raw.Equal(state.Raw) || !response.Identity.Raw.Equal(identity.Raw) {
					t.Fatalf("API failure must preserve state and identity: %v", response.Diagnostics)
				}
			})
		}
	}
}

// TestEndpointsCreateNamespace checks that Create sends the planned namespace unchanged,
// as SDKv2 did. An explicit namespace = "" must reach the API's non-namespaced path and
// fail there, not be redirected to "default", which created an object the plan did not
// describe and failed with "Provider produced inconsistent result after apply".
func TestEndpointsCreateNamespace(t *testing.T) {
	ctx := context.Background()
	for _, testCase := range []struct {
		name      string
		namespace string
		wantPath  string
		wantError bool
	}{
		{name: "explicit empty namespace", namespace: "", wantPath: "/api/v1/endpoints", wantError: true},
		{name: "configured namespace", namespace: "custom", wantPath: "/api/v1/namespaces/custom/endpoints", wantError: false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			requests := make(chan string, 2)
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				requests <- request.Method + " " + request.URL.Path
				if request.URL.Path == "/api/v1/endpoints" {
					http.Error(writer, "the server does not allow this method on the requested resource", http.StatusMethodNotAllowed)
					return
				}
				writer.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(writer).Encode(&api.Endpoints{
					TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Endpoints"},
					ObjectMeta: metav1.ObjectMeta{
						Name: "test", Namespace: testCase.namespace, UID: "uid-1", ResourceVersion: "1",
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

			raw := tfprotov6.RawState{JSON: []byte(fmt.Sprintf(`{
				"id":null,
				"metadata":[{"annotations":null,"labels":null,"generate_name":null,
				"name":"test","namespace":%q,"generation":null,"resource_version":null,"uid":null}],
				"subset":null
			}`, testCase.namespace))}
			planValue, err := raw.Unmarshal(schemaResponse.Schema.Type().TerraformType(ctx))
			if err != nil {
				t.Fatal(err)
			}

			response := fwresource.CreateResponse{
				State:    tfsdk.State{Schema: schemaResponse.Schema, Raw: tftypes.NewValue(schemaResponse.Schema.Type().TerraformType(ctx), nil)},
				Identity: &tfsdk.ResourceIdentity{Schema: common.NamespacedIdentitySchema()},
			}
			endpoints.Create(ctx, fwresource.CreateRequest{
				Plan: tfsdk.Plan{Schema: schemaResponse.Schema, Raw: planValue},
			}, &response)

			if len(requests) != 1 {
				t.Fatalf("made %d requests, want 1", len(requests))
			}
			if request := <-requests; request != http.MethodPost+" "+testCase.wantPath {
				t.Errorf("request = %q, want %q", request, http.MethodPost+" "+testCase.wantPath)
			}

			if testCase.wantError {
				if !response.Diagnostics.HasError() {
					t.Fatal("expected Create to fail for an explicit empty namespace")
				}
				if !response.State.Raw.IsNull() {
					t.Errorf("failed Create must not write state, got %s", response.State.Raw)
				}
				return
			}
			if response.Diagnostics.HasError() {
				t.Fatal(response.Diagnostics)
			}
			var got corev1.EndpointsV1Model
			if diags := response.State.Get(ctx, &got); diags.HasError() {
				t.Fatal(diags)
			}
			if got.ID.ValueString() != testCase.namespace+"/test" || got.Metadata[0].Namespace.ValueString() != testCase.namespace {
				t.Errorf("state id = %q, namespace = %q, want %q and %q",
					got.ID.ValueString(), got.Metadata[0].Namespace.ValueString(), testCase.namespace+"/test", testCase.namespace)
			}
		})
	}
}

// TestEndpointsDelete covers the initial Delete call: success and an already-deleted
// Endpoints (404) both succeed, as in SDKv2, while other API errors are reported.
func TestEndpointsDelete(t *testing.T) {
	ctx := context.Background()
	for _, testCase := range []struct {
		status    int
		wantError bool
	}{
		{status: http.StatusOK, wantError: false},
		{status: http.StatusNotFound, wantError: false},
		{status: http.StatusForbidden, wantError: true},
		{status: http.StatusInternalServerError, wantError: true},
	} {
		t.Run(fmt.Sprintf("%d", testCase.status), func(t *testing.T) {
			requests := make(chan string, 2)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				requests <- req.Method + " " + req.URL.Path
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(testCase.status)
				result := metav1.StatusSuccess
				if testCase.status != http.StatusOK {
					result = metav1.StatusFailure
				}
				if err := json.NewEncoder(w).Encode(metav1.Status{
					TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Status"},
					Status:   result, Code: int32(testCase.status), Message: http.StatusText(testCase.status),
				}); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()

			conn, err := k8sclient.NewForConfig(&rest.Config{Host: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			r := &corev1.EndpointsV1{SDKv2Meta: func() any { return endpointsUpdateClientsets{client: conn} }}
			var schemaResponse fwresource.SchemaResponse
			r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResponse)
			raw := tfprotov6.RawState{JSON: []byte(`{
				"id":"team/web",
				"metadata":[{"name":"web","namespace":"team","uid":"uid-1","resource_version":"10",
				"generation":1,"labels":null,"annotations":null,"generate_name":null}],
				"subset":null
			}`)}
			value, err := raw.Unmarshal(schemaResponse.Schema.Type().TerraformType(ctx))
			if err != nil {
				t.Fatal(err)
			}
			state := tfsdk.State{Schema: schemaResponse.Schema, Raw: value}

			response := fwresource.DeleteResponse{State: state}
			r.Delete(ctx, fwresource.DeleteRequest{State: state}, &response)

			if len(requests) != 1 {
				t.Fatalf("made %d requests, want 1", len(requests))
			}
			if request := <-requests; request != http.MethodDelete+" /api/v1/namespaces/team/endpoints/web" {
				t.Errorf("request = %q, want DELETE /api/v1/namespaces/team/endpoints/web", request)
			}
			if got := response.Diagnostics.HasError(); got != testCase.wantError {
				t.Fatalf("Delete error = %t, want %t: %v", got, testCase.wantError, response.Diagnostics)
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
					// subset, address, not_ready_address and port are sets: Terraform orders their
					// elements by value, so assert membership rather than positional indexes.
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "subset.*", map[string]string{
						"address.#": "1",
						"port.#":    "1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "subset.*", map[string]string{
						"address.#":           "2",
						"not_ready_address.#": "2",
						"port.#":              "3",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "subset.*.address.*", map[string]string{
						"ip": "10.0.0.5",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "subset.*.address.*", map[string]string{
						"ip":        "10.0.0.6",
						"hostname":  "test-hostname",
						"node_name": "test-nodename",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "subset.*.address.*", map[string]string{
						"ip": "10.0.0.7",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "subset.*.not_ready_address.*", map[string]string{
						"ip": "10.0.0.10",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "subset.*.not_ready_address.*", map[string]string{
						"ip": "10.0.0.11",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "subset.*.port.*", map[string]string{
						"name": "httptransport", "port": "82", "protocol": "TCP",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "subset.*.port.*", map[string]string{
						"name": "httpstransport", "port": "443", "protocol": "TCP",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "subset.*.port.*", map[string]string{
						"name": "httpstransport2", "port": "444", "protocol": "TCP",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "subset.*.port.*", map[string]string{
						"name": "aaaa", "port": "442", "protocol": "TCP",
					}),
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
		if apierrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("checking destruction of Endpoints %q: %w", rs.Primary.ID, err)
		}
		return fmt.Errorf("Endpoints %q still exists (uid %s)", rs.Primary.ID, resp.UID)
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

// TestAccKubernetesEndpointsV1_explicitEmptyStrings verifies that explicitly empty
// hostname, node_name and port name values converge. The API omits them, so without
// preserving the configured "" every plan would show an in-place update.
func TestAccKubernetesEndpointsV1_explicitEmptyStrings(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-test")
	checkUID := testAccCheckEndpointsSameUID()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesEndpointV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesEndpointsV1_explicitEmptyStrings(name, 80),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPreRefresh:  []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkUID,
					resource.TestCheckTypeSetElemNestedAttrs(endpointsResourceName, "subset.*.address.*", map[string]string{
						"ip": "10.0.0.4", "hostname": "", "node_name": "",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(endpointsResourceName, "subset.*.not_ready_address.*", map[string]string{
						"ip": "10.0.0.5", "hostname": "",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(endpointsResourceName, "subset.*.port.*", map[string]string{
						"name": "", "port": "80", "protocol": "TCP",
					}),
				),
			},
			{
				Config: testAccKubernetesEndpointsV1_explicitEmptyStrings(name, 80),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: testAccKubernetesEndpointsV1_explicitEmptyStrings(name, 81),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(endpointsResourceName, plancheck.ResourceActionUpdate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkUID,
					resource.TestCheckTypeSetElemNestedAttrs(endpointsResourceName, "subset.*.port.*", map[string]string{
						"name": "", "port": "81", "protocol": "TCP",
					}),
				),
			},
		},
	})
}

// TestAccKubernetesEndpointsV1_protocolDefault changes protocol from UDP to omitted,
// which must plan the schema default TCP as an in-place update.
func TestAccKubernetesEndpointsV1_protocolDefault(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-test")
	checkUID := testAccCheckEndpointsSameUID()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesEndpointV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesEndpointsV1_protocol(name, `protocol = "UDP"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkUID,
					resource.TestCheckTypeSetElemNestedAttrs(endpointsResourceName, "subset.*.port.*", map[string]string{
						"port": "53", "protocol": "UDP",
					}),
				),
			},
			{
				Config: testAccKubernetesEndpointsV1_protocol(name, ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(endpointsResourceName, plancheck.ResourceActionUpdate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkUID,
					resource.TestCheckTypeSetElemNestedAttrs(endpointsResourceName, "subset.*.port.*", map[string]string{
						"port": "53", "protocol": "TCP",
					}),
					testAccCheckEndpointsLive(name, func(ep *api.Endpoints) error {
						if got := ep.Subsets[0].Ports[0].Protocol; got != api.ProtocolTCP {
							return fmt.Errorf("live protocol = %q, want TCP", got)
						}
						return nil
					}),
				),
			},
		},
	})
}

// TestAccKubernetesEndpointsV1_subsetRemoval removes every subset in place and adds
// one back, converging after each step.
func TestAccKubernetesEndpointsV1_subsetRemoval(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-test")
	checkUID := testAccCheckEndpointsSameUID()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesEndpointV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesEndpointsV1_basic(name),
				Check:  resource.ComposeAggregateTestCheckFunc(checkUID, resource.TestCheckResourceAttr(endpointsResourceName, "subset.#", "1")),
			},
			{
				Config: testAccKubernetesEndpointsV1_noSubset(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(endpointsResourceName, plancheck.ResourceActionUpdate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkUID,
					resource.TestCheckResourceAttr(endpointsResourceName, "subset.#", "0"),
					testAccCheckEndpointsLive(name, func(ep *api.Endpoints) error {
						if len(ep.Subsets) != 0 {
							return fmt.Errorf("live subsets = %v, want none", ep.Subsets)
						}
						return nil
					}),
				),
			},
			{
				Config: testAccKubernetesEndpointsV1_basic(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(endpointsResourceName, plancheck.ResourceActionUpdate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(checkUID, resource.TestCheckResourceAttr(endpointsResourceName, "subset.#", "1")),
			},
		},
	})
}

// TestAccKubernetesEndpointsV1_disappears deletes the Endpoints out of band; refresh
// must remove it from state and plan a create.
func TestAccKubernetesEndpointsV1_disappears(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesEndpointV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesEndpointsV1_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(endpointsResourceName, "metadata.0.name", name),
					testAccEndpointsDeleteOutOfBand(name),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccKubernetesEndpointsV1_basic(name),
				Check:  resource.TestCheckResourceAttr(endpointsResourceName, "metadata.0.name", name),
			},
		},
	})
}

// TestAccKubernetesEndpointsV1_validation covers plan-time validation inherited from the
// SDKv2 schema and the documented tightenings.
func TestAccKubernetesEndpointsV1_validation(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-test")
	subset := `
  subset {
    address {
      ip = "10.0.0.4"
    }
  }`

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      `resource "kubernetes_endpoints_v1" "test" {` + subset + "\n}\n",
				ExpectError: regexp.MustCompile(`(?s)Block metadata must have a configuration value.*as required`),
			},
			{
				Config: fmt.Sprintf(`resource "kubernetes_endpoints_v1" "test" {
  metadata {
    name = %q
  }
  metadata {
    name = %q
  }%s
}
`, name, name, subset),
				ExpectError: regexp.MustCompile(`list must contain at most 1 elements`),
			},
			{
				Config: fmt.Sprintf(`resource "kubernetes_endpoints_v1" "test" {
  metadata {
    name          = %q
    generate_name = "tf-acc-"
  }%s
}
`, name, subset),
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
			{
				Config: fmt.Sprintf(`resource "kubernetes_endpoints_v1" "test" {
  metadata {
    name   = %q
    labels = { owner = null }
  }%s
}
`, name, subset),
				ExpectError: regexp.MustCompile(`value must be a string`),
			},
			{
				Config: fmt.Sprintf(`resource "kubernetes_endpoints_v1" "test" {
  metadata {
    name = %q
  }
  subset {
    port {
      name = "http"
    }
  }
}
`, name),
				ExpectError: regexp.MustCompile(`(?s)Missing required argument.*"port"`),
			},
		},
	})
}

// TestAccKubernetesEndpointsV1_emptySubset matches SDKv2: an empty subset {} block is
// not sent to Kubernetes and does not cause a diff, and a subset with only ports is
// rejected by Kubernetes during apply.
func TestAccKubernetesEndpointsV1_emptySubset(t *testing.T) {
	var conf api.Endpoints
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_endpoints_v1.test"
	config := func(subsets string) string {
		return fmt.Sprintf(`resource "kubernetes_endpoints_v1" "test" {
  metadata {
    name = %q
  }
%s
}
`, name, subsets)
	}
	populated := `
  subset {
    address {
      ip = "10.0.0.4"
    }
    port {
      port = 80
    }
  }`

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesEndpointV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: config(populated + "\n  subset {}"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesEndpointV1Exists(resourceName, &conf),
					func(*terraform.State) error {
						if len(conf.Subsets) != 1 {
							return fmt.Errorf("expected 1 subset in Kubernetes, got %d", len(conf.Subsets))
						}
						return nil
					},
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: config("  subset {}"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesEndpointV1Exists(resourceName, &conf),
					func(*terraform.State) error {
						if len(conf.Subsets) != 0 {
							return fmt.Errorf("expected no subsets in Kubernetes, got %d", len(conf.Subsets))
						}
						return nil
					},
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: config(`
  subset {
    port {
      port = 80
    }
  }`),
				ExpectError: regexp.MustCompile("must specify `addresses` or `notReadyAddresses`"),
			},
		},
	})
}

// TestAccKubernetesEndpointsV1_namespace covers Endpoints outside the default namespace:
// import by namespace/name, and replacement when metadata.namespace changes.
func TestAccKubernetesEndpointsV1_namespace(t *testing.T) {
	var conf api.Endpoints
	name := acctest.RandomWithPrefix("tf-acc-test")
	nsA, nsB := name+"-a", name+"-b"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesEndpointV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesEndpointsV1_inNamespace(name, nsA, nsB, "a"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesEndpointV1Exists(endpointsResourceName, &conf),
					resource.TestCheckResourceAttr(endpointsResourceName, "id", nsA+"/"+name),
					resource.TestCheckResourceAttr(endpointsResourceName, "metadata.0.namespace", nsA),
				),
			},
			{
				ResourceName:            endpointsResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesEndpointsV1_inNamespace(name, nsA, nsB, "b"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(endpointsResourceName, plancheck.ResourceActionDestroyBeforeCreate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesEndpointV1Exists(endpointsResourceName, &conf),
					resource.TestCheckResourceAttr(endpointsResourceName, "id", nsB+"/"+name),
					resource.TestCheckResourceAttr(endpointsResourceName, "metadata.0.namespace", nsB),
					func(*terraform.State) error {
						client, err := mainClientset()
						if err != nil {
							return err
						}
						_, err = client.CoreV1().Endpoints(nsA).Get(context.Background(), name, metav1.GetOptions{})
						if !apierrors.IsNotFound(err) {
							return fmt.Errorf("expected Endpoints %s/%s to be deleted after replacement, got: %v", nsA, name, err)
						}
						return nil
					},
				),
			},
		},
	})
}

func TestAccKubernetesEndpointsV1_identity(t *testing.T) {
	for _, tc := range []struct {
		name      string
		namespace func(name string) (config, namespace string)
	}{
		{name: "default namespace", namespace: func(name string) (string, string) {
			return testAccKubernetesEndpointsV1_basic(name), metav1.NamespaceDefault
		}},
		{name: "other namespace", namespace: func(name string) (string, string) {
			ns := name + "-ns"
			return testAccKubernetesEndpointsV1_inNamespace(name, ns, ns+"-b", "a"), ns
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name := acctest.RandomWithPrefix("tf-acc-test")
			config, namespace := tc.namespace(name)

			resource.ParallelTest(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				CheckDestroy:             testAccCheckKubernetesEndpointV1Destroy,
				TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)},
				Steps: []resource.TestStep{
					{
						Config:            config,
						ConfigStateChecks: []statecheck.StateCheck{testAccEndpointsExpectIdentity(namespace, name)},
					},
					{
						// Import blocks already assert a no-op plan and unchanged identity.
						ResourceName:    endpointsResourceName,
						ImportState:     true,
						ImportStateKind: resource.ImportBlockWithResourceIdentity,
					},
				},
			})
		})
	}
}

// TestAccKubernetesEndpointsV1_identityImportDefaultNamespace verifies that an identity
// import without namespace targets the default namespace, as for kubernetes_role_binding_v1.
func TestAccKubernetesEndpointsV1_identityImportDefaultNamespace(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesEndpointV1Destroy,
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)},
		Steps: []resource.TestStep{
			{Config: testAccKubernetesEndpointsV1_basic(name)},
			{
				ResourceName:           endpointsResourceName,
				ImportState:            true,
				ImportStateKind:        resource.ImportBlockWithResourceIdentity,
				ImportStateConfigExact: true,
				Config:                 testAccKubernetesEndpointsV1_identityImport(name, "Endpoints", "v1"),
			},
			{
				Config:            testAccKubernetesEndpointsV1_basic(name),
				ConfigPlanChecks:  resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
				ConfigStateChecks: []statecheck.StateCheck{testAccEndpointsExpectIdentity(metav1.NamespaceDefault, name)},
			},
		},
	})
}

func TestAccKubernetesEndpointsV1_importInvalidIdentity(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesEndpointV1Destroy,
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)},
		Steps: []resource.TestStep{
			{Config: testAccKubernetesEndpointsV1_basic(name)},
			{
				ResourceName: endpointsResourceName, ImportState: true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity, ImportStateConfigExact: true,
				Config:      testAccKubernetesEndpointsV1_identityImport(name, "EndpointSlice", "v1"),
				ExpectError: regexp.MustCompile("invalid identity import"),
			},
			{
				ResourceName: endpointsResourceName, ImportState: true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity, ImportStateConfigExact: true,
				Config:      testAccKubernetesEndpointsV1_identityImport(name, "Endpoints", "discovery.k8s.io/v1"),
				ExpectError: regexp.MustCompile("invalid identity import"),
			},
		},
	})
}

func testAccEndpointsExpectIdentity(namespace, name string) statecheck.StateCheck {
	return statecheck.ExpectIdentity(endpointsResourceName, map[string]knownvalue.Check{
		"api_version": knownvalue.StringExact("v1"),
		"kind":        knownvalue.StringExact("Endpoints"),
		"name":        knownvalue.StringExact(name),
		"namespace":   knownvalue.StringExact(namespace),
	})
}

// testAccCheckEndpointsSameUID records the UID on its first call and fails if a later
// step sees a different one, i.e. the Endpoints was replaced instead of updated.
func testAccCheckEndpointsSameUID() resource.TestCheckFunc {
	var prior string
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[endpointsResourceName]
		if !ok {
			rs, ok = state.RootModule().Resources["kubernetes_endpoints.test"]
		}
		if !ok {
			return fmt.Errorf("missing Endpoints test resource")
		}
		uid := rs.Primary.Attributes["metadata.0.uid"]
		if uid == "" {
			return fmt.Errorf("Endpoints has no uid in state")
		}
		if prior != "" && uid != prior {
			return fmt.Errorf("Endpoints replaced: uid %q became %q", prior, uid)
		}
		prior = uid
		return nil
	}
}

func testAccCheckEndpointsLive(name string, check func(*api.Endpoints) error) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		client, err := mainClientset()
		if err != nil {
			return err
		}
		ep, err := client.CoreV1().Endpoints(metav1.NamespaceDefault).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		return check(ep)
	}
}

func testAccEndpointsDeleteOutOfBand(name string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		client, err := mainClientset()
		if err != nil {
			return err
		}
		return client.CoreV1().Endpoints(metav1.NamespaceDefault).Delete(context.Background(), name, metav1.DeleteOptions{})
	}
}

func testAccKubernetesEndpointsV1_explicitEmptyStrings(name string, port int) string {
	return fmt.Sprintf(`resource "kubernetes_endpoints_v1" "test" {
  metadata {
    name = %q
  }
  subset {
    address {
      ip        = "10.0.0.4"
      hostname  = ""
      node_name = ""
    }
    not_ready_address {
      ip       = "10.0.0.5"
      hostname = ""
    }
    port {
      name     = ""
      port     = %d
      protocol = "TCP"
    }
  }
  subset {
    address {
      ip = "10.0.0.4"
    }
    port {
      port     = %d
      protocol = "TCP"
    }
  }
}
`, name, port, port+1000)
}

func testAccKubernetesEndpointsV1_inNamespace(name, nsA, nsB, use string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "a" {
  metadata {
    name = %q
  }
}

resource "kubernetes_namespace_v1" "b" {
  metadata {
    name = %q
  }
}

resource "kubernetes_endpoints_v1" "test" {
  metadata {
    name      = %q
    namespace = kubernetes_namespace_v1.%s.metadata[0].name
  }
  subset {
    address {
      ip = "10.0.0.4"
    }
    port {
      port     = 80
      protocol = "TCP"
    }
  }
}
`, nsA, nsB, name, use)
}

func testAccKubernetesEndpointsV1_emptyMaps(name string, port int) string {
	return fmt.Sprintf(`resource "kubernetes_endpoints_v1" "test" {
  metadata {
    name        = %q
    labels      = {}
    annotations = {}
  }
  subset {
    address {
      ip = "10.0.0.4"
    }
    port {
      port     = %d
      protocol = "TCP"
    }
  }
}
`, name, port)
}

func testAccKubernetesEndpointsV1_protocol(name, protocol string) string {
	return fmt.Sprintf(`resource "kubernetes_endpoints_v1" "test" {
  metadata {
    name = %q
  }
  subset {
    address {
      ip = "10.0.0.4"
    }
    port {
      name = "dns"
      port = 53
      %s
    }
  }
}
`, name, protocol)
}

func testAccKubernetesEndpointsV1_noSubset(name string) string {
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
    name = %q
  }
}
`, name)
}

func testAccKubernetesEndpointsV1_identityImport(name, kind, apiVersion string) string {
	return testAccKubernetesEndpointsV1_basic(name) + fmt.Sprintf(`
import {
  to = kubernetes_endpoints_v1.test
  identity = {
    name        = %q
    kind        = %q
    api_version = %q
  }
}
`, name, kind, apiVersion)
}

// Metadata kept out of Terraform state must survive updates to the other metadata map and
// to subset. Mirrors TestAccKubernetesNamespaceV1_updateDoesNotAffectIgnoredMetadata: a key
// can be absent from state as an internal key, filtered by common.FlattenMetadata as
// SDKv2's isInternalKey did, or through the provider's ignore_annotations / ignore_labels,
// which need the muxed server so the provider block is honoured. The key never appears in
// plan or state, so the assertions read the live object.
func TestAccKubernetesEndpointsV1_updateDoesNotAffectIgnoredMetadata(t *testing.T) {
	muxFactories := map[string]func() (tfprotov6.ProviderServer, error){
		"kubernetes": func() (tfprotov6.ProviderServer, error) {
			return mux.MuxServer(context.Background(), "test")
		},
	}

	for _, tc := range []struct {
		name          string
		ignored       string // the map Terraform does not manage
		managed       string // the map the update changes
		key           string
		providerBlock string
		factories     map[string]func() (tfprotov6.ProviderServer, error)
	}{
		{
			name: "internal annotation survives updates", ignored: "annotations", managed: "labels",
			key: ignoredMetadataKey, factories: testAccProtoV6ProviderFactories,
		},
		{
			name: "internal label survives updates", ignored: "labels", managed: "annotations",
			key: ignoredMetadataKey, factories: testAccProtoV6ProviderFactories,
		},
		{
			name: "ignore_annotations survives updates", ignored: "annotations", managed: "labels",
			key:           "outside.example/keep",
			providerBlock: "provider \"kubernetes\" {\n  ignore_annotations = [\"^outside[.]example/keep$\"]\n}\n",
			factories:     muxFactories,
		},
		{
			name: "ignore_labels survives updates", ignored: "labels", managed: "annotations",
			key:           "outside.example/keep",
			providerBlock: "provider \"kubernetes\" {\n  ignore_labels = [\"^outside[.]example/keep$\"]\n}\n",
			factories:     muxFactories,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name := acctest.RandomWithPrefix("tf-acc-test")
			checkUID := testAccCheckEndpointsSameUID()

			config := func(value, ignoredMap string, port int) string {
				return tc.providerBlock + testAccKubernetesEndpointsV1_managedMetadata(name, tc.managed, value, ignoredMap, port)
			}
			settles := func(preApply ...plancheck.PlanCheck) resource.ConfigPlanChecks {
				return resource.ConfigPlanChecks{
					PreApply:             preApply,
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				}
			}
			update := settles(plancheck.ExpectResourceAction(endpointsResourceName, plancheck.ResourceActionUpdate))
			ignoredMapIs := func(check knownvalue.Check) []statecheck.StateCheck {
				return []statecheck.StateCheck{statecheck.ExpectKnownValue(endpointsResourceName,
					tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey(tc.ignored), check)}
			}
			survived := func(value string) resource.TestCheckFunc {
				return resource.ComposeAggregateTestCheckFunc(
					checkUID,
					resource.TestCheckResourceAttr(endpointsResourceName, "metadata.0."+tc.managed+".managed", value),
					testAccCheckIgnoredEndpointsMetadata(name, tc.ignored, tc.key),
				)
			}

			resource.ParallelTest(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: tc.factories,
				CheckDestroy:             testAccCheckKubernetesEndpointV1Destroy,
				Steps: []resource.TestStep{
					{
						Config:           config("before", "", 80),
						ConfigPlanChecks: settles(),
						Check:            checkUID,
					},
					{
						// Another controller writes the key Terraform does not manage.
						PreConfig:         func() { testAccInjectIgnoredEndpointsMetadata(t, name, tc.ignored, tc.key) },
						Config:            config("before", "", 80),
						ConfigPlanChecks:  settles(plancheck.ExpectEmptyPlan()),
						ConfigStateChecks: ignoredMapIs(knownvalue.Null()),
						Check:             survived("before"),
					},
					{
						Config:            config("after", "", 80),
						ConfigPlanChecks:  update,
						ConfigStateChecks: ignoredMapIs(knownvalue.Null()),
						Check:             survived("after"),
					},
					{
						// A subset-only update replaces /subsets and must not touch metadata.
						Config:            config("after", "", 81),
						ConfigPlanChecks:  update,
						ConfigStateChecks: ignoredMapIs(knownvalue.Null()),
						Check:             survived("after"),
					},
					{
						// null -> {} on the ignored map changes state only.
						Config:            config("after", tc.ignored+" = {}", 81),
						ConfigPlanChecks:  update,
						ConfigStateChecks: ignoredMapIs(knownvalue.MapExact(map[string]knownvalue.Check{})),
						Check:             survived("after"),
					},
					{
						// and back, {} -> null.
						Config:            config("after", "", 81),
						ConfigPlanChecks:  update,
						ConfigStateChecks: ignoredMapIs(knownvalue.Null()),
						Check:             survived("after"),
					},
				},
			})
		})
	}
}

// testAccKubernetesEndpointsV1_managedMetadata manages one key in field. ignoredMap is ""
// to leave the other map out of the configuration, or a line such as `annotations = {}`.
func testAccKubernetesEndpointsV1_managedMetadata(name, field, value, ignoredMap string, port int) string {
	return fmt.Sprintf(`resource "kubernetes_endpoints_v1" "test" {
  metadata {
    %s = {
      managed = %q
    }
    %s
    name = %q
  }
  subset {
    address {
      ip = "10.0.0.4"
    }
    port {
      port = %d
    }
  }
}
`, field, value, ignoredMap, name, port)
}

// testAccInjectIgnoredEndpointsMetadata writes key into one of the Endpoints' metadata
// maps, standing in for another controller.
func testAccInjectIgnoredEndpointsMetadata(t *testing.T, name, field, key string) {
	t.Helper()

	client, err := mainClientset()
	if err != nil {
		t.Fatal(err)
	}
	patch := fmt.Sprintf(`{"metadata":{%q:{%q:"controller-owned"}}}`, field, key)
	if _, err := client.CoreV1().Endpoints(metav1.NamespaceDefault).Patch(context.Background(), name,
		k8stypes.MergePatchType, []byte(patch), metav1.PatchOptions{}); err != nil {
		t.Fatal(err)
	}
}

func testAccCheckIgnoredEndpointsMetadata(name, field, key string) resource.TestCheckFunc {
	return testAccCheckEndpointsLive(name, func(ep *api.Endpoints) error {
		live := map[string]map[string]string{"annotations": ep.Annotations, "labels": ep.Labels}
		if got := live[field][key]; got != "controller-owned" {
			return fmt.Errorf("ignored %s[%q] = %q, expected %q (live: %v)", field, key, got, "controller-owned", live[field])
		}
		return nil
	})
}
