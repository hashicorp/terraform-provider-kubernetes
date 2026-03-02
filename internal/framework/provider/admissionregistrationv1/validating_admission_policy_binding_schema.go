// Copyright (c) HashiCorp, Inc.
package admissionregistrationv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (r *ValidatingAdmissionPolicyBinding) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Validating Admission Policy Binding Resource`,
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
					"policy_name": schema.StringAttribute{
						MarkdownDescription: `PolicyName references a ValidatingAdmissionPolicy name which the ValidatingAdmissionPolicyBinding binds to. If the referenced resource does not exist, this binding is considered invalid and will be ignored`,
						Required:            true,
					},
					"match_resources": schema.SingleNestedAttribute{
						MarkdownDescription: `MatchResources declares what resources match this binding and will be validated by it. 

  Note that this is intersected with the policy's <code>matchConstraints</code>, so only requests that are matched by the policy can be selected by this.
If this is unset, all resources matched by the policy are validated by this binding. When resourceRules is unset, it does not constrain resource matching. If a resource is matched by the other fields of this object, it will be validated.`,
						Optional:   true,
						Attributes: matchConstraintsFields(),
					},
					"param_ref": schema.SingleNestedAttribute{
						MarkdownDescription: `ParamRef specifies the parameter resource used to configure the admission control policy.
	It should point to a resource of the type specified in <code>ParamKind</code> of the bound ValidatingAdmissionPolicy. If the policy specifies a <code>ParamKind</code> and the resource referred to by <code>ParamRef</code> does not exist, this binding is considered mis-configured and the FailurePolicy of the ValidatingAdmissionPolicy applied.

  If the policy does not specify a <code>ParamKind</code> then this field is ignored, and the rules are evaluated without a param.`,
						Optional:   true,
						Attributes: paramRefFields(),
					},
					"validation_actions": schema.ListAttribute{
						MarkdownDescription: `ValidationActions declares how Validations of the referenced ValidatingAdmissionPolicy are enforced. If a validation evaluates to false it is always enforced according to these actions.

	Failures defined by the ValidatingAdmissionPolicy's FailurePolicy are enforced according
	to these actions only if the FailurePolicy is set to Fail, otherwise the failures are
	ignored. This includes compilation errors, runtime errors and misconfigurations of the policy.

	ValidationActions is declared as a set of action values. Order does not matter. validationActions may not contain duplicates of the same action.

    The supported actions values are:
    - <code>Deny</code> specifies that a validation failure results in a denied request.
    - <code>Warn</code> specifies that a validation failure is reported to the request client in HTTP Warning headers, with a warning code of 299. Warnings can be sent both for allowed or denied admission responses.
    - <code>Audit</code> specifies that a validation failure is included in the published audit event for the request.

    More details on: https://kubernetes.io/docs/reference/kubernetes-api/policy-resources/validating-admission-policy-binding-v1/
    <code>Deny</code> and <code>Warn</code> may not be used together since this combination needlessly duplicates the validation failure both in the API response body and the HTTP warning headers.`,
						Required:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.NoNullValues(),
						},
					},
				},
			},
		},
	}
}

func paramRefFields() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name": schema.StringAttribute{
			MarkdownDescription: "Name is the name of the resource being referenced. One of ***name*** or ***selector*** field must be set, but both are mutually exclusive properties. If one is set, the other must be unset. A single parameter used for all admission requests can be configured by setting the `name` field, leaving `selector` blank, and setting namespace if `paramKind` is namespace-scoped.",
			Optional:            true,
			Validators: []validator.String{
				stringvalidator.ExactlyOneOf(path.Expressions{
					path.MatchRelative(),
					path.MatchRelative().AtParent().AtName("selector"),
				}...),
			},
		},
		"namespace": schema.StringAttribute{
			MarkdownDescription: `Namespace is the namespace of the referenced resource. Allows limiting the search for params to a specific namespace. Applies to both ***name*** and ***selector*** fields.
  A per-namespace parameter may be used by specifying a namespace-scoped <code>paramKind</code> in the policy and leaving this field empty.
	  - If paramKind is cluster-scoped, this field MUST be **unset**. Setting this field results in a configuration error.
	  - If paramKind is namespace-scoped, the namespace of the object being evaluated for admission will be used when this field is left unset.

  Take care that if this is left empty the binding must not match any cluster-scoped resources, which will result in an error.`,
			Optional: true,
		},
		"parameter_not_found_action": schema.StringAttribute{
			MarkdownDescription: `ParameterNotFoundAction controls the behavior of the binding when the resource exists, and ***name*** or ***selector*** is valid, but there are no parameters matched by the binding. 

  If the value is set to "Allow", then no matched parameters will be treated as successful validation by the binding.
	If set to "Deny", then no matched parameters will be subject to the <code>failurePolicy</code> of the policy.

	Allowed values are "Allow" or "Deny", if not set by default the value is "Deny"`,
			Default:  stringdefault.StaticString("Deny"),
			Optional: true,
			Computed: true,
			Validators: []validator.String{
				stringvalidator.OneOf("Allow", "Deny"),
			},
		},
		"selector": schema.SingleNestedAttribute{
			MarkdownDescription: `Selector can be used to match multiple param objects based on their labels. If multiple params are found, they are all evaluated with the policy expressions and the results are ANDed together.

  One of ***name*** or ***selector*** field must be set, but both are mutually exclusive properties. If one is set, the other must be unset.`,
			Optional:   true,
			Attributes: labelSelectorFields(),
		},
	}
}
