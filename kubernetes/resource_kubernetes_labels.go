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

func resourceKubernetesLabels() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesLabelsCreate,
		ReadContext:   resourceKubernetesLabelsRead,
		UpdateContext: resourceKubernetesLabelsUpdate,
		DeleteContext: resourceKubernetesLabelsDelete,
		Schema: map[string]*schema.Schema{
			"api_version": {
				Type:        schema.TypeString,
				Description: "The apiVersion of the resource to label.",
				Required:    true,
				ForceNew:    true,
			},
			"kind": {
				Type:        schema.TypeString,
				Description: "The kind of the resource to label.",
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
			"labels": {
				Type:        schema.TypeMap,
				Description: "A map of labels to apply to the resource.",
				Required:    true,
			},
			"force": {
				Type:        schema.TypeBool,
				Description: "Force overwriting labels that were created or edited outside of Terraform.",
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

func resourceKubernetesLabelsCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	d.SetId(buildIdWithVersionKind(metadata,
		d.Get("api_version").(string),
		d.Get("kind").(string)))
	diag := resourceKubernetesLabelsUpdate(ctx, d, m)
	if diag.HasError() {
		d.SetId("")
	}
	return diag
}

func resourceKubernetesLabelsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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

	// get the resource labels
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

	configuredLabels := d.Get("labels").(map[string]interface{})

	// strip out the labels not managed by Terraform
	fieldManagerName := d.Get("field_manager").(string)
	managedLabels, err := getManagedLabels(res.GetManagedFields(), fieldManagerName)
	if err != nil {
		return diag.FromErr(err)
	}
	labels := res.GetLabels()
	for k := range labels {
		_, managed := managedLabels["f:"+k]
		_, configured := configuredLabels[k]
		if !managed && !configured {
			delete(labels, k)
		}
	}

	d.Set("labels", labels)
	return nil
}

// getManagedLabels reads the field manager metadata to discover which fields we're managing
func getManagedLabels(managedFields []v1.ManagedFieldsEntry, manager string) (map[string]interface{}, error) {
	var labels map[string]interface{}
	for _, m := range managedFields {
		if m.Manager != manager {
			continue
		}
		var mm map[string]interface{}
		err := json.Unmarshal(m.FieldsV1.Raw, &mm)
		if err != nil {
			return nil, err
		}
		if fm, ok := mm["f:metadata"].(map[string]interface{}); ok {
			if l, ok := fm["f:labels"].(map[string]interface{}); ok {
				labels = l
			}
		}

	}
	return labels, nil
}

func resourceKubernetesLabelsUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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

	// craft the patch to update the labels
	labels := d.Get("labels")
	if d.Id() == "" {
		// if we're deleting then just we just patch
		// with an empty labels map
		labels = map[string]interface{}{}
	}

	patchmeta := map[string]interface{}{
		"name":   name,
		"labels": labels,
	}
	if namespacedResource {
		patchmeta["namespace"] = namespace
	}
	patchobj := map[string]interface{}{
		"apiVersion": apiVersion,
		"kind":       kind,
		"metadata":   patchmeta,
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
	return resourceKubernetesLabelsRead(ctx, d, m)
}

func resourceKubernetesLabelsDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	d.SetId("")
	return resourceKubernetesLabelsUpdate(ctx, d, m)
}
