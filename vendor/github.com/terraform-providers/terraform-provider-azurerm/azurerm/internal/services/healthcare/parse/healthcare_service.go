package parse

import (
	"fmt"

	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
)

type HealthcareServiceId struct {
	ResourceGroup string
	Name          string
}

func HealthcareServiceID(input string) (*HealthcareServiceId, error) {
	id, err := azure.ParseAzureResourceID(input)
	if err != nil {
		return nil, fmt.Errorf("[ERROR] Unable to parse Healthcare Service ID %q: %+v", input, err)
	}

	service := HealthcareServiceId{
		ResourceGroup: id.ResourceGroup,
	}

	if service.Name, err = id.PopSegment("services"); err != nil {
		return nil, err
	}

	if err := id.ValidateNoEmptySegments(input); err != nil {
		return nil, err
	}

	return &service, nil
}
