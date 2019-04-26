package kubernetes

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func customResourceDefinitionSpecFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"group": {
			Type:        schema.TypeString,
			Description: "Group is the group this resource belongs in",
			Required:    true,
		},
		"version": {
			Type:        schema.TypeString,
			Description: "Version is the version this resource belongs in Should be always first item in Versions field if provided. Optional, but at least one of Version or Versions must be set. Deprecated: Please use `Versions`.",
			Optional:    true,
		},
		"names": { // https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.10/#customresourcedefinitionnames-v1beta1-apiextensions-k8s-io
			Type:        schema.TypeList,
			Description: "Names are the names used to describe this custom resource",
			Required:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: customResourceDefinitionNamesFields(),
			},
		},
		"scope": {
			Type:         schema.TypeString,
			Description:  "Scope indicates whether this resource is cluster or namespace scoped.  Default is namespaced",
			Optional:     true,
			Default:      "Namespaced",
			ValidateFunc: validation.StringInSlice([]string{"Cluster", "Namespaced"}, false),
		},
		// Intentionally omitting "validation" field; it contains a JSONSchema field that forces
		// a recursive schema, but https://github.com/hashicorp/terraform/issues/18616 says
		// Terraform does not support recursive schemas
		"subresources": {
			Type:        schema.TypeList,
			Description: "Subresources describes the subresources for CustomResource",
			Optional:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: customResourceSubresourcesFields(),
			},
		},
		"versions": {
			Type:        schema.TypeList,
			Description: "Versions is the list of all supported versions for this resource.",
			Optional:    true,
			Elem: &schema.Resource{
				Schema: customResourceDefinitionVersionFields(),
			},
		},
		// "additional_printer_column": {
		// 	Type:        schema.TypeList,
		// 	Description: "AdditionalPrinterColumns are additional columns shown e.g. in kubectl next to the name. Defaults to a created-at column. Optional, the global columns for all versions. Top-level and per-version columns are mutually exclusive.",
		// 	Optional:    true,
		// 	Elem: &schema.Resource{
		// 		Schema: customResourceColumnDefinitionFields(),
		// 	},
		// },
		// "conversion": {
		// 	Type:        schema.TypeList,
		// 	Description: "Names are the names used to describe this custom resource",
		// 	Optional:    true,
		// 	MaxItems:    1,
		// 	Elem: &schema.Resource{
		// 		Schema: customResourceConversionFields(),
		// 	},
		// },
	}
}

// func customResourceConversionFields() map[string]*schema.Schema {
// 	return map[string]*schema.Schema{
// 		"strategy": {
// 			Type:         schema.TypeString,
// 			Description:  "`strategy` specifies the conversion strategy.",
// 			Optional:     true,
// 			Default:      "None",
// 			ValidateFunc: validation.StringInSlice([]string{"None", "Webhook"}, false),
// 		},
// 		"webhook_client_config": {
// 			Type:        schema.TypeList,
// 			Description: "`webhookClientConfig` is the instructions for how to call the webhook if strategy is `Webhook`. This field is alpha-level and is only honored by servers that enable the CustomResourceWebhookConversion feature.",
// 			Optional:    true,
// 			Elem: &schema.Resource{
// 				Schema: map[string]*schema.Schema{
// 					"url": {
// 						Type:        schema.TypeString,
// 						Description: "`url` gives the location of the webhook, in standard URL form (`scheme://host:port/path`). Exactly one of `url` or `service` must be specified.",
// 						Optional:    true,
// 					},
// 					"service": {
// 						Type:        schema.TypeList,
// 						Description: "A list of label selector requirements. The requirements are ANDed.",
// 						Optional:    true,
// 						ForceNew:    true,
// 						MaxItems:    1,
// 						Elem: &schema.Resource{
// 							Schema: map[string]*schema.Schema{
// 								"name": {
// 									Type:        schema.TypeString,
// 									Description: "`name` is the name of the service. Required",
// 									Required:    true,
// 								},
// 								"namespace": {
// 									Type:        schema.TypeString,
// 									Description: "`namespace` is the namespace of the service. Required",
// 									Required:    true,
// 								},
// 								"path": {
// 									Type:        schema.TypeString,
// 									Description: "`path` is an optional URL path which will be sent in any request to this service.",
// 									Optional:    true,
// 								},
// 							},
// 						},
// 					},
// 					"ca_bundle": {
// 						Type:        schema.TypeString,
// 						Description: "`caBundle` is a PEM encoded CA bundle which will be used to validate the webhook's server certificate. If unspecified, system trust roots on the apiserver are used.",
// 						Optional:    true,
// 					},
// 				},
// 			},
// 		},
// 		// "conversion_review_versions": {
// 		// 	Type:        schema.TypeList,
// 		// 	Description: "ConversionReviewVersions is an ordered list of preferred `ConversionReview` versions the Webhook expects. API server will try to use first version in the list which it supports. If none of the versions specified in this list supported by API server, conversion will fail for this object.",
// 		// 	Optional:    true,
// 		// 	Elem: &schema.Schema{
// 		// 		Type: schema.TypeString,
// 		// 	},
// 		// },
// 	}
// }

func customResourceDefinitionNamesFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"plural": {
			Type:         schema.TypeString,
			Description:  "Plural is the plural name of the resource to serve. It must match the name of the CustomResourceDefinition-registration too: plural.group and it must be all lowercase.",
			Required:     true,
			ValidateFunc: validation.StringMatch(regexp.MustCompile(`^([a-z]+)$`), ""),
		},
		"singular": {
			Type:         schema.TypeString,
			Description:  "Singular is the singular name of the resource. It must be all lowercase Defaults to lowercased <kind>",
			Optional:     true,
			ValidateFunc: validation.StringMatch(regexp.MustCompile(`^([a-z]+)$`), ""),
		},
		"short_names": {
			Type:        schema.TypeList,
			Description: "ShortNames are short names for the resource. It must be all lowercase.",
			Optional:    true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"kind": {
			Type:        schema.TypeString,
			Description: "Kind is the serialized kind of the resource. It is normally CamelCase and singular.",
			Required:    true,
		},
		"list_kind": {
			Type:        schema.TypeString,
			Description: "ListKind is the serialized kind of the list for this resource. Defaults to <kind>List.",
			Optional:    true,
		},
		"categories": {
			Type:        schema.TypeList,
			Description: "Categories is a list of grouped resources custom resources belong to (e.g. 'all')",
			Optional:    true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
	}
}

func customResourceSubresourcesFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"scale": {
			Type:        schema.TypeList,
			Description: "Scale denotes the scale subresource for CustomResources",
			Optional:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"spec_replicas_path": {
						Type:        schema.TypeString,
						Description: "SpecReplicasPath defines the JSON path inside of a CustomResource that corresponds to Scale.Spec.Replicas. Only JSON paths without the array notation are allowed. Must be a JSON Path under .spec. If there is no value under the given path in the CustomResource, the /scale subresource will return an error on GET.",
						Required:    true,
					},
					"status_replicas_path": {
						Type:        schema.TypeString,
						Description: "StatusReplicasPath defines the JSON path inside of a CustomResource that corresponds to Scale.Status.Replicas. Only JSON paths without the array notation are allowed. Must be a JSON Path under .status. If there is no value under the given path in the CustomResource, the status replica value in the /scale subresource will default to 0.",
						Required:    true,
					},
					"label_selector_path": {
						Type:        schema.TypeString,
						Description: "LabelSelectorPath defines the JSON path inside of a CustomResource that corresponds to Scale.Status.Selector. Only JSON paths without the array notation are allowed. Must be a JSON Path under .status. Must be set to work with HPA. If there is no value under the given path in the CustomResource, the status label selector value in the /scale subresource will default to the empty string.",
						Optional:    true,
					},
				},
			},
		},
	}
}

func customResourceColumnDefinitionFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"json_path": {
			Type:        schema.TypeString,
			Description: "JSONPath is a simple JSON path, i.e. with array notation.",
			Required:    true,
		},
		"description": {
			Type:        schema.TypeString,
			Description: "description is a human readable description of this column.",
			Optional:    true,
		},
		"format": {
			Type:        schema.TypeString,
			Description: "format is an optional OpenAPI type definition for this column. The 'name' format is applied to the primary identifier column to assist in clients identifying column is the resource name. See https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#data-types for more.",
			Optional:    true,
		},
		"name": {
			Type:        schema.TypeString,
			Description: "name is a human readable name for the column.",
			Required:    true,
		},
		"priority": {
			Type:        schema.TypeInt,
			Description: "priority is an integer defining the relative importance of this column compared to others. Lower numbers are considered higher priority. Columns that may be omitted in limited space scenarios should be given a higher priority.",
			Optional:    true,
		},
		"type": {
			Type:        schema.TypeString,
			Description: "type is an OpenAPI type definition for this column. See https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#data-types for more.",
			Required:    true,
		},
	}
}

func customResourceDefinitionVersionFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name": {
			Type:        schema.TypeString,
			Description: "Name is the version name, e.g. “v1”, “v2beta1”, etc.",
			Required:    true,
		},
		"served": {
			Type:        schema.TypeBool,
			Description: "Served is a flag enabling/disabling this version from being served via REST APIs",
			Optional:    true,
			Default:     true,
		},
		"storage": {
			Type:        schema.TypeBool,
			Description: "Storage flags the version as storage version. There must be exactly one flagged as storage version.",
			Optional:    true,
			Default:     true,
		},
		// Intentionally skipping "schema" field; it contains a JSONSchema field that forces
		// a recursive schema, but https://github.com/hashicorp/terraform/issues/18616 says
		// Terraform does not support recursive schemas
		"subresources": {
			Type:        schema.TypeList,
			Description: "Subresources describes the subresources for CustomResource",
			Optional:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: customResourceSubresourcesFields(),
			},
		},
		// "additional_printer_column": {
		// 	Type:        schema.TypeList,
		// 	Description: "AdditionalPrinterColumns are additional columns shown e.g. in kubectl next to the name. Defaults to a created-at column.",
		// 	Optional:    true,
		// 	Elem: &schema.Resource{
		// 		Schema: customResourceColumnDefinitionFields(),
		// 	},
		// },
	}
}
