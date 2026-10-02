// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// The deprecated alias served batch/v1beta1: schema v1 differs from v1 CronJob
// only by timezone; schema v0 additionally stored container resource maps as
// singleton lists. The v1 resource itself has always used schema version 0.
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
			const summary = "Unable to move kubernetes_cron_job state"
			if req.SourceRawState == nil || len(req.SourceRawState.JSON) == 0 {
				resp.Diagnostics.AddError(summary, "Source JSON state is required; flatmap state is not supported.")
				return
			}
			var source map[string]json.RawMessage
			if err := json.Unmarshal(req.SourceRawState.JSON, &source); err != nil {
				resp.Diagnostics.AddError(summary, err.Error())
				return
			}
			var id string
			if err := json.Unmarshal(source["id"], &id); err != nil {
				resp.Diagnostics.AddError(summary, "The source must have a namespace/name ID.")
				return
			}
			namespace, name, err := cronJobIDParts(id)
			if err != nil {
				resp.Diagnostics.AddError(summary, err.Error())
				return
			}
			var metadata []struct {
				Name      string `json:"name"`
				Namespace string `json:"namespace"`
			}
			if err := json.Unmarshal(source["metadata"], &metadata); err != nil || len(metadata) != 1 {
				resp.Diagnostics.AddError(summary, "The source must have exactly one metadata element.")
				return
			}
			if metadata[0].Name != name || metadata[0].Namespace != namespace {
				resp.Diagnostics.AddError(summary, "The source metadata does not match its namespace/name ID.")
				return
			}
			var specs []map[string]interface{}
			decoder := json.NewDecoder(bytes.NewReader(source["spec"]))
			decoder.UseNumber()
			if err := decoder.Decode(&specs); err != nil || len(specs) != 1 || specs[0] == nil {
				resp.Diagnostics.AddError(summary, "The source must have exactly one spec element.")
				return
			}
			specs[0]["timezone"] = ""
			if req.SourceSchemaVersion == 0 {
				if err := cronJobUpgradeContainerResources(specs[0]); err != nil {
					resp.Diagnostics.AddError(summary, err.Error())
					return
				}
			}
			source["spec"], err = json.Marshal(specs)
			if err != nil {
				resp.Diagnostics.AddError(summary, err.Error())
				return
			}
			data, err := json.Marshal(source)
			if err != nil {
				resp.Diagnostics.AddError(summary, err.Error())
				return
			}
			converted, err := (&tfprotov6.RawState{JSON: data}).Unmarshal(schemaResponse.Schema.Type().TerraformType(ctx))
			if err != nil {
				resp.Diagnostics.AddError(summary, fmt.Sprintf("Could not decode the converted CronJob state: %s", err))
				return
			}
			resp.TargetState.Raw = converted
			if resp.TargetIdentity != nil {
				resp.Diagnostics.Append(resp.TargetIdentity.Set(ctx, cronJobIdentity(namespace, name))...)
			}
		},
	}}
}

func cronJobUpgradeContainerResources(spec map[string]interface{}) error {
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
	return upgradeJobResourcesV0(jobSpec)
}
