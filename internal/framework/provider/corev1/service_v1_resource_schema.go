// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"net"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

func (r *ServiceV1) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     2,
		Description: "A Service is an abstraction which defines a logical set of pods and a policy by which to access them - sometimes called a micro-service.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the Service, in the form namespace/name.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"wait_for_load_balancer": schema.BoolAttribute{
				Description: "Terraform will wait for the load balancer to have at least 1 endpoint before considering the resource created. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"status": schema.ListAttribute{
				Description: "The current status of the Service, including load balancer ingress IP addresses, IP modes, and hostnames.",
				Computed:    true,
				ElementType: serviceStatusType,
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": serviceMetadataSchema(),
			"spec":     serviceSpecSchema(),
			// SDKv2 exposes only create; delete uses its implicit 20-minute timeout.
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true}),
		},
	}
}

func serviceMetadataSchema() schema.ListNestedBlock {
	block := common.NamespacedMetadataSchema("service", true)
	generateName := block.NestedObject.Attributes["generate_name"].(schema.StringAttribute)
	// Service's released state and outputs use "" for an omitted generate_name.
	// Keep this compatibility default local; other migrated resources differ.
	generateName.Computed = true
	generateName.Default = stringdefault.StaticString("")
	block.NestedObject.Attributes["generate_name"] = generateName
	for _, name := range []string{"annotations", "labels"} {
		attribute := block.NestedObject.Attributes[name].(schema.MapAttribute)
		attribute.Computed = true
		attribute.Description += " Omitting a previously nonempty map removes its managed keys; omitted null or empty maps retain their stored representation."
		block.NestedObject.Attributes[name] = attribute
	}
	return block
}

func serviceSpecSchema() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "Spec defines the behavior of a service. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status. Exactly one spec block is required.",
		Validators:  []validator.List{listvalidator.IsRequired(), listvalidator.SizeAtLeast(1), listvalidator.SizeAtMost(1)},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"allocate_load_balancer_node_ports": schema.BoolAttribute{
					Description: "Defines if `NodePorts` will be automatically allocated for services with type `LoadBalancer`. It may be set to `false` if the cluster load-balancer does not rely on `NodePorts`. If the caller requests specific `NodePorts` (by specifying a value), those requests will be respected, regardless of this field. This field may only be set for services with type `LoadBalancer`. Default is `true`. More info: https://kubernetes.io/docs/concepts/services-networking/service/#load-balancer-nodeport-allocation",
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(true),
				},
				"cluster_ip": schema.StringAttribute{
					Description:   "The IP address of the service. It is usually assigned randomly by the master. If an address is specified manually and is not in use by others, it will be allocated to the service; otherwise, creation of the service will fail. `None` can be specified for headless services when proxying is not required. Ignored if type is `ExternalName`. Changing a configured value forces replacement. More info: https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies",
					Optional:      true,
					Computed:      true,
					Validators:    []validator.String{serviceIPValidator{allowNone: true}},
					PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured()},
				},
				"cluster_ips": schema.ListAttribute{
					Description:   "List of IP addresses assigned to this service, and are usually assigned randomly. If an address is specified manually and is not in use by others, it will be allocated to the service; otherwise creation of the service will fail. If this field is not specified, it will be initialized from the `clusterIP` field. If this field is specified, clients must ensure that `clusterIPs[0]` and `clusterIP` have the same value. Changing a configured value forces replacement. Except for ExternalName Services, omit this argument or use null for API-assigned values instead of an empty list. More info: https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies",
					Optional:      true,
					Computed:      true,
					ElementType:   types.StringType,
					Validators:    []validator.List{listvalidator.SizeAtMost(2), listvalidator.ValueStringsAre(serviceIPValidator{allowNone: true})},
					PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplaceIfConfigured()},
				},
				"external_ips": schema.SetAttribute{
					Description: "A list of IP addresses for which nodes in the cluster will also accept traffic for this service. These IPs are not managed by Kubernetes. The user is responsible for ensuring that traffic arrives at a node with this IP. A common example is external load-balancers that are not part of the Kubernetes system. Omitting a previously nonempty set clears it; omitted null or empty sets retain their stored representation.",
					Optional:    true,
					Computed:    true,
					ElementType: types.StringType,
					Validators:  []validator.Set{setvalidator.ValueStringsAre(serviceIPValidator{})},
				},
				// Released Service state uses "" for these omitted string fields.
				// Explicit defaults preserve both no-op upgrades and root outputs.
				"external_name": schema.StringAttribute{
					Description: "The external reference that kubedns or equivalent will return as a CNAME record for this service. No proxying will be involved. Must be a valid DNS name and requires `type` to be `ExternalName`. Defaults to an empty string when no external reference is configured.",
					Optional:    true,
					Computed:    true,
					Default:     stringdefault.StaticString(""),
				},
				"external_traffic_policy": schema.StringAttribute{
					Description: "Denotes if this Service desires to route external traffic to node-local or cluster-wide endpoints. This also applies to externally accessible `ClusterIP` services with nonempty `external_ips`. `Local` preserves the client source IP and avoids a second hop for LoadBalancer and Nodeport type services, but risks potentially imbalanced traffic spreading. `Cluster` obscures the client source IP and may cause a second hop to another node, but should have good overall load-spreading. More info: https://kubernetes.io/docs/tutorials/services/source-ip/",
					Optional:    true,
					Computed:    true,
					Validators:  []validator.String{stringvalidator.OneOf("Local", "Cluster")},
				},
				"ip_families": schema.ListAttribute{
					Description: "IPFamilies is a list of IP families (e.g. IPv4, IPv6) assigned to this service. This field is usually assigned automatically based on cluster configuration and the ipFamilyPolicy field. If this field is specified manually, the requested family is available in the cluster, and ipFamilyPolicy allows it, it will be used; otherwise creation of the service will fail. This field is conditionally mutable: it allows for adding or removing a secondary IP family, but it does not allow changing the primary IP family of the Service. Except for ExternalName Services, omit this argument or use null for API-assigned values instead of an empty list.",
					Optional:    true,
					Computed:    true,
					ElementType: types.StringType,
					Validators:  []validator.List{listvalidator.SizeAtMost(2), listvalidator.ValueStringsAre(stringvalidator.OneOf("IPv4", "IPv6"))},
				},
				"ip_family_policy": schema.StringAttribute{
					Description: "IPFamilyPolicy represents the dual-stack-ness requested or required by this Service. If there is no value provided, then this field will be set to SingleStack. Services can be 'SingleStack' (a single IP family), 'PreferDualStack' (two IP families on dual-stack configured clusters or a single IP family on single-stack clusters), or 'RequireDualStack' (two IP families on dual-stack configured clusters, otherwise fail). The ipFamilies and clusterIPs fields depend on the value of this field.",
					Optional:    true,
					Computed:    true,
					Validators:  []validator.String{stringvalidator.OneOf("SingleStack", "PreferDualStack", "RequireDualStack")},
				},
				"internal_traffic_policy": schema.StringAttribute{
					Description: "Specifies if the cluster internal traffic should be routed to all endpoints or node-local endpoints only. `Cluster` routes internal traffic to a Service to all endpoints. `Local` routes traffic to node-local endpoints only, traffic is dropped if no node-local endpoints are ready. The default value is `Cluster`.",
					Optional:    true,
					Computed:    true,
					Validators:  []validator.String{stringvalidator.OneOf("Cluster", "Local")},
				},
				"load_balancer_class": schema.StringAttribute{
					Description: "The class of the load balancer implementation this Service belongs to. If specified, the value of this field must be a label-style identifier, with an optional prefix. This field can only be set when the Service type is `LoadBalancer`. If not set, the default load balancer implementation is used. This field can only be set when creating or updating a Service to type `LoadBalancer`. Changing or removing a configured class forces replacement. Defaults to an empty string when no class is configured. More info: https://kubernetes.io/docs/concepts/services-networking/service/#load-balancer-class",
					Optional:    true,
					Computed:    true,
					Default:     stringdefault.StaticString(""),
					PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIf(
						func(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
							resp.RequiresReplace = !(req.StateValue.ValueString() == "" && req.PlanValue.IsNull())
						}, "Replace when the load balancer class changes.", "Replace when the load balancer class changes.",
					)},
				},
				"load_balancer_ip": schema.StringAttribute{
					Description: "Only applies to `type = LoadBalancer`. LoadBalancer will get created with the IP specified in this field. This feature depends on whether the underlying cloud-provider supports specifying this field when a load balancer is created. This field will be ignored if the cloud-provider does not support the feature. Defaults to an empty string when no IP is requested.",
					Optional:    true,
					Computed:    true,
					Default:     stringdefault.StaticString(""),
					Validators:  []validator.String{serviceIPValidator{}},
				},
				"load_balancer_source_ranges": schema.SetAttribute{
					Description: "If specified and supported by the platform, this will restrict traffic through the cloud-provider load-balancer will be restricted to the specified client IPs. This field will be ignored if the cloud-provider does not support the feature. Omitting a previously nonempty set clears it; omitted null or empty sets retain their stored representation. More info: http://kubernetes.io/docs/user-guide/services-firewalls",
					Optional:    true,
					Computed:    true,
					ElementType: types.StringType,
					Validators:  []validator.Set{setvalidator.ValueStringsAre(serviceIPValidator{cidr: true})},
				},
				"publish_not_ready_addresses": schema.BoolAttribute{
					Description: "When set to true, indicates that DNS implementations must publish the `notReadyAddresses` of subsets for the Endpoints associated with the Service. The default value is `false`. The primary use case for setting this field is to use a StatefulSet's Headless Service to propagate `SRV` records for its Pods without respect to their readiness for purpose of peer discovery.",
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(false),
				},
				"selector": schema.MapAttribute{
					Description: "Route service traffic to pods with label keys and values matching this selector. Only applies to types `ClusterIP`, `NodePort`, and `LoadBalancer`. Omitting a previously nonempty selector clears it; an omitted null or empty selector retains its stored representation. More info: https://kubernetes.io/docs/concepts/services-networking/service/",
					Optional:    true,
					Computed:    true,
					ElementType: types.StringType,
				},
				"session_affinity": schema.StringAttribute{
					Description: "Used to maintain session affinity. Supports `ClientIP` and `None`. Defaults to `None`. More info: https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies",
					Optional:    true,
					Computed:    true,
					Default:     stringdefault.StaticString("None"),
					Validators:  []validator.String{stringvalidator.OneOf("ClientIP", "None")},
				},
				"session_affinity_config": schema.SingleNestedAttribute{
					Description: "Contains the session affinity configuration. Use object argument syntax, for example `session_affinity_config = { client_ip = { timeout_seconds = 300 } }`. Omit or set to null to retain provider-computed values, or use `{}` to request defaults.",
					Optional:    true,
					Computed:    true,
					Attributes: map[string]schema.Attribute{
						"client_ip": schema.SingleNestedAttribute{
							Description: "Client IP session affinity configuration. Omit or set to null to retain provider-computed values, or use `{}` to request defaults.",
							Optional:    true,
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"timeout_seconds": schema.Int64Attribute{
									Description: "Specifies the seconds of ClientIP session sticky time, from 1 to 86400. When omitted or removed, an existing provider-computed timeout is retained. Kubernetes defaults new ClientIP configurations to 10800 seconds.",
									Optional:    true,
									Computed:    true,
									Validators:  []validator.Int64{int64validator.Between(1, 86400)},
								},
							},
						},
					},
				},
				"type": schema.StringAttribute{
					Description: "Determines how the service is exposed. Defaults to `ClusterIP`. Valid options are `ExternalName`, `ClusterIP`, `NodePort`, and `LoadBalancer`. `ExternalName` maps to the specified `external_name`. More info: https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types",
					Optional:    true,
					Computed:    true,
					Default:     stringdefault.StaticString("ClusterIP"),
					Validators:  []validator.String{stringvalidator.OneOf("ClusterIP", "ExternalName", "NodePort", "LoadBalancer")},
				},
				"health_check_node_port": schema.Int64Attribute{
					Description:   "Specifies the Healthcheck NodePort for the service. Only effects when type is set to `LoadBalancer` and external_traffic_policy is set to `Local`. Changing a configured value forces replacement.",
					Optional:      true,
					Computed:      true,
					Validators:    []validator.Int64{int64validator.Between(1, 65535)},
					PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplaceIfConfigured()},
				},
			},
			Blocks: map[string]schema.Block{
				"port": schema.ListNestedBlock{
					Description: "The list of ports that are exposed by this service. More info: https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies",
					NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
						"app_protocol": schema.StringAttribute{
							Description: "The application protocol for this port. This field follows standard Kubernetes label syntax. Un-prefixed names are reserved for IANA standard service names (as per RFC-6335 and http://www.iana.org/assignments/service-names). Non-standard protocols should use prefixed names such as mycompany.com/my-custom-protocol. Defaults to an empty string when no application protocol is configured.",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString(""),
						},
						// SDKv2's unnamed-port state is "", including after import.
						// Keep that established output rather than converting it to null.
						"name": schema.StringAttribute{
							Description: "The name of this port within the service. All ports within the service must have unique names. Optional if only one ServicePort is defined on this service. Defaults to an empty string for an unnamed port.",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString(""),
						},
						"node_port": schema.Int64Attribute{
							Description: "The port on each node on which this service is exposed when `type` is `NodePort` or `LoadBalancer`. Usually assigned by the system. If specified, it will be allocated to the service if unused or else creation of the service will fail. Default is to auto-allocate a port if the `type` of this service requires one. For automatic allocation (`NodePort`, or `LoadBalancer` with `allocate_load_balancer_node_ports` enabled), omit `node_port` or set it to null; an explicit `0` is rejected. An explicit `0` remains allowed when node ports are not automatically allocated. More info: https://kubernetes.io/docs/concepts/services-networking/service/#type-nodeport",
							Optional:    true,
							Computed:    true,
							Validators:  []validator.Int64{int64validator.Between(0, 65535)},
						},
						"port": schema.Int64Attribute{
							Description: "The port that will be exposed by this service.",
							Required:    true,
							Validators:  []validator.Int64{int64validator.Between(1, 65535)},
						},
						"protocol": schema.StringAttribute{
							Description: "The IP protocol for this port. Supports `TCP` and `UDP`. Default is `TCP`.",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("TCP"),
							Validators:  []validator.String{stringvalidator.OneOf("TCP", "UDP", "SCTP")},
						},
						"target_port": schema.StringAttribute{
							Description: "Number or name of the port to access on the pods targeted by the service. Number must be in the range 1 to 65535. This field is ignored for services with `cluster_ip = \"None\"`. To use or retain the API-resolved target port, omit this argument or set it to null. An explicit empty string or numeric zero is rejected; configurations using the earlier SDK behavior must remove these values. More info: https://kubernetes.io/docs/concepts/services-networking/service/#defining-a-service",
							Optional:    true,
							Computed:    true,
						},
					}},
				},
			},
		},
	}
}

type serviceIPValidator struct {
	allowNone bool
	cidr      bool
}

func (v serviceIPValidator) Description(context.Context) string {
	if v.cidr {
		return "must be a valid CIDR"
	}
	if v.allowNone {
		return "must be a valid IP address or None"
	}
	return "must be a valid IP address"
}

func (v serviceIPValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v serviceIPValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	valid := net.ParseIP(value) != nil || (v.allowNone && value == "None")
	if v.cidr {
		_, _, err := net.ParseCIDR(value)
		valid = err == nil
	}
	if !valid {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid attribute value", v.Description(ctx))
	}
}
