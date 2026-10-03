// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

func TestBatchLegacyStateTemplateValidation(t *testing.T) {
	ctx := context.Background()
	for _, route := range []struct {
		name     string
		resource resource.Resource
		version  int64
		source   string
		required bool
	}{
		{"job upgrade v0", &JobV1{}, 0, "", true},
		{"job upgrade v1", &JobV1{}, 1, "", false},
		{"job move v0", &JobV1{}, 0, "kubernetes_job", true},
		{"job move v1", &JobV1{}, 1, "kubernetes_job", false},
		{"cronjob upgrade v0", &CronJobV1{}, 0, "", false},
		{"cronjob move v0", &CronJobV1{}, 0, "kubernetes_cron_job", true},
		{"cronjob move v1", &CronJobV1{}, 1, "kubernetes_cron_job", false},
	} {
		var schemaResponse resource.SchemaResponse
		route.resource.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
		for _, tc := range []struct {
			name, field string
			invalid     bool
			missing     bool
		}{
			{"absent", "", false, true},
			{"null", `"template":null`, false, true},
			{"empty", `"template":[]`, false, true},
			{"one object", `"template":[{}]`, false, false},
			{"multiple objects", `"template":[{},{}]`, true, false},
			{"null element", `"template":[null]`, true, false},
		} {
			t.Run(route.name+"/"+tc.name, func(t *testing.T) {
				spec := "{" + tc.field + "}"
				if _, cron := route.resource.(*CronJobV1); cron {
					spec = `{"schedule":"0 0 1 1 *","job_template":[{"spec":[` + spec + `]}]}`
				}
				raw := &tfprotov6.RawState{JSON: []byte(`{"id":"ns/test","metadata":[{"name":"test","namespace":"ns"}],"spec":[` + spec + `]}`)}
				state := tfsdk.State{Schema: schemaResponse.Schema}
				wantError := tc.invalid || route.required && tc.missing
				if route.source == "" {
					response := resource.UpgradeStateResponse{State: state}
					upgraders := route.resource.(resource.ResourceWithUpgradeState).UpgradeState(ctx)
					upgraders[route.version].StateUpgrader(ctx, resource.UpgradeStateRequest{RawState: raw}, &response)
					if response.Diagnostics.HasError() != wantError {
						t.Fatalf("diagnostics = %v, want error = %t", response.Diagnostics, wantError)
					}
				} else {
					response := resource.MoveStateResponse{TargetState: state}
					movers := route.resource.(resource.ResourceWithMoveState).MoveState(ctx)
					movers[0].StateMover(ctx, resource.MoveStateRequest{
						SourceTypeName: route.source, SourceSchemaVersion: route.version,
						SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes", SourceRawState: raw,
					}, &response)
					if response.Diagnostics.HasError() != wantError {
						t.Fatalf("diagnostics = %v, want error = %t", response.Diagnostics, wantError)
					}
				}
			})
		}
	}
}
