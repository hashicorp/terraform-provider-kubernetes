package apimanagement

import (
	"fmt"
	"log"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/apimanagement/mgmt/2018-01-01/apimanagement"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/tf"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/features"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func resourceArmApiManagementAuthorizationServer() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmApiManagementAuthorizationServerCreateUpdate,
		Read:   resourceArmApiManagementAuthorizationServerRead,
		Update: resourceArmApiManagementAuthorizationServerCreateUpdate,
		Delete: resourceArmApiManagementAuthorizationServerDelete,
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
			"name": azure.SchemaApiManagementChildName(),

			"api_management_name": azure.SchemaApiManagementName(),

			"resource_group_name": azure.SchemaResourceGroupName(),

			"authorization_endpoint": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},

			"authorization_methods": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						string(apimanagement.DELETE),
						string(apimanagement.GET),
						string(apimanagement.HEAD),
						string(apimanagement.OPTIONS),
						string(apimanagement.PATCH),
						string(apimanagement.POST),
						string(apimanagement.PUT),
						string(apimanagement.TRACE),
					}, false),
				},
				Set: schema.HashString,
			},

			"client_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},

			"client_registration_endpoint": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},

			"display_name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},

			"grant_types": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						string(apimanagement.AuthorizationCode),
						string(apimanagement.ClientCredentials),
						string(apimanagement.Implicit),
						string(apimanagement.ResourceOwnerPassword),
					}, false),
				},
				Set: schema.HashString,
			},

			// Optional
			"bearer_token_sending_methods": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						string(apimanagement.AuthorizationHeader),
						string(apimanagement.Query),
					}, false),
				},
				Set: schema.HashString,
			},

			"client_authentication_method": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						string(apimanagement.Basic),
						string(apimanagement.Body),
					}, false),
				},
				Set: schema.HashString,
			},

			"client_secret": {
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
			},

			"default_scope": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"resource_owner_username": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"resource_owner_password": {
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
			},

			"support_state": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"token_body_parameter": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringIsNotEmpty,
						},
						"value": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringIsNotEmpty,
						},
					},
				},
			},

			"token_endpoint": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceArmApiManagementAuthorizationServerCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).ApiManagement.AuthorizationServersClient
	ctx, cancel := timeouts.ForCreateUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	resourceGroup := d.Get("resource_group_name").(string)
	serviceName := d.Get("api_management_name").(string)
	name := d.Get("name").(string)

	if features.ShouldResourcesBeImported() && d.IsNewResource() {
		existing, err := client.Get(ctx, resourceGroup, serviceName, name)
		if err != nil {
			if !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("Error checking for presence of existing Authorization Server %q (API Management Service %q / Resource Group %q): %s", name, serviceName, resourceGroup, err)
			}
		}

		if existing.ID != nil && *existing.ID != "" {
			return tf.ImportAsExistsError("azurerm_api_management_authorization_server", *existing.ID)
		}
	}

	authorizationEndpoint := d.Get("authorization_endpoint").(string)
	clientId := d.Get("client_id").(string)
	clientRegistrationEndpoint := d.Get("client_registration_endpoint").(string)
	displayName := d.Get("display_name").(string)
	grantTypesRaw := d.Get("grant_types").(*schema.Set).List()
	grantTypes := expandApiManagementAuthorizationServerGrantTypes(grantTypesRaw)

	clientAuthenticationMethodsRaw := d.Get("client_authentication_method").(*schema.Set).List()
	clientAuthenticationMethods := expandApiManagementAuthorizationServerClientAuthenticationMethods(clientAuthenticationMethodsRaw)
	clientSecret := d.Get("client_secret").(string)
	defaultScope := d.Get("default_scope").(string)
	description := d.Get("description").(string)
	resourceOwnerPassword := d.Get("resource_owner_password").(string)
	resourceOwnerUsername := d.Get("resource_owner_username").(string)
	supportState := d.Get("support_state").(bool)
	tokenBodyParametersRaw := d.Get("token_body_parameter").([]interface{})
	tokenBodyParameters := expandApiManagementAuthorizationServerTokenBodyParameters(tokenBodyParametersRaw)

	params := apimanagement.AuthorizationServerContract{
		AuthorizationServerContractProperties: &apimanagement.AuthorizationServerContractProperties{
			// Required
			AuthorizationEndpoint:      utils.String(authorizationEndpoint),
			ClientID:                   utils.String(clientId),
			ClientRegistrationEndpoint: utils.String(clientRegistrationEndpoint),
			DisplayName:                utils.String(displayName),
			GrantTypes:                 grantTypes,

			// Optional
			ClientAuthenticationMethod: clientAuthenticationMethods,
			ClientSecret:               utils.String(clientSecret),
			DefaultScope:               utils.String(defaultScope),
			Description:                utils.String(description),
			ResourceOwnerPassword:      utils.String(resourceOwnerPassword),
			ResourceOwnerUsername:      utils.String(resourceOwnerUsername),
			SupportState:               utils.Bool(supportState),
			TokenBodyParameters:        tokenBodyParameters,
		},
	}

	authorizationMethodsRaw := d.Get("authorization_methods").(*schema.Set).List()
	if len(authorizationMethodsRaw) > 0 {
		authorizationMethods := expandApiManagementAuthorizationServerAuthorizationMethods(authorizationMethodsRaw)
		params.AuthorizationServerContractProperties.AuthorizationMethods = authorizationMethods
	}

	bearerTokenSendingMethodsRaw := d.Get("bearer_token_sending_methods").(*schema.Set).List()
	if len(bearerTokenSendingMethodsRaw) > 0 {
		bearerTokenSendingMethods := expandApiManagementAuthorizationServerBearerTokenSendingMethods(bearerTokenSendingMethodsRaw)
		params.AuthorizationServerContractProperties.BearerTokenSendingMethods = bearerTokenSendingMethods
	}

	if tokenEndpoint := d.Get("token_endpoint").(string); tokenEndpoint != "" {
		params.AuthorizationServerContractProperties.TokenEndpoint = utils.String(tokenEndpoint)
	}

	if _, err := client.CreateOrUpdate(ctx, resourceGroup, serviceName, name, params, ""); err != nil {
		return fmt.Errorf("Error creating/updating Authorization Server %q (API Management Service %q / Resource Group %q): %+v", name, serviceName, resourceGroup, err)
	}

	read, err := client.Get(ctx, resourceGroup, serviceName, name)
	if err != nil {
		return fmt.Errorf("Error retrieving Authorization Server %q (API Management Service %q / Resource Group %q): %+v", name, serviceName, resourceGroup, err)
	}

	if read.ID == nil {
		return fmt.Errorf("Cannot read ID for Authorization Server %q (API Management Service %q / Resource Group %q)", name, serviceName, resourceGroup)
	}

	d.SetId(*read.ID)
	return resourceArmApiManagementAuthorizationServerRead(d, meta)
}

func resourceArmApiManagementAuthorizationServerRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).ApiManagement.AuthorizationServersClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}

	resourceGroup := id.ResourceGroup
	serviceName := id.Path["service"]
	name := id.Path["authorizationServers"]

	resp, err := client.Get(ctx, resourceGroup, serviceName, name)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			log.Printf("[DEBUG] Authorization Server %q (API Management Service %q / Resource Group %q) does not exist - removing from state!", name, serviceName, resourceGroup)
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving Authorization Server %q (API Management Service %q / Resource Group %q): %+v", name, serviceName, resourceGroup, err)
	}

	d.Set("api_management_name", serviceName)
	d.Set("name", name)
	d.Set("resource_group_name", resourceGroup)

	if props := resp.AuthorizationServerContractProperties; props != nil {
		d.Set("authorization_endpoint", props.AuthorizationEndpoint)
		d.Set("client_id", props.ClientID)
		d.Set("client_registration_endpoint", props.ClientRegistrationEndpoint)
		d.Set("client_secret", props.ClientSecret)
		d.Set("default_scope", props.DefaultScope)
		d.Set("description", props.Description)
		d.Set("display_name", props.DisplayName)
		d.Set("resource_owner_password", props.ResourceOwnerPassword)
		d.Set("resource_owner_username", props.ResourceOwnerUsername)
		d.Set("support_state", props.SupportState)
		d.Set("token_endpoint", props.TokenEndpoint)

		if err := d.Set("authorization_methods", flattenApiManagementAuthorizationServerAuthorizationMethods(props.AuthorizationMethods)); err != nil {
			return fmt.Errorf("Error flattening `authorization_methods`: %+v", err)
		}

		if err := d.Set("bearer_token_sending_methods", flattenApiManagementAuthorizationServerBearerTokenSendingMethods(props.BearerTokenSendingMethods)); err != nil {
			return fmt.Errorf("Error flattening `bearer_token_sending_methods`: %+v", err)
		}

		if err := d.Set("client_authentication_method", flattenApiManagementAuthorizationServerClientAuthenticationMethods(props.ClientAuthenticationMethod)); err != nil {
			return fmt.Errorf("Error flattening `client_authentication_method`: %+v", err)
		}

		if err := d.Set("grant_types", flattenApiManagementAuthorizationServerGrantTypes(props.GrantTypes)); err != nil {
			return fmt.Errorf("Error flattening `grant_types`: %+v", err)
		}

		if err := d.Set("token_body_parameter", flattenApiManagementAuthorizationServerTokenBodyParameters(props.TokenBodyParameters)); err != nil {
			return fmt.Errorf("Error flattening `token_body_parameter`: %+v", err)
		}
	}

	return nil
}

func resourceArmApiManagementAuthorizationServerDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).ApiManagement.AuthorizationServersClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}

	resourceGroup := id.ResourceGroup
	serviceName := id.Path["service"]
	name := id.Path["authorizationServers"]

	if resp, err := client.Delete(ctx, resourceGroup, serviceName, name, ""); err != nil {
		if !utils.ResponseWasNotFound(resp) {
			return fmt.Errorf("Error deleting Authorization Server %q (API Management Service %q / Resource Group %q): %s", name, serviceName, resourceGroup, err)
		}
	}

	return nil
}

func expandApiManagementAuthorizationServerGrantTypes(input []interface{}) *[]apimanagement.GrantType {
	outputs := make([]apimanagement.GrantType, 0)

	for _, v := range input {
		grantType := apimanagement.GrantType(v.(string))
		outputs = append(outputs, grantType)
	}

	return &outputs
}

func flattenApiManagementAuthorizationServerGrantTypes(input *[]apimanagement.GrantType) []interface{} {
	outputs := make([]interface{}, 0)
	if input == nil {
		return outputs
	}

	for _, v := range *input {
		outputs = append(outputs, string(v))
	}

	return outputs
}

func expandApiManagementAuthorizationServerAuthorizationMethods(input []interface{}) *[]apimanagement.AuthorizationMethod {
	outputs := make([]apimanagement.AuthorizationMethod, 0)

	for _, v := range input {
		grantType := apimanagement.AuthorizationMethod(v.(string))
		outputs = append(outputs, grantType)
	}

	return &outputs
}

func flattenApiManagementAuthorizationServerAuthorizationMethods(input *[]apimanagement.AuthorizationMethod) []interface{} {
	outputs := make([]interface{}, 0)
	if input == nil {
		return outputs
	}

	for _, v := range *input {
		outputs = append(outputs, string(v))
	}

	return outputs
}

func expandApiManagementAuthorizationServerBearerTokenSendingMethods(input []interface{}) *[]apimanagement.BearerTokenSendingMethod {
	outputs := make([]apimanagement.BearerTokenSendingMethod, 0)

	for _, v := range input {
		grantType := apimanagement.BearerTokenSendingMethod(v.(string))
		outputs = append(outputs, grantType)
	}

	return &outputs
}

func flattenApiManagementAuthorizationServerBearerTokenSendingMethods(input *[]apimanagement.BearerTokenSendingMethod) []interface{} {
	outputs := make([]interface{}, 0)
	if input == nil {
		return outputs
	}

	for _, v := range *input {
		outputs = append(outputs, string(v))
	}

	return outputs
}

func expandApiManagementAuthorizationServerClientAuthenticationMethods(input []interface{}) *[]apimanagement.ClientAuthenticationMethod {
	outputs := make([]apimanagement.ClientAuthenticationMethod, 0)

	for _, v := range input {
		grantType := apimanagement.ClientAuthenticationMethod(v.(string))
		outputs = append(outputs, grantType)
	}

	return &outputs
}

func flattenApiManagementAuthorizationServerClientAuthenticationMethods(input *[]apimanagement.ClientAuthenticationMethod) []interface{} {
	outputs := make([]interface{}, 0)
	if input == nil {
		return outputs
	}

	for _, v := range *input {
		outputs = append(outputs, string(v))
	}

	return outputs
}

func expandApiManagementAuthorizationServerTokenBodyParameters(input []interface{}) *[]apimanagement.TokenBodyParameterContract {
	outputs := make([]apimanagement.TokenBodyParameterContract, 0)

	for _, v := range input {
		vs := v.(map[string]interface{})
		name := vs["name"].(string)
		value := vs["value"].(string)

		output := apimanagement.TokenBodyParameterContract{
			Name:  utils.String(name),
			Value: utils.String(value),
		}
		outputs = append(outputs, output)
	}

	return &outputs
}

func flattenApiManagementAuthorizationServerTokenBodyParameters(input *[]apimanagement.TokenBodyParameterContract) []interface{} {
	outputs := make([]interface{}, 0)
	if input == nil {
		return outputs
	}

	for _, v := range *input {
		output := make(map[string]interface{})

		if v.Name != nil {
			output["name"] = *v.Name
		}

		if v.Value != nil {
			output["value"] = *v.Value
		}

		outputs = append(outputs, output)
	}

	return outputs
}
