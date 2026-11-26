// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/batchv1"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	jsonpatch "gopkg.in/evanphx/json-patch.v4"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/dynamic"
	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
)

func TestSetDaemonSetStateMatchesReflection(t *testing.T) {
	dsType := daemonSetSpecListType().ElemType.(types.ObjectType)
	dsPodSpec := listAttrType(dsType, "template", "spec")
	strategy := dsType.AttrTypes["strategy"].(types.ObjectType)
	ds := func(edit func(*DaemonSetV1SpecModel)) []DaemonSetV1SpecModel {
		spec := DaemonSetV1SpecModel{
			Strategy: types.ObjectNull(strategy.AttrTypes),
			Template: []workloadTemplateModel{{Spec: types.ListNull(dsPodSpec.ElemType)}},
		}
		edit(&spec)
		return []DaemonSetV1SpecModel{spec}
	}
	assertStateMatchesReflection(t, daemonSetFrozenSchema, setDaemonSetState, func(spec []DaemonSetV1SpecModel, t timeouts.Value) DaemonSetV1Model {
		return DaemonSetV1Model{ID: types.StringValue("ns/name"), Spec: spec, Timeouts: t}
	}, map[string][]DaemonSetV1SpecModel{
		"nil spec":               nil,
		"empty spec":             {},
		"null objects and lists": ds(func(*DaemonSetV1SpecModel) {}),
		"unknown objects and lists": ds(func(s *DaemonSetV1SpecModel) {
			s.Strategy = types.ObjectUnknown(strategy.AttrTypes)
			s.Template[0].Spec = types.ListUnknown(dsPodSpec.ElemType)
			s.MinReadySeconds = types.Int64Unknown()
		}),
		"empty objects and slices": ds(func(s *DaemonSetV1SpecModel) {
			s.Strategy = types.ObjectValueMust(strategy.AttrTypes, map[string]attr.Value{"type": types.StringNull(), "rolling_update": types.ObjectNull(daemonSetRollingUpdateObjectType().AttrTypes)})
			s.Selector = []DaemonSetLabelSelectorModel{}
			s.Template[0].Metadata = []common.NamespacedMetadataModel{}
		}),
		"nil template":   ds(func(s *DaemonSetV1SpecModel) { s.Template = nil }),
		"empty template": ds(func(s *DaemonSetV1SpecModel) { s.Template = []workloadTemplateModel{} }),
	})
}

func TestSetStatefulSetStateMatchesReflection(t *testing.T) {
	stsType := statefulSetSpecListType().ElemType.(types.ObjectType)
	stsPodSpec := listAttrType(stsType, "template", "spec")
	retention := stsType.AttrTypes["persistent_volume_claim_retention_policy"].(types.ObjectType)
	sts := func(edit func(*StatefulSetSpecModel)) []StatefulSetSpecModel {
		spec := StatefulSetSpecModel{
			PersistentVolumeClaimRetentionPolicy: types.ObjectNull(retention.AttrTypes),
			Ordinals:                             types.ObjectNull(stsType.AttrTypes["ordinals"].(types.ObjectType).AttrTypes),
			Template:                             []workloadTemplateModel{{Spec: types.ListNull(stsPodSpec.ElemType)}},
		}
		edit(&spec)
		return []StatefulSetSpecModel{spec}
	}
	assertStateMatchesReflection(t, statefulSetFrozenSchema, setStatefulSetState, func(spec []StatefulSetSpecModel, t timeouts.Value) StatefulSetV1Model {
		return StatefulSetV1Model{ID: types.StringValue("ns/name"), Spec: spec, Timeouts: t}
	}, map[string][]StatefulSetSpecModel{
		"nil spec":               nil,
		"null objects and lists": sts(func(*StatefulSetSpecModel) {}),
		"unknown objects and lists": sts(func(s *StatefulSetSpecModel) {
			s.Ordinals = types.ObjectUnknown(stsType.AttrTypes["ordinals"].(types.ObjectType).AttrTypes)
			s.PersistentVolumeClaimRetentionPolicy = types.ObjectUnknown(retention.AttrTypes)
			s.Template[0].Spec = types.ListUnknown(stsPodSpec.ElemType)
			s.Replicas = types.StringUnknown()
		}),
		"empty objects and slices": sts(func(s *StatefulSetSpecModel) {
			s.PersistentVolumeClaimRetentionPolicy = types.ObjectValueMust(retention.AttrTypes, map[string]attr.Value{"when_deleted": types.StringNull(), "when_scaled": types.StringNull()})
			s.UpdateStrategy = []StatefulSetUpdateStrategyModel{}
			s.VolumeClaimTemplate = []PersistentVolumeClaimModel{}
			s.Template = []workloadTemplateModel{}
		}),
	})
}

// assertStateMatchesReflection sets each spec through set and through
// tfsdk.State.Set, and requires the same outcome and state from both.
func assertStateMatchesReflection[M, S any](t *testing.T, schemaFunc common.SchemaFunc, set func(context.Context, *tfsdk.State, M) diag.Diagnostics, model func(S, timeouts.Value) M, specs map[string]S) {
	t.Helper()
	ctx := context.Background()
	var resp resource.SchemaResponse
	schemaFunc(ctx, resource.SchemaRequest{}, &resp)
	nullState := func() *tfsdk.State {
		return &tfsdk.State{Schema: resp.Schema, Raw: tftypes.NewValue(resp.Schema.Type().TerraformType(ctx), nil)}
	}
	timeoutsType := resp.Schema.Blocks["timeouts"].Type().(timeouts.Type)
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			m := model(spec, timeouts.Value{Object: types.ObjectNull(timeoutsType.AttrTypes)})
			want, got := nullState(), nullState()
			wantDiags := want.Set(ctx, &m)
			gotDiags := set(ctx, got, m)
			if gotDiags.HasError() != wantDiags.HasError() {
				t.Fatalf("errors = %v, reflection errors = %v", gotDiags, wantDiags)
			}
			if !got.Raw.Equal(want.Raw) {
				t.Fatalf("state = %s\nreflection state = %s", got.Raw, want.Raw)
			}
		})
	}
}

// listAttrType walks nested list blocks of typ by attribute name.
func listAttrType(typ types.ObjectType, names ...string) types.ListType {
	var list types.ListType
	for _, name := range names {
		list = typ.AttrTypes[name].(types.ListType)
		typ, _ = list.ElemType.(types.ObjectType)
	}
	return list
}

func TestDeploymentSpecModelsMatchReflection(t *testing.T) {
	ctx := context.Background()
	specType := deploymentSpecListType().ElemType.(types.ObjectType)
	templateType := specType.AttrTypes["template"].(types.ListType).ElemType.(types.ObjectType)
	podSpec := listAttrType(specType, "template", "spec").ElemType
	object := func(typ types.ObjectType, edit map[string]attr.Value) attr.Value {
		attrs := map[string]attr.Value{}
		for name, t := range typ.AttrTypes {
			attrs[name], _ = t.ValueFromTerraform(ctx, tftypes.NewValue(t.TerraformType(ctx), nil))
		}
		for name, v := range edit {
			attrs[name] = v
		}
		return types.ObjectValueMust(typ.AttrTypes, attrs)
	}
	spec := func(edit map[string]attr.Value) types.List {
		return types.ListValueMust(specType, []attr.Value{object(specType, edit)})
	}
	template := func(pod types.List) types.List {
		return types.ListValueMust(templateType, []attr.Value{object(templateType, map[string]attr.Value{"spec": pod})})
	}

	for name, value := range map[string]types.List{
		"null":             types.ListNull(specType),
		"unknown":          types.ListUnknown(specType),
		"empty":            types.ListValueMust(specType, nil),
		"null blocks":      spec(nil),
		"empty blocks":     spec(map[string]attr.Value{"selector": types.ListValueMust(specType.AttrTypes["selector"].(types.ListType).ElemType, nil), "template": types.ListValueMust(templateType, nil)}),
		"unknown template": spec(map[string]attr.Value{"template": types.ListUnknown(templateType)}),
		"unknown pod spec": spec(map[string]attr.Value{"template": template(types.ListUnknown(podSpec)), "replicas": types.StringUnknown()}),
		"unknown element":  types.ListValueMust(specType, []attr.Value{types.ObjectUnknown(specType.AttrTypes)}),
	} {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := deploymentSpecModels(ctx, value)
			var want []deploymentSpecModel
			wantDiags := value.ElementsAs(ctx, &want, false)
			if gotDiags.HasError() != wantDiags.HasError() {
				t.Fatalf("errors = %v, reflection errors = %v", gotDiags, wantDiags)
			}
			if !gotDiags.HasError() && !reflect.DeepEqual(got, want) {
				t.Fatalf("models = %#v\nreflection models = %#v", got, want)
			}
		})
	}
}

func TestStatefulSetOrdinalsAndRevisionHistory(t *testing.T) {
	ctx := context.Background()
	typ := map[string]attr.Type{"start": types.Int64Type}
	for _, start := range []int64{0, 5} {
		prior := StatefulSetSpecModel{Ordinals: types.ObjectValueMust(typ, map[string]attr.Value{"start": types.Int64Value(start)}), RevisionHistoryLimit: types.Int64Value(0)}
		api, diags := expandStatefulSetSpec(ctx, prior, nil)
		if diags.HasError() || api.Ordinals == nil || int64(api.Ordinals.Start) != start || api.RevisionHistoryLimit == nil || *api.RevisionHistoryLimit != 0 {
			t.Fatalf("expand: %#v, %v", api, diags)
		}
		actual, diags := flattenStatefulSetSpec(ctx, *api, &prior, true)
		if diags.HasError() || !actual.Ordinals.Equal(prior.Ordinals) {
			t.Fatalf("flatten: %s, %v", actual.Ordinals, diags)
		}
	}
	old := StatefulSetSpecModel{Ordinals: types.ObjectNull(typ)}
	actual, diags := flattenStatefulSetSpec(ctx, appsv1.StatefulSetSpec{Ordinals: &appsv1.StatefulSetOrdinals{Start: 5}, RevisionHistoryLimit: ptr.To(int32(10))}, &old, true)
	if diags.HasError() || !actual.Ordinals.IsNull() {
		t.Fatalf("legacy omitted ordinals must stay null: %s, %v", actual.Ordinals, diags)
	}
	for _, value := range []string{"0%", "10%", "100%", "0", "1"} {
		if !daemonSetRollingValuePattern.MatchString(value) {
			t.Fatalf("valid rolling value rejected: %s", value)
		}
	}
}

type workloadUpdateClients struct {
	kubernetes.KubeClientsets
	client  *k8sclient.Clientset
	dynamic dynamic.Interface
}

func (c workloadUpdateClients) MainClientset() (*k8sclient.Clientset, error) { return c.client, nil }
func (c workloadUpdateClients) DynamicClient() (dynamic.Interface, error)    { return c.dynamic, nil }
func (workloadUpdateClients) GetIgnoreAnnotations() []string                 { return nil }
func (workloadUpdateClients) GetIgnoreLabels() []string                      { return nil }

func TestWorkloadUpdateAdoptsEmptyTCPHost(t *testing.T) {
	ctx := context.Background()
	for _, owner := range []struct {
		name, kind, apiVersion, spec, apiSpec string
		resource                              resource.Resource
	}{
		{"deployment", "Deployment", "apps/v1", `"replicas":"0",`, `"replicas":0,`, &DeploymentV1{}},
		{"daemonset", "DaemonSet", "apps/v1", ``, ``, &DaemonSetV1{}},
		{"statefulset", "StatefulSet", "apps/v1", `"replicas":"0","service_name":"qa",`, `"replicas":0,"serviceName":"qa",`, &StatefulSetV1{}},
		{"cronjob", "CronJob", "batch/v1", ``, ``, &batchv1.CronJobV1{}},
	} {
		for _, adopt := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/adopt=%t", owner.name, adopt), func(t *testing.T) {
				apiTemplate := `"template":{"metadata":{"labels":{"app":"qa"}},"spec":{"containers":[{"name":"injected","image":"sidecar"},{"name":"app","image":"busybox","livenessProbe":{"tcpSocket":{"port":80,"host":"old.example"}},"readinessProbe":{"tcpSocket":{"port":81,"host":"keep.example"}}}]}}`
				tfTemplate := `"template":[{"metadata":[{"labels":{"app":"qa"}}],"spec":[{"container":[{"name":"injected","image":"sidecar"},{"name":"app","image":"busybox","liveness_probe":[{"tcp_socket":[{"port":"80"}]}],"readiness_probe":[{"tcp_socket":[{"port":"81"}]}]}]}]}]`
				apiSpec := `{` + owner.apiSpec + `"selector":{"matchLabels":{"app":"qa"}},` + apiTemplate + `}`
				tfSpec := `{` + owner.spec + `"selector":[{"match_labels":{"app":"qa"}}],` + tfTemplate + `}`
				podPath := path.Root("spec").AtListIndex(0).AtName("template").AtListIndex(0).AtName("spec")
				wait := `,"wait_for_rollout":false`
				if owner.name == "cronjob" {
					apiSpec = `{"schedule":"0 * * * *","jobTemplate":{"spec":{` + apiTemplate + `}}}`
					tfSpec = `{"schedule":"0 * * * *","job_template":[{"spec":[{` + tfTemplate + `}]}]}`
					podPath = path.Root("spec").AtListIndex(0).AtName("job_template").AtListIndex(0).AtName("spec").AtListIndex(0).AtName("template").AtListIndex(0).AtName("spec")
					wait = ""
				}
				current := []byte(`{"kind":"` + owner.kind + `","apiVersion":"` + owner.apiVersion + `","metadata":{"name":"qa","namespace":"default","uid":"keep-uid","resourceVersion":"1"},"spec":` + apiSpec + `}`)
				writes := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					if r.Method == http.MethodPatch {
						data, err := io.ReadAll(r.Body)
						if err != nil {
							t.Error(err)
							w.WriteHeader(http.StatusBadRequest)
							return
						}
						patch, err := jsonpatch.DecodePatch(data)
						if err != nil {
							t.Error(err)
							w.WriteHeader(http.StatusBadRequest)
							return
						}
						current, err = patch.Apply(current)
						if err != nil {
							t.Error(err)
							w.WriteHeader(http.StatusBadRequest)
							return
						}
						writes++
					} else if r.Method != http.MethodGet {
						t.Errorf("unexpected method %s", r.Method)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					_, _ = w.Write(current)
				}))
				defer server.Close()
				config := &rest.Config{Host: server.URL, ContentConfig: rest.ContentConfig{ContentType: "application/json"}}
				client, err := k8sclient.NewForConfig(config)
				if err != nil {
					t.Fatal(err)
				}
				dynamicClient, err := dynamic.NewForConfig(config)
				if err != nil {
					t.Fatal(err)
				}
				var configured resource.ConfigureResponse
				owner.resource.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: func() any { return workloadUpdateClients{client: client, dynamic: dynamicClient} }}, &configured)
				if configured.Diagnostics.HasError() {
					t.Fatal(configured.Diagnostics)
				}
				var schemaResponse resource.SchemaResponse
				owner.resource.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
				raw := tfprotov6.RawState{JSON: []byte(`{"id":"default/qa","metadata":[{"name":"qa","namespace":"default"}],"spec":[` + tfSpec + `]` + wait + `}`)}
				value, err := raw.Unmarshal(schemaResponse.Schema.Type().TerraformType(ctx))
				if err != nil {
					t.Fatal(err)
				}
				state := tfsdk.State{Schema: schemaResponse.Schema, Raw: value}
				var identitySchema resource.IdentitySchemaResponse
				owner.resource.(resource.ResourceWithIdentity).IdentitySchema(ctx, resource.IdentitySchemaRequest{}, &identitySchema)
				identity := &tfsdk.ResourceIdentity{Schema: identitySchema.IdentitySchema}
				refreshed := resource.ReadResponse{State: state, Identity: identity}
				owner.resource.Read(ctx, resource.ReadRequest{State: state}, &refreshed)
				if refreshed.Diagnostics.HasError() {
					t.Fatal(refreshed.Diagnostics)
				}
				state = refreshed.State
				// Refresh follows live container order, so the app is the second container.
				hostPath := podPath.AtListIndex(0).AtName("container").AtListIndex(1).AtName("liveness_probe").AtListIndex(0).AtName("tcp_socket").AtListIndex(0).AtName("host")
				var priorHost types.String
				if d := state.GetAttribute(ctx, hostPath, &priorHost); d.HasError() || !priorHost.IsNull() {
					t.Fatalf("legacy host ownership changed: %s; %v", priorHost, d)
				}
				plan := tfsdk.Plan(state)
				if adopt {
					if d := plan.SetAttribute(ctx, hostPath, types.StringValue("")); d.HasError() {
						t.Fatal(d)
					}
				} else {
					// A metadata update must leave the unowned probe host alone.
					if d := plan.SetAttribute(ctx, path.Root("metadata").AtListIndex(0).AtName("labels"), types.MapValueMust(types.StringType, map[string]attr.Value{"changed": types.StringValue("yes")})); d.HasError() {
						t.Fatal(d)
					}
				}
				response := resource.UpdateResponse{State: state, Identity: identity}
				owner.resource.Update(ctx, resource.UpdateRequest{State: state, Plan: plan, Config: tfsdk.Config(plan)}, &response)
				if response.Diagnostics.HasError() {
					t.Fatal(response.Diagnostics)
				}
				var actual map[string]any
				if err := json.Unmarshal(current, &actual); err != nil {
					t.Fatal(err)
				}
				actualSpec := actual["spec"].(map[string]any)
				if owner.name == "cronjob" {
					actualSpec = actualSpec["jobTemplate"].(map[string]any)["spec"].(map[string]any)
				}
				containers := actualSpec["template"].(map[string]any)["spec"].(map[string]any)["containers"].([]any)
				var app map[string]any
				for _, c := range containers {
					if c.(map[string]any)["name"] == "app" {
						app = c.(map[string]any)
					}
				}
				actualHost, _ := app["livenessProbe"].(map[string]any)["tcpSocket"].(map[string]any)["host"].(string)
				wantHost, wantWrites := "old.example", 1
				if adopt {
					wantHost, wantWrites = "", 1
				}
				if actualHost != wantHost || writes != wantWrites {
					t.Fatalf("live host=%q writes=%d, want %q/%d", actualHost, writes, wantHost, wantWrites)
				}
				if len(containers) != 2 || app["readinessProbe"].(map[string]any)["tcpSocket"].(map[string]any)["host"] != "keep.example" {
					t.Fatal("unrelated container or probe changed")
				}
				again := resource.ReadResponse{State: response.State, Identity: identity}
				owner.resource.Read(ctx, resource.ReadRequest{State: response.State}, &again)
				if again.Diagnostics.HasError() {
					t.Fatal(again.Diagnostics)
				}
				var host, uid types.String
				if d := again.State.GetAttribute(ctx, hostPath, &host); d.HasError() {
					t.Fatal(d)
				}
				if adopt && host != types.StringValue("") || !adopt && !host.IsNull() {
					t.Fatalf("host did not converge: %s", host)
				}
				if d := again.State.GetAttribute(ctx, path.Root("metadata").AtListIndex(0).AtName("uid"), &uid); d.HasError() || uid.ValueString() != "keep-uid" {
					t.Fatalf("identity changed: %s; %v", uid, d)
				}
			})
		}
	}
}
