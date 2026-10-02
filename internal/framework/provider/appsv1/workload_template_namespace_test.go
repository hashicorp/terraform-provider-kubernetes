// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWorkloadTemplateNamespacePlan(t *testing.T) {
	for name, block := range map[string]schema.ListNestedBlock{
		"deployment": templateMetadataBlock(),
		"daemonset":  daemonSetTemplateMetadataSchema(),
	} {
		t.Run(name, func(t *testing.T) {
			testWorkloadTemplateNamespacePlan(t, block.NestedObject.Attributes["namespace"].(schema.StringAttribute))
		})
	}
}

func testWorkloadTemplateNamespacePlan(t *testing.T, attribute schema.StringAttribute) {
	for _, tc := range []struct {
		name        string
		config      types.String
		prior       types.String
		plan        types.String
		want        types.String
		wantReplace bool
		destroy     bool
	}{
		{name: "fresh-omission", config: types.StringNull(), prior: types.StringNull(), plan: types.StringUnknown(), want: types.StringNull()},
		{name: "legacy-empty", config: types.StringNull(), prior: types.StringValue(""), plan: types.StringUnknown(), want: types.StringValue("")},
		{name: "remove-nonempty", config: types.StringNull(), prior: types.StringValue("old"), plan: types.StringUnknown(), want: types.StringNull(), wantReplace: true},
		{name: "explicit-empty", config: types.StringValue(""), prior: types.StringValue(""), plan: types.StringValue(""), want: types.StringValue("")},
		{name: "configured-value", config: types.StringValue("new"), prior: types.StringValue(""), plan: types.StringValue("new"), want: types.StringValue("new"), wantReplace: true},
		{name: "destroy", config: types.StringNull(), prior: types.StringValue(""), plan: types.StringNull(), want: types.StringNull(), destroy: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := tftypes.NewValue(tftypes.Bool, true)
			plan := raw
			if tc.destroy {
				plan = tftypes.NewValue(tftypes.Bool, nil)
			}
			request := planmodifier.StringRequest{
				ConfigValue: tc.config, StateValue: tc.prior, PlanValue: tc.plan,
				Plan: tfsdk.Plan{Raw: plan}, State: tfsdk.State{Raw: raw},
			}
			response := planmodifier.StringResponse{PlanValue: tc.plan}
			for _, modifier := range attribute.PlanModifiers {
				modifier.PlanModifyString(context.Background(), request, &response)
				request.PlanValue = response.PlanValue
			}
			if !response.PlanValue.Equal(tc.want) || response.RequiresReplace != tc.wantReplace {
				t.Fatalf("got namespace=%s replacement=%t; want %s replacement=%t", response.PlanValue, response.RequiresReplace, tc.want, tc.wantReplace)
			}
		})
	}
}

func TestWorkloadTemplateNamespaceRead(t *testing.T) {
	for _, tc := range []struct {
		name  string
		api   string
		prior types.String
		want  types.String
	}{
		{name: "fresh-omission", prior: types.StringNull(), want: types.StringNull()},
		{name: "legacy-empty", prior: types.StringValue(""), want: types.StringValue("")},
		{name: "unknown-prior", prior: types.StringUnknown(), want: types.StringNull()},
		{name: "removed-nonempty", prior: types.StringValue("old"), want: types.StringNull()},
		{name: "new-api-value", api: "new", prior: types.StringValue(""), want: types.StringValue("new")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			metadata, diagnostics := flattenTemplateMetadata(context.Background(), metav1.ObjectMeta{Namespace: tc.api},
				[]common.NamespacedMetadataModel{{Namespace: tc.prior}})
			if diagnostics.HasError() {
				t.Fatal(diagnostics)
			}
			if !metadata[0].Namespace.Equal(tc.want) {
				t.Errorf("deployment: got %s, want %s", metadata[0].Namespace, tc.want)
			}
			metadata, diagnostics = flattenDaemonSetTemplateMetadata(context.Background(), metav1.ObjectMeta{Namespace: tc.api},
				[]common.NamespacedMetadataModel{{Namespace: tc.prior}})
			if diagnostics.HasError() {
				t.Fatal(diagnostics)
			}
			if !metadata[0].Namespace.Equal(tc.want) {
				t.Errorf("daemonset: got %s, want %s", metadata[0].Namespace, tc.want)
			}
		})
	}
}

func TestWorkloadTemplateNamespaceDefaultAndUnknown(t *testing.T) {
	var defaultResponse defaults.StringResponse
	workloadTemplateNamespace{}.DefaultString(context.Background(), defaults.StringRequest{}, &defaultResponse)
	if !defaultResponse.PlanValue.IsNull() {
		t.Fatalf("fresh omissions must default to null, got %s", defaultResponse.PlanValue)
	}
	response := planmodifier.StringResponse{PlanValue: types.StringUnknown()}
	workloadTemplateNamespace{}.PlanModifyString(context.Background(), planmodifier.StringRequest{
		ConfigValue: types.StringUnknown(), StateValue: types.StringValue(""), PlanValue: types.StringUnknown(),
		Plan: tfsdk.Plan{Raw: tftypes.NewValue(tftypes.Bool, true)},
	}, &response)
	if !response.PlanValue.IsUnknown() {
		t.Fatalf("unknown configuration must stay unknown, got %s", response.PlanValue)
	}
}
