// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// The batch/v1beta1 kubernetes_cron_job cannot be created on current clusters,
// so its legacy state shapes are pinned here.
func TestCronJobV1MoveLegacyState(t *testing.T) {
	const metadata = `"id":"ns/cj","metadata":[{"name":"cj","namespace":"ns","self_link":"/apis/batch/v1beta1/namespaces/ns/cronjobs/cj"}]`
	const spec = `"spec":[{"schedule":"0 0 1 1 *","job_template":[{"spec":[{"template":[{"spec":[{"container":[{"name":"c","resources":[%s]}]}]}]}]}]}]`
	for _, tc := range []struct {
		name      string
		version   int64
		resources string
	}{
		{name: "v1 with self_link", version: 1, resources: `{"limits":{"cpu":"100m"},"requests":{}}`},
		{name: "v0 with self_link", version: 0, resources: `{"limits":[{"cpu":"100m"}],"requests":[]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			r := &CronJobV1{}
			var schemaResponse resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
			resp := resource.MoveStateResponse{TargetState: tfsdk.State{Schema: schemaResponse.Schema}}
			r.MoveState(ctx)[0].StateMover(ctx, resource.MoveStateRequest{
				SourceTypeName:        "kubernetes_cron_job",
				SourceSchemaVersion:   tc.version,
				SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes",
				SourceRawState:        &tfprotov6.RawState{JSON: []byte("{" + metadata + "," + fmt.Sprintf(spec, tc.resources) + "}")},
			}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			container := path.Root("spec").AtListIndex(0).AtName("job_template").AtListIndex(0).AtName("spec").AtListIndex(0).
				AtName("template").AtListIndex(0).AtName("spec").AtListIndex(0).AtName("container").AtListIndex(0)
			var cpu, timezone types.String
			resp.Diagnostics.Append(resp.TargetState.GetAttribute(ctx, container.AtName("resources").AtListIndex(0).AtName("limits").AtMapKey("cpu"), &cpu)...)
			resp.Diagnostics.Append(resp.TargetState.GetAttribute(ctx, path.Root("spec").AtListIndex(0).AtName("timezone"), &timezone)...)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			if cpu.ValueString() != "100m" || timezone.IsNull() || timezone.ValueString() != "" {
				t.Fatalf("cpu = %s, timezone = %s", cpu, timezone)
			}
		})
	}
}
