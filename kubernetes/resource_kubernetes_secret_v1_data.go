// Copyright IBM Corp. 2017, 2026
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

func resourceKubernetesSecretV1Data() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesSecretV1DataCreate,
		ReadContext:   resourceKubernetesSecretV1DataRead,
		UpdateContext: resourceKubernetesSecretV1DataUpdate,
		DeleteContext: resourceKubernetesSecretV1DataDelete,

		Schema: map[string]*schema.Schema{
			"metadata": {
				Type:        schema.TypeList,
				Description: "Metadata for the kubernetes Secret.",
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Description: "The name of the Secret.",
							Required:    true,
							ForceNew:    true,
						},
						"namespace": {
							Type:        schema.TypeString,
							Description: "The namespace of the Secret.",
							Optional:    true,
							ForceNew:    true,
							Default:     "default",
						},
					},
				},
			},
			"data": {
				Type:        schema.TypeMap,
				Description: "Data to be stored in the Kubernetes Secret.",
				Required:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"force": {
				Type:        schema.TypeBool,
				Description: "Flag to force updates to the Kubernetes Secret.",
				Optional:    true,
			},
			"field_manager": {
				Type:        schema.TypeString,
				Description: "Set the name of the field manager for the specified labels",
				Optional:    true,
				Default:     defaultFieldManagerName,
			},
		},
	}
}

func resourceKubernetesSecretV1DataCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	metadata := expandMetadata(d.Get("metadata").([]any))
	// Sets the resource id based on the metadata
	d.SetId(buildId(metadata))

	//Calling the update function ensuring resource config is correct
	diag := resourceKubernetesSecretV1DataUpdate(ctx, d, m)
	if diag.HasError() {
		d.SetId("")
	}
	return diag
}

// Retrieves the current state of the k8s secret, and update the current sate
func resourceKubernetesSecretV1DataRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	conn, err := m.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// getting the secret data
	res, err := conn.CoreV1().Secrets(namespace).Get(ctx, name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return diag.Diagnostics{{
				Severity: diag.Warning,
				Summary:  "Secret deleted",
				Detail:   fmt.Sprintf("The underlying secret %q has been deleted. You should recreate the underlying secret, or remove it from your configuration.", name),
			}}
		}
		return diag.FromErr(err)
	}

	configuredData := d.Get("data").(map[string]any)

	// stripping out the data not managed by Terraform
	fieldManagerName := d.Get("field_manager").(string)

	managedSecretData, err := getManagedSecretData(res.GetManagedFields(), fieldManagerName)
	if err != nil {
		return diag.FromErr(err)
	}
	data := res.Data
	for k := range data {
		_, managed := managedSecretData["f:"+k]
		_, configured := configuredData[k]
		if !managed && !configured {
			delete(data, k)
		}

	}
	decodedData := make(map[string]string, len(data))
	for k, v := range data {
		decodedData[k] = string(v)
	}

	d.Set("data", decodedData)

	return nil
}

// getManagedSecretData reads the field manager metadata to discover which fields we're managing
func getManagedSecretData(managedFields []v1.ManagedFieldsEntry, manager string) (map[string]interface{}, error) {
	var data map[string]any
	for _, m := range managedFields {
		// Only consider entries managed by the specified manager
		if m.Manager != manager {
			continue
		}
		var mm map[string]any
		err := json.Unmarshal(m.FieldsV1.Raw, &mm)
		if err != nil {
			return nil, err
		}
		// Check if the "data" field exists and extract it
		if l, ok := mm["f:data"].(map[string]any); ok {
			data = l
		}
	}
	return data, nil
}

func resourceKubernetesSecretV1DataUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	conn, err := m.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]any))
	name := metadata.GetName()
	namespace := metadata.GetNamespace()

	_, err = conn.CoreV1().Secrets(namespace).Get(ctx, name, v1.GetOptions{})
	if err != nil {
		if d.Id() == "" {
			// If we are deleting then there is nothing to do if the resource is gone
			return nil
		}
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return diag.Errorf("The Secret %q does not exist", name)
		}
		return diag.Errorf("Have got the following error while validating the existence of the Secret %q: %v", name, err)
	}

	// Craft the patch to update the data
	data := d.Get("data").(map[string]any)
	if d.Id() == "" {
		// If we're deleting then we just patch with an empty data map
		data = map[string]interface{}{}
	}

	encodedData := make(map[string][]byte, len(data))
	for k, v := range data {
		encodedData[k] = []byte(v.(string))
	}

	patchobj := map[string]any{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata": map[string]any{
			"name":      name,
			"namespace": namespace,
		},
		"data": encodedData,
	}
	patch := unstructured.Unstructured{}
	patch.Object = patchobj
	patchbytes, err := patch.MarshalJSON()
	if err != nil {
		return diag.FromErr(err)
	}

	// Apply the patch
	_, err = conn.CoreV1().Secrets(namespace).Patch(ctx,
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
				Detail:   fmt.Sprintf("Another client is managing a field Terraform tried to update. Set 'force' to true to override: %v", err),
			}}
		}
		return diag.FromErr(err)
	}

	if d.Id() == "" {
		return nil
	}

	return resourceKubernetesSecretV1DataRead(ctx, d, m)
}

func resourceKubernetesSecretV1DataDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// sets resource id to an empty. Simulating the deletion.
	d.SetId("")
	// Now we are calling the update function, to update the resource state
	return resourceKubernetesSecretV1DataUpdate(ctx, d, m)
}
