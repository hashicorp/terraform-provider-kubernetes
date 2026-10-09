// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package podspec

import (
	"fmt"
	"slices"

	corev1 "k8s.io/api/core/v1"
)

// NormalizeFeatureDefaults makes runtime defaults comparable with absent
// pointer fields from state written before these attributes were supported.
func NormalizeFeatureDefaults(spec *corev1.PodSpec) {
	if spec.HostUsers != nil && *spec.HostUsers {
		spec.HostUsers = nil
	}
	if spec.SetHostnameAsFQDN != nil && !*spec.SetHostnameAsFQDN {
		spec.SetHostnameAsFQDN = nil
	}
	if spec.SecurityContext != nil && spec.SecurityContext.SupplementalGroupsPolicy != nil && *spec.SecurityContext.SupplementalGroupsPolicy == corev1.SupplementalGroupsPolicyMerge {
		spec.SecurityContext.SupplementalGroupsPolicy = nil
	}
	for _, containers := range [][]corev1.Container{spec.Containers, spec.InitContainers} {
		for i := range containers {
			container := &containers[i]
			if container.SecurityContext != nil && container.SecurityContext.ProcMount != nil && *container.SecurityContext.ProcMount == corev1.DefaultProcMount {
				container.SecurityContext.ProcMount = nil
			}
			for j := range container.VolumeMounts {
				mount := &container.VolumeMounts[j]
				if mount.RecursiveReadOnly != nil && *mount.RecursiveReadOnly == corev1.RecursiveReadOnlyDisabled {
					mount.RecursiveReadOnly = nil
				}
			}
		}
	}
}

// HasGatedFeatures identifies requests that Kubernetes can silently discard
// when a feature gate is disabled. A dry-run checks API admission only; node
// and container-runtime support is still required to run the Pod.
func HasGatedFeatures(spec *corev1.PodSpec) bool {
	if spec.HostUsers != nil && !*spec.HostUsers {
		return true
	}
	if sc := spec.SecurityContext; sc != nil && (sc.SupplementalGroupsPolicy != nil || sc.SELinuxChangePolicy != nil) {
		return true
	}
	for _, term := range podAffinityTerms(spec.Affinity) {
		if len(term.MatchLabelKeys)+len(term.MismatchLabelKeys) > 0 {
			return true
		}
	}
	for _, containers := range [][]corev1.Container{spec.Containers, spec.InitContainers} {
		for _, container := range containers {
			if container.SecurityContext != nil && container.SecurityContext.ProcMount != nil && *container.SecurityContext.ProcMount == corev1.UnmaskedProcMount {
				return true
			}
			for _, mount := range container.VolumeMounts {
				if mount.RecursiveReadOnly != nil {
					return true
				}
			}
			if container.Lifecycle != nil {
				for _, handler := range []*corev1.LifecycleHandler{container.Lifecycle.PostStart, container.Lifecycle.PreStop} {
					if handler != nil && handler.Sleep != nil {
						return true
					}
				}
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
	if expected.SetHostnameAsFQDN != nil && *expected.SetHostnameAsFQDN != (actual.SetHostnameAsFQDN != nil && *actual.SetHostnameAsFQDN) {
		return featureNotPreserved("set_hostname_as_fqdn", "SetHostnameAsFQDN")
	}
	if want := expected.SecurityContext; want != nil {
		got := actual.SecurityContext
		if got == nil {
			got = &corev1.PodSecurityContext{}
		}
		if !sameDefaultedPointer(want.SupplementalGroupsPolicy, got.SupplementalGroupsPolicy, corev1.SupplementalGroupsPolicyMerge) {
			return featureNotPreserved("supplemental_groups_policy", "SupplementalGroupsPolicy")
		}
		if !sameConfiguredPointer(want.SELinuxChangePolicy, got.SELinuxChangePolicy) {
			return featureNotPreserved("se_linux_change_policy", "SELinuxChangePolicy")
		}
		if !sameAppArmor(want.AppArmorProfile, got.AppArmorProfile) {
			return featureNotPreserved("app_armor_profile", "AppArmor")
		}
	}
	wantTerms, gotTerms := podAffinityTerms(expected.Affinity), podAffinityTerms(actual.Affinity)
	for i, term := range wantTerms {
		if len(term.MatchLabelKeys)+len(term.MismatchLabelKeys) == 0 {
			continue
		}
		if i >= len(gotTerms) || !sameStringSet(term.MatchLabelKeys, gotTerms[i].MatchLabelKeys) || !sameStringSet(term.MismatchLabelKeys, gotTerms[i].MismatchLabelKeys) {
			return featureNotPreserved("affinity label keys", "MatchLabelKeysInPodAffinity")
		}
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
		if want := container.SecurityContext; want != nil {
			security := got.SecurityContext
			if security == nil {
				security = &corev1.SecurityContext{}
			}
			if !sameAppArmor(want.AppArmorProfile, security.AppArmorProfile) {
				return featureNotPreserved(fmt.Sprintf("container %q app_armor_profile", container.Name), "AppArmor")
			}
			if windows := want.WindowsOptions; windows != nil {
				actualWindows := security.WindowsOptions
				if actualWindows == nil {
					actualWindows = &corev1.WindowsSecurityContextOptions{}
				}
				if !sameConfiguredPointer(windows.GMSACredentialSpec, actualWindows.GMSACredentialSpec) || !sameConfiguredPointer(windows.GMSACredentialSpecName, actualWindows.GMSACredentialSpecName) || !sameConfiguredPointer(windows.RunAsUserName, actualWindows.RunAsUserName) || !sameConfiguredPointer(windows.HostProcess, actualWindows.HostProcess) {
					return featureNotPreserved(fmt.Sprintf("container %q windows_options", container.Name), "Windows security options")
				}
			}
		}
		for i, probe := range []*corev1.Probe{container.LivenessProbe, container.StartupProbe} {
			if probe == nil || probe.TerminationGracePeriodSeconds == nil {
				continue
			}
			actualProbe := []*corev1.Probe{got.LivenessProbe, got.StartupProbe}[i]
			if actualProbe == nil || !sameConfiguredPointer(probe.TerminationGracePeriodSeconds, actualProbe.TerminationGracePeriodSeconds) {
				return featureNotPreserved(fmt.Sprintf("container %q probe termination grace period", container.Name), "ProbeTerminationGracePeriod")
			}
		}
		if container.Lifecycle != nil {
			actualLifecycle := got.Lifecycle
			if actualLifecycle == nil {
				actualLifecycle = &corev1.Lifecycle{}
			}
			for i, handler := range []*corev1.LifecycleHandler{container.Lifecycle.PostStart, container.Lifecycle.PreStop} {
				if handler == nil || handler.Sleep == nil {
					continue
				}
				actualHandler := []*corev1.LifecycleHandler{actualLifecycle.PostStart, actualLifecycle.PreStop}[i]
				if actualHandler == nil || actualHandler.Sleep == nil || actualHandler.Sleep.Seconds != handler.Sleep.Seconds {
					return featureNotPreserved(fmt.Sprintf("container %q lifecycle sleep", container.Name), "PodLifecycleSleepAction")
				}
			}
		}
		for _, mount := range container.VolumeMounts {
			if mount.RecursiveReadOnly != nil && !slices.ContainsFunc(got.VolumeMounts, func(candidate corev1.VolumeMount) bool {
				return candidate.Name == mount.Name && candidate.MountPath == mount.MountPath && sameDefaultedPointer(mount.RecursiveReadOnly, candidate.RecursiveReadOnly, corev1.RecursiveReadOnlyDisabled)
			}) {
				return featureNotPreserved(fmt.Sprintf("container %q recursive_read_only", container.Name), "RecursiveReadOnlyMounts")
			}
			if imageVolumes[mount.Name] && !slices.ContainsFunc(got.VolumeMounts, func(candidate corev1.VolumeMount) bool {
				return candidate.Name == mount.Name && candidate.MountPath == mount.MountPath && candidate.SubPath == mount.SubPath && candidate.SubPathExpr == mount.SubPathExpr
			}) {
				return featureNotPreserved(fmt.Sprintf("container %q volume mount %q", container.Name, mount.Name), "ImageVolume")
			}
		}
	}
	return nil
}

func sameConfiguredPointer[T comparable](expected, actual *T) bool {
	return expected == nil || actual != nil && *expected == *actual
}

func sameDefaultedPointer[T comparable](expected, actual *T, fallback T) bool {
	return sameConfiguredPointer(expected, actual) || expected != nil && actual == nil && *expected == fallback
}

func sameAppArmor(expected, actual *corev1.AppArmorProfile) bool {
	return expected == nil || actual != nil && expected.Type == actual.Type && sameConfiguredPointer(expected.LocalhostProfile, actual.LocalhostProfile)
}

func sameStringSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for _, value := range a {
		if !slices.Contains(b, value) {
			return false
		}
	}
	return true
}

func podAffinityTerms(affinity *corev1.Affinity) []*corev1.PodAffinityTerm {
	var terms []*corev1.PodAffinityTerm
	if affinity == nil {
		return terms
	}
	appendTerms := func(required []corev1.PodAffinityTerm, preferred []corev1.WeightedPodAffinityTerm) {
		for i := range required {
			terms = append(terms, &required[i])
		}
		for i := range preferred {
			terms = append(terms, &preferred[i].PodAffinityTerm)
		}
	}
	if affinity.PodAffinity != nil {
		appendTerms(affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution, affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution)
	}
	if affinity.PodAntiAffinity != nil {
		appendTerms(affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution, affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution)
	}
	return terms
}

func featureNotPreserved(field, gate string) error {
	return fmt.Errorf("Kubernetes did not preserve the requested %s. Check that the API server supports the %s feature and that admission policies do not remove or change the configured value", field, gate)
}
