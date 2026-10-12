// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package mux_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
)

// TestMuxServer_namespaceV1Registered calls GetProviderSchema on the real
// MuxServer and asserts that kubernetes_namespace_v1 is present in the
// resource_schemas map. This proves that the Framework corev1.NewNamespaceV1
// registration in internal/framework/provider/provider.go was not dropped.
//
// The test does NOT require a running Kubernetes cluster or TF_ACC; it only
// exercises the provider schema protocol, which is a pure in-process call.
func TestMuxServer_namespaceV1Registered(t *testing.T) {
	ctx := context.Background()

	srv, err := mux.MuxServer(ctx, "test")
	if err != nil {
		t.Fatalf("MuxServer: %v", err)
	}

	resp, err := srv.GetProviderSchema(ctx, nil)
	if err != nil {
		t.Fatalf("GetProviderSchema: %v", err)
	}

	const want = "kubernetes_namespace_v1"
	if _, ok := resp.ResourceSchemas[want]; !ok {
		t.Errorf("resource %q is not registered in the MuxServer schema", want)
		t.Logf("registered resources (Framework server contribution):")
		for name := range resp.ResourceSchemas {
			// Only log Framework resources to keep output manageable.
			if name == "kubernetes_namespace_v1" ||
				name == "kubernetes_role_binding_v1" ||
				name == "kubernetes_validating_admission_policy_v1" {
				t.Logf("  %s", name)
			}
		}
	}
}

// TestMuxServer_roleBindingV1Registered is a companion check: confirms that
// the RoleBinding resource — this PR's primary change — is also registered.
func TestMuxServer_roleBindingV1Registered(t *testing.T) {
	ctx := context.Background()

	srv, err := mux.MuxServer(ctx, "test")
	if err != nil {
		t.Fatalf("MuxServer: %v", err)
	}

	resp, err := srv.GetProviderSchema(ctx, nil)
	if err != nil {
		t.Fatalf("GetProviderSchema: %v", err)
	}

	const want = "kubernetes_role_binding_v1"
	if _, ok := resp.ResourceSchemas[want]; !ok {
		t.Errorf("resource %q is not registered in the MuxServer schema", want)
	}
}
