// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/#taint-based-evictions
var builtInTolerations = map[string]string{
	v1.TaintNodeNotReady:           "",
	v1.TaintNodeUnreachable:        "",
	v1.TaintNodeUnschedulable:      "",
	v1.TaintNodeMemoryPressure:     "",
	v1.TaintNodeDiskPressure:       "",
	v1.TaintNodeNetworkUnavailable: "",
	v1.TaintNodePIDPressure:        "",
}

// Flatteners

func flattenOS(in v1.PodOS) []interface{} {
	att := make(map[string]interface{})
	if in.Name != "" {
		att["name"] = in.Name
	}
	return []interface{}{att}
}

func flattenPodSpec(in v1.PodSpec) ([]interface{}, error) {
	att := make(map[string]interface{})
	if in.ActiveDeadlineSeconds != nil {
		att["active_deadline_seconds"] = *in.ActiveDeadlineSeconds
	}

	if in.Affinity != nil {
		att["affinity"] = flattenAffinity(in.Affinity)
	}

	if in.AutomountServiceAccountToken != nil {
		att["automount_service_account_token"] = *in.AutomountServiceAccountToken
	}

	// To avoid perpetual diff, remove the service account token volume from PodSpec.
	serviceAccountName := "default"
	if in.ServiceAccountName != "" {
		serviceAccountName = in.ServiceAccountName
	}
	serviceAccountRegex := fmt.Sprintf("%s-token-([a-z0-9]{5})", serviceAccountName)

	containers, err := flattenContainers(in.Containers, serviceAccountRegex)
	if err != nil {
		return nil, err
	}
	att["container"] = containers

	att["readiness_gate"] = flattenReadinessGates(in.ReadinessGates)

	initContainers, err := flattenContainers(in.InitContainers, serviceAccountRegex)
	if err != nil {
		return nil, err
	}
	att["init_container"] = initContainers

	att["dns_policy"] = in.DNSPolicy
	if in.DNSConfig != nil {
		v, err := flattenPodDNSConfig(in.DNSConfig)
		if err != nil {
			return []interface{}{att}, err
		}
		att["dns_config"] = v
	}

	if in.EnableServiceLinks != nil {
		att["enable_service_links"] = *in.EnableServiceLinks
	}

	att["host_aliases"] = flattenHostaliases(in.HostAliases)

	att["host_ipc"] = in.HostIPC
	att["host_network"] = in.HostNetwork
	att["host_pid"] = in.HostPID

	if in.Hostname != "" {
		att["hostname"] = in.Hostname
	}
	att["image_pull_secrets"] = flattenLocalObjectReferenceArray(in.ImagePullSecrets)

	if in.OS != nil {
		att["os"] = flattenOS(*in.OS)
	}

	if in.NodeName != "" {
		att["node_name"] = in.NodeName
	}
	if len(in.NodeSelector) > 0 {
		att["node_selector"] = in.NodeSelector
	}
	if in.RuntimeClassName != nil {
		att["runtime_class_name"] = *in.RuntimeClassName
	}
	if in.PriorityClassName != "" {
		att["priority_class_name"] = in.PriorityClassName
	}
	if in.RestartPolicy != "" {
		att["restart_policy"] = in.RestartPolicy
	}

	if in.SecurityContext != nil {
		att["security_context"] = flattenPodSecurityContext(in.SecurityContext)
	}

	if in.SchedulerName != "" {
		att["scheduler_name"] = in.SchedulerName
	}

	if in.ServiceAccountName != "" {
		att["service_account_name"] = in.ServiceAccountName
	}
	if in.ShareProcessNamespace != nil {
		att["share_process_namespace"] = *in.ShareProcessNamespace
	}

	if in.Subdomain != "" {
		att["subdomain"] = in.Subdomain
	}

	if in.TerminationGracePeriodSeconds != nil {
		att["termination_grace_period_seconds"] = *in.TerminationGracePeriodSeconds
	}

	if len(in.Tolerations) > 0 {
		att["toleration"] = flattenTolerations(in.Tolerations)
	}

	if len(in.TopologySpreadConstraints) > 0 {
		att["topology_spread_constraint"] = flattenTopologySpreadConstraints(in.TopologySpreadConstraints)
	}

	if len(in.Volumes) > 0 {
		for i, volume := range in.Volumes {
			// To avoid perpetual diff, remove the service account token volume from PodSpec.
			nameMatchesDefaultToken, err := regexp.MatchString(serviceAccountRegex, volume.Name)
			if err != nil {
				return []interface{}{att}, err
			}
			if nameMatchesDefaultToken || strings.HasPrefix(volume.Name, "kube-api-access") {
				in.Volumes = removeVolumeFromPodSpec(i, in.Volumes)
				break
			}
		}

		att["volume"] = flattenVolumes(in.Volumes)
	}
	return []interface{}{att}, nil
}

// removeVolumeFromPodSpec removes the specified Volume index (i) from the given list of Volumes.
func removeVolumeFromPodSpec(i int, v []v1.Volume) []v1.Volume {
	return append(v[:i], v[i+1:]...)
}

func flattenPodDNSConfig(in *v1.PodDNSConfig) ([]interface{}, error) {
	att := make(map[string]interface{})

	if len(in.Nameservers) > 0 {
		att["nameservers"] = in.Nameservers
	}
	if len(in.Searches) > 0 {
		att["searches"] = in.Searches
	}
	if len(in.Options) > 0 {
		att["option"] = flattenPodDNSConfigOptions(in.Options)
	}

	if len(att) > 0 {
		return []interface{}{att}, nil
	}
	return []interface{}{}, nil
}

func flattenPodDNSConfigOptions(options []v1.PodDNSConfigOption) []interface{} {
	att := make([]interface{}, len(options))
	for i, v := range options {
		obj := map[string]interface{}{}

		if v.Name != "" {
			obj["name"] = v.Name
		}
		if v.Value != nil {
			obj["value"] = *v.Value
		}
		att[i] = obj
	}
	return att
}

func flattenPodSecurityContext(in *v1.PodSecurityContext) []interface{} {
	att := make(map[string]interface{})

	if in.FSGroup != nil {
		att["fs_group"] = strconv.Itoa(int(*in.FSGroup))
	}
	if in.RunAsGroup != nil {
		att["run_as_group"] = strconv.Itoa(int(*in.RunAsGroup))
	}
	if in.RunAsNonRoot != nil {
		att["run_as_non_root"] = *in.RunAsNonRoot
	}
	if in.RunAsUser != nil {
		att["run_as_user"] = strconv.Itoa(int(*in.RunAsUser))
	}
	if in.SeccompProfile != nil {
		att["seccomp_profile"] = flattenSeccompProfile(in.SeccompProfile)
	}
	if in.FSGroupChangePolicy != nil {
		att["fs_group_change_policy"] = *in.FSGroupChangePolicy
	}
	if len(in.SupplementalGroups) > 0 {
		att["supplemental_groups"] = newInt64Set(schema.HashSchema(&schema.Schema{
			Type: schema.TypeInt,
		}), in.SupplementalGroups)
	}
	if in.SELinuxOptions != nil {
		att["se_linux_options"] = flattenSeLinuxOptions(in.SELinuxOptions)
	}
	if in.Sysctls != nil {
		att["sysctl"] = flattenSysctls(in.Sysctls)
	}

	if in.WindowsOptions != nil {
		att["windows_options"] = flattenWindowsOptions(*in.WindowsOptions)
	}

	if len(att) > 0 {
		return []interface{}{att}
	}
	return []interface{}{}
}

func flattenSeccompProfile(in *v1.SeccompProfile) []interface{} {
	att := make(map[string]interface{})
	if in.Type != "" {
		att["type"] = in.Type
		if in.Type == "Localhost" {
			att["localhost_profile"] = in.LocalhostProfile
		}
	}
	return []interface{}{att}
}

func flattenSeLinuxOptions(in *v1.SELinuxOptions) []interface{} {
	att := make(map[string]interface{})
	if in.User != "" {
		att["user"] = in.User
	}
	if in.Role != "" {
		att["role"] = in.Role
	}
	if in.Type != "" {
		att["type"] = in.Type
	}
	if in.Level != "" {
		att["level"] = in.Level
	}
	return []interface{}{att}
}

func flattenSysctls(sysctls []v1.Sysctl) []interface{} {
	att := []interface{}{}
	for _, v := range sysctls {
		obj := map[string]interface{}{}

		if v.Name != "" {
			obj["name"] = v.Name
		}
		if v.Value != "" {
			obj["value"] = v.Value
		}
		att = append(att, obj)
	}
	return att
}

func flattenTolerations(tolerations []v1.Toleration) []interface{} {
	att := []interface{}{}
	for _, v := range tolerations {
		// The API Server may automatically add several Tolerations to pods, strip these to avoid TF diff.
		if _, ok := builtInTolerations[v.Key]; ok {
			log.Printf("[INFO] ignoring toleration with key: %s", v.Key)
			continue
		}
		obj := map[string]interface{}{}

		if v.Effect != "" {
			obj["effect"] = string(v.Effect)
		}
		if v.Key != "" {
			obj["key"] = v.Key
		}
		if v.Operator != "" {
			obj["operator"] = string(v.Operator)
		}
		if v.TolerationSeconds != nil {
			obj["toleration_seconds"] = strconv.FormatInt(*v.TolerationSeconds, 10)
		}
		if v.Value != "" {
			obj["value"] = v.Value
		}
		att = append(att, obj)
	}
	return att
}

func flattenTopologySpreadConstraints(tsc []v1.TopologySpreadConstraint) []interface{} {
	att := []interface{}{}
	for _, v := range tsc {
		obj := map[string]interface{}{}

		if v.TopologyKey != "" {
			obj["topology_key"] = v.TopologyKey
		}
		if v.MaxSkew != 0 {
			obj["max_skew"] = v.MaxSkew
		}
		if v.WhenUnsatisfiable != "" {
			obj["when_unsatisfiable"] = string(v.WhenUnsatisfiable)
		}
		if v.LabelSelector != nil {
			obj["label_selector"] = flattenLabelSelector(v.LabelSelector)
		}
		att = append(att, obj)
	}
	return att
}

func flattenVolumes(volumes []v1.Volume) []interface{} {
	att := make([]interface{}, len(volumes))
	for i, v := range volumes {
		obj := map[string]interface{}{}

		if v.Name != "" {
			obj["name"] = v.Name
		}
		if v.ConfigMap != nil {
			obj["config_map"] = flattenConfigMapVolumeSource(v.ConfigMap)
		}
		if v.GitRepo != nil {
			obj["git_repo"] = flattenGitRepoVolumeSource(v.GitRepo)
		}
		if v.EmptyDir != nil {
			obj["empty_dir"] = flattenEmptyDirVolumeSource(v.EmptyDir)
		}
		if v.DownwardAPI != nil {
			obj["downward_api"] = flattenDownwardAPIVolumeSource(v.DownwardAPI)
		}
		if v.PersistentVolumeClaim != nil {
			obj["persistent_volume_claim"] = flattenPersistentVolumeClaimVolumeSource(v.PersistentVolumeClaim)
		}
		if v.Secret != nil {
			obj["secret"] = flattenSecretVolumeSource(v.Secret)
		}
		if v.Projected != nil {
			obj["projected"] = flattenProjectedVolumeSource(v.Projected)
		}
		if v.GCEPersistentDisk != nil {
			obj["gce_persistent_disk"] = flattenGCEPersistentDiskVolumeSource(v.GCEPersistentDisk)
		}
		if v.AWSElasticBlockStore != nil {
			obj["aws_elastic_block_store"] = flattenAWSElasticBlockStoreVolumeSource(v.AWSElasticBlockStore)
		}
		if v.HostPath != nil {
			obj["host_path"] = flattenHostPathVolumeSource(v.HostPath)
		}
		if v.Glusterfs != nil {
			obj["glusterfs"] = flattenGlusterfsVolumeSource(v.Glusterfs)
		}
		if v.NFS != nil {
			obj["nfs"] = flattenNFSVolumeSource(v.NFS)
		}
		if v.RBD != nil {
			obj["rbd"] = flattenRBDVolumeSource(v.RBD)
		}
		if v.ISCSI != nil {
			obj["iscsi"] = flattenISCSIVolumeSource(v.ISCSI)
		}
		if v.Cinder != nil {
			obj["cinder"] = flattenCinderVolumeSource(v.Cinder)
		}
		if v.CephFS != nil {
			obj["ceph_fs"] = flattenCephFSVolumeSource(v.CephFS)
		}
		if v.CSI != nil {
			obj["csi"] = flattenCSIVolumeSource(v.CSI)
		}
		if v.FC != nil {
			obj["fc"] = flattenFCVolumeSource(v.FC)
		}
		if v.Flocker != nil {
			obj["flocker"] = flattenFlockerVolumeSource(v.Flocker)
		}
		if v.FlexVolume != nil {
			obj["flex_volume"] = flattenFlexVolumeSource(v.FlexVolume)
		}
		if v.AzureFile != nil {
			obj["azure_file"] = flattenAzureFileVolumeSource(v.AzureFile)
		}
		if v.VsphereVolume != nil {
			obj["vsphere_volume"] = flattenVsphereVirtualDiskVolumeSource(v.VsphereVolume)
		}
		if v.Quobyte != nil {
			obj["quobyte"] = flattenQuobyteVolumeSource(v.Quobyte)
		}
		if v.AzureDisk != nil {
			obj["azure_disk"] = flattenAzureDiskVolumeSource(v.AzureDisk)
		}
		if v.PhotonPersistentDisk != nil {
			obj["photon_persistent_disk"] = flattenPhotonPersistentDiskVolumeSource(v.PhotonPersistentDisk)
		}
		if v.Ephemeral != nil {
			obj["ephemeral"] = flattenPodEphemeralVolumeSource(v.Ephemeral)
		}
		att[i] = obj
	}
	return att
}

func flattenPersistentVolumeClaimVolumeSource(in *v1.PersistentVolumeClaimVolumeSource) []interface{} {
	att := make(map[string]interface{})
	if in.ClaimName != "" {
		att["claim_name"] = in.ClaimName
	}
	if in.ReadOnly {
		att["read_only"] = in.ReadOnly
	}

	return []interface{}{att}
}
func flattenGitRepoVolumeSource(in *v1.GitRepoVolumeSource) []interface{} {
	att := make(map[string]interface{})
	if in.Directory != "" {
		att["directory"] = in.Directory
	}

	att["repository"] = in.Repository

	if in.Revision != "" {
		att["revision"] = in.Revision
	}
	return []interface{}{att}
}

func flattenDownwardAPIVolumeSource(in *v1.DownwardAPIVolumeSource) []interface{} {
	att := make(map[string]interface{})
	if in.DefaultMode != nil {
		att["default_mode"] = "0" + strconv.FormatInt(int64(*in.DefaultMode), 8)
	}
	if len(in.Items) > 0 {
		att["items"] = flattenDownwardAPIVolumeFile(in.Items)
	}
	return []interface{}{att}
}

func flattenDownwardAPIVolumeFile(in []v1.DownwardAPIVolumeFile) []interface{} {
	att := make([]interface{}, len(in))
	for i, v := range in {
		m := map[string]interface{}{}
		if v.FieldRef != nil {
			m["field_ref"] = flattenObjectFieldSelector(v.FieldRef)
		}
		if v.Mode != nil {
			m["mode"] = "0" + strconv.FormatInt(int64(*v.Mode), 8)
		}
		if v.Path != "" {
			m["path"] = v.Path
		}
		if v.ResourceFieldRef != nil {
			m["resource_field_ref"] = flattenResourceFieldSelector(v.ResourceFieldRef)
		}
		att[i] = m
	}
	return att
}

func flattenConfigMapVolumeSource(in *v1.ConfigMapVolumeSource) []interface{} {
	att := make(map[string]interface{})
	if in.DefaultMode != nil {
		att["default_mode"] = "0" + strconv.FormatInt(int64(*in.DefaultMode), 8)
	}
	att["name"] = in.Name
	if len(in.Items) > 0 {
		items := make([]interface{}, len(in.Items))
		for i, v := range in.Items {
			m := map[string]interface{}{}
			if v.Key != "" {
				m["key"] = v.Key
			}
			if v.Mode != nil {
				m["mode"] = "0" + strconv.FormatInt(int64(*v.Mode), 8)
			}
			if v.Path != "" {
				m["path"] = v.Path
			}
			items[i] = m
		}
		att["items"] = items
	}
	if in.Optional != nil {
		att["optional"] = *in.Optional
	}
	return []interface{}{att}
}

func flattenEmptyDirVolumeSource(in *v1.EmptyDirVolumeSource) []interface{} {
	att := make(map[string]interface{})
	att["medium"] = string(in.Medium)
	if in.SizeLimit != nil {
		att["size_limit"] = in.SizeLimit.String()
	}
	return []interface{}{att}
}

func flattenSecretVolumeSource(in *v1.SecretVolumeSource) []interface{} {
	att := make(map[string]interface{})
	if in.DefaultMode != nil {
		att["default_mode"] = "0" + strconv.FormatInt(int64(*in.DefaultMode), 8)
	}
	if in.SecretName != "" {
		att["secret_name"] = in.SecretName
	}
	if len(in.Items) > 0 {
		items := make([]interface{}, len(in.Items))
		for i, v := range in.Items {
			m := map[string]interface{}{}
			m["key"] = v.Key
			if v.Mode != nil {
				m["mode"] = "0" + strconv.FormatInt(int64(*v.Mode), 8)
			}
			m["path"] = v.Path
			items[i] = m
		}
		att["items"] = items
	}
	if in.Optional != nil {
		att["optional"] = *in.Optional
	}
	return []interface{}{att}
}

func flattenProjectedVolumeSource(in *v1.ProjectedVolumeSource) []interface{} {
	att := make(map[string]interface{})
	if in.DefaultMode != nil {
		att["default_mode"] = "0" + strconv.FormatInt(int64(*in.DefaultMode), 8)
	}
	if len(in.Sources) > 0 {
		sources := make([]interface{}, 0, len(in.Sources))
		for _, src := range in.Sources {
			s := make(map[string]interface{})
			if src.Secret != nil {
				s["secret"] = flattenSecretProjection(src.Secret)
			}
			if src.ConfigMap != nil {
				s["config_map"] = flattenConfigMapProjection(src.ConfigMap)
			}
			if src.DownwardAPI != nil {
				s["downward_api"] = flattenDownwardAPIProjection(src.DownwardAPI)
			}
			if src.ServiceAccountToken != nil {
				s["service_account_token"] = flattenServiceAccountTokenProjection(src.ServiceAccountToken)
			}
			sources = append(sources, s)
		}
		att["sources"] = sources
	}
	return []interface{}{att}
}

func flattenSecretProjection(in *v1.SecretProjection) []interface{} {
	att := make(map[string]interface{})
	if in.Name != "" {
		att["name"] = in.Name
	}
	if len(in.Items) > 0 {
		items := make([]interface{}, len(in.Items))
		for i, v := range in.Items {
			m := map[string]interface{}{}
			m["key"] = v.Key
			if v.Mode != nil {
				m["mode"] = "0" + strconv.FormatInt(int64(*v.Mode), 8)
			}
			m["path"] = v.Path
			items[i] = m
		}
		att["items"] = items
	}
	if in.Optional != nil {
		att["optional"] = *in.Optional
	}
	return []interface{}{att}
}

func flattenConfigMapProjection(in *v1.ConfigMapProjection) []interface{} {
	att := make(map[string]interface{})
	att["name"] = in.Name
	if len(in.Items) > 0 {
		items := make([]interface{}, len(in.Items))
		for i, v := range in.Items {
			m := map[string]interface{}{}
			if v.Key != "" {
				m["key"] = v.Key
			}
			if v.Mode != nil {
				m["mode"] = "0" + strconv.FormatInt(int64(*v.Mode), 8)
			}
			if v.Path != "" {
				m["path"] = v.Path
			}
			items[i] = m
		}
		att["items"] = items
	}
	return []interface{}{att}
}

func flattenDownwardAPIProjection(in *v1.DownwardAPIProjection) []interface{} {
	att := make(map[string]interface{})
	if len(in.Items) > 0 {
		att["items"] = flattenDownwardAPIVolumeFile(in.Items)
	}
	return []interface{}{att}
}

func flattenServiceAccountTokenProjection(in *v1.ServiceAccountTokenProjection) []interface{} {
	att := make(map[string]interface{})
	if in.Audience != "" {
		att["audience"] = in.Audience
	}
	if in.ExpirationSeconds != nil {
		att["expiration_seconds"] = in.ExpirationSeconds
	}
	if in.Path != "" {
		att["path"] = in.Path
	}
	return []interface{}{att}
}

func flattenReadinessGates(in []v1.PodReadinessGate) []interface{} {
	att := make([]interface{}, len(in))
	for i, v := range in {
		c := make(map[string]interface{})
		c["condition_type"] = v.ConditionType
		att[i] = c
	}
	return att
}

func flattenPersistentVolumeClaimMetadata(in metav1.ObjectMeta) map[string]interface{} {
	att := make(map[string]interface{})

	if len(in.GetLabels()) > 0 {
		att["labels"] = in.GetLabels()
	}
	if len(in.GetAnnotations()) > 0 {
		att["annotations"] = in.GetAnnotations()
	}

	return att
}

func flattenPodEphemeralVolumeClaimTemplate(in *v1.PersistentVolumeClaimTemplate) []interface{} {
	att := make(map[string]interface{})

	m := flattenPersistentVolumeClaimMetadata(in.ObjectMeta)
	if len(m) > 0 {
		att["metadata"] = []interface{}{m}
	}

	att["spec"] = flattenPersistentVolumeClaimSpec(in.Spec)

	return []interface{}{att}
}

func flattenPodEphemeralVolumeSource(in *v1.EphemeralVolumeSource) []interface{} {
	return []interface{}{map[string]interface{}{
		"volume_claim_template": flattenPodEphemeralVolumeClaimTemplate(in.VolumeClaimTemplate),
	}}
}

// Expanders

func expandPodTargetState(p []interface{}) []string {
	if len(p) > 0 {
		t := make([]string, len(p))
		for i, v := range p {
			t[i] = v.(string)
		}
		return t
	}

	return []string{string(v1.PodRunning)}
}

func expandPodSpec(p []interface{}) (*v1.PodSpec, error) {
	obj := &v1.PodSpec{}
	if len(p) == 0 || p[0] == nil {
		return obj, nil
	}
	in := p[0].(map[string]interface{})

	if v, ok := in["active_deadline_seconds"].(int); ok && v > 0 {
		obj.ActiveDeadlineSeconds = ptr.To(int64(v))
	}

	if v, ok := in["affinity"].([]interface{}); ok && len(v) > 0 {
		a, err := expandAffinity(v)
		if err != nil {
			return obj, err
		}
		obj.Affinity = a
	}

	if v, ok := in["automount_service_account_token"].(bool); ok {
		obj.AutomountServiceAccountToken = ptr.To(v)
	}

	if v, ok := in["container"].([]interface{}); ok && len(v) > 0 {
		cs, err := expandContainers(v)
		if err != nil {
			return obj, err
		}
		obj.Containers = cs
	}

	if v, ok := in["readiness_gate"].([]interface{}); ok && len(v) > 0 {
		obj.ReadinessGates = expandReadinessGates(v)
	}

	if v, ok := in["init_container"].([]interface{}); ok && len(v) > 0 {
		cs, err := expandContainers(v)
		if err != nil {
			return obj, err
		}
		obj.InitContainers = cs
	}

	if v, ok := in["dns_policy"].(string); ok {
		obj.DNSPolicy = v1.DNSPolicy(v)
	}

	if v, ok := in["dns_config"].([]interface{}); ok && len(v) > 0 {
		dnsConfig, err := expandPodDNSConfig(v)
		if err != nil {
			return obj, nil
		}
		obj.DNSConfig = dnsConfig
	}

	if v, ok := in["enable_service_links"].(bool); ok {
		obj.EnableServiceLinks = ptr.To(v)
	}

	if v, ok := in["host_aliases"].([]interface{}); ok && len(v) > 0 {
		obj.HostAliases = expandHostaliases(v)
	}

	if v, ok := in["host_ipc"]; ok {
		obj.HostIPC = v.(bool)
	}

	if v, ok := in["host_network"]; ok {
		obj.HostNetwork = v.(bool)
	}

	if v, ok := in["host_pid"]; ok {
		obj.HostPID = v.(bool)
	}

	if v, ok := in["hostname"]; ok {
		obj.Hostname = v.(string)
	}

	if v, ok := in["image_pull_secrets"].([]interface{}); ok {
		cs := expandLocalObjectReferenceArray(v)
		obj.ImagePullSecrets = cs
	}

	if v, ok := in["node_name"]; ok {
		obj.NodeName = v.(string)
	}

	if v, ok := in["node_selector"].(map[string]interface{}); ok {
		nodeSelectors := make(map[string]string)
		for k, v := range v {
			if val, ok := v.(string); ok {
				nodeSelectors[k] = val
			}
		}
		obj.NodeSelector = nodeSelectors
	}

	if v, ok := in["os"].([]interface{}); ok && len(v) != 0 {
		obj.OS = expandOS(v)
	}

	if v, ok := in["runtime_class_name"].(string); ok && v != "" {
		obj.RuntimeClassName = ptr.To(v)
	}

	if v, ok := in["priority_class_name"].(string); ok {
		obj.PriorityClassName = v
	}

	if v, ok := in["restart_policy"].(string); ok {
		obj.RestartPolicy = v1.RestartPolicy(v)
	}

	if v, ok := in["security_context"].([]interface{}); ok && len(v) > 0 {
		ctx, err := expandPodSecurityContext(v)
		if err != nil {
			return obj, err
		}
		obj.SecurityContext = ctx
	}

	if v, ok := in["scheduler_name"].(string); ok {
		obj.SchedulerName = v
	}

	if v, ok := in["service_account_name"].(string); ok {
		obj.ServiceAccountName = v
	}

	if v, ok := in["share_process_namespace"]; ok {
		obj.ShareProcessNamespace = ptr.To(v.(bool))
	}

	if v, ok := in["subdomain"].(string); ok {
		obj.Subdomain = v
	}

	if v, ok := in["termination_grace_period_seconds"].(int); ok {
		obj.TerminationGracePeriodSeconds = ptr.To(int64(v))
	}

	if v, ok := in["toleration"].([]interface{}); ok && len(v) > 0 {
		ts, err := expandTolerations(v)
		if err != nil {
			return obj, err
		}
		for _, t := range ts {
			obj.Tolerations = append(obj.Tolerations, *t)
		}
	}

	if v, ok := in["volume"].([]interface{}); ok && len(v) > 0 {
		cs, err := expandVolumes(v)
		if err != nil {
			return obj, err
		}
		obj.Volumes = cs
	}

	if v, ok := in["topology_spread_constraint"].([]interface{}); ok && len(v) > 0 {
		obj.TopologySpreadConstraints = expandTopologySpreadConstraints(v)
	}

	return obj, nil
}

func expandOS(l []interface{}) *v1.PodOS {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	in := l[0].(map[string]interface{})

	return &v1.PodOS{
		Name: v1.OSName(in["name"].(string)),
	}
}

func expandWindowsOptions(l []interface{}) *v1.WindowsSecurityContextOptions {
	if len(l) == 0 || l[0] == nil {
		return &v1.WindowsSecurityContextOptions{}
	}

	in := l[0].(map[string]interface{})
	obj := &v1.WindowsSecurityContextOptions{}

	if v, ok := in["gmsa_credential_spec"].(string); ok {
		obj.GMSACredentialSpec = ptr.To(v)
	}

	if v, ok := in["host_process"].(bool); ok {
		obj.HostProcess = ptr.To(v)
	}

	if v, ok := in["gmsa_credential_spec_name"].(string); ok {
		obj.GMSACredentialSpecName = ptr.To(v)
	}

	if v, ok := in["run_as_username"].(string); ok {
		obj.RunAsUserName = ptr.To(v)
	}

	return obj
}

func flattenWindowsOptions(in v1.WindowsSecurityContextOptions) []interface{} {
	att := make(map[string]interface{})

	if in.GMSACredentialSpec != nil {
		att["gmsa_credential_spec"] = *in.GMSACredentialSpec
	}

	if in.GMSACredentialSpecName != nil {
		att["gmsa_credential_spec_name"] = *in.GMSACredentialSpecName
	}

	if in.HostProcess != nil {
		att["host_process"] = *in.HostProcess
	}

	if in.RunAsUserName != nil {
		att["run_as_username"] = *in.RunAsUserName
	}

	return []interface{}{att}
}

func expandPodDNSConfig(l []interface{}) (*v1.PodDNSConfig, error) {
	if len(l) == 0 || l[0] == nil {
		return &v1.PodDNSConfig{}, nil
	}
	in := l[0].(map[string]interface{})
	obj := &v1.PodDNSConfig{}
	if v, ok := in["nameservers"].([]interface{}); ok {
		obj.Nameservers = expandStringSlice(v)
	}
	if v, ok := in["searches"].([]interface{}); ok {
		obj.Searches = expandStringSlice(v)
	}
	if v, ok := in["option"].([]interface{}); ok {
		obj.Options = expandDNSConfigOptions(v)
	}
	return obj, nil
}

func expandDNSConfigOptions(options []interface{}) []v1.PodDNSConfigOption {
	if len(options) == 0 {
		return []v1.PodDNSConfigOption{}
	}
	opts := make([]v1.PodDNSConfigOption, len(options))
	for i, c := range options {
		in := c.(map[string]interface{})
		opt := v1.PodDNSConfigOption{}
		if v, ok := in["name"].(string); ok {
			opt.Name = v
		}
		if v, ok := in["value"].(string); ok {
			opt.Value = ptr.To(v)
		}
		opts[i] = opt
	}

	return opts
}

func expandPodSecurityContext(l []interface{}) (*v1.PodSecurityContext, error) {
	obj := &v1.PodSecurityContext{}
	if len(l) == 0 || l[0] == nil {
		return obj, nil
	}
	in := l[0].(map[string]interface{})
	if v, ok := in["fs_group"].(string); ok && v != "" {
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return obj, err
		}
		obj.FSGroup = ptr.To(int64(i))
	}
	if v, ok := in["run_as_group"].(string); ok && v != "" {
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return obj, err
		}
		obj.RunAsGroup = ptr.To(int64(i))
	}
	if v, ok := in["run_as_non_root"].(bool); ok {
		obj.RunAsNonRoot = ptr.To(v)
	}
	if v, ok := in["run_as_user"].(string); ok && v != "" {
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return obj, err
		}
		obj.RunAsUser = ptr.To(int64(i))
	}
	if v, ok := in["seccomp_profile"].([]interface{}); ok && len(v) > 0 {
		obj.SeccompProfile = expandSeccompProfile(v)
	}
	if v, ok := in["se_linux_options"].([]interface{}); ok && len(v) > 0 {
		obj.SELinuxOptions = expandSeLinuxOptions(v)
	}
	if v, ok := in["supplemental_groups"].(*schema.Set); ok {
		obj.SupplementalGroups = schemaSetToInt64Array(v)
	}
	if v, ok := in["sysctl"].([]interface{}); ok && len(v) > 0 {
		obj.Sysctls = expandSysctls(v)
	}
	if v, ok := in["fs_group_change_policy"].(string); ok && v != "" {
		policy := v1.PodFSGroupChangePolicy(v)
		obj.FSGroupChangePolicy = &policy
	}
	if v, ok := in["windows_options"].([]interface{}); ok && len(v) > 0 {
		obj.WindowsOptions = expandWindowsOptions(v)
	}
	return obj, nil
}

func expandSysctls(l []interface{}) []v1.Sysctl {
	if len(l) == 0 {
		return []v1.Sysctl{}
	}
	sysctls := make([]v1.Sysctl, len(l))
	for i, c := range l {
		p := c.(map[string]interface{})
		if v, ok := p["name"].(string); ok {
			sysctls[i].Name = v
		}
		if v, ok := p["value"].(string); ok {
			sysctls[i].Value = v
		}

	}
	return sysctls
}

func expandSeccompProfile(l []interface{}) *v1.SeccompProfile {
	if len(l) == 0 || l[0] == nil {
		return &v1.SeccompProfile{}
	}
	in := l[0].(map[string]interface{})
	obj := &v1.SeccompProfile{}
	if v, ok := in["type"].(string); ok {
		obj.Type = v1.SeccompProfileType(v)
		if v == "Localhost" {
			if lp, ok := in["localhost_profile"].(string); ok {
				obj.LocalhostProfile = &lp
			}
		}
	}
	return obj
}

func expandSeLinuxOptions(l []interface{}) *v1.SELinuxOptions {
	if len(l) == 0 || l[0] == nil {
		return &v1.SELinuxOptions{}
	}
	in := l[0].(map[string]interface{})
	obj := &v1.SELinuxOptions{}
	if v, ok := in["level"]; ok {
		obj.Level = v.(string)
	}
	if v, ok := in["role"]; ok {
		obj.Role = v.(string)
	}
	if v, ok := in["type"]; ok {
		obj.Type = v.(string)
	}
	if v, ok := in["user"]; ok {
		obj.User = v.(string)
	}
	return obj
}

func expandKeyPath(in []interface{}) []v1.KeyToPath {
	if len(in) == 0 {
		return []v1.KeyToPath{}
	}
	keyPaths := make([]v1.KeyToPath, len(in))
	for i, c := range in {
		p := c.(map[string]interface{})
		if v, ok := p["key"].(string); ok {
			keyPaths[i].Key = v
		}
		if v, ok := p["mode"].(string); ok {
			m, err := strconv.ParseInt(v, 8, 32)
			if err == nil {
				keyPaths[i].Mode = ptr.To(int32(m))
			}
		}
		if v, ok := p["path"].(string); ok {
			keyPaths[i].Path = v
		}

	}
	return keyPaths
}

func expandDownwardAPIVolumeFile(in []interface{}) ([]v1.DownwardAPIVolumeFile, error) {
	var err error
	if len(in) == 0 {
		return []v1.DownwardAPIVolumeFile{}, nil
	}
	dapivf := make([]v1.DownwardAPIVolumeFile, len(in))
	for i, c := range in {
		p := c.(map[string]interface{})
		if mode, ok := p["mode"].(string); ok && len(mode) > 0 {
			m, err := strconv.ParseInt(mode, 8, 32)
			if err != nil {
				return dapivf, fmt.Errorf("DownwardAPI volume file: failed to parse 'mode' value: %s", err)
			}
			dapivf[i].Mode = ptr.To(int32(m))
		}
		if v, ok := p["path"].(string); ok {
			dapivf[i].Path = v
		}
		if v, ok := p["field_ref"].([]interface{}); ok && len(v) > 0 {
			dapivf[i].FieldRef = expandFieldRef(v)
		}
		if v, ok := p["resource_field_ref"].([]interface{}); ok && len(v) > 0 {
			dapivf[i].ResourceFieldRef, err = expandResourceFieldRef(v)
			if err != nil {
				return dapivf, err
			}
		}
	}
	return dapivf, nil
}

func expandConfigMapVolumeSource(l []interface{}) (*v1.ConfigMapVolumeSource, error) {
	obj := &v1.ConfigMapVolumeSource{}
	if len(l) == 0 || l[0] == nil {
		return obj, nil
	}
	in := l[0].(map[string]interface{})

	if mode, ok := in["default_mode"].(string); ok && len(mode) > 0 {
		v, err := strconv.ParseInt(mode, 8, 32)
		if err != nil {
			return obj, fmt.Errorf("ConfigMap volume: failed to parse 'default_mode' value: %s", err)
		}
		obj.DefaultMode = ptr.To(int32(v))
	}

	if v, ok := in["name"].(string); ok {
		obj.Name = v
	}

	if opt, ok := in["optional"].(bool); ok {
		obj.Optional = ptr.To(opt)
	}
	if v, ok := in["items"].([]interface{}); ok && len(v) > 0 {
		obj.Items = expandKeyPath(v)
	}

	return obj, nil
}

func expandDownwardAPIVolumeSource(l []interface{}) (*v1.DownwardAPIVolumeSource, error) {
	obj := &v1.DownwardAPIVolumeSource{}
	if len(l) == 0 || l[0] == nil {
		return obj, nil
	}
	in := l[0].(map[string]interface{})

	if mode, ok := in["default_mode"].(string); ok && len(mode) > 0 {
		v, err := strconv.ParseInt(mode, 8, 32)
		if err != nil {
			return obj, fmt.Errorf("Downward API volume: failed to parse 'default_mode' value: %s", err)
		}
		obj.DefaultMode = ptr.To(int32(v))
	}

	if v, ok := in["items"].([]interface{}); ok && len(v) > 0 {
		var err error
		obj.Items, err = expandDownwardAPIVolumeFile(v)
		if err != nil {
			return obj, err
		}
	}
	return obj, nil
}

func expandGitRepoVolumeSource(l []interface{}) *v1.GitRepoVolumeSource {
	if len(l) == 0 || l[0] == nil {
		return &v1.GitRepoVolumeSource{}
	}
	in := l[0].(map[string]interface{})
	obj := &v1.GitRepoVolumeSource{}

	if v, ok := in["directory"].(string); ok {
		obj.Directory = v
	}

	if v, ok := in["repository"].(string); ok {
		obj.Repository = v
	}
	if v, ok := in["revision"].(string); ok {
		obj.Revision = v
	}
	return obj
}

func expandEmptyDirVolumeSource(l []interface{}) (*v1.EmptyDirVolumeSource, error) {
	if len(l) == 0 || l[0] == nil {
		return &v1.EmptyDirVolumeSource{}, nil
	}
	in := l[0].(map[string]interface{})
	obj := &v1.EmptyDirVolumeSource{
		Medium: v1.StorageMedium(in["medium"].(string)),
	}

	if v, ok := in["size_limit"].(string); ok && v != "" {
		s, err := resource.ParseQuantity(v)
		if err != nil {
			return nil, fmt.Errorf("failed to parse size_limit: %w", err)
		}
		obj.SizeLimit = &s
	}

	return obj, nil
}

func expandPersistentVolumeClaimVolumeSource(l []interface{}) *v1.PersistentVolumeClaimVolumeSource {
	if len(l) == 0 || l[0] == nil {
		return &v1.PersistentVolumeClaimVolumeSource{}
	}
	in := l[0].(map[string]interface{})
	obj := &v1.PersistentVolumeClaimVolumeSource{
		ClaimName: in["claim_name"].(string),
		ReadOnly:  in["read_only"].(bool),
	}
	return obj
}

func expandSecretVolumeSource(l []interface{}) (*v1.SecretVolumeSource, error) {
	obj := &v1.SecretVolumeSource{}
	if len(l) == 0 || l[0] == nil {
		return obj, nil
	}
	in := l[0].(map[string]interface{})

	if mode, ok := in["default_mode"].(string); ok && len(mode) > 0 {
		v, err := strconv.ParseInt(mode, 8, 32)
		if err != nil {
			return obj, fmt.Errorf("Secret volume: failed to parse 'default_mode' value: %s", err)
		}
		obj.DefaultMode = ptr.To(int32(v))
	}

	if secret, ok := in["secret_name"].(string); ok {
		obj.SecretName = secret
	}

	if opt, ok := in["optional"].(bool); ok {
		obj.Optional = ptr.To(opt)
	}
	if v, ok := in["items"].([]interface{}); ok && len(v) > 0 {
		obj.Items = expandKeyPath(v)
	}

	return obj, nil
}

func expandProjectedVolumeSource(l []interface{}) (*v1.ProjectedVolumeSource, error) {
	obj := &v1.ProjectedVolumeSource{}
	if len(l) == 0 || l[0] == nil {
		return obj, nil
	}
	in := l[0].(map[string]interface{})

	if mode, ok := in["default_mode"].(string); ok && len(mode) > 0 {
		v, err := strconv.ParseInt(mode, 8, 32)
		if err != nil {
			return obj, fmt.Errorf("Projected volume: failed to parse 'default_mode' value: %s", err)
		}
		obj.DefaultMode = ptr.To(int32(v))
	}
	if v, ok := in["sources"].([]interface{}); ok && len(v) > 0 {
		srcs, err := expandProjectedSources(v)
		if err != nil {
			return obj, fmt.Errorf("Projected volume: failed to parse 'sources' value: %s", err)
		}

		obj.Sources = srcs
	}

	return obj, nil
}

func expandProjectedSources(sources []interface{}) ([]v1.VolumeProjection, error) {
	if len(sources) == 0 || sources[0] == nil {
		return []v1.VolumeProjection{}, nil
	}
	srcs := make([]v1.VolumeProjection, 0, len(sources))
	for _, src := range sources {
		in, ok := src.(map[string]interface{})
		if !ok {
			continue
		}
		if v, ok := in["secret"].([]interface{}); ok {
			srcs = append(srcs, expandProjectedSecrets(v)...)
		}
		if v, ok := in["config_map"].([]interface{}); ok {
			srcs = append(srcs, expandProjectedConfigMaps(v)...)
		}
		if v, ok := in["downward_api"].([]interface{}); ok {
			values, err := expandProjectedDownwardAPIs(v)
			if err != nil {
				return nil, err
			}
			srcs = append(srcs, values...)
		}
		if v, ok := in["service_account_token"].([]interface{}); ok {
			values, err := expandProjectedServiceAccountTokens(v)
			if err != nil {
				return nil, err
			}
			srcs = append(srcs, values...)
		}
	}

	return srcs, nil
}

func expandProjectedSecrets(secrets []interface{}) []v1.VolumeProjection {
	out := make([]v1.VolumeProjection, 0, len(secrets))
	for _, in := range secrets {
		if v, ok := in.(map[string]interface{}); ok {
			out = append(out, v1.VolumeProjection{Secret: expandProjectedSecret(v)})
		}
	}
	return out
}

func expandProjectedSecret(secret map[string]interface{}) *v1.SecretProjection {
	s := &v1.SecretProjection{}
	if value, ok := secret["name"].(string); ok {
		s.Name = value
	}
	if values, ok := secret["items"].([]interface{}); ok {
		s.Items = expandKeyPath(values)
	}
	if value, ok := secret["optional"].(bool); ok {
		s.Optional = ptr.To(value)
	}
	return s
}

func expandProjectedConfigMaps(configMaps []interface{}) []v1.VolumeProjection {
	out := make([]v1.VolumeProjection, 0, len(configMaps))
	for _, in := range configMaps {
		if v, ok := in.(map[string]interface{}); ok {
			var vol v1.VolumeProjection
			vol.ConfigMap = expandProjectedConfigMap(v)
			out = append(out, vol)
		}
	}
	return out
}

func expandProjectedConfigMap(configMap map[string]interface{}) *v1.ConfigMapProjection {
	s := &v1.ConfigMapProjection{}
	if value, ok := configMap["name"].(string); ok {
		s.Name = value
	}
	if values, ok := configMap["items"].([]interface{}); ok {
		s.Items = expandKeyPath(values)
	}
	if value, ok := configMap["optional"].(bool); ok {
		s.Optional = ptr.To(value)
	}
	return s
}

func expandProjectedDownwardAPIs(downwardAPIs []interface{}) ([]v1.VolumeProjection, error) {
	out := make([]v1.VolumeProjection, 0, len(downwardAPIs))
	for i, in := range downwardAPIs {
		if v, ok := in.(map[string]interface{}); ok {
			downwardAPI, err := expandProjectedDownwardAPI(v)
			if err != nil {
				return nil, fmt.Errorf("expanding downward API #%d: %v", i+1, err)
			}
			out = append(out, v1.VolumeProjection{
				DownwardAPI: downwardAPI,
			})
		}
	}
	return out, nil
}

func expandProjectedDownwardAPI(downwardAPI map[string]interface{}) (*v1.DownwardAPIProjection, error) {
	s := &v1.DownwardAPIProjection{}
	if values, ok := downwardAPI["items"].([]interface{}); ok {
		v, err := expandDownwardAPIVolumeFile(values)
		if err != nil {
			return nil, err
		}
		s.Items = v
	}
	return s, nil
}

func expandProjectedServiceAccountTokens(sats []interface{}) ([]v1.VolumeProjection, error) {
	out := make([]v1.VolumeProjection, 0, len(sats))
	for _, in := range sats {
		if v, ok := in.(map[string]interface{}); ok {
			out = append(out, v1.VolumeProjection{
				ServiceAccountToken: expandProjectedServiceAccountToken(v),
			})
		}
	}
	return out, nil
}

func expandProjectedServiceAccountToken(sat map[string]interface{}) *v1.ServiceAccountTokenProjection {
	s := &v1.ServiceAccountTokenProjection{}
	if value, ok := sat["audience"].(string); ok {
		s.Audience = value
	}
	if value, ok := sat["expiration_seconds"].(int); ok {
		s.ExpirationSeconds = ptr.To(int64(value))
	}
	if value, ok := sat["path"].(string); ok {
		s.Path = value
	}
	return s
}

func expandTolerations(tolerations []interface{}) ([]*v1.Toleration, error) {
	if len(tolerations) == 0 {
		return []*v1.Toleration{}, nil
	}
	ts := make([]*v1.Toleration, len(tolerations))
	for i, t := range tolerations {
		m := t.(map[string]interface{})
		ts[i] = &v1.Toleration{}

		if value, ok := m["effect"].(string); ok {
			ts[i].Effect = v1.TaintEffect(value)
		}
		if value, ok := m["key"].(string); ok {
			ts[i].Key = value
		}
		if value, ok := m["operator"].(string); ok {
			ts[i].Operator = v1.TolerationOperator(value)
		}
		if value, ok := m["toleration_seconds"].(string); ok && value != "" {
			seconds, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid toleration_seconds must be int or \"\", got \"%s\"", value)
			}
			ts[i].TolerationSeconds = ptr.To(seconds)
		}
		if value, ok := m["value"]; ok {
			ts[i].Value = value.(string)
		}
	}
	return ts, nil
}

func expandTopologySpreadConstraints(tsc []interface{}) []v1.TopologySpreadConstraint {
	if len(tsc) == 0 {
		return []v1.TopologySpreadConstraint{}
	}
	ts := make([]v1.TopologySpreadConstraint, len(tsc))
	for i, t := range tsc {
		m := t.(map[string]interface{})
		ts[i] = v1.TopologySpreadConstraint{}

		if value, ok := m["topology_key"].(string); ok {
			ts[i].TopologyKey = value
		}

		if v, ok := m["label_selector"].([]interface{}); ok && len(v) > 0 {
			ts[i].LabelSelector = expandLabelSelector(v)
		}

		if value, ok := m["when_unsatisfiable"].(string); ok {
			ts[i].WhenUnsatisfiable = v1.UnsatisfiableConstraintAction(value)
		}

		if value, ok := m["max_skew"].(int); ok {
			ts[i].MaxSkew = int32(value)
		}

	}
	return ts
}

func expandVolumes(volumes []interface{}) ([]v1.Volume, error) {
	if len(volumes) == 0 {
		return []v1.Volume{}, nil
	}
	vl := make([]v1.Volume, len(volumes))
	for i, c := range volumes {
		m := c.(map[string]interface{})

		if value, ok := m["name"]; ok {
			vl[i].Name = value.(string)
		}

		if value, ok := m["config_map"].([]interface{}); ok && len(value) > 0 {
			cfm, err := expandConfigMapVolumeSource(value)
			vl[i].ConfigMap = cfm
			if err != nil {
				return vl, err
			}
		}
		if value, ok := m["git_repo"].([]interface{}); ok && len(value) > 0 {
			vl[i].GitRepo = expandGitRepoVolumeSource(value)
		}

		if value, ok := m["empty_dir"].([]interface{}); ok && len(value) > 0 {
			var err error
			vl[i].EmptyDir, err = expandEmptyDirVolumeSource(value)
			if err != nil {
				return vl, err
			}
		}
		if value, ok := m["downward_api"].([]interface{}); ok && len(value) > 0 {
			var err error
			vl[i].DownwardAPI, err = expandDownwardAPIVolumeSource(value)
			if err != nil {
				return vl, err
			}
		}

		if value, ok := m["persistent_volume_claim"].([]interface{}); ok && len(value) > 0 {
			vl[i].PersistentVolumeClaim = expandPersistentVolumeClaimVolumeSource(value)
		}
		if value, ok := m["secret"].([]interface{}); ok && len(value) > 0 {
			sc, err := expandSecretVolumeSource(value)
			if err != nil {
				return vl, err
			}
			vl[i].Secret = sc
		}
		if value, ok := m["projected"].([]interface{}); ok && len(value) > 0 {
			pj, err := expandProjectedVolumeSource(value)
			if err != nil {
				return vl, err
			}
			vl[i].Projected = pj
		}
		if v, ok := m["gce_persistent_disk"].([]interface{}); ok && len(v) > 0 {
			vl[i].GCEPersistentDisk = expandGCEPersistentDiskVolumeSource(v)
		}
		if v, ok := m["aws_elastic_block_store"].([]interface{}); ok && len(v) > 0 {
			vl[i].AWSElasticBlockStore = expandAWSElasticBlockStoreVolumeSource(v)
		}
		if v, ok := m["host_path"].([]interface{}); ok && len(v) > 0 {
			vl[i].HostPath = expandHostPathVolumeSource(v)
		}
		if v, ok := m["glusterfs"].([]interface{}); ok && len(v) > 0 {
			vl[i].Glusterfs = expandGlusterfsVolumeSource(v)
		}
		if v, ok := m["nfs"].([]interface{}); ok && len(v) > 0 {
			vl[i].NFS = expandNFSVolumeSource(v)
		}
		if v, ok := m["rbd"].([]interface{}); ok && len(v) > 0 {
			vl[i].RBD = expandRBDVolumeSource(v)
		}
		if v, ok := m["iscsi"].([]interface{}); ok && len(v) > 0 {
			vl[i].ISCSI = expandISCSIVolumeSource(v)
		}
		if v, ok := m["cinder"].([]interface{}); ok && len(v) > 0 {
			vl[i].Cinder = expandCinderVolumeSource(v)
		}
		if v, ok := m["ceph_fs"].([]interface{}); ok && len(v) > 0 {
			vl[i].CephFS = expandCephFSVolumeSource(v)
		}
		if v, ok := m["csi"].([]interface{}); ok && len(v) > 0 {
			vl[i].CSI = expandCSIVolumeSource(v)
		}
		if v, ok := m["fc"].([]interface{}); ok && len(v) > 0 {
			vl[i].FC = expandFCVolumeSource(v)
		}
		if v, ok := m["flocker"].([]interface{}); ok && len(v) > 0 {
			vl[i].Flocker = expandFlockerVolumeSource(v)
		}
		if v, ok := m["flex_volume"].([]interface{}); ok && len(v) > 0 {
			vl[i].FlexVolume = expandFlexVolumeSource(v)
		}
		if v, ok := m["azure_file"].([]interface{}); ok && len(v) > 0 {
			vl[i].AzureFile = expandAzureFileVolumeSource(v)
		}
		if v, ok := m["vsphere_volume"].([]interface{}); ok && len(v) > 0 {
			vl[i].VsphereVolume = expandVsphereVirtualDiskVolumeSource(v)
		}
		if v, ok := m["quobyte"].([]interface{}); ok && len(v) > 0 {
			vl[i].Quobyte = expandQuobyteVolumeSource(v)
		}
		if v, ok := m["azure_disk"].([]interface{}); ok && len(v) > 0 {
			vl[i].AzureDisk = expandAzureDiskVolumeSource(v)
		}
		if v, ok := m["photon_persistent_disk"].([]interface{}); ok && len(v) > 0 {
			vl[i].PhotonPersistentDisk = expandPhotonPersistentDiskVolumeSource(v)
		}
		if v, ok := m["ephemeral"].([]interface{}); ok && len(v) > 0 {
			ephemeral, err := expandEphemeralVolumeSource(v)
			if err != nil {
				return vl, err
			}
			vl[i].Ephemeral = ephemeral
		}
	}
	return vl, nil
}

func expandReadinessGates(gates []interface{}) []v1.PodReadinessGate {
	if len(gates) == 0 || gates[0] == nil {
		return []v1.PodReadinessGate{}
	}
	cs := make([]v1.PodReadinessGate, len(gates))
	for i, c := range gates {
		gate := c.(map[string]interface{})

		if v, ok := gate["condition_type"]; ok {
			conType := v1.PodConditionType(v.(string))
			cs[i].ConditionType = conType
		}
	}
	return cs
}

func patchPodSpec(pathPrefix, prefix string, d *schema.ResourceData) (PatchOperations, error) {
	ops := make([]PatchOperation, 0)

	if d.HasChange(prefix + "active_deadline_seconds") {
		v := d.Get(prefix + "active_deadline_seconds").(int)
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/activeDeadlineSeconds",
			Value: v,
		})
	}

	if d.HasChange(prefix + "container") {
		containers := d.Get(prefix + "container").([]interface{})
		value, _ := expandContainers(containers)

		for i, v := range value {
			ops = append(ops, &ReplaceOperation{
				Path:  pathPrefix + "/containers/" + strconv.Itoa(i) + "/image",
				Value: v.Image,
			})

		}

	}
	return ops, nil
}
