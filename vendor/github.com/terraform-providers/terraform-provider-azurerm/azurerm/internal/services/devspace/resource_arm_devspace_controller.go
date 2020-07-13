package devspace

import (
	"fmt"
	"log"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/devspaces/mgmt/2019-04-01/devspaces"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/tf"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/validate"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/features"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/devspace/parse"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tags"
	azSchema "github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tf/schema"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func resourceArmDevSpaceController() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmDevSpaceControllerCreate,
		Read:   resourceArmDevSpaceControllerRead,
		Update: resourceArmDevSpaceControllerUpdate,
		Delete: resourceArmDevSpaceControllerDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Read:   schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
		},

		Importer: azSchema.ValidateResourceIDPriorToImport(func(id string) error {
			_, err := parse.DevSpaceControllerID(id)
			return err
		}),

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.DevSpaceName(),
			},

			"location": azure.SchemaLocation(),

			"resource_group_name": azure.SchemaResourceGroupName(),

			"sku_name": {
				Type:     schema.TypeString,
				Optional: true, // required in 2.0
				Computed: true, // remove in 2.0
				//ForceNew:      true, // uncomment in 2.0 - this should be fine as there is only 1 valid value
				ConflictsWith: []string{"sku"},
				ValidateFunc: validation.StringInSlice([]string{
					"S1",
				}, false),
			},

			"sku": {
				Type:     schema.TypeList,
				Optional: true,
				//ForceNew:      true, // uncomment in 2.0 - this should be fine as there is only 1 valid value
				Computed:      true,
				ConflictsWith: []string{"sku_name"},
				Deprecated:    "This property has been deprecated in favour of the 'sku_name' property and will be removed in version 2.0 of the provider",
				MaxItems:      1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								"S1",
							}, false),
						},
						"tier": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(devspaces.Standard),
							}, false),
						},
					},
				},
			},

			"target_container_host_resource_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: azure.ValidateResourceID,
			},

			"target_container_host_credentials_base64": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Sensitive:    true,
				ValidateFunc: validation.StringIsBase64,
			},

			"tags": tags.Schema(),

			"data_plane_fqdn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"host_suffix": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceArmDevSpaceControllerCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).DevSpace.ControllersClient
	ctx, cancel := timeouts.ForCreate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	log.Printf("[INFO] preparing arguments for DevSpace Controller creation")

	name := d.Get("name").(string)
	resourceGroup := d.Get("resource_group_name").(string)

	if features.ShouldResourcesBeImported() && d.IsNewResource() {
		existing, err := client.Get(ctx, resourceGroup, name)
		if err != nil {
			if !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("Error checking for presence of existing DevSpace Controller %q (Resource Group %q): %s", name, resourceGroup, err)
			}
		}

		if existing.ID != nil && *existing.ID != "" {
			return tf.ImportAsExistsError("azurerm_devspace_controller", *existing.ID)
		}
	}

	location := azure.NormalizeLocation(d.Get("location").(string))
	t := d.Get("tags").(map[string]interface{})

	var sku *devspaces.Sku
	if b, ok := d.GetOk("sku_name"); ok {
		var err error
		sku, err = expandControllerSkuName(b.(string))
		if err != nil {
			return fmt.Errorf("error expanding sku_name for DevSpace Controller %s (Resource Group %q): %v", name, resourceGroup, err)
		}
	} else if _, ok := d.GetOk("sku"); ok {
		sku = expandDevSpaceControllerSku(d)
	} else {
		return fmt.Errorf("One of `sku` or `sku_name` must be set for DevSpace Controller %q (Resource Group %q)", name, resourceGroup)
	}

	controller := devspaces.Controller{
		Location: &location,
		Sku:      sku,
		ControllerProperties: &devspaces.ControllerProperties{
			TargetContainerHostResourceID:        utils.String(d.Get("target_container_host_resource_id").(string)),
			TargetContainerHostCredentialsBase64: utils.String(d.Get("target_container_host_credentials_base64").(string)),
		},
		Tags: tags.Expand(t),
	}

	future, err := client.Create(ctx, resourceGroup, name, controller)
	if err != nil {
		return fmt.Errorf("Error creating DevSpace Controller %q (Resource Group %q): %+v", name, resourceGroup, err)
	}

	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("Error waiting for creation of DevSpace Controller %q (Resource Group %q): %+v", name, resourceGroup, err)
	}

	result, err := client.Get(ctx, resourceGroup, name)
	if err != nil {
		return fmt.Errorf("Error retrieving DevSpace %q (Resource Group %q): %+v", name, resourceGroup, err)
	}

	if result.ID == nil {
		return fmt.Errorf("Cannot read DevSpace Controller %q (Resource Group %q) ID", name, resourceGroup)
	}
	d.SetId(*result.ID)

	return resourceArmDevSpaceControllerRead(d, meta)
}

func resourceArmDevSpaceControllerUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).DevSpace.ControllersClient
	ctx, cancel := timeouts.ForUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	log.Printf("[INFO] preparing arguments for DevSpace Controller updating")

	id, err := parse.DevSpaceControllerID(d.Id())
	if err != nil {
		return err
	}
	params := devspaces.ControllerUpdateParameters{
		Tags: tags.Expand(d.Get("tags").(map[string]interface{})),
	}

	result, err := client.Update(ctx, id.ResourceGroup, id.Name, params)
	if err != nil {
		return fmt.Errorf("Error updating DevSpace Controller %q (Resource Group %q): %+v", id.Name, id.ResourceGroup, err)
	}

	if result.ID == nil {
		return fmt.Errorf("Cannot read DevSpace Controller %q (Resource Group %q) ID", id.Name, id.ResourceGroup)
	}
	d.SetId(*result.ID)

	return resourceArmDevSpaceControllerRead(d, meta)
}

func resourceArmDevSpaceControllerRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).DevSpace.ControllersClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.DevSpaceControllerID(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.Get(ctx, id.ResourceGroup, id.Name)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			log.Printf("[DEBUG] DevSpace Controller %q was not found in Resource Group %q - removing from state!", id.Name, id.ResourceGroup)
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error making Read request on DevSpace Controller %q (Resource Group %q): %+v", id.Name, id.ResourceGroup, err)
	}

	d.Set("name", id.Name)
	d.Set("resource_group_name", id.ResourceGroup)
	if location := resp.Location; location != nil {
		d.Set("location", azure.NormalizeLocation(*location))
	}

	if sku := resp.Sku; sku != nil {
		d.Set("sku_name", sku.Name)
	}

	if err := d.Set("sku", flattenDevSpaceControllerSku(resp.Sku)); err != nil {
		return fmt.Errorf("Error flattenning `sku`: %+v", err)
	}

	if props := resp.ControllerProperties; props != nil {
		d.Set("host_suffix", props.HostSuffix)
		d.Set("data_plane_fqdn", props.DataPlaneFqdn)
		d.Set("target_container_host_resource_id", props.TargetContainerHostResourceID)
	}

	return tags.FlattenAndSet(d, resp.Tags)
}

func resourceArmDevSpaceControllerDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).DevSpace.ControllersClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.DevSpaceControllerID(d.Id())
	if err != nil {
		return err
	}

	future, err := client.Delete(ctx, id.ResourceGroup, id.Name)
	if err != nil {
		return fmt.Errorf("Error deleting DevSpace Controller %q (Resource Group %q): %+v", id.Name, id.ResourceGroup, err)
	}

	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("Error waiting for the deletion of DevSpace Controller %q (Resource Group %q): %+v", id.Name, id.ResourceGroup, err)
	}

	return nil
}

func expandControllerSkuName(skuName string) (*devspaces.Sku, error) {
	var tier devspaces.SkuTier
	switch skuName[0:1] {
	case "S":
		tier = devspaces.Standard
	default:
		return nil, fmt.Errorf("sku_name %s has unknown sku tier %s", skuName, skuName[0:1])
	}

	return &devspaces.Sku{
		Name: utils.String(skuName),
		Tier: tier,
	}, nil
}

func expandDevSpaceControllerSku(d *schema.ResourceData) *devspaces.Sku {
	if _, ok := d.GetOk("sku"); !ok {
		return nil
	}

	skuConfigs := d.Get("sku").([]interface{})
	skuConfig := skuConfigs[0].(map[string]interface{})
	skuName := skuConfig["name"].(string)
	skuTier := devspaces.SkuTier(skuConfig["tier"].(string))

	return &devspaces.Sku{
		Name: &skuName,
		Tier: skuTier,
	}
}

func flattenDevSpaceControllerSku(skuObj *devspaces.Sku) []interface{} {
	if skuObj == nil {
		return []interface{}{}
	}

	skuConfig := make(map[string]interface{})
	if skuObj.Name != nil {
		skuConfig["name"] = *skuObj.Name
	}

	skuConfig["tier"] = skuObj.Tier

	return []interface{}{skuConfig}
}
