package storage

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func dataSourceArmStorageContainer() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceArmStorageContainerRead,

		Timeouts: &schema.ResourceTimeout{
			Read: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"storage_account_name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"container_access_type": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"metadata": MetaDataComputedSchema(),

			// TODO: support for ACL's, Legal Holds and Immutability Policies
			"has_immutability_policy": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"has_legal_hold": {
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}

func dataSourceArmStorageContainerRead(d *schema.ResourceData, meta interface{}) error {
	storageClient := meta.(*clients.Client).Storage
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	containerName := d.Get("name").(string)
	accountName := d.Get("storage_account_name").(string)

	account, err := storageClient.FindAccount(ctx, accountName)
	if err != nil {
		return fmt.Errorf("Error retrieving Account %q for Container %q: %s", accountName, containerName, err)
	}
	if account == nil {
		return fmt.Errorf("Unable to locate Account %q for Storage Container %q", accountName, containerName)
	}

	client, err := storageClient.ContainersClient(ctx, *account)
	if err != nil {
		return fmt.Errorf("Error building Containers Client for Storage Account %q (Resource Group %q): %s", accountName, account.ResourceGroup, err)
	}

	d.SetId(client.GetResourceID(accountName, containerName))

	props, err := client.GetProperties(ctx, accountName, containerName)
	if err != nil {
		if utils.ResponseWasNotFound(props.Response) {
			return fmt.Errorf("Container %q was not found in Account %q / Resource Group %q", containerName, accountName, account.ResourceGroup)
		}

		return fmt.Errorf("Error retrieving Container %q (Account %q / Resource Group %q): %s", containerName, accountName, account.ResourceGroup, err)
	}

	d.Set("name", containerName)

	d.Set("storage_account_name", accountName)

	d.Set("container_access_type", flattenStorageContainerAccessLevel(props.AccessLevel))

	if err := d.Set("metadata", FlattenMetaData(props.MetaData)); err != nil {
		return fmt.Errorf("Error setting `metadata`: %+v", err)
	}

	d.Set("has_immutability_policy", props.HasImmutabilityPolicy)
	d.Set("has_legal_hold", props.HasLegalHold)

	return nil
}
