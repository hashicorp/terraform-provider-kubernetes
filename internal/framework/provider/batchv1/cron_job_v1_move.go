// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

// The deprecated alias served batch/v1beta1: schema v1 differs from v1 CronJob
// only by timezone; schema v0 additionally stored container resource maps as
// singleton lists.
func (r *CronJobV1) MoveState(ctx context.Context) []resource.StateMover {
	var schemaResponse resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
	return []resource.StateMover{{
		StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
			if req.SourceTypeName != "kubernetes_cron_job" ||
				(req.SourceSchemaVersion != 0 && req.SourceSchemaVersion != 1) ||
				!strings.HasSuffix(req.SourceProviderAddress, "/hashicorp/kubernetes") {
				return
			}
			var namespace, name string
			converted, err := common.DecodeLegacyState(ctx, req.SourceRawState, schemaResponse.Schema, func(values map[string]any) error {
				var err error
				if namespace, name, err = common.LegacyStateName(values); err != nil {
					return err
				}
				specs, _ := values["spec"].([]any)
				if len(specs) != 1 {
					return fmt.Errorf("expected exactly one spec element")
				}
				spec, ok := specs[0].(map[string]any)
				if !ok {
					return fmt.Errorf("invalid spec object")
				}
				spec["timezone"] = ""
				return cronJobUpgradeJobSpec(spec, req.SourceSchemaVersion == 0)
			})
			if err != nil {
				resp.Diagnostics.AddError("Unable to move kubernetes_cron_job state", err.Error())
				return
			}
			resp.TargetState.Raw = converted
			if resp.TargetIdentity != nil {
				resp.Diagnostics.Append(resp.TargetIdentity.Set(ctx, cronJobIdentity(namespace, name))...)
			}
		},
	}}
}

// UpgradeState accepts state of the SDKv2 resource, which used schema version 0.
func (r *CronJobV1) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	var schemaResponse resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
	return map[int64]resource.StateUpgrader{
		0: {
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				value, err := common.DecodeLegacyState(ctx, req.RawState, schemaResponse.Schema, func(values map[string]any) error {
					spec, ok := jobLegacyObject(values["spec"])
					if !ok {
						return fmt.Errorf("expected exactly one spec element")
					}
					// Same-type SDKv2 CronJob v0 already stored quantity maps.
					return cronJobUpgradeJobSpec(spec, false)
				})
				if err != nil {
					resp.Diagnostics.AddError("Unable to upgrade kubernetes_cron_job_v1 state", err.Error())
					return
				}
				resp.State = tfsdk.State{Schema: schemaResponse.Schema, Raw: value}
			},
		},
	}
}

func cronJobUpgradeJobSpec(spec map[string]any, legacyQuantities bool) error {
	jobSpec := spec
	for _, field := range []string{"job_template", "spec"} {
		elements, ok := jobSpec[field].([]interface{})
		if !ok || len(elements) != 1 {
			return fmt.Errorf("expected exactly one %s element in historical CronJob state", field)
		}
		jobSpec, ok = elements[0].(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid %s object in historical CronJob state", field)
		}
	}
	return upgradeJobSpecState(jobSpec, "spec[0].job_template[0].spec[0]", legacyQuantities)
}
