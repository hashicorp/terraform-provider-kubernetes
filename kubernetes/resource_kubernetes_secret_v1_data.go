package kubernetes

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			// meta attr, which contains info about the secret. It is required and can have a maxvalue of 1
			"metadata": {
				Type:        schema.TypeList,
				Description: "Metadata for the kubernetes Secret.",
				Required:    true,
				MaxItems:    1,
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
			// map data attr, contains data to be store in secret. Elem, specifies the schema for each value in the map
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
				Default:     false,
			},
			"field_manager": {
				Type:         schema.TypeString,
				Description:  "Set the name of the field manager for the specified labels",
				Optional:     true,
				Default:      defaultFieldManagerName,
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
		},
	}
}

func resourceKubernetesSecretV1DataCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	// Sets the resource id based on the metadata
	d.SetId(buildId(metadata))

	//Calling the update function ensuring resource config is correct
	diag := resourceKubernetesSecretV1DataUpdate(ctx, d, m)
	if diag.HasError() {
		d.SetId("")
	}
	return nil
}

// Retrieves the current state of the k8s secret, and update the current sate
func resourceKubernetesSecretV1DataRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	conn, err := m.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	//extracting ns and name from res id
	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	// Retrieve the K8s secret
	secret, err := conn.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Handle case where the Secret is not found
			return diag.Diagnostics{{
				Severity: diag.Warning,
				Summary:  "Secret deleted",
				Detail:   fmt.Sprintf("The underlying secret %q has been deleted. You should recreate the underlying secret, or remove it from your configuration.", name),
			}}
		}
		return diag.FromErr(err)
	}

	// Extract managed data from the secret
	managedData, err := getManagedSecretData(secret)
	if err != nil {
		return diag.FromErr(err)
	}

	// filter out the data not managed by terraform
	configuredData := d.Get("data").(map[string]interface{})
	for k := range managedData {
		if _, exists := configuredData[k]; !exists {
			delete(managedData, k)
		}
	}
	// Update the state with the managed data
	d.Set("data", managedData)
	return nil
}

// extracts data from the secret that is managed by terraform
func getManagedSecretData(secret *v1.Secret) (map[string]interface{}, error) {
	managedData := make(map[string]interface{})

	//looping through all data in the secret
	for key, value := range secret.Data {
		// decode base64-encoded value
		decodedValue, err := base64.StdEncoding.DecodeString(string(value))
		if err != nil {
			return nil, fmt.Errorf("failed to decode value for key %q: %w", key, err)
		}

		// just storing the decoded value I got in the managed data map
		managedData[key] = string(decodedValue)
	}
	return managedData, nil

}

func resourceKubernetesSecretV1DataUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	conn, err := m.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	name := metadata.GetName()
	namespace := metadata.GetNamespace()

	_, err = conn.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
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
	data := d.Get("data")
	if d.Id() == "" {
		// If we're deleting then we just patch with an empty data map
		data = map[string]interface{}{}
	}
	patchobj := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Secret",
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
	// Apply the patch
	_, err = conn.CoreV1().Secrets(namespace).Patch(ctx,
		name,
		types.ApplyPatchType,
		patchbytes,
		metav1.PatchOptions{
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
