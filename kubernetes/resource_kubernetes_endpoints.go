package kubernetes

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

func resourceKubernetesEndpoints() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesEndpointsCreate,
		Read:   resourceKubernetesEndpointsRead,
		Update: resourceKubernetesEndpointsUpdate,
		Delete: resourceKubernetesEndpointsDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("endpoints", true),
			"subset": {
				Type:        schema.TypeSet,
				Description: "Set of addresses and ports that comprise a service. More info: https://kubernetes.io/docs/concepts/services-networking/service/#services-without-selectors",
				Optional:    true,
				Elem:        schemaEndpointsSubset(),
				Set:         hashEndpointsSubset(),
			},
		},
	}
}

func resourceKubernetesEndpointsCreate(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	ep := api.Endpoints{
		ObjectMeta: metadata,
		Subsets:    expandEndpointsSubsets(d.Get("subset").(*schema.Set)),
	}
	log.Printf("[INFO] Creating new endpoints: %#v", ep)
	out, err := conn.CoreV1().Endpoints(metadata.Namespace).Create(ctx, &ep, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("Failed to create endpoints because: %s", err)
	}
	log.Printf("[INFO] Submitted new endpoints: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesEndpointsRead(d, meta)
}

func resourceKubernetesEndpointsRead(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return fmt.Errorf("Failed to read endpoints because: %s", err)
	}

	log.Printf("[INFO] Reading endpoints %s", name)
	ep, err := conn.CoreV1().Endpoints(namespace).Get(ctx, name, metav1.GetOptions{})
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
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return fmt.Errorf("Failed to update endpoints because: %s", err)
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("subset") {
		subsets := expandEndpointsSubsets(d.Get("subset").(*schema.Set))
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
	out, err := conn.CoreV1().Endpoints(namespace).Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("Failed to update endpoints: %s", err)
	}
	log.Printf("[INFO] Submitted updated endpoints: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesEndpointsRead(d, meta)
}

func resourceKubernetesEndpointsDelete(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return fmt.Errorf("Failed to delete endpoints because: %s", err)
	}
	log.Printf("[INFO] Deleting endpoints: %#v", name)
	err = conn.CoreV1().Endpoints(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("Failed to delete endpoints because: %s", err)
	}
	log.Printf("[INFO] Endpoints %s deleted", name)
	d.SetId("")

	return nil
}
