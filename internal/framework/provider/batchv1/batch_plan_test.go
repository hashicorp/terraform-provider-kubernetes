// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	batch "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
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

func TestPodSpecsEqual(t *testing.T) {
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
			if got := podSpecsEqual(tc.have, tc.want); got != tc.equal {
				t.Errorf("podSpecsEqual = %t, want %t", got, tc.equal)
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
			ops, err := cronJobSpecOps(raw, spec(nil), tc.desired)
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
			if got := ops[0].(*kubernetes.ReplaceOperation).Value; !sameJSON(got, want) {
				t.Errorf("patched spec = %v", got)
			}
		})
	}
}
