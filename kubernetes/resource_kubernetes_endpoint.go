package kubernetes

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
	kubernetes "k8s.io/client-go/kubernetes"
)

func resourceKubernetesEndpoint() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesEndpointCreate,
		Read:   resourceKubernetesEndpointRead,
		Exists: resourceKubernetesEndpointExists,
		Update: resourceKubernetesEndpointUpdate,
		Delete: resourceKubernetesEndpointDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("endpoints", true),
			"subsets": {
				Type:        schema.TypeList,
				Description: "Sets of addresses and ports that comprise a service. More info: https://kubernetes.io/docs/concepts/services-networking/service/#services-without-selectors",
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"addresses": {
							Type:        schema.TypeList,
							Description: "IP addresses which offer the related ports that are marked as ready. These endpoints should be considered safe for load balancers and clients to utilize.",
							Optional:    true,
							MinItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"ip": {
										Type:        schema.TypeString,
										Description: "The IP of this endpoint. May not be loopback (127.0.0.0/8), link-local (169.254.0.0/16), or link-local multicast ((224.0.0.0/24).",
										Required:    true,
									},
									"hostname": {
										Type:        schema.TypeString,
										Description: "The Hostname of this endpoint.",
										Optional:    true,
									},
									"node_name": {
										Type:        schema.TypeString,
										Description: "Node hosting this endpoint. This can be used to determine endpoints local to a node.",
										Optional:    true,
									},
								},
							},
						},
						"not_ready_addresses": {
							Type:        schema.TypeList,
							Description: "IP addresses which offer the related ports but are not currently marked as ready because they have not yet finished starting, have recently failed a readiness check, or have recently failed a liveness check.",
							Optional:    true,
							MinItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"ip": {
										Type:        schema.TypeString,
										Description: "The IP of this endpoint. May not be loopback (127.0.0.0/8), link-local (169.254.0.0/16), or link-local multicast ((224.0.0.0/24).",
										Required:    true,
									},
									"hostname": {
										Type:        schema.TypeString,
										Description: "The Hostname of this endpoint.",
										Optional:    true,
									},
									"node_name": {
										Type:        schema.TypeString,
										Description: "Node hosting this endpoint. This can be used to determine endpoints local to a node.",
										Optional:    true,
									},
								},
							},
						},
						"ports": {
							Type:        schema.TypeList,
							Description: "Port numbers available on the related IP addresses.",
							Optional:    true,
							MinItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:        schema.TypeString,
										Description: "The name of this port within the endpoint. Must be a DNS_LABEL. Optional if only one Port is defined on this endpoint.",
										Optional:    true,
									},
									"port": {
										Type:        schema.TypeInt,
										Description: "The port that will be exposed by this endpoint.",
										Required:    true,
									},
									"protocol": {
										Type:        schema.TypeString,
										Description: "The IP protocol for this port. Supports `TCP` and `UDP`. Default is `TCP`.",
										Optional:    true,
										Default:     "TCP",
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

func resourceKubernetesEndpointCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	ep := api.Endpoints{
		ObjectMeta: metadata,
		Subsets:    expandEndpointSubsets(d.Get("subsets").([]interface{})),
	}
	log.Printf("[INFO] Creating new endpoint: %#v", ep)
	out, err := conn.CoreV1().Endpoints(metadata.Namespace).Create(&ep)
	if err != nil {
		return fmt.Errorf("Failed to create endpoint because: %s", err)
	}
	log.Printf("[INFO] Submitted new endpoint: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesEndpointRead(d, meta)
}

func resourceKubernetesEndpointRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return fmt.Errorf("Failed to read endpoint because: %s", err)
	}

	log.Printf("[INFO] Reading endpoint %s", name)
	ep, err := conn.CoreV1().Endpoints(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return fmt.Errorf("Failed to read endpoint because: %s", err)
	}
	log.Printf("[INFO] Received endpoint: %#v", ep)
	err = d.Set("metadata", flattenMetadata(ep.ObjectMeta))
	if err != nil {
		return fmt.Errorf("Failed to read endpoint because: %s", err)
	}

	flattened := flattenEndpointSubsets(ep.Subsets)
	log.Printf("[DEBUG] Flattened endpoint subset: %#v", flattened)
	err = d.Set("subsets", flattened)
	if err != nil {
		return fmt.Errorf("Failed to read endpoint because: %s", err)
	}

	return nil
}

func resourceKubernetesEndpointUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return fmt.Errorf("Failed to update endpoint because: %s", err)
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("subsets") {
		subsets := expandEndpointSubsets(d.Get("subsets").([]interface{}))
		ops = append(ops, &ReplaceOperation{
			Path:  "/subsets",
			Value: subsets,
		})
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating endpoint %q: %v", name, string(data))
	out, err := conn.CoreV1().Endpoints(namespace).Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return fmt.Errorf("Failed to update endpoint: %s", err)
	}
	log.Printf("[INFO] Submitted updated endpoint: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesEndpointRead(d, meta)
}

func resourceKubernetesEndpointDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return fmt.Errorf("Failed to delete endpoint because: %s", err)
	}
	log.Printf("[INFO] Deleting endpoint: %#v", name)
	err = conn.CoreV1().Endpoints(namespace).Delete(name, &meta_v1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("Failed to delete endpoint because: %s", err)
	}
	log.Printf("[INFO] Endpoint %s deleted", name)
	d.SetId("")

	return nil
}

func resourceKubernetesEndpointExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking endpoint %s", name)
	_, err = conn.CoreV1().Endpoints(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
