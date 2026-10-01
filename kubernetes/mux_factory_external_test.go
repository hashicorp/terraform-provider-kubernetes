// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

func init() {
	kubernetes.TestAccMuxProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"kubernetes": func() (tfprotov6.ProviderServer, error) {
			return mux.MuxServerWithProvider(context.Background(), "test", kubernetes.Provider())
		},
	}
}

func TestMuxFactoryIncludesSDKAndFrameworkWorkloads(t *testing.T) {
	ctx := context.Background()
	server, err := kubernetes.TestAccMuxProviderFactories["kubernetes"]()
	if err != nil {
		t.Fatal(err)
	}
	response, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range response.Diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("%s: %s", diagnostic.Summary, diagnostic.Detail)
		}
	}
	for _, name := range []string{"kubernetes_ingress", "kubernetes_ingress_v1", "kubernetes_service_v1", "kubernetes_deployment_v1", "kubernetes_daemon_set_v1", "kubernetes_stateful_set_v1"} {
		if response.ResourceSchemas[name] == nil {
			t.Errorf("%s missing from mixed-resource test factory", name)
		}
	}
}
