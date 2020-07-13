package logic

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/structure"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
)

func resourceArmLogicAppActionCustom() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmLogicAppActionCustomCreateUpdate,
		Read:   resourceArmLogicAppActionCustomRead,
		Update: resourceArmLogicAppActionCustomCreateUpdate,
		Delete: resourceArmLogicAppActionCustomDelete,
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
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"logic_app_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: azure.ValidateResourceID,
			},

			"body": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateFunc:     validation.StringIsJSON,
				DiffSuppressFunc: structure.SuppressJsonDiff,
			},
		},
	}
}

func resourceArmLogicAppActionCustomCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	logicAppId := d.Get("logic_app_id").(string)
	name := d.Get("name").(string)
	bodyRaw := d.Get("body").(string)

	var body map[string]interface{}
	if err := json.Unmarshal([]byte(bodyRaw), &body); err != nil {
		return fmt.Errorf("Error unmarshalling JSON for Custom Action %q: %+v", name, err)
	}

	if err := resourceLogicAppActionUpdate(d, meta, logicAppId, name, body, "azurerm_logic_app_action_custom"); err != nil {
		return err
	}

	return resourceArmLogicAppActionCustomRead(d, meta)
}

func resourceArmLogicAppActionCustomRead(d *schema.ResourceData, meta interface{}) error {
	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}

	resourceGroup := id.ResourceGroup
	logicAppName := id.Path["workflows"]
	name := id.Path["actions"]

	t, app, err := retrieveLogicAppAction(d, meta, resourceGroup, logicAppName, name)
	if err != nil {
		return err
	}

	if t == nil {
		log.Printf("[DEBUG] Logic App %q (Resource Group %q) does not contain Action %q - removing from state", logicAppName, resourceGroup, name)
		d.SetId("")
		return nil
	}

	action := *t

	d.Set("name", name)
	d.Set("logic_app_id", app.ID)

	body, err := json.Marshal(action)
	if err != nil {
		return fmt.Errorf("Error serializing `body` for Action %q: %+v", name, err)
	}

	if err := d.Set("body", string(body)); err != nil {
		return fmt.Errorf("Error setting `body` for Action %q: %+v", name, err)
	}

	return nil
}

func resourceArmLogicAppActionCustomDelete(d *schema.ResourceData, meta interface{}) error {
	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}

	resourceGroup := id.ResourceGroup
	logicAppName := id.Path["workflows"]
	name := id.Path["actions"]

	err = resourceLogicAppActionRemove(d, meta, resourceGroup, logicAppName, name)
	if err != nil {
		return fmt.Errorf("Error removing Action %q from Logic App %q (Resource Group %q): %+v", name, logicAppName, resourceGroup, err)
	}

	return nil
}
