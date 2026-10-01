// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

func TestPodV1MuxRegistration(t *testing.T) {
	ctx := context.Background()
	legacy := kubernetes.Provider()
	if _, ok := legacy.ResourcesMap["kubernetes_pod_v1"]; ok {
		t.Fatal("managed kubernetes_pod_v1 must not be registered on SDKv2")
	}
	if legacy.ResourcesMap["kubernetes_pod"] == nil ||
		legacy.DataSourcesMap["kubernetes_pod"] == nil ||
		legacy.DataSourcesMap["kubernetes_pod_v1"] == nil {
		t.Fatal("deprecated Pod resource and both Pod data sources must remain registered")
	}
	server, err := mux.MuxServerWithProvider(ctx, "test", legacy)
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
	native := response.ResourceSchemas["kubernetes_pod_v1"]
	alias := response.ResourceSchemas["kubernetes_pod"]
	if native == nil || alias == nil {
		t.Fatal("the full mux must expose both resource types")
		return
	}
	if native.Version != 1 {
		t.Fatalf("schema version = %d, want 1", native.Version)
	}
	if !native.ValueType().Equal(alias.ValueType()) {
		t.Fatalf("native Pod persisted type differs from SDKv2 alias:\nnative: %s\nalias: %s", native.ValueType(), alias.ValueType())
	}
	framework := providerserver.NewProtocol6(provider.New("test", nil))()
	frameworkSchema, err := framework.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil || frameworkSchema.ResourceSchemas["kubernetes_pod_v1"] == nil {
		t.Fatalf("native Pod not served by Framework: %v", err)
	}
}
