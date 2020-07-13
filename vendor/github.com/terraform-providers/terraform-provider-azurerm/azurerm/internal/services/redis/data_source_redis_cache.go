package redis

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tags"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func dataSourceArmRedisCache() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceArmRedisCacheRead,

		Timeouts: &schema.ResourceTimeout{
			Read: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"location": azure.SchemaLocationForDataSource(),

			"resource_group_name": azure.SchemaResourceGroupNameForDataSource(),

			"zones": azure.SchemaZonesComputed(),

			"capacity": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"family": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"sku_name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"minimum_tls_version": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"shard_count": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"enable_non_ssl_port": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"subnet_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"private_static_ip_address": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"redis_configuration": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"maxclients": {
							Type:     schema.TypeInt,
							Computed: true,
						},

						"maxmemory_delta": {
							Type:     schema.TypeInt,
							Computed: true,
						},

						"maxmemory_reserved": {
							Type:     schema.TypeInt,
							Computed: true,
						},

						"maxmemory_policy": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"maxfragmentationmemory_reserved": {
							Type:     schema.TypeInt,
							Computed: true,
						},

						"rdb_backup_enabled": {
							Type:     schema.TypeBool,
							Computed: true,
						},

						"rdb_backup_frequency": {
							Type:     schema.TypeInt,
							Computed: true,
						},

						"rdb_backup_max_snapshot_count": {
							Type:     schema.TypeInt,
							Computed: true,
						},

						"rdb_storage_connection_string": {
							Type:      schema.TypeString,
							Computed:  true,
							Sensitive: true,
						},

						"notify_keyspace_events": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"aof_backup_enabled": {
							Type:     schema.TypeBool,
							Computed: true,
						},

						"aof_storage_connection_string_0": {
							Type:      schema.TypeString,
							Computed:  true,
							Sensitive: true,
						},

						"aof_storage_connection_string_1": {
							Type:      schema.TypeString,
							Computed:  true,
							Sensitive: true,
						},
						"enable_authentication": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},

			"patch_schedule": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"day_of_week": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"start_hour_utc": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},

			"hostname": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"port": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"ssl_port": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"primary_access_key": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},

			"secondary_access_key": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},

			"tags": tags.SchemaDataSource(),
		},
	}
}

func dataSourceArmRedisCacheRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Redis.Client
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	resourceGroup := d.Get("resource_group_name").(string)
	name := d.Get("name").(string)

	resp, err := client.Get(ctx, resourceGroup, name)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			return fmt.Errorf("Error: Redis instance %q (Resource group %q) was not found", name, resourceGroup)
		}
		return fmt.Errorf("Error reading the state of Redis instance %q: %+v", name, err)
	}

	d.SetId(*resp.ID)

	if location := resp.Location; location != nil {
		d.Set("location", azure.NormalizeLocation(*location))
	}

	if zones := resp.Zones; zones != nil {
		d.Set("zones", zones)
	}

	if sku := resp.Sku; sku != nil {
		d.Set("capacity", sku.Capacity)
		d.Set("family", sku.Family)
		d.Set("sku_name", sku.Name)
	}

	if props := resp.Properties; props != nil {
		d.Set("ssl_port", props.SslPort)
		d.Set("hostname", props.HostName)
		d.Set("minimum_tls_version", string(props.MinimumTLSVersion))
		d.Set("port", props.Port)
		d.Set("enable_non_ssl_port", props.EnableNonSslPort)
		if props.ShardCount != nil {
			d.Set("shard_count", props.ShardCount)
		}
		d.Set("private_static_ip_address", props.StaticIP)
		d.Set("subnet_id", props.SubnetID)
	}

	redisConfiguration, err := flattenRedisConfiguration(resp.RedisConfiguration)

	if err != nil {
		return fmt.Errorf("Error flattening `redis_configuration`: %+v", err)
	}
	if err := d.Set("redis_configuration", redisConfiguration); err != nil {
		return fmt.Errorf("Error setting `redis_configuration`: %+v", err)
	}

	patchSchedulesClient := meta.(*clients.Client).Redis.PatchSchedulesClient

	schedule, err := patchSchedulesClient.Get(ctx, resourceGroup, name)
	if err == nil {
		patchSchedule := flattenRedisPatchSchedules(schedule)
		if err = d.Set("patch_schedule", patchSchedule); err != nil {
			return fmt.Errorf("Error setting `patch_schedule`: %+v", err)
		}
	} else {
		d.Set("patch_schedule", []interface{}{})
	}

	keys, err := client.ListKeys(ctx, resourceGroup, name)
	if err != nil {
		return err
	}

	d.Set("primary_access_key", keys.PrimaryKey)
	d.Set("secondary_access_key", keys.SecondaryKey)

	return tags.FlattenAndSet(d, resp.Tags)
}
