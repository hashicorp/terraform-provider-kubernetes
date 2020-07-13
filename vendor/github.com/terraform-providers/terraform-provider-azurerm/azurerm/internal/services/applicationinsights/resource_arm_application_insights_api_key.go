package applicationinsights

import (
	"fmt"
	"log"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/appinsights/mgmt/2015-05-01/insights"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/tf"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/features"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func resourceArmApplicationInsightsAPIKey() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmApplicationInsightsAPIKeyCreate,
		Read:   resourceArmApplicationInsightsAPIKeyRead,
		Delete: resourceArmApplicationInsightsAPIKeyDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Read:   schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.NoZeroValues,
			},

			"application_insights_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: azure.ValidateResourceID,
			},

			"read_permissions": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Set:      schema.HashString,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{"agentconfig", "aggregate", "api", "draft", "extendqueries", "search"}, false),
				},
			},

			"write_permissions": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Set:      schema.HashString,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{"annotations"}, false),
				},
			},

			"api_key": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
		},
	}
}

func resourceArmApplicationInsightsAPIKeyCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).AppInsights.APIKeysClient
	ctx, cancel := timeouts.ForCreate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	log.Printf("[INFO] preparing arguments for AzureRM Application Insights API key creation.")

	name := d.Get("name").(string)
	appInsightsID := d.Get("application_insights_id").(string)

	id, err := azure.ParseAzureResourceID(appInsightsID)
	if err != nil {
		return err
	}

	resGroup := id.ResourceGroup
	appInsightsName := id.Path["components"]

	if features.ShouldResourcesBeImported() {
		var existing insights.ApplicationInsightsComponentAPIKey
		existing, err = client.Get(ctx, resGroup, appInsightsName, name)
		if err != nil {
			if !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("Error checking for presence of existing Application Insights API key %q (Resource Group %q): %s", name, resGroup, err)
			}
		}

		if existing.ID != nil && *existing.ID != "" {
			return tf.ImportAsExistsError("azurerm_application_insights_api_key", *existing.ID)
		}
	}

	apiKeyProperties := insights.APIKeyRequest{
		Name:                  &name,
		LinkedReadProperties:  azure.ExpandApplicationInsightsAPIKeyLinkedProperties(d.Get("read_permissions").(*schema.Set), appInsightsID),
		LinkedWriteProperties: azure.ExpandApplicationInsightsAPIKeyLinkedProperties(d.Get("write_permissions").(*schema.Set), appInsightsID),
	}

	result, err := client.Create(ctx, resGroup, appInsightsName, apiKeyProperties)
	if err != nil {
		return fmt.Errorf("Error creating Application Insights API key %q (Resource Group %q): %+v", name, resGroup, err)
	}

	if result.APIKey == nil {
		return fmt.Errorf("Error creating Application Insights API key %q (Resource Group %q): got empty API key", name, resGroup)
	}

	d.SetId(*result.ID)

	// API key can only retrieved at key creation
	d.Set("api_key", result.APIKey)

	return resourceArmApplicationInsightsAPIKeyRead(d, meta)
}

func resourceArmApplicationInsightsAPIKeyRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).AppInsights.APIKeysClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Reading AzureRM Application Insights API key '%s'", id)

	resGroup := id.ResourceGroup
	appInsightsName := id.Path["components"]
	keyID := id.Path["apikeys"]

	result, err := client.Get(ctx, resGroup, appInsightsName, keyID)
	if err != nil {
		if utils.ResponseWasNotFound(result.Response) {
			log.Printf("[WARN] AzureRM Application Insights API key '%s' not found, removing from state", id)
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error making Read request on AzureRM Application Insights API key '%s': %+v", keyID, err)
	}

	d.Set("application_insights_id", fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/microsoft.insights/components/%s", client.SubscriptionID, resGroup, appInsightsName))

	d.Set("name", result.Name)
	readProps := azure.FlattenApplicationInsightsAPIKeyLinkedProperties(result.LinkedReadProperties)
	if err := d.Set("read_permissions", readProps); err != nil {
		return fmt.Errorf("Error flattening `read_permissions `: %s", err)
	}
	writeProps := azure.FlattenApplicationInsightsAPIKeyLinkedProperties(result.LinkedWriteProperties)
	if err := d.Set("write_permissions", writeProps); err != nil {
		return fmt.Errorf("Error flattening `write_permissions `: %s", err)
	}

	return nil
}

func resourceArmApplicationInsightsAPIKeyDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).AppInsights.APIKeysClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	appInsightsName := id.Path["components"]
	keyID := id.Path["apikeys"]

	log.Printf("[DEBUG] Deleting AzureRM Application Insights API key '%s' (resource group '%s')", keyID, resGroup)

	result, err := client.Delete(ctx, resGroup, appInsightsName, keyID)
	if err != nil {
		if utils.ResponseWasNotFound(result.Response) {
			return nil
		}
		return fmt.Errorf("Error issuing AzureRM delete request for Application Insights API key '%s': %+v", keyID, err)
	}

	return nil
}
