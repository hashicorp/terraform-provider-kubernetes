package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceKubernetesEnv() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesRoleCreate,
		ReadContext:   resourceKubernetesRoleRead,
		UpdateContext: resourceKubernetesRoleUpdate,
		DeleteContext: resourceKubernetesRoleDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "Name of Resource Environment",
				Required:    true,
				Elem:        schema.TypeString,
			},
			"api_version": {
				Type:        schema.TypeString,
				Description: "API Version of Field Manager",
				Required:    true,
				Elem:        schema.TypeString,
			},
			"kind": {
				Type:        schema.TypeString,
				Description: "Type of resource being used",
				Required:    true,
				Elem:        schema.TypeString,
			},
			"env": {
				Type:        schema.TypeList,
				Description: "Rule defining a set of permissions for the role",
				Required:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Name of the environment variable. Must be a C_IDENTIFIER",
						},
						"value": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: `Variable references $(VAR_NAME) are expanded using the previous defined environment variables in the container and any service environment variables. If a variable cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded, regardless of whether the variable exists or not. Defaults to "".`,
						},
					},
				},
			},
		},
	}
}
