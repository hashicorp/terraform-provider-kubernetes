package v1

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	providerschema "github.com/hashicorp/terraform-provider-kubernetes/kubernetes/schema"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes/validators"
)

func MetadataFields(objectName string) map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"annotations": {
			Type:         schema.TypeMap,
			Description:  fmt.Sprintf("An unstructured key value map stored with the %s that may be used to store arbitrary metadata. More info: http://kubernetes.io/docs/user-guide/annotations", objectName),
			Optional:     true,
			Elem:         &schema.Schema{Type: schema.TypeString},
			ValidateFunc: validators.ValidateAnnotations,
		},
		"generation": {
			Type:        schema.TypeInt,
			Description: "A sequence number representing a specific generation of the desired state.",
			Computed:    true,
		},
		"labels": {
			Type:         schema.TypeMap,
			Description:  fmt.Sprintf("Map of string keys and values that can be used to organize and categorize (scope and select) the %s. May match selectors of replication controllers and services. More info: http://kubernetes.io/docs/user-guide/labels", objectName),
			Optional:     true,
			Elem:         &schema.Schema{Type: schema.TypeString},
			ValidateFunc: validators.ValidateLabels,
		},
		"name": {
			Type:         schema.TypeString,
			Description:  fmt.Sprintf("Name of the %s, must be unique. Cannot be updated. More info: http://kubernetes.io/docs/user-guide/identifiers#names", objectName),
			Optional:     true,
			ForceNew:     true,
			Computed:     true,
			ValidateFunc: validators.ValidateName,
		},
		"resource_version": {
			Type:        schema.TypeString,
			Description: fmt.Sprintf("An opaque value that represents the internal version of this %s that can be used by clients to determine when %s has changed. Read more: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency", objectName, objectName),
			Computed:    true,
		},
		"uid": {
			Type:        schema.TypeString,
			Description: fmt.Sprintf("The unique in time and space value for this %s. More info: http://kubernetes.io/docs/user-guide/identifiers#uids", objectName),
			Computed:    true,
		},
	}
}

func MetadataSchema(objectName string, generatableName bool) *schema.Schema {
	fields := MetadataFields(objectName)

	if generatableName {
		fields["generate_name"] = &schema.Schema{
			Type:          schema.TypeString,
			Description:   "Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. Read more: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency",
			Optional:      true,
			ForceNew:      true,
			ValidateFunc:  validators.ValidateGenerateName,
			ConflictsWith: []string{"metadata.0.name"},
		}
		fields["name"].ConflictsWith = []string{"metadata.0.generate_name"}
	}

	return &schema.Schema{
		Type:        schema.TypeList,
		Description: fmt.Sprintf("Standard %s's metadata. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata", objectName),
		Required:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: fields,
		},
	}
}

func MetadataSchemaForceNew(s *schema.Schema) *schema.Schema {
	s.ForceNew = true
	return s
}

func NamespacedMetadataSchema(objectName string, generatableName bool) *schema.Schema {
	return NamespacedMetadataSchemaIsTemplate(objectName, generatableName, false)
}

func NamespacedMetadataSchemaIsTemplate(objectName string, generatableName, isTemplate bool) *schema.Schema {
	fields := MetadataFields(objectName)
	fields["namespace"] = &schema.Schema{
		Type:        schema.TypeString,
		Description: fmt.Sprintf("Namespace defines the space within which name of the %s must be unique.", objectName),
		Optional:    true,
		ForceNew:    true,
		Default:     providerschema.ConditionalDefault(!isTemplate, "default"),
	}
	if generatableName {
		fields["generate_name"] = &schema.Schema{
			Type:          schema.TypeString,
			Description:   "Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. Read more: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency",
			Optional:      true,
			ForceNew:      true,
			ValidateFunc:  validators.ValidateGenerateName,
			ConflictsWith: []string{"metadata.name"},
		}
		fields["name"].ConflictsWith = []string{"metadata.generate_name"}
	}

	return &schema.Schema{
		Type:        schema.TypeList,
		Description: fmt.Sprintf("Standard %s's metadata. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata", objectName),
		Required:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: fields,
		},
	}
}
