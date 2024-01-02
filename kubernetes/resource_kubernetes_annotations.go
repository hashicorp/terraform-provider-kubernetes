// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-kubernetes/util"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	"k8s.io/utils/ptr"
)

func resourceKubernetesAnnotations() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesAnnotationsCreate,
		ReadContext:   resourceKubernetesAnnotationsRead,
		UpdateContext: resourceKubernetesAnnotationsUpdate,
		DeleteContext: resourceKubernetesAnnotationsDelete,
		Schema: map[string]*schema.Schema{
			"api_version": {
				Type:        schema.TypeString,
				Description: "The apiVersion of the resource to annotate.",
				Required:    true,
				ForceNew:    true,
			},
			"kind": {
				Type:        schema.TypeString,
				Description: "The kind of the resource to annotate.",
				Required:    true,
				ForceNew:    true,
			},
			"metadata": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Description: "The name of the resource.",
							Required:    true,
							ForceNew:    true,
						},
						"namespace": {
							Type:        schema.TypeString,
							Description: "The namespace of the resource.",
							Optional:    true,
							ForceNew:    true,
						},
					},
				},
			},
			"annotations": {
				Type:         schema.TypeMap,
				Description:  "A map of annotations to apply to the resource.",
				Optional:     true,
				AtLeastOneOf: []string{"template_annotations", "annotations"},
			},
			"template_annotations": {
				Type:         schema.TypeMap,
				Description:  "A map of annotations to apply to the resource template.",
				Optional:     true,
				AtLeastOneOf: []string{"template_annotations", "annotations"},
			},
			"force": {
				Type:        schema.TypeBool,
				Description: "Force overwriting annotations that were created or edited outside of Terraform.",
				Optional:    true,
			},
			"field_manager": {
				Type:         schema.TypeString,
				Description:  "Set the name of the field manager for the specified labels.",
				Optional:     true,
				Default:      defaultFieldManagerName,
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
		},
	}
}

func resourceKubernetesAnnotationsCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	d.SetId(buildIdWithVersionKind(metadata,
		d.Get("api_version").(string),
		d.Get("kind").(string)))
	diag := resourceKubernetesAnnotationsUpdate(ctx, d, m)
	if diag.HasError() {
		d.SetId("")
	}
	return diag
}

func resourceKubernetesAnnotationsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	conn, err := m.(KubeClientsets).DynamicClient()
	if err != nil {
		return diag.FromErr(err)
	}

	gvk, name, namespace, err := util.ParseResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// figure out which resource client to use
	dc, err := m.(KubeClientsets).DiscoveryClient()
	if err != nil {
		return diag.FromErr(err)
	}
	agr, err := restmapper.GetAPIGroupResources(dc)
	if err != nil {
		return diag.FromErr(err)
	}
	restMapper := restmapper.NewDiscoveryRESTMapper(agr)
	mapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return diag.FromErr(err)
	}

	// determine if the resource is namespaced or not
	var r dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		if namespace == "" {
			namespace = "default"
		}
		r = conn.Resource(mapping.Resource).Namespace(namespace)
	} else {
		r = conn.Resource(mapping.Resource)
	}

	// get the resource annotations
	res, err := r.Get(ctx, name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return diag.Diagnostics{{
				Severity: diag.Warning,
				Summary:  "Resource deleted",
				Detail:   fmt.Sprintf("The underlying resource %q has been deleted. You should recreate the underlying resource, or remove it from your configuration.", name),
			}}
		}
		return diag.FromErr(err)
	}

	fieldManagerName := d.Get("field_manager").(string)

	// strip out the annotations not managed by Terraform
	configuredAnnotations := d.Get("annotations").(map[string]interface{})
	managedAnnotations, err := getManagedAnnotations(res.GetManagedFields(), fieldManagerName)
	if err != nil {
		return diag.FromErr(err)
	}
	annotations := res.GetAnnotations()
	for k := range annotations {
		_, managed := managedAnnotations["f:"+k]
		_, configured := configuredAnnotations[k]
		if !managed && !configured {
			delete(annotations, k)
		}
	}
	d.Set("annotations", annotations)

	kind := d.Get("kind").(string)
	configuredTemplateAnnotations := d.Get("template_annotations").(map[string]interface{})
	managedTemplateAnnotations, err := getTemplateManagedAnnotations(res.GetManagedFields(), fieldManagerName, kind)
	if err != nil {
		return diag.FromErr(err)
	}
	var templateAnnotations map[string]string
	if kind == "CronJob" {
		templateAnnotations, _, err = unstructured.NestedStringMap(res.Object, "spec", "jobTemplate", "spec", "template", "metadata", "annotations")
	} else {
		templateAnnotations, _, err = unstructured.NestedStringMap(res.Object, "spec", "template", "metadata", "annotations")
	}
	if err != nil {
		return diag.FromErr(err)
	}
	for k := range templateAnnotations {
		_, managed := managedTemplateAnnotations["f:"+k]
		_, configured := configuredTemplateAnnotations[k]
		if !managed && !configured {
			delete(templateAnnotations, k)
		}
	}
	d.Set("template_annotations", templateAnnotations)

	return nil
}

// getManagedAnnotations reads the field manager metadata to discover which fields we're managing
func getManagedAnnotations(managedFields []v1.ManagedFieldsEntry, manager string) (map[string]interface{}, error) {
	var annotations map[string]interface{}
	for _, m := range managedFields {
		if m.Manager != manager {
			continue
		}
		var mm map[string]interface{}
		err := json.Unmarshal(m.FieldsV1.Raw, &mm)
		if err != nil {
			return nil, err
		}
		var metadata map[string]interface{}
		if mmm, ok := mm["f:metadata"].(map[string]interface{}); ok {
			metadata = mmm
		}
		if l, ok := metadata["f:annotations"].(map[string]interface{}); ok {
			annotations = l
		}
	}
	return annotations, nil
}

// getTemplateManagedAnnotations reads the field manager metadata to discover which fields we're managing
func getTemplateManagedAnnotations(managedFields []v1.ManagedFieldsEntry, manager string, kind string) (map[string]interface{}, error) {
	var annotations map[string]interface{}
	for _, m := range managedFields {
		if m.Manager != manager {
			continue
		}
		var mm map[string]interface{}
		err := json.Unmarshal(m.FieldsV1.Raw, &mm)
		if err != nil {
			return nil, err
		}
		var spec map[string]interface{}
		if s, ok := mm["f:spec"].(map[string]interface{}); ok {
			spec = s
		}
		if kind == "CronJob" {
			if jt, ok := spec["f:jobTemplate"].(map[string]interface{}); ok {
				spec = jt["f:spec"].(map[string]interface{})
			}
		}
		var template map[string]interface{}
		if t, ok := spec["f:template"].(map[string]interface{}); ok {
			template = t
		}
		var metadata map[string]interface{}
		if mmm, ok := template["f:metadata"].(map[string]interface{}); ok {
			metadata = mmm
		}
		if l, ok := metadata["f:annotations"].(map[string]interface{}); ok {
			annotations = l
		}
	}
	return annotations, nil
}

func resourceKubernetesAnnotationsUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	conn, err := m.(KubeClientsets).DynamicClient()
	if err != nil {
		return diag.FromErr(err)
	}

	apiVersion := d.Get("api_version").(string)
	kind := d.Get("kind").(string)
	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	name := metadata.GetName()
	namespace := metadata.GetNamespace()

	// figure out which resource client to use
	dc, err := m.(KubeClientsets).DiscoveryClient()
	if err != nil {
		return diag.FromErr(err)
	}
	agr, err := restmapper.GetAPIGroupResources(dc)
	if err != nil {
		return diag.FromErr(err)
	}
	restMapper := restmapper.NewDiscoveryRESTMapper(agr)
	gv, err := k8sschema.ParseGroupVersion(apiVersion)
	if err != nil {
		return diag.FromErr(err)

	}
	mapping, err := restMapper.RESTMapping(gv.WithKind(kind).GroupKind(), gv.Version)
	if err != nil {
		return diag.FromErr(err)
	}

	// determine if the resource is namespaced or not
	var r dynamic.ResourceInterface
	namespacedResource := mapping.Scope.Name() == meta.RESTScopeNameNamespace
	if namespacedResource {
		if namespace == "" {
			namespace = "default"
		}
		r = conn.Resource(mapping.Resource).Namespace(namespace)
	} else {
		r = conn.Resource(mapping.Resource)
	}

	// check the resource exists before we try and patch it
	_, err = r.Get(ctx, name, v1.GetOptions{})
	if err != nil {
		if d.Id() == "" {
			// if we are deleting then there is nothing to do
			// if the resource is gone
			return nil
		}
		return diag.Errorf("The resource %q does not exist", name)
	}

	// craft the patch to update the annotations
	annotations := d.Get("annotations")
	templateAnnotations := d.Get("template_annotations")
	if d.Id() == "" {
		// if we're deleting then just we just patch
		// with an empty annotations map
		annotations = map[string]interface{}{}
		templateAnnotations = map[string]interface{}{}
	}
	patchmeta := map[string]interface{}{
		"name": name,
	}
	if namespacedResource {
		patchmeta["namespace"] = namespace
	}
	if _, ok := d.GetOk("annotations"); ok {
		patchmeta["annotations"] = annotations
	}
	patchobj := map[string]interface{}{
		"apiVersion": apiVersion,
		"kind":       kind,
		"metadata":   patchmeta,
	}
	if _, ok := d.GetOk("template_annotations"); ok {
		spec := map[string]interface{}{
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"annotations": templateAnnotations,
				},
			},
		}
		if kind == "CronJob" {
			patchobj["spec"] = map[string]interface{}{
				"jobTemplate": map[string]interface{}{
					"spec": spec,
				},
			}
		} else {
			patchobj["spec"] = spec
		}
	}

	patch := unstructured.Unstructured{}
	patch.Object = patchobj
	patchbytes, err := patch.MarshalJSON()
	if err != nil {
		return diag.FromErr(err)
	}
	// apply the patch
	_, err = r.Patch(ctx,
		name,
		types.ApplyPatchType,
		patchbytes,
		v1.PatchOptions{
			FieldManager: d.Get("field_manager").(string),
			Force:        ptr.To(d.Get("force").(bool)),
		},
	)
	if err != nil {
		if errors.IsConflict(err) {
			return diag.Diagnostics{{
				Severity: diag.Error,
				Summary:  "Field manager conflict",
				Detail:   fmt.Sprintf(`Another client is managing a field Terraform tried to update. Set "force" to true to override: %v`, err),
			}}
		}
		return diag.FromErr(err)
	}

	if d.Id() == "" {
		// don't try to read if we're deleting
		return nil
	}
	return resourceKubernetesAnnotationsRead(ctx, d, m)
}

func resourceKubernetesAnnotationsDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	d.SetId("")
	return resourceKubernetesAnnotationsUpdate(ctx, d, m)
}
