// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
)

// testAccMuxProviderFactories serves the full SDKv2 + manifest + Framework mux stack via
// internal/mux.MuxServer. The secret data source tests need it because they create the
// fixture with the SDKv2 kubernetes_secret_v1 resource (only the SDKv2 server serves it)
// and, for the ignore-metadata case, rely on Terraform Core honouring the HCL provider
// block — which the muxed SDKv2 provider receives, unlike the Framework-only
// testAccProtoV6ProviderFactories used by the migrated resources.
var testAccMuxProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": func() (tfprotov6.ProviderServer, error) {
		return mux.MuxServer(context.Background(), "test")
	},
}
