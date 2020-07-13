package recoveryservices

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/recoveryservices/mgmt/2019-05-13/backup"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/set"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/suppress"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/tf"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/validate"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/features"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tags"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func resourceArmBackupProtectionPolicyVM() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmBackupProtectionPolicyVMCreateUpdate,
		Read:   resourceArmBackupProtectionPolicyVMRead,
		Update: resourceArmBackupProtectionPolicyVMCreateUpdate,
		Delete: resourceArmBackupProtectionPolicyVMDelete,

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
				ValidateFunc: validation.StringMatch(
					regexp.MustCompile("^[a-zA-Z][-_!a-zA-Z0-9]{2,149}$"),
					"Backup Policy name must be 3 - 150 characters long, start with a letter, contain only letters and numbers.",
				),
			},

			"resource_group_name": azure.SchemaResourceGroupName(),

			"recovery_vault_name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: azure.ValidateRecoveryServicesVaultName,
			},

			"timezone": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "UTC",
			},

			"backup": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{

						"frequency": {
							Type:             schema.TypeString,
							Required:         true,
							DiffSuppressFunc: suppress.CaseDifference,
							ValidateFunc: validation.StringInSlice([]string{
								string(backup.ScheduleRunTypeDaily),
								string(backup.ScheduleRunTypeWeekly),
							}, true),
						},

						"time": { //applies to all backup schedules & retention times (they all must be the same)
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringMatch(
								regexp.MustCompile("^([01][0-9]|[2][0-3]):([03][0])$"), //time must be on the hour or half past
								"Time of day must match the format HH:mm where HH is 00-23 and mm is 00 or 30",
							),
						},

						"weekdays": { //only for weekly
							Type:     schema.TypeSet,
							Optional: true,
							Set:      set.HashStringIgnoreCase,
							Elem: &schema.Schema{
								Type:             schema.TypeString,
								DiffSuppressFunc: suppress.CaseDifference,
								ValidateFunc:     validate.DayOfTheWeek(true),
							},
						},
					},
				},
			},

			"retention_daily": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"count": {
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validation.IntBetween(1, 9999),
						},
					},
				},
			},

			"retention_weekly": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"count": {
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validation.IntBetween(1, 9999),
						},

						"weekdays": {
							Type:     schema.TypeSet,
							Required: true,
							Set:      set.HashStringIgnoreCase,
							Elem: &schema.Schema{
								Type:             schema.TypeString,
								DiffSuppressFunc: suppress.CaseDifference,
								ValidateFunc:     validate.DayOfTheWeek(true),
							},
						},
					},
				},
			},

			"retention_monthly": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"count": {
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validation.IntBetween(1, 9999),
						},

						"weeks": {
							Type:     schema.TypeSet,
							Required: true,
							Set:      set.HashStringIgnoreCase,
							Elem: &schema.Schema{
								Type:             schema.TypeString,
								DiffSuppressFunc: suppress.CaseDifference,
								ValidateFunc: validation.StringInSlice([]string{
									string(backup.WeekOfMonthFirst),
									string(backup.WeekOfMonthSecond),
									string(backup.WeekOfMonthThird),
									string(backup.WeekOfMonthFourth),
									string(backup.WeekOfMonthLast),
								}, true),
							},
						},

						"weekdays": {
							Type:     schema.TypeSet,
							Required: true,
							Set:      set.HashStringIgnoreCase,
							Elem: &schema.Schema{
								Type:             schema.TypeString,
								DiffSuppressFunc: suppress.CaseDifference,
								ValidateFunc:     validate.DayOfTheWeek(true),
							},
						},
					},
				},
			},

			"retention_yearly": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"count": {
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validation.IntBetween(1, 9999),
						},

						"months": {
							Type:     schema.TypeSet,
							Required: true,
							Set:      set.HashStringIgnoreCase,
							Elem: &schema.Schema{
								Type:             schema.TypeString,
								DiffSuppressFunc: suppress.CaseDifference,
								ValidateFunc:     validate.Month(true),
							},
						},

						"weeks": {
							Type:     schema.TypeSet,
							Required: true,
							Set:      set.HashStringIgnoreCase,
							Elem: &schema.Schema{
								Type:             schema.TypeString,
								DiffSuppressFunc: suppress.CaseDifference,
								ValidateFunc: validation.StringInSlice([]string{
									string(backup.WeekOfMonthFirst),
									string(backup.WeekOfMonthSecond),
									string(backup.WeekOfMonthThird),
									string(backup.WeekOfMonthFourth),
									string(backup.WeekOfMonthLast),
								}, true),
							},
						},

						"weekdays": {
							Type:     schema.TypeSet,
							Required: true,
							Set:      set.HashStringIgnoreCase,
							Elem: &schema.Schema{
								Type:             schema.TypeString,
								DiffSuppressFunc: suppress.CaseDifference,
								ValidateFunc:     validate.DayOfTheWeek(true),
							},
						},
					},
				},
			},

			"tags": tags.Schema(),
		},

		//if daily, we need daily retention
		//if weekly daily cannot be set, and we need weekly
		CustomizeDiff: func(diff *schema.ResourceDiff, v interface{}) error {
			_, hasDaily := diff.GetOk("retention_daily")
			_, hasWeekly := diff.GetOk("retention_weekly")

			frequencyI, _ := diff.GetOk("backup.0.frequency")
			frequency := strings.ToLower(frequencyI.(string))
			if frequency == "daily" {
				if !hasDaily {
					return fmt.Errorf("`retention_daily` must be set when backup.0.frequency is daily")
				}

				if _, ok := diff.GetOk("backup.0.weekdays"); ok {
					return fmt.Errorf("`backup.0.weekdays` should be not set when backup.0.frequency is daily")
				}
			} else if frequency == "weekly" {
				if hasDaily {
					return fmt.Errorf("`retention_daily` must be not set when backup.0.frequency is weekly")
				}
				if !hasWeekly {
					return fmt.Errorf("`retention_weekly` must be set when backup.0.frequency is weekly")
				}
			} else {
				return fmt.Errorf("Unrecognized value for backup.0.frequency")
			}

			return nil
		},
	}
}

func resourceArmBackupProtectionPolicyVMCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).RecoveryServices.ProtectionPoliciesClient
	ctx, cancel := timeouts.ForCreateUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	policyName := d.Get("name").(string)
	resourceGroup := d.Get("resource_group_name").(string)
	vaultName := d.Get("recovery_vault_name").(string)
	t := d.Get("tags").(map[string]interface{})

	log.Printf("[DEBUG] Creating/updating Azure Backup Protection Policy %s (resource group %q)", policyName, resourceGroup)

	//getting this ready now because its shared between *everything*, time is... complicated for this resource
	timeOfDay := d.Get("backup.0.time").(string)
	dateOfDay, err := time.Parse(time.RFC3339, fmt.Sprintf("2018-07-30T%s:00Z", timeOfDay))
	if err != nil {
		return fmt.Errorf("Error generating time from %q for policy %q (Resource Group %q): %+v", timeOfDay, policyName, resourceGroup, err)
	}
	times := append(make([]date.Time, 0), date.Time{Time: dateOfDay})

	if features.ShouldResourcesBeImported() && d.IsNewResource() {
		existing, err2 := client.Get(ctx, vaultName, resourceGroup, policyName)
		if err2 != nil {
			if !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("Error checking for presence of existing Azure Backup Protection Policy %q (Resource Group %q): %+v", policyName, resourceGroup, err2)
			}
		}

		if existing.ID != nil && *existing.ID != "" {
			return tf.ImportAsExistsError("azurerm_backup_policy_vm", *existing.ID)
		}
	}

	policy := backup.ProtectionPolicyResource{
		Tags: tags.Expand(t),
		Properties: &backup.AzureIaaSVMProtectionPolicy{
			TimeZone:             utils.String(d.Get("timezone").(string)),
			BackupManagementType: backup.BackupManagementTypeAzureIaasVM,
			SchedulePolicy:       expandArmBackupProtectionPolicyVMSchedule(d, times),
			RetentionPolicy: &backup.LongTermRetentionPolicy{ //SimpleRetentionPolicy only has duration property ¯\_(ツ)_/¯
				RetentionPolicyType: backup.RetentionPolicyTypeLongTermRetentionPolicy,
				DailySchedule:       expandArmBackupProtectionPolicyVMRetentionDaily(d, times),
				WeeklySchedule:      expandArmBackupProtectionPolicyVMRetentionWeekly(d, times),
				MonthlySchedule:     expandArmBackupProtectionPolicyVMRetentionMonthly(d, times),
				YearlySchedule:      expandArmBackupProtectionPolicyVMRetentionYearly(d, times),
			},
		},
	}
	if _, err = client.CreateOrUpdate(ctx, vaultName, resourceGroup, policyName, policy); err != nil {
		return fmt.Errorf("Error creating/updating Azure Backup Protection Policy %q (Resource Group %q): %+v", policyName, resourceGroup, err)
	}

	resp, err := resourceArmBackupProtectionPolicyVMWaitForUpdate(ctx, client, vaultName, resourceGroup, policyName, d)
	if err != nil {
		return err
	}

	id := strings.Replace(*resp.ID, "Subscriptions", "subscriptions", 1)
	d.SetId(id)

	return resourceArmBackupProtectionPolicyVMRead(d, meta)
}

func resourceArmBackupProtectionPolicyVMRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).RecoveryServices.ProtectionPoliciesClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}

	policyName := id.Path["backupPolicies"]
	vaultName := id.Path["vaults"]
	resourceGroup := id.ResourceGroup

	log.Printf("[DEBUG] Reading Azure Backup Protection Policy %q (resource group %q)", policyName, resourceGroup)

	resp, err := client.Get(ctx, vaultName, resourceGroup, policyName)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error making Read request on Azure Backup Protection Policy %q (Resource Group %q): %+v", policyName, resourceGroup, err)
	}

	d.Set("name", policyName)
	d.Set("resource_group_name", resourceGroup)
	d.Set("recovery_vault_name", vaultName)

	if properties, ok := resp.Properties.AsAzureIaaSVMProtectionPolicy(); ok && properties != nil {
		d.Set("timezone", properties.TimeZone)

		if schedule, ok := properties.SchedulePolicy.AsSimpleSchedulePolicy(); ok && schedule != nil {
			if err := d.Set("backup", flattenArmBackupProtectionPolicyVMSchedule(schedule)); err != nil {
				return fmt.Errorf("Error setting `backup`: %+v", err)
			}
		}

		if retention, ok := properties.RetentionPolicy.AsLongTermRetentionPolicy(); ok && retention != nil {
			if s := retention.DailySchedule; s != nil {
				if err := d.Set("retention_daily", flattenArmBackupProtectionPolicyVMRetentionDaily(s)); err != nil {
					return fmt.Errorf("Error setting `retention_daily`: %+v", err)
				}
			} else {
				d.Set("retention_daily", nil)
			}

			if s := retention.WeeklySchedule; s != nil {
				if err := d.Set("retention_weekly", flattenArmBackupProtectionPolicyVMRetentionWeekly(s)); err != nil {
					return fmt.Errorf("Error setting `retention_weekly`: %+v", err)
				}
			} else {
				d.Set("retention_weekly", nil)
			}

			if s := retention.MonthlySchedule; s != nil {
				if err := d.Set("retention_monthly", flattenArmBackupProtectionPolicyVMRetentionMonthly(s)); err != nil {
					return fmt.Errorf("Error setting `retention_monthly`: %+v", err)
				}
			} else {
				d.Set("retention_monthly", nil)
			}

			if s := retention.YearlySchedule; s != nil {
				if err := d.Set("retention_yearly", flattenArmBackupProtectionPolicyVMRetentionYearly(s)); err != nil {
					return fmt.Errorf("Error setting `retention_yearly`: %+v", err)
				}
			} else {
				d.Set("retention_yearly", nil)
			}
		}
	}

	return tags.FlattenAndSet(d, resp.Tags)
}

func resourceArmBackupProtectionPolicyVMDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).RecoveryServices.ProtectionPoliciesClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return err
	}

	policyName := id.Path["backupPolicies"]
	resourceGroup := id.ResourceGroup
	vaultName := id.Path["vaults"]

	log.Printf("[DEBUG] Deleting Azure Backup Protected Item %q (resource group %q)", policyName, resourceGroup)

	resp, err := client.Delete(ctx, vaultName, resourceGroup, policyName)
	if err != nil {
		if !utils.ResponseWasNotFound(resp) {
			return fmt.Errorf("Error issuing delete request for Azure Backup Protection Policy %q (Resource Group %q): %+v", policyName, resourceGroup, err)
		}
	}

	if _, err := resourceArmBackupProtectionPolicyVMWaitForDeletion(ctx, client, vaultName, resourceGroup, policyName, d); err != nil {
		return err
	}

	return nil
}

func expandArmBackupProtectionPolicyVMSchedule(d *schema.ResourceData, times []date.Time) *backup.SimpleSchedulePolicy {
	if bb, ok := d.Get("backup").([]interface{}); ok && len(bb) > 0 {
		block := bb[0].(map[string]interface{})

		schedule := backup.SimpleSchedulePolicy{ //LongTermSchedulePolicy has no properties
			SchedulePolicyType: backup.SchedulePolicyTypeSimpleSchedulePolicy,
			ScheduleRunTimes:   &times,
		}

		if v, ok := block["frequency"].(string); ok {
			schedule.ScheduleRunFrequency = backup.ScheduleRunType(v)
		}

		if v, ok := block["weekdays"].(*schema.Set); ok {
			days := make([]backup.DayOfWeek, 0)
			for _, day := range v.List() {
				days = append(days, backup.DayOfWeek(day.(string)))
			}
			schedule.ScheduleRunDays = &days
		}

		return &schedule
	}

	return nil
}

func expandArmBackupProtectionPolicyVMRetentionDaily(d *schema.ResourceData, times []date.Time) *backup.DailyRetentionSchedule {
	if rb, ok := d.Get("retention_daily").([]interface{}); ok && len(rb) > 0 {
		block := rb[0].(map[string]interface{})

		return &backup.DailyRetentionSchedule{
			RetentionTimes: &times,
			RetentionDuration: &backup.RetentionDuration{
				Count:        utils.Int32(int32(block["count"].(int))),
				DurationType: backup.RetentionDurationTypeDays,
			},
		}
	}

	return nil
}

func expandArmBackupProtectionPolicyVMRetentionWeekly(d *schema.ResourceData, times []date.Time) *backup.WeeklyRetentionSchedule {
	if rb, ok := d.Get("retention_weekly").([]interface{}); ok && len(rb) > 0 {
		block := rb[0].(map[string]interface{})

		retention := backup.WeeklyRetentionSchedule{
			RetentionTimes: &times,
			RetentionDuration: &backup.RetentionDuration{
				Count:        utils.Int32(int32(block["count"].(int))),
				DurationType: backup.RetentionDurationTypeWeeks,
			},
		}

		if v, ok := block["weekdays"].(*schema.Set); ok {
			days := make([]backup.DayOfWeek, 0)
			for _, day := range v.List() {
				days = append(days, backup.DayOfWeek(day.(string)))
			}
			retention.DaysOfTheWeek = &days
		}

		return &retention
	}

	return nil
}

func expandArmBackupProtectionPolicyVMRetentionMonthly(d *schema.ResourceData, times []date.Time) *backup.MonthlyRetentionSchedule {
	if rb, ok := d.Get("retention_monthly").([]interface{}); ok && len(rb) > 0 {
		block := rb[0].(map[string]interface{})

		retention := backup.MonthlyRetentionSchedule{
			RetentionScheduleFormatType: backup.RetentionScheduleFormatWeekly, //this is always weekly ¯\_(ツ)_/¯
			RetentionScheduleDaily:      nil,                                  //and this is always nil..
			RetentionScheduleWeekly:     expandArmBackupProtectionPolicyVMRetentionWeeklyFormat(block),
			RetentionTimes:              &times,
			RetentionDuration: &backup.RetentionDuration{
				Count:        utils.Int32(int32(block["count"].(int))),
				DurationType: backup.RetentionDurationTypeMonths,
			},
		}

		return &retention
	}

	return nil
}

func expandArmBackupProtectionPolicyVMRetentionYearly(d *schema.ResourceData, times []date.Time) *backup.YearlyRetentionSchedule {
	if rb, ok := d.Get("retention_yearly").([]interface{}); ok && len(rb) > 0 {
		block := rb[0].(map[string]interface{})

		retention := backup.YearlyRetentionSchedule{
			RetentionScheduleFormatType: backup.RetentionScheduleFormatWeekly, //this is always weekly ¯\_(ツ)_/¯
			RetentionScheduleDaily:      nil,                                  //and this is always nil..
			RetentionScheduleWeekly:     expandArmBackupProtectionPolicyVMRetentionWeeklyFormat(block),
			RetentionTimes:              &times,
			RetentionDuration: &backup.RetentionDuration{
				Count:        utils.Int32(int32(block["count"].(int))),
				DurationType: backup.RetentionDurationTypeYears,
			},
		}

		if v, ok := block["months"].(*schema.Set); ok {
			months := make([]backup.MonthOfYear, 0)
			for _, month := range v.List() {
				months = append(months, backup.MonthOfYear(month.(string)))
			}
			retention.MonthsOfYear = &months
		}

		return &retention
	}

	return nil
}

func expandArmBackupProtectionPolicyVMRetentionWeeklyFormat(block map[string]interface{}) *backup.WeeklyRetentionFormat {
	weekly := backup.WeeklyRetentionFormat{}

	if v, ok := block["weekdays"].(*schema.Set); ok {
		days := make([]backup.DayOfWeek, 0)
		for _, day := range v.List() {
			days = append(days, backup.DayOfWeek(day.(string)))
		}
		weekly.DaysOfTheWeek = &days
	}

	if v, ok := block["weeks"].(*schema.Set); ok {
		weeks := make([]backup.WeekOfMonth, 0)
		for _, week := range v.List() {
			weeks = append(weeks, backup.WeekOfMonth(week.(string)))
		}
		weekly.WeeksOfTheMonth = &weeks
	}

	return &weekly
}

func flattenArmBackupProtectionPolicyVMSchedule(schedule *backup.SimpleSchedulePolicy) []interface{} {
	block := map[string]interface{}{}

	block["frequency"] = string(schedule.ScheduleRunFrequency)

	if times := schedule.ScheduleRunTimes; times != nil && len(*times) > 0 {
		block["time"] = (*times)[0].Format("15:04")
	}

	if days := schedule.ScheduleRunDays; days != nil {
		weekdays := make([]interface{}, 0)
		for _, d := range *days {
			weekdays = append(weekdays, string(d))
		}
		block["weekdays"] = schema.NewSet(schema.HashString, weekdays)
	}

	return []interface{}{block}
}

func flattenArmBackupProtectionPolicyVMRetentionDaily(daily *backup.DailyRetentionSchedule) []interface{} {
	block := map[string]interface{}{}

	if duration := daily.RetentionDuration; duration != nil {
		if v := duration.Count; v != nil {
			block["count"] = *v
		}
	}

	return []interface{}{block}
}

func flattenArmBackupProtectionPolicyVMRetentionWeekly(weekly *backup.WeeklyRetentionSchedule) []interface{} {
	block := map[string]interface{}{}

	if duration := weekly.RetentionDuration; duration != nil {
		if v := duration.Count; v != nil {
			block["count"] = *v
		}
	}

	if days := weekly.DaysOfTheWeek; days != nil {
		weekdays := make([]interface{}, 0)
		for _, d := range *days {
			weekdays = append(weekdays, string(d))
		}
		block["weekdays"] = schema.NewSet(schema.HashString, weekdays)
	}

	return []interface{}{block}
}

func flattenArmBackupProtectionPolicyVMRetentionMonthly(monthly *backup.MonthlyRetentionSchedule) []interface{} {
	block := map[string]interface{}{}

	if duration := monthly.RetentionDuration; duration != nil {
		if v := duration.Count; v != nil {
			block["count"] = *v
		}
	}

	if weekly := monthly.RetentionScheduleWeekly; weekly != nil {
		block["weekdays"], block["weeks"] = flattenArmBackupProtectionPolicyVMRetentionWeeklyFormat(weekly)
	}

	return []interface{}{block}
}

func flattenArmBackupProtectionPolicyVMRetentionYearly(yearly *backup.YearlyRetentionSchedule) []interface{} {
	block := map[string]interface{}{}

	if duration := yearly.RetentionDuration; duration != nil {
		if v := duration.Count; v != nil {
			block["count"] = *v
		}
	}

	if weekly := yearly.RetentionScheduleWeekly; weekly != nil {
		block["weekdays"], block["weeks"] = flattenArmBackupProtectionPolicyVMRetentionWeeklyFormat(weekly)
	}

	if months := yearly.MonthsOfYear; months != nil {
		slice := make([]interface{}, 0)
		for _, d := range *months {
			slice = append(slice, string(d))
		}
		block["months"] = schema.NewSet(schema.HashString, slice)
	}

	return []interface{}{block}
}

func flattenArmBackupProtectionPolicyVMRetentionWeeklyFormat(retention *backup.WeeklyRetentionFormat) (weekdays, weeks *schema.Set) {
	if days := retention.DaysOfTheWeek; days != nil {
		slice := make([]interface{}, 0)
		for _, d := range *days {
			slice = append(slice, string(d))
		}
		weekdays = schema.NewSet(schema.HashString, slice)
	}

	if days := retention.WeeksOfTheMonth; days != nil {
		slice := make([]interface{}, 0)
		for _, d := range *days {
			slice = append(slice, string(d))
		}
		weeks = schema.NewSet(schema.HashString, slice)
	}

	return weekdays, weeks
}

func resourceArmBackupProtectionPolicyVMWaitForUpdate(ctx context.Context, client *backup.ProtectionPoliciesClient, vaultName, resourceGroup, policyName string, d *schema.ResourceData) (backup.ProtectionPolicyResource, error) {
	state := &resource.StateChangeConf{
		MinTimeout: 30 * time.Second,
		Delay:      10 * time.Second,
		Pending:    []string{"NotFound"},
		Target:     []string{"Found"},
		Refresh:    resourceArmBackupProtectionPolicyVMRefreshFunc(ctx, client, vaultName, resourceGroup, policyName),
	}

	if features.SupportsCustomTimeouts() {
		if d.IsNewResource() {
			state.Timeout = d.Timeout(schema.TimeoutCreate)
		} else {
			state.Timeout = d.Timeout(schema.TimeoutUpdate)
		}
	} else {
		state.Timeout = 30 * time.Minute
	}

	resp, err := state.WaitForState()
	if err != nil {
		return resp.(backup.ProtectionPolicyResource), fmt.Errorf("Error waiting for the Azure Backup Protection Policy %q to be true (Resource Group %q) to provision: %+v", policyName, resourceGroup, err)
	}

	return resp.(backup.ProtectionPolicyResource), nil
}

func resourceArmBackupProtectionPolicyVMWaitForDeletion(ctx context.Context, client *backup.ProtectionPoliciesClient, vaultName, resourceGroup, policyName string, d *schema.ResourceData) (backup.ProtectionPolicyResource, error) {
	state := &resource.StateChangeConf{
		MinTimeout: 30 * time.Second,
		Delay:      10 * time.Second,
		Pending:    []string{"Found"},
		Target:     []string{"NotFound"},
		Refresh:    resourceArmBackupProtectionPolicyVMRefreshFunc(ctx, client, vaultName, resourceGroup, policyName),
	}

	if features.SupportsCustomTimeouts() {
		state.Timeout = d.Timeout(schema.TimeoutDelete)
	} else {
		state.Timeout = 30 * time.Minute
	}

	resp, err := state.WaitForState()
	if err != nil {
		return resp.(backup.ProtectionPolicyResource), fmt.Errorf("Error waiting for the Azure Backup Protection Policy %q to be false (Resource Group %q) to provision: %+v", policyName, resourceGroup, err)
	}

	return resp.(backup.ProtectionPolicyResource), nil
}

func resourceArmBackupProtectionPolicyVMRefreshFunc(ctx context.Context, client *backup.ProtectionPoliciesClient, vaultName, resourceGroup, policyName string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := client.Get(ctx, vaultName, resourceGroup, policyName)
		if err != nil {
			if utils.ResponseWasNotFound(resp.Response) {
				return resp, "NotFound", nil
			}

			return resp, "Error", fmt.Errorf("Error making Read request on Azure Backup Protection Policy %q (Resource Group %q): %+v", policyName, resourceGroup, err)
		}

		return resp, "Found", nil
	}
}
