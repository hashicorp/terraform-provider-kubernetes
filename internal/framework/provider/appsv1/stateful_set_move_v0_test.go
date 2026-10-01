// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestStatefulSetSchemaV0AliasMoveProtocol(t *testing.T) {
	ctx := context.Background()
	server, err := testAccProviderFactories["kubernetes"]()
	if err != nil {
		t.Fatal(err)
	}
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := server.MoveResourceState(ctx, &tfprotov6.MoveResourceStateRequest{
		SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes",
		SourceTypeName:        "kubernetes_stateful_set", SourceSchemaVersion: 0,
		TargetTypeName: "kubernetes_stateful_set_v1",
		SourceState: &tfprotov6.RawState{JSON: []byte(`{
		  "id":"default/historical",
		  "metadata":[{"name":"historical","namespace":"default","uid":"original-uid"}],
		  "spec":[{
		    "replicas":"0","service_name":"headless",
		    "template":[{"metadata":[{"labels":{"app":"historical"}}],"spec":[{
		      "container":[{"name":"main","image":"busybox:1.36","resources":[{"requests":[{"cpu":"100m"}],"limits":[]}]}],
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
	if resp.TargetState == nil {
		t.Fatal("schema-v0 alias was not moved")
	}
	state, err := resp.TargetState.Unmarshal(schemas.ResourceSchemas["kubernetes_stateful_set_v1"].ValueType())
	if err != nil {
		t.Fatal(err)
	}
	spec := tftypes.NewAttributePath().WithAttributeName("spec").WithElementKeyInt(0)
	pod := spec.WithAttributeName("template").WithElementKeyInt(0).WithAttributeName("spec").WithElementKeyInt(0)
	for _, tc := range []struct {
		at   *tftypes.AttributePath
		want string
	}{
		{tftypes.NewAttributePath().WithAttributeName("id"), "default/historical"},
		{tftypes.NewAttributePath().WithAttributeName("metadata").WithElementKeyInt(0).WithAttributeName("uid"), "original-uid"},
		{pod.WithAttributeName("container").WithElementKeyInt(0).WithAttributeName("resources").WithElementKeyInt(0).WithAttributeName("requests").WithElementKeyString("cpu"), "100m"},
		{pod.WithAttributeName("init_container").WithElementKeyInt(0).WithAttributeName("resources").WithElementKeyInt(0).WithAttributeName("requests").WithElementKeyString("memory"), "16Mi"},
		{spec.WithAttributeName("volume_claim_template").WithElementKeyInt(0).WithAttributeName("spec").WithElementKeyInt(0).WithAttributeName("resources").WithElementKeyInt(0).WithAttributeName("requests").WithElementKeyString("storage"), "1Gi"},
	} {
		got, _, err := tftypes.WalkAttributePath(state, tc.at)
		if err != nil {
			t.Fatal(err)
		}
		if !got.(tftypes.Value).Equal(tftypes.NewValue(tftypes.String, tc.want)) {
			t.Errorf("%s = %s, want %s", tc.at, got, tc.want)
		}
	}
	if resp.TargetIdentity == nil || resp.TargetIdentity.IdentityData == nil {
		t.Fatal("moved resource identity is missing")
	}
	identity, err := resp.TargetIdentity.IdentityData.Unmarshal(tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"api_version": tftypes.String,
		"kind":        tftypes.String,
		"name":        tftypes.String,
		"namespace":   tftypes.String,
	}})
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{
		"api_version": "apps/v1",
		"kind":        "StatefulSet",
		"name":        "historical",
		"namespace":   "default",
	} {
		got, _, err := tftypes.WalkAttributePath(identity, tftypes.NewAttributePath().WithAttributeName(name))
		if err != nil {
			t.Fatal(err)
		}
		if !got.(tftypes.Value).Equal(tftypes.NewValue(tftypes.String, want)) {
			t.Errorf("identity %s = %s, want %s", name, got, want)
		}
	}
}
