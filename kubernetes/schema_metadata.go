package kubernetes

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func metadataFields(objectName string) map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"annotations": {
			Type:         schema.TypeMap,
			Description:  fmt.Sprintf("An unstructured key value map stored with the %s that may be used to store arbitrary metadata. More info: http://kubernetes.io/docs/user-guide/annotations", objectName),
			Optional:     true,
			Elem:         &schema.Schema{Type: schema.TypeString},
			ValidateFunc: validateAnnotations,
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
			ValidateFunc: validateLabels,
		},
		"name": {
			Type:         schema.TypeString,
			Description:  fmt.Sprintf("Name of the %s, must be unique. Cannot be updated. More info: http://kubernetes.io/docs/user-guide/identifiers#names", objectName),
			Optional:     true,
			ForceNew:     true,
			Computed:     true,
			ValidateFunc: validateName,
		},
		"resource_version": {
			Type:        schema.TypeString,
			Description: fmt.Sprintf("An opaque value that represents the internal version of this %s that can be used by clients to determine when %s has changed. Read more: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#concurrency-control-and-consistency", objectName, objectName),
			Computed:    true,
		},
		"self_link": {
			Type:        schema.TypeString,
			Description: fmt.Sprintf("A URL representing this %s.", objectName),
			Computed:    true,
		},
		"uid": {
			Type:        schema.TypeString,
			Description: fmt.Sprintf("The unique in time and space value for this %s. More info: http://kubernetes.io/docs/user-guide/identifiers#uids", objectName),
			Computed:    true,
		},
	}
}

func metadataSchema(objectName string, generatableName bool) *schema.Schema {
	fields := metadataFields(objectName)

	if generatableName {
		fields["generate_name"] = &schema.Schema{
			Type:          schema.TypeString,
			Description:   "Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. Read more: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#idempotency",
			Optional:      true,
			ForceNew:      true,
			ValidateFunc:  validateGenerateName,
			ConflictsWith: []string{"metadata.name"},
		}
		fields["name"].ConflictsWith = []string{"metadata.generate_name"}
	}

	metadataRequired := true
	switch objectName {
	case "deploymentSpec":
		metadataRequired = false
	}

	return &schema.Schema{
		Type:        schema.TypeList,
		Description: fmt.Sprintf("Standard %s's metadata. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#metadata", objectName),
		Required:    metadataRequired,
		Optional:    !metadataRequired,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: fields,
		},
	}
}

func namespacedMetadataSchema(objectName string, generatableName bool) *schema.Schema {
	fields := metadataFields(objectName)
	fields["namespace"] = &schema.Schema{
		Type:        schema.TypeString,
		Description: fmt.Sprintf("Namespace defines the space within which name of the %s must be unique.", objectName),
		Optional:    true,
		ForceNew:    true,
		Default:     "default",
	}
	if generatableName {
		fields["generate_name"] = &schema.Schema{
			Type:          schema.TypeString,
			Description:   "Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. Read more: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#idempotency",
			Optional:      true,
			ForceNew:      true,
			ValidateFunc:  validateGenerateName,
			ConflictsWith: []string{"metadata.name"},
		}
		fields["name"].ConflictsWith = []string{"metadata.generate_name"}
	}

	return &schema.Schema{
		Type:        schema.TypeList,
		Description: fmt.Sprintf("Standard %s's metadata. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#metadata", objectName),
		Optional:    true,
		Computed:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: fields,
		},
	}
}

// reconcileTopLevelLabels removes metadata labels added to top level
// automatically by Kubernetes.
// If there are NO metadata labels defined in the top level resource,
// but there are some defined at the [Resource]->Spec->Template->Metadata
// level, the Kubernetes API will copy these to the top level.
// This will cause a perpetual diff for Terraform and break tests.
// We'll try to handle this situation by removing labels from the top
// level metadata if this described scenario is true
func reconcileTopLevelLabels(topLevelLabels map[string]string, topMeta, specMeta metav1.ObjectMeta) map[string]string {
	if len(topMeta.Labels) == 0 && len(topLevelLabels) > 0 {
		for k := range specMeta.Labels {
			delete(topLevelLabels, k)
		}
	}
	return topLevelLabels
}
