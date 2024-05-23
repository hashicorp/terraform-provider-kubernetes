package kubernetes

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func resourceKubernetesSecretV1Data() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesSecretV1DataCreate,
		Read:   resourceKubernetesSecretV1DataRead,
		Update: resourceKubernetesSecretV1DataUpdate,
		Delete: resourceKubernetesSecretV1DataDelete,

		Schema: map[string]*schema.Schema{
			// meta attr, which contains info about the secret. It is required and can have a maxvalue of 1
			"metadata": {
				Type:        schema.TypeList,
				Description: "Metadata for the kubernetes Secret.",
				Required:    true,
				MaxItems:    1,
				Elem:        resourceMetaData(),
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
		},
	}
}

func resourceKubernetesSecretV1DataCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	metadata := expandMetaData(d.Get("metadata").([]interface{}))
	// Sets the resource id based on the metadata
	d.SetId(buildId(metadata))

	//Calling the update function ensuring resource config is correct
	diag := resourceKubernetesSecretV1DataUpdate(ctx, d, m)
	if diag.HasError() {
		d.SetId("")
	}
	return diag
}

// Retrieves rthe current state of the k8s secret, and update the current sate
func resourceKubernetesSecretV1DataRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	clientset, err := getClientset(m)
	if err != nil {
		return diag.FromErr(err)
	}

	//extracting ns and name from res id
	namespace, name, err := parseResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	// Retrieve the K8s secret
	secret, err := clientset.Corev1.Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
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
