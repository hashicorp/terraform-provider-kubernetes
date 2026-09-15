// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package mux_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
)

// TestMuxServerSchema builds the production mux and asks it for the full
// provider schema. It needs no cluster and no credentials, and it catches two
// whole-provider failure modes that are otherwise only visible at customer
// runtime:
//
//   - A resource type registered on two servers. The mux rejects a duplicate
//     outright, so every terraform plan in an organisation would fail after a
//     migration that forgot to remove the SDKv2 registration (K8S-MIGRATE-008).
//   - A schema that fails ValidateImplementation — a Framework attribute with a
//     Default but no Computed, for instance, fails GetProviderSchema for EVERY
//     resource, not just the offending one (K8S-MIGRATE-004).
func TestMuxServerSchema(t *testing.T) {
	ctx := context.Background()

	server, err := mux.MuxServer(ctx, "test")
	if err != nil {
		t.Fatalf("building the mux server: %s", err)
	}

	resp, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("GetProviderSchema: %s", err)
	}
	for _, d := range resp.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Errorf("GetProviderSchema diagnostic: %s — %s", d.Summary, d.Detail)
		}
	}

	// Spot-check the resource this migration moved, and the names that
	// deliberately did NOT move with it.
	for _, name := range []string{
		"kubernetes_namespace_v1",
		"kubernetes_namespace",
	} {
		if _, ok := resp.ResourceSchemas[name]; !ok {
			t.Errorf("resource %q is not served by the mux; a type served by neither server simply vanishes", name)
		}
	}
	for _, name := range []string{
		"kubernetes_namespace_v1",
		"kubernetes_namespace",
	} {
		if _, ok := resp.DataSourceSchemas[name]; !ok {
			t.Errorf("data source %q is not served by the mux", name)
		}
	}

	// The migrated resource must be the Framework one: schema version 1 with a
	// metadata BLOCK. A version of 0 would mean the SDKv2 implementation is
	// still answering, and metadata appearing as an attribute rather than a
	// block would mean every user's HCL stopped parsing.
	ns := resp.ResourceSchemas["kubernetes_namespace_v1"]
	if ns == nil || ns.Block == nil {
		t.Fatal("kubernetes_namespace_v1 has no schema block")
	}
	if ns.Version != 1 {
		t.Errorf("kubernetes_namespace_v1 schema version = %d, want 1", ns.Version)
	}

	var foundMetadataBlock bool
	for _, b := range ns.Block.BlockTypes {
		if b.TypeName == "metadata" {
			foundMetadataBlock = true
			if b.Nesting != tfprotov6.SchemaNestedBlockNestingModeList {
				t.Errorf("metadata nesting = %v, want List; anything else changes the stored cty type and breaks metadata[0] indexing", b.Nesting)
			}
		}
	}
	if !foundMetadataBlock {
		t.Error("kubernetes_namespace_v1 has no metadata block; if it became an attribute, every existing configuration stops parsing")
	}

	// id is a published attribute of this resource and in-repo configurations
	// reference it.
	var foundID bool
	for _, a := range ns.Block.Attributes {
		if a.Name == "id" {
			foundID = true
		}
	}
	if !foundID {
		t.Error("kubernetes_namespace_v1 no longer publishes id")
	}
}
