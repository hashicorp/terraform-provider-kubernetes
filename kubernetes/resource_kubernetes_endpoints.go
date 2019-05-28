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

func resourceKubernetesEndpoints() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesEndpointsCreate,
		Read:   resourceKubernetesEndpointsRead,
		Exists: resourceKubernetesEndpointsExists,
		Update: resourceKubernetesEndpointsUpdate,
		Delete: resourceKubernetesEndpointsDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("endpoints", true),
			"subset": {
				Type:        schema.TypeList,
				Description: "Set of addresses and ports that comprise a service. More info: https://kubernetes.io/docs/concepts/services-networking/service/#services-without-selectors",
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"address": {
							Type:        schema.TypeList,
							Description: "IP address which offers the related ports that are marked as ready. These endpoints should be considered safe for load balancers and clients to utilize.",
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
						"not_ready_address": {
							Type:        schema.TypeList,
							Description: "IP address which offers the related ports but is not currently marked as ready because it have not yet finished starting, have recently failed a readiness check, or have recently failed a liveness check.",
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
						"port": {
							Type:        schema.TypeList,
							Description: "Port number available on the related IP addresses.",
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

func resourceKubernetesEndpointsCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	ep := api.Endpoints{
		ObjectMeta: metadata,
		Subsets:    expandEndpointsSubsets(d.Get("subset").([]interface{})),
	}
	log.Printf("[INFO] Creating new endpoints: %#v", ep)
	out, err := conn.CoreV1().Endpoints(metadata.Namespace).Create(&ep)
	if err != nil {
		return fmt.Errorf("Failed to create endpoints because: %s", err)
	}
	log.Printf("[INFO] Submitted new endpoints: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesEndpointsRead(d, meta)
}

func resourceKubernetesEndpointsRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return fmt.Errorf("Failed to read endpoints because: %s", err)
	}

	log.Printf("[INFO] Reading endpoints %s", name)
	ep, err := conn.CoreV1().Endpoints(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return fmt.Errorf("Failed to read endpoint because: %s", err)
	}
	log.Printf("[INFO] Received endpoints: %#v", ep)
	err = d.Set("metadata", flattenMetadata(ep.ObjectMeta, d))
	if err != nil {
		return fmt.Errorf("Failed to read endpoints because: %s", err)
	}

	flattened := flattenEndpointsSubsets(ep.Subsets)
	log.Printf("[DEBUG] Flattened endpoints subset: %#v", flattened)
	err = d.Set("subset", flattened)
	if err != nil {
		return fmt.Errorf("Failed to read endpoints because: %s", err)
	}

	return nil
}

func resourceKubernetesEndpointsUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return fmt.Errorf("Failed to update endpoints because: %s", err)
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("subset") {
		subsets := expandEndpointsSubsets(d.Get("subset").([]interface{}))
		ops = append(ops, &ReplaceOperation{
			Path:  "/subsets",
			Value: subsets,
		})
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating endpoints %q: %v", name, string(data))
	out, err := conn.CoreV1().Endpoints(namespace).Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return fmt.Errorf("Failed to update endpoints: %s", err)
	}
	log.Printf("[INFO] Submitted updated endpoints: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesEndpointsRead(d, meta)
}

func resourceKubernetesEndpointsDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return fmt.Errorf("Failed to delete endpoints because: %s", err)
	}
	log.Printf("[INFO] Deleting endpoints: %#v", name)
	err = conn.CoreV1().Endpoints(namespace).Delete(name, &meta_v1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("Failed to delete endpoints because: %s", err)
	}
	log.Printf("[INFO] Endpoints %s deleted", name)
	d.SetId("")

	return nil
}

func resourceKubernetesEndpointsExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking endpoints %s", name)
	_, err = conn.CoreV1().Endpoints(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
