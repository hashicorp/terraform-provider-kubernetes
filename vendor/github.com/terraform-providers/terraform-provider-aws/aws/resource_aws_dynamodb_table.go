package aws

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/hashicorp/terraform/helper/customdiff"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsDynamoDbTable() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDynamoDbTableCreate,
		Read:   resourceAwsDynamoDbTableRead,
		Update: resourceAwsDynamoDbTableUpdate,
		Delete: resourceAwsDynamoDbTableDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
		},

		CustomizeDiff: customdiff.Sequence(
			func(diff *schema.ResourceDiff, v interface{}) error {
				return validateDynamoDbStreamSpec(diff)
			},
			func(diff *schema.ResourceDiff, v interface{}) error {
				return validateDynamoDbTableAttributes(diff)
			},
			func(diff *schema.ResourceDiff, v interface{}) error {
				if diff.Id() != "" && diff.HasChange("server_side_encryption") {
					o, n := diff.GetChange("server_side_encryption")
					if isDynamoDbTableOptionDisabled(o) && isDynamoDbTableOptionDisabled(n) {
						return diff.Clear("server_side_encryption")
					}
				}
				return nil
			},
			func(diff *schema.ResourceDiff, v interface{}) error {
				if diff.Id() != "" && diff.HasChange("point_in_time_recovery") {
					o, n := diff.GetChange("point_in_time_recovery")
					if isDynamoDbTableOptionDisabled(o) && isDynamoDbTableOptionDisabled(n) {
						return diff.Clear("point_in_time_recovery")
					}
				}
				return nil
			},
		),

		SchemaVersion: 1,
		MigrateState:  resourceAwsDynamoDbTableMigrateState,

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"hash_key": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"range_key": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"billing_mode": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  dynamodb.BillingModeProvisioned,
				ValidateFunc: validation.StringInSlice([]string{
					dynamodb.BillingModePayPerRequest,
					dynamodb.BillingModeProvisioned,
				}, false),
			},
			"write_capacity": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"read_capacity": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"attribute": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"type": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								dynamodb.ScalarAttributeTypeB,
								dynamodb.ScalarAttributeTypeN,
								dynamodb.ScalarAttributeTypeS,
							}, false),
						},
					},
				},
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
					return hashcode.String(buf.String())
				},
			},
			"ttl": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"attribute_name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"enabled": {
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
			},
			"local_secondary_index": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"range_key": {
							Type:     schema.TypeString,
							Required: true,
						},
						"projection_type": {
							Type:     schema.TypeString,
							Required: true,
						},
						"non_key_attributes": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
					return hashcode.String(buf.String())
				},
			},
			"global_secondary_index": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"write_capacity": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"read_capacity": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"hash_key": {
							Type:     schema.TypeString,
							Required: true,
						},
						"range_key": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"projection_type": {
							Type:     schema.TypeString,
							Required: true,
						},
						"non_key_attributes": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"stream_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"stream_view_type": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				StateFunc: func(v interface{}) string {
					value := v.(string)
					return strings.ToUpper(value)
				},
				ValidateFunc: validation.StringInSlice([]string{
					"",
					dynamodb.StreamViewTypeNewImage,
					dynamodb.StreamViewTypeOldImage,
					dynamodb.StreamViewTypeNewAndOldImages,
					dynamodb.StreamViewTypeKeysOnly,
				}, false),
			},
			"stream_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"stream_label": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"server_side_encryption": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Required: true,
							ForceNew: true,
						},
					},
				},
			},
			"tags": tagsSchema(),
			"point_in_time_recovery": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceAwsDynamoDbTableCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dynamodbconn

	keySchemaMap := map[string]interface{}{
		"hash_key": d.Get("hash_key").(string),
	}
	if v, ok := d.GetOk("range_key"); ok {
		keySchemaMap["range_key"] = v.(string)
	}

	log.Printf("[DEBUG] Creating DynamoDB table with key schema: %#v", keySchemaMap)

	req := &dynamodb.CreateTableInput{
		TableName:   aws.String(d.Get("name").(string)),
		BillingMode: aws.String(d.Get("billing_mode").(string)),
		KeySchema:   expandDynamoDbKeySchema(keySchemaMap),
	}

	billingMode := d.Get("billing_mode").(string)
	capacityMap := map[string]interface{}{
		"write_capacity": d.Get("write_capacity"),
		"read_capacity":  d.Get("read_capacity"),
	}

	if err := validateDynamoDbProvisionedThroughput(capacityMap, billingMode); err != nil {
		return err
	}

	req.ProvisionedThroughput = expandDynamoDbProvisionedThroughput(capacityMap, billingMode)

	if v, ok := d.GetOk("attribute"); ok {
		aSet := v.(*schema.Set)
		req.AttributeDefinitions = expandDynamoDbAttributes(aSet.List())
	}

	if v, ok := d.GetOk("local_secondary_index"); ok {
		lsiSet := v.(*schema.Set)
		req.LocalSecondaryIndexes = expandDynamoDbLocalSecondaryIndexes(lsiSet.List(), keySchemaMap)
	}

	if v, ok := d.GetOk("global_secondary_index"); ok {
		globalSecondaryIndexes := []*dynamodb.GlobalSecondaryIndex{}
		gsiSet := v.(*schema.Set)

		for _, gsiObject := range gsiSet.List() {
			gsi := gsiObject.(map[string]interface{})
			if err := validateDynamoDbProvisionedThroughput(gsi, billingMode); err != nil {
				return fmt.Errorf("Failed to create GSI: %v", err)
			}

			gsiObject := expandDynamoDbGlobalSecondaryIndex(gsi, billingMode)
			globalSecondaryIndexes = append(globalSecondaryIndexes, gsiObject)
		}
		req.GlobalSecondaryIndexes = globalSecondaryIndexes
	}

	if v, ok := d.GetOk("stream_enabled"); ok {
		req.StreamSpecification = &dynamodb.StreamSpecification{
			StreamEnabled:  aws.Bool(v.(bool)),
			StreamViewType: aws.String(d.Get("stream_view_type").(string)),
		}
	}

	if v, ok := d.GetOk("server_side_encryption"); ok {
		options := v.([]interface{})
		if options[0] == nil {
			return fmt.Errorf("At least one field is expected inside server_side_encryption")
		}

		s := options[0].(map[string]interface{})
		req.SSESpecification = expandDynamoDbEncryptAtRestOptions(s)
	}

	var output *dynamodb.CreateTableOutput
	err := resource.Retry(2*time.Minute, func() *resource.RetryError {
		var err error
		output, err = conn.CreateTable(req)
		if err != nil {
			if isAWSErr(err, "ThrottlingException", "") {
				return resource.RetryableError(err)
			}
			if isAWSErr(err, dynamodb.ErrCodeLimitExceededException, "can be created, updated, or deleted simultaneously") {
				return resource.RetryableError(err)
			}
			if isAWSErr(err, dynamodb.ErrCodeLimitExceededException, "indexed tables that can be created simultaneously") {
				return resource.RetryableError(err)
			}

			return resource.NonRetryableError(err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	d.SetId(*output.TableDescription.TableName)
	d.Set("arn", output.TableDescription.TableArn)

	if err := waitForDynamoDbTableToBeActive(d.Id(), d.Timeout(schema.TimeoutCreate), conn); err != nil {
		return err
	}

	return resourceAwsDynamoDbTableUpdate(d, meta)
}

func resourceAwsDynamoDbTableUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dynamodbconn
	billingMode := d.Get("billing_mode").(string)

	if d.HasChange("billing_mode") && !d.IsNewResource() {
		req := &dynamodb.UpdateTableInput{
			TableName:   aws.String(d.Id()),
			BillingMode: aws.String(billingMode),
		}
		capacityMap := map[string]interface{}{
			"write_capacity": d.Get("write_capacity"),
			"read_capacity":  d.Get("read_capacity"),
		}

		if err := validateDynamoDbProvisionedThroughput(capacityMap, billingMode); err != nil {
			return err
		}

		_, err := conn.UpdateTable(req)
		if err != nil {
			return fmt.Errorf("Error updating DynamoDB Table (%s) billing mode: %s", d.Id(), err)
		}
		if err := waitForDynamoDbTableToBeActive(d.Id(), d.Timeout(schema.TimeoutUpdate), conn); err != nil {
			return fmt.Errorf("Error waiting for DynamoDB Table update: %s", err)
		}
	}
	// Cannot create or delete index while updating table IOPS
	// so we update IOPS separately
	if !d.HasChange("billing_mode") && d.Get("billing_mode").(string) == dynamodb.BillingModeProvisioned && (d.HasChange("read_capacity") || d.HasChange("write_capacity")) && !d.IsNewResource() {
		_, err := conn.UpdateTable(&dynamodb.UpdateTableInput{
			TableName: aws.String(d.Id()),
			ProvisionedThroughput: expandDynamoDbProvisionedThroughput(map[string]interface{}{
				"read_capacity":  d.Get("read_capacity"),
				"write_capacity": d.Get("write_capacity"),
			}, billingMode),
		})
		if err != nil {
			return err
		}
		if err := waitForDynamoDbTableToBeActive(d.Id(), d.Timeout(schema.TimeoutUpdate), conn); err != nil {
			return fmt.Errorf("Error waiting for DynamoDB Table update: %s", err)
		}
	}

	if (d.HasChange("stream_enabled") || d.HasChange("stream_view_type")) && !d.IsNewResource() {
		toEnable := d.Get("stream_enabled").(bool)
		streamSpec := dynamodb.StreamSpecification{
			StreamEnabled: aws.Bool(toEnable),
		}
		if toEnable {
			streamSpec.StreamViewType = aws.String(d.Get("stream_view_type").(string))
		}
		input := &dynamodb.UpdateTableInput{
			TableName:           aws.String(d.Id()),
			StreamSpecification: &streamSpec,
		}
		_, err := conn.UpdateTable(input)
		if err != nil {
			return err
		}

		if err := waitForDynamoDbTableToBeActive(d.Id(), d.Timeout(schema.TimeoutUpdate), conn); err != nil {
			return fmt.Errorf("Error waiting for DynamoDB Table update: %s", err)
		}
	}

	// Indexes and attributes are tightly coupled (DynamoDB requires attribute definitions
	// for all indexed attributes) so it's necessary to update these together.
	if (d.HasChange("global_secondary_index") || d.HasChange("attribute")) && !d.IsNewResource() {
		attributes := d.Get("attribute").(*schema.Set).List()

		o, n := d.GetChange("global_secondary_index")
		ops, err := diffDynamoDbGSI(o.(*schema.Set).List(), n.(*schema.Set).List(), billingMode)
		if err != nil {
			return fmt.Errorf("Computing difference for global_secondary_index failed: %s", err)
		}
		log.Printf("[DEBUG] Updating global secondary indexes:\n%s", ops)

		input := &dynamodb.UpdateTableInput{
			TableName:            aws.String(d.Id()),
			AttributeDefinitions: expandDynamoDbAttributes(attributes),
		}

		// Only 1 online index can be created or deleted simultaneously per table
		for _, op := range ops {
			input.GlobalSecondaryIndexUpdates = []*dynamodb.GlobalSecondaryIndexUpdate{op}
			log.Printf("[DEBUG] Updating DynamoDB Table: %s", input)
			_, err := conn.UpdateTable(input)
			if err != nil {
				return err
			}
			if op.Create != nil {
				idxName := *op.Create.IndexName
				if err := waitForDynamoDbGSIToBeActive(d.Id(), idxName, conn); err != nil {
					return fmt.Errorf("Error waiting for DynamoDB GSI %q to be created: %s", idxName, err)
				}
			}
			if op.Update != nil {
				idxName := *op.Update.IndexName
				if err := waitForDynamoDbGSIToBeActive(d.Id(), idxName, conn); err != nil {
					return fmt.Errorf("Error waiting for DynamoDB GSI %q to be updated: %s", idxName, err)
				}
			}
			if op.Delete != nil {
				idxName := *op.Delete.IndexName
				if err := waitForDynamoDbGSIToBeDeleted(d.Id(), idxName, conn); err != nil {
					return fmt.Errorf("Error waiting for DynamoDB GSI %q to be deleted: %s", idxName, err)
				}
			}
		}

		// We may only be changing the attribute type
		if len(ops) == 0 {
			_, err := conn.UpdateTable(input)
			if err != nil {
				return err
			}
		}

		if err := waitForDynamoDbTableToBeActive(d.Id(), d.Timeout(schema.TimeoutUpdate), conn); err != nil {
			return fmt.Errorf("Error waiting for DynamoDB Table op: %s", err)
		}
	}

	if d.HasChange("ttl") {
		if err := updateDynamoDbTimeToLive(d, conn); err != nil {
			log.Printf("[DEBUG] Error updating table TimeToLive: %s", err)
			return err
		}
	}

	if d.HasChange("tags") {
		if err := setTagsDynamoDb(conn, d); err != nil {
			return err
		}
	}

	if d.HasChange("point_in_time_recovery") {
		_, enabled := d.GetChange("point_in_time_recovery.0.enabled")
		if !d.IsNewResource() || enabled.(bool) {
			if err := updateDynamoDbPITR(d, conn); err != nil {
				return err
			}
		}
	}

	return resourceAwsDynamoDbTableRead(d, meta)
}

func resourceAwsDynamoDbTableRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dynamodbconn

	result, err := conn.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: aws.String(d.Id()),
	})

	if err != nil {
		if isAWSErr(err, dynamodb.ErrCodeResourceNotFoundException, "") {
			log.Printf("[WARN] Dynamodb Table (%s) not found, error code (404)", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	err = flattenAwsDynamoDbTableResource(d, result.Table)
	if err != nil {
		return err
	}

	ttlOut, err := conn.DescribeTimeToLive(&dynamodb.DescribeTimeToLiveInput{
		TableName: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}
	if ttlOut.TimeToLiveDescription != nil {
		err := d.Set("ttl", flattenDynamoDbTtl(ttlOut.TimeToLiveDescription))
		if err != nil {
			return err
		}
	}

	tags, err := readDynamoDbTableTags(d.Get("arn").(string), conn)
	if err != nil {
		return err
	}
	d.Set("tags", tags)

	pitrOut, err := conn.DescribeContinuousBackups(&dynamodb.DescribeContinuousBackupsInput{
		TableName: aws.String(d.Id()),
	})
	if err != nil && !isAWSErr(err, "UnknownOperationException", "") {
		return err
	}
	d.Set("point_in_time_recovery", flattenDynamoDbPitr(pitrOut))

	return nil
}

func resourceAwsDynamoDbTableDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dynamodbconn

	log.Printf("[DEBUG] DynamoDB delete table: %s", d.Id())

	err := deleteAwsDynamoDbTable(d.Id(), conn)
	if err != nil {
		if isAWSErr(err, dynamodb.ErrCodeResourceNotFoundException, "Requested resource not found: Table: ") {
			return nil
		}
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending: []string{
			dynamodb.TableStatusActive,
			dynamodb.TableStatusDeleting,
		},
		Target:  []string{},
		Timeout: d.Timeout(schema.TimeoutDelete),
		Refresh: func() (interface{}, string, error) {
			out, err := conn.DescribeTable(&dynamodb.DescribeTableInput{
				TableName: aws.String(d.Id()),
			})
			if err != nil {
				if isAWSErr(err, dynamodb.ErrCodeResourceNotFoundException, "") {
					return nil, "", nil
				}

				return 42, "", err
			}
			table := out.Table

			return table, *table.TableStatus, nil
		},
	}
	_, err = stateConf.WaitForState()
	return err
}

func deleteAwsDynamoDbTable(tableName string, conn *dynamodb.DynamoDB) error {
	input := &dynamodb.DeleteTableInput{
		TableName: aws.String(tableName),
	}

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteTable(input)
		if err != nil {
			// Subscriber limit exceeded: Only 10 tables can be created, updated, or deleted simultaneously
			if isAWSErr(err, dynamodb.ErrCodeLimitExceededException, "simultaneously") {
				return resource.RetryableError(err)
			}
			// This handles multiple scenarios in the DynamoDB API:
			// 1. Updating a table immediately before deletion may return:
			//    ResourceInUseException: Attempt to change a resource which is still in use: Table is being updated:
			// 2. Removing a table from a DynamoDB global table may return:
			//    ResourceInUseException: Attempt to change a resource which is still in use: Table is being deleted:
			if isAWSErr(err, dynamodb.ErrCodeResourceInUseException, "") {
				return resource.RetryableError(err)
			}
			if isAWSErr(err, dynamodb.ErrCodeResourceNotFoundException, "Requested resource not found: Table: ") {
				return resource.NonRetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
}

func updateDynamoDbTimeToLive(d *schema.ResourceData, conn *dynamodb.DynamoDB) error {
	toBeEnabled := false
	attributeName := ""

	o, n := d.GetChange("ttl")
	newTtl, ok := n.(*schema.Set)
	blockExists := ok && newTtl.Len() > 0

	if blockExists {
		ttlList := newTtl.List()
		ttlMap := ttlList[0].(map[string]interface{})
		attributeName = ttlMap["attribute_name"].(string)
		toBeEnabled = ttlMap["enabled"].(bool)

	} else if !d.IsNewResource() {
		oldTtlList := o.(*schema.Set).List()
		ttlMap := oldTtlList[0].(map[string]interface{})
		attributeName = ttlMap["attribute_name"].(string)
		toBeEnabled = false
	}

	if attributeName != "" {
		_, err := conn.UpdateTimeToLive(&dynamodb.UpdateTimeToLiveInput{
			TableName: aws.String(d.Id()),
			TimeToLiveSpecification: &dynamodb.TimeToLiveSpecification{
				AttributeName: aws.String(attributeName),
				Enabled:       aws.Bool(toBeEnabled),
			},
		})
		if err != nil {
			if isAWSErr(err, "ValidationException", "TimeToLive is already disabled") {
				return nil
			}
			return err
		}

		err = waitForDynamoDbTtlUpdateToBeCompleted(d.Id(), toBeEnabled, conn)
		if err != nil {
			return fmt.Errorf("Error waiting for DynamoDB TimeToLive to be updated: %s", err)
		}
	}

	return nil
}

func updateDynamoDbPITR(d *schema.ResourceData, conn *dynamodb.DynamoDB) error {
	toEnable := d.Get("point_in_time_recovery.0.enabled").(bool)

	input := &dynamodb.UpdateContinuousBackupsInput{
		TableName: aws.String(d.Id()),
		PointInTimeRecoverySpecification: &dynamodb.PointInTimeRecoverySpecification{
			PointInTimeRecoveryEnabled: aws.Bool(toEnable),
		},
	}

	log.Printf("[DEBUG] Updating DynamoDB point in time recovery status to %v", toEnable)

	err := resource.Retry(20*time.Minute, func() *resource.RetryError {
		_, err := conn.UpdateContinuousBackups(input)
		if err != nil {
			// Backups are still being enabled for this newly created table
			if isAWSErr(err, dynamodb.ErrCodeContinuousBackupsUnavailableException, "Backups are being enabled") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})

	if err != nil {
		return err
	}

	if err := waitForDynamoDbBackupUpdateToBeCompleted(d.Id(), toEnable, conn); err != nil {
		return fmt.Errorf("Error waiting for DynamoDB PITR update: %s", err)
	}

	return nil
}

func readDynamoDbTableTags(arn string, conn *dynamodb.DynamoDB) (map[string]string, error) {
	output, err := conn.ListTagsOfResource(&dynamodb.ListTagsOfResourceInput{
		ResourceArn: aws.String(arn),
	})

	// Do not fail if interfacing with dynamodb-local
	if err != nil && !isAWSErr(err, "UnknownOperationException", "Tagging is not currently supported in DynamoDB Local.") {
		return nil, fmt.Errorf("Error reading tags from dynamodb resource: %s", err)
	}

	result := tagsToMapDynamoDb(output.Tags)

	// TODO Read NextToken if available

	return result, nil
}

// Waiters

func waitForDynamoDbGSIToBeActive(tableName string, gsiName string, conn *dynamodb.DynamoDB) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{
			dynamodb.IndexStatusCreating,
			dynamodb.IndexStatusUpdating,
		},
		Target:  []string{dynamodb.IndexStatusActive},
		Timeout: 10 * time.Minute,
		Refresh: func() (interface{}, string, error) {
			result, err := conn.DescribeTable(&dynamodb.DescribeTableInput{
				TableName: aws.String(tableName),
			})
			if err != nil {
				return 42, "", err
			}

			table := result.Table

			// Find index
			var targetGSI *dynamodb.GlobalSecondaryIndexDescription
			for _, gsi := range table.GlobalSecondaryIndexes {
				if *gsi.IndexName == gsiName {
					targetGSI = gsi
				}
			}

			if targetGSI != nil {
				return table, *targetGSI.IndexStatus, nil
			}

			return nil, "", nil
		},
	}
	_, err := stateConf.WaitForState()
	return err
}

func waitForDynamoDbGSIToBeDeleted(tableName string, gsiName string, conn *dynamodb.DynamoDB) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{
			dynamodb.IndexStatusActive,
			dynamodb.IndexStatusDeleting,
		},
		Target:  []string{},
		Timeout: 10 * time.Minute,
		Refresh: func() (interface{}, string, error) {
			result, err := conn.DescribeTable(&dynamodb.DescribeTableInput{
				TableName: aws.String(tableName),
			})
			if err != nil {
				return 42, "", err
			}

			table := result.Table

			// Find index
			var targetGSI *dynamodb.GlobalSecondaryIndexDescription
			for _, gsi := range table.GlobalSecondaryIndexes {
				if *gsi.IndexName == gsiName {
					targetGSI = gsi
				}
			}

			if targetGSI == nil {
				return nil, "", nil
			}

			return targetGSI, *targetGSI.IndexStatus, nil
		},
	}
	_, err := stateConf.WaitForState()
	return err
}

func waitForDynamoDbTableToBeActive(tableName string, timeout time.Duration, conn *dynamodb.DynamoDB) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{dynamodb.TableStatusCreating, dynamodb.TableStatusUpdating},
		Target:  []string{dynamodb.TableStatusActive},
		Timeout: timeout,
		Refresh: func() (interface{}, string, error) {
			result, err := conn.DescribeTable(&dynamodb.DescribeTableInput{
				TableName: aws.String(tableName),
			})
			if err != nil {
				return 42, "", err
			}

			return result, *result.Table.TableStatus, nil
		},
	}
	_, err := stateConf.WaitForState()

	return err
}

func waitForDynamoDbBackupUpdateToBeCompleted(tableName string, toEnable bool, conn *dynamodb.DynamoDB) error {
	var pending []string
	target := []string{dynamodb.TimeToLiveStatusDisabled}

	if toEnable {
		pending = []string{
			"ENABLING",
		}
		target = []string{dynamodb.PointInTimeRecoveryStatusEnabled}
	}

	stateConf := &resource.StateChangeConf{
		Pending: pending,
		Target:  target,
		Timeout: 10 * time.Second,
		Refresh: func() (interface{}, string, error) {
			result, err := conn.DescribeContinuousBackups(&dynamodb.DescribeContinuousBackupsInput{
				TableName: aws.String(tableName),
			})
			if err != nil {
				return 42, "", err
			}

			if result.ContinuousBackupsDescription == nil || result.ContinuousBackupsDescription.PointInTimeRecoveryDescription == nil {
				return 42, "", errors.New("Error reading backup status from dynamodb resource: empty description")
			}
			pitr := result.ContinuousBackupsDescription.PointInTimeRecoveryDescription

			return result, *pitr.PointInTimeRecoveryStatus, nil
		},
	}
	_, err := stateConf.WaitForState()
	return err
}

func waitForDynamoDbTtlUpdateToBeCompleted(tableName string, toEnable bool, conn *dynamodb.DynamoDB) error {
	pending := []string{
		dynamodb.TimeToLiveStatusEnabled,
		dynamodb.TimeToLiveStatusDisabling,
	}
	target := []string{dynamodb.TimeToLiveStatusDisabled}

	if toEnable {
		pending = []string{
			dynamodb.TimeToLiveStatusDisabled,
			dynamodb.TimeToLiveStatusEnabling,
		}
		target = []string{dynamodb.TimeToLiveStatusEnabled}
	}

	stateConf := &resource.StateChangeConf{
		Pending: pending,
		Target:  target,
		Timeout: 10 * time.Second,
		Refresh: func() (interface{}, string, error) {
			result, err := conn.DescribeTimeToLive(&dynamodb.DescribeTimeToLiveInput{
				TableName: aws.String(tableName),
			})
			if err != nil {
				return 42, "", err
			}

			ttlDesc := result.TimeToLiveDescription

			return result, *ttlDesc.TimeToLiveStatus, nil
		},
	}

	_, err := stateConf.WaitForState()
	return err
}

func isDynamoDbTableOptionDisabled(v interface{}) bool {
	options := v.([]interface{})
	if len(options) == 0 {
		return true
	}
	e := options[0].(map[string]interface{})["enabled"]
	return !e.(bool)
}
