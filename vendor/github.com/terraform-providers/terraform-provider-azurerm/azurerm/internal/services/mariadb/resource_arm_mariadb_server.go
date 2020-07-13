package mariadb

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/mariadb/mgmt/2018-06-01/mariadb"
	"github.com/hashicorp/go-azure-helpers/response"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/tf"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/features"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tags"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func resourceArmMariaDbServer() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmMariaDbServerCreateUpdate,
		Read:   resourceArmMariaDbServerRead,
		Update: resourceArmMariaDbServerCreateUpdate,
		Delete: resourceArmMariaDbServerDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(60 * time.Minute),
			Read:   schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(60 * time.Minute),
			Delete: schema.DefaultTimeout(60 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringMatch(
					regexp.MustCompile("^[-a-zA-Z0-9]{3,50}$"),
					"MariaDB server name must be 3 - 50 characters long, contain only letters, numbers and hyphens.",
				),
			},

			"location": azure.SchemaLocation(),

			"resource_group_name": azure.SchemaResourceGroupName(),

			"sku_name": {
				Type:          schema.TypeString,
				Optional:      true, // required in 2.0
				Computed:      true, // remove in 2.0
				ConflictsWith: []string{"sku"},
				ValidateFunc: validation.StringInSlice([]string{
					"B_Gen5_1",
					"B_Gen5_2",
					"GP_Gen5_2",
					"GP_Gen5_4",
					"GP_Gen5_8",
					"GP_Gen5_16",
					"GP_Gen5_32",
					"MO_Gen5_2",
					"MO_Gen5_4",
					"MO_Gen5_8",
					"MO_Gen5_16",
				}, false),
			},

			// remove in 2.0
			"sku": {
				Type:          schema.TypeList,
				Optional:      true,
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
								"B_Gen5_1",
								"B_Gen5_2",
								"GP_Gen5_2",
								"GP_Gen5_4",
								"GP_Gen5_8",
								"GP_Gen5_16",
								"GP_Gen5_32",
								"MO_Gen5_2",
								"MO_Gen5_4",
								"MO_Gen5_8",
								"MO_Gen5_16",
							}, false),
						},

						"capacity": {
							Type:     schema.TypeInt,
							Required: true,
							ValidateFunc: validation.IntInSlice([]int{
								1,
								2,
								4,
								8,
								16,
								32,
							}),
						},

						"tier": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(mariadb.Basic),
								string(mariadb.GeneralPurpose),
								string(mariadb.MemoryOptimized),
							}, false),
						},

						"family": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								"Gen5",
							}, false),
						},
					},
				},
			},

			"administrator_login": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},

			"administrator_login_password": {
				Type:         schema.TypeString,
				Required:     true,
				Sensitive:    true,
				ValidateFunc: validation.StringIsNotEmpty,
			},

			"version": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					"10.2",
					"10.3",
				}, false),
			},

			"storage_profile": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"storage_mb": {
							Type:     schema.TypeInt,
							Required: true,
							ValidateFunc: validation.All(
								validation.IntBetween(5120, 4096000),
								validation.IntDivisibleBy(1024),
							),
						},

						"backup_retention_days": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validation.IntBetween(7, 35),
						},

						"geo_redundant_backup": {
							Type:     schema.TypeString,
							Optional: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(mariadb.Enabled),
								string(mariadb.Disabled),
							}, false),
						},

						"auto_grow": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  string(mariadb.StorageAutogrowEnabled),
							ValidateFunc: validation.StringInSlice([]string{
								string(mariadb.StorageAutogrowEnabled),
								string(mariadb.StorageAutogrowDisabled),
							}, false),
						},
					},
				},
			},

			"ssl_enforcement": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					string(mariadb.SslEnforcementEnumDisabled),
					string(mariadb.SslEnforcementEnumEnabled),
				}, false),
			},

			"fqdn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tags.Schema(),
		},
	}
}

func resourceArmMariaDbServerCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).MariaDB.ServersClient
	ctx, cancel := timeouts.ForCreateUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	log.Printf("[INFO] preparing arguments for AzureRM MariaDB Server creation.")

	name := d.Get("name").(string)
	resourceGroup := d.Get("resource_group_name").(string)

	if features.ShouldResourcesBeImported() && d.IsNewResource() {
		existing, err := client.Get(ctx, resourceGroup, name)
		if err != nil {
			if !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("Error checking for presence of existing MariaDB Server %q (Resource Group %q): %s", name, resourceGroup, err)
			}
		}

		if existing.ID != nil && *existing.ID != "" {
			return tf.ImportAsExistsError("azurerm_mariadb_server", *existing.ID)
		}
	}

	location := azure.NormalizeLocation(d.Get("location").(string))

	storageProfile := expandAzureRmMariaDbStorageProfile(d)

	var sku *mariadb.Sku
	if b, ok := d.GetOk("sku_name"); ok {
		var err error
		sku, err = expandServerSkuName(b.(string))
		if err != nil {
			return fmt.Errorf("error expanding sku_name for PostgreSQL Server %q (Resource Group %q): %v", name, resourceGroup, err)
		}
	} else if _, ok := d.GetOk("sku"); ok {
		sku = expandAzureRmMariaDbServerSku(d)
	} else {
		return fmt.Errorf("One of `sku` or `sku_name` must be set for PostgreSQL Server %q (Resource Group %q)", name, resourceGroup)
	}

	skuName := sku.Name
	capacity := sku.Capacity
	tier := string(sku.Tier)
	storageMB := storageProfile.StorageMB

	// General sku validation for all sku's
	if !strings.HasSuffix(*skuName, strconv.Itoa(int(*capacity))) {
		return fmt.Errorf("the value in the capacity property must match the capacity value defined in the sku name (sku.capacity: %d, sku.name: %s)", *capacity, *skuName)
	}

	// Specific validation based on sku's pricing tier
	// Basic
	if strings.ToLower(tier) == "basic" {
		if !strings.HasPrefix(*skuName, "B_") {
			return fmt.Errorf("the basic pricing tier sku name must begin with the letter B (sku.name: %s)", *skuName)
		}

		if *storageMB > 1024000 {
			return fmt.Errorf("basic pricing tier only supports upto 1,024,000 MB (1TB) of storage (storageProfile.StorageMB: %d)", *storageMB)
		}

		if *capacity > 2 {
			return fmt.Errorf("basic pricing tier only supports upto 2 vCores (sku.capacity: %d)", *capacity)
		}
	}

	// General Purpose
	if strings.ToLower(tier) == "generalpurpose" {
		if !strings.HasPrefix(*skuName, "GP_") {
			return fmt.Errorf("the general purpose pricing tier sku name must begin with the letters GP (sku.name: %s)", *skuName)
		}

		if *capacity < 2 {
			return fmt.Errorf("general purpose pricing tier must have at least 2 vCores (sku.capacity: %d)", *capacity)
		}
	}

	// Memory Optimized
	if strings.ToLower(tier) == "memoryoptimized" {
		if !strings.HasPrefix(*skuName, "MO_") {
			return fmt.Errorf("the memory optimized pricing tier sku name must begin with the letters MO (sku.name: %s)", *skuName)
		}

		if *capacity < 2 {
			return fmt.Errorf("memory optimized pricing tier must have at least 2 vCores (sku.capacity: %d)", *capacity)
		}

		if *capacity > 16 {
			return fmt.Errorf("memory optimized pricing tier only supports upto 16 vCores (sku.capacity: %d)", *capacity)
		}
	}

	properties := mariadb.ServerForCreate{
		Location: &location,
		Properties: &mariadb.ServerPropertiesForDefaultCreate{
			AdministratorLogin:         utils.String(d.Get("administrator_login").(string)),
			AdministratorLoginPassword: utils.String(d.Get("administrator_login_password").(string)),
			Version:                    mariadb.ServerVersion(d.Get("version").(string)),
			SslEnforcement:             mariadb.SslEnforcementEnum(d.Get("ssl_enforcement").(string)),
			StorageProfile:             storageProfile,
			CreateMode:                 mariadb.CreateModeDefault,
		},
		Sku:  sku,
		Tags: tags.Expand(d.Get("tags").(map[string]interface{})),
	}

	future, err := client.Create(ctx, resourceGroup, name, properties)
	if err != nil {
		return fmt.Errorf("Error creating MariaDB Server %q (Resource Group %q): %+v", name, resourceGroup, err)
	}

	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("Error waiting for creation of MariaDB Server %q (Resource Group %q): %+v", name, resourceGroup, err)
	}

	read, err := client.Get(ctx, resourceGroup, name)
	if err != nil {
		return fmt.Errorf("Error retrieving MariaDB Server %q (Resource Group %q): %+v", name, resourceGroup, err)
	}

	if read.ID == nil {
		return fmt.Errorf("Cannot read MariaDB Server %q (Resource Group %q) ID", name, resourceGroup)
	}

	d.SetId(*read.ID)

	return resourceArmMariaDbServerRead(d, meta)
}

func resourceArmMariaDbServerRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).MariaDB.ServersClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resourceGroup := id.ResourceGroup
	name := id.Path["servers"]

	resp, err := client.Get(ctx, resourceGroup, name)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			log.Printf("[WARN] MariaDB Server %q was not found (Resource Group %q)", name, resourceGroup)
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error making Read request on Azure MariaDB Server %q (Resource Group %q): %+v", name, resourceGroup, err)
	}

	d.Set("name", resp.Name)
	d.Set("resource_group_name", resourceGroup)

	if location := resp.Location; location != nil {
		d.Set("location", azure.NormalizeLocation(*location))
	}

	if sku := resp.Sku; sku != nil {
		d.Set("sku_name", sku.Name)
	}

	if properties := resp.ServerProperties; properties != nil {
		d.Set("administrator_login", properties.AdministratorLogin)
		d.Set("version", string(properties.Version))
		d.Set("ssl_enforcement", string(properties.SslEnforcement))
		// Computed
		d.Set("fqdn", properties.FullyQualifiedDomainName)

		if err := d.Set("storage_profile", flattenMariaDbStorageProfile(properties.StorageProfile)); err != nil {
			return fmt.Errorf("Error setting `storage_profile`: %+v", err)
		}
	}

	if err := d.Set("sku", flattenMariaDbServerSku(resp.Sku)); err != nil {
		return fmt.Errorf("Error setting `sku`: %+v", err)
	}

	return tags.FlattenAndSet(d, resp.Tags)
}

func resourceArmMariaDbServerDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).MariaDB.ServersClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resourceGroup := id.ResourceGroup
	name := id.Path["servers"]

	future, err := client.Delete(ctx, resourceGroup, name)
	if err != nil {
		if response.WasNotFound(future.Response()) {
			return nil
		}

		return fmt.Errorf("Error deleting MariaDB Server %q (Resource Group %q): %+v", name, resourceGroup, err)
	}

	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		if response.WasNotFound(future.Response()) {
			return nil
		}

		return fmt.Errorf("Error waiting for deletion of MariaDB Server %q (Resource Group %q): %+v", name, resourceGroup, err)
	}

	return nil
}

func expandServerSkuName(skuName string) (*mariadb.Sku, error) {
	parts := strings.Split(skuName, "_")
	if len(parts) != 3 {
		return nil, fmt.Errorf("sku_name (%s) has the worng number of parts (%d) after splitting on _", skuName, len(parts))
	}

	var tier mariadb.SkuTier
	switch parts[0] {
	case "B":
		tier = mariadb.Basic
	case "GP":
		tier = mariadb.GeneralPurpose
	case "MO":
		tier = mariadb.MemoryOptimized
	default:
		return nil, fmt.Errorf("sku_name %s has unknown sku tier %s", skuName, parts[0])
	}

	capacity, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("cannot convert skuname %s capcity %s to int", skuName, parts[2])
	}

	return &mariadb.Sku{
		Name:     utils.String(skuName),
		Tier:     tier,
		Capacity: utils.Int32(int32(capacity)),
		Family:   utils.String(parts[1]),
	}, nil
}

func expandAzureRmMariaDbServerSku(d *schema.ResourceData) *mariadb.Sku {
	skus := d.Get("sku").([]interface{})
	sku := skus[0].(map[string]interface{})

	name := sku["name"].(string)
	capacity := sku["capacity"].(int)
	tier := sku["tier"].(string)
	family := sku["family"].(string)

	return &mariadb.Sku{
		Name:     utils.String(name),
		Tier:     mariadb.SkuTier(tier),
		Capacity: utils.Int32(int32(capacity)),
		Family:   utils.String(family),
	}
}

func expandAzureRmMariaDbStorageProfile(d *schema.ResourceData) *mariadb.StorageProfile {
	storageprofiles := d.Get("storage_profile").([]interface{})
	storageprofile := storageprofiles[0].(map[string]interface{})

	backupRetentionDays := storageprofile["backup_retention_days"].(int)
	geoRedundantBackup := storageprofile["geo_redundant_backup"].(string)
	storageMB := storageprofile["storage_mb"].(int)
	autoGrow := storageprofile["auto_grow"].(string)

	return &mariadb.StorageProfile{
		BackupRetentionDays: utils.Int32(int32(backupRetentionDays)),
		GeoRedundantBackup:  mariadb.GeoRedundantBackup(geoRedundantBackup),
		StorageMB:           utils.Int32(int32(storageMB)),
		StorageAutogrow:     mariadb.StorageAutogrow(autoGrow),
	}
}

func flattenMariaDbServerSku(sku *mariadb.Sku) []interface{} {
	values := map[string]interface{}{}

	if sku == nil {
		return []interface{}{}
	}

	if name := sku.Name; name != nil {
		values["name"] = *name
	}

	if capacity := sku.Capacity; capacity != nil {
		values["capacity"] = *capacity
	}

	values["tier"] = string(sku.Tier)

	if family := sku.Family; family != nil {
		values["family"] = *family
	}

	return []interface{}{values}
}

func flattenMariaDbStorageProfile(storage *mariadb.StorageProfile) []interface{} {
	values := map[string]interface{}{}

	if storage == nil {
		return []interface{}{}
	}

	if storageMB := storage.StorageMB; storageMB != nil {
		values["storage_mb"] = *storageMB
	}

	if backupRetentionDays := storage.BackupRetentionDays; backupRetentionDays != nil {
		values["backup_retention_days"] = *backupRetentionDays
	}

	values["geo_redundant_backup"] = string(storage.GeoRedundantBackup)

	values["auto_grow"] = string(storage.StorageAutogrow)

	return []interface{}{values}
}
