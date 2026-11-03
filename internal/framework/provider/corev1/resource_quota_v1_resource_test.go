// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
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
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/corev1"

	k8sv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func resourceQuotaSchema(t *testing.T) schema.Schema {
	t.Helper()
	var resp fwresource.SchemaResponse
	(&corev1.ResourceQuotaV1{}).Schema(context.Background(), fwresource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if diags := resp.Schema.ValidateImplementation(context.Background()); diags.HasError() {
		t.Fatalf("schema is not a valid implementation: %v", diags)
	}
	return resp.Schema
}

// The two state bodies are what provider 3.4.0 (SDKv2) wrote to state, captured from a
// live apply. The model must decode both, including an unset scopes stored as null and as [].
func TestResourceQuotaV1ModelDecodesSDKv2State(t *testing.T) {
	ctx := context.Background()
	s := resourceQuotaSchema(t)

	cases := map[string]struct {
		json      string
		wantHard  int
		scopesNil bool
	}{
		"minimal, scopes stored as null": {
			json: `{"id":"ns/rq","timeouts":null,
				"metadata":[{"annotations":null,"labels":null,"generate_name":"","name":"rq","namespace":"ns","generation":0,"resource_version":"1","uid":"u"}],
				"spec":[{"hard":{"pods":"4"},"scope_selector":[],"scopes":null}]}`,
			wantHard:  1,
			scopesNil: true,
		},
		"no spec configured, SDKv2 stored an empty spec": {
			json: `{"id":"ns/rq","timeouts":null,
				"metadata":[{"annotations":null,"labels":null,"generate_name":"rq-","name":"rq-abc","namespace":"ns","generation":0,"resource_version":"1","uid":"u"}],
				"spec":[{"hard":{},"scope_selector":[],"scopes":[]}]}`,
			wantHard: 0,
		},
		"scopes and scope_selector": {
			json: `{"id":"ns/rq","timeouts":null,
				"metadata":[{"annotations":{"a":"b"},"labels":{"l":"v"},"generate_name":"","name":"rq","namespace":"ns","generation":0,"resource_version":"1","uid":"u"}],
				"spec":[{"hard":{"limits.memory":"2Gi","pods":"4"},"scopes":["BestEffort"],
					"scope_selector":[{"match_expression":[{"scope_name":"PriorityClass","operator":"In","values":["medium"]}]}]}]}`,
			wantHard: 2,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			raw := tfprotov6.RawState{JSON: []byte(tc.json)}
			value, err := raw.Unmarshal(s.Type().TerraformType(ctx))
			if err != nil {
				t.Fatal(err)
			}
			state := tfsdk.State{Schema: s, Raw: value}

			var model corev1.ResourceQuotaV1Model
			if diags := state.Get(ctx, &model); diags.HasError() {
				t.Fatal(diags)
			}
			if len(model.Metadata) != 1 || len(model.Spec) != 1 {
				t.Fatalf("metadata=%d spec=%d, want 1 and 1", len(model.Metadata), len(model.Spec))
			}
			if got := len(model.Spec[0].Hard.Elements()); got != tc.wantHard {
				t.Errorf("hard has %d elements, want %d", got, tc.wantHard)
			}
			if model.Spec[0].Scopes.IsNull() != tc.scopesNil {
				t.Errorf("scopes null = %v, want %v", model.Spec[0].Scopes.IsNull(), tc.scopesNil)
			}
			if !model.Spec[0].Hard.ElementType(ctx).Equal(common.QuantityType{}) {
				t.Errorf("hard element type = %s, want QuantityType", model.Spec[0].Hard.ElementType(ctx))
			}

			// And it must write back: this is the path every Create/Update/Read ends on.
			out := tfsdk.State{Schema: s}
			if diags := out.Set(ctx, &model); diags.HasError() {
				t.Fatalf("model does not write back: %v", diags)
			}
			if !out.Raw.Equal(value) {
				t.Errorf("round trip changed the state:\n got %s\nwant %s", out.Raw, value)
			}
		})
	}
}

// scopes is ForceNew in SDKv2. SDKv2 stores an unset set as either null or [], so a plain
// RequiresReplace would replace the quota when an upgraded [] meets a null plan.
func TestResourceQuotaV1ScopesRequiresReplace(t *testing.T) {
	ctx := context.Background()
	s := resourceQuotaSchema(t)

	specBlock := s.Blocks["spec"].(schema.ListNestedBlock)
	scopes := specBlock.NestedObject.Attributes["scopes"].(schema.SetAttribute)
	if len(scopes.PlanModifiers) != 1 {
		t.Fatalf("scopes has %d plan modifiers, want 1", len(scopes.PlanModifiers))
	}
	modifier := scopes.PlanModifiers[0]

	set := func(v ...string) types.Set {
		elems := make([]attr.Value, len(v))
		for i, x := range v {
			elems[i] = types.StringValue(x)
		}
		return types.SetValueMust(types.StringType, elems)
	}
	null := types.SetNull(types.StringType)

	cases := []struct {
		name     string
		state    types.Set
		plan     types.Set
		wantRepl bool
	}{
		{"empty to null is not a change", set(), null, false},
		{"null to empty is not a change", null, set(), false},
		{"null to null", null, null, false},
		{"same scopes", set("BestEffort"), set("BestEffort"), false},
		{"scope changed", set("BestEffort"), set("NotBestEffort"), true},
		{"scope added", null, set("BestEffort"), true},
		{"scope added from empty", set(), set("BestEffort"), true},
		{"scope removed", set("BestEffort"), null, true},
		{"scope removed to empty", set("BestEffort"), set(), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// The framework's wrapper returns early when the whole state or plan is null
			// (create and destroy), so both must be non-null for the modifier to run.
			live := tftypes.NewValue(tftypes.Object{}, map[string]tftypes.Value{})
			req := planmodifier.SetRequest{
				StateValue:  tc.state,
				PlanValue:   tc.plan,
				ConfigValue: tc.plan,
				State:       tfsdk.State{Raw: live},
				Plan:        tfsdk.Plan{Raw: live},
			}
			resp := &planmodifier.SetResponse{PlanValue: tc.plan}
			modifier.PlanModifySet(ctx, req, resp)
			if resp.RequiresReplace != tc.wantRepl {
				t.Errorf("RequiresReplace = %v, want %v", resp.RequiresReplace, tc.wantRepl)
			}
		})
	}
}

// ---- CRUD against a fake API server -----------------------------------------------------
//
// Same approach as TestNamespaceUpdateNoPatch: the exported Create/Read/Update/Delete run against
// an httptest server, so expanding and flattening the spec is covered through the public surface.

type quotaCall struct{ Method, Path, Body string }

type quotaAPI struct {
	mu    sync.Mutex
	calls []quotaCall
}

func (a *quotaAPI) recorded() []quotaCall {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]quotaCall(nil), a.calls...)
}

func (a *quotaAPI) count(method string) int {
	n := 0
	for _, c := range a.recorded() {
		if c.Method == method {
			n++
		}
	}
	return n
}

// newQuotaAPI starts a fake API server. handle returns the HTTP status and the object to encode;
// a 404 is encoded as a Kubernetes Status so client-go reports NotFound.
func newQuotaAPI(t *testing.T, handle func(call quotaCall) (int, any)) (*quotaAPI, namespaceUpdateClientsets) {
	t.Helper()
	api := &quotaAPI{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		call := quotaCall{Method: r.Method, Path: r.URL.Path, Body: string(body)}
		api.mu.Lock()
		api.calls = append(api.calls, call)
		api.mu.Unlock()

		status, obj := handle(call)
		if status == http.StatusNotFound {
			obj = &metav1.Status{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Status"},
				Status:   metav1.StatusFailure, Reason: metav1.StatusReasonNotFound, Code: http.StatusNotFound,
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if obj != nil {
			_ = json.NewEncoder(w).Encode(obj)
		}
	}))
	t.Cleanup(server.Close)
	client, err := k8sclient.NewForConfig(&rest.Config{Host: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	return api, namespaceUpdateClientsets{client: client}
}

func quotaObject(rv string, specHard, statusHard map[string]string) *k8sv1.ResourceQuota {
	list := func(in map[string]string) k8sv1.ResourceList {
		out := k8sv1.ResourceList{}
		for k, v := range in {
			out[k8sv1.ResourceName(k)] = k8sresource.MustParse(v)
		}
		return out
	}
	return &k8sv1.ResourceQuota{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "ResourceQuota"},
		ObjectMeta: metav1.ObjectMeta{Name: "quota", Namespace: "team", UID: "u1", ResourceVersion: rv},
		Spec:       k8sv1.ResourceQuotaSpec{Hard: list(specHard)},
		Status:     k8sv1.ResourceQuotaStatus{Hard: list(statusHard)},
	}
}

func quotaTimeouts(t *testing.T, s schema.Schema, create string) timeouts.Value {
	t.Helper()
	attrTypes := s.Blocks["timeouts"].Type().(attr.TypeWithAttributeTypes).AttributeTypes()
	if create == "" {
		return timeouts.Value{Object: types.ObjectNull(attrTypes)}
	}
	obj := types.ObjectValueMust(attrTypes, map[string]attr.Value{
		"create": types.StringValue(create), "update": types.StringNull(),
	})
	return timeouts.Value{Object: obj}
}

// quotaFixture is a resource quota as it sits in state after an apply.
func quotaFixture(t *testing.T, s schema.Schema, hard map[string]string) corev1.ResourceQuotaV1Model {
	t.Helper()
	return corev1.ResourceQuotaV1Model{
		ID: types.StringValue("team/quota"),
		Metadata: []common.NamespacedMetadataModel{{
			MetadataModel: common.MetadataModel{
				MetadataBase: common.MetadataBase{
					Annotations: types.MapNull(types.StringType), Labels: types.MapNull(types.StringType),
					Name: types.StringValue("quota"), Generation: types.Int64Value(0),
					ResourceVersion: types.StringValue("1"), UID: types.StringValue("u1"),
				},
				GenerateName: types.StringNull(),
			},
			Namespace: types.StringValue("team"),
		}},
		Spec: []corev1.ResourceQuotaSpecModel{{
			Hard: quotaHard(t, hard), Scopes: types.SetNull(types.StringType),
			ScopeSelector: []corev1.ResourceQuotaScopeSelectorModel{},
		}},
		Timeouts: quotaTimeouts(t, s, ""),
	}
}

func quotaHard(t *testing.T, in map[string]string) types.Map {
	t.Helper()
	elems := map[string]attr.Value{}
	for k, v := range in {
		elems[k] = common.NewQuantityValue(v)
	}
	m, diags := types.MapValue(common.QuantityType{}, elems)
	if diags.HasError() {
		t.Fatal(diags)
	}
	return m
}

func quotaStateOf(t *testing.T, s schema.Schema, m corev1.ResourceQuotaV1Model) tfsdk.State {
	t.Helper()
	st := tfsdk.State{Schema: s}
	if diags := st.Set(context.Background(), &m); diags.HasError() {
		t.Fatal(diags)
	}
	return st
}

func quotaIdentity() *tfsdk.ResourceIdentity {
	s := common.NamespacedIdentitySchema()
	return &tfsdk.ResourceIdentity{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(context.Background()), nil)}
}

func quotaResource(clients namespaceUpdateClientsets) *corev1.ResourceQuotaV1 {
	return &corev1.ResourceQuotaV1{SDKv2Meta: func() any { return clients }}
}

func quotaHardOf(t *testing.T, st tfsdk.State) map[string]string {
	t.Helper()
	var m corev1.ResourceQuotaV1Model
	if diags := st.Get(context.Background(), &m); diags.HasError() {
		t.Fatal(diags)
	}
	out := map[string]string{}
	for k, v := range m.Spec[0].Hard.Elements() {
		out[k] = v.(common.QuantityValue).ValueString()
	}
	return out
}

func TestResourceQuotaCreate(t *testing.T) {
	ctx := context.Background()
	s := resourceQuotaSchema(t)
	nullState := func() tfsdk.State {
		return tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
	}
	// What Terraform hands Create: the server-assigned attributes are unknown.
	plannedFor := func(create string) tfsdk.Plan {
		m := quotaFixture(t, s, map[string]string{"limits.memory": "2048Mi"})
		m.ID = types.StringUnknown()
		m.Metadata[0].Generation = types.Int64Unknown()
		m.Metadata[0].ResourceVersion = types.StringUnknown()
		m.Metadata[0].UID = types.StringUnknown()
		m.Timeouts = quotaTimeouts(t, s, create)
		return tfsdk.Plan{Schema: s, Raw: quotaStateOf(t, s, m).Raw}
	}

	t.Run("waits for status.hard and keeps the planned spelling", func(t *testing.T) {
		gets := 0
		api, clients := newQuotaAPI(t, func(c quotaCall) (int, any) {
			switch c.Method {
			case http.MethodPost:
				return http.StatusCreated, quotaObject("5", map[string]string{"limits.memory": "2Gi"}, nil)
			default:
				gets++
				if gets == 1 { // the quota controller has not published status yet
					return http.StatusOK, quotaObject("6", map[string]string{"limits.memory": "2Gi"}, nil)
				}
				return http.StatusOK, quotaObject("7", map[string]string{"limits.memory": "2Gi"}, map[string]string{"limits.memory": "2Gi"})
			}
		})
		resp := fwresource.CreateResponse{State: nullState(), Identity: quotaIdentity()}
		quotaResource(clients).Create(ctx, fwresource.CreateRequest{Plan: plannedFor("")}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}
		if api.count(http.MethodPost) != 1 || api.count(http.MethodGet) < 2 {
			t.Errorf("calls = %+v, want one POST and a GET that retried", api.recorded())
		}
		var got corev1.ResourceQuotaV1Model
		resp.State.Get(ctx, &got)
		if got.ID.ValueString() != "team/quota" || got.Metadata[0].UID.ValueString() != "u1" || got.Metadata[0].ResourceVersion.ValueString() != "7" {
			t.Errorf("server-assigned fields not resolved: %+v", got.Metadata[0])
		}
		if h := quotaHardOf(t, resp.State); h["limits.memory"] != "2048Mi" {
			t.Errorf("hard = %v, want the planned spelling 2048Mi, not the API's 2Gi", h)
		}
		var id common.NamespacedResourceIdentity
		resp.Identity.Get(ctx, &id)
		if id.Kind.ValueString() != "ResourceQuota" || id.APIVersion.ValueString() != "v1" || id.Name.ValueString() != "quota" || id.Namespace.ValueString() != "team" {
			t.Errorf("identity = %+v", id)
		}
	})

	t.Run("saves state when status.hard never settles", func(t *testing.T) {
		_, clients := newQuotaAPI(t, func(c quotaCall) (int, any) {
			if c.Method == http.MethodPost {
				return http.StatusCreated, quotaObject("5", map[string]string{"limits.memory": "2Gi"}, nil)
			}
			return http.StatusOK, quotaObject("6", map[string]string{"limits.memory": "2Gi"}, nil)
		})
		resp := fwresource.CreateResponse{State: nullState(), Identity: quotaIdentity()}
		quotaResource(clients).Create(ctx, fwresource.CreateRequest{Plan: plannedFor("1s")}, &resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected a timeout error")
		}
		if resp.State.Raw.IsNull() {
			t.Fatal("the created quota must stay in state, or the next apply fails on a name conflict")
		}
		if resp.Identity.Raw.IsNull() {
			t.Error("identity must be set once the object exists")
		}
	})

	t.Run("API error leaves state empty", func(t *testing.T) {
		_, clients := newQuotaAPI(t, func(quotaCall) (int, any) { return http.StatusForbidden, nil })
		resp := fwresource.CreateResponse{State: nullState(), Identity: quotaIdentity()}
		quotaResource(clients).Create(ctx, fwresource.CreateRequest{Plan: plannedFor("")}, &resp)
		if !resp.Diagnostics.HasError() || !resp.State.Raw.IsNull() {
			t.Fatalf("diagnostics: %v, state null: %v", resp.Diagnostics, resp.State.Raw.IsNull())
		}
	})
}

func TestResourceQuotaRead(t *testing.T) {
	ctx := context.Background()
	s := resourceQuotaSchema(t)
	serve := func(spec k8sv1.ResourceQuotaSpec) namespaceUpdateClientsets {
		q := quotaObject("9", nil, nil)
		q.Spec = spec
		_, clients := newQuotaAPI(t, func(quotaCall) (int, any) { return http.StatusOK, q })
		return clients
	}
	read := func(clients namespaceUpdateClientsets, state tfsdk.State) fwresource.ReadResponse {
		resp := fwresource.ReadResponse{State: state, Identity: quotaIdentity()}
		quotaResource(clients).Read(ctx, fwresource.ReadRequest{State: state}, &resp)
		return resp
	}
	gi := func(v string) k8sv1.ResourceList {
		return k8sv1.ResourceList{"limits.memory": k8sresource.MustParse(v)}
	}

	t.Run("keeps the prior spelling of an equivalent quantity", func(t *testing.T) {
		state := quotaStateOf(t, s, quotaFixture(t, s, map[string]string{"limits.memory": "2048Mi"}))
		resp := read(serve(k8sv1.ResourceQuotaSpec{Hard: gi("2Gi")}), state)
		if resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}
		if h := quotaHardOf(t, resp.State); h["limits.memory"] != "2048Mi" {
			t.Errorf("hard = %v, want 2048Mi", h)
		}
	})

	t.Run("a quantity changed outside Terraform takes the API value", func(t *testing.T) {
		state := quotaStateOf(t, s, quotaFixture(t, s, map[string]string{"limits.memory": "2048Mi"}))
		resp := read(serve(k8sv1.ResourceQuotaSpec{Hard: gi("3Gi")}), state)
		if h := quotaHardOf(t, resp.State); h["limits.memory"] != "3Gi" {
			t.Errorf("hard = %v, want 3Gi", h)
		}
	})

	t.Run("unset collections stay null and an empty hard stays empty", func(t *testing.T) {
		m := quotaFixture(t, s, map[string]string{})
		m.Metadata[0].ResourceVersion = types.StringValue("9") // what the fake API serves
		state := quotaStateOf(t, s, m)
		resp := read(serve(k8sv1.ResourceQuotaSpec{}), state)
		var got corev1.ResourceQuotaV1Model
		resp.State.Get(ctx, &got)
		sp := got.Spec[0]
		if sp.Hard.IsNull() || len(sp.Hard.Elements()) != 0 {
			t.Error("hard = {} must stay empty, not null")
		}
		if !sp.Scopes.IsNull() || len(sp.ScopeSelector) != 0 {
			t.Error("unset scopes must stay null and scope_selector empty")
		}
		if !state.Raw.Equal(resp.State.Raw) {
			t.Errorf("a refresh with no drift must not change state:\n got %s\nwant %s", resp.State.Raw, state.Raw)
		}
	})

	t.Run("import reads everything from the API", func(t *testing.T) {
		st := tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
		if diags := st.SetAttribute(ctx, path.Root("id"), "team/quota"); diags.HasError() {
			t.Fatal(diags)
		}
		resp := read(serve(k8sv1.ResourceQuotaSpec{
			Hard:   gi("1Gi"),
			Scopes: []k8sv1.ResourceQuotaScope{"BestEffort"},
			ScopeSelector: &k8sv1.ScopeSelector{MatchExpressions: []k8sv1.ScopedResourceSelectorRequirement{
				{ScopeName: "PriorityClass", Operator: "In", Values: []string{"x"}},
				{ScopeName: "BestEffort", Operator: "Exists"},
			}},
		}), st)
		if resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}
		var got corev1.ResourceQuotaV1Model
		resp.State.Get(ctx, &got)
		sp := got.Spec[0]
		if len(sp.Hard.Elements()) != 1 || len(sp.Scopes.Elements()) != 1 || len(sp.ScopeSelector) != 1 {
			t.Fatalf("incomplete import: %+v", sp)
		}
		exprs := sp.ScopeSelector[0].MatchExpression
		if len(exprs[0].Values.Elements()) != 1 || !exprs[1].Values.IsNull() {
			t.Error("values: a populated set stays a set and an absent one stays null")
		}
		if got.Metadata[0].Namespace.ValueString() != "team" || got.Metadata[0].UID.ValueString() != "u1" {
			t.Errorf("metadata not populated: %+v", got.Metadata[0])
		}
	})

	t.Run("generate_name is kept when the API omits it", func(t *testing.T) {
		m := quotaFixture(t, s, map[string]string{})
		m.Metadata[0].GenerateName = types.StringValue("quota-")
		resp := read(serve(k8sv1.ResourceQuotaSpec{}), quotaStateOf(t, s, m))
		var got corev1.ResourceQuotaV1Model
		resp.State.Get(ctx, &got)
		if got.Metadata[0].GenerateName.ValueString() != "quota-" {
			t.Errorf("generate_name = %v, want quota-", got.Metadata[0].GenerateName)
		}
	})
	// SDKv2 stored an unset generate_name as "". Read must turn it into null, or an upgraded
	// resource plans an update from "" to null on every first plan.
	t.Run("an SDKv2 empty generate_name becomes null", func(t *testing.T) {
		m := quotaFixture(t, s, map[string]string{})
		m.Metadata[0].GenerateName = types.StringValue("")
		resp := read(serve(k8sv1.ResourceQuotaSpec{}), quotaStateOf(t, s, m))
		var got corev1.ResourceQuotaV1Model
		resp.State.Get(ctx, &got)
		if !got.Metadata[0].GenerateName.IsNull() {
			t.Errorf("generate_name = %v, want null", got.Metadata[0].GenerateName)
		}
	})

	t.Run("NotFound removes the resource and still sets identity", func(t *testing.T) {
		_, clients := newQuotaAPI(t, func(quotaCall) (int, any) { return http.StatusNotFound, nil })
		resp := read(clients, quotaStateOf(t, s, quotaFixture(t, s, map[string]string{})))
		if resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}
		if !resp.State.Raw.IsNull() {
			t.Error("state must be removed")
		}
		if resp.Identity.Raw.IsNull() {
			t.Error("identity must be set on removal for Terraform before 1.12")
		}
	})

	t.Run("an invalid id is rejected without calling the API", func(t *testing.T) {
		api, clients := newQuotaAPI(t, func(quotaCall) (int, any) { return http.StatusOK, nil })
		for _, id := range []string{"quota", "/quota", "team/"} {
			m := quotaFixture(t, s, map[string]string{})
			m.ID = types.StringValue(id)
			if resp := read(clients, quotaStateOf(t, s, m)); !resp.Diagnostics.HasError() {
				t.Errorf("id %q: expected an error", id)
			}
		}
		if len(api.recorded()) != 0 {
			t.Errorf("unexpected API calls: %+v", api.recorded())
		}
	})
}

func TestResourceQuotaUpdate(t *testing.T) {
	ctx := context.Background()
	s := resourceQuotaSchema(t)
	update := func(clients namespaceUpdateClientsets, state, plan corev1.ResourceQuotaV1Model) fwresource.UpdateResponse {
		st := quotaStateOf(t, s, state)
		plan.Metadata[0].ResourceVersion = types.StringUnknown()
		resp := fwresource.UpdateResponse{State: st, Identity: quotaIdentity()}
		quotaResource(clients).Update(ctx, fwresource.UpdateRequest{
			State: st, Plan: tfsdk.Plan{Schema: s, Raw: quotaStateOf(t, s, plan).Raw},
		}, &resp)
		return resp
	}

	t.Run("a changed spec is replaced and awaited", func(t *testing.T) {
		api, clients := newQuotaAPI(t, func(c quotaCall) (int, any) {
			if c.Method == http.MethodPatch {
				return http.StatusOK, quotaObject("9", map[string]string{"limits.memory": "3Gi"}, nil)
			}
			return http.StatusOK, quotaObject("10", map[string]string{"limits.memory": "3Gi"}, map[string]string{"limits.memory": "3Gi"})
		})
		resp := update(clients,
			quotaFixture(t, s, map[string]string{"limits.memory": "2048Mi"}),
			quotaFixture(t, s, map[string]string{"limits.memory": "3072Mi"}))
		if resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}
		var patch string
		for _, c := range api.recorded() {
			if c.Method == http.MethodPatch {
				patch = c.Body
			}
		}
		if !strings.Contains(patch, `"replace"`) || !strings.Contains(patch, `"/spec"`) {
			t.Errorf("patch = %s, want a replace of /spec", patch)
		}
		if api.count(http.MethodGet) < 1 {
			t.Error("update must wait for status.hard")
		}
		if h := quotaHardOf(t, resp.State); h["limits.memory"] != "3072Mi" {
			t.Errorf("hard = %v, want the planned spelling", h)
		}
		var got corev1.ResourceQuotaV1Model
		resp.State.Get(ctx, &got)
		if got.Metadata[0].ResourceVersion.ValueString() != "10" {
			t.Errorf("resource_version = %s, want the settled value 10", got.Metadata[0].ResourceVersion)
		}
	})

	t.Run("an equivalent spelling does not touch spec", func(t *testing.T) {
		api, clients := newQuotaAPI(t, func(quotaCall) (int, any) {
			return http.StatusOK, quotaObject("9", map[string]string{"limits.memory": "2Gi"}, map[string]string{"limits.memory": "2Gi"})
		})
		state := quotaFixture(t, s, map[string]string{"limits.memory": "2Gi"})
		plan := quotaFixture(t, s, map[string]string{"limits.memory": "2048Mi"})
		plan.Metadata[0].Labels = types.MapValueMust(types.StringType, map[string]attr.Value{"env": types.StringValue("x")})
		resp := update(clients, state, plan)
		if resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}
		for _, c := range api.recorded() {
			if c.Method == http.MethodPatch && strings.Contains(c.Body, "/spec") {
				t.Errorf("patch must not replace /spec for an equivalent quantity: %s", c.Body)
			}
		}
		if api.count(http.MethodPatch) != 1 {
			t.Errorf("want the label patch, calls = %+v", api.recorded())
		}
	})

	t.Run("a timeouts-only change makes no write", func(t *testing.T) {
		api, clients := newQuotaAPI(t, func(quotaCall) (int, any) {
			return http.StatusOK, quotaObject("1", map[string]string{"limits.memory": "2Gi"}, map[string]string{"limits.memory": "2Gi"})
		})
		state := quotaFixture(t, s, map[string]string{"limits.memory": "2Gi"})
		plan := quotaFixture(t, s, map[string]string{"limits.memory": "2Gi"})
		plan.Timeouts = quotaTimeouts(t, s, "2m")
		if resp := update(clients, state, plan); resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}
		if api.count(http.MethodPatch) != 0 || api.count(http.MethodGet) != 1 {
			t.Errorf("calls = %+v, want a single GET", api.recorded())
		}
	})
}

func TestResourceQuotaDelete(t *testing.T) {
	ctx := context.Background()
	s := resourceQuotaSchema(t)
	del := func(clients namespaceUpdateClientsets) fwresource.DeleteResponse {
		st := quotaStateOf(t, s, quotaFixture(t, s, map[string]string{}))
		resp := fwresource.DeleteResponse{State: st}
		quotaResource(clients).Delete(ctx, fwresource.DeleteRequest{State: st}, &resp)
		return resp
	}

	api, clients := newQuotaAPI(t, func(quotaCall) (int, any) { return http.StatusOK, nil })
	if resp := del(clients); resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if api.count(http.MethodDelete) != 1 {
		t.Errorf("calls = %+v, want one DELETE", api.recorded())
	}

	_, clients = newQuotaAPI(t, func(quotaCall) (int, any) { return http.StatusNotFound, nil })
	if resp := del(clients); resp.Diagnostics.HasError() {
		t.Errorf("an already-deleted quota is success, got %v", resp.Diagnostics)
	}

	_, clients = newQuotaAPI(t, func(quotaCall) (int, any) { return http.StatusForbidden, nil })
	if resp := del(clients); !resp.Diagnostics.HasError() {
		t.Error("a forbidden delete must be reported")
	}
}

// ---- Acceptance tests --------------------------------------------------------------------
//
// Ported from kubernetes/resource_kubernetes_resource_quota_v1_test.go, which now covers the
// deprecated SDKv2 alias. Differences from that file are noted where they occur.

const resourceQuotaResourceName = "kubernetes_resource_quota_v1.test"

func testAccResourceQuotaName() string {
	return fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
}

func testAccCheckKubernetesResourceQuotaV1Destroy(s *terraform.State) error {
	conn, err := mainClientset()
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		// Both type names: a move test that fails partway leaves state under the old
		// address, and matching only the versioned one would let that leak pass silently.
		if rs.Type != "kubernetes_resource_quota_v1" && rs.Type != "kubernetes_resource_quota" {
			continue
		}
		namespace, name, _ := strings.Cut(rs.Primary.ID, "/")
		_, err := conn.CoreV1().ResourceQuotas(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("checking resource quota %q destruction: %w", rs.Primary.ID, err)
		}
		return fmt.Errorf("resource quota still exists: %s", rs.Primary.ID)
	}
	return nil
}

func testAccCheckKubernetesResourceQuotaV1Exists(n string, obj *k8sv1.ResourceQuota) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}
		conn, err := mainClientset()
		if err != nil {
			return err
		}
		namespace, name, _ := strings.Cut(rs.Primary.ID, "/")
		out, err := conn.CoreV1().ResourceQuotas(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		*obj = *out
		return nil
	}
}

// testAccCheckResourceQuotaNotRecreated fails if the quota was replaced rather than updated in
// place: the UID is assigned by the API server and is stable for an object's lifetime.
func testAccCheckResourceQuotaNotRecreated(before, after *k8sv1.ResourceQuota) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if before.UID != after.UID {
			return fmt.Errorf("resource quota was recreated: uid %s -> %s", before.UID, after.UID)
		}
		return nil
	}
}

func testAccCheckResourceQuotaRecreated(before, after *k8sv1.ResourceQuota) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if before.UID == after.UID {
			return fmt.Errorf("resource quota was not recreated: uid %s", after.UID)
		}
		return nil
	}
}

// testAccCheckResourceQuotaLiveHard reads the live object: state keeps the user's spelling, so
// only the API server can say what was actually sent.
func testAccCheckResourceQuotaLiveHard(n, key, want string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		var live k8sv1.ResourceQuota
		if err := testAccCheckKubernetesResourceQuotaV1Exists(n, &live)(s); err != nil {
			return err
		}
		got, ok := live.Spec.Hard[k8sv1.ResourceName(key)]
		if !ok {
			return fmt.Errorf("live spec.hard has no %q: %v", key, live.Spec.Hard)
		}
		wantQ := k8sresource.MustParse(want)
		if got.Cmp(wantQ) != 0 {
			return fmt.Errorf("live spec.hard[%q] = %s, want %s", key, got.String(), want)
		}
		return nil
	}
}

func testAccResourceQuotaDeleteOutOfBand(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}
		conn, err := mainClientset()
		if err != nil {
			return err
		}
		namespace, name, _ := strings.Cut(rs.Primary.ID, "/")
		return conn.CoreV1().ResourceQuotas(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
	}
}

func TestAccKubernetesResourceQuotaV1_basic(t *testing.T) {
	var conf k8sv1.ResourceQuota
	name := testAccResourceQuotaName()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesResourceQuotaV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesResourceQuotaV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaV1Exists(resourceQuotaResourceName, &conf),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "id", name+"/"+name),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "metadata.0.labels.TestLabelFour", "four"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceQuotaResourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceQuotaResourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceQuotaResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.hard.%", "3"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.hard.limits.cpu", "2"),
					// SDKv2 asserted "2Gi" here: it stored the API's spelling of the "2048Mi" in
					// the config. State now keeps the spelling the user wrote (the quantity type's
					// semantic equality), and the live object is checked for the value.
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.hard.limits.memory", "2048Mi"),
					testAccCheckResourceQuotaLiveHard(resourceQuotaResourceName, "limits.memory", "2Gi"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.hard.pods", "4"),
				),
			},
			{
				ResourceName:      resourceQuotaResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"metadata.0.resource_version",
					// Import has no configuration to take a spelling from, so it reports the
					// API's "2Gi" for the "2048Mi" in state. Equivalent values: see above.
					"spec.0.hard.limits.memory",
				},
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if got := states[0].Attributes["spec.0.hard.limits.memory"]; got != "2Gi" {
						return fmt.Errorf("imported spec.hard[limits.memory] = %q, want the API's 2Gi", got)
					}
					return nil
				},
			},
			{
				Config: testAccKubernetesResourceQuotaV1Config_metaModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaV1Exists(resourceQuotaResourceName, &conf),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceQuotaResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.hard.%", "3"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.hard.limits.cpu", "2"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.hard.limits.memory", "2Gi"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.hard.pods", "4"),
				),
			},
			{
				Config: testAccKubernetesResourceQuotaV1Config_specModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaV1Exists(resourceQuotaResourceName, &conf),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceQuotaResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.hard.%", "4"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.hard.limits.cpu", "4"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.hard.requests.cpu", "1"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.hard.limits.memory", "4Gi"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.hard.pods", "10"),
					testAccCheckResourceQuotaLiveHard(resourceQuotaResourceName, "limits.memory", "4Gi"),
				),
			},
		},
	})
}

func TestAccKubernetesResourceQuotaV1_generatedName(t *testing.T) {
	var conf k8sv1.ResourceQuota
	prefix := "tf-acc-test-"
	ns := testAccResourceQuotaName()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesResourceQuotaV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesResourceQuotaV1Config_generatedName(ns, prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaV1Exists(resourceQuotaResourceName, &conf),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr(resourceQuotaResourceName, "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet(resourceQuotaResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.hard.%", "1"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.hard.pods", "10"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.scopes.#", "0"),
				),
			},
			{
				ResourceName:      resourceQuotaResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// generate_name is not stored by the API server for this kind, so an import
				// cannot recover it (kubernetes/kubernetes#44539). SDKv2 listed no such entry
				// because it reported the empty string both sides; here it is null on import.
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "metadata.0.generate_name"},
			},
		},
	})
}

// K8S-MIGRATE-025: generate_name makes metadata.name server-assigned. An in-place change must
// not turn into a replacement because the planned name is unknown.
func TestAccKubernetesResourceQuotaV1_generatedNameUpdateInPlace(t *testing.T) {
	var before, after k8sv1.ResourceQuota
	ns := testAccResourceQuotaName()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesResourceQuotaV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesResourceQuotaV1Config_generatedName(ns, "tf-acc-test-"),
				Check:  testAccCheckKubernetesResourceQuotaV1Exists(resourceQuotaResourceName, &before),
			},
			{
				Config: testAccKubernetesResourceQuotaV1Config_generatedNameLabelled(ns, "tf-acc-test-"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceQuotaResourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaV1Exists(resourceQuotaResourceName, &after),
					testAccCheckResourceQuotaNotRecreated(&before, &after),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "metadata.0.labels.added", "yes"),
				),
			},
		},
	})
}

func TestAccKubernetesResourceQuotaV1_withScopes(t *testing.T) {
	var before, after k8sv1.ResourceQuota
	name := testAccResourceQuotaName()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesResourceQuotaV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesResourceQuotaV1Config_withScopes(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaV1Exists(resourceQuotaResourceName, &before),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.hard.%", "1"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.hard.pods", "10"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.scopes.#", "1"),
					resource.TestCheckTypeSetElemAttr(resourceQuotaResourceName, "spec.0.scopes.*", "BestEffort"),
				),
			},
			{
				ResourceName:            resourceQuotaResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				// scopes is ForceNew in SDKv2, so changing it replaces the quota.
				Config: testAccKubernetesResourceQuotaV1Config_withScopesModified(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceQuotaResourceName, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaV1Exists(resourceQuotaResourceName, &after),
					testAccCheckResourceQuotaRecreated(&before, &after),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.scopes.#", "1"),
					resource.TestCheckTypeSetElemAttr(resourceQuotaResourceName, "spec.0.scopes.*", "NotBestEffort"),
				),
			},
		},
	})
}

func TestAccKubernetesResourceQuotaV1_scopeSelector(t *testing.T) {
	var conf k8sv1.ResourceQuota
	name := testAccResourceQuotaName()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesResourceQuotaV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesResourceQuotaV1ConfigScopeSelector(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaV1Exists(resourceQuotaResourceName, &conf),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.scope_selector.#", "1"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.scope_selector.0.match_expression.#", "1"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.scope_selector.0.match_expression.0.scope_name", "PriorityClass"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.scope_selector.0.match_expression.0.operator", "In"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.scope_selector.0.match_expression.0.values.#", "1"),
					resource.TestCheckTypeSetElemAttr(resourceQuotaResourceName, "spec.0.scope_selector.0.match_expression.0.values.*", "medium"),
				),
			},
			{
				ResourceName:            resourceQuotaResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesResourceQuotaV1ConfigScopeSelectorModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaV1Exists(resourceQuotaResourceName, &conf),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.scope_selector.#", "1"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.scope_selector.0.match_expression.#", "1"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.scope_selector.0.match_expression.0.operator", "NotIn"),
					resource.TestCheckTypeSetElemAttr(resourceQuotaResourceName, "spec.0.scope_selector.0.match_expression.0.values.*", "large"),
				),
			},
			{
				Config: testAccKubernetesResourceQuotaV1ConfigMultipleMatchExpression(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaV1Exists(resourceQuotaResourceName, &conf),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.scope_selector.#", "1"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.scope_selector.0.match_expression.#", "2"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.scope_selector.0.match_expression.0.operator", "NotIn"),
					resource.TestCheckTypeSetElemAttr(resourceQuotaResourceName, "spec.0.scope_selector.0.match_expression.0.values.*", "large"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.scope_selector.0.match_expression.1.operator", "In"),
					resource.TestCheckTypeSetElemAttr(resourceQuotaResourceName, "spec.0.scope_selector.0.match_expression.1.values.*", "low"),
				),
			},
		},
	})
}

// Decision 2: hard is a map of quantities with semantic equality. State keeps the spelling the
// user wrote, and the API's canonical form ("1" for "1000m", "500m" for "0.5") must neither
// leak into state nor show up as a diff on refresh or plan. This is the end-to-end proof that
// the framework applies the equality to map elements inside a nested block.
func TestAccKubernetesResourceQuotaV1_quantitySpelling(t *testing.T) {
	name := testAccResourceQuotaName()
	var before, after k8sv1.ResourceQuota
	empty := resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesResourceQuotaV1Destroy,
		Steps: []resource.TestStep{
			{
				// Create: the API answers "1", "2Gi" and "500m"; state must keep what was written.
				Config: testAccKubernetesResourceQuotaV1Config_spelling(name, "1000m", "2048Mi", "0.5"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaV1Exists(resourceQuotaResourceName, &before),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.hard.limits.cpu", "1000m"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.hard.limits.memory", "2048Mi"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.hard.requests.cpu", "0.5"),
					testAccCheckResourceQuotaLiveHard(resourceQuotaResourceName, "limits.cpu", "1"),
					testAccCheckResourceQuotaLiveHard(resourceQuotaResourceName, "limits.memory", "2Gi"),
					testAccCheckResourceQuotaLiveHard(resourceQuotaResourceName, "requests.cpu", "500m"),
				),
			},
			{
				// Refresh: Read returns the canonical form; semantic equality must keep the prior one.
				RefreshState: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.hard.limits.cpu", "1000m"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.hard.limits.memory", "2048Mi"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.hard.requests.cpu", "0.5"),
				),
			},
			{
				// Plan: the same config is a no-op.
				Config:           testAccKubernetesResourceQuotaV1Config_spelling(name, "1000m", "2048Mi", "0.5"),
				ConfigPlanChecks: empty,
			},
			{
				// Update that changes only the spelling: the value is the same, so the quota is
				// updated in place and the spec is untouched.
				Config: testAccKubernetesResourceQuotaV1Config_spelling(name, "1", "2Gi", "500m"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.hard.limits.cpu", "1"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.hard.limits.memory", "2Gi"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.hard.requests.cpu", "500m"),
				),
			},
			{
				// Update that changes a value, written in a non-canonical spelling.
				Config: testAccKubernetesResourceQuotaV1Config_spelling(name, "1", "3072Mi", "500m"),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(resourceQuotaResourceName, plancheck.ResourceActionUpdate),
				}},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaV1Exists(resourceQuotaResourceName, &after),
					testAccCheckResourceQuotaNotRecreated(&before, &after),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.hard.limits.memory", "3072Mi"),
					testAccCheckResourceQuotaLiveHard(resourceQuotaResourceName, "limits.memory", "3Gi"),
				),
			},
			{
				// Refresh after a real update.
				RefreshState: true,
				Check:        resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.0.hard.limits.memory", "3072Mi"),
			},
			{
				Config:           testAccKubernetesResourceQuotaV1Config_spelling(name, "1", "3072Mi", "500m"),
				ConfigPlanChecks: empty,
			},
		},
	})
}

// SDKv2 stored an omitted spec as a spec with empty collections. The framework keeps it
// omitted, and an explicit empty block stays an empty block.
func TestAccKubernetesResourceQuotaV1_noSpec(t *testing.T) {
	for name, spec := range map[string]string{
		"omitted":    "",
		"emptyBlock": "  spec {}\n",
		"emptyHard":  "  spec {\n    hard = {}\n  }\n",
	} {
		t.Run(name, func(t *testing.T) {
			ns := testAccResourceQuotaName()
			cfg := testAccKubernetesResourceQuotaV1Config_withSpecBlock(ns, spec)
			wantSpec := "1"
			if spec == "" {
				wantSpec = "0"
			}
			resource.ParallelTest(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				CheckDestroy:             testAccCheckKubernetesResourceQuotaV1Destroy,
				Steps: []resource.TestStep{
					{
						Config: cfg,
						Check:  resource.TestCheckResourceAttr(resourceQuotaResourceName, "spec.#", wantSpec),
					},
					{
						Config:           cfg,
						ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
					},
				},
			})
		})
	}
}

func TestAccKubernetesResourceQuotaV1_timeouts(t *testing.T) {
	name := testAccResourceQuotaName()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesResourceQuotaV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesResourceQuotaV1Config_timeouts(name, "2m"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "timeouts.create", "2m"),
					resource.TestCheckResourceAttr(resourceQuotaResourceName, "timeouts.update", "2m"),
				),
			},
			{
				// A timeouts-only change is applied without touching the quota.
				Config: testAccKubernetesResourceQuotaV1Config_timeouts(name, "3m"),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(resourceQuotaResourceName, plancheck.ResourceActionUpdate),
				}},
				Check: resource.TestCheckResourceAttr(resourceQuotaResourceName, "timeouts.update", "3m"),
			},
		},
	})
}

func TestAccKubernetesResourceQuotaV1_identity(t *testing.T) {
	name := testAccResourceQuotaName()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)},
		CheckDestroy:             testAccCheckKubernetesResourceQuotaV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesResourceQuotaV1Config_withScopes(name),
				// api_version and kind are literals because client-go's typed clients clear
				// TypeMeta on responses; the resource hardcodes them for the same reason.
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(resourceQuotaResourceName, map[string]knownvalue.Check{
						"namespace":   knownvalue.StringExact(name),
						"name":        knownvalue.StringExact(name),
						"api_version": knownvalue.StringExact("v1"),
						"kind":        knownvalue.StringExact("ResourceQuota"),
					}),
				},
			},
			{
				// An import block asserts a no-op plan and an unchanged identity.
				ResourceName:    resourceQuotaResourceName,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func TestAccKubernetesResourceQuotaV1_importInvalidID(t *testing.T) {
	for _, tc := range []struct{ name, id string }{
		{"missingSlash", "quota"},
		{"extraSlash", "team/quota/extra"},
		{"emptyNamespace", "/quota"},
		{"emptyName", "team/"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.ParallelTest(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				CheckDestroy:             testAccCheckKubernetesResourceQuotaV1Destroy,
				Steps: []resource.TestStep{
					{Config: testAccKubernetesResourceQuotaV1Config_withScopes(testAccResourceQuotaName())},
					{
						ResourceName:  resourceQuotaResourceName,
						ImportState:   true,
						ImportStateId: tc.id,
						ExpectError:   regexp.MustCompile("(?i)invalid resource ID"),
					},
				},
			})
		})
	}
}

func TestAccKubernetesResourceQuotaV1_importInvalidIdentity(t *testing.T) {
	for _, tc := range []struct{ name, old, replacement string }{
		{"wrongKind", `kind        = "ResourceQuota"`, `kind        = "LimitRange"`},
		{"wrongAPIVersion", `api_version = "v1"`, `api_version = "v2"`},
		{"missingNamespace", `    namespace   = %q` + "\n", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name := testAccResourceQuotaName()
			config := testAccKubernetesResourceQuotaV1Config_withScopes(name) + fmt.Sprintf(`
import {
  to = kubernetes_resource_quota_v1.test
  identity = {
    namespace   = %[1]q
    name        = %[1]q
    kind        = "ResourceQuota"
    api_version = "v1"
  }
}
`, name)
			old := strings.ReplaceAll(tc.old, "%q", fmt.Sprintf("%q", name))
			if !strings.Contains(config, old) {
				t.Fatalf("test bug: %q not in config", old)
			}
			config = strings.Replace(config, old, tc.replacement, 1)

			resource.ParallelTest(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				CheckDestroy:             testAccCheckKubernetesResourceQuotaV1Destroy,
				TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)},
				Steps: []resource.TestStep{
					{Config: testAccKubernetesResourceQuotaV1Config_withScopes(name)},
					{
						ResourceName:           resourceQuotaResourceName,
						ImportState:            true,
						ImportStateKind:        resource.ImportBlockWithResourceIdentity,
						ImportStateConfigExact: true,
						Config:                 config,
						ExpectError:            regexp.MustCompile("(?i)invalid identity import"),
					},
				},
			})
		})
	}
}

// Deleted outside Terraform: the next plan recreates it instead of failing on a 404.
func TestAccKubernetesResourceQuotaV1_disappears(t *testing.T) {
	var conf k8sv1.ResourceQuota
	name := testAccResourceQuotaName()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesResourceQuotaV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesResourceQuotaV1Config_withScopes(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaV1Exists(resourceQuotaResourceName, &conf),
					testAccResourceQuotaDeleteOutOfBand(resourceQuotaResourceName),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccKubernetesResourceQuotaV1Config_withScopes(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(resourceQuotaResourceName, plancheck.ResourceActionCreate),
				}},
				Check: testAccCheckKubernetesResourceQuotaV1Exists(resourceQuotaResourceName, &conf),
			},
			{
				Config:           testAccKubernetesResourceQuotaV1Config_withScopes(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
			},
		},
	})
}

// Negative validators: each ports a check SDKv2 made at validate time.
func TestAccKubernetesResourceQuotaV1_invalidConfig(t *testing.T) {
	for _, tc := range []struct {
		name, spec, want string
	}{
		{"invalidQuantity", "spec {\n    hard = {\n      pods = \"abc\"\n    }\n  }", `(?s)Invalid Attribute Value.*hard`},
		{"nullQuantity", "spec {\n    hard = {\n      pods = null\n    }\n  }", `(?s)Invalid Attribute Value.*quantity`},
		{"invalidScope", "spec {\n    scopes = [\"Bogus\"]\n  }", `(?s)Invalid Attribute Value.*got:\s+"Bogus"`},
		{"invalidScopeName", "spec {\n    scope_selector {\n      match_expression {\n        scope_name = \"Bogus\"\n        operator   = \"Exists\"\n      }\n    }\n  }", `(?s)Invalid Attribute Value.*got:\s+"Bogus"`},
		{"invalidOperator", "spec {\n    scope_selector {\n      match_expression {\n        scope_name = \"PriorityClass\"\n        operator   = \"Bogus\"\n      }\n    }\n  }", `(?s)Invalid Attribute Value.*got:\s+"Bogus"`},
		{"twoSpecs", "spec {}\n  spec {}", `(?s)spec.*at most 1`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.ParallelTest(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{{
					Config: fmt.Sprintf(`resource "kubernetes_resource_quota_v1" "test" {
  metadata {
    name      = "tf-acc-test-invalid"
    namespace = "default"
  }

  %s
}
`, tc.spec),
					PlanOnly:    true,
					ExpectError: regexp.MustCompile(tc.want),
				}},
			})
		})
	}
}

func testAccKubernetesResourceQuotaV1Config_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_resource_quota_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
      TestLabelFour  = "four"
    }

    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata[0].name
  }

  spec {
    hard = {
      "limits.cpu"    = 2
      "limits.memory" = "2048Mi"
      pods            = 4
    }
  }
}
`, name)
}

func testAccKubernetesResourceQuotaV1Config_metaModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_resource_quota_v1" "test" {
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

    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata[0].name
  }

  spec {
    hard = {
      "limits.cpu"    = 2
      "limits.memory" = "2Gi"
      pods            = 4
    }
  }
}
`, name)
}

func testAccKubernetesResourceQuotaV1Config_specModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_resource_quota_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata[0].name
  }

  spec {
    hard = {
      "limits.cpu"    = 4
      "requests.cpu"  = 1
      "limits.memory" = "4Gi"
      pods            = 10
    }
  }
}
`, name)
}

func testAccKubernetesResourceQuotaV1Config_generatedName(ns, prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_resource_quota_v1" "test" {
  metadata {
    generate_name = %[2]q
    namespace     = kubernetes_namespace_v1.test.metadata[0].name
  }

  spec {
    hard = {
      pods = 10
    }
  }
}
`, ns, prefix)
}

func testAccKubernetesResourceQuotaV1Config_generatedNameLabelled(ns, prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_resource_quota_v1" "test" {
  metadata {
    generate_name = %[2]q
    namespace     = kubernetes_namespace_v1.test.metadata[0].name

    labels = {
      added = "yes"
    }
  }

  spec {
    hard = {
      pods = 10
    }
  }
}
`, ns, prefix)
}

func testAccKubernetesResourceQuotaV1Config_withScopes(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_resource_quota_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata[0].name
  }

  spec {
    hard = {
      pods = 10
    }

    scopes = ["BestEffort"]
  }
}
`, name)
}

func testAccKubernetesResourceQuotaV1Config_withScopesModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_resource_quota_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata[0].name
  }

  spec {
    hard = {
      pods = 10
    }

    scopes = ["NotBestEffort"]
  }
}
`, name)
}

func testAccKubernetesResourceQuotaV1ConfigScopeSelector(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_resource_quota_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
      TestLabelFour  = "four"
    }

    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata[0].name
  }

  spec {
    hard = {
      "limits.cpu"    = 2
      "limits.memory" = "2Gi"
      pods            = 4
    }

    scope_selector {
      match_expression {
        scope_name = "PriorityClass"
        operator   = "In"
        values     = ["medium"]
      }
    }
  }
}
`, name)
}

func testAccKubernetesResourceQuotaV1ConfigScopeSelectorModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_resource_quota_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
      TestLabelFour  = "four"
    }

    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata[0].name
  }

  spec {
    hard = {
      "limits.cpu"    = 2
      "limits.memory" = "2Gi"
      pods            = 4
    }

    scope_selector {
      match_expression {
        scope_name = "PriorityClass"
        operator   = "NotIn"
        values     = ["large"]
      }
    }
  }
}
`, name)
}

func testAccKubernetesResourceQuotaV1ConfigMultipleMatchExpression(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_resource_quota_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
      TestLabelFour  = "four"
    }

    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata[0].name
  }

  spec {
    hard = {
      "limits.cpu"    = 2
      "limits.memory" = "2Gi"
      pods            = 4
    }

    scope_selector {
      match_expression {
        scope_name = "PriorityClass"
        operator   = "NotIn"
        values     = ["large"]
      }
      match_expression {
        scope_name = "PriorityClass"
        operator   = "In"
        values     = ["low"]
      }
    }
  }
}
`, name)
}

func testAccKubernetesResourceQuotaV1Config_spelling(name, cpu, memory, requests string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_resource_quota_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata[0].name
  }

  spec {
    hard = {
      "limits.cpu"    = %[2]q
      "limits.memory" = %[3]q
      "requests.cpu"  = %[4]q
    }
  }
}
`, name, cpu, memory, requests)
}

// withSpecBlock takes the spec lines verbatim, so the omitted-spec and empty-block shapes
// share one config.
func testAccKubernetesResourceQuotaV1Config_withSpecBlock(name, spec string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_resource_quota_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata[0].name
  }

%[2]s
}
`, name, spec)
}

func testAccKubernetesResourceQuotaV1Config_timeouts(name, timeout string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_resource_quota_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata[0].name
  }

  spec {
    hard = {
      pods = 10
    }
  }

  timeouts {
    create = "2m"
    update = %[2]q
  }
}
`, name, timeout)
}
