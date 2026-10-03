// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

type writeResultModel struct {
	ID   types.String      `tfsdk:"id"`
	Tags types.Set         `tfsdk:"tags"`
	Spec []writeResultSpec `tfsdk:"spec"`
}

type writeResultSpec struct {
	NodeSelector types.Map              `tfsdk:"node_selector"`
	Container    []writeResultContainer `tfsdk:"container"`
	Toleration   []writeResultEntry     `tfsdk:"toleration"`
}

type writeResultContainer struct {
	Name            types.String       `tfsdk:"name"`
	Image           types.String       `tfsdk:"image"`
	ImagePullPolicy types.String       `tfsdk:"image_pull_policy"`
	Env             []writeResultEntry `tfsdk:"env"`
	Port            []writeResultEntry `tfsdk:"port"`
}

// writeResultEntry stands for env, port and toleration: Computed is set by
// Kubernetes when the configuration leaves it out.
type writeResultEntry struct {
	Name     types.String `tfsdk:"name"`
	Value    types.String `tfsdk:"value"`
	Computed types.String `tfsdk:"computed"`
}

func TestSetWriteResult(t *testing.T) {
	entry := schema.ListNestedBlock{NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
		"name":     schema.StringAttribute{Optional: true},
		"value":    schema.StringAttribute{Optional: true},
		"computed": schema.StringAttribute{Optional: true, Computed: true},
	}}}
	s := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":   schema.StringAttribute{Computed: true},
			"tags": schema.SetAttribute{ElementType: types.StringType, Optional: true, Computed: true},
		},
		Blocks: map[string]schema.Block{"spec": schema.ListNestedBlock{NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{"node_selector": schema.MapAttribute{ElementType: types.StringType, Optional: true}},
			Blocks: map[string]schema.Block{
				"container": schema.ListNestedBlock{NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name":              schema.StringAttribute{Required: true},
						"image":             schema.StringAttribute{Optional: true},
						"image_pull_policy": schema.StringAttribute{Optional: true, Computed: true},
					},
					Blocks: map[string]schema.Block{"env": entry, "port": entry},
				}},
				"toleration": entry,
			},
		}}},
	}
	str, unknown := types.StringValue, types.StringUnknown()
	tags := func(values ...string) types.Set {
		set, _ := types.SetValueFrom(context.Background(), types.StringType, values)
		return set
	}
	env := func(name string) writeResultEntry {
		return writeResultEntry{Name: str(name), Value: str("v"), Computed: str("")}
	}
	app := func(image string, pull types.String, envs ...writeResultEntry) writeResultContainer {
		return writeResultContainer{Name: str("app"), Image: str(image), ImagePullPolicy: pull, Env: envs}
	}
	sidecar := writeResultContainer{Name: str("proxy"), Image: str("proxy"), ImagePullPolicy: str("Always")}
	model := func(id types.String, tags types.Set, nodeSelector types.Map, containers []writeResultContainer, tolerations ...writeResultEntry) writeResultModel {
		return writeResultModel{ID: id, Tags: tags, Spec: []writeResultSpec{{NodeSelector: nodeSelector, Container: containers, Toleration: tolerations}}}
	}
	noSelector := types.MapNull(types.StringType)
	selector := types.MapValueMust(types.StringType, map[string]attr.Value{"injected": str("true")})

	for _, tc := range []struct {
		name         string
		plan, actual writeResultModel
		want         writeResultModel
		wantError    bool
	}{
		{
			name: "additions keep the plan and fill unknowns by name",
			plan: model(unknown, tags("a"), noSelector, []writeResultContainer{app("app:1", unknown, env("A"))}),
			actual: model(str("ns/p"), tags("a"), selector,
				[]writeResultContainer{sidecar, app("app:2", str("IfNotPresent"), env("A"), env("B"))},
				writeResultEntry{Name: str(""), Value: str("injected"), Computed: str("Exists")}),
			want: model(str("ns/p"), tags("a"), noSelector, []writeResultContainer{app("app:1", str("IfNotPresent"), env("A"))}),
		},
		{
			name: "unnamed elements match in order, unnamed ones first",
			plan: model(str("ns/p"), tags(), noSelector, []writeResultContainer{{
				Name: str("app"), Image: str("app"), ImagePullPolicy: str("Always"),
				Port: []writeResultEntry{{Name: str(""), Value: str("80"), Computed: unknown}},
			}}),
			actual: model(str("ns/p"), tags(), noSelector, []writeResultContainer{{
				Name: str("app"), Image: str("app"), ImagePullPolicy: str("Always"),
				Port: []writeResultEntry{{Name: str("metrics"), Value: str("9090"), Computed: str("UDP")}, {Name: str(""), Value: str("80"), Computed: str("TCP")}},
			}}),
			want: model(str("ns/p"), tags(), noSelector, []writeResultContainer{{
				Name: str("app"), Image: str("app"), ImagePullPolicy: str("Always"),
				Port: []writeResultEntry{{Name: str(""), Value: str("80"), Computed: str("TCP")}},
			}}),
		},
		{
			name:   "a computed value Kubernetes did not return is null",
			plan:   model(str("ns/p"), tags(), noSelector, []writeResultContainer{app("app", unknown)}),
			actual: model(str("ns/p"), tags(), noSelector, []writeResultContainer{sidecar}),
			want:   model(str("ns/p"), tags(), noSelector, []writeResultContainer{app("app", types.StringNull())}),
		},
		{
			name:      "any other value Kubernetes did not return is an error",
			plan:      model(str("ns/p"), tags(), noSelector, []writeResultContainer{{Name: str("app"), Image: unknown, ImagePullPolicy: str("Always")}}),
			actual:    model(str("ns/p"), tags(), noSelector, []writeResultContainer{sidecar}),
			wantError: true,
		},
		{
			name:   "a set element is filled from the only element left",
			plan:   model(str("ns/p"), types.SetValueMust(types.StringType, []attr.Value{str("a"), unknown}), noSelector, nil),
			actual: model(str("ns/p"), tags("a", "b"), noSelector, nil),
			want:   model(str("ns/p"), tags("a", "b"), noSelector, nil),
		},
		{
			name:      "an ambiguous set element is an error",
			plan:      model(str("ns/p"), types.SetValueMust(types.StringType, []attr.Value{str("a"), unknown}), noSelector, nil),
			actual:    model(str("ns/p"), tags("a", "b", "c"), noSelector, nil),
			wantError: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			plan := tfsdk.Plan{Schema: s}
			diags := plan.Set(ctx, &tc.plan)
			state := tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
			diags.Append(SetWriteResult(ctx, &state, plan, func(actual *tfsdk.State) diag.Diagnostics {
				return actual.Set(ctx, &tc.actual)
			})...)
			if tc.wantError {
				if !diags.HasError() {
					t.Fatal("expected an error")
				}
				return
			}
			want := tfsdk.State{Schema: s}
			diags.Append(want.Set(ctx, &tc.want)...)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if !state.Raw.Equal(want.Raw) {
				t.Fatalf("got %s\nwant %s", state.Raw, want.Raw)
			}
		})
	}
}
