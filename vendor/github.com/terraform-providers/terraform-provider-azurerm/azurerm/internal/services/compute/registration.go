package compute

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/features"
)

type Registration struct{}

// Name is the name of this Service
func (r Registration) Name() string {
	return "Compute"
}

// SupportedDataSources returns the supported Data Sources supported by this Service
func (r Registration) SupportedDataSources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		"azurerm_availability_set":          dataSourceArmAvailabilitySet(),
		"azurerm_dedicated_host":            dataSourceArmDedicatedHost(),
		"azurerm_dedicated_host_group":      dataSourceArmDedicatedHostGroup(),
		"azurerm_disk_encryption_set":       dataSourceArmDiskEncryptionSet(),
		"azurerm_managed_disk":              dataSourceArmManagedDisk(),
		"azurerm_image":                     dataSourceArmImage(),
		"azurerm_platform_image":            dataSourceArmPlatformImage(),
		"azurerm_proximity_placement_group": dataSourceArmProximityPlacementGroup(),
		"azurerm_shared_image_gallery":      dataSourceArmSharedImageGallery(),
		"azurerm_shared_image_version":      dataSourceArmSharedImageVersion(),
		"azurerm_shared_image":              dataSourceArmSharedImage(),
		"azurerm_snapshot":                  dataSourceArmSnapshot(),
		"azurerm_virtual_machine":           dataSourceArmVirtualMachine(),
	}
}

// SupportedResources returns the supported Resources supported by this Service
func (r Registration) SupportedResources() map[string]*schema.Resource {
	resources := map[string]*schema.Resource{
		"azurerm_availability_set":                     resourceArmAvailabilitySet(),
		"azurerm_dedicated_host":                       resourceArmDedicatedHost(),
		"azurerm_dedicated_host_group":                 resourceArmDedicatedHostGroup(),
		"azurerm_disk_encryption_set":                  resourceArmDiskEncryptionSet(),
		"azurerm_image":                                resourceArmImage(),
		"azurerm_managed_disk":                         resourceArmManagedDisk(),
		"azurerm_marketplace_agreement":                resourceArmMarketplaceAgreement(),
		"azurerm_proximity_placement_group":            resourceArmProximityPlacementGroup(),
		"azurerm_shared_image_gallery":                 resourceArmSharedImageGallery(),
		"azurerm_shared_image_version":                 resourceArmSharedImageVersion(),
		"azurerm_shared_image":                         resourceArmSharedImage(),
		"azurerm_snapshot":                             resourceArmSnapshot(),
		"azurerm_virtual_machine_data_disk_attachment": resourceArmVirtualMachineDataDiskAttachment(),
		"azurerm_virtual_machine_extension":            resourceArmVirtualMachineExtension(),
		"azurerm_virtual_machine_scale_set":            resourceArmVirtualMachineScaleSet(),
		"azurerm_virtual_machine":                      resourceArmVirtualMachine(),
	}

	// 2.0 resources
	if features.SupportsTwoPointZeroResources() {
		resources["azurerm_linux_virtual_machine"] = resourceLinuxVirtualMachine()
		resources["azurerm_linux_virtual_machine_scale_set"] = resourceArmLinuxVirtualMachineScaleSet()
		resources["azurerm_virtual_machine_scale_set_extension"] = resourceArmVirtualMachineScaleSetExtension()
		resources["azurerm_windows_virtual_machine"] = resourceWindowsVirtualMachine()
		resources["azurerm_windows_virtual_machine_scale_set"] = resourceArmWindowsVirtualMachineScaleSet()
	}

	return resources
}
