// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/appsv1"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/batchv1"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/corev1"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	"k8s.io/client-go/dynamic"
	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type stubClientsets struct {
	kubernetes.KubeClientsets
	config *rest.Config
}

func (c stubClientsets) MainClientset() (*k8sclient.Clientset, error) {
	return k8sclient.NewForConfig(c.config)
}
func (c stubClientsets) DynamicClient() (dynamic.Interface, error) {
	return dynamic.NewForConfig(c.config)
}
func (stubClientsets) GetIgnoreAnnotations() []string { return nil }
func (stubClientsets) GetIgnoreLabels() []string      { return nil }

// State without identity, for an object that is gone or whose deletion cannot
// be confirmed, must come back with identity: Terraform keeps the state of a
// failed Update or Delete, and Framework rejects a Read without identity even
// when it removes the resource.
func TestWorkloadIdentityWithoutLiveObject(t *testing.T) {
	ctx := context.Background()
	var getStatus atomic.Int64
	getStatus.Store(http.StatusNotFound)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"kind":"Status","apiVersion":"v1","status":"Success"}`))
			return
		}
		http.Error(w, "unavailable", int(getStatus.Load()))
	}))
	defer server.Close()

	for name, newResource := range map[string]func() resource.Resource{
		"pod":          corev1.NewPodV1,
		"deployment":   appsv1.NewDeploymentV1,
		"daemon set":   appsv1.NewDaemonSetV1,
		"stateful set": appsv1.NewStatefulSetV1,
		"job":          batchv1.NewJobV1,
		"cron job":     batchv1.NewCronJobV1,
	} {
		t.Run(name, func(t *testing.T) {
			r := newResource()
			r.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{
				ProviderData: func() any { return stubClientsets{config: &rest.Config{Host: server.URL}} },
			}, &resource.ConfigureResponse{})
			var schemaResp resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
			var identityResp resource.IdentitySchemaResponse
			r.(resource.ResourceWithIdentity).IdentitySchema(ctx, resource.IdentitySchemaRequest{}, &identityResp)

			raw := tfprotov6.RawState{JSON: []byte(`{"id":"ns/n","metadata":[{"name":"n","namespace":"ns"}]}`)}
			value, err := raw.Unmarshal(schemaResp.Schema.Type().TerraformType(ctx))
			if err != nil {
				t.Fatal(err)
			}
			state := tfsdk.State{Schema: schemaResp.Schema, Raw: value}
			noIdentity := func() *tfsdk.ResourceIdentity {
				return &tfsdk.ResourceIdentity{
					Schema: identityResp.IdentitySchema,
					Raw:    tftypes.NewValue(identityResp.IdentitySchema.Type().TerraformType(ctx), nil),
				}
			}
			check := func(t *testing.T, identity *tfsdk.ResourceIdentity) {
				t.Helper()
				var got common.NamespacedResourceIdentity
				if diags := identity.Get(ctx, &got); diags.HasError() {
					t.Fatal(diags)
				}
				if got.Namespace.ValueString() != "ns" || got.Name.ValueString() != "n" || got.Kind.IsNull() || got.APIVersion.IsNull() {
					t.Fatalf("identity = %v", identity.Raw)
				}
			}

			t.Run("read not found", func(t *testing.T) {
				resp := resource.ReadResponse{State: state, Identity: noIdentity()}
				r.Read(ctx, resource.ReadRequest{State: state, Identity: noIdentity()}, &resp)
				if resp.Diagnostics.HasError() || !resp.State.Raw.IsNull() {
					t.Fatalf("state = %v, diagnostics = %v", resp.State.Raw, resp.Diagnostics)
				}
				check(t, resp.Identity)
			})
			t.Run("update not found", func(t *testing.T) {
				resp := resource.UpdateResponse{State: state, Identity: noIdentity()}
				r.Update(ctx, resource.UpdateRequest{State: state, Plan: tfsdk.Plan(state), Config: tfsdk.Config(state), Identity: noIdentity()}, &resp)
				if !resp.Diagnostics.HasError() {
					t.Fatal("expected an error")
				}
				check(t, resp.Identity)
			})
			t.Run("delete unconfirmed", func(t *testing.T) {
				getStatus.Store(http.StatusServiceUnavailable)
				defer getStatus.Store(http.StatusNotFound)
				resp := resource.DeleteResponse{State: state, Identity: noIdentity()}
				r.Delete(ctx, resource.DeleteRequest{State: state, Identity: noIdentity()}, &resp)
				if !resp.Diagnostics.HasError() {
					t.Fatal("expected an error")
				}
				check(t, resp.Identity)
			})
		})
	}
}
