package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"k8s.io/apimachinery/pkg/api/errors"
	k8smeta "k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	"k8s.io/utils/pointer"
)

const fieldManagerName = "terraform"

func resourceKubernetesManifest() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesManifestApply,
		UpdateContext: resourceKubernetesManifestApply,
		ReadContext:   resourceKubernetesManifestRead,
		DeleteContext: resourceKubernetesManifestDelete,
		CustomizeDiff: resourceKubernetesManifestDiff,
		// Importer: &schema.ResourceImporter{
		// 	StateContext: schema.ImportStatePassthroughContext,
		// },
		Schema: map[string]*schema.Schema{
			"manifest": {
				Type:        schema.TypeString,
				Description: "The manifest for the resource in JSON format.",
				Required:    true,
				// TODO add validation to warn the user
				// not to use resources we already support
			},
			"resource": {
				Type:        schema.TypeString,
				Description: "The API response for the resource in JSON format.",
				Computed:    true,
			},
			"force_apply": {
				Type:        schema.TypeBool,
				Description: "Forcibly reclaim fields not managed by terraform.",
				Optional:    true,
				Default:     false,
			},
		},
	}
}

func resourceKubernetesManifestDiff(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
	conn, err := meta.(KubeClientsets).DynamicClient()
	if err != nil {
		return err
	}

	// parse `manifest into unstructured`
	manifestbytes := []byte(d.Get("manifest").(string))
	u := unstructured.Unstructured{}
	err = u.UnmarshalJSON(manifestbytes)
	if err != nil {
		return err
	}

	if u.GetKind() == "Pod" {
		return fmt.Errorf(`kind "Pod" not allowed: pods should not be managed by Terraform. Use a Job instead.`)
	}

	// figure out the resource client to use
	dc, err := meta.(KubeClientsets).DiscoveryClient()
	if err != nil {
		return err
	}
	agr, err := restmapper.GetAPIGroupResources(dc)
	restMapper := restmapper.NewDiscoveryRESTMapper(agr)
	gvk := u.GetObjectKind().GroupVersionKind()
	mapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		if k8smeta.IsNoMatchError(err) {
			// We couldn't find a mapping for this resource
			// set to computed and hope for the best
			d.SetNewComputed("resource")
			return nil
		}

		return err
	}

	var r dynamic.ResourceInterface
	if mapping.Scope.Name() == k8smeta.RESTScopeNameNamespace {
		ns := u.GetNamespace()
		if ns == "" {
			ns = "default"
		}
		r = conn.Resource(mapping.Resource).Namespace(ns)
	} else {
		r = conn.Resource(mapping.Resource)
	}

	// dry-run patch the resource to get the default values
	res, err := r.Patch(ctx,
		u.GetName(),
		types.ApplyPatchType,
		manifestbytes,
		v1.PatchOptions{
			FieldManager: fieldManagerName,
			DryRun:       []string{"All"},
			Force:        pointer.Bool(d.Get("force_apply").(bool)),
		},
	)
	if err != nil {
		return err
	}

	resbytes, err := res.MarshalJSON()
	if err != nil {
		return err
	}

	old := removeServerSideFields([]byte(d.Get("resource").(string)), true)
	new := removeServerSideFields(resbytes, true)
	if !reflect.DeepEqual(old, new) {
		d.SetNew("resource", string(removeServerSideFields(resbytes, false)))
	}

	return nil
}

func resourceKubernetesManifestApply(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).DynamicClient()
	if err != nil {
		return diag.FromErr(err)
	}

	// parse manifest into unstructured`
	manifestbytes := []byte(d.Get("manifest").(string))
	u := unstructured.Unstructured{}
	err = u.UnmarshalJSON(manifestbytes)
	if err != nil {
		return diag.FromErr(err)
	}

	// figure out the resource client to use
	dc, err := meta.(KubeClientsets).DiscoveryClient()
	if err != nil {
		return diag.FromErr(err)
	}
	agr, err := restmapper.GetAPIGroupResources(dc)
	restMapper := restmapper.NewDiscoveryRESTMapper(agr)
	gvk := u.GetObjectKind().GroupVersionKind()
	mapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return diag.FromErr(err)
	}

	var r dynamic.ResourceInterface
	if mapping.Scope.Name() == k8smeta.RESTScopeNameNamespace {
		ns := u.GetNamespace()
		if ns == "" {
			ns = "default"
		}
		r = conn.Resource(mapping.Resource).Namespace(ns)
	} else {
		r = conn.Resource(mapping.Resource)
	}

	// create the resource
	res, err := r.Patch(ctx,
		u.GetName(),
		types.ApplyPatchType,
		manifestbytes,
		v1.PatchOptions{
			FieldManager: fieldManagerName,
			Force:        pointer.Bool(d.Get("force_apply").(bool)),
		},
	)
	if err != nil {
		// FIXME add the kind/resource to the error so it's easier to find
		return diag.FromErr(err)
	}

	// resource specific logic
	switch gvk.Kind {
	// wait for CRD to be accepted
	case "CustomResourceDefinition":
		resource.RetryContext(ctx, d.Timeout(schema.TimeoutCreate),
			retryUntilCustomResourceDefinitionAccepted(ctx,
				r,
				u.GetName()))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(fmt.Sprintf("%s %s/%s", res.GroupVersionKind(), res.GetNamespace(), res.GetName()))
	return resourceKubernetesManifestRead(ctx, d, meta)
}

func resourceKubernetesManifestRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).DynamicClient()
	if err != nil {
		return diag.FromErr(err)
	}

	// parse `manifest into unstructured`
	manifestbytes := []byte(d.Get("manifest").(string))
	u := unstructured.Unstructured{}
	err = u.UnmarshalJSON(manifestbytes)
	if err != nil {
		return diag.FromErr(err)
	}

	// figure out the resource client to use
	dc, err := meta.(KubeClientsets).DiscoveryClient()
	if err != nil {
		return diag.FromErr(err)
	}
	agr, err := restmapper.GetAPIGroupResources(dc)
	restMapper := restmapper.NewDiscoveryRESTMapper(agr)
	gvk := u.GetObjectKind().GroupVersionKind()
	mapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return diag.FromErr(err)
	}

	var r dynamic.ResourceInterface
	if mapping.Scope.Name() == k8smeta.RESTScopeNameNamespace {
		ns := u.GetNamespace()
		if ns == "" {
			ns = "default"
		}
		r = conn.Resource(mapping.Resource).Namespace(ns)
	} else {
		r = conn.Resource(mapping.Resource)
	}

	// get the resource
	res, err := r.Get(ctx, u.GetName(), v1.GetOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	// save to state
	resbytes, err := res.MarshalJSON()
	if err != nil {
		return diag.FromErr(err)
	}

	resbytes = removeServerSideFields(resbytes, false)
	d.Set("resource", string(resbytes))
	return nil
}

func resourceKubernetesManifestDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).DynamicClient()
	if err != nil {
		return diag.FromErr(err)
	}

	// parse `manifest into unstructured`
	manifestbytes := []byte(d.Get("manifest").(string))
	u := unstructured.Unstructured{}
	err = u.UnmarshalJSON(manifestbytes)
	if err != nil {
		return diag.FromErr(err)
	}

	// figure out the resource client to use
	dc, err := meta.(KubeClientsets).DiscoveryClient()
	if err != nil {
		return diag.FromErr(err)
	}
	agr, err := restmapper.GetAPIGroupResources(dc)
	restMapper := restmapper.NewDiscoveryRESTMapper(agr)
	gvk := u.GetObjectKind().GroupVersionKind()
	mapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return diag.FromErr(err)
	}

	var r dynamic.ResourceInterface
	if mapping.Scope.Name() == k8smeta.RESTScopeNameNamespace {
		ns := u.GetNamespace()
		if ns == "" {
			ns = "default"
		}
		r = conn.Resource(mapping.Resource).Namespace(ns)
	} else {
		r = conn.Resource(mapping.Resource)
	}

	// delete the resource
	err = r.Delete(ctx, u.GetName(), v1.DeleteOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	// wait for deletion
	resource.RetryContext(ctx, d.Timeout(schema.TimeoutCreate),
		retryUntilResourceDeleted(ctx,
			r,
			u.GetName()))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}

func removeServerSideFields(b []byte, removeStatus bool) []byte {
	var m interface{}
	json.Unmarshal(b, &m) // FIXME

	mm, ok := m.(map[string]interface{})
	if !ok {
		return b
	}

	if removeStatus {
		delete(mm, "status")
	}

	if meta, ok := mm["metadata"].(map[string]interface{}); ok {
		delete(meta, "uid")
		delete(meta, "creationTimestamp")
		delete(meta, "resourceVersion")
		delete(meta, "generation")
		delete(meta, "managedFields")
	}

	removed, _ := json.Marshal(mm) // FIXME
	return removed
}

// retryUntilCustomResourceDefinitionAccepted retries until the CustomResourceDefinition has an accepted status
func retryUntilCustomResourceDefinitionAccepted(ctx context.Context, r dynamic.ResourceInterface, name string) resource.RetryFunc {
	return func() *resource.RetryError {
		res, err := r.Get(ctx, name, v1.GetOptions{})
		if err != nil {
			return resource.NonRetryableError(err)
		}

		// FIXME puke, invert this logic
		if status, ok := res.Object["status"]; ok {
			m := status.(map[string]interface{})
			if c, ok := m["conditions"]; ok {
				if cl, ok := c.([]interface{}); ok {
					for _, v := range cl {
						if vm, ok := v.(map[string]interface{}); ok {
							if t, ok := vm["type"].(string); ok {
								if t == "NamesAccepted" {
									if s, ok := vm["status"].(string); ok {
										if s == "True" {
											return nil
										}
									}
								}
							}
						}
					}
				}
			}
		}

		return resource.RetryableError(fmt.Errorf("customresourcedefinition %q has not been accepted yet", name))
	}
}

// retryUntilResourceDelete retries until the resource has been deleted
func retryUntilResourceDeleted(ctx context.Context, r dynamic.ResourceInterface, name string) resource.RetryFunc {
	return func() *resource.RetryError {
		_, err := r.Get(ctx, name, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return resource.NonRetryableError(err)
		}

		return resource.RetryableError(fmt.Errorf("resource %q is being deleted", name))
	}
}
