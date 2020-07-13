package authorization

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/tf"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/features"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func resourceArmRoleDefinition() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmRoleDefinitionCreateUpdate,
		Read:   resourceArmRoleDefinitionRead,
		Update: resourceArmRoleDefinitionCreateUpdate,
		Delete: resourceArmRoleDefinitionDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Read:   schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"role_definition_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"scope": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"permissions": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"actions": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"not_actions": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"data_actions": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Set: schema.HashString,
						},
						"not_data_actions": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Set: schema.HashString,
						},
					},
				},
			},

			"assignable_scopes": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceArmRoleDefinitionCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Authorization.RoleDefinitionsClient
	ctx, cancel := timeouts.ForCreateUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	roleDefinitionId := d.Get("role_definition_id").(string)
	if roleDefinitionId == "" {
		uuid, err := uuid.GenerateUUID()
		if err != nil {
			return fmt.Errorf("Error generating UUID for Role Assignment: %+v", err)
		}

		roleDefinitionId = uuid
	}

	name := d.Get("name").(string)
	scope := d.Get("scope").(string)
	description := d.Get("description").(string)
	roleType := "CustomRole"
	permissions := expandRoleDefinitionPermissions(d)
	assignableScopes := expandRoleDefinitionAssignableScopes(d)

	if features.ShouldResourcesBeImported() && d.IsNewResource() {
		existing, err := client.Get(ctx, scope, roleDefinitionId)
		if err != nil {
			if !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("Error checking for presence of existing Role Definition ID for %q (Scope %q)", name, scope)
			}
		}

		if existing.ID != nil && *existing.ID != "" {
			return tf.ImportAsExistsError("azurerm_role_definition", *existing.ID)
		}
	}

	properties := authorization.RoleDefinition{
		RoleDefinitionProperties: &authorization.RoleDefinitionProperties{
			RoleName:         utils.String(name),
			Description:      utils.String(description),
			RoleType:         utils.String(roleType),
			Permissions:      &permissions,
			AssignableScopes: &assignableScopes,
		},
	}

	if _, err := client.CreateOrUpdate(ctx, scope, roleDefinitionId, properties); err != nil {
		return err
	}

	read, err := client.Get(ctx, scope, roleDefinitionId)
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read Role Definition ID for %q (Scope %q)", name, scope)
	}

	d.SetId(*read.ID)
	return resourceArmRoleDefinitionRead(d, meta)
}

func resourceArmRoleDefinitionRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Authorization.RoleDefinitionsClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	resp, err := client.GetByID(ctx, d.Id())
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			log.Printf("[DEBUG] Role Definition %q was not found - removing from state", d.Id())
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error loading Role Definition %q: %+v", d.Id(), err)
	}

	if id := resp.ID; id != nil {
		roleDefinitionId, err := parseRoleDefinitionId(*id)
		if err != nil {
			return fmt.Errorf("Error parsing Role Definition ID: %+v", err)
		}
		if roleDefinitionId != nil {
			d.Set("role_definition_id", roleDefinitionId.roleDefinitionId)
		}
	}

	if props := resp.RoleDefinitionProperties; props != nil {
		d.Set("name", props.RoleName)
		d.Set("description", props.Description)

		permissions := flattenRoleDefinitionPermissions(props.Permissions)
		if err := d.Set("permissions", permissions); err != nil {
			return err
		}

		assignableScopes := flattenRoleDefinitionAssignableScopes(props.AssignableScopes)
		if err := d.Set("assignable_scopes", assignableScopes); err != nil {
			return err
		}
	}

	return nil
}

func resourceArmRoleDefinitionDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Authorization.RoleDefinitionsClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parseRoleDefinitionId(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.Delete(ctx, id.scope, id.roleDefinitionId)
	if err != nil {
		if !utils.ResponseWasNotFound(resp.Response) {
			return fmt.Errorf("Error deleting Role Definition %q at Scope %q: %+v", id.roleDefinitionId, id.scope, err)
		}
	}

	return nil
}

func expandRoleDefinitionPermissions(d *schema.ResourceData) []authorization.Permission {
	output := make([]authorization.Permission, 0)

	permissions := d.Get("permissions").([]interface{})
	for _, v := range permissions {
		input := v.(map[string]interface{})
		permission := authorization.Permission{}

		actionsOutput := make([]string, 0)
		actions := input["actions"].([]interface{})
		for _, a := range actions {
			actionsOutput = append(actionsOutput, a.(string))
		}
		permission.Actions = &actionsOutput

		dataActionsOutput := make([]string, 0)
		dataActions := input["data_actions"].(*schema.Set)
		for _, a := range dataActions.List() {
			dataActionsOutput = append(dataActionsOutput, a.(string))
		}
		permission.DataActions = &dataActionsOutput

		notActionsOutput := make([]string, 0)
		notActions := input["not_actions"].([]interface{})
		for _, a := range notActions {
			notActionsOutput = append(notActionsOutput, a.(string))
		}
		permission.NotActions = &notActionsOutput

		notDataActionsOutput := make([]string, 0)
		notDataActions := input["not_data_actions"].(*schema.Set)
		for _, a := range notDataActions.List() {
			notDataActionsOutput = append(notDataActionsOutput, a.(string))
		}
		permission.NotDataActions = &notDataActionsOutput

		output = append(output, permission)
	}

	return output
}

func expandRoleDefinitionAssignableScopes(d *schema.ResourceData) []string {
	scopes := make([]string, 0)

	assignableScopes := d.Get("assignable_scopes").([]interface{})
	for _, scope := range assignableScopes {
		scopes = append(scopes, scope.(string))
	}

	return scopes
}

func flattenRoleDefinitionPermissions(input *[]authorization.Permission) []interface{} {
	permissions := make([]interface{}, 0)
	if input == nil {
		return permissions
	}

	for _, permission := range *input {
		output := make(map[string]interface{})

		actions := make([]string, 0)
		if s := permission.Actions; s != nil {
			actions = *s
		}
		output["actions"] = actions

		dataActions := make([]interface{}, 0)
		if permission.DataActions != nil {
			for _, dataAction := range *permission.DataActions {
				dataActions = append(dataActions, dataAction)
			}
		}
		output["data_actions"] = schema.NewSet(schema.HashString, dataActions)

		notActions := make([]string, 0)
		if s := permission.NotActions; s != nil {
			notActions = *s
		}
		output["not_actions"] = notActions

		notDataActions := make([]interface{}, 0)
		if permission.NotDataActions != nil {
			for _, dataAction := range *permission.NotDataActions {
				notDataActions = append(notDataActions, dataAction)
			}
		}
		output["not_data_actions"] = schema.NewSet(schema.HashString, notDataActions)

		permissions = append(permissions, output)
	}

	return permissions
}

func flattenRoleDefinitionAssignableScopes(input *[]string) []interface{} {
	scopes := make([]interface{}, 0)
	if input == nil {
		return scopes
	}

	for _, scope := range *input {
		scopes = append(scopes, scope)
	}

	return scopes
}

type roleDefinitionId struct {
	scope            string
	roleDefinitionId string
}

func parseRoleDefinitionId(input string) (*roleDefinitionId, error) {
	segments := strings.Split(input, "/providers/Microsoft.Authorization/roleDefinitions/")
	if len(segments) != 2 {
		return nil, fmt.Errorf("Expected Role Definition ID to be in the format `{scope}/providers/Microsoft.Authorization/roleDefinitions/{name}` but got %q", input)
	}

	// /{scope}/providers/Microsoft.Authorization/roleDefinitions/{roleDefinitionId}
	id := roleDefinitionId{
		scope:            strings.TrimPrefix(segments[0], "/"),
		roleDefinitionId: segments[1],
	}
	return &id, nil
}
