package kubernetes

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	v1 "k8s.io/api/core/v1"
)

// Flatteners

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

	containers, err := flattenContainers(in.Containers)
	if err != nil {
		return nil, err
	}
	att["container"] = containers

	initContainers, err := flattenContainers(in.InitContainers)
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

	att["host_aliases"] = flattenHostaliases(in.HostAliases)

	att["host_ipc"] = in.HostIPC
	att["host_network"] = in.HostNetwork
	att["host_pid"] = in.HostPID

	if in.Hostname != "" {
		att["hostname"] = in.Hostname
	}
	att["image_pull_secrets"] = flattenLocalObjectReferenceArray(in.ImagePullSecrets)

	if in.NodeName != "" {
		att["node_name"] = in.NodeName
	}
	if len(in.NodeSelector) > 0 {
		att["node_selector"] = in.NodeSelector
	}
	if in.RestartPolicy != "" {
		att["restart_policy"] = in.RestartPolicy
	}

	if in.SecurityContext != nil {
		att["security_context"] = flattenPodSecurityContext(in.SecurityContext)
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

	if len(in.Volumes) > 0 {
		v, err := flattenVolumes(in.Volumes)
		if err != nil {
			return []interface{}{att}, err
		}
		att["volume"] = v
	}
	return []interface{}{att}, nil
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
		v, err := flattenPodDNSConfigOptions(in.Options)
		if err != nil {
			return []interface{}{att}, err
		}
		att["option"] = v
	}

	if len(att) > 0 {
		return []interface{}{att}, nil
	}
	return []interface{}{}, nil
}

func flattenPodDNSConfigOptions(options []v1.PodDNSConfigOption) ([]interface{}, error) {
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
	return att, nil
}

func flattenPodSecurityContext(in *v1.PodSecurityContext) []interface{} {
	att := make(map[string]interface{})

	if in.FSGroup != nil {
		att["fs_group"] = *in.FSGroup
	}
	if in.RunAsGroup != nil {
		att["run_as_group"] = *in.RunAsGroup
	}
	if in.RunAsNonRoot != nil {
		att["run_as_non_root"] = *in.RunAsNonRoot
	}
	if in.RunAsUser != nil {
		att["run_as_user"] = *in.RunAsUser
	}
	if len(in.SupplementalGroups) > 0 {
		att["supplemental_groups"] = newInt64Set(schema.HashSchema(&schema.Schema{
			Type: schema.TypeInt,
		}), in.SupplementalGroups)
	}
	if in.SELinuxOptions != nil {
		att["se_linux_options"] = flattenSeLinuxOptions(in.SELinuxOptions)
	}

	if len(att) > 0 {
		return []interface{}{att}
	}
	return []interface{}{}
}

func flattenSeLinuxOptions(in *v1.SELinuxOptions) []interface{} {
	att := make(map[string]interface{})
	if in.User != "" {
		att["user"] = in.User
	}
	if in.Role != "" {
		att["role"] = in.Role
	}
	if in.User != "" {
		att["type"] = in.Type
	}
	if in.Level != "" {
		att["level"] = in.Level
	}
	return []interface{}{att}
}

func flattenVolumes(volumes []v1.Volume) ([]interface{}, error) {
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
		att[i] = obj
	}
	return att, nil
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

	return []interface{}{att}
}

func flattenEmptyDirVolumeSource(in *v1.EmptyDirVolumeSource) []interface{} {
	att := make(map[string]interface{})
	att["medium"] = in.Medium
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

// Expanders

func expandPodSpec(p []interface{}) (*v1.PodSpec, error) {
	obj := &v1.PodSpec{}
	if len(p) == 0 || p[0] == nil {
		return obj, nil
	}
	in := p[0].(map[string]interface{})

	if v, ok := in["active_deadline_seconds"].(int); ok && v > 0 {
		obj.ActiveDeadlineSeconds = ptrToInt64(int64(v))
	}

	if v, ok := in["affinity"].([]interface{}); ok && len(v) > 0 {
		a, err := expandAffinity(v)
		if err != nil {
			return obj, err
		}
		obj.Affinity = a
	}

	if v, ok := in["automount_service_account_token"].(bool); ok {
		obj.AutomountServiceAccountToken = ptrToBool(v)
	}

	if v, ok := in["container"].([]interface{}); ok && len(v) > 0 {
		cs, err := expandContainers(v)
		if err != nil {
			return obj, err
		}
		obj.Containers = cs
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

	if v, ok := in["host_aliases"].([]interface{}); ok && len(v) > 0 {
		hs, err := expandHostaliases(v)
		if err != nil {
			return obj, err
		}
		obj.HostAliases = hs
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

	if v, ok := in["restart_policy"].(string); ok {
		obj.RestartPolicy = v1.RestartPolicy(v)
	}

	if v, ok := in["security_context"].([]interface{}); ok && len(v) > 0 {
		obj.SecurityContext = expandPodSecurityContext(v)
	}

	if v, ok := in["service_account_name"].(string); ok {
		obj.ServiceAccountName = v
	}

	if v, ok := in["share_process_namespace"]; ok {
		obj.ShareProcessNamespace = ptrToBool(v.(bool))
	}

	if v, ok := in["subdomain"].(string); ok {
		obj.Subdomain = v
	}

	if v, ok := in["termination_grace_period_seconds"].(int); ok {
		obj.TerminationGracePeriodSeconds = ptrToInt64(int64(v))
	}

	if v, ok := in["volume"].([]interface{}); ok && len(v) > 0 {
		cs, err := expandVolumes(v)
		if err != nil {
			return obj, err
		}
		obj.Volumes = cs
	}
	return obj, nil
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
		opts, err := expandDNSConfigOptions(v)
		if err != nil {
			return obj, err
		}
		obj.Options = opts
	}
	return obj, nil
}

func expandDNSConfigOptions(options []interface{}) ([]v1.PodDNSConfigOption, error) {
	if len(options) == 0 {
		return []v1.PodDNSConfigOption{}, nil
	}
	opts := make([]v1.PodDNSConfigOption, len(options))
	for i, c := range options {
		in := c.(map[string]interface{})
		opt := v1.PodDNSConfigOption{}
		if v, ok := in["name"].(string); ok {
			opt.Name = v
		}
		if v, ok := in["value"].(string); ok {
			opt.Value = ptrToString(v)
		}
		opts[i] = opt
	}

	return opts, nil
}

func expandPodSecurityContext(l []interface{}) *v1.PodSecurityContext {
	if len(l) == 0 || l[0] == nil {
		return &v1.PodSecurityContext{}
	}
	in := l[0].(map[string]interface{})
	obj := &v1.PodSecurityContext{}
	if v, ok := in["fs_group"].(int); ok {
		obj.FSGroup = ptrToInt64(int64(v))
	}
	if v, ok := in["run_as_group"].(int); ok {
		obj.RunAsGroup = ptrToInt64(int64(v))
	}
	if v, ok := in["run_as_non_root"].(bool); ok {
		obj.RunAsNonRoot = ptrToBool(v)
	}
	if v, ok := in["run_as_user"].(int); ok {
		obj.RunAsUser = ptrToInt64(int64(v))
	}
	if v, ok := in["se_linux_options"].([]interface{}); ok && len(v) > 0 {
		obj.SELinuxOptions = expandSeLinuxOptions(v)
	}
	if v, ok := in["supplemental_groups"].(*schema.Set); ok {
		obj.SupplementalGroups = schemaSetToInt64Array(v)
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
				keyPaths[i].Mode = ptrToInt32(int32(m))
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
		if v, ok := p["mode"].(string); ok {
			m, err := strconv.ParseInt(v, 8, 32)
			if err != nil {
				return dapivf, fmt.Errorf("DownwardAPI volume file: failed to parse 'mode' value: %s", err)
			}
			dapivf[i].Mode = ptrToInt32(int32(m))
		}
		if v, ok := p["path"].(string); ok {
			dapivf[i].Path = v
		}
		if v, ok := p["field_ref"].([]interface{}); ok && len(v) > 0 {
			dapivf[i].FieldRef, err = expandFieldRef(v)
			if err != nil {
				return dapivf, err
			}
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
		obj.DefaultMode = ptrToInt32(int32(v))
	}

	if v, ok := in["name"].(string); ok {
		obj.Name = v
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
		obj.DefaultMode = ptrToInt32(int32(v))
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

func expandEmptyDirVolumeSource(l []interface{}) *v1.EmptyDirVolumeSource {
	if len(l) == 0 || l[0] == nil {
		return &v1.EmptyDirVolumeSource{}
	}
	in := l[0].(map[string]interface{})
	obj := &v1.EmptyDirVolumeSource{
		Medium: v1.StorageMedium(in["medium"].(string)),
	}
	return obj
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
		obj.DefaultMode = ptrToInt32(int32(v))
	}

	if secret, ok := in["secret_name"].(string); ok {
		obj.SecretName = secret
	}

	if opt, ok := in["optional"].(bool); ok {
		obj.Optional = ptrToBool(opt)
	}
	if v, ok := in["items"].([]interface{}); ok && len(v) > 0 {
		obj.Items = expandKeyPath(v)
	}

	return obj, nil
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
			vl[i].EmptyDir = expandEmptyDirVolumeSource(value)
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
	}
	return vl, nil
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
