// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
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

// MetadataSchema returns the metadata block for a cluster-scoped object whose name may be
// server-generated. It reproduces metadataSchema(objectName, true) from
// kubernetes/schema_metadata.go, including its descriptions, so migrated resources keep
// identical documentation. objectName is interpolated into those descriptions, e.g.
// "namespace". Decode it into MetadataModel.
func MetadataSchema(objectName string) schema.ListNestedBlock {
	// SDKv2 declares metadata as TypeList{Required: true, MaxItems: 1}, which serializes as
	// a one-element array. ListNestedBlock reproduces that shape; SingleNestedBlock would
	// produce a bare object and break existing state.
	return schema.ListNestedBlock{
		Description: fmt.Sprintf("Standard %s's metadata. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata", objectName),
		Validators: []validator.List{
			listvalidator.SizeAtLeast(1), // SDKv2 Required: true
			listvalidator.IsRequired(),
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
				"generate_name": schema.StringAttribute{
					Description: "Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency",
					Optional:    true,
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.RequiresReplace(),
					},
					Validators: []validator.String{
						stringvalidator.ConflictsWith(
							path.MatchRelative().AtParent().AtName("name"),
						),
						DNSLabelPrefixValidator(),
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
					Validators: []validator.String{
						stringvalidator.ConflictsWith(
							path.MatchRelative().AtParent().AtName("generate_name"),
						),
						DNSSubdomainNameValidator(),
					},
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
}
