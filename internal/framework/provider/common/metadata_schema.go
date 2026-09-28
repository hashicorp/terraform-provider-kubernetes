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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const generateNameRequiresReplaceDescription = "Replaces the object when generate_name changes, except when SDKv2-written state holds an empty string for an unset value."

// MetadataSchema mirrors SDKv2 metadataSchema for cluster-scoped objects.
// generatableName adds generate_name and its conflict with name; match the SDKv2 flag.
// Decode into MetadataModel when true, MetadataBase when false.
func MetadataSchema(objectName string, generatableName bool) schema.ListNestedBlock {
	return metadataBlock(objectName, metadataAttributes(objectName, generatableName))
}

// NamespacedMetadataSchema mirrors SDKv2 namespacedMetadataSchema, including the namespace default.
// NamespacedMetadataModel matches the true variant; the false variant needs a model without GenerateName.
func NamespacedMetadataSchema(objectName string, generatableName bool) schema.ListNestedBlock {
	attributes := metadataAttributes(objectName, generatableName)

	// Framework defaults require Computed. No namespace validator, matching SDKv2.
	attributes["namespace"] = schema.StringAttribute{
		Description: fmt.Sprintf("Namespace defines the space within which name of the %s must be unique.", objectName),
		Optional:    true,
		Computed:    true,
		Default:     stringdefault.StaticString("default"),
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
			stringplanmodifier.RequiresReplace(),
		},
	}

	return metadataBlock(objectName, attributes)
}

// metadataAttributes shares SDKv2 metadataFields and the optional generate_name variant.
func metadataAttributes(objectName string, generatableName bool) map[string]schema.Attribute {
	// ConflictsWith comes before the syntax validator so an error names the conflict
	// rather than complaining about a value the user is about to remove.
	nameValidators := []validator.String{}
	if generatableName {
		nameValidators = append(nameValidators, stringvalidator.ConflictsWith(
			path.MatchRelative().AtParent().AtName("generate_name"),
		))
	}
	nameValidators = append(nameValidators, DNSSubdomainNameValidator())

	attributes := map[string]schema.Attribute{
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
			// Resolve computed unknowns before checking replacement on unrelated edits.
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
	}
	if generatableName {
		attributes["generate_name"] = schema.StringAttribute{
			Description: "Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency",
			Optional:    true,
			PlanModifiers: []planmodifier.String{
				// SDKv2 stores unset as ""; Framework uses null. Exempt that transition
				// so plan -refresh=false does not recreate upgraded resources.
				stringplanmodifier.RequiresReplaceIf(
					func(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
						sdkv2UnsetBecomingNull := req.StateValue.Equal(types.StringValue("")) && req.PlanValue.IsNull()
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

	return attributes
}

// metadataBlock wraps a set of metadata attributes in the list block shape SDKv2 produced.
func metadataBlock(objectName string, attributes map[string]schema.Attribute) schema.ListNestedBlock {
	// Preserve SDKv2's one-element list wire shape, not a single object.
	return schema.ListNestedBlock{
		// tfplugindocs cannot infer required blocks from validators.
		Description: fmt.Sprintf("Standard %s's metadata. Exactly one metadata block is required. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata", objectName),
		Validators: []validator.List{
			listvalidator.SizeAtLeast(1),
			listvalidator.IsRequired(),  // SDKv2 Required: true
			listvalidator.SizeAtMost(1), // SDKv2 MaxItems: 1
		},
		NestedObject: schema.NestedBlockObject{Attributes: attributes},
	}
}
