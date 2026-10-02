// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rpcperf

import (
	"context"
	"encoding/json"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

// Run with, for example:
//
//	go test ./internal/framework/provider/internal/rpcperf -run '^$' -bench . -benchtime 10x

func BenchmarkRPCPlanNoop(b *testing.B) {
	for _, tn := range podTemplateResources {
		b.Run(tn, func(b *testing.B) {
			e := newCreatedEnv(b, tn)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				e.planNoop(b)
			}
		})
	}
}

func BenchmarkRPCRead(b *testing.B) {
	for _, tn := range podTemplateResources {
		b.Run(tn, func(b *testing.B) {
			e := newCreatedEnv(b, tn)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				e.read(b)
			}
		})
	}
}

func BenchmarkRPCValidate(b *testing.B) {
	for _, tn := range podTemplateResources {
		b.Run(tn, func(b *testing.B) {
			e := newEnv(b, tn)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				e.validate(b)
			}
		})
	}
}

// BenchmarkRPCApplyCreate measures ApplyResourceChange for a new resource,
// from a plan computed once outside the timed loop.
func BenchmarkRPCApplyCreate(b *testing.B) {
	for _, tn := range podTemplateResources {
		b.Run(tn, func(b *testing.B) {
			e := newEnv(b, tn)
			plan := e.planCreate(b)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				e.applyCreate(b, plan)
			}
		})
	}
}

// TestRPCRefreshPlan50 is the provider-side cost of `terraform plan` for a
// workspace with 50 Deployments: sequential Read + no-op Plan, 50 times.
func TestRPCRefreshPlan50(t *testing.T) {
	if os.Getenv("RPCPERF_SCALE") == "" {
		t.Skip("set RPCPERF_SCALE=1 to run")
	}
	const n = 50
	e := newCreatedEnv(t, "kubernetes_deployment_v1")
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()
	for i := 0; i < n; i++ {
		rr := e.read(t)
		e.state, e.private = rr.NewState, rr.Private
		if np := e.planNoop(t); len(np.RequiresReplace) != 0 {
			t.Fatalf("no-op plan requires replacement: %v", np.RequiresReplace)
		}
	}
	wall := time.Since(start)
	runtime.ReadMemStats(&after)
	t.Logf("50 x (Read + no-op Plan) of kubernetes_deployment_v1: wall=%s totalAlloc=%.1fMB mallocs=%d numGC=%d",
		wall.Round(time.Millisecond), float64(after.TotalAlloc-before.TotalAlloc)/1e6,
		after.Mallocs-before.Mallocs, after.NumGC-before.NumGC)
}

// TestRPCDumpProviderSchema writes the complete GetProviderSchema response of
// the production mux as JSON, to compare builds byte for byte.
func TestRPCDumpProviderSchema(t *testing.T) {
	out := os.Getenv("RPCPERF_SCHEMA_OUT")
	if out == "" {
		t.Skip("set RPCPERF_SCHEMA_OUT=<file> to run")
	}
	ctx := context.Background()
	srv, err := mux.MuxServerWithProvider(ctx, "dev", kubernetes.Provider())
	if err != nil {
		t.Fatal(err)
	}
	resp, err := srv.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if d := diagsErr(resp.Diagnostics); d != "" {
		t.Fatal(d)
	}
	js, err := json.MarshalIndent(resp, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, js, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %d bytes, %d resource schemas", len(js), len(resp.ResourceSchemas))
}

// TestRPCDumpValues writes, for each pod-template resource, the planned state
// of a create, the applied state, the refreshed state, the no-op planned state
// and its RequiresReplace paths, to compare builds byte for byte.
func TestRPCDumpValues(t *testing.T) {
	out := os.Getenv("RPCPERF_VALUES_OUT")
	if out == "" {
		t.Skip("set RPCPERF_VALUES_OUT=<file> to run")
	}
	result := map[string]any{}
	for _, tn := range podTemplateResources {
		e := newEnv(t, tn)
		plan := e.planCreate(t)
		planned, err := plan.PlannedState.Unmarshal(e.typ)
		if err != nil {
			t.Fatal(err)
		}
		e.state, e.private = e.applyCreate(t, plan)
		applied, err := e.state.Unmarshal(e.typ)
		if err != nil {
			t.Fatal(err)
		}
		rr := e.read(t)
		e.state, e.private = rr.NewState, rr.Private
		read, err := rr.NewState.Unmarshal(e.typ)
		if err != nil {
			t.Fatal(err)
		}
		np := e.planNoop(t)
		noop, err := np.PlannedState.Unmarshal(e.typ)
		if err != nil {
			t.Fatal(err)
		}
		var replace []string
		for _, p := range np.RequiresReplace {
			replace = append(replace, p.String())
		}
		result[tn] = map[string]any{
			"planCreate":      toJSONAny(planned),
			"applyCreate":     toJSONAny(applied),
			"read":            toJSONAny(read),
			"planNoop":        toJSONAny(noop),
			"requiresReplace": replace,
		}
	}
	js, err := json.MarshalIndent(result, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, js, 0o600); err != nil {
		t.Fatal(err)
	}
}
