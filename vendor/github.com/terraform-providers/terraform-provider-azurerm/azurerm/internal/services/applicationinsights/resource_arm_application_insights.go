package applicationinsights

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/appinsights/mgmt/2015-05-01/insights"
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

func resourceArmApplicationInsights() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmApplicationInsightsCreateUpdate,
		Read:   resourceArmApplicationInsightsRead,
		Update: resourceArmApplicationInsightsCreateUpdate,
		Delete: resourceArmApplicationInsightsDelete,
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

			"resource_group_name": azure.SchemaResourceGroupName(),

			"location": azure.SchemaLocation(),

			"application_type": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				DiffSuppressFunc: suppress.CaseDifference,
				ValidateFunc: validation.StringInSlice([]string{
					"web",
					"other",
					"java",
					"MobileCenter",
					"phone",
					"store",
					"ios",
					"Node.JS",
				}, true),
			},

			"retention_in_days": {
				Type:     schema.TypeInt,
				Optional: true,
				ValidateFunc: validation.IntInSlice([]int{
					30,
					60,
					90,
					120,
					180,
					270,
					365,
					550,
					730,
				}),
			},

			"sampling_percentage": {
				Type:         schema.TypeFloat,
				Optional:     true,
				Default:      100,
				ValidateFunc: validation.FloatBetween(0, 100),
			},

			"tags": tags.Schema(),

			"daily_data_cap_in_gb": {
				Type:         schema.TypeFloat,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.FloatBetween(0, 1000),
			},

			"daily_data_cap_notifications_disabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"app_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"instrumentation_key": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
		},
	}
}

func resourceArmApplicationInsightsCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).AppInsights.ComponentsClient
	billingClient := meta.(*clients.Client).AppInsights.BillingClient
	ctx, cancel := timeouts.ForCreateUpdate(meta.(*clients.Client).StopContext, d)

	defer cancel()

	log.Printf("[INFO] preparing arguments for AzureRM Application Insights creation.")

	name := d.Get("name").(string)
	resGroup := d.Get("resource_group_name").(string)

	if features.ShouldResourcesBeImported() && d.IsNewResource() {
		existing, err := client.Get(ctx, resGroup, name)
		if err != nil {
			if !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("Error checking for presence of existing Application Insights %q (Resource Group %q): %s", name, resGroup, err)
			}
		}

		if existing.ID != nil && *existing.ID != "" {
			return tf.ImportAsExistsError("azurerm_application_insights", *existing.ID)
		}
	}

	applicationType := d.Get("application_type").(string)
	samplingPercentage := utils.Float(d.Get("sampling_percentage").(float64))
	location := azure.NormalizeLocation(d.Get("location").(string))
	t := d.Get("tags").(map[string]interface{})

	applicationInsightsComponentProperties := insights.ApplicationInsightsComponentProperties{
		ApplicationID:      &name,
		ApplicationType:    insights.ApplicationType(applicationType),
		SamplingPercentage: samplingPercentage,
	}

	if v, ok := d.GetOk("retention_in_days"); ok {
		applicationInsightsComponentProperties.RetentionInDays = utils.Int32(int32(v.(int)))
	}

	insightProperties := insights.ApplicationInsightsComponent{
		Name:                                   &name,
		Location:                               &location,
		Kind:                                   &applicationType,
		ApplicationInsightsComponentProperties: &applicationInsightsComponentProperties,
		Tags:                                   tags.Expand(t),
	}

	resp, err := client.CreateOrUpdate(ctx, resGroup, name, insightProperties)
	if err != nil {
		// @tombuildsstuff - from 2018-08-14 the Create call started returning a 201 instead of 200
		// which doesn't match the Swagger - this works around it until that's fixed
		// BUG: https://github.com/Azure/azure-sdk-for-go/issues/2465
		if resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("Error creating Application Insights %q (Resource Group %q): %+v", name, resGroup, err)
		}
	}

	read, err := client.Get(ctx, resGroup, name)
	if err != nil {
		return fmt.Errorf("Error retrieving Application Insights %q (Resource Group %q): %+v", name, resGroup, err)
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read AzureRM Application Insights '%s' (Resource Group %s) ID", name, resGroup)
	}

	billingRead, err := billingClient.Get(ctx, resGroup, name)
	if err != nil {
		return fmt.Errorf("Error read Application Insights Billing Features %q (Resource Group %q): %+v", name, resGroup, err)
	}

	applicationInsightsComponentBillingFeatures := insights.ApplicationInsightsComponentBillingFeatures{
		CurrentBillingFeatures: billingRead.CurrentBillingFeatures,
		DataVolumeCap:          billingRead.DataVolumeCap,
	}

	if v, ok := d.GetOk("daily_data_cap_in_gb"); ok {
		applicationInsightsComponentBillingFeatures.DataVolumeCap.Cap = utils.Float(v.(float64))
	}

	if v, ok := d.GetOk("daily_data_cap_notifications_disabled"); ok {
		applicationInsightsComponentBillingFeatures.DataVolumeCap.StopSendNotificationWhenHitCap = utils.Bool(v.(bool))
	}

	if _, err = billingClient.Update(ctx, resGroup, name, applicationInsightsComponentBillingFeatures); err != nil {
		return fmt.Errorf("Error update Application Insights Billing Feature %q (Resource Group %q): %+v", name, resGroup, err)
	}

	d.SetId(*read.ID)

	return resourceArmApplicationInsightsRead(d, meta)
}

func resourceArmApplicationInsightsRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).AppInsights.ComponentsClient
	billingClient := meta.(*clients.Client).AppInsights.BillingClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Reading AzureRM Application Insights '%s'", id)

	resGroup := id.ResourceGroup
	name := id.Path["components"]

	resp, err := client.Get(ctx, resGroup, name)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error making Read request on AzureRM Application Insights '%s': %+v", name, err)
	}

	billingResp, err := billingClient.Get(ctx, resGroup, name)
	if err != nil {
		return fmt.Errorf("Error making Read request on AzureRM Application Insights Billing Feature '%s': %+v", name, err)
	}

	d.Set("name", name)
	d.Set("resource_group_name", resGroup)
	if location := resp.Location; location != nil {
		d.Set("location", azure.NormalizeLocation(*location))
	}

	if props := resp.ApplicationInsightsComponentProperties; props != nil {
		d.Set("application_type", string(props.ApplicationType))
		d.Set("app_id", props.AppID)
		d.Set("instrumentation_key", props.InstrumentationKey)
		d.Set("sampling_percentage", props.SamplingPercentage)
		if v := props.RetentionInDays; v != nil {
			d.Set("retention_in_days", v)
		}
	}

	if billingProps := billingResp.DataVolumeCap; billingProps != nil {
		d.Set("daily_data_cap_in_gb", billingProps.Cap)
		d.Set("daily_data_cap_notifications_disabled", billingProps.StopSendNotificationWhenHitCap)
	}

	return tags.FlattenAndSet(d, resp.Tags)
}

func resourceArmApplicationInsightsDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).AppInsights.ComponentsClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["components"]

	log.Printf("[DEBUG] Deleting AzureRM Application Insights '%s' (resource group '%s')", name, resGroup)

	resp, err := client.Delete(ctx, resGroup, name)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return nil
		}
		return fmt.Errorf("Error issuing AzureRM delete request for Application Insights '%s': %+v", name, err)
	}

	return err
}
