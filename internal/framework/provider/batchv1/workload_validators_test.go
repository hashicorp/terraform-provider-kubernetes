// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestWorkloadValidatorsNullAndUnknown(t *testing.T) {
	for _, rule := range []validator.String{
		workloadPortValidator(), workloadIPAddressValidator(), workloadNullableIntValidator(),
		workloadModeBitsValidator(), workloadPathValidator(), workloadQuantityValidator(),
	} {
		for _, value := range []types.String{types.StringNull(), types.StringUnknown()} {
			var response validator.StringResponse
			rule.ValidateString(context.Background(), validator.StringRequest{
				Path: path.Root("value"), ConfigValue: value,
			}, &response)
			if response.Diagnostics.HasError() {
				t.Fatalf("%s rejected %s: %s", rule.Description(context.Background()), value, response.Diagnostics)
			}
		}
	}
}
