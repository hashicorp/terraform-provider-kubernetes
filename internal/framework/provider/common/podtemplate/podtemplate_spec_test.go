// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package podtemplate

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	corev1 "k8s.io/api/core/v1"
	kquantity "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

var templatePath = path.Root("spec").AtListIndex(0).AtName("template").AtListIndex(0).AtName("spec")

func specType() types.ObjectType {
	return SpecBlock(Options{}).NestedObject.Type().(types.ObjectType)
}

func mustFlatten(t *testing.T, spec corev1.PodSpec, baseline types.List) types.List {
	t.Helper()
	value, diagnostics := FlattenSpec(context.Background(), spec, baseline, templatePath)
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	return value
}

func mustExpand(t *testing.T, value types.List) corev1.PodSpec {
	t.Helper()
	spec, diagnostics := ExpandSpec(context.Background(), value, templatePath)
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	return spec
}

func TestFlattenSpecDetectsRemovedAPICollections(t *testing.T) {
	original := fullTemplateSpec()
	prior := mustFlatten(t, original, types.ListNull(specType()))
	cases := []struct {
		name   string
		remove func(*corev1.PodSpec)
		steps  []any
	}{
		{"arguments", func(s *corev1.PodSpec) { s.Containers[0].Args = nil }, []any{0, "container", 0, "args"}},
		{"environment", func(s *corev1.PodSpec) { s.Containers[0].Env = nil }, []any{0, "container", 0, "env"}},
		{"node-selector", func(s *corev1.PodSpec) { s.NodeSelector = nil }, []any{0, "node_selector"}},
		{"image-pull-secrets", func(s *corev1.PodSpec) { s.ImagePullSecrets = nil }, []any{0, "image_pull_secrets"}},
		{"readiness-gates", func(s *corev1.PodSpec) { s.ReadinessGates = nil }, []any{0, "readiness_gate"}},
		{"resource-limits", func(s *corev1.PodSpec) { s.Containers[0].Resources.Limits = nil }, []any{0, "container", 0, "resources", 0, "limits"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			current := original.DeepCopy()
			tc.remove(current)
			refreshed := mustFlatten(t, *current, prior)
			before, after := get(prior, tc.steps...), get(refreshed, tc.steps...)
			if before.Equal(after) {
				t.Fatalf("removed API collection was retained in state: %s", after)
			}
			if after.IsUnknown() {
				t.Fatal("refresh returned an unknown collection")
			}
		})
	}
}

// get walks object attribute names and list indexes (ints) from a value.
func get(value attr.Value, steps ...any) attr.Value {
	for _, step := range steps {
		switch s := step.(type) {
		case int:
			value = value.(types.List).Elements()[s]
		case string:
			value = value.(types.Object).Attributes()[s]
		}
	}
	return value
}

// with returns value with the attribute at steps replaced.
func with(t *testing.T, value attr.Value, replacement attr.Value, steps ...any) attr.Value {
	t.Helper()
	if len(steps) == 0 {
		return replacement
	}
	switch s := steps[0].(type) {
	case int:
		list := value.(types.List)
		elements := list.Elements()
		elements[s] = with(t, elements[s], replacement, steps[1:]...)
		return types.ListValueMust(list.ElementType(context.Background()), elements)
	case string:
		object := value.(types.Object)
		attributes := object.Attributes()
		attributes[s] = with(t, attributes[s], replacement, steps[1:]...)
		return types.ObjectValueMust(object.AttributeTypes(context.Background()), attributes)
	}
	t.Fatalf("invalid step %v", steps[0])
	return nil
}

func emptyObject(objectType types.ObjectType) types.Object {
	attributes := map[string]attr.Value{}
	for name, typ := range objectType.AttrTypes {
		switch v := typ.(type) {
		case types.ListType:
			attributes[name] = types.ListValueMust(v.ElemType, nil)
		case types.SetType:
			attributes[name] = types.SetNull(v.ElemType)
		case types.MapType:
			attributes[name] = types.MapNull(v.ElemType)
		default:
			value, _ := typ.ValueFromTerraform(context.Background(), tftypes.NewValue(typ.TerraformType(context.Background()), nil))
			attributes[name] = value
		}
	}
	return types.ObjectValueMust(objectType.AttrTypes, attributes)
}

func fullTemplateSpec() corev1.PodSpec {
	return corev1.PodSpec{
		Containers: []corev1.Container{{
			Name: "app", Image: "nginx:1", ImagePullPolicy: corev1.PullIfNotPresent,
			Args: []string{"--flag"}, Command: []string{"nginx"},
			TerminationMessagePath: "/dev/termination-log", TerminationMessagePolicy: corev1.TerminationMessageReadFile,
			Resources: corev1.ResourceRequirements{
				Limits:   corev1.ResourceList{corev1.ResourceCPU: kquantity.MustParse("500m")},
				Requests: corev1.ResourceList{corev1.ResourceMemory: kquantity.MustParse("64Mi")},
			},
			Env: []corev1.EnvVar{{Name: "CPU", ValueFrom: &corev1.EnvVarSource{
				ResourceFieldRef: &corev1.ResourceFieldSelector{Resource: "limits.cpu", Divisor: kquantity.MustParse("1m")},
			}}},
			Ports:        []corev1.ContainerPort{{ContainerPort: 80, Protocol: corev1.ProtocolTCP}},
			VolumeMounts: []corev1.VolumeMount{{Name: "scratch", MountPath: "/scratch"}},
		}},
		InitContainers: []corev1.Container{{Name: "init", Image: "busybox", Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{corev1.ResourceCPU: kquantity.MustParse("100m")},
		}}},
		ActiveDeadlineSeconds:         ptr.To(int64(120)),
		AutomountServiceAccountToken:  ptr.To(true),
		EnableServiceLinks:            ptr.To(true),
		TerminationGracePeriodSeconds: ptr.To(int64(30)),
		DNSPolicy:                     corev1.DNSClusterFirst,
		RestartPolicy:                 corev1.RestartPolicyAlways,
		SchedulerName:                 "default-scheduler",
		ServiceAccountName:            "builder",
		NodeSelector:                  map[string]string{"disk": "ssd"},
		ImagePullSecrets:              []corev1.LocalObjectReference{{Name: "registry"}},
		ReadinessGates:                []corev1.PodReadinessGate{{ConditionType: "example.com/Ready"}},
		SecurityContext:               &corev1.PodSecurityContext{SupplementalGroups: []int64{1000}, RunAsUser: ptr.To(int64(1000)), RunAsNonRoot: ptr.To(false)},
		Tolerations: []corev1.Toleration{
			{Key: corev1.TaintNodeNotReady, Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoExecute, TolerationSeconds: ptr.To(int64(300))},
			{Key: "dedicated", Operator: corev1.TolerationOpEqual, Value: "apps", Effect: corev1.TaintEffectNoSchedule},
		},
		Affinity: &corev1.Affinity{NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{NodeSelectorTerms: []corev1.NodeSelectorTerm{{
				MatchExpressions: []corev1.NodeSelectorRequirement{{Key: "zone", Operator: corev1.NodeSelectorOpIn, Values: []string{"a"}}},
				MatchFields:      []corev1.NodeSelectorRequirement{{Key: "metadata.name", Operator: corev1.NodeSelectorOpIn, Values: []string{"node-1"}}},
			}}},
		}},
		Volumes: []corev1.Volume{
			{Name: "scratch", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{SizeLimit: ptr.To(kquantity.MustParse("1Gi"))}}},
			{Name: "share", VolumeSource: corev1.VolumeSource{AzureFile: &corev1.AzureFileVolumeSource{SecretName: "s", ShareName: "share"}}},
		},
	}
}

func TestSpecRoundTrip(t *testing.T) {
	in := fullTemplateSpec()
	before := in.DeepCopy()
	state := mustFlatten(t, in, types.ListNull(specType()))
	if !reflect.DeepEqual(in, *before) {
		t.Fatal("FlattenSpec mutated its API input")
	}
	if !state.ElementType(context.Background()).Equal(specType()) || len(state.Elements()) != 1 {
		t.Fatalf("unexpected flattened value %s", state)
	}
	out := mustExpand(t, state)
	if !sameResources(out.Containers[0].Resources, in.Containers[0].Resources) || len(out.InitContainers) != 1 ||
		out.InitContainers[0].Image != "busybox" || !sameResources(out.InitContainers[0].Resources, in.InitContainers[0].Resources) {
		t.Fatalf("container resources were not preserved: %#v", out.Containers[0].Resources)
	}
	for name, pair := range map[string][2]any{
		"tolerations":        {out.Tolerations, in.Tolerations},
		"affinity":           {out.Affinity, in.Affinity},
		"image pull secrets": {out.ImagePullSecrets, in.ImagePullSecrets},
		"readiness gates":    {out.ReadinessGates, in.ReadinessGates},
		"azure file":         {out.Volumes[1], in.Volumes[1]},
		"node selector":      {out.NodeSelector, in.NodeSelector},
		"security context":   {out.SecurityContext, in.SecurityContext},
		"ports":              {out.Containers[0].Ports, in.Containers[0].Ports},
		"service account":    {out.ServiceAccountName, in.ServiceAccountName},
		"restart policy":     {out.RestartPolicy, in.RestartPolicy},
	} {
		if !reflect.DeepEqual(pair[0], pair[1]) {
			t.Errorf("%s: got %#v, want %#v", name, pair[0], pair[1])
		}
	}
	if divisor := out.Containers[0].Env[0].ValueFrom.ResourceFieldRef.Divisor; divisor.Cmp(kquantity.MustParse("1m")) != 0 {
		t.Errorf("divisor: got %s", divisor.String())
	}
	if size := out.Volumes[0].EmptyDir.SizeLimit; size == nil || size.Cmp(kquantity.MustParse("1Gi")) != 0 {
		t.Errorf("size_limit: got %v", size)
	}
	// A second flatten from the first state is a fixed point (no drift on refresh).
	if again := mustFlatten(t, out, state); !again.Equal(state) {
		t.Fatalf("refresh drift:\n%s\n%s", again, state)
	}
}

// Templates keep the API's built-in tolerations, unlike the Pod resource.
func TestFlattenSpecKeepsTemplateTolerations(t *testing.T) {
	in := corev1.PodSpec{
		Containers:  []corev1.Container{{Name: "app", Image: "nginx"}},
		Tolerations: []corev1.Toleration{{Key: corev1.TaintNodeUnreachable, Operator: corev1.TolerationOpExists}},
	}
	state := mustFlatten(t, in, types.ListNull(specType()))
	tolerations := get(state, 0, "toleration").(types.List)
	if len(tolerations.Elements()) != 1 || get(tolerations, 0, "key").(types.String).ValueString() != corev1.TaintNodeUnreachable {
		t.Fatalf("built-in toleration was filtered from the template: %s", tolerations)
	}
	if out := mustExpand(t, state); len(out.Tolerations) != 1 {
		t.Fatalf("toleration not expanded: %#v", out.Tolerations)
	}
}

func TestExpandSpecNullUnknownEmpty(t *testing.T) {
	ctx := context.Background()
	for name, value := range map[string]types.List{
		"null":  types.ListNull(specType()),
		"empty": types.ListValueMust(specType(), nil),
	} {
		spec, diagnostics := ExpandSpec(ctx, value, templatePath)
		if diagnostics.HasError() || !reflect.DeepEqual(spec, corev1.PodSpec{}) {
			t.Errorf("%s: spec=%#v diagnostics=%v", name, spec, diagnostics)
		}
	}
	if _, diagnostics := ExpandSpec(ctx, types.ListUnknown(specType()), templatePath); !diagnostics.HasError() {
		t.Error("an unknown spec must not reach the API")
	}

	state := mustFlatten(t, corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "nginx"}}}, types.ListNull(specType()))
	plan := state
	if !podSpecComputedPathsFor()["spec.container.resources.limits"] || podSpecComputedPathsFor()["spec.container.image"] {
		t.Fatal("only API-computed paths without defaults may be unknown during apply")
	}
	for _, name := range []string{"service_account_name", "scheduler_name", "node_name", "hostname"} {
		plan = with(t, plan, types.StringUnknown(), 0, name).(types.List)
	}
	for _, name := range []string{"image_pull_secrets", "readiness_gate"} {
		plan = with(t, plan, types.ListUnknown(get(plan, 0, name).(types.List).ElementType(ctx)), 0, name).(types.List)
	}
	resources := get(plan, 0, "container", 0, "resources").(types.List)
	plan = with(t, plan, types.ListUnknown(resources.ElementType(ctx)), 0, "container", 0, "resources").(types.List)
	plan = with(t, plan, types.StringUnknown(), 0, "container", 0, "image_pull_policy").(types.List)
	spec := mustExpand(t, plan)
	if spec.ServiceAccountName != "" || len(spec.ImagePullSecrets) != 0 || spec.Containers[0].Resources.Limits != nil || spec.Containers[0].ImagePullPolicy != "" {
		t.Fatalf("unknown API-computed values must select API defaults: %#v", spec)
	}

	for name, unknown := range map[string][]any{
		"container": {0, "container"},
		"image":     {0, "container", 0, "image"},
		"args":      {0, "container", 0, "args"},
	} {
		var replacement attr.Value
		switch v := get(state, unknown...).(type) {
		case types.List:
			replacement = types.ListUnknown(v.ElementType(ctx))
		case types.String:
			replacement = types.StringUnknown()
		}
		_, diagnostics := ExpandSpec(ctx, with(t, state, replacement, unknown...).(types.List), templatePath)
		if !diagnostics.HasError() {
			t.Errorf("%s: unknown configured value must not reach the API", name)
			continue
		}
		if p := diagnostics.Errors()[0].(interface{ Path() path.Path }).Path(); !p.ParentPath().Equal(templatePath) && !hasPrefix(p, templatePath) {
			t.Errorf("%s: diagnostic path %s is not under %s", name, p, templatePath)
		}
	}
}

func TestExpandSpecRejectsUnknownCollectionElements(t *testing.T) {
	ctx := context.Background()
	baseline := mustFlatten(t, corev1.PodSpec{
		Containers: []corev1.Container{{Name: "app", Image: "nginx"}},
	}, types.ListNull(specType()))
	referenceType := get(baseline, 0, "image_pull_secrets").(types.List).ElementType(ctx).(types.ObjectType)
	for name, testCase := range map[string]struct {
		value attr.Value
		steps []any
	}{
		"reference": {
			value: types.ListValueMust(referenceType, []attr.Value{types.ObjectUnknown(referenceType.AttrTypes)}),
			steps: []any{0, "image_pull_secrets"},
		},
		"quantity": {
			value: types.MapValueMust(types.StringType, map[string]attr.Value{"cpu": types.StringUnknown()}),
			steps: []any{0, "container", 0, "resources", 0, "limits"},
		},
		"arguments": {
			value: types.ListValueMust(types.StringType, []attr.Value{types.StringUnknown()}),
			steps: []any{0, "container", 0, "args"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			plan := with(t, baseline, testCase.value, testCase.steps...).(types.List)
			if _, diagnostics := ExpandSpec(ctx, plan, templatePath); !diagnostics.HasError() {
				t.Fatal("unknown configured collection element reached the API")
			}
		})
	}
}

func sameResources(a, b corev1.ResourceRequirements) bool {
	same := func(x, y corev1.ResourceList) bool {
		if len(x) != len(y) {
			return false
		}
		for name, quantity := range x {
			other, ok := y[name]
			if !ok || quantity.Cmp(other) != 0 {
				return false
			}
		}
		return true
	}
	return same(a.Limits, b.Limits) && same(a.Requests, b.Requests)
}

func podSpecComputedPathsFor() map[string]bool {
	object := podSpecObject()
	computed := map[string]bool{}
	podSpecComputedPaths(object.Attributes, object.Blocks, "spec", computed)
	return computed
}

func hasPrefix(p, prefix path.Path) bool {
	for ; len(p.Steps()) > 0; p = p.ParentPath() {
		if p.Equal(prefix) {
			return true
		}
	}
	return false
}

func TestFlattenSpecCollectionOwnership(t *testing.T) {
	ctx := context.Background()
	in := corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "nginx"}}}
	baseline := mustFlatten(t, in, types.ListNull(specType()))
	pullSecrets := get(baseline, 0, "image_pull_secrets").(types.List)
	if !get(baseline, 0, "node_selector").IsNull() || !get(baseline, 0, "container", 0, "args").IsNull() ||
		!pullSecrets.Equal(types.ListValueMust(pullSecrets.ElementType(ctx), nil)) ||
		!get(baseline, 0, "security_context").(types.List).Equal(types.ListValueMust(get(baseline, 0, "security_context").(types.List).ElementType(ctx), nil)) {
		t.Fatalf("null baseline must follow the SDK flattener: absent optional collections null, computed references and blocks empty: %s", baseline)
	}
	baseline = with(t, baseline, types.ListNull(pullSecrets.ElementType(ctx)), 0, "image_pull_secrets").(types.List)
	stringList := types.ListValueMust(types.StringType, nil)
	baseline = with(t, baseline, types.MapValueMust(types.StringType, map[string]attr.Value{}), 0, "node_selector").(types.List)
	baseline = with(t, baseline, stringList, 0, "container", 0, "args").(types.List)
	references := get(baseline, 0, "readiness_gate").(types.List).ElementType(ctx)
	baseline = with(t, baseline, types.ListUnknown(references), 0, "readiness_gate").(types.List)
	refreshed := mustFlatten(t, in, baseline)
	if !get(refreshed, 0, "node_selector").Equal(types.MapValueMust(types.StringType, map[string]attr.Value{})) {
		t.Error("explicit empty map was not retained")
	}
	if !get(refreshed, 0, "container", 0, "args").Equal(stringList) {
		t.Error("explicit empty list was not retained")
	}
	if !get(refreshed, 0, "container", 0, "command").IsNull() {
		t.Error("null list was not retained")
	}
	if gate := get(refreshed, 0, "readiness_gate"); gate.IsUnknown() || gate.IsNull() {
		t.Errorf("planned-unknown computed list must become known, got %s", gate)
	}
	if !get(refreshed, 0, "image_pull_secrets").IsNull() {
		t.Error("null computed list without API values must stay null")
	}
}

func TestFlattenSpecQuantities(t *testing.T) {
	in := corev1.PodSpec{
		Containers: []corev1.Container{{Name: "app", Image: "nginx",
			Resources: corev1.ResourceRequirements{
				Limits:   corev1.ResourceList{corev1.ResourceCPU: kquantity.MustParse("1"), corev1.ResourceMemory: kquantity.MustParse("1Gi")},
				Requests: corev1.ResourceList{corev1.ResourceCPU: kquantity.MustParse("250m")},
			},
			Env: []corev1.EnvVar{{Name: "MEM", ValueFrom: &corev1.EnvVarSource{ResourceFieldRef: &corev1.ResourceFieldSelector{Resource: "limits.memory", Divisor: kquantity.MustParse("1Mi")}}}},
		}},
		Volumes: []corev1.Volume{{Name: "scratch", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{SizeLimit: ptr.To(kquantity.MustParse("1Gi"))}}}},
	}
	state := mustFlatten(t, in, types.ListNull(specType()))
	limits := types.MapValueMust(types.StringType, map[string]attr.Value{"cpu": types.StringValue("1000m"), "memory": types.StringValue("1024Mi")})
	baseline := with(t, state, limits, 0, "container", 0, "resources", 0, "limits").(types.List)
	baseline = with(t, baseline, types.StringValue("1024Mi"), 0, "volume", 0, "empty_dir", 0, "size_limit").(types.List)
	baseline = with(t, baseline, types.StringValue("1048576"), 0, "container", 0, "env", 0, "value_from", 0, "resource_field_ref", 0, "divisor").(types.List)
	if refreshed := mustFlatten(t, in, baseline); !refreshed.Equal(baseline) {
		t.Fatalf("semantically equal quantities were not retained:\n%s\n%s", refreshed, baseline)
	}
	in.Containers[0].Resources.Limits[corev1.ResourceCPU] = kquantity.MustParse("2")
	drift := mustFlatten(t, in, baseline)
	got := get(drift, 0, "container", 0, "resources", 0, "limits").(types.Map).Elements()
	if got["cpu"].(types.String).ValueString() != "2" || got["memory"].(types.String).ValueString() != "1024Mi" {
		t.Fatalf("quantity drift was suppressed or unrelated spellings changed: %v", got)
	}
}

func TestQuantityMapPlanModifierWholeMap(t *testing.T) {
	ctx := context.Background()
	quantities := func(values map[string]string) types.Map {
		elements := map[string]attr.Value{}
		for k, v := range values {
			elements[k] = types.StringValue(v)
		}
		return types.MapValueMust(types.StringType, elements)
	}
	state := quantities(map[string]string{"cpu": "1", "memory": "1Gi"})
	for name, test := range map[string]struct {
		plan, want types.Map
	}{
		"equivalent":     {quantities(map[string]string{"cpu": "1000m", "memory": "1024Mi"}), state},
		"partial change": {quantities(map[string]string{"cpu": "1000m", "memory": "2Gi"}), quantities(map[string]string{"cpu": "1000m", "memory": "2Gi"})},
		"added key":      {quantities(map[string]string{"cpu": "1", "memory": "1Gi", "gpu": "1"}), quantities(map[string]string{"cpu": "1", "memory": "1Gi", "gpu": "1"})},
		"null":           {types.MapNull(types.StringType), types.MapNull(types.StringType)},
	} {
		response := planmodifier.MapResponse{PlanValue: test.plan}
		podQuantityMapPlanModifier{}.PlanModifyMap(ctx, planmodifier.MapRequest{StateValue: state, PlanValue: test.plan}, &response)
		if !response.PlanValue.Equal(test.want) {
			t.Errorf("%s: plan %s, want %s", name, response.PlanValue, test.want)
		}
	}
	divisor := SpecBlock(Options{}).NestedObject.Blocks["container"].(schema.ListNestedBlock).NestedObject.Blocks["env"].(schema.ListNestedBlock).
		NestedObject.Blocks["value_from"].(schema.ListNestedBlock).NestedObject.Blocks["resource_field_ref"].(schema.ListNestedBlock).
		NestedObject.Attributes["divisor"].(schema.StringAttribute)
	response := planmodifier.StringResponse{PlanValue: types.StringValue("1000m")}
	for _, m := range divisor.PlanModifiers {
		m.PlanModifyString(ctx, planmodifier.StringRequest{StateValue: types.StringValue("1"), PlanValue: response.PlanValue}, &response)
	}
	if response.PlanValue.ValueString() != "1" {
		t.Errorf("equivalent divisor planned %s", response.PlanValue)
	}
}

// API admission (webhooks, LimitRange-style defaulting) may populate container
// resources the configuration omitted. They are computed and owned by the API.
func TestAdmissionPopulatedResourcesAreComputed(t *testing.T) {
	ctx := context.Background()
	for _, name := range []string{"container", "init_container"} {
		resources := SpecBlock(Options{}).NestedObject.Blocks[name].(schema.ListNestedBlock).NestedObject.Attributes["resources"].(schema.ListNestedAttribute)
		if !resources.Optional || !resources.Computed || len(resources.PlanModifiers) != 1 {
			t.Fatalf("%s.resources must be optional+computed with only UseStateForUnknown", name)
		}
		for _, child := range []string{"limits", "requests"} {
			if m := resources.NestedObject.Attributes[child].(schema.MapAttribute); !m.Optional || !m.Computed || podModifiersRequireReplacement(m.PlanModifiers) {
				t.Fatalf("%s.resources.%s must be optional+computed and updatable", name, child)
			}
		}
	}
	configured := mustFlatten(t, corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "nginx"}}}, types.ListNull(specType()))
	resources := get(configured, 0, "container", 0, "resources").(types.List)
	plan := with(t, configured, types.ListUnknown(resources.ElementType(ctx)), 0, "container", 0, "resources").(types.List)
	admitted := corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "nginx", Resources: corev1.ResourceRequirements{
		Limits:   corev1.ResourceList{corev1.ResourceCPU: kquantity.MustParse("1")},
		Requests: corev1.ResourceList{corev1.ResourceCPU: kquantity.MustParse("1")},
	}}}}
	state := mustFlatten(t, admitted, plan)
	got := get(state, 0, "container", 0, "resources", 0)
	if get(got, "limits").(types.Map).Elements()["cpu"].(types.String).ValueString() != "1" ||
		get(got, "requests").(types.Map).Elements()["cpu"].(types.String).ValueString() != "1" {
		t.Fatalf("admission-populated resources were not recorded: %s", got)
	}
}

func TestSpecBlockReplacementBoundaries(t *testing.T) {
	ctx := context.Background()
	object := SpecBlock(Options{}).NestedObject
	for _, name := range []string{"container", "init_container", "toleration", "host_aliases", "dns_config", "security_context"} {
		if b := object.Blocks[name].(schema.ListNestedBlock); len(b.PlanModifiers) != 0 {
			t.Errorf("%s must be updatable in templates, has %v", name, b.PlanModifiers)
		}
	}
	container := object.Blocks["container"].(schema.ListNestedBlock).NestedObject
	if image := container.Attributes["image"].(schema.StringAttribute); len(image.PlanModifiers) != 0 {
		t.Error("container image must be updatable in templates")
	}

	raw := tfsdk.State{Raw: tftypes.NewValue(tftypes.Bool, true)}
	plan := tfsdk.Plan{Raw: tftypes.NewValue(tftypes.Bool, true)}
	requiresReplace := func(block schema.ListNestedBlock, before, after []attr.Value) bool {
		elementType := block.NestedObject.Type()
		request := planmodifier.ListRequest{
			State: raw, Plan: plan,
			StateValue: types.ListValueMust(elementType, before), PlanValue: types.ListValueMust(elementType, after),
		}
		var response planmodifier.ListResponse
		for _, m := range block.PlanModifiers {
			m.PlanModifyList(ctx, request, &response)
		}
		return response.RequiresReplace
	}

	volume := object.Blocks["volume"].(schema.ListNestedBlock)
	volumeType := volume.NestedObject.Type().(types.ObjectType)
	scratch := with(t, emptyObject(volumeType), types.StringValue("scratch"), "name")
	emptyDir := emptyObject(volumeType.AttrTypes["empty_dir"].(types.ListType).ElemType.(types.ObjectType))
	scratch = with(t, scratch, types.ListValueMust(emptyDir.Type(ctx), []attr.Value{emptyDir}), "empty_dir")
	azureType := volumeType.AttrTypes["azure_file"].(types.ListType).ElemType.(types.ObjectType)
	azure := func(namespace types.String) attr.Value {
		file := with(t, emptyObject(azureType), namespace, "secret_namespace")
		return with(t, emptyObject(volumeType), types.ListValueMust(azureType, []attr.Value{file}), "azure_file")
	}
	for name, test := range map[string]struct {
		before, after []attr.Value
		replace       bool
	}{
		"add mutable volume":             {nil, []attr.Value{scratch}, false},
		"remove mutable volume":          {[]attr.Value{scratch}, nil, false},
		"add immutable secret namespace": {[]attr.Value{scratch}, []attr.Value{scratch, azure(types.StringValue("ns"))}, true},
		"add zero secret namespace":      {nil, []attr.Value{azure(types.StringNull())}, false},
	} {
		if got := requiresReplace(volume, test.before, test.after); got != test.replace {
			t.Errorf("volume %s: RequiresReplace=%t, want %t", name, got, test.replace)
		}
	}

	azureBlock := volume.NestedObject.Blocks["azure_file"].(schema.ListNestedBlock)
	secretNamespace := azureBlock.NestedObject.Attributes["secret_namespace"].(schema.StringAttribute)
	for name, test := range map[string]struct {
		before, after types.String
		replace       bool
	}{
		"zero to null": {types.StringValue(""), types.StringNull(), false},
		"change":       {types.StringValue("a"), types.StringValue("b"), true},
	} {
		request := planmodifier.StringRequest{State: raw, Plan: plan, StateValue: test.before, PlanValue: test.after, ConfigValue: test.after}
		var response planmodifier.StringResponse
		response.PlanValue = test.after
		for _, m := range secretNamespace.PlanModifiers {
			m.PlanModifyString(ctx, request, &response)
		}
		if response.RequiresReplace != test.replace {
			t.Errorf("secret_namespace %s: RequiresReplace=%t, want %t", name, response.RequiresReplace, test.replace)
		}
	}

	nodeAffinity := object.Blocks["affinity"].(schema.ListNestedBlock).NestedObject.Blocks["node_affinity"].(schema.ListNestedBlock).
		NestedObject.Blocks["required_during_scheduling_ignored_during_execution"].(schema.ListNestedBlock).
		NestedObject.Blocks["node_selector_term"].(schema.ListNestedBlock)
	term := nodeAffinity.NestedObject.Type().(types.ObjectType)
	matchFields := nodeAffinity.NestedObject.Blocks["match_fields"].(schema.ListNestedBlock)
	fieldType := term.AttrTypes["match_fields"].(types.ListType).ElemType.(types.ObjectType)
	field := with(t, emptyObject(fieldType), types.StringValue("metadata.name"), "key")
	if !requiresReplace(matchFields, nil, []attr.Value{field}) {
		t.Error("adding match_fields must require replacement as in SDKv2")
	}
	if requiresReplace(nodeAffinity, nil, []attr.Value{emptyObject(term)}) {
		t.Error("adding a node selector term without match_fields must be updatable")
	}
	if !requiresReplace(nodeAffinity, nil, []attr.Value{with(t, emptyObject(term), types.ListValueMust(fieldType, []attr.Value{field}), "match_fields")}) {
		t.Error("adding a node selector term with match_fields must require replacement")
	}
}
