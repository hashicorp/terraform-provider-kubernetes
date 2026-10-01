// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func TestPodV1UpdatePatch(t *testing.T) {
	spec := corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "image:1"}}}
	for _, test := range []struct {
		name   string
		prior  metav1.ObjectMeta
		plan   metav1.ObjectMeta
		live   metav1.ObjectMeta
		expect map[string]interface{}
	}{
		{
			name: "state-only empty maps",
			plan: metav1.ObjectMeta{Labels: map[string]string{}, Annotations: map[string]string{}},
			live: metav1.ObjectMeta{Labels: map[string]string{"controller": "keep"}},
		},
		{
			name: "first managed key retains external metadata",
			plan: metav1.ObjectMeta{Labels: map[string]string{"managed": "new"}},
			live: metav1.ObjectMeta{Labels: map[string]string{"controller": "keep"}},
			expect: map[string]interface{}{
				"metadata": map[string]interface{}{"labels": map[string]interface{}{"managed": "new"}},
			},
		},
		{
			name:  "remove only managed keys",
			prior: metav1.ObjectMeta{Annotations: map[string]string{"managed/x~y": "old"}},
			live:  metav1.ObjectMeta{Annotations: map[string]string{"managed/x~y": "old", "controller": "keep"}},
			expect: map[string]interface{}{
				"metadata": map[string]interface{}{"annotations": map[string]interface{}{"managed/x~y": nil}},
			},
		},
		{
			name:  "already converged metadata",
			prior: metav1.ObjectMeta{Labels: map[string]string{"managed": "old"}},
			plan:  metav1.ObjectMeta{Labels: map[string]string{"managed": "new"}},
			live:  metav1.ObjectMeta{Labels: map[string]string{"managed": "new", "controller": "keep"}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			actual := podV1UpdatePatch(test.prior, test.plan, &spec, &spec, &corev1.Pod{
				ObjectMeta: test.live,
				Spec:       spec,
			})
			if test.expect == nil {
				test.expect = map[string]interface{}{}
			}
			if !reflect.DeepEqual(actual, test.expect) {
				t.Fatalf("patch = %#v, want %#v", actual, test.expect)
			}
		})
	}
	t.Run("image edit targets container name and no metadata", func(t *testing.T) {
		planned := spec.DeepCopy()
		planned.Containers[0].Image = "image:2"
		live := corev1.Pod{Spec: corev1.PodSpec{Containers: []corev1.Container{
			{Name: "injected", Image: "sidecar:1"}, spec.Containers[0],
		}}}
		actual := podV1UpdatePatch(metav1.ObjectMeta{}, metav1.ObjectMeta{}, &spec, planned, &live)
		expect := map[string]interface{}{"spec": map[string]interface{}{
			"containers": []map[string]string{{"name": "app", "image": "image:2"}},
		}}
		if !reflect.DeepEqual(actual, expect) {
			t.Fatalf("patch = %#v, want %#v", actual, expect)
		}
	})
	t.Run("active deadline edit", func(t *testing.T) {
		seconds := int64(120)
		planned := spec.DeepCopy()
		planned.ActiveDeadlineSeconds = &seconds
		patch := podV1UpdatePatch(metav1.ObjectMeta{}, metav1.ObjectMeta{}, &spec, planned, &corev1.Pod{Spec: spec})
		data, err := json.Marshal(patch)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != `{"spec":{"activeDeadlineSeconds":120}}` {
			t.Fatalf("unexpected patch: %s", data)
		}
	})
}

func TestPodV1Configure(t *testing.T) {
	ctx := context.Background()
	p := &PodV1{}
	var response resource.ConfigureResponse
	p.Configure(ctx, resource.ConfigureRequest{}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	_, _, diags := p.sdkv2Meta()
	if !diags.HasError() {
		t.Fatal("missing provider configuration should produce a diagnostic")
	}
	p.Configure(ctx, resource.ConfigureRequest{ProviderData: "invalid"}, &response)
	if !response.Diagnostics.HasError() {
		t.Fatal("invalid provider data should produce a diagnostic")
	}
	p.SDKv2Meta = func() any { return "invalid" }
	if _, _, diags := p.sdkv2Meta(); !diags.HasError() {
		t.Fatal("invalid deferred metadata should produce a diagnostic")
	}
}

func TestPodV1TargetStates(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name   string
		input  types.List
		expect []string
	}{
		{"omitted defaults to Running", types.ListNull(types.StringType), []string{"Running"}},
		{"SDK empty default means Running", types.ListValueMust(types.StringType, nil), []string{"Running"}},
		{"explicit phase", types.ListValueMust(types.StringType, []attr.Value{types.StringValue("Succeeded")}), []string{"Succeeded"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, diags := podV1TargetStates(ctx, test.input)
			if diags.HasError() || !reflect.DeepEqual(actual, test.expect) {
				t.Fatalf("targets = %v, diagnostics = %v", actual, diags)
			}
		})
	}
}

type podV1HTTPTestMeta struct {
	kubernetes.KubeClientsets
	client *k8sclient.Clientset
}

func (m podV1HTTPTestMeta) MainClientset() (*k8sclient.Clientset, error) {
	return m.client, nil
}
func (m podV1HTTPTestMeta) GetIgnoreAnnotations() []string { return nil }
func (m podV1HTTPTestMeta) GetIgnoreLabels() []string      { return nil }

func podV1HTTPResource(t *testing.T, handler http.HandlerFunc) *PodV1 {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := k8sclient.NewForConfig(&rest.Config{
		Host:          server.URL,
		ContentConfig: rest.ContentConfig{ContentType: "application/json", AcceptContentTypes: "application/json"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return &PodV1{SDKv2Meta: func() any { return podV1HTTPTestMeta{client: client} }}
}

func podV1HTTPState(t *testing.T, p *PodV1) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	var schemaResponse resource.SchemaResponse
	p.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
	state := tfsdk.State{Schema: schemaResponse.Schema}
	specType := schemaResponse.Schema.Blocks["spec"].Type().(types.ListType)
	model := PodV1Model{
		ID:          types.StringValue("test/example"),
		Spec:        types.ListNull(specType.ElemType),
		TargetState: types.ListNull(types.StringType),
		Timeouts: timeouts.Value{
			Object: types.ObjectNull(timeoutsAttributeTypes(schemaResponse.Schema)),
		},
	}
	if diags := state.Set(ctx, &model); diags.HasError() {
		t.Fatal(diags)
	}
	return state
}

func TestPodV1ReadErrorsPreserveState(t *testing.T) {
	for _, code := range []int{http.StatusNotFound, http.StatusForbidden, http.StatusInternalServerError} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			calls := 0
			p := podV1HTTPResource(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != http.MethodGet || r.URL.Path != "/api/v1/namespaces/test/pods/example" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(code)
				fmt.Fprintf(w, `{"kind":"Status","apiVersion":"v1","status":"Failure","reason":%q,"code":%d}`, http.StatusText(code), code)
			})
			state := podV1HTTPState(t, p)
			response := resource.ReadResponse{State: state}
			p.Read(context.Background(), resource.ReadRequest{State: state}, &response)
			if calls != 1 {
				t.Fatalf("expected one GET, got %d", calls)
			}
			if code == http.StatusNotFound {
				if response.Diagnostics.HasError() || !response.State.Raw.IsNull() {
					t.Fatalf("NotFound should remove state: %v", response.Diagnostics)
				}
			} else if !response.Diagnostics.HasError() || !response.State.Raw.Equal(state.Raw) {
				t.Fatalf("non-NotFound errors must retain state: %v", response.Diagnostics)
			}
		})
	}
}

func TestPodV1DeleteAlreadyAbsent(t *testing.T) {
	p := podV1HTTPResource(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("unexpected request: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"kind":"Status","apiVersion":"v1","status":"Failure","reason":"NotFound","code":404}`)
	})
	state := podV1HTTPState(t, p)
	response := resource.DeleteResponse{State: state}
	p.Delete(context.Background(), resource.DeleteRequest{State: state}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
}

func TestPodV1ReadTargetStateDefault(t *testing.T) {
	for _, target := range []types.List{
		types.ListNull(types.StringType),
		types.ListValueMust(types.StringType, nil),
		types.ListValueMust(types.StringType, []attr.Value{types.StringValue("Pending")}),
	} {
		t.Run(target.String(), func(t *testing.T) {
			ctx := context.Background()
			p := podV1HTTPResource(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/api/v1/namespaces/test/pods/example" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "example", Namespace: "test", UID: "test-uid"},
					Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "image:1"}}},
				}); err != nil {
					t.Error(err)
				}
			})
			state := podV1HTTPState(t, p)
			var model PodV1Model
			if diags := state.Get(ctx, &model); diags.HasError() {
				t.Fatal(diags)
			}
			model.TargetState = target
			if diags := state.Set(ctx, &model); diags.HasError() {
				t.Fatal(diags)
			}
			_, identitySchema := podV1Schemas(t, p)
			response := resource.ReadResponse{
				State:    state,
				Identity: &tfsdk.ResourceIdentity{Schema: identitySchema.IdentitySchema},
			}
			p.Read(ctx, resource.ReadRequest{State: state}, &response)
			if response.Diagnostics.HasError() {
				t.Fatal(response.Diagnostics)
			}
			if diags := response.State.Get(ctx, &model); diags.HasError() {
				t.Fatal(diags)
			}
			want := target
			if target.IsNull() {
				want = types.ListValueMust(types.StringType, nil)
			}
			if !model.TargetState.Equal(want) {
				t.Fatalf("target_state = %v, want %v", model.TargetState, want)
			}
		})
	}
}

func TestPodV1CreateGeneratedNameAndWaitFailure(t *testing.T) {
	for _, test := range []struct {
		name      string
		phase     corev1.PodPhase
		target    string
		wantError bool
	}{
		{"generated name", corev1.PodPending, "Pending", false},
		{"wait failure retains identity", corev1.PodFailed, "Running", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			pod := corev1.Pod{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "generated-example", Namespace: "test", GenerateName: "generated-",
					UID: "test-uid", ResourceVersion: "1",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app", Image: "image:1"}},
				},
				Status: corev1.PodStatus{Phase: test.phase},
			}
			posts, gets := 0, 0
			p := podV1HTTPResource(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/api/v1/namespaces/test/pods":
					posts++
					var request corev1.Pod
					if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
						t.Error(err)
					}
					if request.Name != "" || request.GenerateName != "generated-" {
						t.Errorf("wrong generated-name request: %#v", request.ObjectMeta)
					}
					w.WriteHeader(http.StatusCreated)
				case r.Method == http.MethodGet && r.URL.Path == "/api/v1/namespaces/test/pods/generated-example":
					gets++
				case r.URL.Path == "/api/v1/namespaces/test/events":
					w.WriteHeader(http.StatusForbidden)
					fmt.Fprint(w, `{"kind":"Status","apiVersion":"v1","reason":"Forbidden","code":403}`)
					return
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if err := json.NewEncoder(w).Encode(pod); err != nil {
					t.Error(err)
				}
			})
			state := podV1HTTPState(t, p)
			var model PodV1Model
			if diags := state.Get(ctx, &model); diags.HasError() {
				t.Fatal(diags)
			}
			spec, diags := flattenPodV1Spec(ctx, pod.Spec, model.Spec)
			if diags.HasError() {
				t.Fatal(diags)
			}
			model.ID = types.StringUnknown()
			model.Spec = spec
			model.Metadata = []common.NamespacedMetadataModel{{
				MetadataModel: common.MetadataModel{
					MetadataBase: common.MetadataBase{
						Name: types.StringUnknown(), UID: types.StringUnknown(),
						ResourceVersion: types.StringUnknown(), Generation: types.Int64Unknown(),
						Labels: types.MapNull(types.StringType), Annotations: types.MapNull(types.StringType),
					},
					GenerateName: types.StringValue("generated-"),
				},
				Namespace: types.StringValue("test"),
			}}
			model.TargetState = types.ListValueMust(types.StringType, []attr.Value{types.StringValue(test.target)})
			if diags := state.Set(ctx, &model); diags.HasError() {
				t.Fatal(diags)
			}
			response := resource.CreateResponse{
				State: state,
				Identity: &tfsdk.ResourceIdentity{
					Schema: common.NamespacedIdentitySchema(),
				},
			}
			p.Create(ctx, resource.CreateRequest{
				Plan: tfsdk.Plan(state),
			}, &response)
			if response.Diagnostics.HasError() != test.wantError {
				t.Fatalf("unexpected diagnostics: %v", response.Diagnostics)
			}
			if test.wantError && !strings.Contains(response.Diagnostics[0].Detail(), "Failed") {
				t.Fatalf("event lookup must not hide the original wait failure: %v", response.Diagnostics)
			}
			if diags := response.State.Get(ctx, &model); diags.HasError() {
				t.Fatal(diags)
			}
			if model.ID.ValueString() != "test/generated-example" || model.Metadata[0].UID.ValueString() != "test-uid" {
				t.Fatalf("created Pod identity was not retained: %#v", model)
			}
			var identity common.NamespacedResourceIdentity
			if diags := response.Identity.Get(ctx, &identity); diags.HasError() {
				t.Fatal(diags)
			}
			if identity.Name.ValueString() != pod.Name || posts != 1 || gets == 0 {
				t.Fatalf("identity=%v, POSTs=%d, GETs=%d", identity, posts, gets)
			}
		})
	}
}
