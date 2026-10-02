// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rpcperf

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

// podTemplateResources are the six resources whose schemas embed the PodSpec.
var podTemplateResources = []string{
	"kubernetes_deployment_v1",
	"kubernetes_daemon_set_v1",
	"kubernetes_stateful_set_v1",
	"kubernetes_pod_v1",
	"kubernetes_job_v1",
	"kubernetes_cron_job_v1",
}

var fake *fakeAPI
var fakeServer *httptest.Server

func TestMain(m *testing.M) {
	// Never pick up a real cluster: the provider talks only to the fake API.
	os.Setenv("KUBECONFIG", os.DevNull)
	os.Unsetenv("KUBE_CONFIG_PATH")
	os.Unsetenv("KUBE_CONFIG_PATHS")
	os.Unsetenv("KUBE_CTX")
	log.SetOutput(io.Discard)
	fake = newFakeAPI()
	fakeServer = httptest.NewServer(fake)
	code := m.Run()
	fakeServer.Close()
	os.Exit(code)
}

type providerServer struct {
	srv     tfprotov6.ProviderServer
	schemas map[string]*tfprotov6.Schema
}

// sharedServer is one configured production mux server for the whole package,
// as Terraform runs one provider process per plan.
var sharedServer = sync.OnceValues(func() (*providerServer, error) {
	ctx := context.Background()
	srv, err := mux.MuxServerWithProvider(ctx, "dev", kubernetes.Provider())
	if err != nil {
		return nil, err
	}
	sr, err := srv.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		return nil, err
	}
	if d := diagsErr(sr.Diagnostics); d != "" {
		return nil, fmt.Errorf("GetProviderSchema: %s", d)
	}
	ptyp := sr.Provider.ValueType()
	pjson, err := json.Marshal(fillEmptyBlocks(map[string]any{"host": fakeServer.URL}, sr.Provider.Block))
	if err != nil {
		return nil, err
	}
	// SDKv2 decodes only msgpack, so re-encode the JSON configuration.
	pcfg, err := (&tfprotov6.DynamicValue{JSON: pjson}).Unmarshal(ptyp)
	if err != nil {
		return nil, err
	}
	pdv, err := tfprotov6.NewDynamicValue(ptyp, pcfg)
	if err != nil {
		return nil, err
	}
	cr, err := srv.ConfigureProvider(ctx, &tfprotov6.ConfigureProviderRequest{TerraformVersion: "1.13.0", Config: &pdv})
	if err != nil {
		return nil, err
	}
	if d := diagsErr(cr.Diagnostics); d != "" {
		return nil, fmt.Errorf("ConfigureProvider: %s", d)
	}
	return &providerServer{srv: srv, schemas: sr.ResourceSchemas}, nil
})

// rpcEnv drives one resource through the protocol, as Terraform core would.
type rpcEnv struct {
	srv      tfprotov6.ProviderServer
	typeName string
	typ      tftypes.Type
	computed map[string]bool
	config   tftypes.Value
	state    *tfprotov6.DynamicValue
	private  []byte
}

func newEnv(t testing.TB, typeName string) *rpcEnv {
	t.Helper()
	ps, err := sharedServer()
	if err != nil {
		t.Fatal(err)
	}
	rs, ok := ps.schemas[typeName]
	if !ok {
		t.Fatalf("no schema for %s", typeName)
	}
	var raw any
	if err := json.Unmarshal([]byte(resourceConfigs[typeName]), &raw); err != nil {
		t.Fatalf("%s config: %v", typeName, err)
	}
	b, err := json.Marshal(fillEmptyBlocks(raw, rs.Block))
	if err != nil {
		t.Fatal(err)
	}
	typ := rs.ValueType()
	cfg, err := (&tfprotov6.DynamicValue{JSON: b}).Unmarshal(typ)
	if err != nil {
		t.Fatalf("%s config: %v", typeName, err)
	}
	computed := map[string]bool{}
	computedPaths("", rs.Block, computed)
	return &rpcEnv{srv: ps.srv, typeName: typeName, typ: typ, computed: computed, config: cfg}
}

// newCreatedEnv returns an env whose state is the result of a create followed
// by a refresh, the starting point of every no-op plan. It fails unless that
// plan is really a no-op.
func newCreatedEnv(t testing.TB, typeName string) *rpcEnv {
	t.Helper()
	e := newEnv(t, typeName)
	e.state, e.private = e.applyCreate(t, e.planCreate(t))
	rr := e.read(t)
	e.state, e.private = rr.NewState, rr.Private
	e.assertNoopPlan(t)
	return e
}

// assertNoopPlan fails when planning the unchanged configuration against the
// refreshed state proposes a change or a replacement: the benchmarks and
// allocation ceilings would then measure something else, and users would see
// a perpetual diff or a silent replacement.
func (e *rpcEnv) assertNoopPlan(t testing.TB) {
	t.Helper()
	r := e.planNoop(t)
	if len(r.RequiresReplace) != 0 {
		paths := make([]string, len(r.RequiresReplace))
		for i, p := range r.RequiresReplace {
			paths[i] = p.String()
		}
		t.Fatalf("%s: no-op plan requires replacement for %v", e.typeName, paths)
	}
	prior, err := e.state.Unmarshal(e.typ)
	if err != nil {
		t.Fatal(err)
	}
	planned, err := r.PlannedState.Unmarshal(e.typ)
	if err != nil {
		t.Fatal(err)
	}
	if planned.Equal(prior) {
		return
	}
	diffs, err := prior.Diff(planned)
	if err != nil {
		t.Fatalf("%s: no-op plan differs from the prior state: %v", e.typeName, err)
	}
	var changes []string
	for _, d := range diffs {
		// Diff reports every ancestor of a changed value; keep the leaves.
		if d.Value1 != nil && d.Value2 != nil && !d.Value1.Type().Is(tftypes.Object{}) &&
			!d.Value1.Type().Is(tftypes.List{}) && !d.Value1.Type().Is(tftypes.Set{}) && !d.Value1.Type().Is(tftypes.Map{}) {
			changes = append(changes, fmt.Sprintf("%s: %s -> %s", d.Path, d.Value1, d.Value2))
		}
		if len(changes) == 10 {
			break
		}
	}
	t.Fatalf("%s: no-op plan differs from the prior state in %d places, including %v", e.typeName, len(diffs), changes)
}

func (e *rpcEnv) dv(t testing.TB, v tftypes.Value) *tfprotov6.DynamicValue {
	d, err := tfprotov6.NewDynamicValue(e.typ, v)
	if err != nil {
		t.Fatal(err)
	}
	return &d
}

func (e *rpcEnv) validate(t testing.TB) {
	r, err := e.srv.ValidateResourceConfig(context.Background(), &tfprotov6.ValidateResourceConfigRequest{
		TypeName: e.typeName, Config: e.dv(t, e.config),
	})
	if err != nil || diagsErr(r.Diagnostics) != "" {
		t.Fatalf("validate %s: %v %s", e.typeName, err, diagsErr(r.Diagnostics))
	}
}

func (e *rpcEnv) planCreate(t testing.TB) *tfprotov6.PlanResourceChangeResponse {
	null := tftypes.NewValue(e.typ, nil)
	r, err := e.srv.PlanResourceChange(context.Background(), &tfprotov6.PlanResourceChangeRequest{
		TypeName: e.typeName, PriorState: e.dv(t, null), ProposedNewState: e.dv(t, e.config), Config: e.dv(t, e.config),
	})
	if err != nil || diagsErr(r.Diagnostics) != "" {
		t.Fatalf("plan create %s: %v %s", e.typeName, err, diagsErr(r.Diagnostics))
	}
	return r
}

func (e *rpcEnv) applyCreate(t testing.TB, plan *tfprotov6.PlanResourceChangeResponse) (*tfprotov6.DynamicValue, []byte) {
	null := tftypes.NewValue(e.typ, nil)
	r, err := e.srv.ApplyResourceChange(context.Background(), &tfprotov6.ApplyResourceChangeRequest{
		TypeName: e.typeName, PriorState: e.dv(t, null), PlannedState: plan.PlannedState,
		Config: e.dv(t, e.config), PlannedPrivate: plan.PlannedPrivate,
	})
	if err != nil || diagsErr(r.Diagnostics) != "" {
		t.Fatalf("apply create %s: %v %s", e.typeName, err, diagsErr(r.Diagnostics))
	}
	return r.NewState, r.Private
}

func (e *rpcEnv) read(t testing.TB) *tfprotov6.ReadResourceResponse {
	r, err := e.srv.ReadResource(context.Background(), &tfprotov6.ReadResourceRequest{
		TypeName: e.typeName, CurrentState: e.state, Private: e.private,
	})
	if err != nil || diagsErr(r.Diagnostics) != "" {
		t.Fatalf("read %s: %v %s", e.typeName, err, diagsErr(r.Diagnostics))
	}
	return r
}

// planNoop plans the unchanged configuration against the current state.
func (e *rpcEnv) planNoop(t testing.TB) *tfprotov6.PlanResourceChangeResponse {
	prior, err := e.state.Unmarshal(e.typ)
	if err != nil {
		t.Fatal(err)
	}
	r, err := e.srv.PlanResourceChange(context.Background(), &tfprotov6.PlanResourceChangeRequest{
		TypeName: e.typeName, PriorState: e.state, ProposedNewState: e.dv(t, proposedNewState(e.config, prior, e.computed)),
		Config: e.dv(t, e.config), PriorPrivate: e.private,
	})
	if err != nil || diagsErr(r.Diagnostics) != "" {
		t.Fatalf("plan no-op %s: %v %s", e.typeName, err, diagsErr(r.Diagnostics))
	}
	return r
}

// fillEmptyBlocks sets every absent list or set block to [], as Terraform
// sends unset blocks.
func fillEmptyBlocks(v any, b *tfprotov6.SchemaBlock) any {
	m, ok := v.(map[string]any)
	if !ok || b == nil {
		return v
	}
	for _, nb := range b.BlockTypes {
		cur, present := m[nb.TypeName]
		switch nb.Nesting {
		case tfprotov6.SchemaNestedBlockNestingModeList, tfprotov6.SchemaNestedBlockNestingModeSet:
			if !present || cur == nil {
				m[nb.TypeName] = []any{}
				continue
			}
			for _, elem := range cur.([]any) {
				fillEmptyBlocks(elem, nb.Block)
			}
		default:
			if present && cur != nil {
				fillEmptyBlocks(cur, nb.Block)
			}
		}
	}
	return m
}

func computedPaths(prefix string, b *tfprotov6.SchemaBlock, out map[string]bool) {
	for _, a := range b.Attributes {
		if a.Computed {
			out[prefix+a.Name] = true
		}
	}
	for _, nb := range b.BlockTypes {
		computedPaths(prefix+nb.TypeName+".", nb.Block, out)
	}
}

// attributeKey joins the attribute names of p, dropping element keys, and
// reports whether p ends in an attribute name.
func attributeKey(p *tftypes.AttributePath) (string, bool) {
	var names []string
	steps := p.Steps()
	last := false
	for i, s := range steps {
		if n, ok := s.(tftypes.AttributeName); ok {
			names = append(names, string(n))
			last = i == len(steps)-1
		}
	}
	return strings.Join(names, "."), last
}

// proposedNewState approximates Terraform core's ProposedNewState: the
// configuration, with null computed attributes taken from the prior state.
func proposedNewState(config, prior tftypes.Value, computed map[string]bool) tftypes.Value {
	if prior.IsNull() {
		return config
	}
	out, err := tftypes.Transform(config, func(p *tftypes.AttributePath, v tftypes.Value) (tftypes.Value, error) {
		if !v.IsNull() {
			return v, nil
		}
		k, last := attributeKey(p)
		if !last || !computed[k] {
			return v, nil
		}
		pv, _, err := tftypes.WalkAttributePath(prior, p)
		if err != nil {
			return v, nil
		}
		if val, ok := pv.(tftypes.Value); ok {
			return val, nil
		}
		return v, nil
	})
	if err != nil {
		panic(err)
	}
	return out
}

// toJSONAny converts v to plain Go values for stable JSON comparison.
func toJSONAny(v tftypes.Value) any {
	if !v.IsKnown() {
		return "<unknown>"
	}
	if v.IsNull() {
		return nil
	}
	switch {
	case v.Type().Is(tftypes.String):
		var s string
		_ = v.As(&s)
		return s
	case v.Type().Is(tftypes.Bool):
		var b bool
		_ = v.As(&b)
		return b
	case v.Type().Is(tftypes.Number):
		var n big.Float
		_ = v.As(&n)
		return n.Text('g', -1)
	case v.Type().Is(tftypes.List{}), v.Type().Is(tftypes.Set{}), v.Type().Is(tftypes.Tuple{}):
		var es []tftypes.Value
		_ = v.As(&es)
		out := make([]any, len(es))
		for i, e := range es {
			out[i] = toJSONAny(e)
		}
		return out
	default:
		var m map[string]tftypes.Value
		_ = v.As(&m)
		out := map[string]any{}
		for k, e := range m {
			out[k] = toJSONAny(e)
		}
		return out
	}
}

func diagsErr(ds []*tfprotov6.Diagnostic) string {
	var s []string
	for _, d := range ds {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			s = append(s, d.Summary+": "+d.Detail)
		}
	}
	return strings.Join(s, " | ")
}

// podSpecConfig is a representative pod spec: two containers with ports, env,
// resources, probes and a volume mount, and one config map volume.
const podSpecConfig = `"spec":[{"container":[
  {"name":"app","image":"registry.k8s.io/pause:3.10",
   "port":[{"container_port":80,"name":"http"},{"container_port":9090,"name":"metrics"}],
   "env":[{"name":"A","value":"1"},{"name":"B","value":"2"},{"name":"C","value":"3"}],
   "resources":[{"limits":{"cpu":"500m","memory":"256Mi"},"requests":{"cpu":"100m","memory":"64Mi"}}],
   "liveness_probe":[{"http_get":[{"path":"/healthz","port":"80"}],"initial_delay_seconds":5}],
   "readiness_probe":[{"http_get":[{"path":"/ready","port":"80"}]}],
   "volume_mount":[{"name":"cfg","mount_path":"/etc/cfg"}]},
  {"name":"sidecar","image":"busybox:1.36","command":["sh","-c","sleep 3600"]}],
 "volume":[{"name":"cfg","config_map":[{"name":"web-cfg"}]}]}]`

var batchPodSpecConfig = strings.Replace(podSpecConfig, `"spec":[{`, `"spec":[{"restart_policy":"Never",`, 1)

// resourceConfigs are the configurations, in Terraform JSON value form, used
// for each resource. Absent blocks are filled with [] by fillEmptyBlocks.
var resourceConfigs = map[string]string{
	"kubernetes_deployment_v1": `{"metadata":[{"name":"web","namespace":"default","labels":{"app":"web","tier":"fe"},"annotations":{"team":"a"}}],
	  "wait_for_rollout":false,
	  "spec":[{"replicas":"2","selector":[{"match_labels":{"app":"web"}}],
	    "template":[{"metadata":[{"labels":{"app":"web","tier":"fe"}}],` + podSpecConfig + `}]}]}`,
	"kubernetes_daemon_set_v1": `{"metadata":[{"name":"ds","namespace":"default"}],"wait_for_rollout":false,
	  "spec":[{"selector":[{"match_labels":{"app":"ds"}}],"template":[{"metadata":[{"labels":{"app":"ds"}}],` + podSpecConfig + `}]}]}`,
	"kubernetes_stateful_set_v1": `{"metadata":[{"name":"ss","namespace":"default"}],"wait_for_rollout":false,
	  "spec":[{"service_name":"ss","replicas":"1","selector":[{"match_labels":{"app":"ss"}}],
	    "template":[{"metadata":[{"labels":{"app":"ss"}}],` + podSpecConfig + `}]}]}`,
	"kubernetes_pod_v1": `{"metadata":[{"name":"pod","namespace":"default"}],` + podSpecConfig + `}`,
	"kubernetes_job_v1": `{"metadata":[{"name":"job","namespace":"default"}],"wait_for_completion":false,
	  "spec":[{"template":[{"metadata":[{"labels":{"app":"job"}}],` + batchPodSpecConfig + `}]}]}`,
	"kubernetes_cron_job_v1": `{"metadata":[{"name":"cj","namespace":"default"}],
	  "spec":[{"schedule":"*/5 * * * *","job_template":[{"metadata":[{}],
	    "spec":[{"template":[{"metadata":[{}],` + batchPodSpecConfig + `}]}]}]}]}`,
}
