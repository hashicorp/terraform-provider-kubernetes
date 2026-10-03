// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

func TestPodV1SpecRequiresReplacement(t *testing.T) {
	base := func() corev1.PodSpec {
		return corev1.PodSpec{
			Containers: []corev1.Container{{
				Name: "app", Image: "image:1",
				VolumeMounts: []corev1.VolumeMount{{Name: "data", MountPath: "/data"}},
				Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("500m"),
				}},
			}},
			Volumes: []corev1.Volume{{Name: "data", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}},
		}
	}
	deadline := func(seconds int64) func(*corev1.PodSpec) {
		return func(s *corev1.PodSpec) { s.ActiveDeadlineSeconds = ptr.To(seconds) }
	}
	for name, test := range map[string]struct {
		prior, edit func(*corev1.PodSpec)
		want        bool
	}{
		"unchanged":                  {edit: func(*corev1.PodSpec) {}},
		"active deadline is set":     {edit: deadline(60)},
		"active deadline is lowered": {prior: deadline(120), edit: deadline(60)},
		"active deadline is raised":  {prior: deadline(60), edit: deadline(120), want: true},
		"active deadline is removed": {prior: deadline(60), edit: func(s *corev1.PodSpec) { s.ActiveDeadlineSeconds = nil }, want: true},
		"container image": {edit: func(s *corev1.PodSpec) {
			s.Containers[0].Image = "image:2"
		}, want: true},
		"empty and unset values are equivalent": {edit: func(s *corev1.PodSpec) {
			s.SecurityContext = &corev1.PodSecurityContext{SupplementalGroups: []int64{}}
			s.Containers[0].Args = []string{}
			s.NodeSelector = map[string]string{}
			s.HostNetwork = false
		}},
		"quantity spelling": {edit: func(s *corev1.PodSpec) {
			s.Containers[0].Resources.Limits[corev1.ResourceCPU] = resource.MustParse("0.5")
		}},
		"volume mount path": {edit: func(s *corev1.PodSpec) {
			s.Containers[0].VolumeMounts[0].MountPath = "/other"
		}, want: true},
		"added toleration": {edit: func(s *corev1.PodSpec) {
			s.Tolerations = []corev1.Toleration{{Key: "k", Operator: corev1.TolerationOpExists}}
		}, want: true},
		"init container image": {edit: func(s *corev1.PodSpec) {
			s.InitContainers = []corev1.Container{{Name: "init", Image: "image:1"}}
		}, want: true},
	} {
		t.Run(name, func(t *testing.T) {
			prior, planned := base(), base()
			if test.prior != nil {
				test.prior(&prior)
				test.prior(&planned)
			}
			test.edit(&planned)
			if got := podV1SpecRequiresReplacement(prior, planned); got != test.want {
				t.Fatalf("requires replacement = %t, want %t", got, test.want)
			}
		})
	}
}

func TestRawAttributeUnchangedMatchesDecodedEqual(t *testing.T) {
	ctx := context.Background()
	specType := types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
		"image":  types.StringType,
		"labels": types.MapType{ElemType: types.StringType},
	}}}
	listType := specType.TerraformType(ctx).(tftypes.List)
	spec := func(image any) tftypes.Value {
		return tftypes.NewValue(listType, []tftypes.Value{tftypes.NewValue(listType.ElementType, map[string]tftypes.Value{
			"image": tftypes.NewValue(tftypes.String, image),
			"labels": tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{
				"a": tftypes.NewValue(tftypes.String, "1"),
				"b": tftypes.NewValue(tftypes.String, "2"),
			}),
		})})
	}
	for name, test := range map[string]struct {
		plan, state tftypes.Value
		want        bool
	}{
		"same value":           {plan: spec("app:1"), state: spec("app:1"), want: true},
		"changed value":        {plan: spec("app:2"), state: spec("app:1")},
		"null and empty list":  {plan: tftypes.NewValue(listType, []tftypes.Value{}), state: tftypes.NewValue(listType, nil)},
		"unknown nested value": {plan: spec(tftypes.UnknownValue), state: spec("app:1")},
	} {
		t.Run(name, func(t *testing.T) {
			object := func(v tftypes.Value) tftypes.Value {
				return tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{"spec": listType}}, map[string]tftypes.Value{"spec": v})
			}
			decoded := func(v tftypes.Value) attr.Value {
				value, err := specType.ValueFromTerraform(ctx, v)
				if err != nil {
					t.Fatal(err)
				}
				return value
			}
			got := rawAttributeUnchanged(object(test.plan), object(test.state), "spec")
			if equal := decoded(test.plan).Equal(decoded(test.state)); got != test.want || equal != test.want {
				t.Fatalf("unchanged = %t, decoded equal = %t, want %t", got, equal, test.want)
			}
		})
	}
}
