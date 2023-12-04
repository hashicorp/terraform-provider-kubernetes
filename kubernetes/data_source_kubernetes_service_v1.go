// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dataSourceKubernetesServiceV1() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceKubernetesServiceV1Read,
		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("service", false),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec defines the behavior of a service. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"allocate_load_balancer_node_ports": {
							Type:        schema.TypeBool,
							Description: "Defines if `NodePorts` will be automatically allocated for services with type `LoadBalancer`. It may be set to `false` if the cluster load-balancer does not rely on `NodePorts`.  If the caller requests specific `NodePorts` (by specifying a value), those requests will be respected, regardless of this field. This field may only be set for services with type `LoadBalancer`. Default is `true`.",
							Optional:    true,
							Default:     true,
						},
						"cluster_ip": {
							Type:        schema.TypeString,
							Description: "The IP address of the service. It is usually assigned randomly by the master. If an address is specified manually and is not in use by others, it will be allocated to the service; otherwise, creation of the service will fail. `None` can be specified for headless services when proxying is not required. Ignored if type is `ExternalName`. More info: https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies",
							Computed:    true,
						},
						"cluster_ips": {
							Type:        schema.TypeList,
							Description: "List of IP addresses assigned to this service, and are usually assigned randomly. If an address is specified manually and is not in use by others, it will be allocated to the service; otherwise creation of the service will fail. If this field is not specified, it will be initialized from the `clusterIP` field. If this field is specified, clients must ensure that `clusterIPs[0]` and `clusterIP` have the same value. More info: https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies",
							Computed:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
								ValidateFunc: validation.Any(
									validation.StringInSlice([]string{corev1.ClusterIPNone}, false),
									validation.IsIPAddress,
								),
							},
						},
						"external_ips": {
							Type:        schema.TypeSet,
							Description: "A list of IP addresses for which nodes in the cluster will also accept traffic for this service. These IPs are not managed by Kubernetes. The user is responsible for ensuring that traffic arrives at a node with this IP.  A common example is external load-balancers that are not part of the Kubernetes system.",
							Elem:        &schema.Schema{Type: schema.TypeString},
							Set:         schema.HashString,
							Computed:    true,
						},
						"external_name": {
							Type:        schema.TypeString,
							Description: "The external reference that kubedns or equivalent will return as a CNAME record for this service. No proxying will be involved. Must be a valid DNS name and requires `type` to be `ExternalName`.",
							Computed:    true,
						},
						"external_traffic_policy": {
							Type:         schema.TypeString,
							Description:  "Denotes if this Service desires to route external traffic to node-local or cluster-wide endpoints. `Local` preserves the client source IP and avoids a second hop for LoadBalancer and Nodeport type services, but risks potentially imbalanced traffic spreading. `Cluster` obscures the client source IP and may cause a second hop to another node, but should have good overall load-spreading. More info: https://kubernetes.io/docs/tutorials/services/source-ip/",
							Optional:     true,
							Computed:     true,
							ValidateFunc: validation.StringInSlice([]string{"Local", "Cluster"}, false),
						},
						"ip_families": {
							Type:        schema.TypeList,
							Description: "IPFamilies is a list of IP families (e.g. IPv4, IPv6) assigned to this service. This field is usually assigned automatically based on cluster configuration and the ipFamilyPolicy field. If this field is specified manually, the requested family is available in the cluster, and ipFamilyPolicy allows it, it will be used; otherwise creation of the service will fail. This field is conditionally mutable: it allows for adding or removing a secondary IP family, but it does not allow changing the primary IP family of the Service.",
							Optional:    true,
							Computed:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
								ValidateFunc: validation.StringInSlice([]string{
									string(corev1.IPv4Protocol),
									string(corev1.IPv6Protocol),
								}, false),
							},
						},
						"ip_family_policy": {
							Type:        schema.TypeString,
							Description: "IPFamilyPolicy represents the dual-stack-ness requested or required by this Service. If there is no value provided, then this field will be set to SingleStack. Services can be 'SingleStack' (a single IP family), 'PreferDualStack' (two IP families on dual-stack configured clusters or a single IP family on single-stack clusters), or 'RequireDualStack' (two IP families on dual-stack configured clusters, otherwise fail). The ipFamilies and clusterIPs fields depend on the value of this field.",
							Optional:    true,
							Computed:    true,
							ValidateFunc: validation.StringInSlice([]string{
								string(corev1.IPFamilyPolicySingleStack),
								string(corev1.IPFamilyPolicyPreferDualStack),
								string(corev1.IPFamilyPolicyRequireDualStack),
							}, false),
						},
						"internal_traffic_policy": {
							Type:        schema.TypeString,
							Description: "Specifies if the cluster internal traffic should be routed to all endpoints or node-local endpoints only. `Cluster` routes internal traffic to a Service to all endpoints. `Local` routes traffic to node-local endpoints only, traffic is dropped if no node-local endpoints are ready. The default value is `Cluster`.",
							Computed:    true,
						},
						"load_balancer_class": {
							Type:        schema.TypeString,
							Description: "The class of the load balancer implementation this Service belongs to. If specified, the value of this field must be a label-style identifier, with an optional prefix. This field can only be set when the Service type is `LoadBalancer`. If not set, the default load balancer implementation is used. This field can only be set when creating or updating a Service to type `LoadBalancer`. More info: https://kubernetes.io/docs/concepts/services-networking/service/#load-balancer-class",
							Optional:    true,
						},
						"load_balancer_ip": {
							Type:        schema.TypeString,
							Description: "Only applies to `type = LoadBalancer`. LoadBalancer will get created with the IP specified in this field. This feature depends on whether the underlying cloud-provider supports specifying this field when a load balancer is created. This field will be ignored if the cloud-provider does not support the feature.",
							Computed:    true,
						},
						"load_balancer_source_ranges": {
							Type:        schema.TypeSet,
							Description: "If specified and supported by the platform, this will restrict traffic through the cloud-provider load-balancer will be restricted to the specified client IPs. This field will be ignored if the cloud-provider does not support the feature. More info: http://kubernetes.io/docs/user-guide/services-firewalls",
							Elem:        &schema.Schema{Type: schema.TypeString},
							Set:         schema.HashString,
							Computed:    true,
						},
						"port": {
							Type:        schema.TypeList,
							Description: "The list of ports that are exposed by this service. More info: https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies",
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"app_protocol": {
										Type:        schema.TypeString,
										Description: "The application protocol for this port. This field follows standard Kubernetes label syntax. Un-prefixed names are reserved for IANA standard service names (as per RFC-6335 and http://www.iana.org/assignments/service-names). Non-standard protocols should use prefixed names such as mycompany.com/my-custom-protocol.",
										Optional:    true,
									},
									"name": {
										Type:        schema.TypeString,
										Description: "The name of this port within the service. All ports within the service must have unique names. Optional if only one ServicePort is defined on this service.",
										Computed:    true,
									},
									"node_port": {
										Type:        schema.TypeInt,
										Description: "The port on each node on which this service is exposed when `type` is `NodePort` or `LoadBalancer`. Usually assigned by the system. If specified, it will be allocated to the service if unused or else creation of the service will fail. Default is to auto-allocate a port if the `type` of this service requires one. More info: https://kubernetes.io/docs/concepts/services-networking/service/#type-nodeport",
										Computed:    true,
									},
									"port": {
										Type:        schema.TypeInt,
										Description: "The port that will be exposed by this service.",
										Computed:    true,
									},
									"protocol": {
										Type:        schema.TypeString,
										Description: "The IP protocol for this port. Supports `TCP` and `UDP`. Default is `TCP`.",
										Computed:    true,
									},
									"target_port": {
										Type:        schema.TypeString,
										Description: "Number or name of the port to access on the pods targeted by the service. Number must be in the range 1 to 65535. This field is ignored for services with `cluster_ip = \"None\"`. More info: https://kubernetes.io/docs/concepts/services-networking/service/#defining-a-service",
										Computed:    true,
									},
								},
							},
						},
						"publish_not_ready_addresses": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "When set to true, indicates that DNS implementations must publish the `notReadyAddresses` of subsets for the Endpoints associated with the Service. The default value is `false`. The primary use case for setting this field is to use a StatefulSet's Headless Service to propagate `SRV` records for its Pods without respect to their readiness for purpose of peer discovery.",
						},
						"selector": {
							Type:        schema.TypeMap,
							Description: "Route service traffic to pods with label keys and values matching this selector. Only applies to types `ClusterIP`, `NodePort`, and `LoadBalancer`. More info: https://kubernetes.io/docs/concepts/services-networking/service/",
							Computed:    true,
						},
						"session_affinity": {
							Type:        schema.TypeString,
							Description: "Used to maintain session affinity. Supports `ClientIP` and `None`. Defaults to `None`. More info: https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies",
							Computed:    true,
						},
						"session_affinity_config": {
							Type:        schema.TypeList,
							Description: "Contains the configurations of session affinity. More info: https://kubernetes.io/docs/concepts/services-networking/service/#proxy-mode-ipvs",
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"client_ip": {
										Type:        schema.TypeList,
										Description: "Contains the configurations of Client IP based session affinity",
										Optional:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"timeout_seconds": {
													Type:        schema.TypeInt,
													Description: "Specifies the seconds of ClientIP type session sticky time.",
													Optional:    true,
												},
											},
										},
									},
								},
							},
						},
						"type": {
							Type:        schema.TypeString,
							Description: "Determines how the service is exposed. Defaults to `ClusterIP`. Valid options are `ExternalName`, `ClusterIP`, `NodePort`, and `LoadBalancer`. `ExternalName` maps to the specified `external_name`. More info: https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types",
							Computed:    true,
						},
						"health_check_node_port": {
							Type:        schema.TypeInt,
							Description: "Specifies the Healthcheck NodePort for the service. Only effects when type is set to `LoadBalancer` and external_traffic_policy is set to `Local`.",
							Computed:    true,
						},
					},
				},
			},
			"status": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"load_balancer": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"ingress": {
										Type:     schema.TypeList,
										Computed: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"ip": {
													Type:     schema.TypeString,
													Computed: true,
												},
												"hostname": {
													Type:     schema.TypeString,
													Computed: true,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceKubernetesServiceV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))

	om := metav1.ObjectMeta{
		Namespace: metadata.Namespace,
		Name:      metadata.Name,
	}
	d.SetId(buildId(om))

	log.Printf("[INFO] Reading service %s", metadata.Name)
	svc, err := conn.CoreV1().Services(metadata.Namespace).Get(ctx, metadata.Name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Received service: %#v", svc)

	err = d.Set("metadata", flattenMetadataFields(svc.ObjectMeta))
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("status", []interface{}{
		map[string][]interface{}{
			"load_balancer": flattenLoadBalancerStatus(svc.Status.LoadBalancer),
		},
	})
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("spec", flattenServiceSpec(svc.Spec))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
