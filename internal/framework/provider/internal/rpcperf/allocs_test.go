// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rpcperf

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	frameworkprovider "github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/internal/schematest"
)

// allocCeilings bound the heap allocations of one no-op PlanResourceChange
// and one ReadResource per resource. Allocation counts are deterministic, so
// unlike timings they can gate CI. Each ceiling is about 1.5x the count
// measured with frozen schemas (terraform-plugin-framework v1.16.1). Without
// frozen schemas a no-op plan allocated 4-17 million times and a read 1.3-5.9
// million times, so a lost freeze fails this test by a wide margin.
//
// Measured plan / read allocs per platform with go1.26.3 (single runs; run to
// run variation is 1-3%, and go1.26.4 gives the same counts within 1%):
//
//	resource       darwin/arm64   darwin/amd64   linux/arm64    linux/amd64
//	deployment     774k / 415k    776k / 405k    780k / 405k    790k / 408k
//	daemon_set     758k / 472k    751k / 476k    765k / 476k    761k / 470k
//	stateful_set   783k / 478k    785k / 486k    794k / 476k    772k / 479k
//	pod            481k / 235k    498k / 234k    484k / 236k    488k / 240k
//	job            764k / 400k    745k / 396k    765k / 403k    757k / 397k
//	cron_job      1042k / 553k   1038k / 552k   1041k / 553k   1044k / 558k
var allocCeilings = map[string]struct{ planNoop, read float64 }{
	"kubernetes_deployment_v1":   {planNoop: 1_200_000, read: 620_000},
	"kubernetes_daemon_set_v1":   {planNoop: 1_200_000, read: 720_000},
	"kubernetes_stateful_set_v1": {planNoop: 1_200_000, read: 720_000},
	"kubernetes_pod_v1":          {planNoop: 750_000, read: 360_000},
	"kubernetes_job_v1":          {planNoop: 1_150_000, read: 600_000},
	"kubernetes_cron_job_v1":     {planNoop: 1_600_000, read: 840_000},
}

func TestPodTemplateRPCAllocCeilings(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts are not representative under the race detector")
	}
	// An unfrozen schema makes each RPC take up to a second, so the subtests
	// below could exceed the package timeout before reporting any numbers.
	// Name the cause up front instead.
	frozenSchemaPreflight(t)
	for _, tn := range podTemplateResources {
		t.Run(tn, func(t *testing.T) {
			ceiling, ok := allocCeilings[tn]
			if !ok {
				t.Fatalf("no allocation ceiling for %s", tn)
			}
			e := newCreatedEnv(t, tn)
			// newCreatedEnv has already warmed up the server, and the counts
			// are deterministic to a few percent, so a single run suffices.
			plan := testing.AllocsPerRun(1, func() { e.planNoop(t) })
			read := testing.AllocsPerRun(1, func() { e.read(t) })
			t.Logf("allocs/op: no-op plan %.0f (ceiling %.0f), read %.0f (ceiling %.0f)", plan, ceiling.planNoop, read, ceiling.read)
			if plan > ceiling.planNoop {
				t.Errorf("no-op PlanResourceChange allocates %.0f times; ceiling %.0f", plan, ceiling.planNoop)
			}
			if read > ceiling.read {
				t.Errorf("ReadResource allocates %.0f times; ceiling %.0f", read, ceiling.read)
			}
			if plan > 2*ceiling.planNoop || read > 2*ceiling.read {
				t.Fatal("more than twice the ceiling; skipping the remaining resources")
			}
		})
		if t.Failed() {
			break
		}
	}
}

// frozenSchemaPreflight fails the test when a pod-template resource serves a
// schema with nested blocks lacking a precomputed type (common.FrozenSchema).
func frozenSchemaPreflight(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	want := map[string]bool{}
	for _, tn := range podTemplateResources {
		want[tn] = true
	}
	for _, newResource := range frameworkprovider.New("dev", nil).Resources(ctx) {
		r := newResource()
		var md resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "kubernetes"}, &md)
		if !want[md.TypeName] {
			continue
		}
		delete(want, md.TypeName)
		var sr resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &sr)
		if paths := schematest.UnfrozenBlockPaths(sr.Schema.Blocks); len(paths) != 0 {
			t.Fatalf("%s serves %d nested blocks without a precomputed type (is its Schema wrapped in common.FrozenSchema?), e.g. %v",
				md.TypeName, len(paths), paths[:min(len(paths), 5)])
		}
	}
	if len(want) != 0 {
		t.Fatalf("Framework provider does not serve %v", want)
	}
}
