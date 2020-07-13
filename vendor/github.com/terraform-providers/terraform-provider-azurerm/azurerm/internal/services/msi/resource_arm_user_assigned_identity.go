package msi

import (
	"fmt"
	"log"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/preview/msi/mgmt/2015-08-31-preview/msi"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/tf"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/features"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tags"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func resourceArmUserAssignedIdentity() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmUserAssignedIdentityCreateUpdate,
		Read:   resourceArmUserAssignedIdentityRead,
		Update: resourceArmUserAssignedIdentityCreateUpdate,
		Delete: resourceArmUserAssignedIdentityDelete,
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
				ValidateFunc: validation.StringLenBetween(3, 128),
			},

			"resource_group_name": azure.SchemaResourceGroupName(),

			"location": azure.SchemaLocation(),

			"tags": tags.Schema(),

			"principal_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"client_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceArmUserAssignedIdentityCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).MSI.UserAssignedIdentitiesClient
	ctx, cancel := timeouts.ForCreateUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	log.Printf("[INFO] preparing arguments for Azure ARM user identity creation.")

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)
	t := d.Get("tags").(map[string]interface{})

	if features.ShouldResourcesBeImported() && d.IsNewResource() {
		existing, err := client.Get(ctx, resGroup, name)
		if err != nil {
			if !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("Error checking for presence of existing User Assigned Identity %q (Resource Group %q): %+v", name, resGroup, err)
			}
		}

		if existing.ID != nil && *existing.ID != "" {
			return tf.ImportAsExistsError("azurerm_user_assigned_identity", *existing.ID)
		}
	}

	identity := msi.Identity{
		Name:     &name,
		Location: &location,
		Tags:     tags.Expand(t),
	}

	if _, err := client.CreateOrUpdate(ctx, resGroup, name, identity); err != nil {
		return fmt.Errorf("Error Creating/Updating User Assigned Identity %q (Resource Group %q): %+v", name, resGroup, err)
	}

	read, err := client.Get(ctx, resGroup, name)
	if err != nil {
		return err
	}

	if read.ID == nil {
		return fmt.Errorf("Cannot read User Assigned Identity %q ID (resource group %q) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmUserAssignedIdentityRead(d, meta)
}

func resourceArmUserAssignedIdentityRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).MSI.UserAssignedIdentitiesClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}

	resGroup := id.ResourceGroup
	name := id.Path["userAssignedIdentities"]
	resp, err := client.Get(ctx, resGroup, name)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error making Read request on User Assigned Identity %q (Resource Group %q): %+v", name, resGroup, err)
	}

	d.Set("name", resp.Name)
	d.Set("resource_group_name", resGroup)
	d.Set("location", resp.Location)

	if props := resp.IdentityProperties; props != nil {
		if principalId := props.PrincipalID; principalId != nil {
			d.Set("principal_id", principalId.String())
		}

		if clientId := props.ClientID; clientId != nil {
			d.Set("client_id", clientId.String())
		}
	}

	return tags.FlattenAndSet(d, resp.Tags)
}

func resourceArmUserAssignedIdentityDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).MSI.UserAssignedIdentitiesClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}

	resGroup := id.ResourceGroup
	name := id.Path["userAssignedIdentities"]

	_, err = client.Delete(ctx, resGroup, name)
	if err != nil {
		return fmt.Errorf("Error deleting User Assigned Identity %q (Resource Group %q): %+v", name, resGroup, err)
	}

	return nil
}
