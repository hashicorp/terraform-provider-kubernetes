package loganalytics

import (
	"fmt"
	"log"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/preview/operationalinsights/mgmt/2015-11-01-preview/operationalinsights"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/suppress"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/tf"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/features"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tags"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func resourceArmLogAnalyticsWorkspaceLinkedService() *schema.Resource {
	return &schema.Resource{
		DeprecationMessage: `The 'azurerm_log_analytics_workspace_linked_service' resource is deprecated in favour of the renamed version 'azurerm_log_analytics_linked_service'.

Information on migrating to the renamed resource can be found here: https://terraform.io/docs/providers/azurerm/guides/migrating-between-renamed-resources.html

As such the existing 'azurerm_log_analytics_workspace_linked_service' resource is deprecated and will be removed in the next major version of the AzureRM Provider (2.0).
`,

		Create: resourceArmLogAnalyticsWorkspaceLinkedServiceCreateUpdate,
		Read:   resourceArmLogAnalyticsWorkspaceLinkedServiceRead,
		Update: resourceArmLogAnalyticsWorkspaceLinkedServiceCreateUpdate,
		Delete: resourceArmLogAnalyticsWorkspaceLinkedServiceDelete,

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
			"resource_group_name": azure.SchemaResourceGroupNameDiffSuppress(),

			"workspace_name": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				DiffSuppressFunc: suppress.CaseDifference,
				ValidateFunc:     ValidateAzureRmLogAnalyticsWorkspaceName,
			},

			"linked_service_name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "automation",
				ValidateFunc: validation.StringInSlice([]string{
					"automation",
				}, false),
			},

			"resource_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ValidateFunc:  azure.ValidateResourceID,
				ConflictsWith: []string{"linked_service_properties.0"},
			},

			"linked_service_properties": {
				Type:          schema.TypeList,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"resource_id"},
				MaxItems:      1,
				Deprecated:    "This property has been deprecated in favour of the 'resource_id' property and will be removed in version 2.0 of the provider",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"resource_id": {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: azure.ValidateResourceID,
						},
					},
				},
			},

			// Exported properties
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tags.Schema(),
		},
	}
}

func resourceArmLogAnalyticsWorkspaceLinkedServiceCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).LogAnalytics.LinkedServicesClient
	ctx, cancel := timeouts.ForCreateUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	log.Printf("[INFO] preparing arguments for AzureRM Log Analytics Linked Services creation.")

	resGroup := d.Get("resource_group_name").(string)
	workspaceName := d.Get("workspace_name").(string)
	lsName := d.Get("linked_service_name").(string)

	if features.ShouldResourcesBeImported() && d.IsNewResource() {
		existing, err := client.Get(ctx, resGroup, workspaceName, lsName)
		if err != nil {
			if !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("Error checking for presence of existing Linked Service %q (Workspace %q / Resource Group %q): %s", lsName, workspaceName, resGroup, err)
			}
		}

		if existing.ID != nil && *existing.ID != "" {
			return tf.ImportAsExistsError("azurerm_log_analytics_workspace_linked_service", *existing.ID)
		}
	}

	resourceId := d.Get("resource_id").(string)
	if resourceId == "" {
		props := d.Get("linked_service_properties").([]interface{})
		resourceId = expandLogAnalyticsWorkspaceLinkedServiceProperties(props)
		if resourceId == "" {
			return fmt.Errorf("A `resource_id` must be specified either using the `resource_id` field at the top level or within the `linked_service_properties` block")
		}
	}
	t := d.Get("tags").(map[string]interface{})

	parameters := operationalinsights.LinkedService{
		LinkedServiceProperties: &operationalinsights.LinkedServiceProperties{
			ResourceID: utils.String(resourceId),
		},
		Tags: tags.Expand(t),
	}

	if _, err := client.CreateOrUpdate(ctx, resGroup, workspaceName, lsName, parameters); err != nil {
		return fmt.Errorf("Error creating Linked Service %q (Workspace %q / Resource Group %q): %+v", lsName, workspaceName, resGroup, err)
	}

	read, err := client.Get(ctx, resGroup, workspaceName, lsName)
	if err != nil {
		return fmt.Errorf("Error retrieving Linked Service %q (Worksppce %q / Resource Group %q): %+v", lsName, workspaceName, resGroup, err)
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read Linked Service %q (Workspace %q / Resource Group %q) ID", lsName, workspaceName, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmLogAnalyticsWorkspaceLinkedServiceRead(d, meta)
}

func resourceArmLogAnalyticsWorkspaceLinkedServiceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).LogAnalytics.LinkedServicesClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}

	resGroup := id.ResourceGroup
	workspaceName := id.Path["workspaces"]
	lsName := id.Path["linkedservices"]

	resp, err := client.Get(ctx, resGroup, workspaceName, lsName)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error making Read request on AzureRM Log Analytics Linked Service '%s': %+v", lsName, err)
	}

	d.Set("name", resp.Name)
	d.Set("resource_group_name", resGroup)
	d.Set("workspace_name", workspaceName)
	d.Set("linked_service_name", lsName)

	if props := resp.LinkedServiceProperties; props != nil {
		d.Set("resource_id", props.ResourceID)
	}

	linkedServiceProperties := flattenLogAnalyticsWorkspaceLinkedServiceProperties(resp.LinkedServiceProperties)
	if err := d.Set("linked_service_properties", linkedServiceProperties); err != nil {
		return fmt.Errorf("Error setting `linked_service_properties`: %+v", err)
	}

	return tags.FlattenAndSet(d, resp.Tags)
}

func resourceArmLogAnalyticsWorkspaceLinkedServiceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).LogAnalytics.LinkedServicesClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}

	resGroup := id.ResourceGroup
	workspaceName := id.Path["workspaces"]
	lsName := id.Path["linkedservices"]

	resp, err := client.Delete(ctx, resGroup, workspaceName, lsName)
	if err != nil {
		if utils.ResponseWasNotFound(resp) {
			return nil
		}

		return fmt.Errorf("Error deleting Linked Service %q (Workspace %q / Resource Group %q): %+v", lsName, workspaceName, resGroup, err)
	}

	return nil
}

func expandLogAnalyticsWorkspaceLinkedServiceProperties(input []interface{}) string {
	if len(input) == 0 {
		return ""
	}

	props := input[0].(map[string]interface{})
	return props["resource_id"].(string)
}

func flattenLogAnalyticsWorkspaceLinkedServiceProperties(input *operationalinsights.LinkedServiceProperties) []interface{} {
	if input == nil {
		return []interface{}{}
	}

	properties := make(map[string]interface{})

	// resource id linked service
	if resourceID := input.ResourceID; resourceID != nil {
		properties["resource_id"] = interface{}(*resourceID)
	}

	return []interface{}{properties}
}
