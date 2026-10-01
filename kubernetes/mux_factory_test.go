// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import "github.com/hashicorp/terraform-plugin-go/tfprotov6"

// TestAccMuxProviderFactories is initialized by the external test package, which
// can import the production mux without introducing a kubernetes import cycle.
var TestAccMuxProviderFactories map[string]func() (tfprotov6.ProviderServer, error)
