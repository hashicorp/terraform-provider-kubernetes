// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

// Package rpcperf holds protocol-level performance tests for the Framework
// pod-template resources (Deployment, DaemonSet, StatefulSet, Pod, Job and
// CronJob). The tests drive the production mux server against an in-process
// fake Kubernetes API, so they need no cluster and no Terraform binary.
//
// The package has no production code; everything lives in its test files:
//
//   - TestPodTemplateRPCAllocCeilings fails when one of the six resources
//     serves a schema that is not frozen, or when its no-op plan or read
//     allocates more than its ceiling. It is part of make test, which runs
//     ./internal/framework/provider/...; the Unit Tests workflow runs it on
//     pushes to main and, once its pull_request path filter includes
//     internal/framework, on pull requests.
//
// Tests and benchmarks that start from an existing resource create and refresh
// it first, and fail unless planning the unchanged configuration then is a
// no-op without replacement.
//   - BenchmarkRPC{PlanNoop,Read,ApplyCreate,Validate}/<resource> report
//     ns/op, B/op and allocs/op per RPC.
//   - TestRPCRefreshPlan50 (RPCPERF_SCALE=1) runs 50 sequential Read + no-op
//     Plan calls for one Deployment and reports the totals.
//   - TestRPCDumpProviderSchema (RPCPERF_SCHEMA_OUT=file) and TestRPCDumpValues
//     (RPCPERF_VALUES_OUT=file) write the GetProviderSchema response and the
//     RPC values of each resource as JSON, for before/after comparisons.
package rpcperf
