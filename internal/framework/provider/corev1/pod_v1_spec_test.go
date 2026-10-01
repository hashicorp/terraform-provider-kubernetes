// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	legacyschema "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	api "k8s.io/api/core/v1"
	kquantity "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

func TestPodV1SpecTargetStateLegacyValidation(t *testing.T) {
	legacy := &legacyschema.Resource{Schema: map[string]*legacyschema.Schema{
		"target_state": kubernetes.Provider().ResourcesMap["kubernetes_pod"].Schema["target_state"],
	}}
	for _, test := range []struct {
		name      string
		config    map[string]interface{}
		wantError bool
	}{
		{"omitted", map[string]interface{}{}, false},
		{"empty configured", map[string]interface{}{"target_state": []interface{}{}}, true},
		{"valid configured", map[string]interface{}{"target_state": []interface{}{"Succeeded", "Failed"}}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			diagnostics := legacy.Validate(terraform.NewResourceConfigRaw(test.config))
			if diagnostics.HasError() != test.wantError {
				t.Fatalf("legacy validation: %v; want error %v", diagnostics, test.wantError)
			}
		})
	}
}

func TestPodV1SpecSchema(t *testing.T) {
	ctx := context.Background()
	var response resource.SchemaResponse
	(&PodV1{}).Schema(ctx, resource.SchemaRequest{}, &response)
	if response.Schema.Version != 1 {
		t.Fatalf("schema version = %d, want 1", response.Schema.Version)
	}
	if diagnostics := response.Schema.ValidateImplementation(ctx); diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	paths := map[string]string{}
	podSpecTestPaths(types.ListType{ElemType: podSpecObject().Type()}, "spec", paths)
	// SDKv2's implicit-string maps omit their element from the inventory;
	// Framework explicitly types those additional 15 map-element paths.
	if len(paths) != 758 {
		t.Fatalf("native PodSpec has %d typed paths, want 758 (743 inventoried plus 15 implicit map elements)", len(paths))
	}
	// An opt-in comparison against a captured legacy inventory makes local
	// migrations check every path without making CI depend on machine artifacts.
	if inventory := os.Getenv("TF_POD_SCHEMA_INVENTORY"); inventory != "" {
		data, err := os.ReadFile(inventory)
		if err != nil {
			t.Fatal(err)
		}
		var captured struct {
			Fields []struct {
				Path string `json:"path"`
				Type string `json:"type"`
			} `json:"fields"`
		}
		if err := json.Unmarshal(data, &captured); err != nil {
			t.Fatal(err)
		}
		for _, field := range captured.Fields {
			if strings.HasPrefix(field.Path, "spec") && paths[field.Path] != field.Type {
				t.Errorf("%s: native %q, SDKv2 %q", field.Path, paths[field.Path], field.Type)
			}
		}
	}
	for _, name := range []string{"container", "init_container"} {
		container := podSpecObject().Blocks[name].(schema.ListNestedBlock)
		if _, ok := container.NestedObject.Attributes["resources"].(schema.ListNestedAttribute); !ok {
			t.Errorf("%s.resources must remain a list nested attribute", name)
		}
		if _, ok := container.NestedObject.Blocks["resources"]; ok {
			t.Errorf("%s.resources cannot remain a noncomputed block", name)
		}
	}
	for _, name := range []string{"image_pull_secrets", "readiness_gate"} {
		attribute, ok := podSpecObject().Attributes[name].(schema.ListAttribute)
		if !ok || !attribute.Optional || !attribute.Computed {
			t.Errorf("%s must remain a plain optional-computed list attribute", name)
			continue
		}
		if len(attribute.Validators) == 0 {
			t.Errorf("%s must explicitly validate required children", name)
		}
	}
}

func TestPodV1SpecResourceCardinalityValidation(t *testing.T) {
	ctx := context.Background()
	for _, block := range []string{"container", "init_container"} {
		container := podSpecObject().Blocks[block].(schema.ListNestedBlock)
		resources := container.NestedObject.Attributes["resources"].(schema.ListNestedAttribute)
		elementType := resources.NestedObject.Type().(types.ObjectType)
		object := types.ObjectValueMust(elementType.AttrTypes, map[string]attr.Value{
			"limits":   types.MapNull(types.StringType),
			"requests": types.MapNull(types.StringType),
		})
		for _, test := range []struct {
			name      string
			value     types.List
			wantError bool
		}{
			{"null", types.ListNull(elementType), false},
			{"unknown", types.ListUnknown(elementType), false},
			{"singleton object", types.ListValueMust(elementType, []attr.Value{object}), false},
			{"empty list", types.ListValueMust(elementType, []attr.Value{}), true},
			{"multiple objects", types.ListValueMust(elementType, []attr.Value{object, object}), true},
		} {
			t.Run(block+"/"+test.name, func(t *testing.T) {
				var response validator.ListResponse
				for _, check := range resources.Validators {
					check.ValidateList(ctx, validator.ListRequest{
						Path:        path.Root("spec").AtListIndex(0).AtName(block).AtListIndex(0).AtName("resources"),
						ConfigValue: test.value,
					}, &response)
				}
				if response.Diagnostics.HasError() != test.wantError {
					t.Fatalf("validation: %v; want error %v", response.Diagnostics, test.wantError)
				}
			})
		}
	}
}

func TestPodV1SpecReferenceCardinalityValidation(t *testing.T) {
	ctx := context.Background()
	for name, child := range map[string]string{"image_pull_secrets": "name", "readiness_gate": "condition_type"} {
		references := podSpecObject().Attributes[name].(schema.ListAttribute)
		elementType := references.ElementType.(types.ObjectType)
		object := types.ObjectValueMust(elementType.AttrTypes, map[string]attr.Value{
			child: types.StringValue("reference"),
		})
		for _, test := range []struct {
			name      string
			value     types.List
			wantError bool
		}{
			{"null", types.ListNull(elementType), false},
			{"unknown", types.ListUnknown(elementType), false},
			{"singleton", types.ListValueMust(elementType, []attr.Value{object}), false},
			{"empty", types.ListValueMust(elementType, []attr.Value{}), true},
			{"multiple", types.ListValueMust(elementType, []attr.Value{object, object}), false},
		} {
			t.Run(name+"/"+test.name, func(t *testing.T) {
				var response validator.ListResponse
				for _, check := range references.Validators {
					check.ValidateList(ctx, validator.ListRequest{
						Path:        path.Root("spec").AtListIndex(0).AtName(name),
						ConfigValue: test.value,
					}, &response)
				}
				if response.Diagnostics.HasError() != test.wantError {
					t.Fatalf("validation: %v; want error %v", response.Diagnostics, test.wantError)
				}
			})
		}
	}
}

func podSpecTestPaths(typ attr.Type, prefix string, paths map[string]string) {
	switch value := typ.(type) {
	case types.ObjectType:
		for name, child := range value.AttrTypes {
			podSpecTestPaths(child, prefix+"."+name, paths)
		}
	case types.ListType:
		paths[prefix] = "TypeList"
		podSpecTestPaths(value.ElemType, prefix+".*", paths)
	case types.SetType:
		paths[prefix] = "TypeSet"
		podSpecTestPaths(value.ElemType, prefix+".*", paths)
	case types.MapType:
		paths[prefix] = "TypeMap"
		podSpecTestPaths(value.ElemType, prefix+".*", paths)
	default:
		switch {
		case typ.Equal(types.StringType):
			paths[prefix] = "TypeString"
		case typ.Equal(types.BoolType):
			paths[prefix] = "TypeBool"
		case typ.Equal(types.Int64Type):
			paths[prefix] = "TypeInt"
		default:
			paths[prefix] = typ.String()
		}
	}
}

func TestPodV1SpecConversion(t *testing.T) {
	ctx := context.Background()
	in := api.PodSpec{
		Containers: []api.Container{{
			Name: "app", Image: "image:1", ImagePullPolicy: api.PullIfNotPresent,
			Resources: api.ResourceRequirements{
				Limits:   api.ResourceList{api.ResourceCPU: kquantity.MustParse("1")},
				Requests: api.ResourceList{api.ResourceMemory: kquantity.MustParse("64Mi")},
			},
			Env: []api.EnvVar{{Name: "LIMIT", ValueFrom: &api.EnvVarSource{
				ResourceFieldRef: &api.ResourceFieldSelector{Resource: "limits.cpu", Divisor: kquantity.MustParse("1")},
			}}},
			SecurityContext: &api.SecurityContext{RunAsNonRoot: ptr.To(true)},
		}},
		InitContainers: []api.Container{{Name: "init", Image: "image:init", Resources: api.ResourceRequirements{
			Requests: api.ResourceList{api.ResourceCPU: kquantity.MustParse("100m")},
		}}},
		ActiveDeadlineSeconds:        ptr.To(int64(120)),
		AutomountServiceAccountToken: ptr.To(true),
		EnableServiceLinks:           ptr.To(true), ShareProcessNamespace: ptr.To(false),
		TerminationGracePeriodSeconds: ptr.To(int64(30)),
		DNSPolicy:                     api.DNSClusterFirst, RestartPolicy: api.RestartPolicyAlways,
		ImagePullSecrets: []api.LocalObjectReference{{Name: "injected-secret"}},
		ReadinessGates:   []api.PodReadinessGate{{ConditionType: "example.com/Ready"}},
		SecurityContext:  &api.PodSecurityContext{SupplementalGroups: []int64{1000, 2000}},
		Affinity: &api.Affinity{NodeAffinity: &api.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &api.NodeSelector{NodeSelectorTerms: []api.NodeSelectorTerm{{
				MatchExpressions: []api.NodeSelectorRequirement{{Key: "zone", Operator: api.NodeSelectorOpIn, Values: []string{"a", "b"}}},
			}}},
		}},
		Volumes: []api.Volume{{Name: "scratch", VolumeSource: api.VolumeSource{EmptyDir: &api.EmptyDirVolumeSource{
			SizeLimit: ptr.To(kquantity.MustParse("1Gi")),
		}}}},
	}
	state, diagnostics := flattenPodV1Spec(ctx, in, types.ListNull(podSpecObject().Type()))
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	expanded, diagnostics := expandPodV1Spec(ctx, state)
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	if len(expanded.ImagePullSecrets) != 1 || expanded.ImagePullSecrets[0].Name != "injected-secret" {
		t.Fatal("service-account-injected image pull secrets were discarded")
	}
	if len(expanded.ReadinessGates) != 1 || len(expanded.InitContainers) != 1 {
		t.Fatal("computed readiness gates or init containers were discarded")
	}
	if expanded.Containers[0].Resources.Limits.Cpu().Cmp(kquantity.MustParse("1")) != 0 ||
		expanded.Containers[0].Resources.Requests.Memory().Cmp(kquantity.MustParse("64Mi")) != 0 {
		t.Fatal("resource quantities were not preserved")
	}
	slices.Sort(expanded.SecurityContext.SupplementalGroups)
	slices.Sort(expanded.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values)
	if !reflect.DeepEqual(expanded.SecurityContext.SupplementalGroups, in.SecurityContext.SupplementalGroups) ||
		!reflect.DeepEqual(expanded.Affinity, in.Affinity) {
		t.Fatalf("set-valued fields were not preserved: groups=%v affinity=%#v", expanded.SecurityContext.SupplementalGroups, expanded.Affinity)
	}
}

func TestPodV1SpecAdmissionFiltersAndQuantity(t *testing.T) {
	ctx := context.Background()
	in := api.PodSpec{
		Containers: []api.Container{{Name: "app", Image: "image:1",
			VolumeMounts: []api.VolumeMount{{Name: "kube-api-access-abcde", MountPath: "/var/run/secrets/kubernetes.io/serviceaccount"}},
			Resources:    api.ResourceRequirements{Limits: api.ResourceList{api.ResourceCPU: kquantity.MustParse("1")}},
		}},
		Volumes:     []api.Volume{{Name: "kube-api-access-abcde"}},
		Tolerations: []api.Toleration{{Key: api.TaintNodeNotReady, Operator: api.TolerationOpExists}},
	}
	before := in.DeepCopy()
	state, diagnostics := flattenPodV1Spec(ctx, in, types.ListNull(podSpecObject().Type()))
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	if !reflect.DeepEqual(in, *before) {
		t.Fatal("flatten mutated its API input")
	}
	expanded, diagnostics := expandPodV1Spec(ctx, state)
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	if len(expanded.Volumes) != 0 || len(expanded.Containers[0].VolumeMounts) != 0 || len(expanded.Tolerations) != 0 {
		t.Fatal("API-injected service account volume/mount or built-in toleration was retained")
	}
	object := state.Elements()[0].(types.Object)
	containers := object.Attributes()["container"].(types.List)
	container := containers.Elements()[0].(types.Object)
	resources := container.Attributes()["resources"].(types.List)
	resource := resources.Elements()[0].(types.Object)
	fields := resource.Attributes()
	fields["limits"] = types.MapValueMust(types.StringType, map[string]attr.Value{"cpu": types.StringValue("1000m")})
	resource = types.ObjectValueMust(resource.AttributeTypes(ctx), fields)
	fields = container.Attributes()
	fields["resources"] = types.ListValueMust(resources.ElementType(ctx), []attr.Value{resource})
	container = types.ObjectValueMust(container.AttributeTypes(ctx), fields)
	fields = object.Attributes()
	fields["container"] = types.ListValueMust(containers.ElementType(ctx), []attr.Value{container})
	prior := types.ListValueMust(state.ElementType(ctx), []attr.Value{types.ObjectValueMust(object.AttributeTypes(ctx), fields)})
	refreshed, diagnostics := flattenPodV1Spec(ctx, in, prior)
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	got := refreshed.Elements()[0].(types.Object).Attributes()["container"].(types.List).Elements()[0].(types.Object).
		Attributes()["resources"].(types.List).Elements()[0].(types.Object).Attributes()["limits"].(types.Map).
		Elements()["cpu"].(types.String).ValueString()
	if got != "1000m" {
		t.Fatalf("equivalent configured quantity = %q, want 1000m", got)
	}
	in.Containers[0].Resources.Limits[api.ResourceCPU] = kquantity.MustParse("2")
	drift, diagnostics := flattenPodV1Spec(ctx, in, prior)
	if diagnostics.HasError() || drift.Equal(prior) {
		t.Fatal("a genuinely different resource quantity was suppressed", diagnostics)
	}
}

func TestPodV1SpecRequiredReference(t *testing.T) {
	ctx := context.Background()
	for _, child := range []string{"name", "condition_type"} {
		elementType := types.ObjectType{AttrTypes: map[string]attr.Type{child: types.StringType}}
		for _, test := range []struct {
			name      string
			value     types.List
			wantError bool
		}{
			{"null", types.ListNull(elementType), false},
			{"unknown", types.ListUnknown(elementType), false},
			{"missing child", types.ListValueMust(elementType, []attr.Value{types.ObjectValueMust(elementType.AttrTypes, map[string]attr.Value{child: types.StringNull()})}), true},
			{"unknown child", types.ListValueMust(elementType, []attr.Value{types.ObjectValueMust(elementType.AttrTypes, map[string]attr.Value{child: types.StringUnknown()})}), false},
		} {
			t.Run(child+"/"+test.name, func(t *testing.T) {
				var response validator.ListResponse
				podRequiredReference{child: child}.ValidateList(ctx, validator.ListRequest{Path: path.Root("reference"), ConfigValue: test.value}, &response)
				if response.Diagnostics.HasError() != test.wantError {
					t.Fatal(response.Diagnostics)
				}
			})
		}
	}
}

func TestPodV1SpecCollectionReplacement(t *testing.T) {
	ctx := context.Background()
	objectType := types.ObjectType{AttrTypes: map[string]attr.Type{"value": types.StringType}}
	element := func(value string) attr.Value {
		return types.ObjectValueMust(objectType.AttrTypes, map[string]attr.Value{"value": types.StringValue(value)})
	}
	old := types.ListValueMust(objectType, []attr.Value{element("old")})
	for _, test := range []struct {
		name    string
		planned types.List
		replace bool
	}{
		{"child edit", types.ListValueMust(objectType, []attr.Value{element("new")}), false},
		{"added", types.ListValueMust(objectType, []attr.Value{element("old"), element("new")}), true},
		{"removed", types.ListValueMust(objectType, nil), true},
	} {
		t.Run(test.name, func(t *testing.T) {
			raw, err := old.ToTerraformValue(ctx)
			if err != nil {
				t.Fatal(err)
			}
			var response planmodifier.ListResponse
			podListStructureRequiresReplace{}.PlanModifyList(ctx, planmodifier.ListRequest{
				StateValue: old, PlanValue: test.planned,
				State: tfsdk.State{Raw: raw}, Plan: tfsdk.Plan{Raw: raw},
			}, &response)
			if response.RequiresReplace != test.replace {
				t.Fatalf("RequiresReplace = %t, want %t", response.RequiresReplace, test.replace)
			}
		})
	}
}

func TestPodV1SpecUnknownBoundary(t *testing.T) {
	ctx := context.Background()
	in := api.PodSpec{Containers: []api.Container{{Name: "app", Image: "image:1"}}}
	state, diagnostics := flattenPodV1Spec(ctx, in, types.ListNull(podSpecObject().Type()))
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	object := state.Elements()[0].(types.Object)
	attributes := object.Attributes()
	attributes["service_account_name"] = types.StringUnknown()
	attributes["image_pull_secrets"] = types.ListUnknown(attributes["image_pull_secrets"].(types.List).ElementType(ctx))
	plan := types.ListValueMust(state.ElementType(ctx), []attr.Value{types.ObjectValueMust(object.AttributeTypes(ctx), attributes)})
	if _, diagnostics := expandPodV1Spec(ctx, plan); diagnostics.HasError() {
		t.Fatal("API-computed unknowns must select API defaults", diagnostics)
	}
	attributes["container"] = types.ListUnknown(attributes["container"].(types.List).ElementType(ctx))
	plan = types.ListValueMust(state.ElementType(ctx), []attr.Value{types.ObjectValueMust(object.AttributeTypes(ctx), attributes)})
	if _, diagnostics := expandPodV1Spec(ctx, plan); !diagnostics.HasError() {
		t.Fatal("unknown configured blocks must not be sent to the API")
	}
}

func TestPodV1SpecEmptyCollectionOwnership(t *testing.T) {
	ctx := context.Background()
	in := api.PodSpec{Containers: []api.Container{{Name: "app", Image: "image:1"}}}
	prior, diagnostics := flattenPodV1Spec(ctx, in, types.ListNull(podSpecObject().Type()))
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	object := prior.Elements()[0].(types.Object)
	fields := object.Attributes()
	fields["node_selector"] = types.MapValueMust(types.StringType, map[string]attr.Value{})
	fields["image_pull_secrets"] = types.ListNull(fields["image_pull_secrets"].(types.List).ElementType(ctx))
	prior = types.ListValueMust(prior.ElementType(ctx), []attr.Value{types.ObjectValueMust(object.AttributeTypes(ctx), fields)})
	refreshed, diagnostics := flattenPodV1Spec(ctx, in, prior)
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	fields = refreshed.Elements()[0].(types.Object).Attributes()
	if fields["node_selector"].IsNull() || !fields["image_pull_secrets"].IsNull() {
		t.Fatal("null/empty collection ownership changed")
	}
}
