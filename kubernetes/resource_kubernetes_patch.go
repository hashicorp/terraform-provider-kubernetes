package kubernetes

import (
	"context"
	"fmt"
	"log"
	"reflect"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	k8smeta "k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
)

func resourceKubernetesPatch() *schema.Resource {
	metadataSchema := namespacedMetadataSchemaIsTemplate("patch", false, true)
	metadataSchema.ForceNew = true
	return &schema.Resource{
		CreateContext: resourceKubernetesPatchCreate,
		ReadContext:   resourceKubernetesPatchRead,
		DeleteContext: resourceKubernetesPatchDelete,
		CustomizeDiff: resourceKubernetesPatchDiff,
		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema,
			"kind": {
				Type:        schema.TypeString,
				Description: "The kind of resource we are going to patch.",
				Required:    true,
				ForceNew:    true,
			},
			"patch": {
				Type:        schema.TypeString,
				Description: "The patch for the resource in JSON format.",
				Required:    true,
				ForceNew:    true,
			},
			"patch_type": {
				Type:        schema.TypeString,
				Description: "The type of patch to apply.",
				Optional:    true,
				ForceNew:    true,
				Default:     "strategic",
				ValidateFunc: validation.StringInSlice([]string{
					"strategic",
					"merge",
					"json",
				}, true),
			},
			"patched_resource": {
				Type:        schema.TypeString,
				Description: "The result of the patch to the resource manifest.",
				Computed:    true,
			},
		},
	}
}

func resourceKubernetesPatchDiff(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
	conn, err := meta.(KubeClientsets).DynamicClient()
	if err != nil {
		return err
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	kind := d.Get("kind").(string)

	// figure out which version
	dc, err := meta.(KubeClientsets).DiscoveryClient()
	if err != nil {
		return err
	}
	agr, err := restmapper.GetAPIGroupResources(dc)
	gvk := k8sschema.GroupVersionKind{
		Kind: kind,
	}
	for i := range agr {
		for v, vr := range agr[i].VersionedResources {
			for ii := range vr {
				if vr[ii].Kind == kind {
					gvk.Group = agr[i].Group.Name
					gvk.Version = agr[i].Group.PreferredVersion.Version
					if gvk.Version == "" {
						gvk.Version = v
					}
					log.Println("found version", gvk.Version)
					break
				}
			}
		}
	}
	if gvk.Version == "" {
		return fmt.Errorf("could not find version for kind %q", kind)
	}

	restMapper := restmapper.NewDiscoveryRESTMapper(agr)
	mapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}

	var r dynamic.ResourceInterface
	if mapping.Scope.Name() == k8smeta.RESTScopeNameNamespace {
		ns := metadata.GetNamespace()
		if ns == "" {
			ns = "default"
		}
		r = conn.Resource(mapping.Resource).Namespace(ns)
	} else {
		r = conn.Resource(mapping.Resource)
	}

	log.Println("[DEBUG] getting patch target resource.")
	res, err := r.Get(ctx, metadata.GetName(), v1.GetOptions{})
	if err != nil {
		return err
	}

	resbytes, err := res.MarshalJSON()
	if err != nil {
		return err
	}

	log.Println("[DEBUG] running dry-run patch.")
	patch := []byte(d.Get("patch").(string))
	patchres, err := r.Patch(ctx, metadata.GetName(),
		getPatchType(d.Get("patch_type").(string)),
		patch, v1.PatchOptions{
			FieldManager: fieldManagerName,
			DryRun:       []string{"All"},
		})
	if err != nil {
		return err
	}

	patchedbytes, err := patchres.MarshalJSON()
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(
		removeServerSideFields(resbytes, true),
		removeServerSideFields(patchedbytes, true),
	) {
		log.Println("[DEBUG] resource needs to be patched.")
		d.SetNew("patched_resource", string(removeServerSideFields(patchedbytes, false)))
		d.ForceNew("patched_resource")
	}

	log.Println("[DEBUG] resource does not need to be patched.")
	return nil
}

func resourceKubernetesPatchCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).DynamicClient()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	kind := d.Get("kind").(string)

	// figure out which version
	dc, err := meta.(KubeClientsets).DiscoveryClient()
	if err != nil {
		return diag.FromErr(err)
	}
	agr, err := restmapper.GetAPIGroupResources(dc)
	gvk := k8sschema.GroupVersionKind{
		Kind: kind,
	}
	for i := range agr {
		for v, vr := range agr[i].VersionedResources {
			for ii := range vr {
				if vr[ii].Kind == kind {
					gvk.Group = agr[i].Group.Name
					gvk.Version = agr[i].Group.PreferredVersion.Version
					if gvk.Version == "" {
						gvk.Version = v
					}
					log.Println("found version", gvk.Version)
					break
				}
			}
		}
	}
	if gvk.Version == "" {
		return diag.Errorf("could not find version for kind %q", kind)
	}

	restMapper := restmapper.NewDiscoveryRESTMapper(agr)
	mapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return diag.FromErr(err)
	}

	var r dynamic.ResourceInterface
	if mapping.Scope.Name() == k8smeta.RESTScopeNameNamespace {
		ns := metadata.GetNamespace()
		if ns == "" {
			ns = "default"
		}
		r = conn.Resource(mapping.Resource).Namespace(ns)
	} else {
		r = conn.Resource(mapping.Resource)
	}

	patch := []byte(d.Get("patch").(string))

	res, err := r.Patch(ctx, metadata.GetName(),
		getPatchType(d.Get("patch_type").(string)),
		patch, v1.PatchOptions{
			FieldManager: fieldManagerName,
		})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(metadata.GetName())

	manifestBytes, err := res.MarshalJSON()
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("patched_resource", string(removeServerSideFields(manifestBytes, false)))

	return nil
}

func resourceKubernetesPatchRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).DynamicClient()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	kind := d.Get("kind").(string)

	// figure out which version
	dc, err := meta.(KubeClientsets).DiscoveryClient()
	if err != nil {
		return diag.FromErr(err)
	}
	agr, err := restmapper.GetAPIGroupResources(dc)
	gvk := k8sschema.GroupVersionKind{
		Kind: kind,
	}
	for i := range agr {
		for v, vr := range agr[i].VersionedResources {
			for ii := range vr {
				if vr[ii].Kind == kind {
					gvk.Group = agr[i].Group.Name
					gvk.Version = agr[i].Group.PreferredVersion.Version
					if gvk.Version == "" {
						gvk.Version = v
					}
					log.Println("found version", gvk.Version)
					break
				}
			}
		}
	}
	if gvk.Version == "" {
		return diag.Errorf("could not find version for kind %q", kind)
	}

	restMapper := restmapper.NewDiscoveryRESTMapper(agr)
	mapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return diag.FromErr(err)
	}

	var r dynamic.ResourceInterface
	if mapping.Scope.Name() == k8smeta.RESTScopeNameNamespace {
		ns := metadata.GetNamespace()
		if ns == "" {
			ns = "default"
		}
		r = conn.Resource(mapping.Resource).Namespace(ns)
	} else {
		r = conn.Resource(mapping.Resource)
	}

	res, err := r.Get(ctx, metadata.GetName(), v1.GetOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	resbytes, err := res.MarshalJSON()
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("patched_resource", string(removeServerSideFields(resbytes, false)))
	return nil
}

func resourceKubernetesPatchDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	d.SetId("")
	return nil
}

func getPatchType(t string) types.PatchType {
	switch t {
	case "strategic":
		return types.StrategicMergePatchType
	case "merge":
		return types.MergePatchType
	case "json":
		return types.JSONPatchType
	}
	return types.PatchType(t)
}
