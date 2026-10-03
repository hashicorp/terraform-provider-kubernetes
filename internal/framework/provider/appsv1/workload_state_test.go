// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

func TestSetDaemonSetStateMatchesReflection(t *testing.T) {
	dsType := daemonSetSpecListType().ElemType.(types.ObjectType)
	dsPodSpec := listAttrType(dsType, "template", "spec")
	strategy := dsType.AttrTypes["strategy"].(types.ObjectType)
	ds := func(edit func(*DaemonSetV1SpecModel)) []DaemonSetV1SpecModel {
		spec := DaemonSetV1SpecModel{
			Strategy: types.ObjectNull(strategy.AttrTypes),
			Template: []workloadTemplateModel{{Spec: types.ListNull(dsPodSpec.ElemType)}},
		}
		edit(&spec)
		return []DaemonSetV1SpecModel{spec}
	}
	assertStateMatchesReflection(t, daemonSetFrozenSchema, setDaemonSetState, func(spec []DaemonSetV1SpecModel, t timeouts.Value) DaemonSetV1Model {
		return DaemonSetV1Model{ID: types.StringValue("ns/name"), Spec: spec, Timeouts: t}
	}, map[string][]DaemonSetV1SpecModel{
		"nil spec":               nil,
		"empty spec":             {},
		"null objects and lists": ds(func(*DaemonSetV1SpecModel) {}),
		"unknown objects and lists": ds(func(s *DaemonSetV1SpecModel) {
			s.Strategy = types.ObjectUnknown(strategy.AttrTypes)
			s.Template[0].Spec = types.ListUnknown(dsPodSpec.ElemType)
			s.MinReadySeconds = types.Int64Unknown()
		}),
		"empty objects and slices": ds(func(s *DaemonSetV1SpecModel) {
			s.Strategy = types.ObjectValueMust(strategy.AttrTypes, map[string]attr.Value{"type": types.StringNull(), "rolling_update": types.ObjectNull(daemonSetRollingUpdateObjectType().AttrTypes)})
			s.Selector = []DaemonSetLabelSelectorModel{}
			s.Template[0].Metadata = []common.NamespacedMetadataModel{}
		}),
		"nil template":   ds(func(s *DaemonSetV1SpecModel) { s.Template = nil }),
		"empty template": ds(func(s *DaemonSetV1SpecModel) { s.Template = []workloadTemplateModel{} }),
	})
}

func TestSetStatefulSetStateMatchesReflection(t *testing.T) {
	stsType := statefulSetSpecListType().ElemType.(types.ObjectType)
	stsPodSpec := listAttrType(stsType, "template", "spec")
	retention := stsType.AttrTypes["persistent_volume_claim_retention_policy"].(types.ObjectType)
	sts := func(edit func(*StatefulSetSpecModel)) []StatefulSetSpecModel {
		spec := StatefulSetSpecModel{
			PersistentVolumeClaimRetentionPolicy: types.ObjectNull(retention.AttrTypes),
			Template:                             []workloadTemplateModel{{Spec: types.ListNull(stsPodSpec.ElemType)}},
		}
		edit(&spec)
		return []StatefulSetSpecModel{spec}
	}
	assertStateMatchesReflection(t, statefulSetFrozenSchema, setStatefulSetState, func(spec []StatefulSetSpecModel, t timeouts.Value) StatefulSetV1Model {
		return StatefulSetV1Model{ID: types.StringValue("ns/name"), Spec: spec, Timeouts: t}
	}, map[string][]StatefulSetSpecModel{
		"nil spec":               nil,
		"null objects and lists": sts(func(*StatefulSetSpecModel) {}),
		"unknown objects and lists": sts(func(s *StatefulSetSpecModel) {
			s.PersistentVolumeClaimRetentionPolicy = types.ObjectUnknown(retention.AttrTypes)
			s.Template[0].Spec = types.ListUnknown(stsPodSpec.ElemType)
			s.Replicas = types.StringUnknown()
		}),
		"empty objects and slices": sts(func(s *StatefulSetSpecModel) {
			s.PersistentVolumeClaimRetentionPolicy = types.ObjectValueMust(retention.AttrTypes, map[string]attr.Value{"when_deleted": types.StringNull(), "when_scaled": types.StringNull()})
			s.UpdateStrategy = []StatefulSetUpdateStrategyModel{}
			s.VolumeClaimTemplate = []PersistentVolumeClaimModel{}
			s.Template = []workloadTemplateModel{}
		}),
	})
}

// assertStateMatchesReflection sets each spec through set and through
// tfsdk.State.Set, and requires the same outcome and state from both.
func assertStateMatchesReflection[M, S any](t *testing.T, schemaFunc common.SchemaFunc, set func(context.Context, *tfsdk.State, M) diag.Diagnostics, model func(S, timeouts.Value) M, specs map[string]S) {
	t.Helper()
	ctx := context.Background()
	var resp resource.SchemaResponse
	schemaFunc(ctx, resource.SchemaRequest{}, &resp)
	nullState := func() *tfsdk.State {
		return &tfsdk.State{Schema: resp.Schema, Raw: tftypes.NewValue(resp.Schema.Type().TerraformType(ctx), nil)}
	}
	timeoutsType := resp.Schema.Blocks["timeouts"].Type().(timeouts.Type)
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			m := model(spec, timeouts.Value{Object: types.ObjectNull(timeoutsType.AttrTypes)})
			want, got := nullState(), nullState()
			wantDiags := want.Set(ctx, &m)
			gotDiags := set(ctx, got, m)
			if gotDiags.HasError() != wantDiags.HasError() {
				t.Fatalf("errors = %v, reflection errors = %v", gotDiags, wantDiags)
			}
			if !got.Raw.Equal(want.Raw) {
				t.Fatalf("state = %s\nreflection state = %s", got.Raw, want.Raw)
			}
		})
	}
}

// listAttrType walks nested list blocks of typ by attribute name.
func listAttrType(typ types.ObjectType, names ...string) types.ListType {
	var list types.ListType
	for _, name := range names {
		list = typ.AttrTypes[name].(types.ListType)
		typ, _ = list.ElemType.(types.ObjectType)
	}
	return list
}

func TestDeploymentSpecModelsMatchReflection(t *testing.T) {
	ctx := context.Background()
	specType := deploymentSpecListType().ElemType.(types.ObjectType)
	templateType := specType.AttrTypes["template"].(types.ListType).ElemType.(types.ObjectType)
	podSpec := listAttrType(specType, "template", "spec").ElemType
	object := func(typ types.ObjectType, edit map[string]attr.Value) attr.Value {
		attrs := map[string]attr.Value{}
		for name, t := range typ.AttrTypes {
			attrs[name], _ = t.ValueFromTerraform(ctx, tftypes.NewValue(t.TerraformType(ctx), nil))
		}
		for name, v := range edit {
			attrs[name] = v
		}
		return types.ObjectValueMust(typ.AttrTypes, attrs)
	}
	spec := func(edit map[string]attr.Value) types.List {
		return types.ListValueMust(specType, []attr.Value{object(specType, edit)})
	}
	template := func(pod types.List) types.List {
		return types.ListValueMust(templateType, []attr.Value{object(templateType, map[string]attr.Value{"spec": pod})})
	}

	for name, value := range map[string]types.List{
		"null":             types.ListNull(specType),
		"unknown":          types.ListUnknown(specType),
		"empty":            types.ListValueMust(specType, nil),
		"null blocks":      spec(nil),
		"empty blocks":     spec(map[string]attr.Value{"selector": types.ListValueMust(specType.AttrTypes["selector"].(types.ListType).ElemType, nil), "template": types.ListValueMust(templateType, nil)}),
		"unknown template": spec(map[string]attr.Value{"template": types.ListUnknown(templateType)}),
		"unknown pod spec": spec(map[string]attr.Value{"template": template(types.ListUnknown(podSpec)), "replicas": types.StringUnknown()}),
		"unknown element":  types.ListValueMust(specType, []attr.Value{types.ObjectUnknown(specType.AttrTypes)}),
	} {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := deploymentSpecModels(ctx, value)
			var want []deploymentSpecModel
			wantDiags := value.ElementsAs(ctx, &want, false)
			if gotDiags.HasError() != wantDiags.HasError() {
				t.Fatalf("errors = %v, reflection errors = %v", gotDiags, wantDiags)
			}
			if !gotDiags.HasError() && !reflect.DeepEqual(got, want) {
				t.Fatalf("models = %#v\nreflection models = %#v", got, want)
			}
		})
	}
}
