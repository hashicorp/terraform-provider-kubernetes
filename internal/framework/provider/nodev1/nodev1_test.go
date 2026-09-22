// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

// Package nodev1_test contains the shared test infrastructure for the nodev1
// framework package. It mirrors the pattern established in admissionregistrationv1.
//
// The external test package (_test suffix) is intentional: it tests the public
// API of the package, not its internals, and avoids import cycles.
package nodev1_test

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	sdkv2 "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// sdkv2providerMeta creates a minimal SDKv2 provider configuration and returns
// the Meta() function pointer. This bridges the SDKv2 Kubernetes client into
// the Framework provider so acceptance tests can reach the cluster.
func sdkv2providerMeta() func() any {
	p := kubernetes.Provider()
	p.Configure(context.Background(), sdkv2.NewResourceConfigRaw(nil))
	return p.Meta
}

// testAccProtoV6ProviderFactories is the provider factory map passed to every
// resource.TestCase. It wires the Framework provider (Protocol v6) with the
// SDKv2 client bridge so tests have a fully operational Kubernetes provider.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": providerserver.NewProtocol6WithError(provider.New("test", sdkv2providerMeta())),
}
