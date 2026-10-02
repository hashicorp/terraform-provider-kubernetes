// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rpcperf

import (
	"testing"
)

// allocCeilings bound the heap allocations of one no-op PlanResourceChange
// and one ReadResource per resource. Allocation counts are deterministic, so
// unlike timings they can gate CI. Each ceiling is about 1.5x the count
// measured with frozen schemas (terraform-plugin-framework v1.16.1, go1.26).
// Without frozen schemas a no-op plan allocated 4-17 million times and a read
// 1.3-5.9 million times, so a lost freeze fails this test by a wide margin.
var allocCeilings = map[string]struct{ planNoop, read float64 }{
	// Measured on darwin/arm64: plan ~785k, read ~413k.
	"kubernetes_deployment_v1": {planNoop: 1_200_000, read: 620_000},
	// Measured: plan ~776k, read ~476k.
	"kubernetes_daemon_set_v1": {planNoop: 1_200_000, read: 720_000},
	// Measured: plan ~785k, read ~483k.
	"kubernetes_stateful_set_v1": {planNoop: 1_200_000, read: 720_000},
	// Measured: plan ~496k, read ~235k.
	"kubernetes_pod_v1": {planNoop: 750_000, read: 360_000},
	// Measured: plan ~759k, read ~402k.
	"kubernetes_job_v1": {planNoop: 1_150_000, read: 600_000},
	// Measured: plan ~1047k, read ~558k.
	"kubernetes_cron_job_v1": {planNoop: 1_600_000, read: 840_000},
}

func TestPodTemplateRPCAllocCeilings(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts are not representative under the race detector")
	}
	for _, tn := range podTemplateResources {
		t.Run(tn, func(t *testing.T) {
			ceiling, ok := allocCeilings[tn]
			if !ok {
				t.Fatalf("no allocation ceiling for %s", tn)
			}
			e := newCreatedEnv(t, tn)
			plan := testing.AllocsPerRun(2, func() { e.planNoop(t) })
			read := testing.AllocsPerRun(2, func() { e.read(t) })
			t.Logf("allocs/op: no-op plan %.0f (ceiling %.0f), read %.0f (ceiling %.0f)", plan, ceiling.planNoop, read, ceiling.read)
			if plan > ceiling.planNoop {
				t.Errorf("no-op PlanResourceChange allocates %.0f times; ceiling %.0f", plan, ceiling.planNoop)
			}
			if read > ceiling.read {
				t.Errorf("ReadResource allocates %.0f times; ceiling %.0f", read, ceiling.read)
			}
		})
	}
}
