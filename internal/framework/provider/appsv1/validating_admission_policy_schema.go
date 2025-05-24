package appsv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (r *ValidatingAdmissionPolicy) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Validating Admission Policy Resource`,
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: `The unique ID for this terraform resource`,
				Optional:            true,
				Computed:            true,
			},
			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: `Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata`,
				Optional:            true,
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
					},
					"namespace": schema.StringAttribute{
						MarkdownDescription: `Namespace defines the space within which each name must be unique. An empty namespace is equivalent to the "default" namespace, but "default" is the canonical representation. Not all objects are required to be scoped to a namespace - the value of this field for those objects will be empty.

Must be a DNS_LABEL. Cannot be updated. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces`,
						Optional: true,
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
			"spec": schema.SingleNestedAttribute{
				MarkdownDescription: "Rule defining a set of permissions for the role",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"audit_annotations": schema.ListNestedAttribute{
						MarkdownDescription: "auditAnnotations contains CEL expressions which are used to produce audit annotations for the audit event of the API request.",
						Required:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: auditAnnotationsFields(),
						},
					},
					"failure_policy": schema.StringAttribute{
						MarkdownDescription: "failurePolicy defines how to handle failures for the admission policy.",
						Required:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("Fail", "Ignore"),
						},
					},
					"match_conditions": schema.ListNestedAttribute{
						MarkdownDescription: "MatchConditions is a list of conditions that must be met for a request to be validated.",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: matchConditionsFields(),
						},
					},
					"match_constraints": schema.SingleNestedAttribute{
						MarkdownDescription: "MatchConstraints specifies what resources this policy is designed to validate.",
						Required:            true,
						Attributes:          matchConstraintsFields(),
					},
					"param_kind": schema.SingleNestedAttribute{
						MarkdownDescription: "ParamKind specifies the kind of resources used to parameterize this policy",
						Optional:            true,
						Attributes:          paramKindFields(),
					},
					"validations": schema.ListNestedAttribute{
						MarkdownDescription: "Validations contain CEL expressions which is used to apply the validation.",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: validationFields(),
						},
					},
					"variables": schema.ListNestedAttribute{
						MarkdownDescription: "Variables contain definitions of variables that can be used in composition of other expressions.",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: variableFields(),
						},
					},
				},
			},
			"status": schema.SingleNestedAttribute{
				MarkdownDescription: `The status of the ValidatingAdmissionPolicy, including warnings that are useful to determine if the policy behaves in the expected way.`,
				Optional:            true,
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"conditions": schema.ListNestedAttribute{
						MarkdownDescription: `The conditions represent the latest available observations of a policy's current state.`,
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"last_transition_time": schema.StringAttribute{
									Computed:            true,
									MarkdownDescription: `Last time the condition transitioned from one status to another.`,
									Optional:            true,
								},
								"message": schema.StringAttribute{
									Computed:            true,
									MarkdownDescription: `A human readable message indicating details about the transition.`,
									Optional:            true,
								},
								"observed_generation": schema.Int64Attribute{
									Computed:            true,
									MarkdownDescription: `The generation observed by the deployment controller.`,
									Optional:            true,
								},
								"reason": schema.StringAttribute{
									Computed:            true,
									MarkdownDescription: `The reason for the condition's last transition.`,
									Optional:            true,
								},
								"status": schema.StringAttribute{
									Computed:            true,
									MarkdownDescription: `Status of the condition, one of True, False, Unknown.`,
									Optional:            true,
								},
								"type": schema.StringAttribute{
									Computed:            true,
									MarkdownDescription: `Type of deployment condition.`,
									Optional:            true,
								},
							},
						},
					},
					"observed_generation": schema.Int64Attribute{
						Computed:            true,
						MarkdownDescription: `The generation observed by the deployment controller.`,
						Optional:            true,
					},
					"type_checking": schema.SingleNestedAttribute{
						Computed:            true,
						MarkdownDescription: `The results of type checking for each expression. Presence of this field indicates the completion of the type checking.`,
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"expression_warning": schema.ListNestedAttribute{
								MarkdownDescription: `The type checking warnings for each expression.`,
								Optional:            true,
								Computed:            true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"field_ref": schema.StringAttribute{
											Description: "The path to the field that refers the expression. For example, the reference to the expression of the first item of validations is \"spec.validations[0].expression\"",
											Optional:    true,
											Computed:    true,
										},
										"warning": schema.StringAttribute{
											MarkdownDescription: `The content of type checking information in a human-readable form. Each line of the warning contains the type that the expression is checked against, followed by the type check error from the compiler.`,
											Optional:            true,
											Computed:            true,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func auditAnnotationsFields() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"key": schema.StringAttribute{
			Description: "key specifies the audit annotation key.",
			Required:    true,
		},
		"value_expression": schema.StringAttribute{
			Description: "valueExpression represents the expression which is evaluated by CEL to produce an audit annotation value.",
			Required:    true,
		},
	}
}

func matchConditionsFields() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"expression": schema.StringAttribute{
			Description: "Expression represents the expression which will be evaluated by CEL.",
			Required:    true,
		},
		"name": schema.StringAttribute{
			Description: "Name is an identifier for this match condition, used for strategic merging of MatchConditions, as well as providing an identifier for logging purposes.",
			Required:    true,
		},
	}
}

func matchConstraintsFields() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"exclude_resource_rules": schema.ListNestedAttribute{
			Description: "ExcludeResourceRules describes what operations on what resources/subresources the ValidatingAdmissionPolicy should not care about.",
			Optional:    true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: namedRuleWithOperationsFields(),
			},
		},
		"match_policy": schema.StringAttribute{
			Optional:    true,
			Description: "matchPolicy defines how the MatchResources list is used to match incoming requests. Allowed values are Exact or Equivalent.",
			Validators: []validator.String{
				stringvalidator.OneOf("Exact", "Equivalent"),
			},
		},
		"namespace_selector": schema.SingleNestedAttribute{
			Description: "NamespaceSelector decides whether to run the admission control policy on an object based on whether the namespace for that object matches the selector.",
			Optional:    true,
			Attributes:  labelSelectorFields(),
		},
		"object_selector": schema.SingleNestedAttribute{
			Description: "ObjectSelector decides whether to run the validation based on if the object has matching labels.",
			Optional:    true,
			Attributes:  labelSelectorFields(),
		},
		"resource_rules": schema.ListNestedAttribute{
			Description: "ResourceRules describes what operations on what resources/subresources the ValidatingAdmissionPolicy matches.",
			Optional:    true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: namedRuleWithOperationsFields(),
			},
		},
	}
}

func paramKindFields() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"api_version": schema.StringAttribute{
			Description: "APIVersion is the API group version the resources belong to. In format of \"group/version\"",
			Required:    true,
		},
		"kind": schema.StringAttribute{
			Description: "Kind is the API kind the resources belong to.",
			Required:    true,
		},
	}
}

func validationFields() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"expression": schema.StringAttribute{
			Description: "Expression represents the expression which will be evaluated by CEL.",
			Required:    true,
		},
		"message": schema.StringAttribute{
			Description: "Message represents the message displayed when validation fails.",
			Required:    true,
		},
		"message_expression": schema.StringAttribute{
			Description: "Message Expression declares a CEL expression that evaluates to the validation failure message that is returned when this rule fails.",
			Optional:    true,
		},
		"reason": schema.StringAttribute{
			Description: "Reason represents a machine-readable description of why this validation failed.",
			Optional:    true,
		},
	}
}

func variableFields() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"expression": schema.StringAttribute{
			Description: "Expression is the expression that will be evaluated as the value of the variable.",
			Optional:    true,
		},
		"name": schema.StringAttribute{
			Description: "Name is the name of the variable.",
			Optional:    true,
		},
	}
}

func namedRuleWithOperationsFields() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"api_groups": schema.ListAttribute{
			Description: "APIGroups is the API groups the resources belong to. '\\*' is all groups. If '\\*' is present, the length of the slice must be one.",
			Required:    true,
			ElementType: types.StringType,
		},
		"api_versions": schema.ListAttribute{
			Description: "APIVersions is the API versions the resources belong to. '\\*' is all versions. If '\\*' is present, the length of the slice must be one. Required.",
			Required:    true,
			ElementType: types.StringType,
		},
		"operations": schema.ListAttribute{
			Description: "Operations is the operations the admission hook cares about - CREATE, UPDATE, DELETE, CONNECT or * for all of those operations and any future admission operations that are added.",
			Required:    true,
			ElementType: types.StringType,
		},
		"resource_names": schema.ListAttribute{
			Description: "ResourceNames is an optional white list of names that the rule applies to. An empty set means that everything is allowed.",
			Optional:    true,
			ElementType: types.StringType,
		},
		"resources": schema.ListAttribute{
			Description: "Resources is a list of resources this rule applies to.",
			Required:    true,
			ElementType: types.StringType,
		},
		"scope": schema.StringAttribute{
			Description: "scope specifies the scope of this rule.",
			Optional:    true,
		},
	}
}

func labelSelectorFields() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"label_selector": schema.SingleNestedAttribute{
			MarkdownDescription: `A label query over a set of resources, in this case pods.`,
			Optional:            true,
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
	}

}
