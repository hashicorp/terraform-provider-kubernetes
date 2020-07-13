package automation

import (
	"fmt"
	"log"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/automation/mgmt/2015-10-31/automation"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/tf"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/features"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func resourceArmAutomationCredential() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmAutomationCredentialCreateUpdate,
		Read:   resourceArmAutomationCredentialRead,
		Update: resourceArmAutomationCredentialCreateUpdate,
		Delete: resourceArmAutomationCredentialDelete,

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
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"resource_group_name": azure.SchemaResourceGroupName(),

			//this is AutomationAccountName in the SDK
			"account_name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				Deprecated:    "account_name has been renamed to automation_account_name for clarity and to match the azure API",
				ConflictsWith: []string{"automation_account_name"},
				ValidateFunc:  azure.ValidateAutomationAccountName(),
			},

			"automation_account_name": {
				Type:          schema.TypeString,
				Optional:      true, //todo change to required once account_name has been removed
				Computed:      true, // todo remove once account_name has been removed
				ForceNew:      true,
				ConflictsWith: []string{"account_name"}, // todo remove once account_name has been removed
				ValidateFunc:  azure.ValidateAutomationAccountName(),
			},

			"username": {
				Type:     schema.TypeString,
				Required: true,
			},

			"password": {
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceArmAutomationCredentialCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Automation.CredentialClient
	ctx, cancel := timeouts.ForCreateUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	log.Printf("[INFO] preparing arguments for AzureRM Automation Credential creation.")

	name := d.Get("name").(string)
	resGroup := d.Get("resource_group_name").(string)
	// todo remove this once `account_name` is removed
	accountName := ""
	if v, ok := d.GetOk("automation_account_name"); ok {
		accountName = v.(string)
	} else if v, ok := d.GetOk("account_name"); ok {
		accountName = v.(string)
	}

	if features.ShouldResourcesBeImported() && d.IsNewResource() {
		existing, err := client.Get(ctx, resGroup, accountName, name)
		if err != nil {
			if !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("Error checking for presence of existing Automation Credential %q (Account %q / Resource Group %q): %s", name, accountName, resGroup, err)
			}
		}

		if existing.ID != nil && *existing.ID != "" {
			return tf.ImportAsExistsError("azurerm_automation_credential", *existing.ID)
		}
	}

	user := d.Get("username").(string)
	password := d.Get("password").(string)
	description := d.Get("description").(string)

	parameters := automation.CredentialCreateOrUpdateParameters{
		CredentialCreateOrUpdateProperties: &automation.CredentialCreateOrUpdateProperties{
			UserName:    &user,
			Password:    &password,
			Description: &description,
		},
		Name: &name,
	}

	if _, err := client.CreateOrUpdate(ctx, resGroup, accountName, name, parameters); err != nil {
		return err
	}

	read, err := client.Get(ctx, resGroup, accountName, name)
	if err != nil {
		return err
	}

	if read.ID == nil {
		return fmt.Errorf("Cannot read Automation Credential '%s' (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmAutomationCredentialRead(d, meta)
}

func resourceArmAutomationCredentialRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Automation.CredentialClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	accountName := id.Path["automationAccounts"]
	name := id.Path["credentials"]

	resp, err := client.Get(ctx, resGroup, accountName, name)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error making Read request on AzureRM Automation Credential '%s': %+v", name, err)
	}

	d.Set("name", resp.Name)
	d.Set("resource_group_name", resGroup)
	d.Set("automation_account_name", accountName)
	d.Set("account_name", accountName)
	if props := resp.CredentialProperties; props != nil {
		d.Set("username", props.UserName)
	}
	d.Set("description", resp.Description)

	return nil
}

func resourceArmAutomationCredentialDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Automation.CredentialClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	accountName := id.Path["automationAccounts"]
	name := id.Path["credentials"]

	resp, err := client.Delete(ctx, resGroup, accountName, name)
	if err != nil {
		if utils.ResponseWasNotFound(resp) {
			return nil
		}

		return fmt.Errorf("Error issuing AzureRM delete request for Automation Credential '%s': %+v", name, err)
	}

	return nil
}
