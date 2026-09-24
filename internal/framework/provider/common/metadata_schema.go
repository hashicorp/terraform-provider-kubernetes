// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const generateNameRequiresReplaceDescription = "Replaces the object when generate_name changes, except when SDKv2-written state holds an empty string for an unset value."

// MetadataSchema returns the metadata block for a cluster-scoped object. It reproduces
// metadataSchema(objectName, generatableName) from kubernetes/schema_metadata.go,
// including its descriptions, so migrated resources keep identical documentation.
// objectName is interpolated into those descriptions, e.g. "namespace".
//
// The signature mirrors SDKv2's deliberately: a migrated call site is a literal
// transcription of the one it replaces, so parity can be checked by reading the two
// side by side.
//
// generatableName adds the generate_name attribute, and with it the name/generate_name
// ConflictsWith pair — SDKv2 declares that conflict only in the same branch. Resources
// generally pass true; data sources vary (the SDKv2 namespace and config_map data
// sources pass false, secret passes true), and the value must match the SDKv2 schema
// exactly, because it decides what appears in state.
//
// The model must match what is returned: decode into MetadataModel when generatableName
// is true, and into a model without GenerateName when it is false. A struct field with
// no corresponding attribute fails at decode time, not at compile time.
func MetadataSchema(objectName string, generatableName bool) schema.ListNestedBlock {
	// ConflictsWith comes before the syntax validator so an error names the conflict
	// rather than complaining about a value the user is about to remove.
	nameValidators := []validator.String{}
	if generatableName {
		nameValidators = append(nameValidators, stringvalidator.ConflictsWith(
			path.MatchRelative().AtParent().AtName("generate_name"),
		))
	}
	nameValidators = append(nameValidators, DNSSubdomainNameValidator())

	// SDKv2 declares metadata as TypeList{Required: true, MaxItems: 1}, which serializes as
	// a one-element array. ListNestedBlock reproduces that shape; SingleNestedBlock would
	// produce a bare object and break existing state.
	block := schema.ListNestedBlock{
		// "Exactly one" is stated in the description because tfplugindocs cannot see it
		// otherwise. Framework blocks have no Required field, so required-ness lives in the
		// validators below, which the docs generator does not read — without this sentence
		// the published schema reads "Optional" where SDKv2 rendered
		// "Required ... Min: 1, Max: 1".
		Description: fmt.Sprintf("Standard %s's metadata. Exactly one metadata block is required. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata", objectName),
		Validators: []validator.List{
			listvalidator.SizeAtLeast(1),
			listvalidator.IsRequired(),  // SDKv2 Required: true
			listvalidator.SizeAtMost(1), // SDKv2 MaxItems: 1
		},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"annotations": schema.MapAttribute{
					Description: fmt.Sprintf("An unstructured key value map stored with the %s that may be used to store arbitrary metadata. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/", objectName),
					ElementType: types.StringType,
					Optional:    true,
					Validators: []validator.Map{
						AnnotationsValidator(),
					},
				},
				"generation": schema.Int64Attribute{
					Description: "A sequence number representing a specific generation of the desired state.",
					Computed:    true,
				},
				"labels": schema.MapAttribute{
					Description: fmt.Sprintf("Map of string keys and values that can be used to organize and categorize (scope and select) the %s. May match selectors of replication controllers and services. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/", objectName),
					ElementType: types.StringType,
					Optional:    true,
					Validators: []validator.Map{
						LabelsValidator(),
					},
				},
				"name": schema.StringAttribute{
					Description: fmt.Sprintf("Name of the %s, must be unique. Cannot be updated. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names", objectName),
					Optional:    true,
					Computed:    true,
					// UseStateForUnknown must come first. name is Computed, so it plans as
					// unknown whenever any other attribute changes, and RequiresReplace treats
					// unknown as a change — replacing the object on an unrelated edit.
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
						stringplanmodifier.RequiresReplace(),
					},
					Validators: nameValidators,
				},
				"resource_version": schema.StringAttribute{
					Description: fmt.Sprintf("An opaque value that represents the internal version of this %s that can be used by clients to determine when %s has changed. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency", objectName, objectName),
					Computed:    true,
				},
				"uid": schema.StringAttribute{
					Description: fmt.Sprintf("The unique in time and space value for this %s. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids", objectName),
					Computed:    true,
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
			},
		},
	}

	if generatableName {
		block.NestedObject.Attributes["generate_name"] = schema.StringAttribute{
			Description: "Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency",
			Optional:    true,
			PlanModifiers: []planmodifier.String{
				// RequiresReplace, except when state holds "" and the plan is null. SDKv2 has
				// no null for primitives, so it stores an unset generate_name as "" where this
				// provider stores null. Both mean unset, but RequiresReplace compares values:
				// without this, a plan that skips refresh (terraform plan -refresh=false) reads
				// "" from SDKv2-written state, plans null from config, and recreates the object.
				// Safe because "" is not a value a practitioner can set — both providers reject
				// generate_name = "" at validate — so only zero-filled state matches.
				stringplanmodifier.RequiresReplaceIf(
					func(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
						sdkv2UnsetBecomingNull := req.StateValue.ValueString() == "" && req.PlanValue.IsNull()
						resp.RequiresReplace = !sdkv2UnsetBecomingNull
					},
					generateNameRequiresReplaceDescription,
					generateNameRequiresReplaceDescription,
				),
			},
			Validators: []validator.String{
				stringvalidator.ConflictsWith(
					path.MatchRelative().AtParent().AtName("name"),
				),
				DNSLabelPrefixValidator(),
			},
		}
	}

	return block
}
