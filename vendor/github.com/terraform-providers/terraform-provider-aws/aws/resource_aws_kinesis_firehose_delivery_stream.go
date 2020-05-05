package aws

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/hashicorp/terraform-plugin-sdk/helper/hashcode"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

const (
	firehoseDeliveryStreamStatusDeleted = "DESTROYED"
)

func cloudWatchLoggingOptionsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		MaxItems: 1,
		Optional: true,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"enabled": {
					Type:     schema.TypeBool,
					Optional: true,
					Default:  false,
				},

				"log_group_name": {
					Type:     schema.TypeString,
					Optional: true,
				},

				"log_stream_name": {
					Type:     schema.TypeString,
					Optional: true,
				},
			},
		},
	}
}

func s3ConfigurationSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		MaxItems: 1,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"bucket_arn": {
					Type:     schema.TypeString,
					Required: true,
				},

				"buffer_size": {
					Type:     schema.TypeInt,
					Optional: true,
					Default:  5,
				},

				"buffer_interval": {
					Type:     schema.TypeInt,
					Optional: true,
					Default:  300,
				},

				"compression_format": {
					Type:     schema.TypeString,
					Optional: true,
					Default:  "UNCOMPRESSED",
				},

				"kms_key_arn": {
					Type:         schema.TypeString,
					Optional:     true,
					ValidateFunc: validateArn,
				},

				"role_arn": {
					Type:     schema.TypeString,
					Required: true,
				},

				"prefix": {
					Type:     schema.TypeString,
					Optional: true,
				},

				"cloudwatch_logging_options": cloudWatchLoggingOptionsSchema(),
			},
		},
	}
}

func processingConfigurationSchema() *schema.Schema {
	return &schema.Schema{
		Type:             schema.TypeList,
		Optional:         true,
		MaxItems:         1,
		DiffSuppressFunc: suppressMissingOptionalConfigurationBlock,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"enabled": {
					Type:     schema.TypeBool,
					Optional: true,
				},
				"processors": {
					Type:     schema.TypeList,
					Optional: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"parameters": {
								Type:     schema.TypeList,
								Optional: true,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"parameter_name": {
											Type:     schema.TypeString,
											Required: true,
											ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
												value := v.(string)
												if value != "LambdaArn" && value != "NumberOfRetries" && value != "RoleArn" && value != "BufferSizeInMBs" && value != "BufferIntervalInSeconds" {
													errors = append(errors, fmt.Errorf(
														"%q must be one of 'LambdaArn', 'NumberOfRetries', 'RoleArn', 'BufferSizeInMBs', 'BufferIntervalInSeconds'", k))
												}
												return
											},
										},
										"parameter_value": {
											Type:     schema.TypeString,
											Required: true,
											ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
												value := v.(string)
												if len(value) < 1 || len(value) > 512 {
													errors = append(errors, fmt.Errorf(
														"%q must be at least one character long and at most 512 characters long", k))
												}
												return
											},
										},
									},
								},
							},
							"type": {
								Type:     schema.TypeString,
								Required: true,
								ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
									value := v.(string)
									if value != "Lambda" {
										errors = append(errors, fmt.Errorf(
											"%q must be 'Lambda'", k))
									}
									return
								},
							},
						},
					},
				},
			},
		},
	}
}

func cloudwatchLoggingOptionsHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%t-", m["enabled"].(bool)))
	if m["enabled"].(bool) {
		buf.WriteString(fmt.Sprintf("%s-", m["log_group_name"].(string)))
		buf.WriteString(fmt.Sprintf("%s-", m["log_stream_name"].(string)))
	}
	return hashcode.String(buf.String())
}

func flattenCloudwatchLoggingOptions(clo *firehose.CloudWatchLoggingOptions) *schema.Set {
	if clo == nil {
		return schema.NewSet(cloudwatchLoggingOptionsHash, []interface{}{})
	}

	cloudwatchLoggingOptions := map[string]interface{}{
		"enabled": aws.BoolValue(clo.Enabled),
	}
	if aws.BoolValue(clo.Enabled) {
		cloudwatchLoggingOptions["log_group_name"] = aws.StringValue(clo.LogGroupName)
		cloudwatchLoggingOptions["log_stream_name"] = aws.StringValue(clo.LogStreamName)
	}
	return schema.NewSet(cloudwatchLoggingOptionsHash, []interface{}{cloudwatchLoggingOptions})
}

func flattenFirehoseElasticsearchConfiguration(description *firehose.ElasticsearchDestinationDescription) []map[string]interface{} {
	if description == nil {
		return []map[string]interface{}{}
	}

	m := map[string]interface{}{
		"cloudwatch_logging_options": flattenCloudwatchLoggingOptions(description.CloudWatchLoggingOptions),
		"domain_arn":                 aws.StringValue(description.DomainARN),
		"role_arn":                   aws.StringValue(description.RoleARN),
		"type_name":                  aws.StringValue(description.TypeName),
		"index_name":                 aws.StringValue(description.IndexName),
		"s3_backup_mode":             aws.StringValue(description.S3BackupMode),
		"index_rotation_period":      aws.StringValue(description.IndexRotationPeriod),
		"processing_configuration":   flattenProcessingConfiguration(description.ProcessingConfiguration, aws.StringValue(description.RoleARN)),
	}

	if description.BufferingHints != nil {
		m["buffering_interval"] = int(aws.Int64Value(description.BufferingHints.IntervalInSeconds))
		m["buffering_size"] = int(aws.Int64Value(description.BufferingHints.SizeInMBs))
	}

	if description.RetryOptions != nil {
		m["retry_duration"] = int(aws.Int64Value(description.RetryOptions.DurationInSeconds))
	}

	return []map[string]interface{}{m}
}

func flattenFirehoseExtendedS3Configuration(description *firehose.ExtendedS3DestinationDescription) []map[string]interface{} {
	if description == nil {
		return []map[string]interface{}{}
	}

	m := map[string]interface{}{
		"bucket_arn":                           aws.StringValue(description.BucketARN),
		"cloudwatch_logging_options":           flattenCloudwatchLoggingOptions(description.CloudWatchLoggingOptions),
		"compression_format":                   aws.StringValue(description.CompressionFormat),
		"data_format_conversion_configuration": flattenFirehoseDataFormatConversionConfiguration(description.DataFormatConversionConfiguration),
		"error_output_prefix":                  aws.StringValue(description.ErrorOutputPrefix),
		"prefix":                               aws.StringValue(description.Prefix),
		"processing_configuration":             flattenProcessingConfiguration(description.ProcessingConfiguration, aws.StringValue(description.RoleARN)),
		"role_arn":                             aws.StringValue(description.RoleARN),
		"s3_backup_configuration":              flattenFirehoseS3Configuration(description.S3BackupDescription),
		"s3_backup_mode":                       aws.StringValue(description.S3BackupMode),
	}

	if description.BufferingHints != nil {
		m["buffer_interval"] = int(aws.Int64Value(description.BufferingHints.IntervalInSeconds))
		m["buffer_size"] = int(aws.Int64Value(description.BufferingHints.SizeInMBs))
	}

	if description.EncryptionConfiguration != nil && description.EncryptionConfiguration.KMSEncryptionConfig != nil {
		m["kms_key_arn"] = aws.StringValue(description.EncryptionConfiguration.KMSEncryptionConfig.AWSKMSKeyARN)
	}

	return []map[string]interface{}{m}
}

func flattenFirehoseRedshiftConfiguration(description *firehose.RedshiftDestinationDescription, configuredPassword string) []map[string]interface{} {
	if description == nil {
		return []map[string]interface{}{}
	}

	m := map[string]interface{}{
		"cloudwatch_logging_options": flattenCloudwatchLoggingOptions(description.CloudWatchLoggingOptions),
		"cluster_jdbcurl":            aws.StringValue(description.ClusterJDBCURL),
		"password":                   configuredPassword,
		"processing_configuration":   flattenProcessingConfiguration(description.ProcessingConfiguration, aws.StringValue(description.RoleARN)),
		"role_arn":                   aws.StringValue(description.RoleARN),
		"s3_backup_configuration":    flattenFirehoseS3Configuration(description.S3BackupDescription),
		"s3_backup_mode":             aws.StringValue(description.S3BackupMode),
		"username":                   aws.StringValue(description.Username),
	}

	if description.CopyCommand != nil {
		m["copy_options"] = aws.StringValue(description.CopyCommand.CopyOptions)
		m["data_table_columns"] = aws.StringValue(description.CopyCommand.DataTableColumns)
		m["data_table_name"] = aws.StringValue(description.CopyCommand.DataTableName)
	}

	if description.RetryOptions != nil {
		m["retry_duration"] = int(aws.Int64Value(description.RetryOptions.DurationInSeconds))
	}

	return []map[string]interface{}{m}
}

func flattenFirehoseSplunkConfiguration(description *firehose.SplunkDestinationDescription) []map[string]interface{} {
	if description == nil {
		return []map[string]interface{}{}
	}
	m := map[string]interface{}{
		"cloudwatch_logging_options": flattenCloudwatchLoggingOptions(description.CloudWatchLoggingOptions),
		"hec_acknowledgment_timeout": int(aws.Int64Value(description.HECAcknowledgmentTimeoutInSeconds)),
		"hec_endpoint_type":          aws.StringValue(description.HECEndpointType),
		"hec_endpoint":               aws.StringValue(description.HECEndpoint),
		"hec_token":                  aws.StringValue(description.HECToken),
		"processing_configuration":   flattenProcessingConfiguration(description.ProcessingConfiguration, ""),
		"s3_backup_mode":             aws.StringValue(description.S3BackupMode),
	}

	if description.RetryOptions != nil {
		m["retry_duration"] = int(aws.Int64Value(description.RetryOptions.DurationInSeconds))
	}

	return []map[string]interface{}{m}
}

func flattenFirehoseS3Configuration(description *firehose.S3DestinationDescription) []map[string]interface{} {
	if description == nil {
		return []map[string]interface{}{}
	}

	m := map[string]interface{}{
		"bucket_arn":                 aws.StringValue(description.BucketARN),
		"cloudwatch_logging_options": flattenCloudwatchLoggingOptions(description.CloudWatchLoggingOptions),
		"compression_format":         aws.StringValue(description.CompressionFormat),
		"prefix":                     aws.StringValue(description.Prefix),
		"role_arn":                   aws.StringValue(description.RoleARN),
	}

	if description.BufferingHints != nil {
		m["buffer_interval"] = int(aws.Int64Value(description.BufferingHints.IntervalInSeconds))
		m["buffer_size"] = int(aws.Int64Value(description.BufferingHints.SizeInMBs))
	}

	if description.EncryptionConfiguration != nil && description.EncryptionConfiguration.KMSEncryptionConfig != nil {
		m["kms_key_arn"] = aws.StringValue(description.EncryptionConfiguration.KMSEncryptionConfig.AWSKMSKeyARN)
	}

	return []map[string]interface{}{m}
}

func flattenFirehoseDataFormatConversionConfiguration(dfcc *firehose.DataFormatConversionConfiguration) []map[string]interface{} {
	if dfcc == nil {
		return []map[string]interface{}{}
	}

	enabled := aws.BoolValue(dfcc.Enabled)
	ifc := flattenFirehoseInputFormatConfiguration(dfcc.InputFormatConfiguration)
	ofc := flattenFirehoseOutputFormatConfiguration(dfcc.OutputFormatConfiguration)
	sc := flattenFirehoseSchemaConfiguration(dfcc.SchemaConfiguration)

	// The AWS SDK can represent "no data format conversion configuration" in two ways:
	// 1. With a nil value
	// 2. With enabled set to false and nil for ALL the config sections.
	// We normalize this with an empty configuration in the state due
	// to the existing Default: true on the enabled attribute.
	if !enabled && len(ifc) == 0 && len(ofc) == 0 && len(sc) == 0 {
		return []map[string]interface{}{}
	}

	m := map[string]interface{}{
		"enabled":                     enabled,
		"input_format_configuration":  ifc,
		"output_format_configuration": ofc,
		"schema_configuration":        sc,
	}

	return []map[string]interface{}{m}
}

func flattenFirehoseInputFormatConfiguration(ifc *firehose.InputFormatConfiguration) []map[string]interface{} {
	if ifc == nil {
		return []map[string]interface{}{}
	}

	m := map[string]interface{}{
		"deserializer": flattenFirehoseDeserializer(ifc.Deserializer),
	}

	return []map[string]interface{}{m}
}

func flattenFirehoseDeserializer(deserializer *firehose.Deserializer) []map[string]interface{} {
	if deserializer == nil {
		return []map[string]interface{}{}
	}

	m := map[string]interface{}{
		"hive_json_ser_de":   flattenFirehoseHiveJsonSerDe(deserializer.HiveJsonSerDe),
		"open_x_json_ser_de": flattenFirehoseOpenXJsonSerDe(deserializer.OpenXJsonSerDe),
	}

	return []map[string]interface{}{m}
}

func flattenFirehoseHiveJsonSerDe(hjsd *firehose.HiveJsonSerDe) []map[string]interface{} {
	if hjsd == nil {
		return []map[string]interface{}{}
	}

	m := map[string]interface{}{
		"timestamp_formats": flattenStringList(hjsd.TimestampFormats),
	}

	return []map[string]interface{}{m}
}

func flattenFirehoseOpenXJsonSerDe(oxjsd *firehose.OpenXJsonSerDe) []map[string]interface{} {
	if oxjsd == nil {
		return []map[string]interface{}{}
	}

	m := map[string]interface{}{
		"column_to_json_key_mappings":              aws.StringValueMap(oxjsd.ColumnToJsonKeyMappings),
		"convert_dots_in_json_keys_to_underscores": aws.BoolValue(oxjsd.ConvertDotsInJsonKeysToUnderscores),
	}

	// API omits default values
	// Return defaults that are not type zero values to prevent extraneous difference

	m["case_insensitive"] = true
	if oxjsd.CaseInsensitive != nil {
		m["case_insensitive"] = aws.BoolValue(oxjsd.CaseInsensitive)
	}

	return []map[string]interface{}{m}
}

func flattenFirehoseOutputFormatConfiguration(ofc *firehose.OutputFormatConfiguration) []map[string]interface{} {
	if ofc == nil {
		return []map[string]interface{}{}
	}

	m := map[string]interface{}{
		"serializer": flattenFirehoseSerializer(ofc.Serializer),
	}

	return []map[string]interface{}{m}
}

func flattenFirehoseSerializer(serializer *firehose.Serializer) []map[string]interface{} {
	if serializer == nil {
		return []map[string]interface{}{}
	}

	m := map[string]interface{}{
		"orc_ser_de":     flattenFirehoseOrcSerDe(serializer.OrcSerDe),
		"parquet_ser_de": flattenFirehoseParquetSerDe(serializer.ParquetSerDe),
	}

	return []map[string]interface{}{m}
}

func flattenFirehoseOrcSerDe(osd *firehose.OrcSerDe) []map[string]interface{} {
	if osd == nil {
		return []map[string]interface{}{}
	}

	m := map[string]interface{}{
		"bloom_filter_columns":     aws.StringValueSlice(osd.BloomFilterColumns),
		"dictionary_key_threshold": aws.Float64Value(osd.DictionaryKeyThreshold),
		"enable_padding":           aws.BoolValue(osd.EnablePadding),
	}

	// API omits default values
	// Return defaults that are not type zero values to prevent extraneous difference

	m["block_size_bytes"] = 268435456
	if osd.BlockSizeBytes != nil {
		m["block_size_bytes"] = int(aws.Int64Value(osd.BlockSizeBytes))
	}

	m["bloom_filter_false_positive_probability"] = 0.05
	if osd.BloomFilterFalsePositiveProbability != nil {
		m["bloom_filter_false_positive_probability"] = aws.Float64Value(osd.BloomFilterFalsePositiveProbability)
	}

	m["compression"] = firehose.OrcCompressionSnappy
	if osd.Compression != nil {
		m["compression"] = aws.StringValue(osd.Compression)
	}

	m["format_version"] = firehose.OrcFormatVersionV012
	if osd.FormatVersion != nil {
		m["format_version"] = aws.StringValue(osd.FormatVersion)
	}

	m["padding_tolerance"] = 0.05
	if osd.PaddingTolerance != nil {
		m["padding_tolerance"] = aws.Float64Value(osd.PaddingTolerance)
	}

	m["row_index_stride"] = 10000
	if osd.RowIndexStride != nil {
		m["row_index_stride"] = int(aws.Int64Value(osd.RowIndexStride))
	}

	m["stripe_size_bytes"] = 67108864
	if osd.StripeSizeBytes != nil {
		m["stripe_size_bytes"] = int(aws.Int64Value(osd.StripeSizeBytes))
	}

	return []map[string]interface{}{m}
}

func flattenFirehoseParquetSerDe(psd *firehose.ParquetSerDe) []map[string]interface{} {
	if psd == nil {
		return []map[string]interface{}{}
	}

	m := map[string]interface{}{
		"enable_dictionary_compression": aws.BoolValue(psd.EnableDictionaryCompression),
		"max_padding_bytes":             int(aws.Int64Value(psd.MaxPaddingBytes)),
	}

	// API omits default values
	// Return defaults that are not type zero values to prevent extraneous difference

	m["block_size_bytes"] = 268435456
	if psd.BlockSizeBytes != nil {
		m["block_size_bytes"] = int(aws.Int64Value(psd.BlockSizeBytes))
	}

	m["compression"] = firehose.ParquetCompressionSnappy
	if psd.Compression != nil {
		m["compression"] = aws.StringValue(psd.Compression)
	}

	m["page_size_bytes"] = 1048576
	if psd.PageSizeBytes != nil {
		m["page_size_bytes"] = int(aws.Int64Value(psd.PageSizeBytes))
	}

	m["writer_version"] = firehose.ParquetWriterVersionV1
	if psd.WriterVersion != nil {
		m["writer_version"] = aws.StringValue(psd.WriterVersion)
	}

	return []map[string]interface{}{m}
}

func flattenFirehoseSchemaConfiguration(sc *firehose.SchemaConfiguration) []map[string]interface{} {
	if sc == nil {
		return []map[string]interface{}{}
	}

	m := map[string]interface{}{
		"catalog_id":    aws.StringValue(sc.CatalogId),
		"database_name": aws.StringValue(sc.DatabaseName),
		"region":        aws.StringValue(sc.Region),
		"role_arn":      aws.StringValue(sc.RoleARN),
		"table_name":    aws.StringValue(sc.TableName),
		"version_id":    aws.StringValue(sc.VersionId),
	}

	return []map[string]interface{}{m}
}

func flattenProcessingConfiguration(pc *firehose.ProcessingConfiguration, roleArn string) []map[string]interface{} {
	if pc == nil {
		return []map[string]interface{}{}
	}

	processingConfiguration := make([]map[string]interface{}, 1)

	// It is necessary to explicitly filter this out
	// to prevent diffs during routine use and retain the ability
	// to show diffs if any field has drifted
	defaultLambdaParams := map[string]string{
		"NumberOfRetries":         "3",
		"RoleArn":                 roleArn,
		"BufferSizeInMBs":         "3",
		"BufferIntervalInSeconds": "60",
	}

	processors := make([]interface{}, len(pc.Processors))
	for i, p := range pc.Processors {
		t := aws.StringValue(p.Type)
		parameters := make([]interface{}, 0)

		for _, params := range p.Parameters {
			name := aws.StringValue(params.ParameterName)
			value := aws.StringValue(params.ParameterValue)

			if t == firehose.ProcessorTypeLambda {
				// Ignore defaults
				if v, ok := defaultLambdaParams[name]; ok && v == value {
					continue
				}
			}

			parameters = append(parameters, map[string]interface{}{
				"parameter_name":  name,
				"parameter_value": value,
			})
		}

		processors[i] = map[string]interface{}{
			"type":       t,
			"parameters": parameters,
		}
	}
	processingConfiguration[0] = map[string]interface{}{
		"enabled":    aws.BoolValue(pc.Enabled),
		"processors": processors,
	}
	return processingConfiguration
}

func flattenKinesisFirehoseDeliveryStream(d *schema.ResourceData, s *firehose.DeliveryStreamDescription) error {
	d.Set("version_id", s.VersionId)
	d.Set("arn", s.DeliveryStreamARN)
	d.Set("name", s.DeliveryStreamName)

	sseOptions := map[string]interface{}{
		"enabled": false,
	}
	if s.DeliveryStreamEncryptionConfiguration != nil && aws.StringValue(s.DeliveryStreamEncryptionConfiguration.Status) == firehose.DeliveryStreamEncryptionStatusEnabled {
		sseOptions["enabled"] = true
	}
	if err := d.Set("server_side_encryption", []map[string]interface{}{sseOptions}); err != nil {
		return fmt.Errorf("error setting server_side_encryption: %s", err)
	}

	if len(s.Destinations) > 0 {
		destination := s.Destinations[0]
		if destination.RedshiftDestinationDescription != nil {
			d.Set("destination", "redshift")
			configuredPassword := d.Get("redshift_configuration.0.password").(string)
			if err := d.Set("redshift_configuration", flattenFirehoseRedshiftConfiguration(destination.RedshiftDestinationDescription, configuredPassword)); err != nil {
				return fmt.Errorf("error setting redshift_configuration: %s", err)
			}
			if err := d.Set("s3_configuration", flattenFirehoseS3Configuration(destination.RedshiftDestinationDescription.S3DestinationDescription)); err != nil {
				return fmt.Errorf("error setting s3_configuration: %s", err)
			}
		} else if destination.ElasticsearchDestinationDescription != nil {
			d.Set("destination", "elasticsearch")
			if err := d.Set("elasticsearch_configuration", flattenFirehoseElasticsearchConfiguration(destination.ElasticsearchDestinationDescription)); err != nil {
				return fmt.Errorf("error setting elasticsearch_configuration: %s", err)
			}
			if err := d.Set("s3_configuration", flattenFirehoseS3Configuration(destination.ElasticsearchDestinationDescription.S3DestinationDescription)); err != nil {
				return fmt.Errorf("error setting s3_configuration: %s", err)
			}
		} else if destination.SplunkDestinationDescription != nil {
			d.Set("destination", "splunk")
			if err := d.Set("splunk_configuration", flattenFirehoseSplunkConfiguration(destination.SplunkDestinationDescription)); err != nil {
				return fmt.Errorf("error setting splunk_configuration: %s", err)
			}
			if err := d.Set("s3_configuration", flattenFirehoseS3Configuration(destination.SplunkDestinationDescription.S3DestinationDescription)); err != nil {
				return fmt.Errorf("error setting s3_configuration: %s", err)
			}
		} else if d.Get("destination").(string) == "s3" {
			d.Set("destination", "s3")
			if err := d.Set("s3_configuration", flattenFirehoseS3Configuration(destination.S3DestinationDescription)); err != nil {
				return fmt.Errorf("error setting s3_configuration: %s", err)
			}
		} else {
			d.Set("destination", "extended_s3")
			if err := d.Set("extended_s3_configuration", flattenFirehoseExtendedS3Configuration(destination.ExtendedS3DestinationDescription)); err != nil {
				return fmt.Errorf("error setting extended_s3_configuration: %s", err)
			}
		}
		d.Set("destination_id", destination.DestinationId)
	}
	return nil
}

func resourceAwsKinesisFirehoseDeliveryStream() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsKinesisFirehoseDeliveryStreamCreate,
		Read:   resourceAwsKinesisFirehoseDeliveryStreamRead,
		Update: resourceAwsKinesisFirehoseDeliveryStreamUpdate,
		Delete: resourceAwsKinesisFirehoseDeliveryStreamDelete,

		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				idErr := fmt.Errorf("Expected ID in format of arn:PARTITION:firehose:REGION:ACCOUNTID:deliverystream/NAME and provided: %s", d.Id())
				resARN, err := arn.Parse(d.Id())
				if err != nil {
					return nil, idErr
				}
				resourceParts := strings.Split(resARN.Resource, "/")
				if len(resourceParts) != 2 {
					return nil, idErr
				}
				d.Set("name", resourceParts[1])
				return []*schema.ResourceData{d}, nil
			},
		},

		SchemaVersion: 1,
		MigrateState:  resourceAwsKinesisFirehoseMigrateState,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if len(value) > 64 {
						errors = append(errors, fmt.Errorf(
							"%q cannot be longer than 64 characters", k))
					}
					return
				},
			},

			"tags": tagsSchema(),

			"server_side_encryption": {
				Type:             schema.TypeList,
				Optional:         true,
				MaxItems:         1,
				DiffSuppressFunc: suppressMissingOptionalConfigurationBlock,
				ConflictsWith:    []string{"kinesis_source_configuration"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
			},

			"kinesis_source_configuration": {
				Type:          schema.TypeList,
				ForceNew:      true,
				Optional:      true,
				MaxItems:      1,
				ConflictsWith: []string{"server_side_encryption"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"kinesis_stream_arn": {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validateArn,
						},

						"role_arn": {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validateArn,
						},
					},
				},
			},

			"destination": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				StateFunc: func(v interface{}) string {
					value := v.(string)
					return strings.ToLower(value)
				},
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if value != "s3" && value != "extended_s3" && value != "redshift" && value != "elasticsearch" && value != "splunk" {
						errors = append(errors, fmt.Errorf(
							"%q must be one of 's3', 'extended_s3', 'redshift', 'elasticsearch', 'splunk'", k))
					}
					return
				},
			},

			"s3_configuration": s3ConfigurationSchema(),

			"extended_s3_configuration": {
				Type:          schema.TypeList,
				Optional:      true,
				ConflictsWith: []string{"s3_configuration"},
				MaxItems:      1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"bucket_arn": {
							Type:     schema.TypeString,
							Required: true,
						},

						"buffer_size": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  5,
						},

						"buffer_interval": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  300,
						},

						"compression_format": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "UNCOMPRESSED",
						},

						"data_format_conversion_configuration": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"enabled": {
										Type:     schema.TypeBool,
										Optional: true,
										Default:  true,
									},
									"input_format_configuration": {
										Type:     schema.TypeList,
										Required: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"deserializer": {
													Type:     schema.TypeList,
													Required: true,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"hive_json_ser_de": {
																Type:          schema.TypeList,
																Optional:      true,
																MaxItems:      1,
																ConflictsWith: []string{"extended_s3_configuration.0.data_format_conversion_configuration.0.input_format_configuration.0.deserializer.0.open_x_json_ser_de"},
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"timestamp_formats": {
																			Type:     schema.TypeList,
																			Optional: true,
																			Elem:     &schema.Schema{Type: schema.TypeString},
																		},
																	},
																},
															},
															"open_x_json_ser_de": {
																Type:          schema.TypeList,
																Optional:      true,
																MaxItems:      1,
																ConflictsWith: []string{"extended_s3_configuration.0.data_format_conversion_configuration.0.input_format_configuration.0.deserializer.0.hive_json_ser_de"},
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"case_insensitive": {
																			Type:     schema.TypeBool,
																			Optional: true,
																			Default:  true,
																		},
																		"column_to_json_key_mappings": {
																			Type:     schema.TypeMap,
																			Optional: true,
																			Elem:     &schema.Schema{Type: schema.TypeString},
																		},
																		"convert_dots_in_json_keys_to_underscores": {
																			Type:     schema.TypeBool,
																			Optional: true,
																			Default:  false,
																		},
																	},
																},
															},
														},
													},
												},
											},
										},
									},
									"output_format_configuration": {
										Type:     schema.TypeList,
										Required: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"serializer": {
													Type:     schema.TypeList,
													Required: true,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"orc_ser_de": {
																Type:          schema.TypeList,
																Optional:      true,
																MaxItems:      1,
																ConflictsWith: []string{"extended_s3_configuration.0.data_format_conversion_configuration.0.output_format_configuration.0.serializer.0.parquet_ser_de"},
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"block_size_bytes": {
																			Type:     schema.TypeInt,
																			Optional: true,
																			// 256 MiB
																			Default: 268435456,
																			// 64 MiB
																			ValidateFunc: validation.IntAtLeast(67108864),
																		},
																		"bloom_filter_columns": {
																			Type:     schema.TypeList,
																			Optional: true,
																			Elem:     &schema.Schema{Type: schema.TypeString},
																		},
																		"bloom_filter_false_positive_probability": {
																			Type:     schema.TypeFloat,
																			Optional: true,
																			Default:  0.05,
																		},
																		"compression": {
																			Type:     schema.TypeString,
																			Optional: true,
																			Default:  firehose.OrcCompressionSnappy,
																			ValidateFunc: validation.StringInSlice([]string{
																				firehose.OrcCompressionNone,
																				firehose.OrcCompressionSnappy,
																				firehose.OrcCompressionZlib,
																			}, false),
																		},
																		"dictionary_key_threshold": {
																			Type:     schema.TypeFloat,
																			Optional: true,
																			Default:  0.0,
																		},
																		"enable_padding": {
																			Type:     schema.TypeBool,
																			Optional: true,
																			Default:  false,
																		},
																		"format_version": {
																			Type:     schema.TypeString,
																			Optional: true,
																			Default:  firehose.OrcFormatVersionV012,
																			ValidateFunc: validation.StringInSlice([]string{
																				firehose.OrcFormatVersionV011,
																				firehose.OrcFormatVersionV012,
																			}, false),
																		},
																		"padding_tolerance": {
																			Type:     schema.TypeFloat,
																			Optional: true,
																			Default:  0.05,
																		},
																		"row_index_stride": {
																			Type:         schema.TypeInt,
																			Optional:     true,
																			Default:      10000,
																			ValidateFunc: validation.IntAtLeast(1000),
																		},
																		"stripe_size_bytes": {
																			Type:     schema.TypeInt,
																			Optional: true,
																			// 64 MiB
																			Default: 67108864,
																			// 8 MiB
																			ValidateFunc: validation.IntAtLeast(8388608),
																		},
																	},
																},
															},
															"parquet_ser_de": {
																Type:          schema.TypeList,
																Optional:      true,
																MaxItems:      1,
																ConflictsWith: []string{"extended_s3_configuration.0.data_format_conversion_configuration.0.output_format_configuration.0.serializer.0.orc_ser_de"},
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"block_size_bytes": {
																			Type:     schema.TypeInt,
																			Optional: true,
																			// 256 MiB
																			Default: 268435456,
																			// 64 MiB
																			ValidateFunc: validation.IntAtLeast(67108864),
																		},
																		"compression": {
																			Type:     schema.TypeString,
																			Optional: true,
																			Default:  firehose.ParquetCompressionSnappy,
																			ValidateFunc: validation.StringInSlice([]string{
																				firehose.ParquetCompressionGzip,
																				firehose.ParquetCompressionSnappy,
																				firehose.ParquetCompressionUncompressed,
																			}, false),
																		},
																		"enable_dictionary_compression": {
																			Type:     schema.TypeBool,
																			Optional: true,
																			Default:  false,
																		},
																		"max_padding_bytes": {
																			Type:     schema.TypeInt,
																			Optional: true,
																			Default:  0,
																		},
																		"page_size_bytes": {
																			Type:     schema.TypeInt,
																			Optional: true,
																			// 1 MiB
																			Default: 1048576,
																			// 64 KiB
																			ValidateFunc: validation.IntAtLeast(65536),
																		},
																		"writer_version": {
																			Type:     schema.TypeString,
																			Optional: true,
																			Default:  firehose.ParquetWriterVersionV1,
																			ValidateFunc: validation.StringInSlice([]string{
																				firehose.ParquetWriterVersionV1,
																				firehose.ParquetWriterVersionV2,
																			}, false),
																		},
																	},
																},
															},
														},
													},
												},
											},
										},
									},
									"schema_configuration": {
										Type:     schema.TypeList,
										Required: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"catalog_id": {
													Type:     schema.TypeString,
													Optional: true,
													Computed: true,
												},
												"database_name": {
													Type:     schema.TypeString,
													Required: true,
												},
												"region": {
													Type:     schema.TypeString,
													Optional: true,
													Computed: true,
												},
												"role_arn": {
													Type:     schema.TypeString,
													Required: true,
												},
												"table_name": {
													Type:     schema.TypeString,
													Required: true,
												},
												"version_id": {
													Type:     schema.TypeString,
													Optional: true,
													Default:  "LATEST",
												},
											},
										},
									},
								},
							},
						},

						"error_output_prefix": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"kms_key_arn": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateArn,
						},

						"role_arn": {
							Type:     schema.TypeString,
							Required: true,
						},

						"prefix": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"s3_backup_mode": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "Disabled",
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								value := v.(string)
								if value != "Disabled" && value != "Enabled" {
									errors = append(errors, fmt.Errorf(
										"%q must be one of 'Disabled', 'Enabled'", k))
								}
								return
							},
						},

						"s3_backup_configuration": s3ConfigurationSchema(),

						"cloudwatch_logging_options": cloudWatchLoggingOptionsSchema(),

						"processing_configuration": processingConfigurationSchema(),
					},
				},
			},

			"redshift_configuration": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cluster_jdbcurl": {
							Type:     schema.TypeString,
							Required: true,
						},

						"username": {
							Type:     schema.TypeString,
							Required: true,
						},

						"password": {
							Type:      schema.TypeString,
							Required:  true,
							Sensitive: true,
						},

						"processing_configuration": processingConfigurationSchema(),

						"role_arn": {
							Type:     schema.TypeString,
							Required: true,
						},

						"s3_backup_mode": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "Disabled",
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								value := v.(string)
								if value != "Disabled" && value != "Enabled" {
									errors = append(errors, fmt.Errorf(
										"%q must be one of 'Disabled', 'Enabled'", k))
								}
								return
							},
						},

						"s3_backup_configuration": s3ConfigurationSchema(),

						"retry_duration": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  3600,
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								value := v.(int)
								if value < 0 || value > 7200 {
									errors = append(errors, fmt.Errorf(
										"%q must be in the range from 0 to 7200 seconds.", k))
								}
								return
							},
						},

						"copy_options": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"data_table_columns": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"data_table_name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"cloudwatch_logging_options": cloudWatchLoggingOptionsSchema(),
					},
				},
			},

			"elasticsearch_configuration": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"buffering_interval": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  300,
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								value := v.(int)
								if value < 60 || value > 900 {
									errors = append(errors, fmt.Errorf(
										"%q must be in the range from 60 to 900 seconds.", k))
								}
								return
							},
						},

						"buffering_size": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  5,
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								value := v.(int)
								if value < 1 || value > 100 {
									errors = append(errors, fmt.Errorf(
										"%q must be in the range from 1 to 100 MB.", k))
								}
								return
							},
						},

						"domain_arn": {
							Type:     schema.TypeString,
							Required: true,
						},

						"index_name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"index_rotation_period": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "OneDay",
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								value := v.(string)
								if value != "NoRotation" && value != "OneHour" && value != "OneDay" && value != "OneWeek" && value != "OneMonth" {
									errors = append(errors, fmt.Errorf(
										"%q must be one of 'NoRotation', 'OneHour', 'OneDay', 'OneWeek', 'OneMonth'", k))
								}
								return
							},
						},

						"retry_duration": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  300,
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								value := v.(int)
								if value < 0 || value > 7200 {
									errors = append(errors, fmt.Errorf(
										"%q must be in the range from 0 to 7200 seconds.", k))
								}
								return
							},
						},

						"role_arn": {
							Type:     schema.TypeString,
							Required: true,
						},

						"s3_backup_mode": {
							Type:     schema.TypeString,
							ForceNew: true,
							Optional: true,
							Default:  "FailedDocumentsOnly",
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								value := v.(string)
								if value != "FailedDocumentsOnly" && value != "AllDocuments" {
									errors = append(errors, fmt.Errorf(
										"%q must be one of 'FailedDocumentsOnly', 'AllDocuments'", k))
								}
								return
							},
						},

						"type_name": {
							Type:     schema.TypeString,
							Optional: true,
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								value := v.(string)
								if len(value) > 100 {
									errors = append(errors, fmt.Errorf(
										"%q cannot be longer than 100 characters", k))
								}
								return
							},
						},

						"cloudwatch_logging_options": cloudWatchLoggingOptionsSchema(),

						"processing_configuration": processingConfigurationSchema(),
					},
				},
			},

			"splunk_configuration": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"hec_acknowledgment_timeout": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      180,
							ValidateFunc: validation.IntBetween(180, 600),
						},

						"hec_endpoint": {
							Type:     schema.TypeString,
							Required: true,
						},

						"hec_endpoint_type": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  firehose.HECEndpointTypeRaw,
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								value := v.(string)
								if value != firehose.HECEndpointTypeRaw && value != firehose.HECEndpointTypeEvent {
									errors = append(errors, fmt.Errorf(
										"%q must be one of 'Raw', 'Event'", k))
								}
								return
							},
						},

						"hec_token": {
							Type:     schema.TypeString,
							Required: true,
						},

						"s3_backup_mode": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  firehose.SplunkS3BackupModeFailedEventsOnly,
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								value := v.(string)
								if value != firehose.SplunkS3BackupModeFailedEventsOnly && value != firehose.SplunkS3BackupModeAllEvents {
									errors = append(errors, fmt.Errorf(
										"%q must be one of 'FailedEventsOnly', 'AllEvents'", k))
								}
								return
							},
						},

						"retry_duration": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      3600,
							ValidateFunc: validation.IntBetween(0, 7200),
						},

						"cloudwatch_logging_options": cloudWatchLoggingOptionsSchema(),

						"processing_configuration": processingConfigurationSchema(),
					},
				},
			},

			"arn": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"version_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"destination_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func createSourceConfig(source map[string]interface{}) *firehose.KinesisStreamSourceConfiguration {

	configuration := &firehose.KinesisStreamSourceConfiguration{
		KinesisStreamARN: aws.String(source["kinesis_stream_arn"].(string)),
		RoleARN:          aws.String(source["role_arn"].(string)),
	}

	return configuration
}

func createS3Config(d *schema.ResourceData) *firehose.S3DestinationConfiguration {
	s3 := d.Get("s3_configuration").([]interface{})[0].(map[string]interface{})

	configuration := &firehose.S3DestinationConfiguration{
		BucketARN: aws.String(s3["bucket_arn"].(string)),
		RoleARN:   aws.String(s3["role_arn"].(string)),
		BufferingHints: &firehose.BufferingHints{
			IntervalInSeconds: aws.Int64(int64(s3["buffer_interval"].(int))),
			SizeInMBs:         aws.Int64(int64(s3["buffer_size"].(int))),
		},
		Prefix:                  extractPrefixConfiguration(s3),
		CompressionFormat:       aws.String(s3["compression_format"].(string)),
		EncryptionConfiguration: extractEncryptionConfiguration(s3),
	}

	if _, ok := s3["cloudwatch_logging_options"]; ok {
		configuration.CloudWatchLoggingOptions = extractCloudWatchLoggingConfiguration(s3)
	}

	return configuration
}

func expandS3BackupConfig(d map[string]interface{}) *firehose.S3DestinationConfiguration {
	config := d["s3_backup_configuration"].([]interface{})
	if len(config) == 0 {
		return nil
	}

	s3 := config[0].(map[string]interface{})

	configuration := &firehose.S3DestinationConfiguration{
		BucketARN: aws.String(s3["bucket_arn"].(string)),
		RoleARN:   aws.String(s3["role_arn"].(string)),
		BufferingHints: &firehose.BufferingHints{
			IntervalInSeconds: aws.Int64(int64(s3["buffer_interval"].(int))),
			SizeInMBs:         aws.Int64(int64(s3["buffer_size"].(int))),
		},
		Prefix:                  extractPrefixConfiguration(s3),
		CompressionFormat:       aws.String(s3["compression_format"].(string)),
		EncryptionConfiguration: extractEncryptionConfiguration(s3),
	}

	if _, ok := s3["cloudwatch_logging_options"]; ok {
		configuration.CloudWatchLoggingOptions = extractCloudWatchLoggingConfiguration(s3)
	}

	return configuration
}

func createExtendedS3Config(d *schema.ResourceData) *firehose.ExtendedS3DestinationConfiguration {
	s3 := d.Get("extended_s3_configuration").([]interface{})[0].(map[string]interface{})

	configuration := &firehose.ExtendedS3DestinationConfiguration{
		BucketARN: aws.String(s3["bucket_arn"].(string)),
		RoleARN:   aws.String(s3["role_arn"].(string)),
		BufferingHints: &firehose.BufferingHints{
			IntervalInSeconds: aws.Int64(int64(s3["buffer_interval"].(int))),
			SizeInMBs:         aws.Int64(int64(s3["buffer_size"].(int))),
		},
		Prefix:                            extractPrefixConfiguration(s3),
		CompressionFormat:                 aws.String(s3["compression_format"].(string)),
		DataFormatConversionConfiguration: expandFirehoseDataFormatConversionConfiguration(s3["data_format_conversion_configuration"].([]interface{})),
		EncryptionConfiguration:           extractEncryptionConfiguration(s3),
	}

	if _, ok := s3["processing_configuration"]; ok {
		configuration.ProcessingConfiguration = extractProcessingConfiguration(s3)
	}

	if _, ok := s3["cloudwatch_logging_options"]; ok {
		configuration.CloudWatchLoggingOptions = extractCloudWatchLoggingConfiguration(s3)
	}

	if v, ok := s3["error_output_prefix"]; ok && v.(string) != "" {
		configuration.ErrorOutputPrefix = aws.String(v.(string))
	}

	if s3BackupMode, ok := s3["s3_backup_mode"]; ok {
		configuration.S3BackupMode = aws.String(s3BackupMode.(string))
		configuration.S3BackupConfiguration = expandS3BackupConfig(d.Get("extended_s3_configuration").([]interface{})[0].(map[string]interface{}))
	}

	return configuration
}

func updateS3Config(d *schema.ResourceData) *firehose.S3DestinationUpdate {
	s3 := d.Get("s3_configuration").([]interface{})[0].(map[string]interface{})

	configuration := &firehose.S3DestinationUpdate{
		BucketARN: aws.String(s3["bucket_arn"].(string)),
		RoleARN:   aws.String(s3["role_arn"].(string)),
		BufferingHints: &firehose.BufferingHints{
			IntervalInSeconds: aws.Int64((int64)(s3["buffer_interval"].(int))),
			SizeInMBs:         aws.Int64((int64)(s3["buffer_size"].(int))),
		},
		Prefix:                   extractPrefixConfiguration(s3),
		CompressionFormat:        aws.String(s3["compression_format"].(string)),
		EncryptionConfiguration:  extractEncryptionConfiguration(s3),
		CloudWatchLoggingOptions: extractCloudWatchLoggingConfiguration(s3),
	}

	if _, ok := s3["cloudwatch_logging_options"]; ok {
		configuration.CloudWatchLoggingOptions = extractCloudWatchLoggingConfiguration(s3)
	}

	return configuration
}

func updateS3BackupConfig(d map[string]interface{}) *firehose.S3DestinationUpdate {
	config := d["s3_backup_configuration"].([]interface{})
	if len(config) == 0 {
		return nil
	}

	s3 := config[0].(map[string]interface{})

	configuration := &firehose.S3DestinationUpdate{
		BucketARN: aws.String(s3["bucket_arn"].(string)),
		RoleARN:   aws.String(s3["role_arn"].(string)),
		BufferingHints: &firehose.BufferingHints{
			IntervalInSeconds: aws.Int64((int64)(s3["buffer_interval"].(int))),
			SizeInMBs:         aws.Int64((int64)(s3["buffer_size"].(int))),
		},
		Prefix:                   extractPrefixConfiguration(s3),
		CompressionFormat:        aws.String(s3["compression_format"].(string)),
		EncryptionConfiguration:  extractEncryptionConfiguration(s3),
		CloudWatchLoggingOptions: extractCloudWatchLoggingConfiguration(s3),
	}

	if _, ok := s3["cloudwatch_logging_options"]; ok {
		configuration.CloudWatchLoggingOptions = extractCloudWatchLoggingConfiguration(s3)
	}

	return configuration
}

func updateExtendedS3Config(d *schema.ResourceData) *firehose.ExtendedS3DestinationUpdate {
	s3 := d.Get("extended_s3_configuration").([]interface{})[0].(map[string]interface{})

	configuration := &firehose.ExtendedS3DestinationUpdate{
		BucketARN: aws.String(s3["bucket_arn"].(string)),
		RoleARN:   aws.String(s3["role_arn"].(string)),
		BufferingHints: &firehose.BufferingHints{
			IntervalInSeconds: aws.Int64((int64)(s3["buffer_interval"].(int))),
			SizeInMBs:         aws.Int64((int64)(s3["buffer_size"].(int))),
		},
		Prefix:                            extractPrefixConfiguration(s3),
		CompressionFormat:                 aws.String(s3["compression_format"].(string)),
		EncryptionConfiguration:           extractEncryptionConfiguration(s3),
		DataFormatConversionConfiguration: expandFirehoseDataFormatConversionConfiguration(s3["data_format_conversion_configuration"].([]interface{})),
		CloudWatchLoggingOptions:          extractCloudWatchLoggingConfiguration(s3),
		ProcessingConfiguration:           extractProcessingConfiguration(s3),
	}

	if _, ok := s3["cloudwatch_logging_options"]; ok {
		configuration.CloudWatchLoggingOptions = extractCloudWatchLoggingConfiguration(s3)
	}

	if v, ok := s3["error_output_prefix"]; ok && v.(string) != "" {
		configuration.ErrorOutputPrefix = aws.String(v.(string))
	}

	if s3BackupMode, ok := s3["s3_backup_mode"]; ok {
		configuration.S3BackupMode = aws.String(s3BackupMode.(string))
		configuration.S3BackupUpdate = updateS3BackupConfig(d.Get("extended_s3_configuration").([]interface{})[0].(map[string]interface{}))
	}

	return configuration
}

func expandFirehoseDataFormatConversionConfiguration(l []interface{}) *firehose.DataFormatConversionConfiguration {
	if len(l) == 0 || l[0] == nil {
		// It is possible to just pass nil here, but this seems to be the
		// canonical form that AWS uses, and is less likely to produce diffs.
		return &firehose.DataFormatConversionConfiguration{
			Enabled: aws.Bool(false),
		}
	}

	m := l[0].(map[string]interface{})

	return &firehose.DataFormatConversionConfiguration{
		Enabled:                   aws.Bool(m["enabled"].(bool)),
		InputFormatConfiguration:  expandFirehoseInputFormatConfiguration(m["input_format_configuration"].([]interface{})),
		OutputFormatConfiguration: expandFirehoseOutputFormatConfiguration(m["output_format_configuration"].([]interface{})),
		SchemaConfiguration:       expandFirehoseSchemaConfiguration(m["schema_configuration"].([]interface{})),
	}
}

func expandFirehoseInputFormatConfiguration(l []interface{}) *firehose.InputFormatConfiguration {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	m := l[0].(map[string]interface{})

	return &firehose.InputFormatConfiguration{
		Deserializer: expandFirehoseDeserializer(m["deserializer"].([]interface{})),
	}
}

func expandFirehoseDeserializer(l []interface{}) *firehose.Deserializer {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	m := l[0].(map[string]interface{})

	return &firehose.Deserializer{
		HiveJsonSerDe:  expandFirehoseHiveJsonSerDe(m["hive_json_ser_de"].([]interface{})),
		OpenXJsonSerDe: expandFirehoseOpenXJsonSerDe(m["open_x_json_ser_de"].([]interface{})),
	}
}

func expandFirehoseHiveJsonSerDe(l []interface{}) *firehose.HiveJsonSerDe {
	if len(l) == 0 {
		return nil
	}

	if l[0] == nil {
		return &firehose.HiveJsonSerDe{}
	}

	m := l[0].(map[string]interface{})

	return &firehose.HiveJsonSerDe{
		TimestampFormats: expandStringList(m["timestamp_formats"].([]interface{})),
	}
}

func expandFirehoseOpenXJsonSerDe(l []interface{}) *firehose.OpenXJsonSerDe {
	if len(l) == 0 {
		return nil
	}

	if l[0] == nil {
		return &firehose.OpenXJsonSerDe{}
	}

	m := l[0].(map[string]interface{})

	return &firehose.OpenXJsonSerDe{
		CaseInsensitive:                    aws.Bool(m["case_insensitive"].(bool)),
		ColumnToJsonKeyMappings:            stringMapToPointers(m["column_to_json_key_mappings"].(map[string]interface{})),
		ConvertDotsInJsonKeysToUnderscores: aws.Bool(m["convert_dots_in_json_keys_to_underscores"].(bool)),
	}
}

func expandFirehoseOutputFormatConfiguration(l []interface{}) *firehose.OutputFormatConfiguration {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	m := l[0].(map[string]interface{})

	return &firehose.OutputFormatConfiguration{
		Serializer: expandFirehoseSerializer(m["serializer"].([]interface{})),
	}
}

func expandFirehoseSerializer(l []interface{}) *firehose.Serializer {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	m := l[0].(map[string]interface{})

	return &firehose.Serializer{
		OrcSerDe:     expandFirehoseOrcSerDe(m["orc_ser_de"].([]interface{})),
		ParquetSerDe: expandFirehoseParquetSerDe(m["parquet_ser_de"].([]interface{})),
	}
}

func expandFirehoseOrcSerDe(l []interface{}) *firehose.OrcSerDe {
	if len(l) == 0 {
		return nil
	}

	if l[0] == nil {
		return &firehose.OrcSerDe{}
	}

	m := l[0].(map[string]interface{})

	orcSerDe := &firehose.OrcSerDe{
		BlockSizeBytes:                      aws.Int64(int64(m["block_size_bytes"].(int))),
		BloomFilterFalsePositiveProbability: aws.Float64(m["bloom_filter_false_positive_probability"].(float64)),
		Compression:                         aws.String(m["compression"].(string)),
		DictionaryKeyThreshold:              aws.Float64(m["dictionary_key_threshold"].(float64)),
		EnablePadding:                       aws.Bool(m["enable_padding"].(bool)),
		FormatVersion:                       aws.String(m["format_version"].(string)),
		PaddingTolerance:                    aws.Float64(m["padding_tolerance"].(float64)),
		RowIndexStride:                      aws.Int64(int64(m["row_index_stride"].(int))),
		StripeSizeBytes:                     aws.Int64(int64(m["stripe_size_bytes"].(int))),
	}

	if v, ok := m["bloom_filter_columns"].([]interface{}); ok && len(v) > 0 {
		orcSerDe.BloomFilterColumns = expandStringList(v)
	}

	return orcSerDe
}

func expandFirehoseParquetSerDe(l []interface{}) *firehose.ParquetSerDe {
	if len(l) == 0 {
		return nil
	}

	if l[0] == nil {
		return &firehose.ParquetSerDe{}
	}

	m := l[0].(map[string]interface{})

	return &firehose.ParquetSerDe{
		BlockSizeBytes:              aws.Int64(int64(m["block_size_bytes"].(int))),
		Compression:                 aws.String(m["compression"].(string)),
		EnableDictionaryCompression: aws.Bool(m["enable_dictionary_compression"].(bool)),
		MaxPaddingBytes:             aws.Int64(int64(m["max_padding_bytes"].(int))),
		PageSizeBytes:               aws.Int64(int64(m["page_size_bytes"].(int))),
		WriterVersion:               aws.String(m["writer_version"].(string)),
	}
}

func expandFirehoseSchemaConfiguration(l []interface{}) *firehose.SchemaConfiguration {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	m := l[0].(map[string]interface{})

	config := &firehose.SchemaConfiguration{
		DatabaseName: aws.String(m["database_name"].(string)),
		RoleARN:      aws.String(m["role_arn"].(string)),
		TableName:    aws.String(m["table_name"].(string)),
		VersionId:    aws.String(m["version_id"].(string)),
	}

	if v, ok := m["catalog_id"].(string); ok && v != "" {
		config.CatalogId = aws.String(v)
	}
	if v, ok := m["region"].(string); ok && v != "" {
		config.Region = aws.String(v)
	}

	return config
}

func extractProcessingConfiguration(s3 map[string]interface{}) *firehose.ProcessingConfiguration {
	config := s3["processing_configuration"].([]interface{})
	if len(config) == 0 {
		// It is possible to just pass nil here, but this seems to be the
		// canonical form that AWS uses, and is less likely to produce diffs.
		return &firehose.ProcessingConfiguration{
			Enabled:    aws.Bool(false),
			Processors: []*firehose.Processor{},
		}
	}

	processingConfiguration := config[0].(map[string]interface{})

	return &firehose.ProcessingConfiguration{
		Enabled:    aws.Bool(processingConfiguration["enabled"].(bool)),
		Processors: extractProcessors(processingConfiguration["processors"].([]interface{})),
	}
}

func extractProcessors(processingConfigurationProcessors []interface{}) []*firehose.Processor {
	processors := []*firehose.Processor{}

	for _, processor := range processingConfigurationProcessors {
		processors = append(processors, extractProcessor(processor.(map[string]interface{})))
	}

	return processors
}

func extractProcessor(processingConfigurationProcessor map[string]interface{}) *firehose.Processor {
	return &firehose.Processor{
		Type:       aws.String(processingConfigurationProcessor["type"].(string)),
		Parameters: extractProcessorParameters(processingConfigurationProcessor["parameters"].([]interface{})),
	}
}

func extractProcessorParameters(processorParameters []interface{}) []*firehose.ProcessorParameter {
	parameters := []*firehose.ProcessorParameter{}

	for _, attr := range processorParameters {
		parameters = append(parameters, extractProcessorParameter(attr.(map[string]interface{})))
	}

	return parameters
}

func extractProcessorParameter(processorParameter map[string]interface{}) *firehose.ProcessorParameter {
	parameter := &firehose.ProcessorParameter{
		ParameterName:  aws.String(processorParameter["parameter_name"].(string)),
		ParameterValue: aws.String(processorParameter["parameter_value"].(string)),
	}

	return parameter
}

func extractEncryptionConfiguration(s3 map[string]interface{}) *firehose.EncryptionConfiguration {
	if key, ok := s3["kms_key_arn"]; ok && len(key.(string)) > 0 {
		return &firehose.EncryptionConfiguration{
			KMSEncryptionConfig: &firehose.KMSEncryptionConfig{
				AWSKMSKeyARN: aws.String(key.(string)),
			},
		}
	}

	return &firehose.EncryptionConfiguration{
		NoEncryptionConfig: aws.String("NoEncryption"),
	}
}

func extractCloudWatchLoggingConfiguration(s3 map[string]interface{}) *firehose.CloudWatchLoggingOptions {
	config := s3["cloudwatch_logging_options"].(*schema.Set).List()
	if len(config) == 0 {
		return nil
	}

	loggingConfig := config[0].(map[string]interface{})
	loggingOptions := &firehose.CloudWatchLoggingOptions{
		Enabled: aws.Bool(loggingConfig["enabled"].(bool)),
	}

	if v, ok := loggingConfig["log_group_name"]; ok {
		loggingOptions.LogGroupName = aws.String(v.(string))
	}

	if v, ok := loggingConfig["log_stream_name"]; ok {
		loggingOptions.LogStreamName = aws.String(v.(string))
	}

	return loggingOptions

}

func extractPrefixConfiguration(s3 map[string]interface{}) *string {
	if v, ok := s3["prefix"]; ok {
		return aws.String(v.(string))
	}

	return nil
}

func createRedshiftConfig(d *schema.ResourceData, s3Config *firehose.S3DestinationConfiguration) (*firehose.RedshiftDestinationConfiguration, error) {
	redshiftRaw, ok := d.GetOk("redshift_configuration")
	if !ok {
		return nil, fmt.Errorf("Error loading Redshift Configuration for Kinesis Firehose: redshift_configuration not found")
	}
	rl := redshiftRaw.([]interface{})

	redshift := rl[0].(map[string]interface{})

	configuration := &firehose.RedshiftDestinationConfiguration{
		ClusterJDBCURL:  aws.String(redshift["cluster_jdbcurl"].(string)),
		RetryOptions:    extractRedshiftRetryOptions(redshift),
		Password:        aws.String(redshift["password"].(string)),
		Username:        aws.String(redshift["username"].(string)),
		RoleARN:         aws.String(redshift["role_arn"].(string)),
		CopyCommand:     extractCopyCommandConfiguration(redshift),
		S3Configuration: s3Config,
	}

	if _, ok := redshift["cloudwatch_logging_options"]; ok {
		configuration.CloudWatchLoggingOptions = extractCloudWatchLoggingConfiguration(redshift)
	}
	if _, ok := redshift["processing_configuration"]; ok {
		configuration.ProcessingConfiguration = extractProcessingConfiguration(redshift)
	}
	if s3BackupMode, ok := redshift["s3_backup_mode"]; ok {
		configuration.S3BackupMode = aws.String(s3BackupMode.(string))
		configuration.S3BackupConfiguration = expandS3BackupConfig(d.Get("redshift_configuration").([]interface{})[0].(map[string]interface{}))
	}

	return configuration, nil
}

func updateRedshiftConfig(d *schema.ResourceData, s3Update *firehose.S3DestinationUpdate) (*firehose.RedshiftDestinationUpdate, error) {
	redshiftRaw, ok := d.GetOk("redshift_configuration")
	if !ok {
		return nil, fmt.Errorf("Error loading Redshift Configuration for Kinesis Firehose: redshift_configuration not found")
	}
	rl := redshiftRaw.([]interface{})

	redshift := rl[0].(map[string]interface{})

	configuration := &firehose.RedshiftDestinationUpdate{
		ClusterJDBCURL: aws.String(redshift["cluster_jdbcurl"].(string)),
		RetryOptions:   extractRedshiftRetryOptions(redshift),
		Password:       aws.String(redshift["password"].(string)),
		Username:       aws.String(redshift["username"].(string)),
		RoleARN:        aws.String(redshift["role_arn"].(string)),
		CopyCommand:    extractCopyCommandConfiguration(redshift),
		S3Update:       s3Update,
	}

	if _, ok := redshift["cloudwatch_logging_options"]; ok {
		configuration.CloudWatchLoggingOptions = extractCloudWatchLoggingConfiguration(redshift)
	}
	if _, ok := redshift["processing_configuration"]; ok {
		configuration.ProcessingConfiguration = extractProcessingConfiguration(redshift)
	}
	if s3BackupMode, ok := redshift["s3_backup_mode"]; ok {
		configuration.S3BackupMode = aws.String(s3BackupMode.(string))
		configuration.S3BackupUpdate = updateS3BackupConfig(d.Get("redshift_configuration").([]interface{})[0].(map[string]interface{}))
	}

	return configuration, nil
}

func createElasticsearchConfig(d *schema.ResourceData, s3Config *firehose.S3DestinationConfiguration) (*firehose.ElasticsearchDestinationConfiguration, error) {
	esConfig, ok := d.GetOk("elasticsearch_configuration")
	if !ok {
		return nil, fmt.Errorf("Error loading Elasticsearch Configuration for Kinesis Firehose: elasticsearch_configuration not found")
	}
	esList := esConfig.([]interface{})

	es := esList[0].(map[string]interface{})

	config := &firehose.ElasticsearchDestinationConfiguration{
		BufferingHints:  extractBufferingHints(es),
		DomainARN:       aws.String(es["domain_arn"].(string)),
		IndexName:       aws.String(es["index_name"].(string)),
		RetryOptions:    extractElasticSearchRetryOptions(es),
		RoleARN:         aws.String(es["role_arn"].(string)),
		TypeName:        aws.String(es["type_name"].(string)),
		S3Configuration: s3Config,
	}

	if _, ok := es["cloudwatch_logging_options"]; ok {
		config.CloudWatchLoggingOptions = extractCloudWatchLoggingConfiguration(es)
	}

	if _, ok := es["processing_configuration"]; ok {
		config.ProcessingConfiguration = extractProcessingConfiguration(es)
	}

	if indexRotationPeriod, ok := es["index_rotation_period"]; ok {
		config.IndexRotationPeriod = aws.String(indexRotationPeriod.(string))
	}
	if s3BackupMode, ok := es["s3_backup_mode"]; ok {
		config.S3BackupMode = aws.String(s3BackupMode.(string))
	}

	return config, nil
}

func updateElasticsearchConfig(d *schema.ResourceData, s3Update *firehose.S3DestinationUpdate) (*firehose.ElasticsearchDestinationUpdate, error) {
	esConfig, ok := d.GetOk("elasticsearch_configuration")
	if !ok {
		return nil, fmt.Errorf("Error loading Elasticsearch Configuration for Kinesis Firehose: elasticsearch_configuration not found")
	}
	esList := esConfig.([]interface{})

	es := esList[0].(map[string]interface{})

	update := &firehose.ElasticsearchDestinationUpdate{
		BufferingHints: extractBufferingHints(es),
		DomainARN:      aws.String(es["domain_arn"].(string)),
		IndexName:      aws.String(es["index_name"].(string)),
		RetryOptions:   extractElasticSearchRetryOptions(es),
		RoleARN:        aws.String(es["role_arn"].(string)),
		TypeName:       aws.String(es["type_name"].(string)),
		S3Update:       s3Update,
	}

	if _, ok := es["cloudwatch_logging_options"]; ok {
		update.CloudWatchLoggingOptions = extractCloudWatchLoggingConfiguration(es)
	}

	if _, ok := es["processing_configuration"]; ok {
		update.ProcessingConfiguration = extractProcessingConfiguration(es)
	}

	if indexRotationPeriod, ok := es["index_rotation_period"]; ok {
		update.IndexRotationPeriod = aws.String(indexRotationPeriod.(string))
	}

	return update, nil
}

func createSplunkConfig(d *schema.ResourceData, s3Config *firehose.S3DestinationConfiguration) (*firehose.SplunkDestinationConfiguration, error) {
	splunkRaw, ok := d.GetOk("splunk_configuration")
	if !ok {
		return nil, fmt.Errorf("Error loading Splunk Configuration for Kinesis Firehose: splunk_configuration not found")
	}
	sl := splunkRaw.([]interface{})

	splunk := sl[0].(map[string]interface{})

	configuration := &firehose.SplunkDestinationConfiguration{
		HECToken:                          aws.String(splunk["hec_token"].(string)),
		HECEndpointType:                   aws.String(splunk["hec_endpoint_type"].(string)),
		HECEndpoint:                       aws.String(splunk["hec_endpoint"].(string)),
		HECAcknowledgmentTimeoutInSeconds: aws.Int64(int64(splunk["hec_acknowledgment_timeout"].(int))),
		RetryOptions:                      extractSplunkRetryOptions(splunk),
		S3Configuration:                   s3Config,
	}

	if _, ok := splunk["processing_configuration"]; ok {
		configuration.ProcessingConfiguration = extractProcessingConfiguration(splunk)
	}

	if _, ok := splunk["cloudwatch_logging_options"]; ok {
		configuration.CloudWatchLoggingOptions = extractCloudWatchLoggingConfiguration(splunk)
	}
	if s3BackupMode, ok := splunk["s3_backup_mode"]; ok {
		configuration.S3BackupMode = aws.String(s3BackupMode.(string))
	}

	return configuration, nil
}

func updateSplunkConfig(d *schema.ResourceData, s3Update *firehose.S3DestinationUpdate) (*firehose.SplunkDestinationUpdate, error) {
	splunkRaw, ok := d.GetOk("splunk_configuration")
	if !ok {
		return nil, fmt.Errorf("Error loading Splunk Configuration for Kinesis Firehose: splunk_configuration not found")
	}
	sl := splunkRaw.([]interface{})

	splunk := sl[0].(map[string]interface{})

	configuration := &firehose.SplunkDestinationUpdate{
		HECToken:                          aws.String(splunk["hec_token"].(string)),
		HECEndpointType:                   aws.String(splunk["hec_endpoint_type"].(string)),
		HECEndpoint:                       aws.String(splunk["hec_endpoint"].(string)),
		HECAcknowledgmentTimeoutInSeconds: aws.Int64(int64(splunk["hec_acknowledgment_timeout"].(int))),
		RetryOptions:                      extractSplunkRetryOptions(splunk),
		S3Update:                          s3Update,
	}

	if _, ok := splunk["processing_configuration"]; ok {
		configuration.ProcessingConfiguration = extractProcessingConfiguration(splunk)
	}

	if _, ok := splunk["cloudwatch_logging_options"]; ok {
		configuration.CloudWatchLoggingOptions = extractCloudWatchLoggingConfiguration(splunk)
	}
	if s3BackupMode, ok := splunk["s3_backup_mode"]; ok {
		configuration.S3BackupMode = aws.String(s3BackupMode.(string))
	}

	return configuration, nil
}

func extractBufferingHints(es map[string]interface{}) *firehose.ElasticsearchBufferingHints {
	bufferingHints := &firehose.ElasticsearchBufferingHints{}

	if bufferingInterval, ok := es["buffering_interval"].(int); ok {
		bufferingHints.IntervalInSeconds = aws.Int64(int64(bufferingInterval))
	}
	if bufferingSize, ok := es["buffering_size"].(int); ok {
		bufferingHints.SizeInMBs = aws.Int64(int64(bufferingSize))
	}

	return bufferingHints
}

func extractElasticSearchRetryOptions(es map[string]interface{}) *firehose.ElasticsearchRetryOptions {
	retryOptions := &firehose.ElasticsearchRetryOptions{}

	if retryDuration, ok := es["retry_duration"].(int); ok {
		retryOptions.DurationInSeconds = aws.Int64(int64(retryDuration))
	}

	return retryOptions
}

func extractRedshiftRetryOptions(redshift map[string]interface{}) *firehose.RedshiftRetryOptions {
	retryOptions := &firehose.RedshiftRetryOptions{}

	if retryDuration, ok := redshift["retry_duration"].(int); ok {
		retryOptions.DurationInSeconds = aws.Int64(int64(retryDuration))
	}

	return retryOptions
}

func extractSplunkRetryOptions(splunk map[string]interface{}) *firehose.SplunkRetryOptions {
	retryOptions := &firehose.SplunkRetryOptions{}

	if retryDuration, ok := splunk["retry_duration"].(int); ok {
		retryOptions.DurationInSeconds = aws.Int64(int64(retryDuration))
	}

	return retryOptions
}

func extractCopyCommandConfiguration(redshift map[string]interface{}) *firehose.CopyCommand {
	cmd := &firehose.CopyCommand{
		DataTableName: aws.String(redshift["data_table_name"].(string)),
	}
	if copyOptions, ok := redshift["copy_options"]; ok {
		cmd.CopyOptions = aws.String(copyOptions.(string))
	}
	if columns, ok := redshift["data_table_columns"]; ok {
		cmd.DataTableColumns = aws.String(columns.(string))
	}

	return cmd
}

func resourceAwsKinesisFirehoseDeliveryStreamCreate(d *schema.ResourceData, meta interface{}) error {
	validateError := validateAwsKinesisFirehoseSchema(d)

	if validateError != nil {
		return validateError
	}

	conn := meta.(*AWSClient).firehoseconn

	sn := d.Get("name").(string)

	createInput := &firehose.CreateDeliveryStreamInput{
		DeliveryStreamName: aws.String(sn),
	}

	if v, ok := d.GetOk("kinesis_source_configuration"); ok {
		sourceConfig := createSourceConfig(v.([]interface{})[0].(map[string]interface{}))
		createInput.KinesisStreamSourceConfiguration = sourceConfig
		createInput.DeliveryStreamType = aws.String(firehose.DeliveryStreamTypeKinesisStreamAsSource)
	} else {
		createInput.DeliveryStreamType = aws.String(firehose.DeliveryStreamTypeDirectPut)
	}

	if d.Get("destination").(string) == "extended_s3" {
		extendedS3Config := createExtendedS3Config(d)
		createInput.ExtendedS3DestinationConfiguration = extendedS3Config
	} else {
		s3Config := createS3Config(d)

		if d.Get("destination").(string) == "s3" {
			createInput.S3DestinationConfiguration = s3Config
		} else if d.Get("destination").(string) == "elasticsearch" {
			esConfig, err := createElasticsearchConfig(d, s3Config)
			if err != nil {
				return err
			}
			createInput.ElasticsearchDestinationConfiguration = esConfig
		} else if d.Get("destination").(string) == "redshift" {
			rc, err := createRedshiftConfig(d, s3Config)
			if err != nil {
				return err
			}
			createInput.RedshiftDestinationConfiguration = rc
		} else if d.Get("destination").(string) == "splunk" {
			rc, err := createSplunkConfig(d, s3Config)
			if err != nil {
				return err
			}
			createInput.SplunkDestinationConfiguration = rc
		}
	}

	if v, ok := d.GetOk("tags"); ok {
		createInput.Tags = tagsFromMapKinesisFirehose(v.(map[string]interface{}))
	}

	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		_, err := conn.CreateDeliveryStream(createInput)
		if err != nil {
			log.Printf("[DEBUG] Error creating Firehose Delivery Stream: %s", err)

			// Retry for IAM eventual consistency
			if isAWSErr(err, firehose.ErrCodeInvalidArgumentException, "is not authorized to") {
				return resource.RetryableError(err)
			}
			// InvalidArgumentException: Verify that the IAM role has access to the ElasticSearch domain.
			if isAWSErr(err, firehose.ErrCodeInvalidArgumentException, "Verify that the IAM role has access") {
				return resource.RetryableError(err)
			}
			// IAM roles can take ~10 seconds to propagate in AWS:
			// http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/iam-roles-for-amazon-ec2.html#launch-instance-with-role-console
			if isAWSErr(err, firehose.ErrCodeInvalidArgumentException, "Firehose is unable to assume role") {
				log.Printf("[DEBUG] Firehose could not assume role referenced, retrying...")
				return resource.RetryableError(err)
			}
			// Not retryable
			return resource.NonRetryableError(err)
		}

		return nil
	})
	if isResourceTimeoutError(err) {
		_, err = conn.CreateDeliveryStream(createInput)
	}
	if err != nil {
		return fmt.Errorf("error creating Kinesis Firehose Delivery Stream: %s", err)
	}

	s, err := waitForKinesisFirehoseDeliveryStreamCreation(conn, sn)
	if err != nil {
		return fmt.Errorf("error waiting for Kinesis Firehose Delivery Stream (%s) creation: %s", sn, err)
	}

	d.SetId(aws.StringValue(s.DeliveryStreamARN))
	d.Set("arn", s.DeliveryStreamARN)

	if v, ok := d.GetOk("server_side_encryption"); ok && !isKinesisFirehoseDeliveryStreamOptionDisabled(v) {
		_, err := conn.StartDeliveryStreamEncryption(&firehose.StartDeliveryStreamEncryptionInput{
			DeliveryStreamName: aws.String(sn),
		})
		if err != nil {
			return fmt.Errorf("error starting Kinesis Firehose Delivery Stream (%s) encryption: %s", sn, err)
		}

		if err := waitForKinesisFirehoseDeliveryStreamSSEEnabled(conn, sn); err != nil {
			return fmt.Errorf("error waiting for Kinesis Firehose Delivery Stream (%s) encryption to be enabled: %s", sn, err)
		}
	}

	return resourceAwsKinesisFirehoseDeliveryStreamRead(d, meta)
}

func validateAwsKinesisFirehoseSchema(d *schema.ResourceData) error {

	_, s3Exists := d.GetOk("s3_configuration")
	_, extendedS3Exists := d.GetOk("extended_s3_configuration")

	if d.Get("destination").(string) == "extended_s3" {
		if !extendedS3Exists {
			return fmt.Errorf(
				"When destination is 'extended_s3', extended_s3_configuration is required",
			)
		} else if s3Exists {
			return fmt.Errorf(
				"When destination is 'extended_s3', s3_configuration must not be set",
			)
		}
	} else {
		if !s3Exists {
			return fmt.Errorf(
				"When destination is %s, s3_configuration is required",
				d.Get("destination").(string),
			)
		} else if extendedS3Exists {
			return fmt.Errorf(
				"extended_s3_configuration can only be used when destination is 'extended_s3'",
			)
		}
	}

	return nil
}

func resourceAwsKinesisFirehoseDeliveryStreamUpdate(d *schema.ResourceData, meta interface{}) error {
	validateError := validateAwsKinesisFirehoseSchema(d)

	if validateError != nil {
		return validateError
	}

	conn := meta.(*AWSClient).firehoseconn

	sn := d.Get("name").(string)
	updateInput := &firehose.UpdateDestinationInput{
		DeliveryStreamName:             aws.String(sn),
		CurrentDeliveryStreamVersionId: aws.String(d.Get("version_id").(string)),
		DestinationId:                  aws.String(d.Get("destination_id").(string)),
	}

	if d.Get("destination").(string) == "extended_s3" {
		extendedS3Config := updateExtendedS3Config(d)
		updateInput.ExtendedS3DestinationUpdate = extendedS3Config
	} else {
		s3Config := updateS3Config(d)

		if d.Get("destination").(string) == "s3" {
			updateInput.S3DestinationUpdate = s3Config
		} else if d.Get("destination").(string) == "elasticsearch" {
			esUpdate, err := updateElasticsearchConfig(d, s3Config)
			if err != nil {
				return err
			}
			updateInput.ElasticsearchDestinationUpdate = esUpdate
		} else if d.Get("destination").(string) == "redshift" {
			rc, err := updateRedshiftConfig(d, s3Config)
			if err != nil {
				return err
			}
			updateInput.RedshiftDestinationUpdate = rc
		} else if d.Get("destination").(string) == "splunk" {
			rc, err := updateSplunkConfig(d, s3Config)
			if err != nil {
				return err
			}
			updateInput.SplunkDestinationUpdate = rc
		}
	}

	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		_, err := conn.UpdateDestination(updateInput)
		if err != nil {
			log.Printf("[DEBUG] Error updating Firehose Delivery Stream: %s", err)

			// Retry for IAM eventual consistency
			if isAWSErr(err, firehose.ErrCodeInvalidArgumentException, "is not authorized to") {
				return resource.RetryableError(err)
			}
			// InvalidArgumentException: Verify that the IAM role has access to the ElasticSearch domain.
			if isAWSErr(err, firehose.ErrCodeInvalidArgumentException, "Verify that the IAM role has access") {
				return resource.RetryableError(err)
			}
			// IAM roles can take ~10 seconds to propagate in AWS:
			// http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/iam-roles-for-amazon-ec2.html#launch-instance-with-role-console
			if isAWSErr(err, firehose.ErrCodeInvalidArgumentException, "Firehose is unable to assume role") {
				log.Printf("[DEBUG] Firehose could not assume role referenced, retrying...")
				return resource.RetryableError(err)
			}
			// Not retryable
			return resource.NonRetryableError(err)
		}

		return nil
	})

	if isResourceTimeoutError(err) {
		_, err = conn.UpdateDestination(updateInput)
	}

	if err != nil {
		return fmt.Errorf(
			"Error Updating Kinesis Firehose Delivery Stream: \"%s\"\n%s",
			sn, err)
	}

	if err := setTagsKinesisFirehose(conn, d, sn); err != nil {
		return fmt.Errorf(
			"Error Updating Kinesis Firehose Delivery Stream tags: \"%s\"\n%s",
			sn, err)
	}

	if d.HasChange("server_side_encryption") {
		_, n := d.GetChange("server_side_encryption")
		if isKinesisFirehoseDeliveryStreamOptionDisabled(n) {
			_, err := conn.StopDeliveryStreamEncryption(&firehose.StopDeliveryStreamEncryptionInput{
				DeliveryStreamName: aws.String(sn),
			})
			if err != nil {
				return fmt.Errorf("error stopping Kinesis Firehose Delivery Stream (%s) encryption: %s", sn, err)
			}

			if err := waitForKinesisFirehoseDeliveryStreamSSEDisabled(conn, sn); err != nil {
				return fmt.Errorf("error waiting for Kinesis Firehose Delivery Stream (%s) encryption to be disabled: %s", sn, err)
			}
		} else {
			_, err := conn.StartDeliveryStreamEncryption(&firehose.StartDeliveryStreamEncryptionInput{
				DeliveryStreamName: aws.String(sn),
			})
			if err != nil {
				return fmt.Errorf("error starting Kinesis Firehose Delivery Stream (%s) encryption: %s", sn, err)
			}

			if err := waitForKinesisFirehoseDeliveryStreamSSEEnabled(conn, sn); err != nil {
				return fmt.Errorf("error waiting for Kinesis Firehose Delivery Stream (%s) encryption to be enabled: %s", sn, err)
			}
		}
	}

	return resourceAwsKinesisFirehoseDeliveryStreamRead(d, meta)
}

func resourceAwsKinesisFirehoseDeliveryStreamRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).firehoseconn

	sn := d.Get("name").(string)
	resp, err := conn.DescribeDeliveryStream(&firehose.DescribeDeliveryStreamInput{
		DeliveryStreamName: aws.String(sn),
	})

	if err != nil {
		if isAWSErr(err, firehose.ErrCodeResourceNotFoundException, "") {
			log.Printf("[WARN] Kinesis Firehose Delivery Stream (%s) not found, removing from state", sn)
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error reading Kinesis Firehose Delivery Stream: %s", err)
	}

	s := resp.DeliveryStreamDescription
	err = flattenKinesisFirehoseDeliveryStream(d, s)
	if err != nil {
		return err
	}

	if err := getTagsKinesisFirehose(conn, d, sn); err != nil {
		return err
	}

	return nil
}

func resourceAwsKinesisFirehoseDeliveryStreamDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).firehoseconn

	sn := d.Get("name").(string)
	_, err := conn.DeleteDeliveryStream(&firehose.DeleteDeliveryStreamInput{
		DeliveryStreamName: aws.String(sn),
	})
	if err != nil {
		return fmt.Errorf("error deleting Kinesis Firehose Delivery Stream (%s): %s", sn, err)
	}

	if err := waitForKinesisFirehoseDeliveryStreamDeletion(conn, sn); err != nil {
		return fmt.Errorf("error waiting for Kinesis Firehose Delivery Stream (%s) deletion: %s", sn, err)
	}

	return nil
}

func firehoseDeliveryStreamStateRefreshFunc(conn *firehose.Firehose, sn string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeDeliveryStream(&firehose.DescribeDeliveryStreamInput{
			DeliveryStreamName: aws.String(sn),
		})
		if err != nil {
			if isAWSErr(err, firehose.ErrCodeResourceNotFoundException, "") {
				return &firehose.DeliveryStreamDescription{}, firehoseDeliveryStreamStatusDeleted, nil
			}
			return nil, "", err
		}

		return resp.DeliveryStreamDescription, aws.StringValue(resp.DeliveryStreamDescription.DeliveryStreamStatus), nil
	}
}

func firehoseDeliveryStreamSSEStateRefreshFunc(conn *firehose.Firehose, sn string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeDeliveryStream(&firehose.DescribeDeliveryStreamInput{
			DeliveryStreamName: aws.String(sn),
		})
		if err != nil {
			return nil, "", err
		}

		return resp.DeliveryStreamDescription, aws.StringValue(resp.DeliveryStreamDescription.DeliveryStreamEncryptionConfiguration.Status), nil
	}
}

func waitForKinesisFirehoseDeliveryStreamCreation(conn *firehose.Firehose, deliveryStreamName string) (*firehose.DeliveryStreamDescription, error) {
	stateConf := &resource.StateChangeConf{
		Pending:    []string{firehose.DeliveryStreamStatusCreating},
		Target:     []string{firehose.DeliveryStreamStatusActive},
		Refresh:    firehoseDeliveryStreamStateRefreshFunc(conn, deliveryStreamName),
		Timeout:    20 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	v, err := stateConf.WaitForState()
	if err != nil {
		return nil, err
	}

	return v.(*firehose.DeliveryStreamDescription), nil
}

func waitForKinesisFirehoseDeliveryStreamDeletion(conn *firehose.Firehose, deliveryStreamName string) error {
	stateConf := &resource.StateChangeConf{
		Pending:    []string{firehose.DeliveryStreamStatusDeleting},
		Target:     []string{firehoseDeliveryStreamStatusDeleted},
		Refresh:    firehoseDeliveryStreamStateRefreshFunc(conn, deliveryStreamName),
		Timeout:    20 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err := stateConf.WaitForState()

	return err
}

func waitForKinesisFirehoseDeliveryStreamSSEEnabled(conn *firehose.Firehose, deliveryStreamName string) error {
	stateConf := &resource.StateChangeConf{
		Pending:    []string{firehose.DeliveryStreamEncryptionStatusEnabling},
		Target:     []string{firehose.DeliveryStreamEncryptionStatusEnabled},
		Refresh:    firehoseDeliveryStreamSSEStateRefreshFunc(conn, deliveryStreamName),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err := stateConf.WaitForState()

	return err
}

func waitForKinesisFirehoseDeliveryStreamSSEDisabled(conn *firehose.Firehose, deliveryStreamName string) error {
	stateConf := &resource.StateChangeConf{
		Pending:    []string{firehose.DeliveryStreamEncryptionStatusDisabling},
		Target:     []string{firehose.DeliveryStreamEncryptionStatusDisabled},
		Refresh:    firehoseDeliveryStreamSSEStateRefreshFunc(conn, deliveryStreamName),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err := stateConf.WaitForState()

	return err
}

func isKinesisFirehoseDeliveryStreamOptionDisabled(v interface{}) bool {
	options := v.([]interface{})
	if len(options) == 0 || options[0] == nil {
		return true
	}
	m := options[0].(map[string]interface{})

	var enabled bool

	if v, ok := m["enabled"]; ok {
		enabled = v.(bool)
	}

	return !enabled
}
