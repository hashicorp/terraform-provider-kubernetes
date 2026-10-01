// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	api "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	kquantity "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

const podSpecReportedQuantityConfig = `
resource "kubernetes_pod_v1" "test" {
  metadata {
    labels    = { app = "pod_label" }
    name      = "example"
    namespace = "test"
  }
  spec {
    container {
      image = "busybox:1.36"
      name  = "containername"
      resources = [{
        limits = {
          cpu                 = "0.5"
          memory              = "512Mi"
          "ephemeral-storage" = "512Mi"
        }
        requests = {
          cpu                 = "250m"
          memory              = "50Mi"
          "ephemeral-storage" = "128Mi"
        }
      }]
    }
  }
}`

// This uses the production resource, import handler, client-go JSON transport,
// and Terraform CLI, but the API endpoint is an in-process HTTP server.
func TestPodV1SpecQuantityCanonicalStateCore(t *testing.T) {
	for _, canonicalizeCreate := range []bool{false, true} {
		t.Run(fmt.Sprintf("canonicalize_create_%t", canonicalizeCreate), func(t *testing.T) {
			p := podSpecQuantityHTTPResource(t)
			create := testresource.TestStep{
				Config: podSpecReportedQuantityConfig,
				Check: testresource.TestCheckResourceAttr(
					"kubernetes_pod_v1.test", "spec.0.container.0.resources.0.limits.cpu", "0.5"),
			}
			if canonicalizeCreate {
				create.Check = nil
				create.ExpectError = regexp.MustCompile("Provider produced inconsistent result after apply")
			}
			testresource.UnitTest(t, testresource.TestCase{
				ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
					"kubernetes": providerserver.NewProtocol6WithError(podSpecQuantityNativeProvider{
						resource: &podSpecQuantityNativeResource{PodV1: p, canonicalizeCreate: canonicalizeCreate},
					}),
				},
				Steps: []testresource.TestStep{create},
			})
		})
	}
}

func TestPodV1SpecQuantityImportCanonicalCore(t *testing.T) {
	for _, test := range []struct{ configured, canonical string }{
		{"0.5", "500m"},
		{"500m", "500m"},
		{"1000m", "1"},
	} {
		t.Run(test.configured, func(t *testing.T) {
			p := podSpecQuantityHTTPResource(t)
			config := strings.Replace(podSpecReportedQuantityConfig, `"0.5"`, fmt.Sprintf("%q", test.configured), 1)
			testresource.UnitTest(t, testresource.TestCase{
				ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
					"kubernetes": providerserver.NewProtocol6WithError(podSpecQuantityNativeProvider{resource: p}),
				},
				Steps: []testresource.TestStep{
					{
						Config: config,
						Check: testresource.TestCheckResourceAttr("kubernetes_pod_v1.test",
							"spec.0.container.0.resources.0.limits.cpu", test.configured),
					},
					{
						ResourceName:      "kubernetes_pod_v1.test",
						ImportState:       true,
						ImportStateVerify: test.configured == test.canonical,
						ImportStateCheck: func(states []*terraform.InstanceState) error {
							if len(states) != 1 {
								return fmt.Errorf("imported %d states, want one", len(states))
							}
							for key, want := range map[string]string{
								"spec.0.container.0.resources.0.limits.cpu":                 test.canonical,
								"spec.0.container.0.resources.0.limits.memory":              "512Mi",
								"spec.0.container.0.resources.0.limits.ephemeral-storage":   "512Mi",
								"spec.0.container.0.resources.0.requests.cpu":               "250m",
								"spec.0.container.0.resources.0.requests.memory":            "50Mi",
								"spec.0.container.0.resources.0.requests.ephemeral-storage": "128Mi",
								"metadata.0.uid": "quantity-uid",
							} {
								if got := states[0].Attributes[key]; got != want {
									return fmt.Errorf("imported %s = %q, want %q", key, got, want)
								}
							}
							return nil
						},
					},
					{
						ResourceName:    "kubernetes_pod_v1.test",
						ImportState:     true,
						ImportStateKind: testresource.ImportBlockWithID,
						ImportPlanChecks: testresource.ImportPlanChecks{PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction("kubernetes_pod_v1.test", plancheck.ResourceActionNoop),
						}},
					},
				},
			})
		})
	}
}

type podSpecQuantityNativeProvider struct {
	resource resource.Resource
}

func (podSpecQuantityNativeProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "kubernetes"
}
func (podSpecQuantityNativeProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = providerschema.Schema{}
}
func (podSpecQuantityNativeProvider) Configure(context.Context, provider.ConfigureRequest, *provider.ConfigureResponse) {
}
func (podSpecQuantityNativeProvider) DataSources(context.Context) []func() datasource.DataSource {
	return nil
}
func (p podSpecQuantityNativeProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{func() resource.Resource { return p.resource }}
}

type podSpecQuantityNativeResource struct {
	*PodV1
	canonicalizeCreate bool
}

func (r *podSpecQuantityNativeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.PodV1.Create(ctx, req, resp)
	if !r.canonicalizeCreate || resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx,
		path.Root("spec").AtListIndex(0).AtName("container").AtListIndex(0).
			AtName("resources").AtListIndex(0).AtName("limits").AtMapKey("cpu"),
		types.StringValue("500m"))...)
}

func podSpecQuantityHTTPResource(t *testing.T) *PodV1 {
	return podSpecQuantityHTTPResourceWithDefaults(t, nil)
}

func podSpecQuantityHTTPResourceWithDefaults(t *testing.T, defaults func(*api.Pod)) *PodV1 {
	t.Helper()
	var mu sync.Mutex
	var pod *api.Pod
	return podV1HTTPResource(t, func(w http.ResponseWriter, req *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/api/v1/namespaces/test/pods":
			pod = &api.Pod{}
			if err := json.NewDecoder(req.Body).Decode(pod); err != nil {
				t.Error(err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			pod.APIVersion, pod.Kind = "v1", "Pod"
			pod.Namespace, pod.UID, pod.ResourceVersion = "test", "quantity-uid", "1"
			pod.Status.Phase = api.PodRunning
			for _, containers := range [][]api.Container{pod.Spec.Containers, pod.Spec.InitContainers} {
				for i := range containers {
					container := &containers[i]
					container.ImagePullPolicy = api.PullIfNotPresent
					container.TerminationMessagePolicy = api.TerminationMessageReadFile
					for _, quantities := range []api.ResourceList{container.Resources.Limits, container.Resources.Requests} {
						for name, quantity := range quantities {
							number, suffix := quantity.CanonicalizeBytes(nil)
							quantities[name] = kquantity.MustParse(string(number) + string(suffix))
						}
					}
				}
			}
			if defaults != nil {
				defaults(pod)
			}
			w.WriteHeader(http.StatusCreated)
		case req.Method == http.MethodPatch && req.URL.Path == "/api/v1/namespaces/test/pods/example":
			if pod == nil {
				t.Error("patch before create")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			var patch json.RawMessage
			if err := json.NewDecoder(req.Body).Decode(&patch); err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			prior, err := json.Marshal(pod)
			if err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			merged, err := strategicpatch.StrategicMergePatch(prior, patch, api.Pod{})
			if err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			var updated api.Pod
			if err := json.Unmarshal(merged, &updated); err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if updated.UID != pod.UID || !apiequality.Semantic.DeepEqual(updated.Spec, pod.Spec) {
				t.Error("metadata-only update changed Pod UID or Spec")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			pod = &updated
			pod.ResourceVersion = "2"
		case req.Method == http.MethodDelete && req.URL.Path == "/api/v1/namespaces/test/pods/example":
			pod = nil
			fmt.Fprint(w, `{"kind":"Status","apiVersion":"v1","status":"Success"}`)
			return
		case req.Method == http.MethodGet && req.URL.Path == "/api/v1/namespaces/test/pods/example":
			if pod == nil {
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprint(w, `{"kind":"Status","apiVersion":"v1","reason":"NotFound","code":404}`)
				return
			}
		default:
			t.Errorf("unexpected API operation: %s %s", req.Method, req.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := json.NewEncoder(w).Encode(pod); err != nil {
			t.Error(err)
		}
	})
}
