// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
)

func resourceKubernetesConfigMapV1Data() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesConfigMapV1DataCreate,
		ReadContext:   resourceKubernetesConfigMapV1DataRead,
		UpdateContext: resourceKubernetesConfigMapV1DataUpdate,
		DeleteContext: resourceKubernetesConfigMapV1DataDelete,
		Schema: map[string]*schema.Schema{
			"metadata": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Description: "The name of the ConfigMap.",
							Required:    true,
							ForceNew:    true,
						},
						"namespace": {
							Type:        schema.TypeString,
							Description: "The namespace of the ConfigMap.",
							Optional:    true,
							ForceNew:    true,
							Default:     "default",
						},
					},
				},
			},
			"data": {
				Type:        schema.TypeMap,
				Description: "The data we want to add to the ConfigMap.",
				Required:    true,
			},
			"force": {
				Type:        schema.TypeBool,
				Description: "Force overwriting data that is managed outside of Terraform.",
				Optional:    true,
			},
			"field_manager": {
				Type:        schema.TypeString,
				Description: "Set the name of the field manager for the specified labels.",
				Optional:    true,
				Default:     defaultFieldManagerName,
			},
		},
	}
}

func resourceKubernetesConfigMapV1DataCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	d.SetId(buildId(metadata))
	diag := resourceKubernetesConfigMapV1DataUpdate(ctx, d, m)
	if diag.HasError() {
		d.SetId("")
	}
	return diag
}

func resourceKubernetesConfigMapV1DataRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	conn, err := m.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// get the configmap data
	res, err := conn.CoreV1().ConfigMaps(namespace).Get(ctx, name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return diag.Diagnostics{{
				Severity: diag.Warning,
				Summary:  "ConfigMap deleted",
				Detail:   fmt.Sprintf("The underlying configmap %q has been deleted. You should recreate the underlying configmap, or remove it from your configuration.", name),
			}}
		}
		return diag.FromErr(err)
	}

	configuredData := d.Get("data").(map[string]interface{})

	// strip out the data not managed by Terraform
	fieldManagerName := d.Get("field_manager").(string)
	managedConfigMapData, err := getManagedConfigMapData(res.GetManagedFields(), fieldManagerName)
	if err != nil {
		return diag.FromErr(err)
	}
	data := res.Data
	for k := range data {
		_, managed := managedConfigMapData["f:"+k]
		_, configured := configuredData[k]
		if !managed && !configured {
			delete(data, k)
		}
	}

	d.Set("data", data)
	return nil
}

// getManagedConfigMapData reads the field manager metadata to discover which fields we're managing
func getManagedConfigMapData(managedFields []v1.ManagedFieldsEntry, manager string) (map[string]interface{}, error) {
	var data map[string]interface{}
	for _, m := range managedFields {
		if m.Manager != manager {
			continue
		}
		var mm map[string]interface{}
		err := json.Unmarshal(m.FieldsV1.Raw, &mm)
		if err != nil {
			return nil, err
		}
		if l, ok := mm["f:data"].(map[string]interface{}); ok {
			data = l
		}
	}
	return data, nil
}

func resourceKubernetesConfigMapV1DataUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	conn, err := m.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	name := metadata.GetName()
	namespace := metadata.GetNamespace()

	// check the resource exists before we try and patch it
	_, err = conn.CoreV1().ConfigMaps(namespace).Get(ctx, name, v1.GetOptions{})
	if err != nil {
		if d.Id() == "" {
			// if we are deleting then there is nothing to do
			// if the resource is gone
			return nil
		}
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return diag.Errorf("The ConfigMap %q does not exist", name)
		}

		return diag.Errorf("Have got the following error while validating the existence of the ConfigMap %q: %v", name, err)
	}

	// craft the patch to update the data
	data := d.Get("data")
	if d.Id() == "" {
		// if we're deleting then just we just patch
		// with an empty data map
		data = map[string]interface{}{}
	}
	patchobj := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
		},
		"data": data,
	}
	patch := unstructured.Unstructured{}
	patch.Object = patchobj
	patchbytes, err := patch.MarshalJSON()
	if err != nil {
		return diag.FromErr(err)
	}
	// apply the patch
	_, err = conn.CoreV1().ConfigMaps(namespace).Patch(ctx,
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
	return resourceKubernetesConfigMapV1DataRead(ctx, d, m)
}

func resourceKubernetesConfigMapV1DataDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	d.SetId("")
	return resourceKubernetesConfigMapV1DataUpdate(ctx, d, m)
}
