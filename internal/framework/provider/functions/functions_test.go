// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package functions_test

import (
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider"
)

var testAccProtoV5ProviderFactories = map[string]func() (tfprotov5.ProviderServer, error){
	"kubernetes": providerserver.NewProtocol5WithError(provider.New("test", nil)),
}
