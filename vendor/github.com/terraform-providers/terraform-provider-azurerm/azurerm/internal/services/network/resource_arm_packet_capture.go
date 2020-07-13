package network

import (
	"fmt"
	"log"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-09-01/network"
	"github.com/hashicorp/go-azure-helpers/response"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/tf"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/features"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func resourceArmPacketCapture() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmPacketCaptureCreate,
		Read:   resourceArmPacketCaptureRead,
		Delete: resourceArmPacketCaptureDelete,
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

			"network_watcher_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"target_resource_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"maximum_bytes_per_packet": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				Default:  0,
			},

			"maximum_bytes_per_session": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				Default:  1073741824,
			},

			"maximum_capture_duration": {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				Default:      18000,
				ValidateFunc: validation.IntBetween(1, 18000),
			},

			"storage_location": {
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"file_path": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"storage_account_id": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"storage_path": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"filter": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"local_ip_address": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"local_port": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"protocol": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(network.PcProtocolAny),
								string(network.PcProtocolTCP),
								string(network.PcProtocolUDP),
							}, false),
						},
						"remote_ip_address": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"remote_port": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},
		},
	}
}

func resourceArmPacketCaptureCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Network.PacketCapturesClient
	ctx, cancel := timeouts.ForCreate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	name := d.Get("name").(string)
	watcherName := d.Get("network_watcher_name").(string)
	resourceGroup := d.Get("resource_group_name").(string)

	targetResourceId := d.Get("target_resource_id").(string)
	bytesToCapturePerPacket := d.Get("maximum_bytes_per_packet").(int)
	totalBytesPerSession := d.Get("maximum_bytes_per_session").(int)
	timeLimitInSeconds := d.Get("maximum_capture_duration").(int)

	if features.ShouldResourcesBeImported() {
		existing, err := client.Get(ctx, resourceGroup, watcherName, name)
		if err != nil {
			if !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("Error checking for presence of existing Packet Capture %q (Resource Group %q): %s", name, resourceGroup, err)
			}
		}

		if existing.ID != nil && *existing.ID != "" {
			return tf.ImportAsExistsError("azurerm_packet_capture", *existing.ID)
		}
	}

	storageLocation, err := expandArmPacketCaptureStorageLocation(d)
	if err != nil {
		return err
	}

	filters, err := expandArmPacketCaptureFilters(d)
	if err != nil {
		return err
	}

	properties := network.PacketCapture{
		PacketCaptureParameters: &network.PacketCaptureParameters{
			Target:                  utils.String(targetResourceId),
			StorageLocation:         storageLocation,
			BytesToCapturePerPacket: utils.Int32(int32(bytesToCapturePerPacket)),
			TimeLimitInSeconds:      utils.Int32(int32(timeLimitInSeconds)),
			TotalBytesPerSession:    utils.Int32(int32(totalBytesPerSession)),
			Filters:                 filters,
		},
	}

	future, err := client.Create(ctx, resourceGroup, watcherName, name, properties)
	if err != nil {
		return fmt.Errorf("Error creating Packet Capture %q (Watcher %q / Resource Group %q): %+v", name, watcherName, resourceGroup, err)
	}

	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("Error waiting for creation of Packet Capture %q (Watcher %q / Resource Group %q): %+v", name, watcherName, resourceGroup, err)
	}

	resp, err := client.Get(ctx, resourceGroup, watcherName, name)
	if err != nil {
		return fmt.Errorf("Error retrieving Packet Capture %q (Watcher %q / Resource Group %q): %+v", name, watcherName, resourceGroup, err)
	}

	d.SetId(*resp.ID)

	return resourceArmPacketCaptureRead(d, meta)
}

func resourceArmPacketCaptureRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Network.PacketCapturesClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}

	resourceGroup := id.ResourceGroup
	watcherName := id.Path["networkWatchers"]
	name := id.Path["packetCaptures"]

	resp, err := client.Get(ctx, resourceGroup, watcherName, name)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			log.Printf("[WARN] Packet Capture %q (Watcher %q / Resource Group %q) %qw not found - removing from state", name, watcherName, resourceGroup, id)
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error reading Packet Capture %q (Watcher %q / Resource Group %q) %+v", name, watcherName, resourceGroup, err)
	}

	d.Set("name", name)
	d.Set("network_watcher_name", watcherName)
	d.Set("resource_group_name", resourceGroup)

	if props := resp.PacketCaptureResultProperties; props != nil {
		d.Set("target_resource_id", props.Target)
		d.Set("maximum_bytes_per_packet", int(*props.BytesToCapturePerPacket))
		d.Set("maximum_bytes_per_session", int(*props.TotalBytesPerSession))
		d.Set("maximum_capture_duration", int(*props.TimeLimitInSeconds))

		location := flattenArmPacketCaptureStorageLocation(props.StorageLocation)
		if err := d.Set("storage_location", location); err != nil {
			return fmt.Errorf("Error setting `storage_location`: %+v", err)
		}

		filters := flattenArmPacketCaptureFilters(props.Filters)
		if err := d.Set("filter", filters); err != nil {
			return fmt.Errorf("Error setting `filter`: %+v", err)
		}
	}

	return nil
}

func resourceArmPacketCaptureDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Network.PacketCapturesClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}

	resourceGroup := id.ResourceGroup
	watcherName := id.Path["networkWatchers"]
	name := id.Path["packetCaptures"]

	future, err := client.Delete(ctx, resourceGroup, watcherName, name)
	if err != nil {
		if response.WasNotFound(future.Response()) {
			return nil
		}

		return fmt.Errorf("Error deleting Packet Capture %q (Watcher %q / Resource Group %q): %+v", name, watcherName, resourceGroup, err)
	}

	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		if response.WasNotFound(future.Response()) {
			return nil
		}

		return fmt.Errorf("Error waiting for the deletion of Packet Capture %q (Watcher %q / Resource Group %q): %+v", name, watcherName, resourceGroup, err)
	}

	return nil
}

func expandArmPacketCaptureStorageLocation(d *schema.ResourceData) (*network.PacketCaptureStorageLocation, error) {
	locations := d.Get("storage_location").([]interface{})
	if len(locations) == 0 {
		return nil, fmt.Errorf("Error expandng `storage_location`: not found")
	}

	location := locations[0].(map[string]interface{})

	storageLocation := network.PacketCaptureStorageLocation{}

	if v := location["file_path"]; v != "" {
		storageLocation.FilePath = utils.String(v.(string))
	}
	if v := location["storage_account_id"]; v != "" {
		storageLocation.StorageID = utils.String(v.(string))
	}

	return &storageLocation, nil
}

func flattenArmPacketCaptureStorageLocation(input *network.PacketCaptureStorageLocation) []interface{} {
	if input == nil {
		return []interface{}{}
	}

	output := make(map[string]interface{})

	if path := input.FilePath; path != nil {
		output["file_path"] = *path
	}

	if account := input.StorageID; account != nil {
		output["storage_account_id"] = *account
	}

	if path := input.StoragePath; path != nil {
		output["storage_path"] = *path
	}

	return []interface{}{output}
}

func expandArmPacketCaptureFilters(d *schema.ResourceData) (*[]network.PacketCaptureFilter, error) {
	inputFilters := d.Get("filter").([]interface{})
	if len(inputFilters) == 0 {
		return nil, nil
	}

	filters := make([]network.PacketCaptureFilter, 0)

	for _, v := range inputFilters {
		inputFilter := v.(map[string]interface{})

		localIPAddress := inputFilter["local_ip_address"].(string)
		localPort := inputFilter["local_port"].(string) // TODO: should this be an int?
		protocol := inputFilter["protocol"].(string)
		remoteIPAddress := inputFilter["remote_ip_address"].(string)
		remotePort := inputFilter["remote_port"].(string)

		filter := network.PacketCaptureFilter{
			LocalIPAddress:  utils.String(localIPAddress),
			LocalPort:       utils.String(localPort),
			Protocol:        network.PcProtocol(protocol),
			RemoteIPAddress: utils.String(remoteIPAddress),
			RemotePort:      utils.String(remotePort),
		}
		filters = append(filters, filter)
	}

	return &filters, nil
}

func flattenArmPacketCaptureFilters(input *[]network.PacketCaptureFilter) []interface{} {
	filters := make([]interface{}, 0)

	if inFilter := input; inFilter != nil {
		for _, v := range *inFilter {
			filter := make(map[string]interface{})

			if address := v.LocalIPAddress; address != nil {
				filter["local_ip_address"] = *address
			}

			if port := v.LocalPort; port != nil {
				filter["local_port"] = *port
			}

			filter["protocol"] = string(v.Protocol)

			if address := v.RemoteIPAddress; address != nil {
				filter["remote_ip_address"] = *address
			}

			if port := v.RemotePort; port != nil {
				filter["remote_port"] = *port
			}

			filters = append(filters, filter)
		}
	}

	return filters
}
