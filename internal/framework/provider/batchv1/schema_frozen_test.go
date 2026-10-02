// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

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
		"job":      {&JobV1{}, buildJobSchema},
		"cron_job": {&CronJobV1{}, buildCronJobSchema},
	} {
		t.Run(name, func(t *testing.T) {
			schematest.AssertFrozenSchemaUnchanged(t, c.r, c.build)
		})
	}
}

// The memoised value-field trees must describe the same types as the blocks
// they were derived from, and must be free after the first call.
func TestBatchSpecValueFieldsMemoised(t *testing.T) {
	if !jobSpecValueField().typ.Equal(jobSpecBlock(false).Type()) {
		t.Error("jobSpecValueField type differs from jobSpecBlock(false).Type()")
	}
	if !cronJobSpecValueField().typ.Equal(cronJobSpecBlock().Type()) {
		t.Error("cronJobSpecValueField type differs from cronJobSpecBlock().Type()")
	}
	if !jobSpecType().Equal(jobSpecBlock(false).NestedObject.Type()) {
		t.Error("jobSpecType differs from jobSpecBlock(false).NestedObject.Type()")
	}
	allocs := testing.AllocsPerRun(10, func() {
		_ = jobSpecValueField()
		_ = cronJobSpecValueField()
		_ = jobSpecType()
	})
	if allocs != 0 {
		t.Fatalf("memoised value fields allocate %.0f per call; want 0", allocs)
	}
}
