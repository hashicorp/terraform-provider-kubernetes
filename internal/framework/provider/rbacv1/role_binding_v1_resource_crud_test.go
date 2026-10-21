// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	rbacv1api "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type testRoleBindingMeta struct {
	kubernetes.KubeClientsets
	client *client.Clientset
}

func (m testRoleBindingMeta) MainClientset() (*client.Clientset, error) { return m.client, nil }
func (m testRoleBindingMeta) GetIgnoreAnnotations() []string            { return nil }
func (m testRoleBindingMeta) GetIgnoreLabels() []string                 { return nil }

func newTestRoleBinding(t *testing.T, handler http.HandlerFunc) *RoleBindingV1 {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	conn, err := client.NewForConfig(&rest.Config{Host: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	return &RoleBindingV1{SDKv2Meta: func() any { return testRoleBindingMeta{client: conn} }}
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, body any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		t.Error(err)
	}
}

func nullIdentity(ctx context.Context) tfsdk.ResourceIdentity {
	s := common.NamespacedIdentitySchema()
	return tfsdk.ResourceIdentity{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
}

func testRoleBindingModel(subjects []SubjectModel) RoleBindingModel {
	return RoleBindingModel{
		ID: types.StringValue("team/reader"),
		Metadata: []common.NamespacedMetadataModel{{
			MetadataModel: common.MetadataModel{
				MetadataBase: common.MetadataBase{
					Annotations:     types.MapNull(types.StringType),
					Generation:      types.Int64Value(0),
					Labels:          types.MapNull(types.StringType),
					Name:            types.StringValue("reader"),
					ResourceVersion: types.StringValue("10"),
					UID:             types.StringValue("uid-1"),
				},
				GenerateName: types.StringNull(),
			},
			Namespace: types.StringValue("team"),
		}},
		RoleRef: []RoleRefModel{{
			APIGroup: types.StringValue(rbacv1api.GroupName),
			Kind:     types.StringValue("Role"),
			Name:     types.StringValue("reader"),
		}},
		Subject: subjects,
	}
}

func TestRoleBindingReadNotFoundSetsIdentity(t *testing.T) {
	ctx := context.Background()
	for _, storedIdentity := range []bool{false, true} {
		t.Run(fmt.Sprintf("identity=%t", storedIdentity), func(t *testing.T) {
			r := newTestRoleBinding(t, func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodGet || req.URL.Path != "/apis/rbac.authorization.k8s.io/v1/namespaces/team/rolebindings/reader" {
					t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
				}
				writeJSON(t, w, http.StatusNotFound, metav1.Status{
					TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Status"},
					Status:   metav1.StatusFailure, Code: http.StatusNotFound, Reason: metav1.StatusReasonNotFound,
				})
			})

			var schema resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schema)
			state := tfsdk.State{Schema: schema.Schema}
			if diags := state.Set(ctx, testRoleBindingModel([]SubjectModel{{
				APIGroup: types.StringValue(rbacv1api.GroupName), Kind: types.StringValue("User"),
				Name: types.StringValue("alice"), Namespace: types.StringValue(""),
			}})); diags.HasError() {
				t.Fatal(diags)
			}

			identity := nullIdentity(ctx)
			want := common.NamespacedResourceIdentity{
				ResourceIdentity: common.ResourceIdentity{
					APIVersion: types.StringValue(roleBindingAPIVersion),
					Kind:       types.StringValue(roleBindingKind),
					Name:       types.StringValue("reader"),
				},
				Namespace: types.StringValue("team"),
			}
			if storedIdentity {
				// A distinct value proves an existing identity is not rebuilt.
				want.Name = types.StringValue("existing-reader")
				if diags := identity.Set(ctx, want); diags.HasError() {
					t.Fatal(diags)
				}
			}

			respIdentity := identity
			resp := resource.ReadResponse{State: state, Identity: &respIdentity}
			r.Read(ctx, resource.ReadRequest{State: state, Identity: &identity}, &resp)

			if resp.Diagnostics.HasError() || !resp.State.Raw.IsNull() {
				t.Fatalf("missing RoleBinding must be removed without error: %v", resp.Diagnostics)
			}
			var got common.NamespacedResourceIdentity
			if diags := resp.Identity.Get(ctx, &got); diags.HasError() {
				t.Fatal(diags)
			}
			if got != want {
				t.Fatalf("identity = %#v, want %#v", got, want)
			}
		})
	}
}
