// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestyaml

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (r *ManifestYAML) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Applies a single Kubernetes object from raw YAML using Server-Side Apply " +
			"(SSA). Unlike `kubernetes_manifest`, it does not require the typed OpenAPI schema at plan time. " +
			"See RFC-011.",
		Attributes: map[string]schema.Attribute{
			"yaml_body": schema.StringAttribute{
				MarkdownDescription: "Raw YAML for exactly one Kubernetes object. Marked sensitive because " +
					"the manifest may contain a Secret.",
				Required:  true,
				Sensitive: true,
			},
			"field_manager": schema.StringAttribute{
				MarkdownDescription: "Server-Side Apply field manager name. Use a unique value when sharing " +
					"an object with other managers.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("terraform"),
			},
			"force_conflicts": schema.BoolAttribute{
				MarkdownDescription: "Force apply, taking ownership of fields managed by another field manager.",
				Optional:            true,
			},
			"ignore_fields": schema.ListAttribute{
				MarkdownDescription: "Dotted field paths (e.g. \"spec.replicas\") to exclude from the owned-field " +
					"projection, so an external controller (e.g. an HPA) can own them without causing drift.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"force_replace_on": schema.ListAttribute{
				MarkdownDescription: "Dotted field paths (e.g. \"spec.volumeClaimTemplates\", \"spec.serviceName\") " +
					"that force replacement of the object when their value changes, instead of an in-place update. " +
					"Use this for Kubernetes-immutable fields (e.g. a StatefulSet's `volumeClaimTemplates`, `serviceName`, " +
					"`selector`, `podManagementPolicy`) that would otherwise be rejected on apply. Combine with " +
					"`delete { propagation_policy = \"Orphan\" }` to keep the underlying Pods/PVCs across the replacement " +
					"(the recreated object re-adopts them).",
				ElementType: types.StringType,
				Optional:    true,
			},

			// Computed identity / projection.
			"id": schema.StringAttribute{
				MarkdownDescription: "apiVersion=<>,kind=<>,namespace=<>,name=<>",
				Computed:            true,
			},
			"api_version":      schema.StringAttribute{Computed: true},
			"kind":             schema.StringAttribute{Computed: true},
			"name":             schema.StringAttribute{Computed: true},
			"namespace":        schema.StringAttribute{Computed: true},
			"uid":              schema.StringAttribute{Computed: true},
			"resource_version": schema.StringAttribute{Computed: true},
			"status": schema.StringAttribute{
				MarkdownDescription: "The live object's `.status` as JSON. Parse with `jsondecode(...)` " +
					"to read values such as `loadBalancer.ingress[0].ip`.",
				Computed: true,
			},
			"live_manifest": schema.StringAttribute{
				MarkdownDescription: "Canonical JSON of the fields owned by this resource's field manager " +
					"(the drift anchor). Only these fields are compared on plan.",
				Computed:  true,
				Sensitive: true,
			},
		},
		Blocks: map[string]schema.Block{
			"wait": schema.SingleNestedBlock{
				MarkdownDescription: "Block until the object becomes ready. On failure, pod-level errors " +
					"(CrashLoopBackOff, ImagePullBackOff, FailedScheduling, …) are surfaced.",
				Attributes: map[string]schema.Attribute{
					"rollout": schema.BoolAttribute{
						MarkdownDescription: "Wait for a workload (Deployment/StatefulSet/DaemonSet) rollout to complete.",
						Optional:            true,
					},
					"condition": schema.StringAttribute{
						MarkdownDescription: "Wait for a status condition, e.g. \"Available=True\" or \"Ready\".",
						Optional:            true,
					},
					"fields": schema.MapAttribute{
						MarkdownDescription: "Map of jsonpath expression → expected string value, e.g. " +
							"{\"{.status.readyReplicas}\" = \"3\"}.",
						ElementType: types.StringType,
						Optional:    true,
					},
					"timeout": schema.StringAttribute{
						MarkdownDescription: "Wait timeout as a Go duration (default 5m).",
						Optional:            true,
					},
				},
			},
			"delete": schema.SingleNestedBlock{
				MarkdownDescription: "Deletion options.",
				Attributes: map[string]schema.Attribute{
					"propagation_policy": schema.StringAttribute{
						MarkdownDescription: "Foreground | Background | Orphan (default: Background). " +
							"Use `Orphan` to keep dependent objects (e.g. a StatefulSet's Pods/PVCs) when the object is deleted.",
						Optional: true,
						Validators: []validator.String{
							stringvalidator.OneOf("Foreground", "Background", "Orphan"),
						},
					},
					"grace_period_seconds": schema.Int64Attribute{
						Optional: true,
					},
				},
			},
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}
