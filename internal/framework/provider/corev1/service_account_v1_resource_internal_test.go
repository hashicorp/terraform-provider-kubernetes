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
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	jsonpatch "gopkg.in/evanphx/json-patch.v4"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func TestServiceAccountInternalSecretsPreserveToken(t *testing.T) {
	token := corev1.ObjectReference{Name: "account-token-abc", UID: k8stypes.UID("token-uid"), Namespace: "ns"}
	got := serviceAccountSecrets([]string{"new"}, token.Name, []corev1.ObjectReference{token, {Name: "old"}})
	want := []corev1.ObjectReference{{Name: "new"}, token}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("references = %#v, want %#v", got, want)
	}
	got = serviceAccountSecrets([]string{token.Name}, token.Name, []corev1.ObjectReference{token})
	if len(got) != 1 || got[0] != token {
		t.Fatalf("token duplicated or changed: %#v", got)
	}
}

func TestServiceAccountInternalNoopPatch(t *testing.T) {
	ctx := context.Background()
	metadata := []common.NamespacedMetadataModel{{MetadataModel: common.MetadataModel{
		MetadataBase: common.MetadataBase{Labels: types.MapNull(types.StringType), Annotations: types.MapNull(types.StringType)},
	}}}
	state := ServiceAccountV1Model{
		Metadata: metadata, Secret: types.SetNull(serviceAccountReferenceType), ImagePullSecret: types.SetNull(serviceAccountReferenceType),
		AutomountServiceAccountToken: types.BoolValue(true),
	}
	plan := state
	plan.Secret = types.SetValueMust(serviceAccountReferenceType, []attr.Value{})
	plan.ImagePullSecret = types.SetValueMust(serviceAccountReferenceType, []attr.Value{})
	ops, diags := serviceAccountPatch(ctx, state, plan, &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{ResourceVersion: "123"}})
	if diags.HasError() || len(ops) != 0 {
		t.Fatalf("empty representation change produced patch: %#v, %v", ops, diags)
	}
}

func TestServiceAccountInternalDefaultAdoptionPreservesReferences(t *testing.T) {
	ctx := context.Background()
	for _, empty := range []bool{false, true} {
		t.Run(fmt.Sprintf("empty=%t", empty), func(t *testing.T) {
			current := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name: "default", Namespace: "ns", ResourceVersion: "100",
					Labels:      map[string]string{"controller": "keep"},
					Annotations: map[string]string{"controller": "keep"},
				},
				Secrets:          []corev1.ObjectReference{{Name: "pre-existing", Namespace: "ns", UID: k8stypes.UID("secret-uid")}},
				ImagePullSecrets: []corev1.LocalObjectReference{{Name: "pre-existing-pull"}},
			}
			before := current.DeepCopy()
			plan := ServiceAccountV1Model{
				Metadata:                     []common.NamespacedMetadataModel{{}},
				Secret:                       types.SetNull(serviceAccountReferenceType),
				ImagePullSecret:              types.SetNull(serviceAccountReferenceType),
				AutomountServiceAccountToken: types.BoolValue(true),
			}
			if empty {
				plan.Secret = types.SetValueMust(serviceAccountReferenceType, []attr.Value{})
				plan.ImagePullSecret = types.SetValueMust(serviceAccountReferenceType, []attr.Value{})
			}
			ops, diags := serviceAccountAdoptionPatch(ctx, plan, current)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if len(ops) != 2 || ops[0].GetPath() != "/metadata/resourceVersion" || ops[1].GetPath() != "/automountServiceAccountToken" {
				t.Fatalf("omitted collections generated adoption patches: %#v", ops)
			}
			if !reflect.DeepEqual(current, before) {
				t.Fatalf("adoption altered pre-existing references: %#v", current)
			}
		})
	}
}

func TestServiceAccountInternalMetadataOwnership(t *testing.T) {
	desired := types.MapValueMust(types.StringType, map[string]attr.Value{"managed/key": types.StringValue("value")})
	ops := serviceAccountMetadataPatch("labels", types.MapNull(types.StringType), desired, map[string]string{"controller": "keep"})
	payload, err := ops.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var decoded []map[string]interface{}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 1 || decoded[0]["path"] != "/metadata/labels/managed~1key" || decoded[0]["op"] != "add" {
		t.Fatalf("first managed key replaced controller map: %s", payload)
	}
	ops = serviceAccountMetadataPatch("labels", desired, types.MapNull(types.StringType), map[string]string{"controller": "keep", "managed/key": "value"})
	payload, err = ops.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 1 || decoded[0]["path"] != "/metadata/labels/managed~1key" || decoded[0]["op"] != "remove" {
		t.Fatalf("removal changed controller map: %s", payload)
	}
}

func TestServiceAccountInternalReferencePresence(t *testing.T) {
	ctx := context.Background()
	for _, empty := range []bool{false, true} {
		prior := types.SetNull(serviceAccountReferenceType)
		if empty {
			prior = types.SetValueMust(serviceAccountReferenceType, []attr.Value{})
		}
		got, diags := serviceAccountReferenceSet(ctx, nil, prior)
		if diags.HasError() || !got.Equal(prior) {
			t.Fatalf("presence lost: got %v, want %v, diagnostics %v", got, prior, diags)
		}
	}
	got, diags := serviceAccountSDKReferences(ctx, nil)
	if diags.HasError() || !got.IsNull() {
		t.Fatalf("null SDK references changed: %v, %v", got, diags)
	}
	got, diags = serviceAccountSDKReferences(ctx, []serviceAccountSDKReferenceV0{})
	if diags.HasError() || got.IsNull() || len(got.Elements()) != 0 {
		t.Fatalf("empty SDK references changed: %v, %v", got, diags)
	}
}

func TestServiceAccountInternalMetadataPlanning(t *testing.T) {
	ctx := context.Background()
	empty := types.MapValueMust(types.StringType, map[string]attr.Value{})
	populated := types.MapValueMust(types.StringType, map[string]attr.Value{"old": types.StringValue("remove")})
	for _, tc := range []struct {
		name   string
		prior  types.Map
		config types.Map
		want   types.Map
	}{
		{"omitted_null", types.MapNull(types.StringType), types.MapNull(types.StringType), types.MapNull(types.StringType)},
		{"omitted_import_empty", empty, types.MapNull(types.StringType), empty},
		{"remove_managed_keys", populated, types.MapNull(types.StringType), types.MapNull(types.StringType)},
		{"explicit_empty", populated, empty, empty},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := planmodifier.MapResponse{PlanValue: tc.config}
			serviceAccountMetadataMapPlanModifier{}.PlanModifyMap(ctx, planmodifier.MapRequest{
				Plan:       tfsdk.Plan{Raw: tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{}}, map[string]tftypes.Value{})},
				StateValue: tc.prior, ConfigValue: tc.config,
			}, &resp)
			if !resp.PlanValue.Equal(tc.want) {
				t.Fatalf("got %v, want %v", resp.PlanValue, tc.want)
			}
		})
	}
}

type serviceAccountTestFilters struct{}

func (serviceAccountTestFilters) GetIgnoreAnnotations() []string { return nil }
func (serviceAccountTestFilters) GetIgnoreLabels() []string      { return nil }

func TestServiceAccountInternalFlattenNilAutomount(t *testing.T) {
	model := ServiceAccountV1Model{
		AutomountServiceAccountToken: types.BoolValue(true),
		Secret:                       types.SetNull(serviceAccountReferenceType), ImagePullSecret: types.SetNull(serviceAccountReferenceType),
	}
	diags := serviceAccountFlatten(context.Background(), &model, &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "account", Namespace: "ns"}}, serviceAccountTestFilters{})
	if diags.HasError() || model.AutomountServiceAccountToken.ValueBool() {
		t.Fatalf("nil automount did not flatten false: %#v, %v", model, diags)
	}
}

func TestServiceAccountInternalFlattenPreservesKnownLegacyToken(t *testing.T) {
	ctx := context.Background()
	const tokenName = "account-token-legacy"
	model := ServiceAccountV1Model{
		DefaultSecretName: types.StringValue(tokenName),
		Secret:            types.SetNull(serviceAccountReferenceType),
		ImagePullSecret:   types.SetNull(serviceAccountReferenceType),
	}
	account := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: "account", Namespace: "ns"},
		Secrets:    []corev1.ObjectReference{{Name: tokenName}, {Name: "user-secret"}},
	}
	diags := serviceAccountFlatten(ctx, &model, account, serviceAccountTestFilters{})
	if diags.HasError() || model.DefaultSecretName.ValueString() != tokenName {
		t.Fatalf("known legacy token lost: %v, %v", model.DefaultSecretName, diags)
	}
	names, diags := serviceAccountReferenceNames(ctx, model.Secret)
	if diags.HasError() || !reflect.DeepEqual(names, []string{"user-secret"}) {
		t.Fatalf("legacy token leaked into managed secrets: %v, %v", names, diags)
	}
}

func TestServiceAccountInternalImportedLegacyToken(t *testing.T) {
	for _, serverVersion := range []string{"v1.23.0", "v1.34.0"} {
		for _, name := range []string{"account", "default"} {
			t.Run(serverVersion+"/"+name, func(t *testing.T) {
				tokenName := name + "-token-legacy"
				account := &corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "ns", UID: k8stypes.UID("account-uid")},
					Secrets:    []corev1.ObjectReference{{Name: tokenName, UID: k8stypes.UID("token-uid")}},
				}
				token := &corev1.Secret{
					TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
					ObjectMeta: metav1.ObjectMeta{
						Name: tokenName, Namespace: "ns",
						Annotations: map[string]string{
							corev1.ServiceAccountNameKey: name,
							corev1.ServiceAccountUIDKey:  string(account.UID),
						},
					},
					Type: corev1.SecretTypeServiceAccountToken,
				}
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method != http.MethodGet {
						t.Errorf("discovery attempted a mutation: %s %s", r.Method, r.URL.Path)
						http.Error(w, "unexpected mutation", http.StatusMethodNotAllowed)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					var response any
					switch r.URL.Path {
					case "/version":
						response = map[string]string{"gitVersion": serverVersion}
					case "/api/v1/namespaces/ns/secrets/" + tokenName:
						response = token
					default:
						t.Errorf("unexpected discovery request: %s", r.URL.Path)
						http.NotFound(w, r)
						return
					}
					if err := json.NewEncoder(w).Encode(response); err != nil {
						t.Error(err)
					}
				}))
				defer server.Close()
				conn, err := clientset.NewForConfig(&rest.Config{Host: server.URL})
				if err != nil {
					t.Fatal(err)
				}
				ctx := context.Background()
				discovered, diags := serviceAccountDiscoverDefaultSecret(ctx, conn, account)
				if diags.HasError() || discovered != tokenName {
					t.Fatalf("import lost existing default token: name=%q diagnostics=%v", discovered, diags)
				}
				state := ServiceAccountV1Model{
					DefaultSecretName: types.StringValue(discovered),
					Secret:            types.SetNull(serviceAccountReferenceType),
					ImagePullSecret:   types.SetNull(serviceAccountReferenceType),
				}
				if diags := serviceAccountFlatten(ctx, &state, account, serviceAccountTestFilters{}); diags.HasError() {
					t.Fatal(diags)
				}
				if len(state.Secret.Elements()) != 0 {
					t.Fatal("controller token was exposed as a managed secret")
				}
				plan := state
				plan.Secret = types.SetValueMust(serviceAccountReferenceType, []attr.Value{})
				ops, diags := serviceAccountPatch(ctx, state, plan, account)
				if diags.HasError() || len(ops) != 0 {
					t.Fatalf("omitted secret configuration would alter the imported token: %#v, %v", ops, diags)
				}
			})
		}
	}
}

func TestServiceAccountInternalConcreteModelRoundtrip(t *testing.T) {
	ctx := context.Background()
	for _, defaultAccount := range []bool{false, true} {
		s := serviceAccountSchema(ctx, defaultAccount)
		state := tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
		name := "account"
		if defaultAccount {
			name = "default"
		}
		original := ServiceAccountV1Model{
			ID: types.StringValue("ns/" + name),
			Metadata: []common.NamespacedMetadataModel{{
				MetadataModel: common.MetadataModel{
					MetadataBase: common.MetadataBase{
						Name: types.StringValue(name), UID: types.StringValue("uid"),
						ResourceVersion: types.StringValue("123"), Generation: types.Int64Value(0),
						Labels:      types.MapNull(types.StringType),
						Annotations: types.MapValueMust(types.StringType, map[string]attr.Value{}),
					},
					GenerateName: types.StringNull(),
				},
				Namespace: types.StringValue("ns"),
			}},
			Secret:                       types.SetNull(serviceAccountReferenceType),
			ImagePullSecret:              types.SetValueMust(serviceAccountReferenceType, []attr.Value{}),
			AutomountServiceAccountToken: types.BoolValue(false),
			DefaultSecretName:            types.StringValue(""),
			Timeouts:                     timeouts.Value{Object: types.ObjectNull(timeoutsAttributeTypes(s))},
		}
		diags := serviceAccountWriteModel(ctx, &state, original, defaultAccount)
		if diags.HasError() {
			t.Fatal(diags)
		}
		originalRaw := state.Raw
		model, diags := serviceAccountReadModel(ctx, state, defaultAccount)
		if diags.HasError() || !model.ID.Equal(original.ID) || len(model.Metadata) != 1 || !model.Metadata[0].GenerateName.IsNull() {
			t.Fatalf("state decode failed: %#v, %v", model, diags)
		}
		diags = serviceAccountWriteModel(ctx, &state, model, defaultAccount)
		if diags.HasError() {
			t.Fatalf("state encode failed: %v", diags)
		}
		if !state.Raw.Equal(originalRaw) {
			t.Fatalf("state changed during roundtrip: got %s, want %s", state.Raw, originalRaw)
		}
	}
}

func TestServiceAccountInternalConfigureErrors(t *testing.T) {
	var callback func() any
	resp := resource.ConfigureResponse{}
	serviceAccountConfigure(resource.ConfigureRequest{ProviderData: 1}, &resp, &callback)
	if !resp.Diagnostics.HasError() {
		t.Fatal("wrong callback type accepted")
	}
	for _, callback := range []func() any{nil, func() any { return nil }, func() any { return 1 }} {
		_, _, diags := serviceAccountClient(callback)
		if !diags.HasError() {
			t.Fatal("invalid metadata accepted")
		}
	}
}

func TestServiceAccountInternalGenerateNamePlan(t *testing.T) {
	ctx := context.Background()
	metadata := serviceAccountSchema(ctx, false).Blocks["metadata"].(schema.ListNestedBlock)
	field := metadata.NestedObject.Attributes["generate_name"].(schema.StringAttribute)
	rawType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{}}
	present := tftypes.NewValue(rawType, map[string]tftypes.Value{})
	absent := tftypes.NewValue(rawType, nil)
	for _, tc := range []struct {
		name                   string
		prior, config, planned types.String
		want                   types.String
		create, destroy        bool
		replace                bool
	}{
		{"create_omitted", types.StringNull(), types.StringNull(), types.StringUnknown(), types.StringValue(""), true, false, false},
		{"sdk_empty", types.StringValue(""), types.StringNull(), types.StringValue(""), types.StringValue(""), false, false, false},
		{"raw_null", types.StringNull(), types.StringNull(), types.StringNull(), types.StringNull(), false, false, false},
		{"unrelated_update", types.StringValue(""), types.StringNull(), types.StringUnknown(), types.StringValue(""), false, false, false},
		{"prefix_unchanged", types.StringValue("old-"), types.StringValue("old-"), types.StringValue("old-"), types.StringValue("old-"), false, false, false},
		{"prefix_changed", types.StringValue("old-"), types.StringValue("new-"), types.StringValue("new-"), types.StringValue("new-"), false, false, true},
		{"prefix_removed", types.StringValue("old-"), types.StringNull(), types.StringValue("old-"), types.StringValue(""), false, false, true},
		{"destroy", types.StringValue("old-"), types.StringNull(), types.StringNull(), types.StringNull(), false, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := planmodifier.StringRequest{
				State: tfsdk.State{Raw: present}, Plan: tfsdk.Plan{Raw: present},
				StateValue: tc.prior, ConfigValue: tc.config, PlanValue: tc.planned,
			}
			if tc.create {
				req.State.Raw = absent
			}
			if tc.destroy {
				req.Plan.Raw = absent
			}
			resp := planmodifier.StringResponse{PlanValue: tc.planned}
			for _, modifier := range field.PlanModifiers {
				modifier.PlanModifyString(ctx, req, &resp)
				req.PlanValue = resp.PlanValue
			}
			if resp.Diagnostics.HasError() || !resp.PlanValue.Equal(tc.want) || resp.RequiresReplace != tc.replace {
				t.Fatalf("plan=%s replacement=%t diagnostics=%v; want plan=%s replacement=%t",
					resp.PlanValue, resp.RequiresReplace, resp.Diagnostics, tc.want, tc.replace)
			}
		})
	}
}

type serviceAccountInternalProvider struct{}

func (serviceAccountInternalProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "kubernetes"
}

func (serviceAccountInternalProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = providerschema.Schema{}
}

func (serviceAccountInternalProvider) Configure(context.Context, provider.ConfigureRequest, *provider.ConfigureResponse) {
}

func (serviceAccountInternalProvider) DataSources(context.Context) []func() datasource.DataSource {
	return nil
}

func (serviceAccountInternalProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{NewServiceAccountV1, NewDefaultServiceAccountV1}
}

func TestServiceAccountInternalLateMetadataRemovalPlan(t *testing.T) {
	ctx := context.Background()
	server := providerserver.NewProtocol6(serviceAccountInternalProvider{})()
	empty := types.MapValueMust(types.StringType, map[string]attr.Value{})
	populated := types.MapValueMust(types.StringType, map[string]attr.Value{"managed": types.StringValue("old")})
	for _, defaultAccount := range []bool{false, true} {
		for _, tc := range []struct {
			name       string
			field      string
			prior      types.Map
			config     types.Map
			wantChange bool
		}{
			{"remove_labels", "labels", populated, types.MapNull(types.StringType), true},
			{"remove_annotations", "annotations", populated, types.MapNull(types.StringType), true},
			{"unchanged_labels", "labels", populated, populated, false},
			{"sdk_empty_omitted", "labels", empty, types.MapNull(types.StringType), false},
			{"sdk_null_omitted", "annotations", types.MapNull(types.StringType), types.MapNull(types.StringType), false},
		} {
			t.Run(fmt.Sprintf("default=%t/%s", defaultAccount, tc.name), func(t *testing.T) {
				s := serviceAccountSchema(ctx, defaultAccount)
				state := tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
				name, resourceType := "account", "kubernetes_service_account_v1"
				if defaultAccount {
					name, resourceType = "default", "kubernetes_default_service_account_v1"
				}
				model := ServiceAccountV1Model{
					ID: types.StringValue("ns/" + name),
					Metadata: []common.NamespacedMetadataModel{{
						MetadataModel: common.MetadataModel{
							MetadataBase: common.MetadataBase{
								Name: types.StringValue(name), UID: types.StringValue("uid"),
								ResourceVersion: types.StringValue("100"), Generation: types.Int64Value(0),
								Labels: types.MapNull(types.StringType), Annotations: types.MapNull(types.StringType),
							},
							GenerateName: types.StringNull(),
						},
						Namespace: types.StringValue("ns"),
					}},
					Secret:                       types.SetValueMust(serviceAccountReferenceType, []attr.Value{}),
					ImagePullSecret:              types.SetValueMust(serviceAccountReferenceType, []attr.Value{}),
					AutomountServiceAccountToken: types.BoolValue(true),
					DefaultSecretName:            types.StringValue(name + "-token-legacy"),
					Timeouts:                     timeouts.Value{Object: types.ObjectNull(timeoutsAttributeTypes(s))},
				}
				if tc.field == "labels" {
					model.Metadata[0].Labels = tc.prior
				} else {
					model.Metadata[0].Annotations = tc.prior
				}
				if diags := serviceAccountWriteModel(ctx, &state, model, defaultAccount); diags.HasError() {
					t.Fatal(diags)
				}
				config := tfsdk.State{Schema: s, Raw: state.Raw}
				for _, key := range []string{"id", "default_secret_name"} {
					if diags := config.SetAttribute(ctx, path.Root(key), types.StringNull()); diags.HasError() {
						t.Fatal(diags)
					}
				}
				for _, key := range []string{"uid", "resource_version"} {
					if diags := config.SetAttribute(ctx, path.Root("metadata").AtListIndex(0).AtName(key), types.StringNull()); diags.HasError() {
						t.Fatal(diags)
					}
				}
				if diags := config.SetAttribute(ctx, path.Root("metadata").AtListIndex(0).AtName("generation"), types.Int64Null()); diags.HasError() {
					t.Fatal(diags)
				}
				fieldPath := path.Root("metadata").AtListIndex(0).AtName(tc.field)
				if diags := config.SetAttribute(ctx, fieldPath, tc.config); diags.HasError() {
					t.Fatal(diags)
				}
				prior, err := tfprotov6.NewDynamicValue(s.Type().TerraformType(ctx), state.Raw)
				if err != nil {
					t.Fatal(err)
				}
				configValue, err := tfprotov6.NewDynamicValue(s.Type().TerraformType(ctx), config.Raw)
				if err != nil {
					t.Fatal(err)
				}
				// Core's proposed state inherits Optional+Computed prior maps.
				// Thus it is initially identical even when configuration removes
				// the last managed map; only our attribute modifier reveals it.
				resp, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
					TypeName: resourceType, PriorState: &prior, ProposedNewState: &prior, Config: &configValue,
				})
				if err != nil {
					t.Fatal(err)
				}
				for _, diagnostic := range resp.Diagnostics {
					if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
						t.Fatalf("planning error: %s: %s", diagnostic.Summary, diagnostic.Detail)
					}
				}
				if len(resp.RequiresReplace) != 0 {
					t.Fatalf("metadata planning unexpectedly requires replacement: %v", resp.RequiresReplace)
				}
				raw, err := resp.PlannedState.Unmarshal(s.Type().TerraformType(ctx))
				if err != nil {
					t.Fatal(err)
				}
				if !tc.wantChange && !raw.Equal(state.Raw) {
					t.Fatalf("unchanged SDK state no longer plans empty:\nprior: %s\nplan: %s", state.Raw, raw)
				}
				plan, diags := serviceAccountReadModel(ctx, tfsdk.Plan{Schema: s, Raw: raw}, defaultAccount)
				if diags.HasError() {
					t.Fatal(diags)
				}
				if tc.wantChange {
					if !plan.Metadata[0].ResourceVersion.IsUnknown() || !plan.Metadata[0].Generation.IsUnknown() {
						t.Fatalf("late metadata removal retained mutable computed values: resource_version=%s, generation=%s",
							plan.Metadata[0].ResourceVersion, plan.Metadata[0].Generation)
					}
				}
				if !plan.ID.Equal(model.ID) || !plan.Metadata[0].UID.Equal(model.Metadata[0].UID) ||
					!plan.DefaultSecretName.Equal(model.DefaultSecretName) {
					t.Fatal("metadata removal changed stable identity or token fields")
				}
			})
		}
	}
}

func TestServiceAccountInternalPatchRejectsConcurrentChanges(t *testing.T) {
	ctx := context.Background()
	current := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: "account", Namespace: "ns", ResourceVersion: "100",
			Labels: map[string]string{"managed": "old", "controller": "keep"},
		},
		Secrets: []corev1.ObjectReference{{Name: "old-secret"}},
	}
	prior := ServiceAccountV1Model{
		Metadata: []common.NamespacedMetadataModel{{MetadataModel: common.MetadataModel{
			MetadataBase: common.MetadataBase{
				Labels:      types.MapValueMust(types.StringType, map[string]attr.Value{"managed": types.StringValue("old")}),
				Annotations: types.MapNull(types.StringType),
			},
		}}},
		ImagePullSecret:              types.SetNull(serviceAccountReferenceType),
		AutomountServiceAccountToken: types.BoolValue(true),
	}
	var diags diag.Diagnostics
	prior.Secret, diags = serviceAccountReferenceSet(ctx, []string{"old-secret"}, types.SetNull(serviceAccountReferenceType))
	if diags.HasError() {
		t.Fatal(diags)
	}
	plan := prior
	plan.Metadata = slices.Clone(prior.Metadata)
	plan.Metadata[0].Labels = types.MapNull(types.StringType)
	plan.Secret = types.SetValueMust(serviceAccountReferenceType, []attr.Value{})
	ops, diags := serviceAccountPatch(ctx, prior, plan, current)
	if diags.HasError() {
		t.Fatal(diags)
	}
	payload, err := ops.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	patch, err := jsonpatch.DecodePatch(payload)
	if err != nil {
		t.Fatal(err)
	}
	original, err := json.Marshal(current)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := patch.Apply(original); err != nil {
		t.Fatalf("patch rejected the version it was built against: %s: %v", payload, err)
	}
	concurrent := current.DeepCopy()
	concurrent.ResourceVersion = "101"
	concurrent.Labels["managed"] = "changed-concurrently"
	concurrent.Secrets = append(concurrent.Secrets, corev1.ObjectReference{Name: "added-concurrently"})
	changed, err := json.Marshal(concurrent)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := patch.Apply(changed); err == nil {
		t.Fatalf("patch overwrote concurrent metadata/reference edits: %s", payload)
	}
}
