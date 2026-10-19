// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
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

func testRoleBindingModel(labels map[string]string, subjects []SubjectModel) RoleBindingModel {
	return RoleBindingModel{
		ID: types.StringValue("team/reader"),
		Metadata: []common.NamespacedMetadataModel{{
			MetadataModel: common.MetadataModel{
				MetadataBase: common.MetadataBase{
					Annotations:     types.MapNull(types.StringType),
					Generation:      types.Int64Value(0),
					Labels:          stringMap(labels),
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

func stringMap(in map[string]string) types.Map {
	if in == nil {
		return types.MapNull(types.StringType)
	}
	out := make(map[string]attr.Value, len(in))
	for k, v := range in {
		out[k] = types.StringValue(v)
	}
	return types.MapValueMust(types.StringType, out)
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
			if diags := state.Set(ctx, testRoleBindingModel(nil, []SubjectModel{{
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

// Reproduces the review scenario: two subjects in state, one removed outside Terraform,
// then a label-only change applied without refresh. Update must not patch subjects.
func TestRoleBindingUpdateSkipsUnchangedSubjects(t *testing.T) {
	ctx := context.Background()
	stateSubjects := []SubjectModel{
		{APIGroup: types.StringValue(""), Kind: types.StringValue("ServiceAccount"), Name: types.StringValue("a"), Namespace: types.StringValue("team")},
		{APIGroup: types.StringValue(""), Kind: types.StringValue("ServiceAccount"), Name: types.StringValue("b"), Namespace: types.StringValue("team")},
	}
	// api_group is Optional+Computed and unconfigured, so the plan carries unknown.
	planSubjects := []SubjectModel{
		{APIGroup: types.StringUnknown(), Kind: types.StringValue("ServiceAccount"), Name: types.StringValue("a"), Namespace: types.StringValue("team")},
		{APIGroup: types.StringUnknown(), Kind: types.StringValue("ServiceAccount"), Name: types.StringValue("b"), Namespace: types.StringValue("team")},
	}

	var patchBody string
	r := newTestRoleBinding(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPatch {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
		b, _ := io.ReadAll(req.Body)
		patchBody = string(b)
		if strings.Contains(patchBody, "/subjects") {
			// Mirrors the API server rejecting a replace of a removed index.
			writeJSON(t, w, http.StatusUnprocessableEntity, metav1.Status{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Status"},
				Status:   metav1.StatusFailure, Code: http.StatusUnprocessableEntity, Reason: metav1.StatusReasonInvalid,
			})
			return
		}
		writeJSON(t, w, http.StatusOK, rbacv1api.RoleBinding{
			TypeMeta: metav1.TypeMeta{APIVersion: roleBindingAPIVersion, Kind: roleBindingKind},
			ObjectMeta: metav1.ObjectMeta{
				Name: "reader", Namespace: "team", UID: "uid-1", ResourceVersion: "11",
				Labels: map[string]string{"env": "new"},
			},
			RoleRef:  rbacv1api.RoleRef{APIGroup: rbacv1api.GroupName, Kind: "Role", Name: "reader"},
			Subjects: []rbacv1api.Subject{{Kind: "ServiceAccount", Name: "a", Namespace: "team"}},
		})
	})

	var schema resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schema)
	state := tfsdk.State{Schema: schema.Schema}
	if diags := state.Set(ctx, testRoleBindingModel(map[string]string{"env": "old"}, stateSubjects)); diags.HasError() {
		t.Fatal(diags)
	}
	plan := tfsdk.Plan{Schema: schema.Schema}
	if diags := plan.Set(ctx, testRoleBindingModel(map[string]string{"env": "new"}, planSubjects)); diags.HasError() {
		t.Fatal(diags)
	}

	identity := nullIdentity(ctx)
	resp := resource.UpdateResponse{State: state, Identity: &identity}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("update failed (patch %s): %v", patchBody, resp.Diagnostics)
	}
	if !strings.Contains(patchBody, "/metadata/labels") {
		t.Fatalf("expected a label patch, got %s", patchBody)
	}

	var got RoleBindingModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatal(diags)
	}
	if len(got.Subject) != len(stateSubjects) {
		t.Fatalf("subject count = %d, want %d", len(got.Subject), len(stateSubjects))
	}
	for i := range stateSubjects {
		if got.Subject[i] != stateSubjects[i] {
			t.Errorf("subject[%d] = %#v, want %#v", i, got.Subject[i], stateSubjects[i])
		}
	}
}

func TestRoleBindingImportState(t *testing.T) {
	ctx := context.Background()
	r := &RoleBindingV1{}
	var schema resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schema)

	for _, tc := range []struct {
		name       string
		id         string
		identity   *common.NamespacedResourceIdentity
		wantID     string
		wantErrSub string
	}{
		{name: "string id", id: "team/reader", wantID: "team/reader"},
		{
			name: "identity with namespace",
			identity: &common.NamespacedResourceIdentity{
				ResourceIdentity: common.ResourceIdentity{
					APIVersion: types.StringValue(roleBindingAPIVersion), Kind: types.StringValue(roleBindingKind), Name: types.StringValue("reader"),
				},
				Namespace: types.StringValue("team"),
			},
			wantID: "team/reader",
		},
		{
			name: "identity without namespace defaults to default",
			identity: &common.NamespacedResourceIdentity{
				ResourceIdentity: common.ResourceIdentity{
					APIVersion: types.StringValue(roleBindingAPIVersion), Kind: types.StringValue(roleBindingKind), Name: types.StringValue("reader"),
				},
				Namespace: types.StringNull(),
			},
			wantID: "default/reader",
		},
		{
			name: "identity with wrong kind",
			identity: &common.NamespacedResourceIdentity{
				ResourceIdentity: common.ResourceIdentity{
					APIVersion: types.StringValue(roleBindingAPIVersion), Kind: types.StringValue("ConfigMap"), Name: types.StringValue("reader"),
				},
				Namespace: types.StringValue("team"),
			},
			wantErrSub: "invalid identity import",
		},
		{
			name: "identity with wrong api_version",
			identity: &common.NamespacedResourceIdentity{
				ResourceIdentity: common.ResourceIdentity{
					APIVersion: types.StringValue("v1"), Kind: types.StringValue(roleBindingKind), Name: types.StringValue("reader"),
				},
				Namespace: types.StringValue("team"),
			},
			wantErrSub: "invalid identity import",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			identity := nullIdentity(ctx)
			if tc.identity != nil {
				if diags := identity.Set(ctx, *tc.identity); diags.HasError() {
					t.Fatal(diags)
				}
			}
			state := tfsdk.State{Schema: schema.Schema, Raw: tftypes.NewValue(schema.Schema.Type().TerraformType(ctx), nil)}
			respIdentity := nullIdentity(ctx)
			resp := resource.ImportStateResponse{State: state, Identity: &respIdentity}
			r.ImportState(ctx, resource.ImportStateRequest{ID: tc.id, Identity: &identity}, &resp)

			if tc.wantErrSub != "" {
				if !resp.Diagnostics.HasError() || !strings.Contains(fmt.Sprint(resp.Diagnostics), tc.wantErrSub) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErrSub, resp.Diagnostics)
				}
				return
			}
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			var id types.String
			if diags := resp.State.GetAttribute(ctx, path.Root("id"), &id); diags.HasError() {
				t.Fatal(diags)
			}
			if id.ValueString() != tc.wantID {
				t.Fatalf("id = %q, want %q", id.ValueString(), tc.wantID)
			}
		})
	}
}
