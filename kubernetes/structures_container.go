// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"regexp"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

func flattenCapability(in []v1.Capability) []string {
	att := make([]string, len(in))
	for i, v := range in {
		att[i] = string(v)
	}
	return att
}

func flattenContainerSecurityContext(in *v1.SecurityContext) []interface{} {
	att := make(map[string]interface{})

	if in.AllowPrivilegeEscalation != nil {
		att["allow_privilege_escalation"] = *in.AllowPrivilegeEscalation
	}
	if in.Capabilities != nil {
		att["capabilities"] = flattenSecurityCapabilities(in.Capabilities)
	}
	if in.Privileged != nil {
		att["privileged"] = *in.Privileged
	}
	if in.ReadOnlyRootFilesystem != nil {
		att["read_only_root_filesystem"] = *in.ReadOnlyRootFilesystem
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
	if in.SELinuxOptions != nil {
		att["se_linux_options"] = flattenSeLinuxOptions(in.SELinuxOptions)
	}
	return []interface{}{att}

}

func flattenSecurityCapabilities(in *v1.Capabilities) []interface{} {
	att := make(map[string]interface{})

	if in.Add != nil {
		att["add"] = flattenCapability(in.Add)
	}
	if in.Drop != nil {
		att["drop"] = flattenCapability(in.Drop)
	}

	return []interface{}{att}
}

func flattenLifecycleHandler(in *v1.LifecycleHandler) []interface{} {
	att := make(map[string]interface{})

	if in.Exec != nil {
		att["exec"] = flattenExec(in.Exec)
	}
	if in.HTTPGet != nil {
		att["http_get"] = flattenHTTPGet(in.HTTPGet)
	}
	if in.TCPSocket != nil {
		att["tcp_socket"] = flattenTCPSocket(in.TCPSocket)
	}

	return []interface{}{att}
}

func flattenHTTPHeader(in []v1.HTTPHeader) []interface{} {
	att := make([]interface{}, len(in))
	for i, v := range in {
		m := map[string]interface{}{}

		if v.Name != "" {
			m["name"] = v.Name
		}

		if v.Value != "" {
			m["value"] = v.Value
		}
		att[i] = m
	}
	return att
}

func flattenHTTPGet(in *v1.HTTPGetAction) []interface{} {
	att := make(map[string]interface{})

	if in.Host != "" {
		att["host"] = in.Host
	}
	if in.Path != "" {
		att["path"] = in.Path
	}
	att["port"] = in.Port.String()
	att["scheme"] = in.Scheme
	if len(in.HTTPHeaders) > 0 {
		att["http_header"] = flattenHTTPHeader(in.HTTPHeaders)
	}

	return []interface{}{att}
}

func flattenTCPSocket(in *v1.TCPSocketAction) []interface{} {
	att := make(map[string]interface{})
	att["port"] = in.Port.String()
	return []interface{}{att}
}

func flattenGRPC(in *v1.GRPCAction) []interface{} {
	att := make(map[string]interface{})
	att["port"] = in.Port
	if in.Service != nil {
		att["service"] = *in.Service
	}
	return []interface{}{att}
}

func flattenExec(in *v1.ExecAction) []interface{} {
	att := make(map[string]interface{})
	if len(in.Command) > 0 {
		att["command"] = in.Command
	}
	return []interface{}{att}
}

func flattenLifeCycle(in *v1.Lifecycle) []interface{} {
	att := make(map[string]interface{})

	if in.PostStart != nil {
		att["post_start"] = flattenLifecycleHandler(in.PostStart)
	}
	if in.PreStop != nil {
		att["pre_stop"] = flattenLifecycleHandler(in.PreStop)
	}

	return []interface{}{att}
}

func flattenProbe(in *v1.Probe) []interface{} {
	att := make(map[string]interface{})

	att["failure_threshold"] = in.FailureThreshold
	att["initial_delay_seconds"] = in.InitialDelaySeconds
	att["period_seconds"] = in.PeriodSeconds
	att["success_threshold"] = in.SuccessThreshold
	att["timeout_seconds"] = in.TimeoutSeconds

	if in.Exec != nil {
		att["exec"] = flattenExec(in.Exec)
	}
	if in.HTTPGet != nil {
		att["http_get"] = flattenHTTPGet(in.HTTPGet)
	}
	if in.TCPSocket != nil {
		att["tcp_socket"] = flattenTCPSocket(in.TCPSocket)
	}
	if in.GRPC != nil {
		att["grpc"] = flattenGRPC(in.GRPC)
	}

	return []interface{}{att}
}

func flattenConfigMapRef(in *v1.ConfigMapEnvSource) []interface{} {
	att := make(map[string]interface{})

	if in.Name != "" {
		att["name"] = in.Name
	}
	if in.Optional != nil {
		att["optional"] = *in.Optional
	}
	return []interface{}{att}
}

func flattenConfigMapKeyRef(in *v1.ConfigMapKeySelector) []interface{} {
	att := make(map[string]interface{})

	if in.Key != "" {
		att["key"] = in.Key
	}
	if in.Name != "" {
		att["name"] = in.Name
	}
	if in.Optional != nil {
		att["optional"] = *in.Optional
	}
	return []interface{}{att}
}

func flattenObjectFieldSelector(in *v1.ObjectFieldSelector) []interface{} {
	att := make(map[string]interface{})

	if in.APIVersion != "" {
		att["api_version"] = in.APIVersion
	}
	if in.FieldPath != "" {
		att["field_path"] = in.FieldPath
	}
	return []interface{}{att}
}

func flattenResourceFieldSelector(in *v1.ResourceFieldSelector) []interface{} {
	att := make(map[string]interface{})

	if in.ContainerName != "" {
		att["container_name"] = in.ContainerName
	}
	if in.Resource != "" {
		att["resource"] = in.Resource
	}
	if in.Divisor.String() != "" {
		att["divisor"] = in.Divisor.String()
	}
	return []interface{}{att}
}

func flattenSecretRef(in *v1.SecretEnvSource) []interface{} {
	att := make(map[string]interface{})

	if in.Name != "" {
		att["name"] = in.Name
	}
	if in.Optional != nil {
		att["optional"] = *in.Optional
	}
	return []interface{}{att}
}

func flattenSecretKeyRef(in *v1.SecretKeySelector) []interface{} {
	att := make(map[string]interface{})

	if in.Key != "" {
		att["key"] = in.Key
	}
	if in.Name != "" {
		att["name"] = in.Name
	}
	if in.Optional != nil {
		att["optional"] = *in.Optional
	}
	return []interface{}{att}
}

func flattenValueFrom(in *v1.EnvVarSource) []interface{} {
	att := make(map[string]interface{})

	if in.ConfigMapKeyRef != nil {
		att["config_map_key_ref"] = flattenConfigMapKeyRef(in.ConfigMapKeyRef)
	}
	if in.ResourceFieldRef != nil {
		att["resource_field_ref"] = flattenResourceFieldSelector(in.ResourceFieldRef)
	}
	if in.SecretKeyRef != nil {
		att["secret_key_ref"] = flattenSecretKeyRef(in.SecretKeyRef)
	}
	if in.FieldRef != nil {
		att["field_ref"] = flattenObjectFieldSelector(in.FieldRef)
	}
	return []interface{}{att}
}

func flattenContainerVolumeMounts(in []v1.VolumeMount) []interface{} {
	att := make([]interface{}, len(in))

	for i, v := range in {
		m := map[string]interface{}{}
		m["read_only"] = v.ReadOnly

		if v.MountPath != "" {
			m["mount_path"] = v.MountPath

		}
		if v.Name != "" {
			m["name"] = v.Name

		}
		if v.SubPath != "" {
			m["sub_path"] = v.SubPath
		}
		if v.SubPathExpr != "" {
			m["sub_path_expr"] = v.SubPathExpr
		}

		m["mount_propagation"] = string(v1.MountPropagationNone)
		if v.MountPropagation != nil {
			m["mount_propagation"] = string(*v.MountPropagation)
		}
		att[i] = m
	}
	return att
}

func flattenContainerVolumeDevices(in []v1.VolumeDevice) []interface{} {
	att := make([]interface{}, len(in))

	for i, v := range in {
		m := map[string]interface{}{}

		if v.DevicePath != "" {
			m["device_path"] = v.DevicePath
		}

		if v.Name != "" {
			m["name"] = v.Name
		}

		att[i] = m
	}
	return att
}

func flattenContainerEnvs(in []v1.EnvVar) []interface{} {
	att := make([]interface{}, len(in))
	for i, v := range in {
		m := map[string]interface{}{}
		if v.Name != "" {
			m["name"] = v.Name
		}
		if v.Value != "" {
			m["value"] = v.Value
		}
		if v.ValueFrom != nil {
			m["value_from"] = flattenValueFrom(v.ValueFrom)
		}

		att[i] = m
	}
	return att
}

func flattenContainerEnvFroms(in []v1.EnvFromSource) []interface{} {
	att := make([]interface{}, len(in))
	for i, v := range in {
		m := map[string]interface{}{}
		if v.ConfigMapRef != nil {
			m["config_map_ref"] = flattenConfigMapRef(v.ConfigMapRef)
		}
		if v.Prefix != "" {
			m["prefix"] = v.Prefix
		}
		if v.SecretRef != nil {
			m["secret_ref"] = flattenSecretRef(v.SecretRef)
		}

		att[i] = m
	}
	return att
}

func flattenContainerPorts(in []v1.ContainerPort) []interface{} {
	att := make([]interface{}, len(in))
	for i, v := range in {
		m := map[string]interface{}{}
		m["container_port"] = v.ContainerPort
		if v.HostIP != "" {
			m["host_ip"] = v.HostIP
		}
		m["host_port"] = v.HostPort
		if v.Name != "" {
			m["name"] = v.Name
		}
		if v.Protocol != "" {
			m["protocol"] = v.Protocol
		}
		att[i] = m
	}
	return att
}

func flattenContainerResourceRequirements(in v1.ResourceRequirements) []interface{} {
	att := make(map[string]interface{})
	att["limits"] = flattenResourceList(in.Limits)
	att["requests"] = flattenResourceList(in.Requests)
	return []interface{}{att}
}

func flattenContainers(in []v1.Container, serviceAccountRegex string) ([]interface{}, error) {
	att := make([]interface{}, len(in))
	for i, v := range in {
		c := make(map[string]interface{})
		c["image"] = v.Image
		c["name"] = v.Name
		if len(v.Command) > 0 {
			c["command"] = v.Command
		}
		if len(v.Args) > 0 {
			c["args"] = v.Args
		}

		c["image_pull_policy"] = v.ImagePullPolicy
		c["termination_message_path"] = v.TerminationMessagePath
		c["termination_message_policy"] = v.TerminationMessagePolicy
		c["stdin"] = v.Stdin
		c["stdin_once"] = v.StdinOnce
		c["tty"] = v.TTY
		c["working_dir"] = v.WorkingDir
		c["resources"] = flattenContainerResourceRequirements(v.Resources)
		if v.LivenessProbe != nil {
			c["liveness_probe"] = flattenProbe(v.LivenessProbe)
		}
		if v.ReadinessProbe != nil {
			c["readiness_probe"] = flattenProbe(v.ReadinessProbe)
		}
		if v.StartupProbe != nil {
			c["startup_probe"] = flattenProbe(v.StartupProbe)
		}
		if v.Lifecycle != nil {
			c["lifecycle"] = flattenLifeCycle(v.Lifecycle)
		}

		if v.SecurityContext != nil {
			c["security_context"] = flattenContainerSecurityContext(v.SecurityContext)
		}
		if len(v.Ports) > 0 {
			c["port"] = flattenContainerPorts(v.Ports)
		}
		if len(v.Env) > 0 {
			c["env"] = flattenContainerEnvs(v.Env)
		}
		if len(v.EnvFrom) > 0 {
			c["env_from"] = flattenContainerEnvFroms(v.EnvFrom)
		}

		if len(v.VolumeMounts) > 0 {
			for num, m := range v.VolumeMounts {
				// To avoid perpetual diff, remove the default service account token volume from the container's list of volumeMounts.
				nameMatchesDefaultToken, err := regexp.MatchString(serviceAccountRegex, m.Name)
				if err != nil {
					return att, err
				}
				if nameMatchesDefaultToken || strings.HasPrefix(m.Name, "kube-api-access") {
					v.VolumeMounts = removeVolumeMountFromContainer(num, v.VolumeMounts)
					break
				}
			}
			c["volume_mount"] = flattenContainerVolumeMounts(v.VolumeMounts)
		}

		if len(v.VolumeDevices) > 0 {
			c["volume_device"] = flattenContainerVolumeDevices(v.VolumeDevices)
		}

		att[i] = c
	}
	return att, nil
}

// removeVolumeMountFromContainer removes the specified VolumeMount index (i) from the given list of VolumeMounts.
func removeVolumeMountFromContainer(i int, v []v1.VolumeMount) []v1.VolumeMount {
	return append(v[:i], v[i+1:]...)
}

func expandContainers(ctrs []interface{}) ([]v1.Container, error) {
	if len(ctrs) == 0 {
		return []v1.Container{}, nil
	}
	cs := make([]v1.Container, len(ctrs))
	for i, c := range ctrs {
		ctr := c.(map[string]interface{})

		if image, ok := ctr["image"]; ok {
			cs[i].Image = image.(string)
		}
		if name, ok := ctr["name"]; ok {
			cs[i].Name = name.(string)
		}
		if command, ok := ctr["command"].([]interface{}); ok {
			cs[i].Command = expandStringSlice(command)
		} else {
			// https://github.com/hashicorp/terraform-plugin-sdk/issues/142
			// Set defaults manually until List defaults are supported in the SDK.
			cs[i].Command = []string{}
		}
		if args, ok := ctr["args"].([]interface{}); ok {
			cs[i].Args = expandStringSlice(args)
		} else {
			cs[i].Args = []string{}
		}

		if v, ok := ctr["resources"].([]interface{}); ok && len(v) > 0 {

			var err error
			crr, err := expandContainerResourceRequirements(v)
			if err != nil {
				return cs, err
			}
			cs[i].Resources = *crr
		}

		if v, ok := ctr["port"].([]interface{}); ok && len(v) > 0 {
			cp := expandContainerPort(v)
			for _, p := range cp {
				cs[i].Ports = append(cs[i].Ports, *p)
			}
		}
		if v, ok := ctr["env"].([]interface{}); ok && len(v) > 0 {
			var err error
			cs[i].Env, err = expandContainerEnv(v)
			if err != nil {
				return cs, err
			}
		}
		if v, ok := ctr["env_from"].([]interface{}); ok && len(v) > 0 {
			var err error
			cs[i].EnvFrom, err = expandContainerEnvFrom(v)
			if err != nil {
				return cs, err
			}
		}

		if policy, ok := ctr["image_pull_policy"]; ok {
			cs[i].ImagePullPolicy = v1.PullPolicy(policy.(string))
		}

		if v, ok := ctr["lifecycle"].([]interface{}); ok && len(v) > 0 {
			cs[i].Lifecycle = expandLifeCycle(v)
		}

		if v, ok := ctr["liveness_probe"].([]interface{}); ok && len(v) > 0 {
			cs[i].LivenessProbe = expandProbe(v)
		}

		if v, ok := ctr["readiness_probe"].([]interface{}); ok && len(v) > 0 {
			cs[i].ReadinessProbe = expandProbe(v)
		}
		if v, ok := ctr["startup_probe"].([]interface{}); ok && len(v) > 0 {
			cs[i].StartupProbe = expandProbe(v)
		}
		if v, ok := ctr["stdin"]; ok {
			cs[i].Stdin = v.(bool)
		}
		if v, ok := ctr["stdin_once"]; ok {
			cs[i].StdinOnce = v.(bool)
		}
		if v, ok := ctr["termination_message_path"]; ok {
			cs[i].TerminationMessagePath = v.(string)
		}
		if v, ok := ctr["termination_message_policy"]; ok {
			cs[i].TerminationMessagePolicy = v1.TerminationMessagePolicy(v.(string))
		}
		if v, ok := ctr["tty"]; ok {
			cs[i].TTY = v.(bool)
		}
		if v, ok := ctr["security_context"].([]interface{}); ok && len(v) > 0 {
			ctx, err := expandContainerSecurityContext(v)
			if err != nil {
				return cs, err
			}
			cs[i].SecurityContext = ctx
		}

		if v, ok := ctr["volume_mount"].([]interface{}); ok && len(v) > 0 {
			cs[i].VolumeMounts = expandContainerVolumeMounts(v)
		}

		if v, ok := ctr["volume_device"].([]interface{}); ok && len(v) > 0 {
			cs[i].VolumeDevices = expandContainerVolumeDevices(v)
		}

		if v, ok := ctr["working_dir"].(string); ok && v != "" {
			cs[i].WorkingDir = v
		}
	}
	return cs, nil
}

func expandExec(l []interface{}) *v1.ExecAction {
	if len(l) == 0 || l[0] == nil {
		return &v1.ExecAction{}
	}
	in := l[0].(map[string]interface{})
	obj := v1.ExecAction{}
	if v, ok := in["command"].([]interface{}); ok && len(v) > 0 {
		obj.Command = expandStringSlice(v)
	}
	return &obj
}

func expandHTTPHeaders(l []interface{}) []v1.HTTPHeader {
	if len(l) == 0 {
		return []v1.HTTPHeader{}
	}
	headers := make([]v1.HTTPHeader, len(l))
	for i, c := range l {
		m := c.(map[string]interface{})
		if v, ok := m["name"]; ok {
			headers[i].Name = v.(string)
		}
		if v, ok := m["value"]; ok {
			headers[i].Value = v.(string)
		}
	}
	return headers
}
func expandContainerSecurityContext(l []interface{}) (*v1.SecurityContext, error) {
	if len(l) == 0 || l[0] == nil {
		return &v1.SecurityContext{}, nil
	}
	in := l[0].(map[string]interface{})
	obj := v1.SecurityContext{}
	if v, ok := in["allow_privilege_escalation"]; ok {
		obj.AllowPrivilegeEscalation = ptr.To(v.(bool))
	}
	if v, ok := in["capabilities"].([]interface{}); ok && len(v) > 0 {
		obj.Capabilities = expandSecurityCapabilities(v)
	}
	if v, ok := in["privileged"]; ok {
		obj.Privileged = ptr.To(v.(bool))
	}
	if v, ok := in["read_only_root_filesystem"]; ok {
		obj.ReadOnlyRootFilesystem = ptr.To(v.(bool))
	}
	if v, ok := in["run_as_group"].(string); ok && v != "" {
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return &obj, err
		}
		obj.RunAsGroup = ptr.To(int64(i))
	}
	if v, ok := in["run_as_non_root"]; ok {
		obj.RunAsNonRoot = ptr.To(v.(bool))
	}
	if v, ok := in["run_as_user"].(string); ok && v != "" {
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return &obj, err
		}
		obj.RunAsUser = ptr.To(int64(i))
	}
	if v, ok := in["seccomp_profile"].([]interface{}); ok && len(v) > 0 {
		obj.SeccompProfile = expandSeccompProfile(v)
	}
	if v, ok := in["se_linux_options"].([]interface{}); ok && len(v) > 0 {
		obj.SELinuxOptions = expandSeLinuxOptions(v)
	}

	return &obj, nil
}

func expandCapabilitySlice(s []interface{}) []v1.Capability {
	result := make([]v1.Capability, len(s))
	for k, v := range s {
		result[k] = v1.Capability(v.(string))
	}
	return result
}

func expandSecurityCapabilities(l []interface{}) *v1.Capabilities {
	if len(l) == 0 || l[0] == nil {
		return &v1.Capabilities{}
	}
	in := l[0].(map[string]interface{})
	obj := v1.Capabilities{}
	if v, ok := in["add"].([]interface{}); ok {
		obj.Add = expandCapabilitySlice(v)
	}
	if v, ok := in["drop"].([]interface{}); ok {
		obj.Drop = expandCapabilitySlice(v)
	}
	return &obj
}

func expandTCPSocket(l []interface{}) *v1.TCPSocketAction {
	if len(l) == 0 || l[0] == nil {
		return &v1.TCPSocketAction{}
	}
	in := l[0].(map[string]interface{})
	obj := v1.TCPSocketAction{}
	if v, ok := in["port"].(string); ok && len(v) > 0 {
		obj.Port = intstr.Parse(v)
	}
	return &obj
}

func expandGRPC(l []interface{}) *v1.GRPCAction {
	if len(l) == 0 || l[0] == nil {
		return &v1.GRPCAction{}
	}
	in := l[0].(map[string]interface{})
	obj := v1.GRPCAction{}
	if v, ok := in["port"].(int); ok {
		obj.Port = int32(v)
	}
	if v, ok := in["service"].(string); ok {
		obj.Service = ptr.To(v)
	}
	return &obj
}

func expandHTTPGet(l []interface{}) *v1.HTTPGetAction {
	if len(l) == 0 || l[0] == nil {
		return &v1.HTTPGetAction{}
	}
	in := l[0].(map[string]interface{})
	obj := v1.HTTPGetAction{}
	if v, ok := in["host"].(string); ok && len(v) > 0 {
		obj.Host = v
	}
	if v, ok := in["path"].(string); ok && len(v) > 0 {
		obj.Path = v
	}
	if v, ok := in["scheme"].(string); ok && len(v) > 0 {
		obj.Scheme = v1.URIScheme(v)
	}

	if v, ok := in["port"].(string); ok && len(v) > 0 {
		obj.Port = intstr.Parse(v)
	}

	if v, ok := in["http_header"].([]interface{}); ok && len(v) > 0 {
		obj.HTTPHeaders = expandHTTPHeaders(v)
	}
	return &obj
}

func expandProbe(l []interface{}) *v1.Probe {
	if len(l) == 0 || l[0] == nil {
		return &v1.Probe{}
	}
	in := l[0].(map[string]interface{})
	obj := v1.Probe{}
	if v, ok := in["exec"].([]interface{}); ok && len(v) > 0 {
		obj.Exec = expandExec(v)
	}
	if v, ok := in["http_get"].([]interface{}); ok && len(v) > 0 {
		obj.HTTPGet = expandHTTPGet(v)
	}
	if v, ok := in["tcp_socket"].([]interface{}); ok && len(v) > 0 {
		obj.TCPSocket = expandTCPSocket(v)
	}
	if v, ok := in["grpc"].([]interface{}); ok && len(v) > 0 {
		obj.GRPC = expandGRPC(v)
	}
	if v, ok := in["failure_threshold"].(int); ok {
		obj.FailureThreshold = int32(v)
	}
	if v, ok := in["initial_delay_seconds"].(int); ok {
		obj.InitialDelaySeconds = int32(v)
	}
	if v, ok := in["period_seconds"].(int); ok {
		obj.PeriodSeconds = int32(v)
	}
	if v, ok := in["success_threshold"].(int); ok {
		obj.SuccessThreshold = int32(v)
	}
	if v, ok := in["timeout_seconds"].(int); ok {
		obj.TimeoutSeconds = int32(v)
	}

	return &obj
}

func expandLifecycleHandlers(l []interface{}) *v1.LifecycleHandler {
	if len(l) == 0 || l[0] == nil {
		return &v1.LifecycleHandler{}
	}
	in := l[0].(map[string]interface{})
	obj := v1.LifecycleHandler{}
	if v, ok := in["exec"].([]interface{}); ok && len(v) > 0 {
		obj.Exec = expandExec(v)
	}
	if v, ok := in["http_get"].([]interface{}); ok && len(v) > 0 {
		obj.HTTPGet = expandHTTPGet(v)
	}
	if v, ok := in["tcp_socket"].([]interface{}); ok && len(v) > 0 {
		obj.TCPSocket = expandTCPSocket(v)
	}
	return &obj

}
func expandLifeCycle(l []interface{}) *v1.Lifecycle {
	if len(l) == 0 || l[0] == nil {
		return &v1.Lifecycle{}
	}
	in := l[0].(map[string]interface{})
	obj := &v1.Lifecycle{}
	if v, ok := in["post_start"].([]interface{}); ok && len(v) > 0 {
		obj.PostStart = expandLifecycleHandlers(v)
	}
	if v, ok := in["pre_stop"].([]interface{}); ok && len(v) > 0 {
		obj.PreStop = expandLifecycleHandlers(v)
	}
	return obj
}

func expandContainerVolumeMounts(in []interface{}) []v1.VolumeMount {
	if len(in) == 0 {
		return []v1.VolumeMount{}
	}
	vmp := make([]v1.VolumeMount, len(in))
	for i, c := range in {
		p := c.(map[string]interface{})
		if mountPath, ok := p["mount_path"]; ok {
			vmp[i].MountPath = mountPath.(string)
		}
		if name, ok := p["name"]; ok {
			vmp[i].Name = name.(string)
		}
		if readOnly, ok := p["read_only"]; ok {
			vmp[i].ReadOnly = readOnly.(bool)
		}
		if subPath, ok := p["sub_path"]; ok {
			vmp[i].SubPath = subPath.(string)
		}
		if subPathExpr, ok := p["sub_path_expr"]; ok {
			vmp[i].SubPathExpr = subPathExpr.(string)
		}
		if mountPropagation, ok := p["mount_propagation"]; ok {
			mp := v1.MountPropagationMode(mountPropagation.(string))
			vmp[i].MountPropagation = &mp
		}
	}
	return vmp
}

func expandContainerVolumeDevices(in []interface{}) []v1.VolumeDevice {
	if len(in) == 0 {
		return []v1.VolumeDevice{}
	}
	volumeDevices := make([]v1.VolumeDevice, len(in))
	for i, c := range in {
		p := c.(map[string]interface{})
		if devicePath, ok := p["device_path"]; ok {
			volumeDevices[i].DevicePath = devicePath.(string)
		}
		if name, ok := p["name"]; ok {
			volumeDevices[i].Name = name.(string)
		}
	}
	return volumeDevices
}

func expandContainerEnv(in []interface{}) ([]v1.EnvVar, error) {
	if len(in) == 0 {
		return []v1.EnvVar{}, nil
	}
	envs := []v1.EnvVar{}
	for _, c := range in {
		p, ok := c.(map[string]interface{})
		if !ok {
			continue
		}

		env := v1.EnvVar{}
		if name, ok := p["name"]; ok {
			env.Name = name.(string)
		}
		if value, ok := p["value"]; ok {
			env.Value = value.(string)
		}
		if v, ok := p["value_from"].([]interface{}); ok && len(v) > 0 {
			var err error
			env.ValueFrom, err = expandEnvValueFrom(v)
			if err != nil {
				return envs, err
			}
		}
		envs = append(envs, env)
	}
	return envs, nil
}

func expandContainerEnvFrom(in []interface{}) ([]v1.EnvFromSource, error) {
	if len(in) == 0 {
		return []v1.EnvFromSource{}, nil
	}
	envFroms := make([]v1.EnvFromSource, len(in))
	for i, c := range in {
		p := c.(map[string]interface{})
		if v, ok := p["config_map_ref"].([]interface{}); ok && len(v) > 0 {
			envFroms[i].ConfigMapRef = expandConfigMapRef(v)
		}
		if value, ok := p["prefix"]; ok {
			envFroms[i].Prefix = value.(string)
		}
		if v, ok := p["secret_ref"].([]interface{}); ok && len(v) > 0 {
			envFroms[i].SecretRef = expandSecretRef(v)
		}
	}
	return envFroms, nil
}

func expandContainerPort(in []interface{}) []*v1.ContainerPort {
	ports := make([]*v1.ContainerPort, len(in))
	if len(in) == 0 {
		return ports
	}
	for i, c := range in {
		p := c.(map[string]interface{})
		ports[i] = &v1.ContainerPort{}
		if containerPort, ok := p["container_port"]; ok {
			ports[i].ContainerPort = int32(containerPort.(int))
		}
		if hostIP, ok := p["host_ip"]; ok {
			ports[i].HostIP = hostIP.(string)
		}
		if hostPort, ok := p["host_port"]; ok {
			ports[i].HostPort = int32(hostPort.(int))
		}
		if name, ok := p["name"]; ok {
			ports[i].Name = name.(string)
		}
		if protocol, ok := p["protocol"]; ok {
			ports[i].Protocol = v1.Protocol(protocol.(string))
		}
	}
	return ports
}

func expandConfigMapKeyRef(r []interface{}) *v1.ConfigMapKeySelector {
	if len(r) == 0 || r[0] == nil {
		return &v1.ConfigMapKeySelector{}
	}
	in := r[0].(map[string]interface{})
	obj := &v1.ConfigMapKeySelector{}

	if v, ok := in["key"].(string); ok {
		obj.Key = v
	}
	if v, ok := in["name"].(string); ok {
		obj.Name = v
	}
	if v, ok := in["optional"]; ok {
		obj.Optional = ptr.To(v.(bool))
	}
	return obj

}
func expandFieldRef(r []interface{}) *v1.ObjectFieldSelector {
	if len(r) == 0 || r[0] == nil {
		return &v1.ObjectFieldSelector{}
	}
	in := r[0].(map[string]interface{})
	obj := &v1.ObjectFieldSelector{}

	if v, ok := in["api_version"].(string); ok {
		obj.APIVersion = v
	}
	if v, ok := in["field_path"].(string); ok {
		obj.FieldPath = v
	}
	return obj
}
func expandResourceFieldRef(r []interface{}) (*v1.ResourceFieldSelector, error) {
	if len(r) == 0 || r[0] == nil {
		return &v1.ResourceFieldSelector{}, nil
	}
	in := r[0].(map[string]interface{})
	obj := &v1.ResourceFieldSelector{}

	if v, ok := in["container_name"].(string); ok {
		obj.ContainerName = v
	}
	if v, ok := in["resource"].(string); ok {
		obj.Resource = v
	}
	if v, ok := in["divisor"].(string); ok {
		q, err := resource.ParseQuantity(v)
		if err != nil {
			return obj, err
		}
		obj.Divisor = q
	}
	return obj, nil
}

func expandSecretRef(r []interface{}) *v1.SecretEnvSource {
	if len(r) == 0 || r[0] == nil {
		return &v1.SecretEnvSource{}
	}
	in := r[0].(map[string]interface{})
	obj := &v1.SecretEnvSource{}

	if v, ok := in["name"].(string); ok {
		obj.Name = v
	}
	if v, ok := in["optional"]; ok {
		obj.Optional = ptr.To(v.(bool))
	}

	return obj
}

func expandSecretKeyRef(r []interface{}) *v1.SecretKeySelector {
	if len(r) == 0 || r[0] == nil {
		return &v1.SecretKeySelector{}
	}
	in := r[0].(map[string]interface{})
	obj := &v1.SecretKeySelector{}

	if v, ok := in["key"].(string); ok {
		obj.Key = v
	}
	if v, ok := in["name"].(string); ok {
		obj.Name = v
	}
	if v, ok := in["optional"]; ok {
		obj.Optional = ptr.To(v.(bool))
	}
	return obj
}

func expandEnvValueFrom(r []interface{}) (*v1.EnvVarSource, error) {
	if len(r) == 0 || r[0] == nil {
		return &v1.EnvVarSource{}, nil
	}
	in := r[0].(map[string]interface{})
	obj := &v1.EnvVarSource{}

	var err error
	if v, ok := in["config_map_key_ref"].([]interface{}); ok && len(v) > 0 {
		obj.ConfigMapKeyRef = expandConfigMapKeyRef(v)
	}
	if v, ok := in["field_ref"].([]interface{}); ok && len(v) > 0 {
		obj.FieldRef = expandFieldRef(v)
	}
	if v, ok := in["secret_key_ref"].([]interface{}); ok && len(v) > 0 {
		obj.SecretKeyRef = expandSecretKeyRef(v)
	}
	if v, ok := in["resource_field_ref"].([]interface{}); ok && len(v) > 0 {
		obj.ResourceFieldRef, err = expandResourceFieldRef(v)
		if err != nil {
			return obj, err
		}
	}
	return obj, nil

}

func expandConfigMapRef(r []interface{}) *v1.ConfigMapEnvSource {
	if len(r) == 0 || r[0] == nil {
		return &v1.ConfigMapEnvSource{}
	}
	in := r[0].(map[string]interface{})
	obj := &v1.ConfigMapEnvSource{}

	if v, ok := in["name"].(string); ok {
		obj.Name = v
	}
	if v, ok := in["optional"]; ok {
		obj.Optional = ptr.To(v.(bool))
	}

	return obj
}

func expandContainerResourceRequirements(l []interface{}) (*v1.ResourceRequirements, error) {
	obj := &v1.ResourceRequirements{}
	if len(l) == 0 || l[0] == nil {
		return obj, nil
	}
	in := l[0].(map[string]interface{})

	if v, ok := in["limits"].(map[string]interface{}); ok && len(v) > 0 {
		r, err := expandMapToResourceList(v)
		if err != nil {
			return obj, err
		}
		obj.Limits = *r
	}

	if v, ok := in["requests"].(map[string]interface{}); ok && len(v) > 0 {
		r, err := expandMapToResourceList(v)
		if err != nil {
			return obj, err
		}
		obj.Requests = *r
	}

	return obj, nil
}
