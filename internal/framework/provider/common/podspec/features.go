// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package podspec

import (
	"fmt"
	"slices"

	corev1 "k8s.io/api/core/v1"
)

// HasGatedFeatures identifies requests that Kubernetes can silently discard
// when a feature gate is disabled. A dry-run checks API admission only; node
// and container-runtime support is still required to run the Pod.
func HasGatedFeatures(spec *corev1.PodSpec) bool {
	if spec.HostUsers != nil && !*spec.HostUsers {
		return true
	}
	for _, containers := range [][]corev1.Container{spec.Containers, spec.InitContainers} {
		for _, container := range containers {
			if container.SecurityContext != nil && container.SecurityContext.ProcMount != nil && *container.SecurityContext.ProcMount == corev1.UnmaskedProcMount {
				return true
			}
		}
	}
	return slices.ContainsFunc(spec.Volumes, func(volume corev1.Volume) bool { return volume.Image != nil })
}

// CheckPodFeaturePreservation verifies requested feature values before planned-value
// preservation can hide API pruning. Admission may add or reorder containers
// and volumes, so comparisons use their names.
func CheckPodFeaturePreservation(expected, actual *corev1.PodSpec) error {
	if expected.HostUsers != nil && *expected.HostUsers != (actual.HostUsers == nil || *actual.HostUsers) {
		return featureNotPreserved("host_users", "UserNamespacesSupport")
	}
	imageVolumes := make(map[string]bool)
	for _, volume := range expected.Volumes {
		if volume.Image == nil {
			continue
		}
		imageVolumes[volume.Name] = true
		index := slices.IndexFunc(actual.Volumes, func(candidate corev1.Volume) bool { return candidate.Name == volume.Name })
		if index < 0 || actual.Volumes[index].Image == nil {
			return featureNotPreserved(fmt.Sprintf("volume %q image", volume.Name), "ImageVolume")
		}
		got := actual.Volumes[index].Image
		if volume.Image.Reference != "" && volume.Image.Reference != got.Reference || volume.Image.PullPolicy != "" && volume.Image.PullPolicy != got.PullPolicy {
			return featureNotPreserved(fmt.Sprintf("volume %q image", volume.Name), "ImageVolume")
		}
	}
	if err := checkContainerFeatures(expected.Containers, actual.Containers, imageVolumes); err != nil {
		return err
	}
	return checkContainerFeatures(expected.InitContainers, actual.InitContainers, imageVolumes)
}

func checkContainerFeatures(expected, actual []corev1.Container, imageVolumes map[string]bool) error {
	for _, container := range expected {
		index := slices.IndexFunc(actual, func(candidate corev1.Container) bool { return candidate.Name == container.Name })
		var got corev1.Container
		if index >= 0 {
			got = actual[index]
		}
		if container.SecurityContext != nil && container.SecurityContext.ProcMount != nil {
			procMount := corev1.DefaultProcMount
			if got.SecurityContext != nil && got.SecurityContext.ProcMount != nil {
				procMount = *got.SecurityContext.ProcMount
			}
			if index < 0 || *container.SecurityContext.ProcMount != procMount {
				return featureNotPreserved(fmt.Sprintf("container %q proc_mount", container.Name), "ProcMountType")
			}
		}
		for _, mount := range container.VolumeMounts {
			if imageVolumes[mount.Name] && !slices.ContainsFunc(got.VolumeMounts, func(candidate corev1.VolumeMount) bool {
				return candidate.Name == mount.Name && candidate.MountPath == mount.MountPath && candidate.SubPath == mount.SubPath && candidate.SubPathExpr == mount.SubPathExpr
			}) {
				return featureNotPreserved(fmt.Sprintf("container %q volume mount %q", container.Name, mount.Name), "ImageVolume")
			}
		}
	}
	return nil
}

func featureNotPreserved(field, gate string) error {
	return fmt.Errorf("Kubernetes did not preserve the requested %s. Check that the API server supports the %s feature and that admission policies do not remove or change the configured value", field, gate)
}
