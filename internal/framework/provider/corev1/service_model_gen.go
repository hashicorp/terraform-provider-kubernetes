package corev1

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ServiceModel struct {
	Timeouts timeouts.Value `tfsdk:"timeouts"`

	ID       types.String `tfsdk:"id" manifest:""`
	Metadata struct {
		Annotations     map[string]types.String `tfsdk:"annotations" manifest:"annotations"`
		GenerateName    types.String            `tfsdk:"generate_name" manifest:"generateName"`
		Generation      types.Int64             `tfsdk:"generation" manifest:"generation"`
		Labels          map[string]types.String `tfsdk:"labels" manifest:"labels"`
		Name            types.String            `tfsdk:"name" manifest:"name"`
		Namespace       types.String            `tfsdk:"namespace" manifest:"namespace"`
		ResourceVersion types.String            `tfsdk:"resource_version" manifest:"resourceVersion"`
		UID             types.String            `tfsdk:"uid" manifest:"uid"`
	} `tfsdk:"metadata" manifest:"metadata"`
	Spec struct {
		AllocateLoadBalancerNodePorts types.Bool     `tfsdk:"allocate_load_balancer_node_ports" manifest:"allocateLoadBalancerNodePorts"`
		ClusterIp                     types.String   `tfsdk:"cluster_ip" manifest:"clusterIp"`
		ClusterIps                    []types.String `tfsdk:"cluster_ips" manifest:"clusterIps"`
		ExternalIps                   []types.String `tfsdk:"external_ips" manifest:"externalIps"`
		ExternalName                  types.String   `tfsdk:"external_name" manifest:"externalName"`
		ExternalTrafficPolicy         types.String   `tfsdk:"external_traffic_policy" manifest:"externalTrafficPolicy"`
		HealthCheckNodePort           types.Int64    `tfsdk:"health_check_node_port" manifest:"healthCheckNodePort"`
		InternalTrafficPolicy         types.String   `tfsdk:"internal_traffic_policy" manifest:"internalTrafficPolicy"`
		IpFamilies                    []types.String `tfsdk:"ip_families" manifest:"ipFamilies"`
		IpFamilyPolicy                types.String   `tfsdk:"ip_family_policy" manifest:"ipFamilyPolicy"`
		LoadBalancerClass             types.String   `tfsdk:"load_balancer_class" manifest:"loadBalancerClass"`
		LoadBalancerIp                types.String   `tfsdk:"load_balancer_ip" manifest:"loadBalancerIp"`
		LoadBalancerSourceRanges      []types.String `tfsdk:"load_balancer_source_ranges" manifest:"loadBalancerSourceRanges"`
		Ports                         []struct {
			AppProtocol types.String `tfsdk:"app_protocol" manifest:"appProtocol"`
			Name        types.String `tfsdk:"name" manifest:"name"`
			NodePort    types.Int64  `tfsdk:"node_port" manifest:"nodePort"`
			Port        types.Int64  `tfsdk:"port" manifest:"port"`
			Protocol    types.String `tfsdk:"protocol" manifest:"protocol"`
			TargetPort  types.String `tfsdk:"target_port" manifest:"targetPort"`
		} `tfsdk:"ports" manifest:"ports"`
		PublishNotReadyAddresses types.Bool              `tfsdk:"publish_not_ready_addresses" manifest:"publishNotReadyAddresses"`
		Selector                 map[string]types.String `tfsdk:"selector" manifest:"selector"`
		SessionAffinity          types.String            `tfsdk:"session_affinity" manifest:"sessionAffinity"`
		SessionAffinityConfig    struct {
			ClientIp struct {
				TimeoutSeconds types.Int64 `tfsdk:"timeout_seconds" manifest:"timeoutSeconds"`
			} `tfsdk:"client_ip" manifest:"clientIp"`
		} `tfsdk:"session_affinity_config" manifest:"sessionAffinityConfig"`
		Type types.String `tfsdk:"type" manifest:"type"`
	} `tfsdk:"spec" manifest:"spec"`
}
