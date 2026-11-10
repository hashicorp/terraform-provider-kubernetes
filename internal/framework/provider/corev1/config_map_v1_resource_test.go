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
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	k8sv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// configMapFakeMeta satisfies both interfaces the resource asserts on SDKv2 meta, backed by
// a clientset pointed at a test server.
type configMapFakeMeta struct {
	kubernetes.KubeClientsets
	client *client.Clientset
}

func (m configMapFakeMeta) MainClientset() (*client.Clientset, error) { return m.client, nil }
func (configMapFakeMeta) GetIgnoreAnnotations() []string              { return nil }
func (configMapFakeMeta) GetIgnoreLabels() []string                   { return nil }

type recordedRequest struct {
	Method, Path, ContentType, Body string
}

// configMapServer answers every request with the given status and body and records them.
func configMapServer(t *testing.T, status int, body any) (*corev1.ConfigMapV1, func() []recordedRequest) {
	t.Helper()
	var mu sync.Mutex
	var seen []recordedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		b, _ := io.ReadAll(req.Body)
		mu.Lock()
		seen = append(seen, recordedRequest{req.Method, req.URL.Path, req.Header.Get("Content-Type"), string(b)})
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(server.Close)
	conn, err := client.NewForConfig(&rest.Config{Host: server.URL, ContentConfig: rest.ContentConfig{ContentType: "application/json"}})
	if err != nil {
		t.Fatal(err)
	}
	r := &corev1.ConfigMapV1{SDKv2Meta: func() any { return configMapFakeMeta{client: conn} }}
	return r, func() []recordedRequest {
		mu.Lock()
		defer mu.Unlock()
		return append([]recordedRequest(nil), seen...)
	}
}

func statusBody(code int) metav1.Status {
	return metav1.Status{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Status"},
		Status:   "Failure", Code: int32(code), Message: http.StatusText(code),
	}
}

func configMapSchema(t *testing.T) frameworkresource.SchemaResponse {
	t.Helper()
	var resp frameworkresource.SchemaResponse
	(&corev1.ConfigMapV1{}).Schema(context.Background(), frameworkresource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	return resp
}

// configMapState builds a tfsdk.State from SDKv2-shaped JSON so the test exercises real
// decoding, not hand-populated models.
func configMapState(t *testing.T, raw string) tfsdk.State {
	t.Helper()
	s := configMapSchema(t)
	v, err := (&tfprotov6.RawState{JSON: []byte(raw)}).Unmarshal(s.Schema.Type().TerraformType(context.Background()))
	if err != nil {
		t.Fatal(err)
	}
	return tfsdk.State{Schema: s.Schema, Raw: v}
}

func nullIdentity(t *testing.T) tfsdk.ResourceIdentity {
	t.Helper()
	s := common.NamespacedIdentitySchema()
	return tfsdk.ResourceIdentity{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(context.Background()), nil)}
}

const storedConfigMap = `{
	"id":"team/cfg",
	"metadata":[{"name":"cfg","namespace":"team","uid":"uid-1","resource_version":"10","generation":0,
		"labels":null,"annotations":null,"generate_name":null}],
	"data":{"one":"first"},
	"binary_data":null,
	"immutable":false
}`

func TestConfigMapSchema(t *testing.T) {
	s := configMapSchema(t).Schema

	if diags := s.ValidateImplementation(context.Background()); diags.HasError() {
		t.Fatal(diags)
	}

	// data, binary_data: Optional only. Computed would turn "unset" into a known value.
	for _, name := range []string{"data", "binary_data"} {
		a := s.Attributes[name]
		if !a.IsOptional() || a.IsRequired() || a.IsComputed() {
			t.Errorf("%s must be Optional and not Computed", name)
		}
	}
	// immutable: Optional+Computed+Default(false) — SDKv2 stored an unset bool as false.
	im := s.Attributes["immutable"]
	if !im.IsOptional() || !im.IsComputed() {
		t.Error("immutable must be Optional+Computed")
	}
	if !s.Attributes["id"].IsComputed() {
		t.Error("id must be Computed")
	}

	meta, ok := s.Blocks["metadata"]
	if !ok {
		t.Fatal("metadata block missing")
	}
	for _, name := range []string{"name", "generate_name", "namespace", "labels", "annotations", "uid", "resource_version", "generation"} {
		if _, ok := meta.GetNestedObject().GetAttributes()[name]; !ok {
			t.Errorf("metadata.%s missing", name)
		}
	}
	if !meta.GetNestedObject().GetAttributes()["namespace"].IsComputed() {
		t.Error("metadata.namespace must be Optional+Computed with default \"default\"")
	}
}

func TestConfigMapIdentitySchema(t *testing.T) {
	var resp frameworkresource.IdentitySchemaResponse
	(&corev1.ConfigMapV1{}).IdentitySchema(context.Background(), frameworkresource.IdentitySchemaRequest{}, &resp)
	if resp.IdentitySchema.Version != 1 {
		t.Fatalf("identity version = %d; must stay 1 to match SDKv2", resp.IdentitySchema.Version)
	}
	ns, ok := resp.IdentitySchema.Attributes["namespace"].(identityschema.StringAttribute)
	if !ok || !ns.OptionalForImport || ns.RequiredForImport {
		t.Error("namespace must be OptionalForImport")
	}
	for _, n := range []string{"name", "kind", "api_version"} {
		a, ok := resp.IdentitySchema.Attributes[n].(identityschema.StringAttribute)
		if !ok || !a.RequiredForImport {
			t.Errorf("%s must be RequiredForImport", n)
		}
	}
}

func TestConfigMapType(t *testing.T) {
	var resp frameworkresource.MetadataResponse
	(&corev1.ConfigMapV1{}).Metadata(context.Background(), frameworkresource.MetadataRequest{ProviderTypeName: "kubernetes"}, &resp)
	if resp.TypeName != "kubernetes_config_map_v1" {
		t.Fatalf("type name = %q", resp.TypeName)
	}
}

func TestConfigMapConfigure(t *testing.T) {
	r := &corev1.ConfigMapV1{}
	var resp frameworkresource.ConfigureResponse
	r.Configure(context.Background(), frameworkresource.ConfigureRequest{ProviderData: "wrong"}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("wrong ProviderData type must be a diagnostic, not a panic")
	}
	resp = frameworkresource.ConfigureResponse{}
	r.Configure(context.Background(), frameworkresource.ConfigureRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal("nil ProviderData must be tolerated")
	}
}

func TestConfigMapBase64Validator(t *testing.T) {
	s := configMapSchema(t).Schema
	binaryAttr := s.Attributes["binary_data"]
	mapAttr, ok := binaryAttr.(interface{ MapValidators() []validator.Map })
	if !ok {
		t.Fatal("binary_data has no map validators")
	}
	cases := []struct {
		name    string
		val     types.Map
		wantErr bool
	}{
		{"valid", types.MapValueMust(types.StringType, map[string]attr.Value{"a": types.StringValue("aGVsbG8=")}), false},
		{"empty", types.MapValueMust(types.StringType, map[string]attr.Value{}), false},
		{"invalid", types.MapValueMust(types.StringType, map[string]attr.Value{"a": types.StringValue("not base64!")}), true},
		{"null map", types.MapNull(types.StringType), false},
		{"unknown map", types.MapUnknown(types.StringType), false},
		{"unknown element", types.MapValueMust(types.StringType, map[string]attr.Value{"a": types.StringUnknown()}), false},
		{"null element", types.MapValueMust(types.StringType, map[string]attr.Value{"a": types.StringNull()}), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var diagErr bool
			for _, v := range mapAttr.MapValidators() {
				resp := &validator.MapResponse{}
				v.ValidateMap(context.Background(), validator.MapRequest{ConfigValue: tc.val}, resp)
				diagErr = diagErr || resp.Diagnostics.HasError()
			}
			if diagErr != tc.wantErr {
				t.Fatalf("error = %v, want %v", diagErr, tc.wantErr)
			}
		})
	}
}

func TestConfigMapModifyPlanImmutable(t *testing.T) {
	ctx := context.Background()
	s := configMapSchema(t).Schema
	r := &corev1.ConfigMapV1{}

	build := func(immutable bool, data, binary string) tfsdk.State {
		return configMapState(t, `{
			"id":"team/cfg",
			"metadata":[{"name":"cfg","namespace":"team","uid":"u","resource_version":"1","generation":0,"labels":null,"annotations":null,"generate_name":null}],
			"data":`+data+`,"binary_data":`+binary+`,"immutable":`+map[bool]string{true: "true", false: "false"}[immutable]+`}`)
	}

	for _, tc := range []struct {
		name         string
		state, plan  tfsdk.State
		wantReplaced []string
	}{
		{"mutable data change updates in place", build(false, `{"a":"1"}`, `null`), build(false, `{"a":"2"}`, `null`), nil},
		{"immutable, unchanged", build(true, `{"a":"1"}`, `null`), build(true, `{"a":"1"}`, `null`), nil},
		{"immutable, data changed replaces", build(true, `{"a":"1"}`, `null`), build(true, `{"a":"2"}`, `null`), []string{"data"}},
		{"immutable, binary_data changed replaces", build(true, `null`, `{"a":"aGk="}`), build(true, `null`, `{"a":"Ynll"}`), []string{"binary_data"}},
		{"immutable, true to false replaces", build(true, `{"a":"1"}`, `null`), build(false, `{"a":"1"}`, `null`), []string{"immutable"}},
		{"turning immutable on is an in-place update", build(false, `{"a":"1"}`, `null`), build(true, `{"a":"1"}`, `null`), nil},
		// SDKv2 cannot tell null from {}; replacing for that would destroy untouched data.
		{"immutable, null vs {} is not a change", build(true, `null`, `null`), build(true, `{}`, `{}`), nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := &frameworkresource.ModifyPlanResponse{Plan: tfsdk.Plan{Schema: s, Raw: tc.plan.Raw}}
			r.ModifyPlan(ctx, frameworkresource.ModifyPlanRequest{
				State: tc.state, Plan: tfsdk.Plan{Schema: s, Raw: tc.plan.Raw}, Config: tfsdk.Config{Schema: s, Raw: tc.plan.Raw},
			}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			var got []string
			for _, p := range resp.RequiresReplace {
				got = append(got, p.String())
			}
			if strings.Join(got, ",") != strings.Join(tc.wantReplaced, ",") {
				t.Fatalf("RequiresReplace = %v, want %v", got, tc.wantReplaced)
			}
		})
	}

	t.Run("create and destroy are ignored", func(t *testing.T) {
		null := tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
		resp := &frameworkresource.ModifyPlanResponse{}
		r.ModifyPlan(ctx, frameworkresource.ModifyPlanRequest{State: null, Plan: tfsdk.Plan{Schema: s, Raw: build(false, `null`, `null`).Raw}}, resp)
		r.ModifyPlan(ctx, frameworkresource.ModifyPlanRequest{State: build(true, `null`, `null`), Plan: tfsdk.Plan{Schema: s, Raw: null.Raw}}, resp)
		if resp.Diagnostics.HasError() || len(resp.RequiresReplace) != 0 {
			t.Fatalf("unexpected result: %v %v", resp.Diagnostics, resp.RequiresReplace)
		}
	})
}

func TestConfigMapReadMissingAndErrors(t *testing.T) {
	ctx := context.Background()
	for _, status := range []int{http.StatusNotFound, http.StatusForbidden, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			r, reqs := configMapServer(t, status, statusBody(status))
			state := configMapState(t, storedConfigMap)
			identity := nullIdentity(t)
			respIdentity := identity
			resp := frameworkresource.ReadResponse{State: state, Identity: &respIdentity}
			r.Read(ctx, frameworkresource.ReadRequest{State: state, Identity: &identity}, &resp)

			got := reqs()
			if len(got) != 1 || got[0].Method != http.MethodGet || got[0].Path != "/api/v1/namespaces/team/configmaps/cfg" {
				t.Fatalf("unexpected requests: %+v", got)
			}
			if status == http.StatusNotFound {
				if resp.Diagnostics.HasError() || !resp.State.Raw.IsNull() {
					t.Fatalf("a missing ConfigMap must be removed from state: %v", resp.Diagnostics)
				}
				var id common.NamespacedResourceIdentity
				resp.Diagnostics.Append(resp.Identity.Get(ctx, &id)...)
				if resp.Diagnostics.HasError() || id.Name.ValueString() != "cfg" || id.Namespace.ValueString() != "team" {
					t.Fatalf("identity must be populated when removing: %#v %v", id, resp.Diagnostics)
				}
				return
			}
			if !resp.Diagnostics.HasError() || !resp.State.Raw.Equal(state.Raw) {
				t.Fatalf("API failure must be an error that preserves state: %v", resp.Diagnostics)
			}
		})
	}
}

func TestConfigMapReadInvalidID(t *testing.T) {
	for _, id := range []string{"cfg", "/cfg", "team/", "a/b/c"} {
		t.Run(id, func(t *testing.T) {
			r, reqs := configMapServer(t, http.StatusOK, nil)
			state := configMapState(t, strings.Replace(storedConfigMap, `"id":"team/cfg"`, `"id":"`+id+`"`, 1))
			identity := nullIdentity(t)
			resp := frameworkresource.ReadResponse{State: state, Identity: &identity}
			r.Read(context.Background(), frameworkresource.ReadRequest{State: state, Identity: &identity}, &resp)
			if !resp.Diagnostics.HasError() || len(reqs()) != 0 {
				t.Fatalf("invalid ID %q must fail before any API call: %v", id, resp.Diagnostics)
			}
		})
	}
}

// Read flattens what the API returns: base64 re-encoding of binaryData, immutable nil as
// false, and empty maps staying null when prior state had null.
func TestConfigMapReadFlatten(t *testing.T) {
	ctx := context.Background()
	body := map[string]any{
		"apiVersion": "v1", "kind": "ConfigMap",
		"metadata": map[string]any{"name": "cfg", "namespace": "team", "uid": "uid-2", "resourceVersion": "11"},
		"data":     map[string]any{"one": "first"},
		// "hello" -> aGVsbG8=
		"binaryData": map[string]any{"bin": "aGVsbG8="},
	}
	r, _ := configMapServer(t, http.StatusOK, body)
	state := configMapState(t, storedConfigMap)
	identity := nullIdentity(t)
	respIdentity := identity
	resp := frameworkresource.ReadResponse{State: state, Identity: &respIdentity}
	r.Read(ctx, frameworkresource.ReadRequest{State: state, Identity: &identity}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	var got corev1.ConfigMapV1Model
	resp.Diagnostics.Append(resp.State.Get(ctx, &got)...)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if got.ID.ValueString() != "team/cfg" || got.Immutable.ValueBool() {
		t.Errorf("id/immutable wrong: %q %v", got.ID.ValueString(), got.Immutable)
	}
	if v := got.BinaryData.Elements()["bin"]; v == nil || v.(types.String).ValueString() != "aGVsbG8=" {
		t.Errorf("binary_data must be re-encoded to base64, got %v", got.BinaryData)
	}
	if got.Metadata[0].Labels.IsNull() != true || got.Metadata[0].Annotations.IsNull() != true {
		t.Errorf("empty labels/annotations must stay null when prior state was null")
	}
	if !got.Metadata[0].GenerateName.IsNull() {
		t.Errorf("generate_name must stay null")
	}
}

func TestConfigMapDeleteNotFoundIsSuccess(t *testing.T) {
	r, reqs := configMapServer(t, http.StatusNotFound, statusBody(http.StatusNotFound))
	resp := frameworkresource.DeleteResponse{}
	r.Delete(context.Background(), frameworkresource.DeleteRequest{State: configMapState(t, storedConfigMap)}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("deleting an already-absent ConfigMap must succeed: %v", resp.Diagnostics)
	}
	got := reqs()
	if len(got) != 1 || got[0].Method != http.MethodDelete || got[0].Path != "/api/v1/namespaces/team/configmaps/cfg" {
		t.Fatalf("unexpected requests: %+v", got)
	}
}

func TestConfigMapDeleteErrorSurfaces(t *testing.T) {
	r, _ := configMapServer(t, http.StatusForbidden, statusBody(http.StatusForbidden))
	resp := frameworkresource.DeleteResponse{}
	r.Delete(context.Background(), frameworkresource.DeleteRequest{State: configMapState(t, storedConfigMap)}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("a non-NotFound delete failure must be reported")
	}
}

// Update must send a JSON patch limited to what changed, and "add" for immutable.
func TestConfigMapUpdatePatch(t *testing.T) {
	ctx := context.Background()
	s := configMapSchema(t).Schema
	responseBody := map[string]any{
		"apiVersion": "v1", "kind": "ConfigMap",
		"metadata": map[string]any{"name": "cfg", "namespace": "team", "uid": "uid-1", "resourceVersion": "12"},
	}
	r, reqs := configMapServer(t, http.StatusOK, responseBody)

	state := configMapState(t, storedConfigMap)
	plan := configMapState(t, `{
		"id":"team/cfg",
		"metadata":[{"name":"cfg","namespace":"team","uid":"uid-1","resource_version":"10","generation":0,"labels":null,"annotations":null,"generate_name":null}],
		"data":{"one":"first","two":"second"},"binary_data":null,"immutable":true}`)

	identity := nullIdentity(t)
	resp := frameworkresource.UpdateResponse{State: state, Identity: &identity}
	r.Update(ctx, frameworkresource.UpdateRequest{
		State: state, Plan: tfsdk.Plan{Schema: s, Raw: plan.Raw},
	}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}

	got := reqs()
	if len(got) != 1 || got[0].Method != http.MethodPatch || got[0].ContentType != "application/json-patch+json" {
		t.Fatalf("expected one JSON patch, got %+v", got)
	}
	var ops []map[string]any
	if err := json.Unmarshal([]byte(got[0].Body), &ops); err != nil {
		t.Fatal(err)
	}
	var sawData, sawImmutable bool
	for _, op := range ops {
		switch p := op["path"].(string); {
		case strings.HasPrefix(p, "/data"):
			sawData = true
		case p == "/immutable":
			sawImmutable = true
			if op["op"] != "add" || op["value"] != true {
				t.Errorf("immutable op = %v", op)
			}
		default:
			t.Errorf("unexpected patch op: %v", op)
		}
	}
	if !sawData || !sawImmutable {
		t.Fatalf("patch missing data or immutable: %s", got[0].Body)
	}
}

// With nothing to change on the API (null -> {} only), Update must GET rather than send an
// empty patch, and must still resolve computed metadata.
func TestConfigMapUpdateNoOpReadsInsteadOfPatching(t *testing.T) {
	ctx := context.Background()
	s := configMapSchema(t).Schema
	r, reqs := configMapServer(t, http.StatusOK, map[string]any{
		"apiVersion": "v1", "kind": "ConfigMap",
		"metadata": map[string]any{"name": "cfg", "namespace": "team", "uid": "uid-1", "resourceVersion": "10"},
		"data":     map[string]any{"one": "first"},
	})
	state := configMapState(t, storedConfigMap)
	plan := configMapState(t, strings.Replace(storedConfigMap, `"binary_data":null`, `"binary_data":{}`, 1))

	identity := nullIdentity(t)
	resp := frameworkresource.UpdateResponse{State: state, Identity: &identity}
	r.Update(ctx, frameworkresource.UpdateRequest{State: state, Plan: tfsdk.Plan{Schema: s, Raw: plan.Raw}}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	got := reqs()
	if len(got) != 1 || got[0].Method != http.MethodGet {
		t.Fatalf("expected a single GET, got %+v", got)
	}
}

func TestConfigMapCreateSendsImmutableFalseAndDecodesBinary(t *testing.T) {
	ctx := context.Background()
	s := configMapSchema(t).Schema
	r, reqs := configMapServer(t, http.StatusCreated, map[string]any{
		"apiVersion": "v1", "kind": "ConfigMap",
		"metadata": map[string]any{"name": "cfg", "namespace": "team", "uid": "uid-1", "resourceVersion": "1"},
	})
	plan := configMapState(t, `{
		"id":null,
		"metadata":[{"name":"cfg","namespace":"team","uid":null,"resource_version":null,"generation":null,"labels":null,"annotations":null,"generate_name":null}],
		"data":{"one":"first"},"binary_data":{"bin":"aGVsbG8="},"immutable":false}`)

	identity := nullIdentity(t)
	resp := frameworkresource.CreateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}, Identity: &identity}
	r.Create(ctx, frameworkresource.CreateRequest{Plan: tfsdk.Plan{Schema: s, Raw: plan.Raw}}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	got := reqs()
	if len(got) != 1 || got[0].Method != http.MethodPost || got[0].Path != "/api/v1/namespaces/team/configmaps" {
		t.Fatalf("unexpected requests: %+v", got)
	}
	var sent map[string]any
	if err := json.Unmarshal([]byte(got[0].Body), &sent); err != nil {
		t.Fatal(err)
	}
	if v, ok := sent["immutable"]; !ok || v != false {
		t.Errorf("immutable=false must be sent explicitly, as SDKv2 did: %v", sent["immutable"])
	}
	// The API wire format base64-encodes []byte, so "aGVsbG8=" must come out unchanged,
	// proving the provider decoded exactly once.
	if bd, _ := sent["binaryData"].(map[string]any); bd["bin"] != "aGVsbG8=" {
		t.Errorf("binaryData round trip wrong: %v", sent["binaryData"])
	}
	var created corev1.ConfigMapV1Model
	resp.Diagnostics.Append(resp.State.Get(ctx, &created)...)
	if resp.Diagnostics.HasError() || created.ID.ValueString() != "team/cfg" || created.Metadata[0].UID.ValueString() != "uid-1" {
		t.Fatalf("state not populated from response: %#v %v", created, resp.Diagnostics)
	}
}

func TestConfigMapImportState(t *testing.T) {
	ctx := context.Background()
	s := configMapSchema(t).Schema
	r := &corev1.ConfigMapV1{}

	t.Run("id", func(t *testing.T) {
		resp := frameworkresource.ImportStateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}}
		r.ImportState(ctx, frameworkresource.ImportStateRequest{ID: "team/cfg"}, &resp)
		var id types.String
		resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("id"), &id)...)
		if resp.Diagnostics.HasError() || id.ValueString() != "team/cfg" {
			t.Fatalf("id = %q, %v", id.ValueString(), resp.Diagnostics)
		}
	})

	identityFor := func(t *testing.T, id common.NamespacedResourceIdentity) *tfsdk.ResourceIdentity {
		t.Helper()
		ident := nullIdentity(t)
		if d := ident.Set(ctx, id); d.HasError() {
			t.Fatal(d)
		}
		return &ident
	}

	for _, tc := range []struct {
		name      string
		identity  common.NamespacedResourceIdentity
		wantID    string
		wantError bool
	}{
		{"identity", configMapIdentityValue("team", "cfg", "v1", "ConfigMap"), "team/cfg", false},
		{"identity defaults namespace", configMapIdentityValue("", "cfg", "v1", "ConfigMap"), "default/cfg", false},
		{"wrong kind", configMapIdentityValue("team", "cfg", "v1", "Secret"), "", true},
		{"wrong api version", configMapIdentityValue("team", "cfg", "apps/v1", "ConfigMap"), "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := frameworkresource.ImportStateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}}
			r.ImportState(ctx, frameworkresource.ImportStateRequest{Identity: identityFor(t, tc.identity)}, &resp)
			if resp.Diagnostics.HasError() != tc.wantError {
				t.Fatalf("diagnostics = %v, wantError %v", resp.Diagnostics, tc.wantError)
			}
			if tc.wantError {
				return
			}
			var id types.String
			resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("id"), &id)...)
			if id.ValueString() != tc.wantID {
				t.Fatalf("id = %q, want %q", id.ValueString(), tc.wantID)
			}
		})
	}
}

func TestConfigMapUpgradeIdentity(t *testing.T) {
	ctx := context.Background()
	r := &corev1.ConfigMapV1{}
	upgrader, ok := r.UpgradeIdentity(ctx)[0]
	if !ok {
		t.Fatal("missing version 0 upgrader")
	}
	schema := common.NamespacedIdentitySchema()
	for _, tc := range []struct {
		name, raw string
		wantError bool
	}{
		{name: "preIdentity"},
		{name: "stored", raw: `{"name":"cfg","namespace":"team"}`},
		{name: "invalidJSON", raw: `{`, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := frameworkresource.UpgradeIdentityRequest{}
			if tc.raw != "" {
				req.RawIdentity = &tfprotov6.RawState{JSON: []byte(tc.raw)}
			}
			resp := frameworkresource.UpgradeIdentityResponse{Identity: &tfsdk.ResourceIdentity{
				Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil),
			}}
			upgrader.IdentityUpgrader(ctx, req, &resp)
			if resp.Diagnostics.HasError() != tc.wantError {
				t.Fatalf("diagnostics: %v", resp.Diagnostics)
			}
			if tc.wantError {
				return
			}
			var got common.NamespacedResourceIdentity
			if d := resp.Identity.Get(ctx, &got); d.HasError() {
				t.Fatal(d)
			}
			if tc.raw == "" {
				if !got.Name.IsNull() || !got.Namespace.IsNull() || !got.Kind.IsNull() || !got.APIVersion.IsNull() {
					t.Fatalf("pre-identity state must stay null until Read: %#v", got)
				}
			} else if got.Name.ValueString() != "cfg" || got.Namespace.ValueString() != "team" ||
				got.Kind.ValueString() != "ConfigMap" || got.APIVersion.ValueString() != "v1" {
				t.Fatalf("incorrect upgraded identity: %#v", got)
			}
		})
	}
}

func configMapIdentityValue(namespace, name, apiVersion, kind string) common.NamespacedResourceIdentity {
	ns := types.StringValue(namespace)
	if namespace == "" {
		ns = types.StringNull()
	}
	return common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{
			Name: types.StringValue(name), Kind: types.StringValue(kind), APIVersion: types.StringValue(apiVersion),
		},
		Namespace: ns,
	}
}

const configMapResourceName = "kubernetes_config_map_v1.test"

// Acceptance tests for the Framework kubernetes_config_map_v1. They port the SDKv2 tests in
// kubernetes/resource_kubernetes_config_map_v1_test.go (basic, binaryData, generatedName,
// immutable, identity) and add the cases only the Framework can express: null versus empty
// maps, drift, and a ConfigMap deleted behind Terraform's back.

func testAccCheckConfigMapV1Data(m *k8sv1.ConfigMap, expected map[string]string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if len(expected) == 0 && len(m.Data) == 0 {
			return nil
		}
		if !reflect.DeepEqual(m.Data, expected) {
			return fmt.Errorf("%s data don't match.\nExpected: %q\nGiven: %q", m.Name, expected, m.Data)
		}
		return nil
	}
}

func testAccCheckKubernetesConfigMapV1Destroy(s *terraform.State) error {
	conn, err := mainClientset()
	if err != nil {
		return err
	}
	ctx := context.Background()
	for _, rs := range s.RootModule().Resources {
		// Both type names: a MoveState test that fails partway leaves state under the
		// deprecated address, and matching only one would let a leak pass silently.
		if rs.Type != "kubernetes_config_map_v1" && rs.Type != "kubernetes_config_map" {
			continue
		}
		namespace, name, err := kubernetes.IdParts(rs.Primary.ID)
		if err != nil {
			return err
		}
		_, err = conn.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("checking config map %q destruction: %w", rs.Primary.ID, err)
		}
		return fmt.Errorf("config map still exists: %s", rs.Primary.ID)
	}
	return nil
}

func testAccCheckKubernetesConfigMapV1Exists(n string, obj *k8sv1.ConfigMap) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}
		conn, err := mainClientset()
		if err != nil {
			return err
		}
		namespace, name, err := kubernetes.IdParts(rs.Primary.ID)
		if err != nil {
			return err
		}
		out, err := conn.CoreV1().ConfigMaps(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		*obj = *out
		return nil
	}
}

// testAccCheckConfigMapNotRecreated fails if the ConfigMap was replaced rather than updated
// in place. UID is stable for an object's lifetime, so a change means delete+create.
func testAccCheckConfigMapNotRecreated(before, after *k8sv1.ConfigMap) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if before.UID != after.UID {
			return fmt.Errorf("config map was recreated: uid %s -> %s", before.UID, after.UID)
		}
		return nil
	}
}

// testAccConfigMapFixtureDir is the absolute path of kubernetes/test-fixtures. Absolute
// because Terraform resolves filebase64 against its own working directory, not the
// package under test.
func testAccConfigMapFixtureDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs("../../../../kubernetes/test-fixtures")
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestAccKubernetesConfigMapV1_basic(t *testing.T) {
	var conf k8sv1.ConfigMap
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesConfigMapV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccConfigMapV1Config_emptyData(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapV1Exists(configMapResourceName, &conf),
					resource.TestCheckResourceAttr(configMapResourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(configMapResourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(configMapResourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(configMapResourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(configMapResourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(configMapResourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(configMapResourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(configMapResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(configMapResourceName, "metadata.0.namespace", "default"),
					resource.TestCheckResourceAttrSet(configMapResourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(configMapResourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(configMapResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(configMapResourceName, "data.%", "0"),
					resource.TestCheckResourceAttr(configMapResourceName, "immutable", "false"),
				),
			},
			{
				Config: testAccConfigMapV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapV1Exists(configMapResourceName, &conf),
					resource.TestCheckResourceAttr(configMapResourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(configMapResourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(configMapResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(configMapResourceName, "data.%", "2"),
					resource.TestCheckResourceAttr(configMapResourceName, "data.one", "first"),
					resource.TestCheckResourceAttr(configMapResourceName, "data.two", "second"),
					testAccCheckConfigMapV1Data(&conf, map[string]string{"one": "first", "two": "second"}),
				),
			},
			// Imported against the non-empty config: import cannot know whether `data` was
			// written as {} or omitted, so it reads an empty map back as null.
			{
				ResourceName:            configMapResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccConfigMapV1Config_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapV1Exists(configMapResourceName, &conf),
					resource.TestCheckResourceAttr(configMapResourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(configMapResourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(configMapResourceName, "metadata.0.annotations.Different", "1234"),
					resource.TestCheckResourceAttr(configMapResourceName, "metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr(configMapResourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(configMapResourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(configMapResourceName, "data.%", "3"),
					resource.TestCheckResourceAttr(configMapResourceName, "data.nine", "ninth"),
					testAccCheckConfigMapV1Data(&conf, map[string]string{"one": "first", "two": "second", "nine": "ninth"}),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(configMapResourceName, plancheck.ResourceActionUpdate),
					},
				},
			},
			// Everything optional removed: omitted maps are null, not {}.
			{
				Config: testAccConfigMapV1Config_noData(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapV1Exists(configMapResourceName, &conf),
					resource.TestCheckResourceAttr(configMapResourceName, "metadata.0.name", name),
					resource.TestCheckNoResourceAttr(configMapResourceName, "data.%"),
					testAccCheckConfigMapV1Data(&conf, map[string]string{}),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(configMapResourceName,
						tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("labels"), knownvalue.Null()),
					statecheck.ExpectKnownValue(configMapResourceName,
						tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("annotations"), knownvalue.Null()),
					statecheck.ExpectKnownValue(configMapResourceName, tfjsonpath.New("data"), knownvalue.Null()),
				},
			},
		},
	})
}

func TestAccKubernetesConfigMapV1_binaryData(t *testing.T) {
	var conf k8sv1.ConfigMap
	prefix := "tf-acc-test-gen-"
	dir := testAccConfigMapFixtureDir(t)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesConfigMapV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccConfigMapV1Config_binaryData(prefix, dir),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapV1Exists(configMapResourceName, &conf),
					resource.TestCheckResourceAttr(configMapResourceName, "binary_data.%", "1"),
					resource.TestCheckResourceAttr(configMapResourceName, "data.%", "1"),
					resource.TestCheckResourceAttr(configMapResourceName, "data.two", "second"),
				),
			},
			{
				Config: testAccConfigMapV1Config_binaryData2(prefix, dir),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapV1Exists(configMapResourceName, &conf),
					resource.TestCheckResourceAttr(configMapResourceName, "binary_data.%", "3"),
					resource.TestCheckResourceAttr(configMapResourceName, "binary_data.raw", "UmF3IGRhdGEgc2hvdWxkIGNvbWUgYmFjayBhcyBpcyBpbiB0aGUgcG9k"),
					resource.TestCheckResourceAttr(configMapResourceName, "data.%", "1"),
					resource.TestCheckResourceAttr(configMapResourceName, "data.three", "third"),
				),
			},
			// The API holds decoded bytes; Read must re-encode them to the planned base64
			// or this refresh-and-plan would be non-empty.
			{
				Config: testAccConfigMapV1Config_binaryData2(prefix, dir),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestAccKubernetesConfigMapV1_invalidBinaryData(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesConfigMapV1Destroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccConfigMapV1Config_invalidBinaryData(name),
				ExpectError: regexp.MustCompile(`(?i)base64`),
			},
		},
	})
}

func TestAccKubernetesConfigMapV1_generatedName(t *testing.T) {
	var conf k8sv1.ConfigMap
	prefix := "tf-acc-test-gen-"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesConfigMapV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccConfigMapV1Config_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapV1Exists(configMapResourceName, &conf),
					resource.TestCheckResourceAttr(configMapResourceName, "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr(configMapResourceName, "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet(configMapResourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(configMapResourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(configMapResourceName, "metadata.0.uid"),
				),
			},
			{
				ResourceName:            configMapResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			// A server-assigned name must not make the next plan replace the object.
			{
				Config: testAccConfigMapV1Config_generatedName(prefix),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestAccKubernetesConfigMapV1_nameAndGenerateName(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesConfigMapV1Destroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccConfigMapV1Config_nameAndGenerateName(name),
				ExpectError: regexp.MustCompile(`(?is)generate_name|name`),
			},
		},
	})
}

func TestAccKubernetesConfigMapV1_immutable(t *testing.T) {
	var before, after k8sv1.ConfigMap
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesConfigMapV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccConfigMapV1Config_immutable(name, true, "second"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapV1Exists(configMapResourceName, &before),
					resource.TestCheckResourceAttr(configMapResourceName, "immutable", "true"),
					resource.TestCheckResourceAttr(configMapResourceName, "data.two", "second"),
				),
			},
			// The API server rejects changes to an immutable ConfigMap, so every one of
			// these replaces it, as in SDKv2.
			{
				Config: testAccConfigMapV1Config_immutable(name, true, "third"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(configMapResourceName, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapV1Exists(configMapResourceName, &after),
					resource.TestCheckResourceAttr(configMapResourceName, "data.two", "third"),
				),
			},
			{
				Config: testAccConfigMapV1Config_immutable(name, false, "third"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(configMapResourceName, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.TestCheckResourceAttr(configMapResourceName, "immutable", "false"),
			},
			// Mutable again: ordinary in-place update.
			{
				Config: testAccConfigMapV1Config_immutable(name, false, "fourth"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(configMapResourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestCheckResourceAttr(configMapResourceName, "data.two", "fourth"),
			},
		},
	})
}

// Turning immutable on is an ordinary update, because replacement is decided by the
// *prior* value. Only afterwards does the object become fixed.
func TestAccKubernetesConfigMapV1_enableImmutable(t *testing.T) {
	var before, after k8sv1.ConfigMap
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesConfigMapV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccConfigMapV1Config_immutable(name, false, "second"),
				Check:  testAccCheckKubernetesConfigMapV1Exists(configMapResourceName, &before),
			},
			{
				Config: testAccConfigMapV1Config_immutable(name, true, "second"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(configMapResourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapV1Exists(configMapResourceName, &after),
					testAccCheckConfigMapNotRecreated(&before, &after),
					resource.TestCheckResourceAttr(configMapResourceName, "immutable", "true"),
					func(*terraform.State) error {
						if after.Immutable == nil || !*after.Immutable {
							return fmt.Errorf("API object is not immutable")
						}
						return nil
					},
				),
			},
		},
	})
}

// Metadata is the one thing that stays mutable on an immutable ConfigMap.
func TestAccKubernetesConfigMapV1_immutableMetadataUpdatesInPlace(t *testing.T) {
	var before, after k8sv1.ConfigMap
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesConfigMapV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccConfigMapV1Config_immutable(name, true, "second"),
				Check:  testAccCheckKubernetesConfigMapV1Exists(configMapResourceName, &before),
			},
			{
				Config: testAccConfigMapV1Config_immutableChangedLabel(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(configMapResourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapV1Exists(configMapResourceName, &after),
					testAccCheckConfigMapNotRecreated(&before, &after),
					resource.TestCheckResourceAttr(configMapResourceName, "metadata.0.labels.TestLabelOne", "changed"),
				),
			},
		},
	})
}

func TestAccKubernetesConfigMapV1_namespace(t *testing.T) {
	var conf k8sv1.ConfigMap
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesConfigMapV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccConfigMapV1Config_namespace(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapV1Exists(configMapResourceName, &conf),
					resource.TestCheckResourceAttr(configMapResourceName, "metadata.0.namespace", "kube-public"),
					resource.TestCheckResourceAttr(configMapResourceName, "id", "kube-public/"+name),
				),
			},
			{
				ResourceName:            configMapResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
		},
	})
}

func TestAccKubernetesConfigMapV1_identity(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesConfigMapV1Destroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccConfigMapV1Config_basic(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(configMapResourceName, map[string]knownvalue.Check{
						"namespace":   knownvalue.StringExact("default"),
						"name":        knownvalue.StringExact(name),
						"api_version": knownvalue.StringExact("v1"),
						"kind":        knownvalue.StringExact("ConfigMap"),
					}),
				},
			},
			{
				ResourceName:    configMapResourceName,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func TestAccKubernetesConfigMapV1_importInvalidID(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesConfigMapV1Destroy,
		Steps: []resource.TestStep{
			{Config: testAccConfigMapV1Config_basic(name)},
			{
				ResourceName:  configMapResourceName,
				ImportState:   true,
				ImportStateId: "no-separator",
				ExpectError:   regexp.MustCompile(`Unexpected ID format`),
			},
			{
				ResourceName:  configMapResourceName,
				ImportState:   true,
				ImportStateId: "default/",
				ExpectError:   regexp.MustCompile(`Unexpected ID format`),
			},
		},
	})
}

// A ConfigMap that is already gone when Terraform refreshes is removed from state and
// recreated, not an error.
func TestAccKubernetesConfigMapV1_disappears(t *testing.T) {
	var conf k8sv1.ConfigMap
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesConfigMapV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccConfigMapV1Config_basic(name),
				Check:  testAccCheckKubernetesConfigMapV1Exists(configMapResourceName, &conf),
			},
			{
				PreConfig: func() {
					conn, err := mainClientset()
					if err != nil {
						t.Fatal(err)
					}
					if err := conn.CoreV1().ConfigMaps("default").Delete(context.Background(), name, metav1.DeleteOptions{}); err != nil {
						t.Fatal(err)
					}
				},
				Config: testAccConfigMapV1Config_basic(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(configMapResourceName, plancheck.ResourceActionCreate),
					},
				},
				Check: testAccCheckKubernetesConfigMapV1Exists(configMapResourceName, &conf),
			},
		},
	})
}

// Drift: data changed outside Terraform is detected on refresh and reverted by apply.
func TestAccKubernetesConfigMapV1_drift(t *testing.T) {
	var conf k8sv1.ConfigMap
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesConfigMapV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccConfigMapV1Config_basic(name),
				Check:  testAccCheckKubernetesConfigMapV1Exists(configMapResourceName, &conf),
			},
			{
				PreConfig: func() {
					conn, err := mainClientset()
					if err != nil {
						t.Fatal(err)
					}
					cm, err := conn.CoreV1().ConfigMaps("default").Get(context.Background(), name, metav1.GetOptions{})
					if err != nil {
						t.Fatal(err)
					}
					cm.Data["one"] = "tampered"
					if _, err := conn.CoreV1().ConfigMaps("default").Update(context.Background(), cm, metav1.UpdateOptions{}); err != nil {
						t.Fatal(err)
					}
				},
				Config: testAccConfigMapV1Config_basic(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(configMapResourceName, plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapV1Exists(configMapResourceName, &conf),
					testAccCheckConfigMapV1Data(&conf, map[string]string{"one": "first", "two": "second"}),
				),
			},
		},
	})
}

func testAccConfigMapV1Metadata(name string) string {
	return fmt.Sprintf(`  metadata {
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
`, name)
}

func testAccConfigMapV1Config_emptyData(name string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1" "test" {
%s
  data = {}
}
`, testAccConfigMapV1Metadata(name))
}

func testAccConfigMapV1Config_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1" "test" {
%s
  data = {
    one = "first"
    two = "second"
  }
}
`, testAccConfigMapV1Metadata(name))
}

func testAccConfigMapV1Config_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      Different         = "1234"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
    }

    name = "%s"
  }

  data = {
    one  = "first"
    two  = "second"
    nine = "ninth"
  }
}
`, name)
}

func testAccConfigMapV1Config_noData(name string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1" "test" {
  metadata {
    name = "%s"
  }
}
`, name)
}

func testAccConfigMapV1Config_generatedName(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1" "test" {
  metadata {
    generate_name = "%s"
  }

  data = {
    one = "first"
    two = "second"
  }
}
`, prefix)
}

func testAccConfigMapV1Config_binaryData(prefix, dir string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1" "test" {
  metadata {
    generate_name = "%s"
  }

  binary_data = {
    one = filebase64("%s/binary.data")
  }

  data = {
    two = "second"
  }
}
`, prefix, dir)
}

func testAccConfigMapV1Config_binaryData2(prefix, dir string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1" "test" {
  metadata {
    generate_name = "%s"
  }

  binary_data = {
    one = filebase64("%s/binary.data")
    two = filebase64("%s/binary2.data")
    raw = base64encode("Raw data should come back as is in the pod")
  }

  data = {
    three = "third"
  }
}
`, prefix, dir, dir)
}

func testAccConfigMapV1Config_invalidBinaryData(name string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1" "test" {
  metadata {
    name = "%s"
  }

  binary_data = {
    bad = "not base64!"
  }
}
`, name)
}

func testAccConfigMapV1Config_nameAndGenerateName(name string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1" "test" {
  metadata {
    name          = "%s"
    generate_name = "%s-"
  }
}
`, name, name)
}

func testAccConfigMapV1Config_immutable(name string, immutable bool, data string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1" "test" {
%s
  immutable = %t

  data = {
    one = "first"
    two = "%s"
  }
}
`, testAccConfigMapV1Metadata(name), immutable, data)
}

func testAccConfigMapV1Config_immutableChangedLabel(name string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }

    labels = {
      TestLabelOne   = "changed"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    name = "%s"
  }

  immutable = true

  data = {
    one = "first"
    two = "second"
  }
}
`, name)
}

func testAccConfigMapV1Config_namespace(name string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1" "test" {
  metadata {
    name      = "%s"
    namespace = "kube-public"
  }

  data = {
    one = "first"
  }
}
`, name)
}
