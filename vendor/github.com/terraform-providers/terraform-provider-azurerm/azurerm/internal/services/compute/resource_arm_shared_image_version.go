package compute

import (
	"fmt"
	"log"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/hashicorp/go-azure-helpers/response"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/tf"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/validate"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/features"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tags"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func resourceArmSharedImageVersion() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmSharedImageVersionCreateUpdate,
		Read:   resourceArmSharedImageVersionRead,
		Update: resourceArmSharedImageVersionCreateUpdate,
		Delete: resourceArmSharedImageVersionDelete,
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
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.SharedImageVersionName,
			},

			"gallery_name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.SharedImageGalleryName,
			},

			"image_name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.SharedImageName,
			},

			"location": azure.SchemaLocation(),

			"resource_group_name": azure.SchemaResourceGroupName(),

			"managed_image_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"target_region": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:             schema.TypeString,
							Required:         true,
							StateFunc:        azure.NormalizeLocation,
							DiffSuppressFunc: azure.SuppressLocationDiff,
						},

						"regional_replica_count": {
							Type:     schema.TypeInt,
							Required: true,
						},

						"storage_account_type": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},
					},
				},
			},

			"exclude_from_latest": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"tags": tags.Schema(),
		},
	}
}

func resourceArmSharedImageVersionCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Compute.GalleryImageVersionsClient
	ctx, cancel := timeouts.ForCreateUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	imageVersion := d.Get("name").(string)
	imageName := d.Get("image_name").(string)
	galleryName := d.Get("gallery_name").(string)
	resourceGroup := d.Get("resource_group_name").(string)
	location := azure.NormalizeLocation(d.Get("location").(string))
	managedImageId := d.Get("managed_image_id").(string)
	excludeFromLatest := d.Get("exclude_from_latest").(bool)

	if features.ShouldResourcesBeImported() && d.IsNewResource() {
		existing, err := client.Get(ctx, resourceGroup, galleryName, imageName, imageVersion, "")
		if err != nil {
			if !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("Error checking for presence of existing Shared Image Version %q (Image %q / Gallery %q / Resource Group %q): %+v", imageVersion, imageName, galleryName, resourceGroup, err)
			}
		}

		if existing.ID != nil && *existing.ID != "" {
			return tf.ImportAsExistsError("azurerm_shared_image_version", *existing.ID)
		}
	}

	targetRegions := expandSharedImageVersionTargetRegions(d)
	t := d.Get("tags").(map[string]interface{})

	version := compute.GalleryImageVersion{
		Location: utils.String(location),
		GalleryImageVersionProperties: &compute.GalleryImageVersionProperties{
			PublishingProfile: &compute.GalleryImageVersionPublishingProfile{
				ExcludeFromLatest: utils.Bool(excludeFromLatest),
				TargetRegions:     targetRegions,
			},
			StorageProfile: &compute.GalleryImageVersionStorageProfile{
				Source: &compute.GalleryArtifactVersionSource{
					ID: utils.String(managedImageId),
				},
			},
		},
		Tags: tags.Expand(t),
	}

	future, err := client.CreateOrUpdate(ctx, resourceGroup, galleryName, imageName, imageVersion, version)
	if err != nil {
		return fmt.Errorf("Error creating Shared Image Version %q (Image %q / Gallery %q / Resource Group %q): %+v", imageVersion, imageName, galleryName, resourceGroup, err)
	}

	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("Error waiting for the creation of Shared Image Version %q (Image %q / Gallery %q / Resource Group %q): %+v", imageVersion, imageName, galleryName, resourceGroup, err)
	}

	read, err := client.Get(ctx, resourceGroup, galleryName, imageName, imageVersion, "")
	if err != nil {
		return fmt.Errorf("Error retrieving Shared Image Version %q (Image %q / Gallery %q / Resource Group %q): %+v", imageVersion, imageName, galleryName, resourceGroup, err)
	}

	d.SetId(*read.ID)

	return resourceArmSharedImageVersionRead(d, meta)
}

func resourceArmSharedImageVersionRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Compute.GalleryImageVersionsClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}

	imageVersion := id.Path["versions"]
	imageName := id.Path["images"]
	galleryName := id.Path["galleries"]
	resourceGroup := id.ResourceGroup

	resp, err := client.Get(ctx, resourceGroup, galleryName, imageName, imageVersion, compute.ReplicationStatusTypesReplicationStatus)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			log.Printf("[DEBUG] Shared Image Version %q (Image %q / Gallery %q / Resource Group %q) was not found - removing from state", imageVersion, imageName, galleryName, resourceGroup)
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error retrieving Shared Image Version %q (Image %q / Gallery %q / Resource Group %q): %+v", imageVersion, imageName, galleryName, resourceGroup, err)
	}

	d.Set("name", resp.Name)
	d.Set("image_name", imageName)
	d.Set("gallery_name", galleryName)
	d.Set("resource_group_name", resourceGroup)

	if location := resp.Location; location != nil {
		d.Set("location", azure.NormalizeLocation(*location))
	}

	if props := resp.GalleryImageVersionProperties; props != nil {
		if profile := props.PublishingProfile; profile != nil {
			d.Set("exclude_from_latest", profile.ExcludeFromLatest)

			flattenedRegions := flattenSharedImageVersionTargetRegions(profile.TargetRegions)
			if err := d.Set("target_region", flattenedRegions); err != nil {
				return fmt.Errorf("Error setting `target_region`: %+v", err)
			}
		}

		if profile := props.StorageProfile; profile != nil {
			if source := profile.Source; source != nil {
				d.Set("managed_image_id", source.ID)
			}
		}
	}

	return tags.FlattenAndSet(d, resp.Tags)
}

func resourceArmSharedImageVersionDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Compute.GalleryImageVersionsClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}

	imageVersion := id.Path["versions"]
	imageName := id.Path["images"]
	galleryName := id.Path["galleries"]
	resourceGroup := id.ResourceGroup

	future, err := client.Delete(ctx, resourceGroup, galleryName, imageName, imageVersion)
	if err != nil {
		if response.WasNotFound(future.Response()) {
			return nil
		}

		return fmt.Errorf("Error deleting Shared Image Version %q (Image %q / Gallery %q / Resource Group %q): %+v", imageVersion, imageName, galleryName, resourceGroup, err)
	}

	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		if !response.WasNotFound(future.Response()) {
			return fmt.Errorf("Error deleting Shared Image Version %q (Image %q / Gallery %q / Resource Group %q): %+v", imageVersion, imageName, galleryName, resourceGroup, err)
		}
	}

	return nil
}
func expandSharedImageVersionTargetRegions(d *schema.ResourceData) *[]compute.TargetRegion {
	vs := d.Get("target_region").(*schema.Set)
	results := make([]compute.TargetRegion, 0)

	for _, v := range vs.List() {
		input := v.(map[string]interface{})

		name := input["name"].(string)
		regionalReplicaCount := input["regional_replica_count"].(int)
		storageAccountType := input["storage_account_type"].(string)

		output := compute.TargetRegion{
			Name:                 utils.String(name),
			RegionalReplicaCount: utils.Int32(int32(regionalReplicaCount)),
			StorageAccountType:   compute.StorageAccountType(storageAccountType),
		}
		results = append(results, output)
	}

	return &results
}

func flattenSharedImageVersionTargetRegions(input *[]compute.TargetRegion) []interface{} {
	results := make([]interface{}, 0)

	if input != nil {
		for _, v := range *input {
			output := make(map[string]interface{})

			if v.Name != nil {
				output["name"] = azure.NormalizeLocation(*v.Name)
			}

			if v.RegionalReplicaCount != nil {
				output["regional_replica_count"] = int(*v.RegionalReplicaCount)
			}

			output["storage_account_type"] = string(v.StorageAccountType)

			results = append(results, output)
		}
	}

	return results
}
