// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// namespaceSchemaVersion is 1 where the SDKv2 resource was 0.
//
// The bump is not cosmetic and is not a type change: it is the only mechanism
// that lets UpgradeState run at all. State written by the SDKv2 implementation
// holds a known empty map for an omitted `annotations`/`labels` and an empty
// string for an omitted `generate_name`, because SDKv2 zero-fills every leaf of
// a block it writes. The Framework plans null for those same configurations, so
// without a one-time conversion every existing namespace would plan
// `annotations: {} -> null` on the first plan after upgrade.
//
// UpgradeState is only consulted when the STORED version is lower than the
// current one, so keeping version 0 would mean the conversion never runs.
const namespaceSchemaVersion = 1

func namespaceV1Schema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Version:     namespaceSchemaVersion,
		Description: "Kubernetes supports multiple virtual clusters backed by the same physical cluster. These virtual clusters are called namespaces. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of this resource.",
				// SDKv2 injected `id` as Optional+Computed into every resource
				// and set it to the namespace name. It is published and
				// referenced by existing configurations, so it is declared
				// rather than dropped. The name is ForceNew, so the id cannot
				// change without a replacement and may be taken from state
				// instead of being re-planned as unknown.
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"wait_for_default_service_account": schema.BoolAttribute{
				Description: "Terraform will wait for the default service account to be created.",
				Optional:    true,
				// Computed is mandatory alongside Default: a non-Computed
				// attribute with a default fails ValidateImplementation for the
				// whole provider schema, not just this resource
				// (rule K8S-MIGRATE-004).
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": schema.ListNestedBlock{
				Description: "Standard namespace's metadata. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata",
				NestedObject: schema.NestedBlockObject{
					Attributes: namespaceMetadataAttributes(),
				},
				// SDKv2 declared this block Required with MaxItems: 1. Blocks
				// carry no Required flag of their own and the size validators
				// return early on null, so presence and cardinality need two
				// separate checks or an omitted block reaches Create and panics
				// on plan.Metadata[0] (rule K8S-MIGRATE-002).
				Validators: []validator.List{
					listvalidator.IsRequired(),
					listvalidator.SizeBetween(1, 1),
				},
			},
			// SDKv2 declared Delete only. Adding create/read/update keys here
			// would widen the published schema, so the create waiter uses
			// SDKv2's system default instead (see namespace_v1_crud.go).
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Delete: true}),
		},
	}
}

func namespaceMetadataAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"annotations": schema.MapAttribute{
			Description: "An unstructured key value map stored with the namespace that may be used to store arbitrary metadata. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/",
			ElementType: types.StringType,
			Optional:    true,
			Validators: []validator.Map{
				mapvalidator.KeysAre(annotationKeyValidator()),
			},
		},
		"generate_name": schema.StringAttribute{
			Description: "Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency",
			Optional:    true,
			Validators: []validator.String{
				generateNameValidator(),
				stringvalidator.ConflictsWith(
					path.MatchRelative().AtParent().AtName("name"),
				),
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"generation": schema.Int64Attribute{
			Description: "A sequence number representing a specific generation of the desired state.",
			Computed:    true,
		},
		"labels": schema.MapAttribute{
			Description: "Map of string keys and values that can be used to organize and categorize (scope and select) the namespace. May match selectors of replication controllers and services. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/",
			ElementType: types.StringType,
			Optional:    true,
			Validators: []validator.Map{
				mapvalidator.KeysAre(labelKeyValidator()),
				mapvalidator.ValueStringsAre(labelValueValidator()),
			},
		},
		"name": schema.StringAttribute{
			Description: "Name of the namespace, must be unique. Cannot be updated. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
			Optional:    true,
			Computed:    true,
			Validators: []validator.String{
				nameValidator(),
				stringvalidator.ConflictsWith(
					path.MatchRelative().AtParent().AtName("generate_name"),
				),
			},
			// ORDER IS LOAD-BEARING. Plan modifiers run in declaration order,
			// each one's output feeds the next, and a replace decision cannot
			// be cleared by a later modifier
			// (fwserver/attribute_plan_modification.go:2391-2424). Computed
			// nils are marked unknown before modifiers run
			// (server_planresourcechange.go:252 precedes :293), so with
			// `name` omitted — every generate_name user — RequiresReplace
			// placed first would compare unknown against the stored name, never
			// match, and propose destroying the namespace on every plan.
			// UseStateForUnknown must resolve the value first.
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
				stringplanmodifier.RequiresReplace(),
			},
		},
		"resource_version": schema.StringAttribute{
			Description: "An opaque value that represents the internal version of this namespace that can be used by clients to determine when namespace has changed. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency",
			Computed:    true,
		},
		"uid": schema.StringAttribute{
			Description: "The unique in time and space value for this namespace. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids",
			Computed:    true,
		},
	}
}

func (r *NamespaceV1) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = namespaceV1Schema(ctx)
}
