// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/internal/schematest"
)

// Every nested block of the pod-template schemas must carry a precomputed
// type; without it each RPC rebuilds the PodSpec type per value node.
// Freezing must not change what Terraform receives from GetProviderSchema.
func TestPodTemplateSchemasFrozen(t *testing.T) {
	for name, c := range map[string]struct {
		r     resource.Resource
		build common.SchemaFunc
	}{
		"pod": {&PodV1{}, buildPodV1Schema},
	} {
		t.Run(name, func(t *testing.T) {
			schematest.AssertFrozenSchemaUnchanged(t, c.r, c.build)
		})
	}
}

// The memoised Pod spec type must equal the type of the schema's spec object
// and be free after the first call.
func TestPodSpecDerivedMemoised(t *testing.T) {
	if !podSpecDerived().objectType.Equal(podSpecObject().Type()) {
		t.Fatal("memoised Pod spec type differs from podSpecObject().Type()")
	}
	allocs := testing.AllocsPerRun(10, func() { _ = podSpecDerived() })
	if allocs != 0 {
		t.Fatalf("podSpecDerived allocates %.0f per call; want 0", allocs)
	}
}
