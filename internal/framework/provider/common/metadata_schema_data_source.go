// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The data source metadata block, the counterpart of what SDKv2 builds by calling
// metadataSchema / namespacedMetadataSchema from a data source.
//
// This is a separate construction rather than a reuse of MetadataSchema, even though the
// resource block's Go type does satisfy datasource/schema.Block. Three things differ:
//
//   - There is no generatableName flag. SDKv2 needed one only because a single function
//     served both resources and data sources; generate_name is a create-time field and has
//     no meaning when looking up an existing object.
//   - Plan modifiers and defaults do not apply. A data source is read every time, so
//     UseStateForUnknown, RequiresReplace and stringdefault have nothing to act on.
//   - Nothing here is ForceNew or a replace trigger, so none of the ""-versus-null
//     reconciliation the resource schema carries is needed.
//
// What is kept identical to SDKv2 is the attribute set, the Optional/Computed flags and the
// descriptions, because those decide the state shape and the generated documentation.

// DataSourceMetadataSchema returns the metadata block for a cluster-scoped data source.
func DataSourceMetadataSchema(objectName string) schema.ListNestedBlock {
	return dataSourceMetadataBlock(objectName, dataSourceMetadataAttributes(objectName))
}

// NamespacedDataSourceMetadataSchema returns the metadata block for a namespaced data
// source: the cluster-scoped attributes plus namespace.
//
// Decode into NamespacedMetadataBase. Unlike the resource schema, namespace carries no
// default here — a data source reads whatever the practitioner asked for, and defaulting it
// to "default" would silently look up the wrong object.
func NamespacedDataSourceMetadataSchema(objectName string) schema.ListNestedBlock {
	attributes := dataSourceMetadataAttributes(objectName)
	attributes["namespace"] = schema.StringAttribute{
		Description: fmt.Sprintf("Namespace defines the space within which name of the %s must be unique.", objectName),
		Optional:    true,
		Computed:    true,
	}

	return dataSourceMetadataBlock(objectName, attributes)
}

// dataSourceMetadataAttributes mirrors SDKv2's metadataFields, minus generate_name.
//
// annotations and labels are Optional **and** Computed. SDKv2 declares them Optional only,
// but that cannot work here: Read returns every key the object carries, including ones the
// configuration never mentioned, and without Computed the framework rejects a result that
// does not match config. SDKv2 tolerated the mismatch through its legacy type system, which
// the framework deliberately does not have.
//
// name is Required, which diverges from SDKv2's Optional + Computed. There is nothing for a
// data source to look up without it, so SDKv2's declaration only ever deferred the failure:
// an omitted name validated, then failed at read. Declaring it Required moves that to plan
// time, where the practitioner gets a clear message pointing at the attribute.
//
// This is a breaking change and needs a release note. A configuration that omits
// metadata.name validates against SDKv2 today and will stop validating — though it could
// never have worked, so nothing that currently succeeds is affected.
func dataSourceMetadataAttributes(objectName string) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"annotations": schema.MapAttribute{
			Description: fmt.Sprintf("An unstructured key value map stored with the %s that may be used to store arbitrary metadata. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/", objectName),
			ElementType: types.StringType,
			Optional:    true,
			Computed:    true,
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
			Computed:    true,
			Validators: []validator.Map{
				LabelsValidator(),
			},
		},
		"name": schema.StringAttribute{
			Description: fmt.Sprintf("Name of the %s, must be unique. Cannot be updated. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names", objectName),
			Required:    true,
			Validators: []validator.String{
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
		},
	}
}

// dataSourceMetadataBlock wraps the attributes in the list block shape SDKv2 produced, so
// existing state and configuration keep working.
func dataSourceMetadataBlock(objectName string, attributes map[string]schema.Attribute) schema.ListNestedBlock {
	return schema.ListNestedBlock{
		// "Exactly one" is stated in the description because tfplugindocs cannot see it
		// otherwise: framework blocks have no Required field, so required-ness lives in the
		// validators, which the docs generator does not read.
		Description: fmt.Sprintf("Standard %s's metadata. Exactly one metadata block is required. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata", objectName),
		Validators: []validator.List{
			listvalidator.IsRequired(),  // SDKv2 Required: true
			listvalidator.SizeAtMost(1), // SDKv2 MaxItems: 1
		},
		NestedObject: schema.NestedBlockObject{Attributes: attributes},
	}
}
