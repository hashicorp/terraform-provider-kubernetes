package rbacv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (r *ClusterRole) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `cluster roles contain rules that represent a set of permissions`,
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: `The unique ID for this terraform resource`,
				Optional:            true,
				Computed:            true,
			},
			"aggregation_rule": schema.SingleNestedAttribute{
				MarkdownDescription: `AggregationRule is an optional field that describes how to build the Rules for this ClusterRole. If AggregationRule is set, then the Rules are controller managed and direct changes to Rules will be stomped by the controller.`,
				Optional:            true,

				Attributes: map[string]schema.Attribute{
					"cluster_role_selectors": schema.ListNestedAttribute{
						MarkdownDescription: `ClusterRoleSelectors holds a list of selectors which will be used to find ClusterRoles and create the rules. If any of the selectors match, then the ClusterRole's permissions will be added`,
						Optional:            true,

						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"match_expressions": schema.ListNestedAttribute{
									MarkdownDescription: `matchExpressions is a list of label selector requirements. The requirements are ANDed.`,
									Optional:            true,

									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"key": schema.StringAttribute{
												MarkdownDescription: `key is the label key that the selector applies to.`,
												Optional:            true,
											},
											"operator": schema.StringAttribute{
												MarkdownDescription: `operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.`,
												Optional:            true,
											},
											"values": schema.ListAttribute{
												MarkdownDescription: `values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.`,
												ElementType:         types.StringType,
												Optional:            true,
											},
										},
									},
								},
								"match_labels": schema.MapAttribute{
									MarkdownDescription: `matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.`,
									ElementType:         types.StringType,
									Optional:            true,
								},
							},
						},
					},
				},
			},
			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: `Standard object's metadata.`,
				Required:            true,

				Attributes: map[string]schema.Attribute{
					"annotations": schema.MapAttribute{
						MarkdownDescription: `Annotations is an unstructured key value map stored with a resource that may be set by external tools to store and retrieve arbitrary metadata. They are not queryable and should be preserved when modifying objects. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations`,
						ElementType:         types.StringType,
						Optional:            true,
					},
					"generate_name": schema.StringAttribute{
						MarkdownDescription: `GenerateName is an optional prefix, used by the server, to generate a unique name ONLY IF the Name field has not been provided. If this field is used, the name returned to the client will be different than the name passed. This value will also be combined with a unique suffix. The provided value has the same validation rules as the Name field, and may be truncated by the length of the suffix required to make the value unique on the server.

If this field is specified and the generated name exists, the server will return a 409.

Applied only if Name is not specified. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#idempotency`,
						Optional: true,
					},
					"generation": schema.Int64Attribute{
						MarkdownDescription: `A sequence number representing a specific generation of the desired state. Populated by the system. Read-only.`,
						Optional:            true,
						Computed:            true,
					},
					"labels": schema.MapAttribute{
						MarkdownDescription: `Map of string keys and values that can be used to organize and categorize (scope and select) objects. May match selectors of replication controllers and services. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels`,
						ElementType:         types.StringType,
						Optional:            true,
					},
					"name": schema.StringAttribute{
						MarkdownDescription: `Name must be unique within a namespace. Is required when creating resources, although some resources may allow a client to request the generation of an appropriate name automatically. Name is primarily intended for creation idempotence and configuration definition. Cannot be updated. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#names`,
						Optional:            true,
						Computed:            true,

						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"resource_version": schema.StringAttribute{
						MarkdownDescription: `An opaque value that represents the internal version of this object that can be used by clients to determine when objects have changed. May be used for optimistic concurrency, change detection, and the watch operation on a resource or set of resources. Clients must treat these values as opaque and passed unmodified back to the server. They may only be valid for a particular resource or set of resources.

Populated by the system. Read-only. Value must be treated as opaque by clients and . More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency`,
						Optional: true,
						Computed: true,
					},
					"uid": schema.StringAttribute{
						MarkdownDescription: `UID is the unique in time and space value for this object. It is typically generated by the server on successful creation of a resource and is not allowed to change on PUT operations.

Populated by the system. Read-only. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#uids`,
						Optional: true,
						Computed: true,
					},
				},
			},
			"rules": schema.ListNestedAttribute{
				MarkdownDescription: `Rules holds all the PolicyRules for this ClusterRole`,
				Optional:            true,

				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"api_groups": schema.ListAttribute{
							MarkdownDescription: `APIGroups is the name of the APIGroup that contains the resources.  If multiple API groups are specified, any action requested against one of the enumerated resources in any API group will be allowed. "" represents the core API group and "*" represents all API groups.`,
							ElementType:         types.StringType,
							Optional:            true,
						},
						"non_resource_urls": schema.ListAttribute{
							MarkdownDescription: `NonResourceURLs is a set of partial urls that a user should have access to.  *s are allowed, but only as the full, final step in the path Since non-resource URLs are not namespaced, this field is only applicable for ClusterRoles referenced from a ClusterRoleBinding. Rules can either apply to API resources (such as "pods" or "secrets") or non-resource URL paths (such as "/api"),  but not both.`,
							ElementType:         types.StringType,
							Optional:            true,
						},
						"resource_names": schema.ListAttribute{
							MarkdownDescription: `ResourceNames is an optional white list of names that the rule applies to.  An empty set means that everything is allowed.`,
							ElementType:         types.StringType,
							Optional:            true,
						},
						"resources": schema.ListAttribute{
							MarkdownDescription: `Resources is a list of resources this rule applies to. '*' represents all resources.`,
							ElementType:         types.StringType,
							Optional:            true,
						},
						"verbs": schema.ListAttribute{
							MarkdownDescription: `Verbs is a list of Verbs that apply to ALL the ResourceKinds contained in this rule. '*' represents all verbs.`,
							ElementType:         types.StringType,
							Optional:            true,
						},
					},
				},
			},
		},
	}
}
