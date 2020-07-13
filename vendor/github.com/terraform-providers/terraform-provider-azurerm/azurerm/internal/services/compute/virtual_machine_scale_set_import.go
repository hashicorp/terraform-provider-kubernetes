package compute

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
)

func importVirtualMachineScaleSet(osType compute.OperatingSystemTypes, resourceType string) func(d *schema.ResourceData, meta interface{}) (data []*schema.ResourceData, err error) {
	return func(d *schema.ResourceData, meta interface{}) (data []*schema.ResourceData, err error) {
		id, err := ParseVirtualMachineScaleSetID(d.Id())
		if err != nil {
			return []*schema.ResourceData{}, err
		}

		client := meta.(*clients.Client).Compute.VMScaleSetClient
		ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
		defer cancel()

		vm, err := client.Get(ctx, id.ResourceGroup, id.Name)
		if err != nil {
			return []*schema.ResourceData{}, fmt.Errorf("Error retrieving Virtual Machine Scale Set %q (Resource Group %q): %+v", id.Name, id.ResourceGroup, err)
		}

		if vm.VirtualMachineScaleSetProperties == nil {
			return []*schema.ResourceData{}, fmt.Errorf("Error retrieving Virtual Machine Scale Set %q (Resource Group %q): `properties` was nil", id.Name, id.ResourceGroup)
		}

		if vm.VirtualMachineScaleSetProperties.VirtualMachineProfile == nil {
			return []*schema.ResourceData{}, fmt.Errorf("Error retrieving Virtual Machine Scale Set %q (Resource Group %q): `properties.virtualMachineProfile` was nil", id.Name, id.ResourceGroup)
		}

		if vm.VirtualMachineScaleSetProperties.VirtualMachineProfile.OsProfile == nil {
			return []*schema.ResourceData{}, fmt.Errorf("Error retrieving Virtual Machine Scale Set %q (Resource Group %q): `properties.virtualMachineProfile.osProfile` was nil", id.Name, id.ResourceGroup)
		}

		isCorrectOS := false
		hasSshKeys := false
		if profile := vm.VirtualMachineScaleSetProperties.VirtualMachineProfile.OsProfile; profile != nil {
			if profile.LinuxConfiguration != nil && osType == compute.Linux {
				isCorrectOS = true

				if profile.LinuxConfiguration.SSH != nil && profile.LinuxConfiguration.SSH.PublicKeys != nil {
					hasSshKeys = len(*profile.LinuxConfiguration.SSH.PublicKeys) > 0
				}
			}

			if profile.WindowsConfiguration != nil && osType == compute.Windows {
				isCorrectOS = true
			}
		}

		if !isCorrectOS {
			return []*schema.ResourceData{}, fmt.Errorf("The %q resource only supports %s Virtual Machine Scale Sets", resourceType, string(osType))
		}

		if !hasSshKeys {
			d.Set("admin_password", "ignored-as-imported")
		}

		return []*schema.ResourceData{d}, nil
	}
}
