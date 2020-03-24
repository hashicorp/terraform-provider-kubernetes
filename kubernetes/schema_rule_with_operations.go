package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func ruleWithOperationsFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"apiGroups": {
			Type:        schema.TypeList,
			Description: `APIGroups is the API groups the resources belong to. '\*' is all groups. If '\*' is present, the length of the slice must be one. Required.`,
			Required:    true,
			MinItems:    1,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"apiVersions": {
			Type:        schema.TypeList,
			Description: `APIVersions is the API versions the resources belong to. '\*' is all versions. If '\*' is present, the length of the slice must be one. Required.`,
			Required:    true,
			MinItems:    1,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"operations": {
			Type:        schema.TypeList,
			Description: `Operations is the operations the admission hook cares about - CREATE, UPDATE, or * for all operations. If '\*' is present, the length of the slice must be one. Required.`,
			Required:    true,
			MinItems:    1,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"resources": {
			Type:        schema.TypeList,
			Description: `Resources is a list of resources this rule applies to. For example: 'pods' means pods. 'pods/log' means the log subresource of pods. '\*' means all resources, but not subresources. 'pods/\*' means all subresources of pods. '\*/scale' means all scale subresources. '\*/\*' means all resources and their subresources. If wildcard is present, the validation rule will ensure resources do not overlap with each other. Depending on the enclosing object, subresources might not be allowed. Required.`,
			Required:    true,
			MinItems:    1,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"scope": {
			Type:        schema.TypeString,
			Description: `specifies the scope of this rule. Valid values are "Cluster", "Namespaced", and "*" "Cluster" means that only cluster-scoped resources will match this rule. Namespace API objects are cluster-scoped. "Namespaced" means that only namespaced resources will match this rule. "*" means that there are no scope restrictions. Subresources match the scope of their parent resource. Default is "*".`,
			Optional:    true,
			Default:     "*",
		},
	}
}
