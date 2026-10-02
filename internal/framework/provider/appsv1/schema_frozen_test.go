// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"bytes"
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common/podtemplate"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/internal/schematest"
)

// Every nested block of the pod-template schemas must carry a precomputed
// type; without it each RPC rebuilds the PodSpec type per value node (W5).
// Freezing must not change what Terraform receives from GetProviderSchema.
func TestPodTemplateSchemasFrozen(t *testing.T) {
	ctx := context.Background()
	for name, c := range map[string]struct {
		r     resource.Resource
		build common.SchemaFunc
	}{
		"deployment":   {&DeploymentV1{}, buildDeploymentSchema},
		"daemon_set":   {&DaemonSetV1{}, buildDaemonSetSchema},
		"stateful_set": {&StatefulSetV1{}, buildStatefulSetSchema},
	} {
		t.Run(name, func(t *testing.T) {
			var frozen, built resource.SchemaResponse
			c.r.Schema(ctx, resource.SchemaRequest{}, &frozen)
			c.build(ctx, resource.SchemaRequest{}, &built)
			if paths := common.UnfrozenBlockPaths(frozen.Schema.Blocks); len(paths) != 0 {
				t.Fatalf("blocks without a precomputed type: %v", paths)
			}
			if !frozen.Schema.Type().Equal(built.Schema.Type()) {
				t.Fatal("frozen schema type differs from the built schema type")
			}
			got, err := schematest.ProtocolSchemaJSON(ctx, c.r)
			if err != nil {
				t.Fatal(err)
			}
			want, err := schematest.ProtocolSchemaJSON(ctx, schematest.WithSchema(c.r, c.build))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Fatal("GetProviderSchema differs between the frozen and the built schema")
			}
		})
	}
}

// The memoised template spec type stands in for SpecBlock(options) types, so
// it must not depend on the options.
func TestSpecObjectTypeMatchesTemplateSpecBlocks(t *testing.T) {
	for name, options := range map[string]podtemplate.Options{
		"deployment":   {RestartPolicyAlways: true},
		"daemon_set":   {RestartPolicyAlways: false},
		"stateful_set": statefulSetPodTemplateOptions(),
	} {
		if !podtemplate.SpecBlock(options).NestedObject.Type().Equal(podtemplate.SpecObjectType()) {
			t.Errorf("%s: SpecBlock type differs from SpecObjectType", name)
		}
	}
	if !deploymentSpecBlock().Type().Equal(deploymentSpecListType()) {
		t.Error("deploymentSpecListType differs from deploymentSpecBlock().Type()")
	}
}

// Schema-derived types are needed on every RPC; after the first call they
// must be free.
func TestPodTemplateDerivedTypesMemoised(t *testing.T) {
	_, _ = podtemplate.SpecObjectType(), deploymentSpecListType()
	allocs := testing.AllocsPerRun(10, func() {
		_ = podtemplate.SpecObjectType()
		_ = deploymentSpecListType()
		_ = templateSpecNull()
	})
	if allocs != 0 {
		t.Fatalf("memoised schema types allocate %.0f per call; want 0", allocs)
	}
}

// TypeAtTerraformPath runs for every prefix of every value path during each
// RPC. On the frozen schema it must not rebuild the PodSpec type.
func TestDeploymentTypeAtTerraformPathAllocs(t *testing.T) {
	ctx := context.Background()
	deep := tftypes.NewAttributePath().WithAttributeName("spec").WithElementKeyInt(0).
		WithAttributeName("template").WithElementKeyInt(0).
		WithAttributeName("spec").WithElementKeyInt(0).
		WithAttributeName("container").WithElementKeyInt(0).
		WithAttributeName("liveness_probe").WithElementKeyInt(0).
		WithAttributeName("http_get").WithElementKeyInt(0)
	var frozen, built resource.SchemaResponse
	(&DeploymentV1{}).Schema(ctx, resource.SchemaRequest{}, &frozen)
	buildDeploymentSchema(ctx, resource.SchemaRequest{}, &built)
	measure := func(s resource.SchemaResponse) float64 {
		return testing.AllocsPerRun(20, func() {
			for i := 1; i <= len(deep.Steps()); i++ {
				if _, err := s.Schema.TypeAtTerraformPath(ctx, tftypes.NewAttributePathWithSteps(deep.Steps()[:i])); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
	builtAllocs, frozenAllocs := measure(built), measure(frozen)
	t.Logf("allocs for all %d prefixes: built=%.0f frozen=%.0f", len(deep.Steps()), builtAllocs, frozenAllocs)
	if frozenAllocs > 1000 { // measured 405 frozen vs 14927 unfrozen
		t.Fatalf("TypeAtTerraformPath allocates %.0f per deep path on the frozen schema; want <= 1000", frozenAllocs)
	}
}
