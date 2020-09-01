package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dataSourceKubernetesService() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceKubernetesServiceRead,

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("service", false),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec defines the behavior of a service. https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#spec-and-status",
				MaxItems:    1,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cluster_ip": {
							Type:        schema.TypeString,
							Description: "The IP address of the service. It is usually assigned randomly by the master. If an address is specified manually and is not in use by others, it will be allocated to the service; otherwise, creation of the service will fail. `None` can be specified for headless services when proxying is not required. Ignored if type is `ExternalName`. More info: http://kubernetes.io/docs/user-guide/services#virtual-ips-and-service-proxies",
							Computed:    true,
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
							Description: "The list of ports that are exposed by this service. More info: http://kubernetes.io/docs/user-guide/services#virtual-ips-and-service-proxies",
							MinItems:    1,
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:        schema.TypeString,
										Description: "The name of this port within the service. All ports within the service must have unique names. Optional if only one ServicePort is defined on this service.",
										Computed:    true,
									},
									"node_port": {
										Type:        schema.TypeInt,
										Description: "The port on each node on which this service is exposed when `type` is `NodePort` or `LoadBalancer`. Usually assigned by the system. If specified, it will be allocated to the service if unused or else creation of the service will fail. Default is to auto-allocate a port if the `type` of this service requires one. More info: http://kubernetes.io/docs/user-guide/services#type--nodeport",
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
										Description: "Number or name of the port to access on the pods targeted by the service. Number must be in the range 1 to 65535. This field is ignored for services with `cluster_ip = \"None\"`. More info: http://kubernetes.io/docs/user-guide/services#defining-a-service",
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
							Description: "Route service traffic to pods with label keys and values matching this selector. Only applies to types `ClusterIP`, `NodePort`, and `LoadBalancer`. More info: http://kubernetes.io/docs/user-guide/services#overview",
							Computed:    true,
						},
						"session_affinity": {
							Type:        schema.TypeString,
							Description: "Used to maintain session affinity. Supports `ClientIP` and `None`. Defaults to `None`. More info: http://kubernetes.io/docs/user-guide/services#virtual-ips-and-service-proxies",
							Computed:    true,
						},
						"type": {
							Type:        schema.TypeString,
							Description: "Determines how the service is exposed. Defaults to `ClusterIP`. Valid options are `ExternalName`, `ClusterIP`, `NodePort`, and `LoadBalancer`. `ExternalName` maps to the specified `external_name`. More info: http://kubernetes.io/docs/user-guide/services#overview",
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
			"load_balancer_ingress": {
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
	}
}

func dataSourceKubernetesServiceRead(d *schema.ResourceData, meta interface{}) error {
	om := meta_v1.ObjectMeta{
		Namespace: d.Get("metadata.0.namespace").(string),
		Name:      d.Get("metadata.0.name").(string),
	}
	d.SetId(buildId(om))

	return resourceKubernetesServiceRead(d, meta)
}
