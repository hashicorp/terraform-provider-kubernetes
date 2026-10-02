// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"testing"

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
	for name, test := range map[string]struct {
		edit func(*corev1.PodSpec)
		want bool
	}{
		"unchanged": {edit: func(*corev1.PodSpec) {}},
		"active deadline is patched": {edit: func(s *corev1.PodSpec) {
			s.ActiveDeadlineSeconds = ptr.To(int64(60))
		}},
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
			planned := base()
			test.edit(&planned)
			if got := podV1SpecRequiresReplacement(base(), planned); got != test.want {
				t.Fatalf("requires replacement = %t, want %t", got, test.want)
			}
		})
	}
}
