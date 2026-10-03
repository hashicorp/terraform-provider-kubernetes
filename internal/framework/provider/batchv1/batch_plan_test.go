// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	batch "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
)

func TestResolveUnconfigured(t *testing.T) {
	typ := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"policy": tftypes.String, "port": tftypes.String, "labels": tftypes.Map{ElementType: tftypes.String},
	}}
	str := func(s string) tftypes.Value { return tftypes.NewValue(tftypes.String, s) }
	null := tftypes.NewValue(tftypes.String, nil)
	unknown := tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
	labels := tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{"a": str("b")})
	noLabels := tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil)
	object := func(policy, port, labels tftypes.Value) tftypes.Value {
		return tftypes.NewValue(typ, map[string]tftypes.Value{"policy": policy, "port": port, "labels": labels})
	}
	// Only "policy" is API-defaulted.
	apiDefaulted := func(at *tftypes.AttributePath) bool {
		return at.Equal(tftypes.NewAttributePath().WithAttributeName("policy"))
	}
	state := object(str("Always"), str("80"), labels)
	older := object(str("Always"), null, labels)
	none := tftypes.NewValue(typ, nil)
	for name, tc := range map[string]struct {
		config, plan tftypes.Value
		want         tftypes.Value
		ok           bool
		state        *tftypes.Value
	}{
		"empty API-defaulted string keeps the prior value": {
			object(str(""), str("80"), labels), object(str(""), str("80"), labels), state, true, nil,
		},
		"empty string clears any other field": {
			object(str("Always"), str(""), labels), object(str("Always"), str(""), labels),
			object(str("Always"), str(""), labels), true, nil,
		},
		"unset scalar is removed": {
			object(null, str("80"), labels), object(null, str("80"), labels), object(null, str("80"), labels), true, nil,
		},
		"unset scalar keeps its planned default": {
			object(str("Always"), null, labels), object(str("Always"), str(""), labels),
			object(str("Always"), str(""), labels), true, nil,
		},
		"configured change is kept": {
			object(str("Never"), str("8080"), labels), object(str("Never"), str("8080"), labels),
			object(str("Never"), str("8080"), labels), true, nil,
		},
		"unset collection is removed": {
			object(str("Always"), str("80"), noLabels), object(str("Always"), str("80"), noLabels),
			object(str("Always"), str("80"), noLabels), true, nil,
		},
		"zero default of a field older state lacks": {
			object(str("Always"), null, labels), object(str("Always"), str(""), labels), older, true, &older,
		},
		"unknown configuration cannot be resolved": {
			object(null, unknown, labels), object(str(""), unknown, labels), tftypes.Value{}, false, nil,
		},
		"unset unknown without a prior stays unknown": {
			object(null, str("80"), labels), object(unknown, str("80"), labels), object(unknown, str("80"), labels), true,
			&none,
		},
	} {
		t.Run(name, func(t *testing.T) {
			prior := state
			if tc.state != nil {
				prior = *tc.state
			}
			got, ok := resolveUnconfigured(tftypes.NewAttributePath(), tc.config, tc.plan, prior, apiDefaulted)
			if ok != tc.ok || (ok && !got.Equal(tc.want)) {
				t.Errorf("got %s, %t; want %s, %t", got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestAPIDefaultedStrings(t *testing.T) {
	var resp resource.SchemaResponse
	(&CronJobV1{}).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	apiDefaulted := apiDefaultedStrings(context.Background(), resp.Schema)
	spec := tftypes.NewAttributePath().WithAttributeName("spec").WithElementKeyInt(0)
	jobSpec := spec.WithAttributeName("job_template").WithElementKeyInt(0).WithAttributeName("spec").WithElementKeyInt(0)
	podSpec := jobSpec.WithAttributeName("template").WithElementKeyInt(0).WithAttributeName("spec").WithElementKeyInt(0)
	container := podSpec.WithAttributeName("container").WithElementKeyInt(0)
	templateMeta := jobSpec.WithAttributeName("template").WithElementKeyInt(0).WithAttributeName("metadata").WithElementKeyInt(0)
	for at, want := range map[*tftypes.AttributePath]bool{
		spec.WithAttributeName("timezone"):                                                             false,
		spec.WithAttributeName("concurrency_policy"):                                                   false,
		jobSpec.WithAttributeName("completion_mode"):                                                   true,
		jobSpec.WithAttributeName("ttl_seconds_after_finished"):                                        false,
		podSpec.WithAttributeName("scheduler_name"):                                                    true,
		podSpec.WithAttributeName("service_account_name"):                                              true,
		podSpec.WithAttributeName("priority_class_name"):                                               false,
		templateMeta.WithAttributeName("uid"):                                                          false,
		container.WithAttributeName("image_pull_policy"):                                               true,
		container.WithAttributeName("working_dir"):                                                     false,
		container.WithAttributeName("volume_mount").WithElementKeyInt(0).WithAttributeName("sub_path"): false,
	} {
		if got := apiDefaulted(at); got != want {
			t.Errorf("%s: got %t, want %t", at, got, want)
		}
	}
}

func TestPodTemplatesEqual(t *testing.T) {
	container := func(sc *corev1.SecurityContext) corev1.PodSpec {
		return corev1.PodSpec{Containers: []corev1.Container{{Name: "c", SecurityContext: sc}}}
	}
	cpu := func(q string) corev1.PodSpec {
		return corev1.PodSpec{Containers: []corev1.Container{{Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{"cpu": apiresource.MustParse(q)}}}}}
	}
	for name, tc := range map[string]struct {
		have, want corev1.PodSpec
		equal      bool
	}{
		"allow_privilege_escalation false": {container(&corev1.SecurityContext{AllowPrivilegeEscalation: ptr.To(false)}), container(nil), false},
		"container run_as_non_root false":  {container(&corev1.SecurityContext{RunAsNonRoot: ptr.To(false)}), container(nil), false},
		"read_only_root_filesystem false":  {container(&corev1.SecurityContext{ReadOnlyRootFilesystem: ptr.To(false)}), container(nil), true},
		"privileged false":                 {container(&corev1.SecurityContext{Privileged: ptr.To(false)}), container(nil), true},
		"pod run_as_non_root false to remove": {
			corev1.PodSpec{SecurityContext: &corev1.PodSecurityContext{RunAsNonRoot: ptr.To(false), SupplementalGroups: []int64{}}},
			corev1.PodSpec{}, false,
		},
		"job controller label": {
			corev1.PodSpec{}, corev1.PodSpec{}, true,
		},
		"automount_service_account_token false": {corev1.PodSpec{AutomountServiceAccountToken: ptr.To(false)}, corev1.PodSpec{}, false},
		"enable_service_links false":            {corev1.PodSpec{EnableServiceLinks: ptr.To(false)}, corev1.PodSpec{}, false},
		"share_process_namespace false":         {corev1.PodSpec{ShareProcessNamespace: ptr.To(false)}, corev1.PodSpec{}, true},
		"termination_grace_period_seconds 0":    {corev1.PodSpec{TerminationGracePeriodSeconds: ptr.To(int64(0))}, corev1.PodSpec{}, false},
		"container run_as_user 0":               {container(&corev1.SecurityContext{RunAsUser: ptr.To(int64(0))}), container(nil), false},
		"empty and absent maps":                 {corev1.PodSpec{NodeSelector: map[string]string{}}, corev1.PodSpec{}, true},
		"respelled quantity":                    {cpu("0.5"), cpu("500m"), true},
		"changed value":                         {corev1.PodSpec{SecurityContext: &corev1.PodSecurityContext{RunAsNonRoot: ptr.To(true)}}, corev1.PodSpec{}, false},
		"element order": {
			corev1.PodSpec{Containers: []corev1.Container{{Name: "a"}, {Name: "b"}}},
			corev1.PodSpec{Containers: []corev1.Container{{Name: "b"}, {Name: "a"}}},
			false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			have := corev1.PodTemplateSpec{Spec: tc.have}
			have.Labels = map[string]string{"job-name": "j"}
			if got := podTemplatesEqual(have, corev1.PodTemplateSpec{Spec: tc.want}); got != tc.equal {
				t.Errorf("podTemplatesEqual = %t, want %t", got, tc.equal)
			}
		})
	}
}

func TestCronJobSpecOps(t *testing.T) {
	spec := func(change func(*batch.CronJobSpec)) batch.CronJobSpec {
		s := batch.CronJobSpec{Schedule: "0 0 * * *", TimeZone: ptr.To("UTC"), StartingDeadlineSeconds: ptr.To(int64(10))}
		s.JobTemplate.Spec.BackoffLimit = ptr.To(int32(6))
		s.JobTemplate.Spec.Template.Spec.Containers = []corev1.Container{{Name: "c", Image: "busybox", VolumeMounts: []corev1.VolumeMount{{Name: "v", MountPath: "/d", SubPath: "x"}}}}
		if change != nil {
			change(&s)
		}
		return s
	}
	container := func(s *batch.CronJobSpec) *corev1.Container { return &s.JobTemplate.Spec.Template.Spec.Containers[0] }
	// The live object holds the API's defaults and a field another tool set.
	live := spec(func(s *batch.CronJobSpec) {
		s.Suspend = ptr.To(false)
		container(s).ImagePullPolicy = corev1.PullIfNotPresent
		s.JobTemplate.Spec.Template.Spec.PriorityClassName = "admission"
	})
	for name, tc := range map[string]struct {
		desired batch.CronJobSpec
		changed bool
	}{
		"unchanged":                    {spec(nil), false},
		"suspend false is the default": {spec(func(s *batch.CronJobSpec) { s.Suspend = ptr.To(false) }), false},
		"allow_privilege_escalation false is added": {spec(func(s *batch.CronJobSpec) {
			container(s).SecurityContext = &corev1.SecurityContext{AllowPrivilegeEscalation: ptr.To(false)}
		}), true},
		"suspend true":                 {spec(func(s *batch.CronJobSpec) { s.Suspend = ptr.To(true) }), true},
		"timezone is cleared":          {spec(func(s *batch.CronJobSpec) { s.TimeZone = nil }), true},
		"sub_path is cleared":          {spec(func(s *batch.CronJobSpec) { container(s).VolumeMounts[0].SubPath = "" }), true},
		"starting deadline is cleared": {spec(func(s *batch.CronJobSpec) { s.StartingDeadlineSeconds = nil }), true},
		"backoff limit zero":           {spec(func(s *batch.CronJobSpec) { s.JobTemplate.Spec.BackoffLimit = ptr.To(int32(0)) }), true},
	} {
		t.Run(name, func(t *testing.T) {
			object, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&batch.CronJob{Spec: live})
			if err != nil {
				t.Fatal(err)
			}
			raw := &unstructured.Unstructured{Object: object}
			ops, err := common.StrategicMergeSpecOps(raw, spec(nil), tc.desired, batch.CronJob{})
			if err != nil {
				t.Fatal(err)
			}
			if changed := len(ops) > 0; changed != tc.changed {
				t.Fatalf("changed = %t, want %t", changed, tc.changed)
			}
			if !tc.changed {
				return
			}
			// The result is the desired spec, with what the provider does not
			// manage kept as it is live.
			want := *tc.desired.DeepCopy()
			if want.Suspend == nil {
				want.Suspend = ptr.To(false)
			}
			container(&want).ImagePullPolicy = corev1.PullIfNotPresent
			want.JobTemplate.Spec.Template.Spec.PriorityClassName = "admission"
			wantSpec, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&batch.CronJob{Spec: want})
			if err != nil {
				t.Fatal(err)
			}
			if got := ops[0].(*kubernetes.ReplaceOperation).Value; !reflect.DeepEqual(got, wantSpec["spec"]) {
				t.Errorf("patched spec = %v", got)
			}
		})
	}
}

func TestJobTemplateKeysReplace(t *testing.T) {
	ctx := context.Background()
	nulls := func(typ types.ObjectType) map[string]attr.Value {
		out := map[string]attr.Value{}
		for name, at := range typ.AttrTypes {
			out[name], _ = at.ValueFromTerraform(ctx, tftypes.NewValue(at.TerraformType(ctx), nil))
		}
		return out
	}
	templateType := jobSpecType().AttrTypes["template"].(types.ListType)
	templateObject := templateType.ElemType.(types.ObjectType)
	metadataObject := templateObject.AttrTypes["metadata"].(types.ListType).ElemType.(types.ObjectType)
	root := func(labels map[string]string) tftypes.Value {
		metadata := nulls(metadataObject)
		if labels != nil {
			metadata["labels"], _ = types.MapValueFrom(ctx, types.StringType, labels)
		}
		template := nulls(templateObject)
		template["metadata"] = types.ListValueMust(metadataObject, []attr.Value{types.ObjectValueMust(metadataObject.AttrTypes, metadata)})
		value, err := types.ListValueMust(templateObject, []attr.Value{types.ObjectValueMust(templateObject.AttrTypes, template)}).ToTerraformValue(ctx)
		if err != nil {
			t.Fatal(err)
		}
		spec := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"template": value.Type()}}
		return tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{"spec": tftypes.List{ElementType: spec}}},
			map[string]tftypes.Value{"spec": tftypes.NewValue(tftypes.List{ElementType: spec}, []tftypes.Value{
				tftypes.NewValue(spec, map[string]tftypes.Value{"template": value}),
			})})
	}
	app := map[string]string{"app": "a"}
	injected := map[string]string{"app": "a", "admission.example.com/injected": "true"}
	for _, tc := range []struct {
		name          string
		state, config map[string]string
		owned, want   bool
	}{
		{"key only in state", injected, app, true, true},
		{"key only in state of an unset map", map[string]string{"admission.example.com/injected": "true"}, nil, true, true},
		{"key only in state not owned", injected, app, false, false},
		{"key only in state of an unset map not owned", map[string]string{"admission.example.com/injected": "true"}, nil, false, false},
		{"changed value", app, map[string]string{"app": "b"}, true, true},
		{"changed value not owned", app, map[string]string{"app": "b"}, false, true},
		{"added key not owned", app, injected, false, true},
		{"generated label only in state", map[string]string{"app": "a", "job-name": "j"}, app, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := root(tc.config)
			got, diags := jobTemplateChanged(ctx, config, config, root(tc.state), func(*tftypes.AttributePath) bool { return false }, tc.owned)
			if diags.HasError() || got != tc.want {
				t.Errorf("got %t (%v), want %t", got, diags, tc.want)
			}
		})
	}
}

func TestTemplateMap(t *testing.T) {
	live := map[string]string{"app": "a", "admission.example.com/injected": "true", "job-name": "j"}
	app := types.MapValueMust(types.StringType, map[string]attr.Value{"app": types.StringValue("a")})
	overridden := types.MapValueMust(types.StringType, map[string]attr.Value{"app": types.StringValue("z")})
	all := types.MapValueMust(types.StringType, map[string]attr.Value{
		"app": types.StringValue("a"), "admission.example.com/injected": types.StringValue("true"),
	})
	for _, tc := range []struct {
		name  string
		prior types.Map
		live  bool
		want  types.Map
	}{
		{"unset map stays null", types.MapNull(types.StringType), false, types.MapNull(types.StringType)},
		{"recorded map keeps its keys and values", overridden, false, overridden},
		{"live read records every key", app, true, all},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := templateMap(live, tc.prior, tc.live, jobGeneratedLabels); !got.Equal(tc.want) {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

// Kubernetes drops empty selector values; a configured empty set is kept.
func TestSelectorEmptyValues(t *testing.T) {
	typ := jobSpecObjectType(true).AttrTypes["selector"].(types.ObjectType)
	expressionType := typ.AttrTypes["match_expressions"].(types.ListType).ElemType.(types.ObjectType)
	selector := func(values types.Set) types.Object {
		expression := types.ObjectValueMust(expressionType.AttrTypes, map[string]attr.Value{
			"key": types.StringValue("app"), "operator": types.StringValue("Exists"), "values": values,
		})
		return types.ObjectValueMust(typ.AttrTypes, map[string]attr.Value{
			"match_labels":      types.MapNull(types.StringType),
			"match_expressions": types.ListValueMust(expressionType, []attr.Value{expression}),
		})
	}
	empty, null := types.SetValueMust(types.StringType, []attr.Value{}), types.SetNull(types.StringType)
	live := &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{{Key: "app", Operator: metav1.LabelSelectorOpExists}}}
	for name, tc := range map[string]struct {
		prior attr.Value
		want  types.Object
	}{
		"configured empty": {selector(empty), selector(empty)},
		"unset":            {selector(null), selector(null)},
		"no prior":         {types.ObjectNull(typ.AttrTypes), selector(null)},
	} {
		if got := flattenLabelSelector(live, tc.prior, typ, nil); !got.Equal(tc.want) {
			t.Errorf("%s: got %s, want %s", name, got, tc.want)
		}
	}
}

type planClientsets struct {
	kubernetes.KubeClientsets
	client *k8sclient.Clientset
}

func (c planClientsets) MainClientset() (*k8sclient.Clientset, error) { return c.client, nil }
func (planClientsets) GetIgnoreAnnotations() []string                 { return nil }
func (planClientsets) GetIgnoreLabels() []string                      { return nil }

// The prior state cannot tell a removed runAsNonRoot: false from an unset field.
func TestJobV1ModifyPlanReadsLiveJob(t *testing.T) {
	ctx := context.Background()
	for name, test := range map[string]struct {
		status      int
		wantReplace bool
		wantError   bool
	}{
		"live Job holds the removed block": {status: http.StatusOK, wantReplace: true},
		"Job no longer exists":             {status: http.StatusNotFound},
		"forbidden":                        {status: http.StatusForbidden, wantError: true},
		"unavailable":                      {status: http.StatusServiceUnavailable, wantError: true},
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/apis/batch/v1/namespaces/ns/jobs/j" || test.status != http.StatusOK {
					http.Error(w, "unavailable", test.status)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(&batch.Job{Spec: batch.JobSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{
					Containers:      []corev1.Container{{Name: "c", Image: "i"}},
					SecurityContext: &corev1.PodSecurityContext{RunAsNonRoot: ptr.To(false)},
				}}}})
			}))
			defer server.Close()
			client, err := k8sclient.NewForConfig(&rest.Config{Host: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			job := &JobV1{SDKv2Meta: func() any { return planClientsets{client: client} }}
			var schemaResp resource.SchemaResponse
			job.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
			value := func(podSpec string) tfsdk.State {
				raw := tfprotov6.RawState{JSON: []byte(`{"id":"ns/j","wait_for_completion":false,
					"metadata":[{"name":"j","namespace":"ns"}],
					"spec":[{"template":[{"spec":[{"container":[{"name":"c","image":"i"}]` + podSpec + `}]}]}]}`)}
				v, err := raw.Unmarshal(schemaResp.Schema.Type().TerraformType(ctx))
				if err != nil {
					t.Fatal(err)
				}
				return tfsdk.State{Schema: schemaResp.Schema, Raw: v}
			}
			state, planned := value(`,"security_context":[{"run_as_non_root":false}]`), value("")
			req := resource.ModifyPlanRequest{State: state, Plan: tfsdk.Plan(planned), Config: tfsdk.Config(planned)}
			resp := resource.ModifyPlanResponse{Plan: req.Plan}
			job.ModifyPlan(ctx, req, &resp)
			if resp.Diagnostics.HasError() != test.wantError {
				t.Fatalf("diagnostics: %v", resp.Diagnostics)
			}
			if replace := len(resp.RequiresReplace) == 1 && resp.RequiresReplace[0].Equal(jobTemplatePath); replace != test.wantReplace {
				t.Fatalf("requires replace = %v, want %t", resp.RequiresReplace, test.wantReplace)
			}
		})
	}
}
