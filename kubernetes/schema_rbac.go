package kubernetes

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	pathValidation "k8s.io/apimachinery/pkg/api/validation/path"
)

func metadataSchemaClusterRole() *schema.Schema {
	m := metadataSchema("clusterRole", false)
	metadataFields := (m.Elem.(*schema.Resource)).Schema
	nameSchema := metadataFields["name"]
	if nameSchema != nil {
		nameSchema.ValidateFunc = validateRBACNameFunc
	}
	return m
}

func metadataSchemaRole() *schema.Schema {
	m := namespacedMetadataSchema("role", true)
	metadataFields := (m.Elem.(*schema.Resource)).Schema
	nameSchema := metadataFields["name"]
	if nameSchema != nil {
		nameSchema.ValidateFunc = validateRBACNameFunc
	}
	generateNameSchema := metadataFields["generate_name"]
	if generateNameSchema != nil {
		generateNameSchema.ValidateFunc = validateRBACNameFunc
	}
	return m
}

func validateRBACNameFunc(value interface{}, key string) (ws []string, es []error) {
	errors := pathValidation.IsValidPathSegmentName(value.(string))

	if len(errors) > 0 {
		for _, err := range errors {
			es = append(es, fmt.Errorf("%s %s", key, err))
		}
	}
	return
}

func rbacRoleRefSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"api_group": {
			Type:         schema.TypeString,
			Description:  "The API group of the user. The only value possible at the moment is `rbac.authorization.k8s.io`.",
			Required:     true,
			ValidateFunc: validation.StringInSlice([]string{"rbac.authorization.k8s.io"}, false),
		},
		"kind": {
			Type:         schema.TypeString,
			Description:  "The kind of resource.",
			Required:     true,
			ValidateFunc: validation.StringInSlice([]string{"Role", "ClusterRole"}, false),
		},
		"name": {
			Type:        schema.TypeString,
			Description: "The name of the User to bind to.",
			Required:    true,
		},
	}
}

func rbacSubjectSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"api_group": {
			Type:        schema.TypeString,
			Description: "The API group of the subject resource.",
			Optional:    true,
			Computed:    true,
		},
		"kind": {
			Type:        schema.TypeString,
			Description: "The kind of resource.",
			Required:    true,
		},
		"name": {
			Type:        schema.TypeString,
			Description: "The name of the resource to bind to.",
			Required:    true,
		},
		"namespace": {
			Type:        schema.TypeString,
			Description: "The Namespace of the subject resource.",
			Optional:    true,
			Default:     "default",
		},
	}
}

func policyRuleSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"api_groups": {
			Type:        schema.TypeList,
			Description: "APIGroups is the name of the APIGroup that contains the resources. If multiple API groups are specified, any action requested against one of the enumerated resources in any API group will be allowed.",
			Optional:    true,
			MinItems:    1,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"non_resource_urls": {
			Type:        schema.TypeList,
			Description: `NonResourceURLs is a set of partial urls that a user should have access to. *s are allowed, but only as the full, final step in the path Since non-resource URLs are not namespaced, this field is only applicable for ClusterRoles referenced from a ClusterRoleBinding. Rules can either apply to API resources (such as "pods" or "secrets") or non-resource URL paths (such as "/api"), but not both.`,
			Optional:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"resource_names": {
			Type:        schema.TypeList,
			Description: "ResourceNames is an optional white list of names that the rule applies to. An empty set means that everything is allowed.",
			Optional:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"resources": {
			Type:        schema.TypeList,
			Description: "Resources is a list of resources this rule applies to. ResourceAll represents all resources.",
			Optional:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"verbs": {
			Type:        schema.TypeList,
			Description: "Verbs is a list of Verbs that apply to ALL the ResourceKinds and AttributeRestrictions contained in this rule. VerbAll represents all kinds.",
			Required:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
	}
}
