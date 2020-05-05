package aws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceAwsCloudWatchLogDestination() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCloudWatchLogDestinationPut,
		Update: resourceAwsCloudWatchLogDestinationPut,
		Read:   resourceAwsCloudWatchLogDestinationRead,
		Delete: resourceAwsCloudWatchLogDestinationDelete,

		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				d.Set("name", d.Id())
				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"role_arn": {
				Type:     schema.TypeString,
				Required: true,
			},

			"target_arn": {
				Type:     schema.TypeString,
				Required: true,
			},

			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsCloudWatchLogDestinationPut(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn

	name := d.Get("name").(string)
	role_arn := d.Get("role_arn").(string)
	target_arn := d.Get("target_arn").(string)

	params := &cloudwatchlogs.PutDestinationInput{
		DestinationName: aws.String(name),
		RoleArn:         aws.String(role_arn),
		TargetArn:       aws.String(target_arn),
	}

	var resp *cloudwatchlogs.PutDestinationOutput
	var err error
	err = resource.Retry(3*time.Minute, func() *resource.RetryError {
		resp, err = conn.PutDestination(params)

		if isAWSErr(err, cloudwatchlogs.ErrCodeInvalidParameterException, "Could not deliver test message to specified") {
			return resource.RetryableError(err)
		}
		if err != nil {
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if isResourceTimeoutError(err) {
		resp, err = conn.PutDestination(params)
	}
	if err != nil {
		return fmt.Errorf("Error putting cloudwatch log destination: %s", err)
	}
	d.SetId(name)
	d.Set("arn", resp.Destination.Arn)
	return nil
}

func resourceAwsCloudWatchLogDestinationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn
	name := d.Get("name").(string)
	destination, exists, err := lookupCloudWatchLogDestination(conn, name, nil)
	if err != nil {
		return err
	}

	if !exists {
		d.SetId("")
		return nil
	}

	d.SetId(name)
	d.Set("arn", destination.Arn)
	d.Set("role_arn", destination.RoleArn)
	d.Set("target_arn", destination.TargetArn)

	return nil
}

func resourceAwsCloudWatchLogDestinationDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn

	name := d.Get("name").(string)

	params := &cloudwatchlogs.DeleteDestinationInput{
		DestinationName: aws.String(name),
	}
	_, err := conn.DeleteDestination(params)
	if err != nil {
		return fmt.Errorf("Error deleting Destination with name %s", name)
	}

	return nil
}

func lookupCloudWatchLogDestination(conn *cloudwatchlogs.CloudWatchLogs,
	name string, nextToken *string) (*cloudwatchlogs.Destination, bool, error) {
	input := &cloudwatchlogs.DescribeDestinationsInput{
		DestinationNamePrefix: aws.String(name),
		NextToken:             nextToken,
	}
	resp, err := conn.DescribeDestinations(input)
	if err != nil {
		return nil, true, err
	}

	for _, destination := range resp.Destinations {
		if *destination.DestinationName == name {
			return destination, true, nil
		}
	}

	if resp.NextToken != nil {
		return lookupCloudWatchLogDestination(conn, name, resp.NextToken)
	}

	return nil, false, nil
}
