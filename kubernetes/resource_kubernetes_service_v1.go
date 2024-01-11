// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

func resourceKubernetesServiceV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesServiceV1Create,
		ReadContext:   resourceKubernetesServiceV1Read,
		UpdateContext: resourceKubernetesServiceV1Update,
		DeleteContext: resourceKubernetesServiceV1Delete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
		},
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    resourceKubernetesServiceV0().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceKubernetesServiceStateUpgradeV0,
				Version: 0,
			},
		},
		Schema: resourceKubernetesServiceSchemaV1(),
	}
}

func resourceKubernetesServiceSchemaV1() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"metadata": namespacedMetadataSchema("service", true),
		"spec": {
			Type:        schema.TypeList,
			Description: "Spec defines the behavior of a service. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status",
			Required:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"allocate_load_balancer_node_ports": {
						Type:        schema.TypeBool,
						Description: "Defines if `NodePorts` will be automatically allocated for services with type `LoadBalancer`. It may be set to `false` if the cluster load-balancer does not rely on `NodePorts`.  If the caller requests specific `NodePorts` (by specifying a value), those requests will be respected, regardless of this field. This field may only be set for services with type `LoadBalancer`. Default is `true`. More info: https://kubernetes.io/docs/concepts/services-networking/service/#load-balancer-nodeport-allocation",
						Optional:    true,
						Default:     true,
					},
					"cluster_ip": {
						Type:        schema.TypeString,
						Description: "The IP address of the service. It is usually assigned randomly by the master. If an address is specified manually and is not in use by others, it will be allocated to the service; otherwise, creation of the service will fail. `None` can be specified for headless services when proxying is not required. Ignored if type is `ExternalName`. More info: https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies",
						Optional:    true,
						ForceNew:    true,
						Computed:    true,
						ValidateFunc: validation.Any(
							validation.StringInSlice([]string{corev1.ClusterIPNone}, false),
							validation.IsIPAddress,
						),
					},
					"cluster_ips": {
						Type:        schema.TypeList,
						Description: "List of IP addresses assigned to this service, and are usually assigned randomly. If an address is specified manually and is not in use by others, it will be allocated to the service; otherwise creation of the service will fail. If this field is not specified, it will be initialized from the `clusterIP` field. If this field is specified, clients must ensure that `clusterIPs[0]` and `clusterIP` have the same value. More info: https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies",
						Optional:    true,
						ForceNew:    true,
						Computed:    true,
						MaxItems:    2,
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
						Optional:    true,
						Elem: &schema.Schema{
							Type:         schema.TypeString,
							ValidateFunc: validation.IsIPAddress,
						},
					},
					"external_name": {
						Type:        schema.TypeString,
						Description: "The external reference that kubedns or equivalent will return as a CNAME record for this service. No proxying will be involved. Must be a valid DNS name and requires `type` to be `ExternalName`.",
						Optional:    true,
					},
					"external_traffic_policy": {
						Type:        schema.TypeString,
						Description: "Denotes if this Service desires to route external traffic to node-local or cluster-wide endpoints. `Local` preserves the client source IP and avoids a second hop for LoadBalancer and Nodeport type services, but risks potentially imbalanced traffic spreading. `Cluster` obscures the client source IP and may cause a second hop to another node, but should have good overall load-spreading. More info: https://kubernetes.io/docs/tutorials/services/source-ip/",
						Optional:    true,
						Computed:    true,
						ValidateFunc: validation.StringInSlice([]string{
							string(corev1.ServiceExternalTrafficPolicyTypeLocal),
							string(corev1.ServiceExternalTrafficPolicyTypeCluster),
						}, false),
					},
					"ip_families": {
						Type:        schema.TypeList,
						Description: "IPFamilies is a list of IP families (e.g. IPv4, IPv6) assigned to this service. This field is usually assigned automatically based on cluster configuration and the ipFamilyPolicy field. If this field is specified manually, the requested family is available in the cluster, and ipFamilyPolicy allows it, it will be used; otherwise creation of the service will fail. This field is conditionally mutable: it allows for adding or removing a secondary IP family, but it does not allow changing the primary IP family of the Service.",
						Optional:    true,
						Computed:    true,
						MaxItems:    2,
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
						Optional:    true,
						Computed:    true,
						ValidateFunc: validation.StringInSlice([]string{
							string(corev1.ServiceInternalTrafficPolicyCluster),
							string(corev1.ServiceInternalTrafficPolicyLocal),
						}, false),
					},
					"load_balancer_class": {
						Type:        schema.TypeString,
						Description: "The class of the load balancer implementation this Service belongs to. If specified, the value of this field must be a label-style identifier, with an optional prefix. This field can only be set when the Service type is `LoadBalancer`. If not set, the default load balancer implementation is used. This field can only be set when creating or updating a Service to type `LoadBalancer`. More info: https://kubernetes.io/docs/concepts/services-networking/service/#load-balancer-class",
						Optional:    true,
						ForceNew:    true,
					},
					"load_balancer_ip": {
						Type:         schema.TypeString,
						Description:  "Only applies to `type = LoadBalancer`. LoadBalancer will get created with the IP specified in this field. This feature depends on whether the underlying cloud-provider supports specifying this field when a load balancer is created. This field will be ignored if the cloud-provider does not support the feature.",
						Optional:     true,
						ValidateFunc: validation.IsIPAddress,
					},
					"load_balancer_source_ranges": {
						Type:        schema.TypeSet,
						Description: "If specified and supported by the platform, this will restrict traffic through the cloud-provider load-balancer will be restricted to the specified client IPs. This field will be ignored if the cloud-provider does not support the feature. More info: http://kubernetes.io/docs/user-guide/services-firewalls",
						Optional:    true,
						Elem: &schema.Schema{
							Type:         schema.TypeString,
							ValidateFunc: validation.IsCIDR,
						},
					},
					"port": {
						Type:        schema.TypeList,
						Description: "The list of ports that are exposed by this service. More info: https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies",
						Optional:    true,
						MinItems:    1,
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
									Optional:    true,
								},
								"node_port": {
									Type:         schema.TypeInt,
									Description:  "The port on each node on which this service is exposed when `type` is `NodePort` or `LoadBalancer`. Usually assigned by the system. If specified, it will be allocated to the service if unused or else creation of the service will fail. Default is to auto-allocate a port if the `type` of this service requires one. More info: https://kubernetes.io/docs/concepts/services-networking/service/#type-nodeport",
									Computed:     true,
									Optional:     true,
									ValidateFunc: validation.IsPortNumberOrZero,
								},
								"port": {
									Type:         schema.TypeInt,
									Description:  "The port that will be exposed by this service.",
									Required:     true,
									ValidateFunc: validation.IsPortNumber,
								},
								"protocol": {
									Type:        schema.TypeString,
									Description: "The IP protocol for this port. Supports `TCP` and `UDP`. Default is `TCP`.",
									Optional:    true,
									Default:     string(corev1.ProtocolTCP),
									ValidateFunc: validation.StringInSlice([]string{
										string(corev1.ProtocolTCP),
										string(corev1.ProtocolUDP),
										string(corev1.ProtocolSCTP),
									}, false),
								},
								"target_port": {
									Type:        schema.TypeString,
									Description: "Number or name of the port to access on the pods targeted by the service. Number must be in the range 1 to 65535. This field is ignored for services with `cluster_ip = \"None\"`. More info: https://kubernetes.io/docs/concepts/services-networking/service/#defining-a-service",
									Optional:    true,
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
						Optional:    true,
					},
					"session_affinity": {
						Type:        schema.TypeString,
						Description: "Used to maintain session affinity. Supports `ClientIP` and `None`. Defaults to `None`. More info: https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies",
						Optional:    true,
						Default:     string(corev1.ServiceAffinityNone),
						ValidateFunc: validation.StringInSlice([]string{
							string(corev1.ServiceAffinityClientIP),
							string(corev1.ServiceAffinityNone),
						}, false),
					},
					"session_affinity_config": {
						Type:        schema.TypeList,
						Description: "Contains the configurations of session affinity. More info: https://kubernetes.io/docs/concepts/services-networking/service/#proxy-mode-ipvs",
						Optional:    true,
						Computed:    true,
						MaxItems:    1,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"client_ip": {
									Type:        schema.TypeList,
									Description: "Contains the configurations of Client IP based session affinity.",
									Optional:    true,
									Computed:    true,
									MaxItems:    1,
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"timeout_seconds": {
												Type:         schema.TypeInt,
												Description:  "Specifies the seconds of `ClientIP` type session sticky time. The value must be > 0 and <= 86400(for 1 day) if `ServiceAffinity` == `ClientIP`.",
												Optional:     true,
												Computed:     true,
												ValidateFunc: validation.IntBetween(1, 86400),
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
						Optional:    true,
						Default:     string(corev1.ServiceTypeClusterIP),
						ValidateFunc: validation.StringInSlice([]string{
							string(corev1.ServiceTypeClusterIP),
							string(corev1.ServiceTypeExternalName),
							string(corev1.ServiceTypeNodePort),
							string(corev1.ServiceTypeLoadBalancer),
						}, false),
					},
					"health_check_node_port": {
						Type:         schema.TypeInt,
						Description:  "Specifies the Healthcheck NodePort for the service. Only effects when type is set to `LoadBalancer` and external_traffic_policy is set to `Local`.",
						Optional:     true,
						Computed:     true,
						ForceNew:     true,
						ValidateFunc: validation.IsPortNumber,
					},
				},
			},
		},
		"wait_for_load_balancer": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     true,
			Description: "Terraform will wait for the load balancer to have at least 1 endpoint before considering the resource created.",
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
	}
}

func resourceKubernetesServiceV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	svc := corev1.Service{
		ObjectMeta: metadata,
		Spec:       expandServiceSpec(d.Get("spec").([]interface{})),
	}
	log.Printf("[INFO] Creating new service: %#v", svc)
	out, err := conn.CoreV1().Services(metadata.Namespace).Create(ctx, &svc, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Submitted new service: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	if out.Spec.Type == corev1.ServiceTypeLoadBalancer && d.Get("wait_for_load_balancer").(bool) {
		log.Printf("[DEBUG] Waiting for load balancer to assign IP/hostname")

		err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutCreate), func() *retry.RetryError {
			svc, err := conn.CoreV1().Services(out.Namespace).Get(ctx, out.Name, metav1.GetOptions{})
			if err != nil {
				log.Printf("[DEBUG] Received error: %#v", err)
				return retry.NonRetryableError(err)
			}

			lbIngress := svc.Status.LoadBalancer.Ingress

			log.Printf("[INFO] Received service status: %#v", svc.Status)
			if len(lbIngress) > 0 {
				return nil
			}

			return retry.RetryableError(fmt.Errorf(
				"Waiting for service %q to assign IP/hostname for a load balancer", d.Id()))
		})
		if err != nil {
			lastWarnings, wErr := getLastWarningsForObject(ctx, conn, out.ObjectMeta, "Service", 3)
			if wErr != nil {
				return diag.FromErr(wErr)
			}
			return diag.Errorf("%s%s", err, stringifyEvents(lastWarnings))
		}
	}

	return resourceKubernetesServiceV1Read(ctx, d, meta)
}

func resourceKubernetesServiceV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesServiceV1Exists(ctx, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		d.SetId("")
		return diag.Diagnostics{}
	}
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Reading service %s", name)
	svc, err := conn.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Received service: %#v", svc)
	err = d.Set("metadata", flattenMetadata(svc.ObjectMeta, d, meta))
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

	flattened := flattenServiceSpec(svc.Spec)
	log.Printf("[DEBUG] Flattened service spec: %#v", flattened)
	err = d.Set("spec", flattened)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesServiceV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("spec") {
		serverVersion, err := getServerVersion(conn)
		if err != nil {
			return diag.FromErr(err)
		}
		diffOps := patchServiceSpec("spec.0.", "/spec/", d, serverVersion)
		ops = append(ops, diffOps...)
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating service %q: %v", name, string(data))
	out, err := conn.CoreV1().Services(namespace).Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.Errorf("Failed to update service: %s", err)
	}
	log.Printf("[INFO] Submitted updated service: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesServiceV1Read(ctx, d, meta)
}

func resourceKubernetesServiceV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting service: %#v", name)
	err = conn.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return nil
		}
		return diag.FromErr(err)
	}

	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutDelete), func() *retry.RetryError {
		_, err := conn.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
				return nil
			}
			return retry.NonRetryableError(err)
		}

		e := fmt.Errorf("Service (%s) still exists", d.Id())
		return retry.RetryableError(e)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Service %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesServiceV1Exists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking service %s", name)
	_, err = conn.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
