package v1

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	providermetav1 "github.com/hashicorp/terraform-provider-kubernetes/kubernetes/meta/v1"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes/structures"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

func ResourceKubernetesEndpoints() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesEndpointsCreate,
		ReadContext:   resourceKubernetesEndpointsRead,
		UpdateContext: resourceKubernetesEndpointsUpdate,
		DeleteContext: resourceKubernetesEndpointsDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"metadata": providermetav1.NamespacedMetadataSchema("endpoints", true),
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

func resourceKubernetesEndpointsCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(provider.KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := providermetav1.ExpandMetadata(d.Get("metadata").([]interface{}))
	ep := api.Endpoints{
		ObjectMeta: metadata,
		Subsets:    expandEndpointsSubsets(d.Get("subset").(*schema.Set)),
	}
	log.Printf("[INFO] Creating new endpoints: %#v", ep)
	out, err := conn.CoreV1().Endpoints(metadata.Namespace).Create(ctx, &ep, metav1.CreateOptions{})
	if err != nil {
		return diag.Errorf("Failed to create endpoints because: %s", err)
	}
	log.Printf("[INFO] Submitted new endpoints: %#v", out)
	d.SetId(providermetav1.BuildId(out.ObjectMeta))

	return resourceKubernetesEndpointsRead(ctx, d, meta)
}

func resourceKubernetesEndpointsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesEndpointsExists(ctx, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		d.SetId("")
		return diag.Diagnostics{}
	}
	conn, err := meta.(provider.KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := providermetav1.IdParts(d.Id())
	if err != nil {
		return diag.Errorf("Failed to read endpoints because: %s", err)
	}

	log.Printf("[INFO] Reading endpoints %s", name)
	ep, err := conn.CoreV1().Endpoints(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.Errorf("Failed to read endpoint because: %s", err)
	}
	log.Printf("[INFO] Received endpoints: %#v", ep)
	err = d.Set("metadata", providermetav1.FlattenMetadata(ep.ObjectMeta, d, meta))
	if err != nil {
		return diag.Errorf("Failed to read endpoints because: %s", err)
	}

	flattened := flattenEndpointsSubsets(ep.Subsets)
	log.Printf("[DEBUG] Flattened endpoints subset: %#v", flattened)
	err = d.Set("subset", flattened)
	if err != nil {
		return diag.Errorf("Failed to read endpoints because: %s", err)
	}

	return nil
}

func resourceKubernetesEndpointsUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(provider.KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := providermetav1.IdParts(d.Id())
	if err != nil {
		return diag.Errorf("Failed to update endpoints because: %s", err)
	}

	ops := providermetav1.PatchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("subset") {
		subsets := expandEndpointsSubsets(d.Get("subset").(*schema.Set))
		ops = append(ops, &structures.ReplaceOperation{
			Path:  "/subsets",
			Value: subsets,
		})
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating endpoints %q: %v", name, string(data))
	out, err := conn.CoreV1().Endpoints(namespace).Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.Errorf("Failed to update endpoints: %s", err)
	}
	log.Printf("[INFO] Submitted updated endpoints: %#v", out)
	d.SetId(providermetav1.BuildId(out.ObjectMeta))

	return resourceKubernetesEndpointsRead(ctx, d, meta)
}

func resourceKubernetesEndpointsDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(provider.KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := providermetav1.IdParts(d.Id())
	if err != nil {
		return diag.Errorf("Failed to delete endpoints because: %s", err)
	}
	log.Printf("[INFO] Deleting endpoints: %#v", name)
	err = conn.CoreV1().Endpoints(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return diag.Errorf("Failed to delete endpoints because: %s", err)
	}
	log.Printf("[INFO] Endpoints %s deleted", name)
	d.SetId("")

	return nil
}

func resourceKubernetesEndpointsExists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(provider.KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	namespace, name, err := providermetav1.IdParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking endpoints %s", name)
	_, err = conn.CoreV1().Endpoints(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
