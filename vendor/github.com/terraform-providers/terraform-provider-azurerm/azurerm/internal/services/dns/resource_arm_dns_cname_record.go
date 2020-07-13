package dns

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/tf"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/features"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tags"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func resourceArmDnsCNameRecord() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmDnsCNameRecordCreateUpdate,
		Read:   resourceArmDnsCNameRecordRead,
		Update: resourceArmDnsCNameRecordCreateUpdate,
		Delete: resourceArmDnsCNameRecordDelete,
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

			"zone_name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"records": {
				Type:     schema.TypeString,
				Optional: true,
				Removed:  "Use `record` instead. This attribute will be removed in a future version",
			},

			"record": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"target_resource_id"},
			},

			"ttl": {
				Type:     schema.TypeInt,
				Required: true,
			},

			"fqdn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"target_resource_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  azure.ValidateResourceID,
				ConflictsWith: []string{"records"},
			},

			"tags": tags.Schema(),
		},
	}
}

func resourceArmDnsCNameRecordCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Dns.RecordSetsClient
	ctx, cancel := timeouts.ForCreateUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	name := d.Get("name").(string)
	resGroup := d.Get("resource_group_name").(string)
	zoneName := d.Get("zone_name").(string)

	if features.ShouldResourcesBeImported() && d.IsNewResource() {
		existing, err := client.Get(ctx, resGroup, zoneName, name, dns.CNAME)
		if err != nil {
			if !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("Error checking for presence of existing DNS CNAME Record %q (Zone %q / Resource Group %q): %s", name, zoneName, resGroup, err)
			}
		}

		if existing.ID != nil && *existing.ID != "" {
			return tf.ImportAsExistsError("azurerm_dns_cname_record", *existing.ID)
		}
	}

	ttl := int64(d.Get("ttl").(int))
	record := d.Get("record").(string)
	t := d.Get("tags").(map[string]interface{})
	targetResourceId := d.Get("target_resource_id").(string)

	parameters := dns.RecordSet{
		Name: &name,
		RecordSetProperties: &dns.RecordSetProperties{
			Metadata:       tags.Expand(t),
			TTL:            &ttl,
			CnameRecord:    &dns.CnameRecord{},
			TargetResource: &dns.SubResource{},
		},
	}

	if record != "" {
		parameters.RecordSetProperties.CnameRecord.Cname = utils.String(record)
	}

	if targetResourceId != "" {
		parameters.RecordSetProperties.TargetResource.ID = utils.String(targetResourceId)
	}

	// TODO: this can be removed when the provider SDK is upgraded
	if record == "" && targetResourceId == "" {
		return fmt.Errorf("One of either `record` or `target_resource_id` must be specified")
	}

	eTag := ""
	ifNoneMatch := "" // set to empty to allow updates to records after creation
	if _, err := client.CreateOrUpdate(ctx, resGroup, zoneName, name, dns.CNAME, parameters, eTag, ifNoneMatch); err != nil {
		return fmt.Errorf("Error creating/updating CNAME Record %q (DNS Zone %q / Resource Group %q): %s", name, zoneName, resGroup, err)
	}

	resp, err := client.Get(ctx, resGroup, zoneName, name, dns.CNAME)
	if err != nil {
		return fmt.Errorf("Error retrieving CNAME Record %q (DNS Zone %q / Resource Group %q): %s", name, zoneName, resGroup, err)
	}

	if resp.ID == nil {
		return fmt.Errorf("Error retrieving CNAME Record %q (DNS Zone %q / Resource Group %q): ID was nil", name, zoneName, resGroup)
	}

	d.SetId(*resp.ID)

	return resourceArmDnsCNameRecordRead(d, meta)
}

func resourceArmDnsCNameRecordRead(d *schema.ResourceData, meta interface{}) error {
	dnsClient := meta.(*clients.Client).Dns.RecordSetsClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}

	resGroup := id.ResourceGroup
	name := id.Path["CNAME"]
	zoneName := id.Path["dnszones"]

	resp, err := dnsClient.Get(ctx, resGroup, zoneName, name, dns.CNAME)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error retrieving CNAME Record %s (DNS Zone %q / Resource Group %q): %+v", name, zoneName, resGroup, err)
	}

	d.Set("name", name)
	d.Set("resource_group_name", resGroup)
	d.Set("zone_name", zoneName)
	d.Set("fqdn", resp.Fqdn)
	d.Set("ttl", resp.TTL)

	if props := resp.RecordSetProperties; props != nil {
		cname := ""
		if props.CnameRecord != nil && props.CnameRecord.Cname != nil {
			cname = *props.CnameRecord.Cname
		}
		d.Set("record", cname)

		targetResourceId := ""
		if props.TargetResource != nil && props.TargetResource.ID != nil {
			targetResourceId = *props.TargetResource.ID
		}
		d.Set("target_resource_id", targetResourceId)
	}

	return tags.FlattenAndSet(d, resp.Metadata)
}

func resourceArmDnsCNameRecordDelete(d *schema.ResourceData, meta interface{}) error {
	dnsClient := meta.(*clients.Client).Dns.RecordSetsClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}

	resGroup := id.ResourceGroup
	name := id.Path["CNAME"]
	zoneName := id.Path["dnszones"]

	resp, err := dnsClient.Delete(ctx, resGroup, zoneName, name, dns.CNAME, "")
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Error deleting CNAME Record %q (DNS Zone %q / Resource Group %q): %+v", name, zoneName, resGroup, err)
	}

	return nil
}
