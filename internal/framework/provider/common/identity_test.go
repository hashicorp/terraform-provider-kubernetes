// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestIdentitySchemaVersionMatchesSDKv2(t *testing.T) {
	if IdentitySchemaVersion != 1 {
		t.Fatalf("IdentitySchemaVersion = %d, want 1 to match kubernetes/resourceidentity.go", IdentitySchemaVersion)
	}
	for name, got := range map[string]int64{
		"IdentitySchema":           IdentitySchema().Version,
		"NamespacedIdentitySchema": NamespacedIdentitySchema().Version,
	} {
		if got != IdentitySchemaVersion {
			t.Errorf("%s().Version = %d, want %d", name, got, IdentitySchemaVersion)
		}
	}
}

func TestNamespacedIdentitySchemaMatchesClusterScoped(t *testing.T) {
	clusterAttrs := IdentitySchema().Attributes
	nsAttrs := NamespacedIdentitySchema().Attributes

	if len(nsAttrs) != len(clusterAttrs)+1 {
		t.Fatalf("namespaced identity has %d attributes, want %d", len(nsAttrs), len(clusterAttrs)+1)
	}
	for name := range clusterAttrs {
		if _, ok := nsAttrs[name]; !ok {
			t.Errorf("namespaced identity is missing %q", name)
		}
	}

	ns, ok := nsAttrs["namespace"]
	if !ok {
		t.Fatal("namespaced identity is missing the namespace attribute")
	}
	strAttr, ok := ns.(identityschema.StringAttribute)
	if !ok {
		t.Fatalf("namespace attribute is %T, want identityschema.StringAttribute", ns)
	}
	if !strAttr.OptionalForImport {
		t.Error("namespace must be OptionalForImport, matching resourceIdentitySchemaNamespaced()")
	}
	if strAttr.RequiredForImport {
		t.Error("namespace must not be RequiredForImport")
	}
}

func emptyIdentity(t *testing.T, s identityschema.Schema) *tfsdk.ResourceIdentity {
	t.Helper()
	return &tfsdk.ResourceIdentity{
		Schema: s,
		Raw:    tftypes.NewValue(s.Type().TerraformType(context.Background()), nil),
	}
}

func runUpgrade(t *testing.T, upgraders map[int64]resource.IdentityUpgrader, s identityschema.Schema, storedJSON string) *tfsdk.ResourceIdentity {
	t.Helper()
	upgrader, ok := upgraders[0]
	if !ok {
		t.Fatal("no upgrader registered for identity schema version 0")
	}

	req := resource.UpgradeIdentityRequest{}
	if storedJSON != "" {
		req.RawIdentity = &tfprotov6.RawState{JSON: []byte(storedJSON)}
	}
	resp := &resource.UpgradeIdentityResponse{Identity: emptyIdentity(t, s)}

	upgrader.IdentityUpgrader(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("upgrade returned diagnostics: %v", resp.Diagnostics)
	}
	return resp.Identity
}

func TestUpgradeIdentityWithNoStoredIdentity(t *testing.T) {
	identity := runUpgrade(t, UpgradeIdentity("Namespace", "v1"), IdentitySchema(), "")

	var got ResourceIdentity
	if diags := identity.Get(context.Background(), &got); diags.HasError() {
		t.Fatalf("reading upgraded identity: %v", diags)
	}
	for field, v := range map[string]types.String{
		"APIVersion": got.APIVersion, "Kind": got.Kind, "Name": got.Name,
	} {
		if !v.IsNull() {
			t.Errorf("%s = %v, want null when no identity was stored", field, v)
		}
	}
}

func TestUpgradeIdentityWithStoredIdentity(t *testing.T) {
	identity := runUpgrade(t, UpgradeIdentity("Namespace", "v1"), IdentitySchema(), `{"name":"my-ns"}`)

	var got ResourceIdentity
	if diags := identity.Get(context.Background(), &got); diags.HasError() {
		t.Fatalf("reading upgraded identity: %v", diags)
	}
	if !got.Name.Equal(types.StringValue("my-ns")) {
		t.Errorf("Name = %v, want my-ns (carried from stored identity)", got.Name)
	}
	if !got.APIVersion.Equal(types.StringValue("v1")) || !got.Kind.Equal(types.StringValue("Namespace")) {
		t.Errorf("api_version/kind = %v/%v, want v1/Namespace", got.APIVersion, got.Kind)
	}
}

func TestUpgradeIdentityInvalidJSON(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		schema    identityschema.Schema
		upgraders map[int64]resource.IdentityUpgrader
	}{
		{"cluster-scoped", IdentitySchema(), UpgradeIdentity("Namespace", "v1")},
		{"namespaced", NamespacedIdentitySchema(), UpgradeNamespacedIdentity("Role", "rbac.authorization.k8s.io/v1")},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			identity := emptyIdentity(t, testCase.schema)
			prior := identity.Raw
			response := resource.UpgradeIdentityResponse{Identity: identity}
			testCase.upgraders[0].IdentityUpgrader(context.Background(), resource.UpgradeIdentityRequest{
				RawIdentity: &tfprotov6.RawState{JSON: []byte(`{"name":`)},
			}, &response)
			if !response.Diagnostics.HasError() {
				t.Fatal("expected diagnostics for invalid JSON")
			}
			if !response.Identity.Raw.Equal(prior) {
				t.Error("invalid JSON must not overwrite identity")
			}
		})
	}
}

func TestUpgradeNamespacedIdentity(t *testing.T) {
	testCases := []struct {
		name          string
		stored        string
		wantName      types.String
		wantNamespace types.String
	}{
		{"nothing stored", "", types.StringNull(), types.StringNull()},
		{
			"name and namespace stored",
			`{"name":"my-role","namespace":"team-a"}`,
			types.StringValue("my-role"), types.StringValue("team-a"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			identity := runUpgrade(t, UpgradeNamespacedIdentity("Role", "rbac.authorization.k8s.io/v1"),
				NamespacedIdentitySchema(), tc.stored)

			var got NamespacedResourceIdentity
			if diags := identity.Get(context.Background(), &got); diags.HasError() {
				t.Fatalf("reading upgraded identity: %v", diags)
			}
			if !got.Name.Equal(tc.wantName) {
				t.Errorf("Name = %v, want %v", got.Name, tc.wantName)
			}
			if !got.Namespace.Equal(tc.wantNamespace) {
				t.Errorf("Namespace = %v, want %v", got.Namespace, tc.wantNamespace)
			}
			wantAPIVersion, wantKind := types.StringNull(), types.StringNull()
			if tc.stored != "" {
				wantAPIVersion = types.StringValue("rbac.authorization.k8s.io/v1")
				wantKind = types.StringValue("Role")
			}
			if !got.APIVersion.Equal(wantAPIVersion) || !got.Kind.Equal(wantKind) {
				t.Errorf("api_version/kind = %v/%v, want %v/%v", got.APIVersion, got.Kind, wantAPIVersion, wantKind)
			}
		})
	}
}
