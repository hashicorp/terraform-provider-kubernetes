package mariadb

import (
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/mariadb/mgmt/2018-06-01/mariadb"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func resourceArmMariaDbConfiguration() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmMariaDbConfigurationCreateUpdate,
		Read:   resourceArmMariaDbConfigurationRead,
		Delete: resourceArmMariaDbConfigurationDelete,

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

			"resource_group_name": azure.SchemaResourceGroupName(),

			"server_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringMatch(
					regexp.MustCompile("^[-a-zA-Z0-9]{3,50}$"),
					"server_name must be 3 - 50 characters long, and contain only letters, numbers and hyphens",
				),
			},

			"value": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceArmMariaDbConfigurationCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).MariaDB.ConfigurationsClient
	ctx, cancel := timeouts.ForCreateUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	log.Printf("[INFO] preparing arguments for AzureRM MariaDb Configuration creation.")

	name := d.Get("name").(string)
	resourceGroup := d.Get("resource_group_name").(string)
	serverName := d.Get("server_name").(string)
	value := d.Get("value").(string)

	properties := mariadb.Configuration{
		ConfigurationProperties: &mariadb.ConfigurationProperties{
			Value: utils.String(value),
		},
	}

	future, err := client.CreateOrUpdate(ctx, resourceGroup, serverName, name, properties)
	if err != nil {
		return fmt.Errorf("Error issuing create/update request for MariaDb Configuration %s (resource group %s, server name %s): %v", name, resourceGroup, serverName, err)
	}

	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("Error waiting for create/update of MariaDb Configuration %s (resource group %s, server name %s): %v", name, resourceGroup, serverName, err)
	}

	read, err := client.Get(ctx, resourceGroup, serverName, name)
	if err != nil {
		return fmt.Errorf("Error issuing get request for MariaDb Configuration %s (resource group %s, server name %s): %v", name, resourceGroup, serverName, err)
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read MariaDb Configuration %s (resource group %s, server name %s) ID", name, resourceGroup, serverName)
	}

	d.SetId(*read.ID)

	return resourceArmMariaDbConfigurationRead(d, meta)
}

func resourceArmMariaDbConfigurationRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).MariaDB.ConfigurationsClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resourceGroup := id.ResourceGroup
	serverName := id.Path["servers"]
	name := id.Path["configurations"]

	resp, err := client.Get(ctx, resourceGroup, serverName, name)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			log.Printf("[WARN] MariaDb Configuration %q was not found (resource group %q)", name, resourceGroup)
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error making Read request on Azure MariaDb Configuration %q (Resource Group %q): %+v", name, resourceGroup, err)
	}

	d.Set("name", resp.Name)
	d.Set("server_name", serverName)
	d.Set("resource_group_name", resourceGroup)
	d.Set("value", resp.ConfigurationProperties.Value)

	return nil
}

func resourceArmMariaDbConfigurationDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).MariaDB.ConfigurationsClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resourceGroup := id.ResourceGroup
	serverName := id.Path["servers"]
	name := id.Path["configurations"]

	// "delete" = resetting this to the default value
	resp, err := client.Get(ctx, resourceGroup, serverName, name)
	if err != nil {
		return fmt.Errorf("Error retrieving MariaDb Configuration %q (Resource Group %q): %+v", name, resourceGroup, err)
	}

	properties := mariadb.Configuration{
		ConfigurationProperties: &mariadb.ConfigurationProperties{
			// we can alternatively set `source: "system-default"`
			Value: resp.DefaultValue,
		},
	}

	future, err := client.CreateOrUpdate(ctx, resourceGroup, serverName, name, properties)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, client.Client)
}
