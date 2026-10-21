// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	corev1 "k8s.io/api/core/v1"
)

// The data source metadata block, the counterpart of SDKv2's metadataSchema /
// namespacedMetadataSchema when called from a data source.
// DataSourceMetadataSchema returns the metadata block for a cluster-scoped data source.
// Decode into MetadataBase.
func DataSourceMetadataSchema(objectName string) schema.ListNestedBlock {
	return dataSourceMetadataBlock(objectName, dataSourceMetadataAttributes(objectName))
}

// NamespacedDataSourceMetadataSchema returns the metadata block for a namespaced data
// source: the cluster-scoped attributes plus namespace. Decode into
// NamespacedMetadataBase.
//
// namespace is Optional but carries no default, because the framework has no defaults
// for data sources at all — datasource/schema attributes have no Default field, since
// defaults are applied during plan modification and a data source has none.
//
// SDKv2 does default it. namespacedMetadataSchema sets Default: "default" (via
// conditionalDefault in kubernetes/schema_metadata.go) and the 10 namespaced data
// sources using it read from the default namespace when the configuration omits one.
// Callers must therefore apply the fallback in Read — see ResolveDataSourceNamespace —
// and write the result back into the model, or omitting namespace will look up a
// different object than SDKv2 did.
func NamespacedDataSourceMetadataSchema(objectName string) schema.ListNestedBlock {
	attributes := dataSourceMetadataAttributes(objectName)
	attributes["namespace"] = schema.StringAttribute{
		Description: fmt.Sprintf("Namespace defines the space within which name of the %s must be unique. Defaults to %q when not set.", objectName, corev1.NamespaceDefault),
		Optional:    true,
		Computed:    true,
	}

	return dataSourceMetadataBlock(objectName, attributes)
}

// dataSourceMetadataAttributes mirrors SDKv2's metadataFields, minus generate_name.
//
// annotations and labels stay settable, as in SDKv2, even though the read consumes only
// metadata.name: making them read-only would reject configurations that validate today,
// which is a breaking change. They are Optional and Computed rather than SDKv2's Optional
// alone, because the read returns every key the object carries, and without Computed the
// framework rejects a result that does not match configuration.
//
// name is Required, diverging from SDKv2's Optional + Computed. That is not a breaking
// change: there is nothing to look up without a name, so a configuration that omitted it
// could never succeed — SDKv2's declaration only deferred the failure from plan to read.
func dataSourceMetadataAttributes(objectName string) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"annotations": schema.MapAttribute{
			Description: fmt.Sprintf("An unstructured key value map stored with the %s that may be used to store arbitrary metadata. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/", objectName),
			ElementType: types.StringType,
			Optional:    true,
			Computed:    true,
			Validators: []validator.Map{
				dataSourceAnnotationsValidator(),
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
				dataSourceLabelsValidator(),
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

// dataSourceMetadataBlock wraps the attributes in the list block shape SDKv2 produced,
// so existing state and configuration keep working.
func dataSourceMetadataBlock(objectName string, attributes map[string]schema.Attribute) schema.ListNestedBlock {
	return schema.ListNestedBlock{
		// "Exactly one" is in the description because tfplugindocs cannot see it
		// otherwise: framework blocks have no Required field, so required-ness lives in
		// the validators, which the docs generator does not read.
		Description: fmt.Sprintf("Standard %s's metadata. Exactly one metadata block is required. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata", objectName),
		Validators: []validator.List{
			listvalidator.IsRequired(),  // SDKv2 Required: true
			listvalidator.SizeAtMost(1), // SDKv2 MaxItems: 1
		},
		NestedObject: schema.NestedBlockObject{Attributes: attributes},
	}
}
