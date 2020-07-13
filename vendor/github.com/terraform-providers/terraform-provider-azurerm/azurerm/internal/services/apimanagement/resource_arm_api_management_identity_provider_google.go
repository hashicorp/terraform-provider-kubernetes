package apimanagement

import (
	"fmt"
	"log"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/apimanagement/mgmt/2018-01-01/apimanagement"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/tf"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/validate"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/features"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func resourceArmApiManagementIdentityProviderGoogle() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmApiManagementIdentityProviderGoogleCreateUpdate,
		Read:   resourceArmApiManagementIdentityProviderGoogleRead,
		Update: resourceArmApiManagementIdentityProviderGoogleCreateUpdate,
		Delete: resourceArmApiManagementIdentityProviderGoogleDelete,
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
			"resource_group_name": azure.SchemaResourceGroupName(),

			"api_management_name": azure.SchemaApiManagementName(),

			"client_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validate.GoogleClientID,
			},

			"client_secret": {
				Type:         schema.TypeString,
				Required:     true,
				Sensitive:    true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
		},
	}
}

func resourceArmApiManagementIdentityProviderGoogleCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).ApiManagement.IdentityProviderClient
	ctx, cancel := timeouts.ForCreateUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	resourceGroup := d.Get("resource_group_name").(string)
	serviceName := d.Get("api_management_name").(string)
	clientID := d.Get("client_id").(string)
	clientSecret := d.Get("client_secret").(string)

	if features.ShouldResourcesBeImported() && d.IsNewResource() {
		existing, err := client.Get(ctx, resourceGroup, serviceName, apimanagement.Google)
		if err != nil {
			if !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("Error checking for presence of existing Identity Provider %q (API Management Service %q / Resource Group %q): %s", apimanagement.Google, serviceName, resourceGroup, err)
			}
		}

		if existing.ID != nil && *existing.ID != "" {
			return tf.ImportAsExistsError("azurerm_api_management_identity_provider_google", *existing.ID)
		}
	}

	parameters := apimanagement.IdentityProviderContract{
		IdentityProviderContractProperties: &apimanagement.IdentityProviderContractProperties{
			ClientID:     utils.String(clientID),
			ClientSecret: utils.String(clientSecret),
			Type:         apimanagement.Google,
		},
	}

	if _, err := client.CreateOrUpdate(ctx, resourceGroup, serviceName, apimanagement.Google, parameters, ""); err != nil {
		return fmt.Errorf("Error creating or updating Identity Provider %q (Resource Group %q / API Management Service %q): %+v", apimanagement.Google, resourceGroup, serviceName, err)
	}

	resp, err := client.Get(ctx, resourceGroup, serviceName, apimanagement.Google)
	if err != nil {
		return fmt.Errorf("Error retrieving Identity Provider %q (Resource Group %q / API Management Service %q): %+v", apimanagement.Google, resourceGroup, serviceName, err)
	}
	if resp.ID == nil {
		return fmt.Errorf("Cannot read ID for Identity Provider %q (Resource Group %q / API Management Service %q)", apimanagement.Google, resourceGroup, serviceName)
	}
	d.SetId(*resp.ID)

	return resourceArmApiManagementIdentityProviderGoogleRead(d, meta)
}

func resourceArmApiManagementIdentityProviderGoogleRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).ApiManagement.IdentityProviderClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resourceGroup := id.ResourceGroup
	serviceName := id.Path["service"]
	identityProviderName := id.Path["identityProviders"]

	resp, err := client.Get(ctx, resourceGroup, serviceName, apimanagement.IdentityProviderType(identityProviderName))
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			log.Printf("[DEBUG] Identity Provider %q (Resource Group %q / API Management Service %q) was not found - removing from state!", identityProviderName, resourceGroup, serviceName)
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error making Read request for Identity Provider %q (Resource Group %q / API Management Service %q): %+v", identityProviderName, resourceGroup, serviceName, err)
	}

	d.Set("resource_group_name", resourceGroup)
	d.Set("api_management_name", serviceName)

	if props := resp.IdentityProviderContractProperties; props != nil {
		d.Set("client_id", props.ClientID)
		d.Set("client_secret", props.ClientSecret)
	}

	return nil
}

func resourceArmApiManagementIdentityProviderGoogleDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).ApiManagement.IdentityProviderClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resourceGroup := id.ResourceGroup
	serviceName := id.Path["service"]
	identityProviderName := id.Path["identityProviders"]

	if resp, err := client.Delete(ctx, resourceGroup, serviceName, apimanagement.IdentityProviderType(identityProviderName), ""); err != nil {
		if !utils.ResponseWasNotFound(resp) {
			return fmt.Errorf("Error deleting Identity Provider %q (Resource Group %q / API Management Service %q): %+v", identityProviderName, resourceGroup, serviceName, err)
		}
	}

	return nil
}
