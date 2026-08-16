// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package nodev1

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// dnsSubdomainRegexp validates a Kubernetes DNS subdomain name (RFC 1123).
// Used for metadata.name.
var dnsSubdomainRegexp = regexp.MustCompile(`^[a-z0-9]([a-z0-9\-\.]*[a-z0-9])?$`)

// qualifiedNameRegexp validates a Kubernetes qualified name used as annotation/label key.
// Format: [prefix/]name where prefix is an optional DNS subdomain and name is a DNS label.
var qualifiedNameRegexp = regexp.MustCompile(`^([a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*\/)?[a-zA-Z0-9]([-a-zA-Z0-9_.]*[a-zA-Z0-9])?$`)

func (r *RuntimeClassV1) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// Version 0 — matches the SDKv2 schema version so existing state files
		// are compatible with no state upgrade required.
		Description: "A RuntimeClass is used to determine which container runtime is used to run " +
			"all containers in a pod. RuntimeClass objects in the node.k8s.io API group select a " +
			"specific handler. More info: https://kubernetes.io/docs/concepts/containers/runtime-class/",

		Blocks: map[string]schema.Block{
			// metadata is a ListNestedBlock with no maximum explicitly enforced in
			// schema (Framework does not support MaxItems on blocks natively), but
			// exactly one block is expected and validated by ConflictsWith / ExactlyOneOf
			// on name vs generate_name. This matches the SDKv2 TypeList(MaxItems:1) shape
			// so HCL syntax (metadata { }) and state paths (metadata.0.name) are unchanged.
			"metadata": schema.ListNestedBlock{
				Description: "Standard RuntimeClass's metadata. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"annotations": schema.MapAttribute{
							Description: "An unstructured key value map stored with the RuntimeClass that may be used to store arbitrary metadata. " +
								"More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/",
							ElementType: types.StringType,
							Optional:    true,
							Validators: []validator.Map{
								mapvalidator.KeysAre(
									stringvalidator.RegexMatches(
										qualifiedNameRegexp,
										"must be a qualified name: an optional DNS subdomain prefix followed by a name",
									),
								),
							},
						},

						"generate_name": schema.StringAttribute{
							Description: "Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. " +
								"This value will also be combined with a unique suffix. " +
								"More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency",
							Optional: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
							Validators: []validator.String{
								stringvalidator.ExactlyOneOf(
									path.MatchRelative().AtParent().AtName("name"),
								),
								stringvalidator.RegexMatches(
									dnsLabelRegexp,
									"must be a valid DNS label: lowercase alphanumeric characters or '-', "+
										"must start and end with an alphanumeric character",
								),
							},
						},

						"generation": schema.Int64Attribute{
							Description: "A sequence number representing a specific generation of the desired state.",
							Computed:    true,
						},

						"labels": schema.MapAttribute{
							Description: "Map of string keys and values that can be used to organize and categorize (scope and select) the RuntimeClass. " +
								"May match selectors of replication controllers and services. " +
								"More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/",
							ElementType: types.StringType,
							Optional:    true,
							Validators: []validator.Map{
								mapvalidator.KeysAre(
									stringvalidator.RegexMatches(
										qualifiedNameRegexp,
										"must be a qualified name: an optional DNS subdomain prefix followed by a name",
									),
								),
							},
						},

						"name": schema.StringAttribute{
							Description: "Name of the RuntimeClass, must be unique. Cannot be updated. " +
								"More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
							Optional: true,
							Computed: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
							Validators: []validator.String{
								stringvalidator.ExactlyOneOf(
									path.MatchRelative().AtParent().AtName("generate_name"),
								),
								stringvalidator.RegexMatches(
									dnsSubdomainRegexp,
									"must be a valid DNS subdomain: lowercase alphanumeric characters, '-' or '.', "+
										"must start and end with an alphanumeric character",
								),
							},
						},

						"resource_version": schema.StringAttribute{
							Description: "An opaque value that represents the internal version of this RuntimeClass that can be used by clients to determine when RuntimeClass has changed. " +
								"More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency",
							Computed: true,
						},

						"uid": schema.StringAttribute{
							Description: "The unique in time and space value for this RuntimeClass. " +
								"More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids",
							Computed: true,
						},
					},
				},
			},
		},

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for this resource (the RuntimeClass name).",
				Computed:    true,
			},

			"handler": schema.StringAttribute{
				Description: "Specifies the underlying runtime and configuration that the CRI implementation will use to handle pods of this class. " +
					"The RuntimeClass.Handler field is directly analogous to the PodSpec.RuntimeClassName field. " +
					"See https://kubernetes.io/docs/concepts/containers/runtime-class/ for more information.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						dnsLabelRegexp,
						"must be a valid DNS label: lowercase alphanumeric characters or '-', "+
							"must start and end with an alphanumeric character",
					),
				},
			},
		},
	}
}
