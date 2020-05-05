package aws

import (
	"errors"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func dataSourceAwsEc2TransitGatewayDxGatewayAttachment() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsEc2TransitGatewayDxGatewayAttachmentRead,

		Schema: map[string]*schema.Schema{
			"dx_gateway_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"tags": tagsSchemaComputed(),
			"transit_gateway_id": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func dataSourceAwsEc2TransitGatewayDxGatewayAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	input := &ec2.DescribeTransitGatewayAttachmentsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("resource-id"),
				Values: []*string{aws.String(d.Get("dx_gateway_id").(string))},
			},
			{
				Name:   aws.String("resource-type"),
				Values: []*string{aws.String("direct-connect-gateway")}, // Not yet defined in ec2/api.go.
			},
			{
				Name:   aws.String("transit-gateway-id"),
				Values: []*string{aws.String(d.Get("transit_gateway_id").(string))},
			},
		},
	}

	log.Printf("[DEBUG] Reading EC2 Transit Gateway Direct Connect Gateway Attachments: %s", input)
	output, err := conn.DescribeTransitGatewayAttachments(input)

	if err != nil {
		return fmt.Errorf("error reading EC2 Transit Gateway Direct Connect Gateway Attachment: %s", err)
	}

	if output == nil || len(output.TransitGatewayAttachments) == 0 || output.TransitGatewayAttachments[0] == nil {
		return errors.New("error reading EC2 Transit Gateway Direct Connect Gateway Attachment: no results found")
	}

	if len(output.TransitGatewayAttachments) > 1 {
		return errors.New("error reading EC2 Transit Gateway Direct Connect Gateway Attachment: multiple results found, try adjusting search criteria")
	}

	transitGatewayAttachment := output.TransitGatewayAttachments[0]

	if err := d.Set("tags", tagsToMap(transitGatewayAttachment.Tags)); err != nil {
		return fmt.Errorf("error setting tags: %s", err)
	}

	d.Set("transit_gateway_id", aws.StringValue(transitGatewayAttachment.TransitGatewayId))
	d.Set("dx_gateway_id", aws.StringValue(transitGatewayAttachment.ResourceId))

	d.SetId(aws.StringValue(transitGatewayAttachment.TransitGatewayAttachmentId))

	return nil
}
