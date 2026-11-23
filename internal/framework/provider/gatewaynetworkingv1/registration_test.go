// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package gatewaynetworkingv1_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
	"github.com/stretchr/testify/require"
)

// TestGatewayClassMuxRegistration needs no cluster. It checks that the muxed
// provider, not only the Framework server, serves both types, and that
// GetProviderSchema reports no errors, such as a type name defined by more
// than one of the muxed servers.
func TestGatewayClassMuxRegistration(t *testing.T) {
	ctx := context.Background()
	server, err := mux.MuxServer(ctx, "test")
	require.NoError(t, err)
	response, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	require.NoError(t, err)
	for _, diagnostic := range response.Diagnostics {
		require.NotEqual(t, tfprotov6.DiagnosticSeverityError, diagnostic.Severity, diagnostic.Detail)
	}
	require.Contains(t, response.ResourceSchemas, "kubernetes_gateway_class_v1")
	require.Contains(t, response.DataSourceSchemas, "kubernetes_gateway_class_v1")
}
