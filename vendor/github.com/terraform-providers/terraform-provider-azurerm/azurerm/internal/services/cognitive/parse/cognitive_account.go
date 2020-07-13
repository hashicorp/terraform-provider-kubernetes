package parse

import (
	"fmt"

	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
)

type CognitiveAccountId struct {
	ResourceGroup string
	Name          string
}

func CognitiveAccountID(input string) (*CognitiveAccountId, error) {
	id, err := azure.ParseAzureResourceID(input)
	if err != nil {
		return nil, fmt.Errorf("[ERROR] Unable to parse Cognitive Account ID %q: %+v", input, err)
	}

	account := CognitiveAccountId{
		ResourceGroup: id.ResourceGroup,
	}

	if account.Name, err = id.PopSegment("accounts"); err != nil {
		return nil, err
	}

	if err := id.ValidateNoEmptySegments(input); err != nil {
		return nil, err
	}

	return &account, nil
}
