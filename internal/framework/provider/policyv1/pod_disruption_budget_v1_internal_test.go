// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package policyv1

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	policy "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
)

func pdbTestSchema(t *testing.T) schema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	(&PodDisruptionBudgetV1{}).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	pdbNoErrors(t, resp.Diagnostics)
	return resp.Schema
}

func pdbNoErrors(t *testing.T, diags diag.Diagnostics) {
	t.Helper()
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
}

func pdbProtocolNoErrors(t *testing.T, diags []*tfprotov6.Diagnostic) {
	t.Helper()
	for _, d := range diags {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("unexpected protocol diagnostic: %s: %s", d.Summary, d.Detail)
		}
	}
}

func pdbTestModel() podDisruptionBudgetV1Model {
	return podDisruptionBudgetV1Model{
		ID: types.StringValue("test/budget"),
		Metadata: []common.NamespacedMetadataModel{{
			MetadataModel: common.MetadataModel{
				MetadataBase: common.MetadataBase{
					Annotations: types.MapNull(types.StringType), Labels: types.MapNull(types.StringType),
					Name: types.StringValue("budget"), Generation: types.Int64Value(1),
					ResourceVersion: types.StringValue("1"), UID: types.StringValue("uid"),
				},
				GenerateName: types.StringNull(),
			},
			Namespace: types.StringValue("test"),
		}},
		Spec: []podDisruptionBudgetSpecModel{{
			MinAvailable: types.StringValue("1"), MaxUnavailable: types.StringValue(""),
			Selector: []pdbLabelSelectorModel{{
				MatchLabels: types.MapNull(types.StringType), MatchExpressions: []pdbLabelRequirementModel{},
			}},
		}},
	}
}

func pdbTestState(t *testing.T, model podDisruptionBudgetV1Model) tfsdk.State {
	t.Helper()
	state := tfsdk.State{Schema: pdbTestSchema(t)}
	pdbNoErrors(t, state.Set(context.Background(), &model))
	return state
}

func pdbTestConfig(model podDisruptionBudgetV1Model) podDisruptionBudgetV1Model {
	model.ID = types.StringNull()
	model.Metadata = append([]common.NamespacedMetadataModel(nil), model.Metadata...)
	model.Metadata[0].Generation = types.Int64Null()
	model.Metadata[0].ResourceVersion = types.StringNull()
	model.Metadata[0].UID = types.StringNull()
	return model
}

type pdbProtocolProvider struct{}

func (pdbProtocolProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "kubernetes"
}
func (pdbProtocolProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = providerschema.Schema{}
}
func (pdbProtocolProvider) Configure(context.Context, provider.ConfigureRequest, *provider.ConfigureResponse) {
}
func (pdbProtocolProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{NewPodDisruptionBudgetV1}
}
func (pdbProtocolProvider) DataSources(context.Context) []func() datasource.DataSource { return nil }

func pdbTestDynamic(t *testing.T, state tfsdk.State) *tfprotov6.DynamicValue {
	t.Helper()
	value, err := tfprotov6.NewDynamicValue(state.Schema.Type().TerraformType(context.Background()), state.Raw)
	if err != nil {
		t.Fatal(err)
	}
	return &value
}

func pdbTestProtocolPlan(t *testing.T, before, after podDisruptionBudgetV1Model) *tfprotov6.PlanResourceChangeResponse {
	t.Helper()
	ctx := context.Background()
	server := providerserver.NewProtocol6(pdbProtocolProvider{})()
	prior := pdbTestState(t, before)
	config := pdbTestState(t, pdbTestConfig(after))
	proposed := pdbTestState(t, after)
	resp, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
		TypeName: "kubernetes_pod_disruption_budget_v1", PriorState: pdbTestDynamic(t, prior),
		Config: pdbTestDynamic(t, config), ProposedNewState: pdbTestDynamic(t, proposed),
	})
	if err != nil {
		t.Fatal(err)
	}
	pdbProtocolNoErrors(t, resp.Diagnostics)
	return resp
}

func TestPDBSchemaContract(t *testing.T) {
	s := pdbTestSchema(t)
	if s.Version != 0 || len(s.Blocks) != 2 || len(s.Attributes) != 1 {
		t.Fatalf("unexpected schema surface: %#v", s)
	}
	r := NewPodDisruptionBudgetV1()
	if _, ok := r.(resource.ResourceWithIdentity); ok {
		t.Fatal("PDB must not gain a new identity")
	}
	if _, ok := r.(resource.ResourceWithMoveState); ok {
		t.Fatal("policy/v1beta1 is not a compatible alias")
	}
	var metadata resource.MetadataResponse
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "kubernetes"}, &metadata)
	if metadata.TypeName != "kubernetes_pod_disruption_budget_v1" {
		t.Fatal(metadata.TypeName)
	}
	spec := s.Blocks["spec"].(schema.ListNestedBlock)
	for _, name := range []string{"min_available", "max_unavailable"} {
		a := spec.NestedObject.Attributes[name].(schema.StringAttribute)
		if !a.Optional || !a.Computed || a.Default == nil {
			t.Fatalf("%s lost its legacy empty-string default", name)
		}
	}
	selector := spec.NestedObject.Blocks["selector"].(schema.ListNestedBlock)
	expressions := selector.NestedObject.Blocks["match_expressions"].(schema.ListNestedBlock)
	if _, ok := expressions.NestedObject.Attributes["values"].(schema.SetAttribute); !ok {
		t.Fatal("expression values must remain a set")
	}
	ctx := context.Background()
	server := providerserver.NewProtocol6(pdbProtocolProvider{})()
	resp, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	pdbProtocolNoErrors(t, resp.Diagnostics)
}

func TestPDBThresholdValidation(t *testing.T) {
	for _, value := range []string{"", "0", "1", "+01", "-1", "-2147483648", "2147483647", "0%", "+01%", "100%", "-0%"} {
		t.Run("valid-"+value, func(t *testing.T) {
			if err := validateNullableIntOrPercent(value); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, value := range []string{"2147483648", "-2147483649", "-1%", "101%", "1.5", "1.5%", " 1", "1 ", "%", "1%%", "x"} {
		t.Run("invalid-"+value, func(t *testing.T) {
			if err := validateNullableIntOrPercent(value); err == nil {
				t.Fatalf("accepted %q", value)
			}
		})
	}
	for _, value := range []types.String{types.StringNull(), types.StringUnknown()} {
		var resp validator.StringResponse
		nullableIntOrPercentValidator{}.ValidateString(context.Background(), validator.StringRequest{ConfigValue: value}, &resp)
		pdbNoErrors(t, resp.Diagnostics)
	}
}

func TestPDBProtocolValidation(t *testing.T) {
	for _, tc := range []struct {
		name    string
		mutate  func(*podDisruptionBudgetV1Model)
		wantErr bool
	}{
		{"valid", func(m *podDisruptionBudgetV1Model) {}, false},
		{"missing-spec", func(m *podDisruptionBudgetV1Model) { m.Spec = nil }, true},
		{"empty-spec", func(m *podDisruptionBudgetV1Model) { m.Spec = []podDisruptionBudgetSpecModel{} }, true},
		{"missing-selector", func(m *podDisruptionBudgetV1Model) { m.Spec[0].Selector = nil }, true},
		{"multiple-selectors", func(m *podDisruptionBudgetV1Model) {
			m.Spec[0].Selector = append(m.Spec[0].Selector, m.Spec[0].Selector[0])
		}, true},
		{"invalid-threshold", func(m *podDisruptionBudgetV1Model) { m.Spec[0].MinAvailable = types.StringValue("101%") }, true},
		{"unknown-threshold", func(m *podDisruptionBudgetV1Model) { m.Spec[0].MinAvailable = types.StringUnknown() }, false},
		{"neither-threshold", func(m *podDisruptionBudgetV1Model) { m.Spec[0].MinAvailable = types.StringNull() }, false},
		{"both-thresholds", func(m *podDisruptionBudgetV1Model) { m.Spec[0].MaxUnavailable = types.StringValue("1") }, false},
		{"expression-legacy-zero", func(m *podDisruptionBudgetV1Model) {
			m.Spec[0].Selector[0].MatchExpressions = []pdbLabelRequirementModel{{Key: types.StringNull(), Operator: types.StringNull(), Values: types.SetNull(types.StringType)}}
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			model := pdbTestConfig(pdbTestModel())
			tc.mutate(&model)
			server := providerserver.NewProtocol6(pdbProtocolProvider{})()
			resp, err := server.ValidateResourceConfig(context.Background(), &tfprotov6.ValidateResourceConfigRequest{
				TypeName: "kubernetes_pod_disruption_budget_v1", Config: pdbTestDynamic(t, pdbTestState(t, model)),
			})
			if err != nil {
				t.Fatal(err)
			}
			hasError := false
			for _, d := range resp.Diagnostics {
				hasError = hasError || d.Severity == tfprotov6.DiagnosticSeverityError
			}
			if hasError != tc.wantErr {
				t.Fatalf("want error %t, got %v", tc.wantErr, resp.Diagnostics)
			}
		})
	}
}

func TestPDBProtocolSpecReplacement(t *testing.T) {
	expression := func(key string, values types.Set) pdbLabelRequirementModel {
		return pdbLabelRequirementModel{Key: types.StringValue(key), Operator: types.StringValue("In"), Values: values}
	}
	set := func(values ...string) types.Set {
		v, d := types.SetValueFrom(context.Background(), types.StringType, values)
		pdbNoErrors(t, d)
		return v
	}
	labels := types.MapValueMust(types.StringType, map[string]attr.Value{"app": types.StringValue("test")})
	for _, tc := range []struct {
		name   string
		mutate func(*podDisruptionBudgetV1Model, *podDisruptionBudgetV1Model)
		want   bool
	}{
		{"unchanged", func(a, b *podDisruptionBudgetV1Model) {}, false},
		{"empty-labels-added", func(a, b *podDisruptionBudgetV1Model) {
			b.Spec[0].Selector[0].MatchLabels = types.MapValueMust(types.StringType, map[string]attr.Value{})
		}, false},
		{"empty-labels-removed", func(a, b *podDisruptionBudgetV1Model) {
			a.Spec[0].Selector[0].MatchLabels = types.MapValueMust(types.StringType, map[string]attr.Value{})
		}, false},
		{"labels-added", func(a, b *podDisruptionBudgetV1Model) { b.Spec[0].Selector[0].MatchLabels = labels }, true},
		{"labels-removed", func(a, b *podDisruptionBudgetV1Model) { a.Spec[0].Selector[0].MatchLabels = labels }, true},
		{"label-value-changed", func(a, b *podDisruptionBudgetV1Model) {
			a.Spec[0].Selector[0].MatchLabels = labels
			b.Spec[0].Selector[0].MatchLabels = types.MapValueMust(types.StringType, map[string]attr.Value{"app": types.StringValue("changed")})
		}, true},
		{"threshold-changed", func(a, b *podDisruptionBudgetV1Model) { b.Spec[0].MinAvailable = types.StringValue("2") }, true},
		{"threshold-removed", func(a, b *podDisruptionBudgetV1Model) { b.Spec[0].MinAvailable = types.StringNull() }, true},
		{"legacy-threshold-default", func(a, b *podDisruptionBudgetV1Model) { b.Spec[0].MaxUnavailable = types.StringNull() }, false},
		{"expressions-added", func(a, b *podDisruptionBudgetV1Model) {
			b.Spec[0].Selector[0].MatchExpressions = []pdbLabelRequirementModel{expression("app", set("x"))}
		}, true},
		{"expressions-removed", func(a, b *podDisruptionBudgetV1Model) {
			a.Spec[0].Selector[0].MatchExpressions = []pdbLabelRequirementModel{expression("app", set("x"))}
		}, true},
		{"expression-inserted-between", func(a, b *podDisruptionBudgetV1Model) {
			a.Spec[0].Selector[0].MatchExpressions = []pdbLabelRequirementModel{expression("app", set("x")), expression("other", set("y"))}
			b.Spec[0].Selector[0].MatchExpressions = []pdbLabelRequirementModel{expression("app", set("x")), expression("new", set("z")), expression("other", set("y"))}
		}, true},
		{"expression-removed-between", func(a, b *podDisruptionBudgetV1Model) {
			a.Spec[0].Selector[0].MatchExpressions = []pdbLabelRequirementModel{expression("app", set("x")), expression("old", set("z")), expression("other", set("y"))}
			b.Spec[0].Selector[0].MatchExpressions = []pdbLabelRequirementModel{expression("app", set("x")), expression("other", set("y"))}
		}, true},
		{"expressions-reordered", func(a, b *podDisruptionBudgetV1Model) {
			a.Spec[0].Selector[0].MatchExpressions = []pdbLabelRequirementModel{expression("app", set("x")), expression("other", set("y"))}
			b.Spec[0].Selector[0].MatchExpressions = []pdbLabelRequirementModel{expression("other", set("y")), expression("app", set("x"))}
		}, true},
		{"expression-key-changed", func(a, b *podDisruptionBudgetV1Model) {
			a.Spec[0].Selector[0].MatchExpressions = []pdbLabelRequirementModel{expression("app", set("x"))}
			b.Spec[0].Selector[0].MatchExpressions = []pdbLabelRequirementModel{expression("other", set("x"))}
		}, true},
		{"expression-operator-changed", func(a, b *podDisruptionBudgetV1Model) {
			a.Spec[0].Selector[0].MatchExpressions = []pdbLabelRequirementModel{expression("app", set("x"))}
			b.Spec[0].Selector[0].MatchExpressions = []pdbLabelRequirementModel{expression("app", set("x"))}
			b.Spec[0].Selector[0].MatchExpressions[0].Operator = types.StringValue("NotIn")
		}, true},
		{"expression-values-changed", func(a, b *podDisruptionBudgetV1Model) {
			a.Spec[0].Selector[0].MatchExpressions = []pdbLabelRequirementModel{expression("app", set("x"))}
			b.Spec[0].Selector[0].MatchExpressions = []pdbLabelRequirementModel{expression("app", set("y"))}
		}, true},
		{"expression-values-added", func(a, b *podDisruptionBudgetV1Model) {
			a.Spec[0].Selector[0].MatchExpressions = []pdbLabelRequirementModel{expression("app", types.SetNull(types.StringType))}
			b.Spec[0].Selector[0].MatchExpressions = []pdbLabelRequirementModel{expression("app", set("x"))}
		}, true},
		{"expression-values-reordered", func(a, b *podDisruptionBudgetV1Model) {
			a.Spec[0].Selector[0].MatchExpressions = []pdbLabelRequirementModel{expression("app", set("x", "y"))}
			b.Spec[0].Selector[0].MatchExpressions = []pdbLabelRequirementModel{expression("app", set("y", "x"))}
		}, false},
		{"expression-values-empty-added", func(a, b *podDisruptionBudgetV1Model) {
			a.Spec[0].Selector[0].MatchExpressions = []pdbLabelRequirementModel{expression("app", types.SetNull(types.StringType))}
			b.Spec[0].Selector[0].MatchExpressions = []pdbLabelRequirementModel{expression("app", types.SetValueMust(types.StringType, []attr.Value{}))}
		}, false},
		{"expression-values-empty-removed", func(a, b *podDisruptionBudgetV1Model) {
			a.Spec[0].Selector[0].MatchExpressions = []pdbLabelRequirementModel{expression("app", types.SetValueMust(types.StringType, []attr.Value{}))}
			b.Spec[0].Selector[0].MatchExpressions = []pdbLabelRequirementModel{expression("app", types.SetNull(types.StringType))}
		}, false},
		{"expression-values-removed", func(a, b *podDisruptionBudgetV1Model) {
			a.Spec[0].Selector[0].MatchExpressions = []pdbLabelRequirementModel{expression("app", set("x"))}
			b.Spec[0].Selector[0].MatchExpressions = []pdbLabelRequirementModel{expression("app", types.SetNull(types.StringType))}
		}, true},
		{"metadata-only", func(a, b *podDisruptionBudgetV1Model) { b.Metadata[0].Labels = labels }, false},
		{"unknown-threshold", func(a, b *podDisruptionBudgetV1Model) { b.Spec[0].MinAvailable = types.StringUnknown() }, true},
		{"unknown-labels", func(a, b *podDisruptionBudgetV1Model) {
			b.Spec[0].Selector[0].MatchLabels = types.MapUnknown(types.StringType)
		}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before, after := pdbTestModel(), pdbTestModel()
			tc.mutate(&before, &after)
			resp := pdbTestProtocolPlan(t, before, after)
			if got := len(resp.RequiresReplace) > 0; got != tc.want {
				t.Fatalf("RequiresReplace=%v, want %t", resp.RequiresReplace, tc.want)
			}
		})
	}
}

func TestPDBProtocolDefaults(t *testing.T) {
	before, after := pdbTestModel(), pdbTestModel()
	before.Spec[0].Selector[0].MatchExpressions = []pdbLabelRequirementModel{{
		Key: types.StringValue(""), Operator: types.StringValue(""), Values: types.SetNull(types.StringType),
	}}
	after.Spec[0].Selector[0].MatchExpressions = []pdbLabelRequirementModel{{
		Key: types.StringNull(), Operator: types.StringNull(), Values: types.SetNull(types.StringType),
	}}
	after.Spec[0].MaxUnavailable = types.StringNull()
	resp := pdbTestProtocolPlan(t, before, after)
	if len(resp.RequiresReplace) != 0 {
		t.Fatalf("legacy zero values caused replacement: %v", resp.RequiresReplace)
	}
	s := pdbTestSchema(t)
	raw, err := resp.PlannedState.Unmarshal(s.Type().TerraformType(context.Background()))
	if err != nil {
		t.Fatal(err)
	}
	var planned podDisruptionBudgetV1Model
	state := tfsdk.State{Schema: s, Raw: raw}
	pdbNoErrors(t, state.Get(context.Background(), &planned))
	for _, value := range []types.String{
		planned.Spec[0].MaxUnavailable, planned.Spec[0].Selector[0].MatchExpressions[0].Key,
		planned.Spec[0].Selector[0].MatchExpressions[0].Operator,
	} {
		if !value.Equal(types.StringValue("")) {
			t.Fatalf("legacy default must be known empty, got %s", value)
		}
	}
}

func TestPDBUnknownBlocks(t *testing.T) {
	ctx := context.Background()
	s := pdbTestSchema(t)
	for _, attributePath := range []path.Path{
		path.Root("metadata"),
		path.Root("spec"),
		path.Root("spec").AtListIndex(0).AtName("selector"),
		path.Root("spec").AtListIndex(0).AtName("selector").AtListIndex(0).AtName("match_expressions"),
	} {
		t.Run(attributePath.String(), func(t *testing.T) {
			state := pdbTestState(t, pdbTestConfig(pdbTestModel()))
			typ, d := s.TypeAtPath(ctx, attributePath)
			pdbNoErrors(t, d)
			listType := typ.(types.ListType)
			pdbNoErrors(t, state.SetAttribute(ctx, attributePath, types.ListUnknown(listType.ElemType)))
			server := providerserver.NewProtocol6(pdbProtocolProvider{})()
			validation, err := server.ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{
				TypeName: "kubernetes_pod_disruption_budget_v1", Config: pdbTestDynamic(t, state),
			})
			if err != nil {
				t.Fatal(err)
			}
			pdbProtocolNoErrors(t, validation.Diagnostics)
			r := &PodDisruptionBudgetV1{}
			create := resource.CreateResponse{State: tfsdk.State{Schema: s}}
			r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: s, Raw: state.Raw}}, &create)
			if !create.Diagnostics.HasError() {
				t.Fatal("unknown block at apply must diagnose, not be silently omitted")
			}
		})
	}
}

func TestPDBModelBoundaries(t *testing.T) {
	ctx := context.Background()
	for _, empty := range []bool{false, true} {
		model := pdbTestModel()
		if empty {
			model.Spec[0].Selector[0].MatchLabels = types.MapValueMust(types.StringType, map[string]attr.Value{})
		}
		spec, d := expandPodDisruptionBudgetSpec(ctx, model.Spec)
		pdbNoErrors(t, d)
		if spec.Selector == nil || spec.MinAvailable == nil || spec.MinAvailable.IntVal != 1 || spec.MaxUnavailable != nil {
			t.Fatalf("unexpected spec: %#v", spec)
		}
		flattened, d := flattenPodDisruptionBudgetSpec(ctx, spec, model.Spec)
		pdbNoErrors(t, d)
		if !flattened[0].Selector[0].MatchLabels.Equal(model.Spec[0].Selector[0].MatchLabels) {
			t.Fatal("lost map null/empty distinction")
		}
	}
	one := intstr.FromInt32(1)
	if got := flattenPDBThreshold(&one, types.StringValue("+01")); got.ValueString() != "+01" {
		t.Fatalf("lost equivalent known integer spelling: %s", got)
	}
	if got := flattenPDBThreshold(&one, types.StringValue("1%")); got.ValueString() != "1" {
		t.Fatalf("integer and percentage are not equivalent: %s", got)
	}
	model := pdbTestModel()
	model.Spec[0].MinAvailable = types.StringUnknown()
	_, d := expandPodDisruptionBudgetSpec(ctx, model.Spec)
	if !d.HasError() {
		t.Fatal("unknown threshold silently omitted")
	}
	_, d = expandPodDisruptionBudgetSpec(ctx, nil)
	if !d.HasError() {
		t.Fatal("empty spec accepted")
	}
	for _, value := range []types.Set{types.SetNull(types.StringType), types.SetValueMust(types.StringType, []attr.Value{})} {
		model = pdbTestModel()
		model.Spec[0].Selector[0].MatchExpressions = []pdbLabelRequirementModel{{
			Key: types.StringValue("app"), Operator: types.StringValue("Exists"), Values: value,
		}}
		spec, d := expandPodDisruptionBudgetSpec(ctx, model.Spec)
		pdbNoErrors(t, d)
		flattened, d := flattenPodDisruptionBudgetSpec(ctx, spec, model.Spec)
		pdbNoErrors(t, d)
		if !flattened[0].Selector[0].MatchExpressions[0].Values.Equal(value) {
			t.Fatal("lost expression set null/empty distinction")
		}
	}
}

type pdbClientMeta struct {
	kubernetes.KubeClientsets
	client *clientset.Clientset
	err    error
}

func (m *pdbClientMeta) MainClientset() (*clientset.Clientset, error) { return m.client, m.err }
func (*pdbClientMeta) GetIgnoreAnnotations() []string                 { return []string{"^ignored/"} }
func (*pdbClientMeta) GetIgnoreLabels() []string                      { return []string{"^ignored/"} }

func pdbTestResource(t *testing.T, handler http.HandlerFunc) *PodDisruptionBudgetV1 {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := clientset.NewForConfig(&rest.Config{
		Host: server.URL,
		ContentConfig: rest.ContentConfig{
			ContentType: "application/json", AcceptContentTypes: "application/json",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return &PodDisruptionBudgetV1{SDKv2Meta: func() any { return &pdbClientMeta{client: client} }}
}

func TestPDBConfigureAndClientErrors(t *testing.T) {
	ctx := context.Background()
	r := &PodDisruptionBudgetV1{}
	for _, data := range []any{nil, "bad", (func() any)(nil)} {
		var resp resource.ConfigureResponse
		r.Configure(ctx, resource.ConfigureRequest{ProviderData: data}, &resp)
		if (data != nil) != resp.Diagnostics.HasError() {
			t.Fatalf("unexpected Configure diagnostics for %T: %s", data, resp.Diagnostics)
		}
	}
	calls := 0
	var configured resource.ConfigureResponse
	r.Configure(ctx, resource.ConfigureRequest{ProviderData: func() any { calls++; return nil }}, &configured)
	pdbNoErrors(t, configured.Diagnostics)
	if calls != 0 {
		t.Fatal("Configure must not resolve SDK metadata eagerly")
	}
	for _, meta := range []any{nil, (*pdbClientMeta)(nil), "wrong", &pdbClientMeta{}, &pdbClientMeta{err: errors.New("client failed")}, struct{ kubernetes.KubeClientsets }{}} {
		r.SDKv2Meta = func() any { return meta }
		_, _, d := r.client()
		if !d.HasError() {
			t.Fatalf("expected client diagnostic for %T", meta)
		}
	}
	r.SDKv2Meta = nil
	_, _, d := r.client()
	if !d.HasError() {
		t.Fatal("missing provider configuration must diagnose")
	}
}

func TestPDBImportID(t *testing.T) {
	ctx := context.Background()
	for _, id := range []string{"", "budget", "/budget", "test/", "test/budget/extra", " test/budget", "test/budget ", "test/budget"} {
		s := pdbTestSchema(t)
		resp := resource.ImportStateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}}
		(&PodDisruptionBudgetV1{}).ImportState(ctx, resource.ImportStateRequest{ID: id}, &resp)
		if (id != "test/budget") != resp.Diagnostics.HasError() {
			t.Fatalf("unexpected import diagnostics for %q: %s", id, resp.Diagnostics)
		}
		if id == "test/budget" {
			var actual types.String
			pdbNoErrors(t, resp.State.GetAttribute(ctx, path.Root("id"), &actual))
			if actual.ValueString() != id {
				t.Fatal(actual)
			}
		}
	}
}

func TestPDBLifecycleHTTP(t *testing.T) {
	ctx := context.Background()
	model := pdbTestModel()
	model.Spec[0].MinAvailable = types.StringValue("+01")
	var methods []string
	object := policy.PodDisruptionBudget{
		TypeMeta: metav1.TypeMeta{APIVersion: "policy/v1", Kind: "PodDisruptionBudget"},
		ObjectMeta: metav1.ObjectMeta{Name: "budget", Namespace: "test", UID: "uid", ResourceVersion: "2", Generation: 1,
			Labels: map[string]string{"ignored/controller": "keep"}},
		Spec: policy.PodDisruptionBudgetSpec{MinAvailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 1}, Selector: &metav1.LabelSelector{}},
	}
	r := pdbTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		methods = append(methods, req.Method)
		if !strings.HasPrefix(req.URL.Path, "/apis/policy/v1/namespaces/test/poddisruptionbudgets") {
			t.Errorf("unexpected URL: %s", req.URL)
		}
		if req.Method == http.MethodPost {
			var sent policy.PodDisruptionBudget
			if err := json.NewDecoder(req.Body).Decode(&sent); err != nil {
				t.Error(err)
			}
			if sent.Spec.MinAvailable == nil || sent.Spec.MinAvailable.Type != intstr.Int || sent.Spec.MinAvailable.IntVal != 1 || sent.Spec.Selector == nil {
				t.Errorf("unexpected request spec: %#v", sent.Spec)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(object); err != nil {
			t.Error(err)
		}
	})
	state := pdbTestState(t, model)
	create := resource.CreateResponse{State: tfsdk.State{Schema: state.Schema}}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: state.Schema, Raw: state.Raw}}, &create)
	pdbNoErrors(t, create.Diagnostics)
	if !reflect.DeepEqual(methods, []string{"POST"}) {
		t.Fatalf("Create made a fallible follow-up request: %v", methods)
	}
	var created podDisruptionBudgetV1Model
	pdbNoErrors(t, create.State.Get(ctx, &created))
	if created.ID.ValueString() != "test/budget" || !created.Metadata[0].Labels.IsNull() || created.Spec[0].MinAvailable.ValueString() != "+01" {
		t.Fatalf("Create did not preserve planned values: %#v", created)
	}
	read := resource.ReadResponse{State: create.State}
	r.Read(ctx, resource.ReadRequest{State: create.State}, &read)
	pdbNoErrors(t, read.Diagnostics)
	var refreshed podDisruptionBudgetV1Model
	pdbNoErrors(t, read.State.Get(ctx, &refreshed))
	if !refreshed.Metadata[0].Labels.IsNull() || refreshed.Spec[0].MinAvailable.ValueString() != "+01" {
		t.Fatal("Read changed managed empty maps or equivalent integer spelling")
	}
	del := resource.DeleteResponse{State: read.State}
	r.Delete(ctx, resource.DeleteRequest{State: read.State}, &del)
	pdbNoErrors(t, del.Diagnostics)
	if !reflect.DeepEqual(methods, []string{"POST", "GET", "DELETE"}) {
		t.Fatalf("unexpected lifecycle requests: %v", methods)
	}
}

func TestPDBGeneratedNameAndImportRead(t *testing.T) {
	ctx := context.Background()
	object := policy.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{Name: "budget-generated", GenerateName: "budget-", Namespace: "test", ResourceVersion: "2", UID: "uid"},
		Spec:       policy.PodDisruptionBudgetSpec{MaxUnavailable: &intstr.IntOrString{Type: intstr.String, StrVal: "50%"}, Selector: &metav1.LabelSelector{}},
	}
	r := pdbTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost {
			var sent policy.PodDisruptionBudget
			if err := json.NewDecoder(req.Body).Decode(&sent); err != nil {
				t.Error(err)
			}
			if sent.Name != "" || sent.GenerateName != "budget-" || sent.Namespace != "test" {
				t.Errorf("incorrect generated metadata: %#v", sent.ObjectMeta)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(object)
	})
	model := pdbTestModel()
	model.ID = types.StringUnknown()
	model.Metadata[0].Name = types.StringUnknown()
	model.Metadata[0].GenerateName = types.StringValue("budget-")
	model.Metadata[0].Generation = types.Int64Unknown()
	model.Metadata[0].ResourceVersion = types.StringUnknown()
	model.Metadata[0].UID = types.StringUnknown()
	model.Spec[0].MinAvailable = types.StringValue("")
	model.Spec[0].MaxUnavailable = types.StringValue("50%")
	plan := pdbTestState(t, model)
	create := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema}}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: plan.Schema, Raw: plan.Raw}}, &create)
	pdbNoErrors(t, create.Diagnostics)
	var created podDisruptionBudgetV1Model
	pdbNoErrors(t, create.State.Get(ctx, &created))
	if created.ID.ValueString() != "test/budget-generated" || created.Metadata[0].Name.ValueString() != "budget-generated" {
		t.Fatal("generated name was not persisted")
	}
	imported := resource.ImportStateResponse{State: tfsdk.State{
		Schema: plan.Schema, Raw: tftypes.NewValue(plan.Schema.Type().TerraformType(ctx), nil),
	}}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "test/budget-generated"}, &imported)
	pdbNoErrors(t, imported.Diagnostics)
	read := resource.ReadResponse{State: imported.State}
	r.Read(ctx, resource.ReadRequest{State: imported.State}, &read)
	pdbNoErrors(t, read.Diagnostics)
	var refreshed podDisruptionBudgetV1Model
	pdbNoErrors(t, read.State.Get(ctx, &refreshed))
	if refreshed.Spec[0].MinAvailable.ValueString() != "" || refreshed.Spec[0].MaxUnavailable.ValueString() != "50%" ||
		refreshed.Metadata[0].GenerateName.ValueString() != "budget-" || refreshed.Metadata[0].Namespace.ValueString() != "test" {
		t.Fatalf("incomplete imported state: %#v", refreshed)
	}
}

func TestPDBMetadataPatchOwnership(t *testing.T) {
	ctx := context.Background()
	model := pdbTestModel()
	before := model.Metadata[0].MetadataModel
	after := before
	after.Labels = types.MapValueMust(types.StringType, map[string]attr.Value{"managed/key~": types.StringValue("value")})
	ops, d := pdbMetadataPatchOps(ctx, metav1.ObjectMeta{
		ResourceVersion: "3", Labels: map[string]string{"controller": "keep"},
	}, before, after)
	pdbNoErrors(t, d)
	data, err := ops.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 2 || decoded[0]["op"] != "test" || decoded[0]["path"] != "/metadata/resourceVersion" ||
		decoded[1]["path"] != "/metadata/labels/managed~1key~0" {
		t.Fatalf("unsafe metadata patch: %s", data)
	}
	calls := 0
	r := pdbTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		calls++
		if req.Method == http.MethodPatch {
			if req.Header.Get("Content-Type") != "application/json-patch+json" {
				t.Errorf("wrong patch type: %s", req.Header.Get("Content-Type"))
			}
			var actual []map[string]any
			if err := json.NewDecoder(req.Body).Decode(&actual); err != nil {
				t.Error(err)
			}
			if !reflect.DeepEqual(actual, decoded) {
				t.Errorf("unexpected patch: %#v", actual)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(policy.PodDisruptionBudget{ObjectMeta: metav1.ObjectMeta{
			Name: "budget", Namespace: "test", ResourceVersion: "3", UID: "uid",
			Labels: map[string]string{"controller": "keep", "ignored/extra": "keep"},
		}})
	})
	model.Metadata[0].Labels = types.MapNull(types.StringType)
	state := pdbTestState(t, model)
	model.Metadata[0].MetadataModel = after
	plan := pdbTestState(t, model)
	resp := resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{State: state, Plan: tfsdk.Plan{Schema: plan.Schema, Raw: plan.Raw}}, &resp)
	pdbNoErrors(t, resp.Diagnostics)
	if calls != 2 {
		t.Fatalf("expected GET and PATCH, got %d requests", calls)
	}
}

func TestPDBUpdatePatchRetry(t *testing.T) {
	ctx := context.Background()
	live := policy.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{Name: "budget", Namespace: "test", ResourceVersion: "1", UID: "uid"},
		Spec:       policy.PodDisruptionBudgetSpec{MinAvailable: &intstr.IntOrString{IntVal: 1}, Selector: &metav1.LabelSelector{}},
		Status:     policy.PodDisruptionBudgetStatus{DisruptionsAllowed: 2},
	}
	gets, patches := 0, 0
	var mu sync.Mutex
	r := pdbTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if req.Method == http.MethodGet {
			gets++
			_ = json.NewEncoder(w).Encode(live)
			return
		}
		if req.Method != http.MethodPatch {
			t.Errorf("unexpected request: %s", req.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		patches++
		var ops []map[string]any
		if err := json.NewDecoder(req.Body).Decode(&ops); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if len(ops) != 3 || ops[0]["op"] != "test" || ops[0]["path"] != "/metadata/resourceVersion" {
			t.Errorf("unsafe patch: %#v", ops)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if patches == 1 {
			live.ResourceVersion = "2"
			live.Labels = map[string]string{"controller": "keep"}
			live.Annotations = map[string]string{"controller": "keep"}
			live.Status.DisruptionsAllowed = 3
		}
		if ops[0]["value"] != live.ResourceVersion {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_ = json.NewEncoder(w).Encode(metav1.Status{
				TypeMeta: metav1.TypeMeta{Kind: "Status", APIVersion: "v1"},
				Status:   metav1.StatusFailure, Reason: metav1.StatusReasonInvalid, Code: 422,
				Message: "the server rejected our request due to an error in our request",
			})
			return
		}
		for _, op := range ops[1:] {
			if op["op"] != "add" || op["value"] != "new" {
				t.Errorf("unexpected metadata edit: %#v", op)
			}
			switch op["path"] {
			case "/metadata/labels/user":
				live.Labels["user"] = "new"
			case "/metadata/annotations/user":
				live.Annotations["user"] = "new"
			default:
				t.Errorf("retry must rebuild patch without replacing controller-owned maps: %#v", op)
			}
		}
		live.ResourceVersion = "3"
		_ = json.NewEncoder(w).Encode(live)
	})
	model := pdbTestModel()
	state := pdbTestState(t, model)
	model.Metadata[0].Labels = types.MapValueMust(types.StringType, map[string]attr.Value{"user": types.StringValue("new")})
	model.Metadata[0].Annotations = model.Metadata[0].Labels
	plan := pdbTestState(t, model)
	resp := resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{State: state, Plan: tfsdk.Plan{Schema: plan.Schema, Raw: plan.Raw}}, &resp)
	pdbNoErrors(t, resp.Diagnostics)
	mu.Lock()
	defer mu.Unlock()
	if patches != 2 || gets != 3 {
		t.Fatalf("expected one stale-patch retry with refetch, got %d GETs and %d PATCHes", gets, patches)
	}
	expected := map[string]string{"controller": "keep", "user": "new"}
	if !reflect.DeepEqual(live.Labels, expected) || !reflect.DeepEqual(live.Annotations, expected) {
		t.Fatalf("controller or user metadata lost: labels=%v annotations=%v", live.Labels, live.Annotations)
	}
	if live.Spec.MinAvailable.IntVal != 1 || live.Status.DisruptionsAllowed != 3 {
		t.Fatal("metadata patch changed spec or status")
	}
	var updated podDisruptionBudgetV1Model
	pdbNoErrors(t, resp.State.Get(ctx, &updated))
	if updated.Metadata[0].ResourceVersion.ValueString() != "3" ||
		!updated.Metadata[0].Labels.Equal(model.Metadata[0].Labels) || !updated.Metadata[0].Annotations.Equal(model.Metadata[0].Annotations) {
		t.Fatalf("successful retry did not preserve planned metadata: %#v", updated.Metadata[0])
	}
}

func TestPDBUpdatePatchError(t *testing.T) {
	for _, tc := range []struct {
		name           string
		status         int
		reason         metav1.StatusReason
		changeRevision bool
		invalidField   bool
		refetchFails   bool
		wantGets       int
		wantPatches    int
	}{
		{name: "conflict exhausted", status: 409, reason: metav1.StatusReasonConflict, changeRevision: true, wantGets: retry.DefaultRetry.Steps, wantPatches: retry.DefaultRetry.Steps},
		{name: "stale test exhausted", status: 422, reason: metav1.StatusReasonInvalid, changeRevision: true, wantGets: retry.DefaultRetry.Steps * 2, wantPatches: retry.DefaultRetry.Steps},
		{name: "unchanged revision", status: 422, reason: metav1.StatusReasonInvalid, wantGets: 2, wantPatches: 1},
		{name: "forbidden", status: 403, reason: metav1.StatusReasonForbidden, changeRevision: true, wantGets: 1, wantPatches: 1},
		{name: "bad request", status: 400, reason: metav1.StatusReasonBadRequest, changeRevision: true, wantGets: 1, wantPatches: 1},
		{name: "invalid metadata with concurrent change", status: 422, reason: metav1.StatusReasonInvalid, changeRevision: true, invalidField: true, wantGets: 1, wantPatches: 1},
		{name: "refetch forbidden", status: 422, reason: metav1.StatusReasonInvalid, changeRevision: true, refetchFails: true, wantGets: 2, wantPatches: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			gets, patches, revision := 0, 0, 1
			var mu sync.Mutex
			r := pdbTestResource(t, func(w http.ResponseWriter, req *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if req.Method == http.MethodPatch {
					patches++
					if tc.changeRevision {
						revision++
					}
					status := metav1.Status{
						TypeMeta: metav1.TypeMeta{Kind: "Status", APIVersion: "v1"},
						Status:   metav1.StatusFailure, Reason: tc.reason, Code: int32(tc.status),
						Message: "the server rejected our request due to an error in our request",
					}
					if tc.invalidField {
						status.Message = `PodDisruptionBudget "budget" is invalid: metadata.labels: Invalid value`
						status.Details = &metav1.StatusDetails{Causes: []metav1.StatusCause{{
							Type: metav1.CauseTypeFieldValueInvalid, Field: "metadata.labels",
						}}}
					}
					w.WriteHeader(tc.status)
					_ = json.NewEncoder(w).Encode(status)
					return
				}
				gets++
				if tc.refetchFails && gets > 1 {
					w.WriteHeader(http.StatusForbidden)
					_ = json.NewEncoder(w).Encode(metav1.Status{
						TypeMeta: metav1.TypeMeta{Kind: "Status", APIVersion: "v1"},
						Status:   metav1.StatusFailure, Reason: metav1.StatusReasonForbidden, Code: 403, Message: "refetch denied",
					})
					return
				}
				_ = json.NewEncoder(w).Encode(policy.PodDisruptionBudget{ObjectMeta: metav1.ObjectMeta{
					Name: "budget", Namespace: "test", ResourceVersion: strconv.Itoa(revision), UID: "uid",
				}})
			})
			model := pdbTestModel()
			state := pdbTestState(t, model)
			model.Metadata[0].Labels = types.MapValueMust(types.StringType, map[string]attr.Value{"app": types.StringValue("new")})
			plan := pdbTestState(t, model)
			resp := resource.UpdateResponse{State: state}
			r.Update(ctx, resource.UpdateRequest{State: state, Plan: tfsdk.Plan{Schema: plan.Schema, Raw: plan.Raw}}, &resp)
			mu.Lock()
			defer mu.Unlock()
			if !resp.Diagnostics.HasError() || !resp.State.Raw.Equal(state.Raw) {
				t.Fatalf("PATCH failure must retain state with diagnostics: %s", resp.Diagnostics)
			}
			if gets != tc.wantGets || patches != tc.wantPatches {
				t.Fatalf("expected %d GETs and %d PATCHes, got %d and %d", tc.wantGets, tc.wantPatches, gets, patches)
			}
			if tc.refetchFails && !strings.Contains(resp.Diagnostics[0].Detail(), "refetch denied") {
				t.Fatalf("refetch error not reported: %s", resp.Diagnostics)
			}
			if tc.wantPatches > 1 && !strings.Contains(resp.Diagnostics[0].Detail(), "the server rejected our request") {
				t.Fatalf("retry exhaustion lost the API error: %s", resp.Diagnostics)
			}
		})
	}
}

func TestPDBHTTPFailures(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusForbidden, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			ctx := context.Background()
			r := pdbTestResource(t, func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				reason := metav1.StatusReasonForbidden
				if status == http.StatusNotFound {
					reason = metav1.StatusReasonNotFound
				}
				_ = json.NewEncoder(w).Encode(metav1.Status{
					TypeMeta: metav1.TypeMeta{Kind: "Status", APIVersion: "v1"},
					Status:   metav1.StatusFailure, Reason: reason, Code: int32(status), Message: "test failure",
				})
			})
			state := pdbTestState(t, pdbTestModel())
			read := resource.ReadResponse{State: state}
			r.Read(ctx, resource.ReadRequest{State: state}, &read)
			if (status != http.StatusNotFound) != read.Diagnostics.HasError() || read.State.Raw.IsNull() != (status == http.StatusNotFound) {
				t.Fatalf("incorrect Read failure handling: %s, null=%t", read.Diagnostics, read.State.Raw.IsNull())
			}
			del := resource.DeleteResponse{State: state}
			r.Delete(ctx, resource.DeleteRequest{State: state}, &del)
			if (status != http.StatusNotFound) != del.Diagnostics.HasError() {
				t.Fatalf("incorrect Delete failure handling: %s", del.Diagnostics)
			}
			create := resource.CreateResponse{State: tfsdk.State{Schema: state.Schema}}
			r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: state.Schema, Raw: state.Raw}}, &create)
			if !create.Diagnostics.HasError() {
				t.Fatal("Create suppressed an API error")
			}
			update := resource.UpdateResponse{State: state}
			r.Update(ctx, resource.UpdateRequest{State: state, Plan: tfsdk.Plan{Schema: state.Schema, Raw: state.Raw}}, &update)
			if !update.Diagnostics.HasError() || update.State.Raw.IsNull() {
				t.Fatal("Update suppressed an API error or removed state")
			}
		})
	}
}
