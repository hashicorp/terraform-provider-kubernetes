// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1_test

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	framework "github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

func TestBatchMuxRegistration(t *testing.T) {
	ctx := context.Background()
	sdk := kubernetes.Provider()
	for _, name := range []string{"kubernetes_job_v1", "kubernetes_cron_job_v1"} {
		if _, exists := sdk.ResourcesMap[name]; exists {
			t.Fatalf("%s is registered on SDKv2 and Framework", name)
		}
	}
	for _, name := range []string{"kubernetes_job", "kubernetes_cron_job"} {
		if _, exists := sdk.ResourcesMap[name]; !exists {
			t.Fatalf("deprecated alias %s must remain available", name)
		}
	}

	provider := framework.New("test", sdk.Meta)
	counts := map[string]int{}
	for _, constructor := range provider.Resources(ctx) {
		var response fwresource.MetadataResponse
		constructor().Metadata(ctx, fwresource.MetadataRequest{ProviderTypeName: "kubernetes"}, &response)
		counts[response.TypeName]++
	}
	for _, name := range []string{"kubernetes_job_v1", "kubernetes_cron_job_v1"} {
		if counts[name] != 1 {
			t.Fatalf("%s has %d Framework registrations, want one", name, counts[name])
		}
	}

	server, err := mux.MuxServer(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	response, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range response.Diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("mux schema: %s: %s", diagnostic.Summary, diagnostic.Detail)
		}
	}
	for _, name := range []string{
		"kubernetes_job_v1", "kubernetes_cron_job_v1",
		"kubernetes_job", "kubernetes_cron_job", "kubernetes_namespace_v1", "kubernetes_manifest",
	} {
		if response.ResourceSchemas[name] == nil {
			t.Fatalf("production mux does not serve %s", name)
		}
	}
	if !response.ResourceSchemas["kubernetes_job_v1"].ValueType().Equal(response.ResourceSchemas["kubernetes_job"].ValueType()) {
		t.Fatal("Job migration changed persisted value types")
	}
	if response.ResourceSchemas["kubernetes_job_v1"].Version != 1 {
		t.Fatal("Job schema version must retain version 1")
	}
	if response.ResourceSchemas["kubernetes_cron_job_v1"].Version != 0 {
		t.Fatal("CronJob v1 schema version must retain version 0")
	}
}
