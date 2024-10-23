package helper

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func DynamicToBool(d types.Dynamic) bool {
	// treat null or unknown as unset/falsy
	if d.IsUnknown() || d.IsNull() {
		return false
	}
	switch value := d.UnderlyingValue().(type) {
	case types.Bool:
		return value.ValueBool()
	case types.String:
		// Terraform performs some conversions on assignment we should honor
		// https://developer.hashicorp.com/terraform/language/expressions/types#type-conversion
		switch value.ValueString() {
		case "true":
			return true
		case "false":
			return false
		default:
			panic(fmt.Errorf("%v is not a bool", value.ValueString()))
		}
	default:
		panic(fmt.Errorf("%v is not a bool", value))
	}
}
