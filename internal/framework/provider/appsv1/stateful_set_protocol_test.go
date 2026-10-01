// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestStatefulSetVolumeClaimTemplateReplacementProtocol(t *testing.T) {
	const before = `{"metadata":[{"name":"data","namespace":"default"}],"spec":[{"access_modes":["ReadWriteOnce"],"resources":[{"requests":{"storage":"1Gi"}}],"storage_class_name":"standard"}]}`
	for name, tc := range map[string]struct {
		before, after string
		replace       bool
	}{
		"unchanged":                   {before, before, false},
		"added":                       {"", before, true},
		"removed":                     {before, "", true},
		"request":                     {before, `{"metadata":[{"name":"data","namespace":"default"}],"spec":[{"access_modes":["ReadWriteOnce"],"resources":[{"requests":{"storage":"2Gi"}}],"storage_class_name":"standard"}]}`, true},
		"access_modes":                {before, `{"metadata":[{"name":"data","namespace":"default"}],"spec":[{"access_modes":["ReadWriteMany"],"resources":[{"requests":{"storage":"1Gi"}}],"storage_class_name":"standard"}]}`, true},
		"labels":                      {before, `{"metadata":[{"name":"data","namespace":"default","labels":{"edited":"true"}}],"spec":[{"access_modes":["ReadWriteOnce"],"resources":[{"requests":{"storage":"1Gi"}}],"storage_class_name":"standard"}]}`, true},
		"computed_metadata":           {`{"metadata":[{"name":"data","namespace":"default","generation":1,"resource_version":"42","uid":"claim-template"}],"spec":[{"access_modes":["ReadWriteOnce"],"resources":[{"requests":{"storage":"1Gi"}}],"storage_class_name":"standard"}]}`, before, false},
		"legacy_empty_metadata":       {`{"metadata":[{"name":"data","namespace":"default","annotations":{},"labels":{},"generation":0,"resource_version":"","uid":""}],"spec":[{"access_modes":["ReadWriteOnce"],"resources":[{"requests":{"storage":"1Gi"}}],"storage_class_name":"standard"}]}`, before, false},
		"legacy_empty_generated_name": {`{"metadata":[{"name":"data","namespace":"default","generate_name":""}],"spec":[{"access_modes":["ReadWriteOnce"],"resources":[{"requests":{"storage":"1Gi"}}],"storage_class_name":"standard"}]}`, before, false},
		"remove_nonempty_metadata":    {`{"metadata":[{"name":"data","namespace":"default","labels":{"managed":"true"}}],"spec":[{"access_modes":["ReadWriteOnce"],"resources":[{"requests":{"storage":"1Gi"}}],"storage_class_name":"standard"}]}`, before, true},
		"equivalent_quantity":         {before, `{"metadata":[{"name":"data","namespace":"default"}],"spec":[{"access_modes":["ReadWriteOnce"],"resources":[{"requests":{"storage":"1024Mi"}}],"storage_class_name":"standard"}]}`, false},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			server, err := testAccProviderFactories["kubernetes"]()
			if err != nil {
				t.Fatal(err)
			}
			schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
			if err != nil {
				t.Fatal(err)
			}
			typ := schemas.ResourceSchemas["kubernetes_stateful_set_v1"].ValueType()
			config := statefulSetProtocolValue(t, typ, fmt.Sprintf(`{"spec":[{"volume_claim_template":[%s]}]}`, tc.after))
			state := statefulSetProtocolValue(t, typ, fmt.Sprintf(`{"id":"default/example","spec":[{"volume_claim_template":[%s]}]}`, tc.before))
			resp, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
				TypeName: "kubernetes_stateful_set_v1", Config: &config, PriorState: &state, ProposedNewState: &config,
			})
			if err != nil {
				t.Fatal(err)
			}
			for _, d := range resp.Diagnostics {
				if d.Severity == tfprotov6.DiagnosticSeverityError {
					t.Fatalf("%s: %s", d.Summary, d.Detail)
				}
			}
			if got := len(resp.RequiresReplace) != 0; got != tc.replace {
				t.Fatalf("replacement = %t, want %t: %v", got, tc.replace, resp.RequiresReplace)
			}
		})
	}
}

func TestStatefulSetEmptyReplicasProtocolPlan(t *testing.T) {
	ctx := context.Background()
	server, err := testAccProviderFactories["kubernetes"]()
	if err != nil {
		t.Fatal(err)
	}
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	typ := schemas.ResourceSchemas["kubernetes_stateful_set_v1"].ValueType()
	config := statefulSetProtocolValue(t, typ, `{"spec":[{"replicas":""}]}`)
	state := statefulSetProtocolValue(t, typ, `{"id":"default/example","spec":[{"replicas":"3"}]}`)
	resp, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
		TypeName:         "kubernetes_stateful_set_v1",
		Config:           &config,
		PriorState:       &state,
		ProposedNewState: &config,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range resp.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("%s: %s", d.Summary, d.Detail)
		}
	}
	plan, err := resp.PlannedState.Unmarshal(typ)
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := tftypes.WalkAttributePath(plan, tftypes.NewAttributePath().WithAttributeName("spec").WithElementKeyInt(0).WithAttributeName("replicas"))
	if err != nil {
		t.Fatal(err)
	}
	if !got.(tftypes.Value).Equal(tftypes.NewValue(tftypes.String, "3")) {
		t.Fatalf("empty replicas plan = %s, want prior replica count 3", got)
	}
	if len(resp.RequiresReplace) != 0 {
		t.Fatalf("ignoring replicas unexpectedly requires replacement: %v", resp.RequiresReplace)
	}
}

func statefulSetProtocolValue(t *testing.T, typ tftypes.Type, raw string) tfprotov6.DynamicValue {
	t.Helper()
	value, err := (&tfprotov6.RawState{JSON: []byte(raw)}).Unmarshal(typ)
	if err != nil {
		t.Fatal(err)
	}
	dynamic, err := tfprotov6.NewDynamicValue(typ, value)
	if err != nil {
		t.Fatal(err)
	}
	return dynamic
}

func TestStatefulSetTemplateSpecRequiredProtocol(t *testing.T) {
	ctx := context.Background()
	server, err := testAccProviderFactories["kubernetes"]()
	if err != nil {
		t.Fatal(err)
	}
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	typ := schemas.ResourceSchemas["kubernetes_stateful_set_v1"].ValueType()
	config := statefulSetProtocolValue(t, typ, `{
		  "metadata":[{"name":"required-template"}],
		  "spec":[{
		    "service_name":"headless",
		    "selector":[{"match_labels":{"app":"required-template"}}],
		    "template":[{"metadata":[{"labels":{"app":"required-template"}}],"spec":[]}]
		  }]
		}`)
	resp, err := server.ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{
		TypeName: "kubernetes_stateful_set_v1", Config: &config,
	})
	if err != nil {
		t.Fatal(err)
	}
	at := tftypes.NewAttributePath().WithAttributeName("spec").WithElementKeyInt(0).WithAttributeName("template").WithElementKeyInt(0).WithAttributeName("spec")
	for _, d := range resp.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError && d.Attribute.Equal(at) {
			return
		}
	}
	t.Fatalf("missing required template.spec accepted: %v", resp.Diagnostics)
}

func TestStatefulSetSchemaV0UpgradeProtocol(t *testing.T) {
	ctx := context.Background()
	server, err := testAccProviderFactories["kubernetes"]()
	if err != nil {
		t.Fatal(err)
	}
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := server.UpgradeResourceState(ctx, &tfprotov6.UpgradeResourceStateRequest{
		TypeName: "kubernetes_stateful_set_v1", Version: 0,
		RawState: &tfprotov6.RawState{JSON: []byte(`{
		  "id":"default/historical",
		  "metadata":[{"name":"historical","namespace":"default","uid":"original-uid","generation":7}],
		  "spec":[{
		    "replicas":"0","service_name":"headless",
		    "template":[{"metadata":[{"labels":{"app":"historical"}}],"spec":[{
		      "container":[{"name":"main","image":"busybox:1.36","resources":[{"requests":[{"cpu":"100m","memory":"32Mi"}],"limits":[{"cpu":"200m"}]}]}],
		      "init_container":[{"name":"init","image":"busybox:1.36","resources":[{"requests":[{"memory":"16Mi"}],"limits":[]}]}]
		    }]}],
		    "volume_claim_template":[{"metadata":[{"name":"data","namespace":"default"}],"spec":[{"access_modes":["ReadWriteOnce"],"resources":[{"requests":{"storage":"1Gi"}}]}]}]
		  }]
		}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range resp.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("%s: %s", d.Summary, d.Detail)
		}
	}
	state, err := resp.UpgradedState.Unmarshal(schemas.ResourceSchemas["kubernetes_stateful_set_v1"].ValueType())
	if err != nil {
		t.Fatal(err)
	}
	spec := tftypes.NewAttributePath().WithAttributeName("spec").WithElementKeyInt(0)
	pod := spec.WithAttributeName("template").WithElementKeyInt(0).WithAttributeName("spec").WithElementKeyInt(0)
	for _, tc := range []struct {
		at   *tftypes.AttributePath
		want tftypes.Value
	}{
		{tftypes.NewAttributePath().WithAttributeName("id"), tftypes.NewValue(tftypes.String, "default/historical")},
		{tftypes.NewAttributePath().WithAttributeName("metadata").WithElementKeyInt(0).WithAttributeName("uid"), tftypes.NewValue(tftypes.String, "original-uid")},
		{spec.WithAttributeName("replicas"), tftypes.NewValue(tftypes.String, "0")},
		{pod.WithAttributeName("container").WithElementKeyInt(0).WithAttributeName("resources").WithElementKeyInt(0).WithAttributeName("requests").WithElementKeyString("cpu"), tftypes.NewValue(tftypes.String, "100m")},
		{pod.WithAttributeName("init_container").WithElementKeyInt(0).WithAttributeName("resources").WithElementKeyInt(0).WithAttributeName("requests").WithElementKeyString("memory"), tftypes.NewValue(tftypes.String, "16Mi")},
		{spec.WithAttributeName("volume_claim_template").WithElementKeyInt(0).WithAttributeName("spec").WithElementKeyInt(0).WithAttributeName("resources").WithElementKeyInt(0).WithAttributeName("requests").WithElementKeyString("storage"), tftypes.NewValue(tftypes.String, "1Gi")},
	} {
		got, _, err := tftypes.WalkAttributePath(state, tc.at)
		if err != nil {
			t.Fatal(err)
		}
		if !got.(tftypes.Value).Equal(tc.want) {
			t.Errorf("%s = %s, want %s", tc.at, got, tc.want)
		}
	}
}
