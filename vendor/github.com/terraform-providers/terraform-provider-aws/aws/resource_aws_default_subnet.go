package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceAwsDefaultSubnet() *schema.Resource {
	// reuse aws_subnet schema, and methods for READ, UPDATE
	dsubnet := resourceAwsSubnet()
	dsubnet.Create = resourceAwsDefaultSubnetCreate
	dsubnet.Delete = resourceAwsDefaultSubnetDelete

	// availability_zone is a required value for Default Subnets
	dsubnet.Schema["availability_zone"] = &schema.Schema{
		Type:     schema.TypeString,
		Required: true,
	}
	// availability_zone_id is a computed value for Default Subnets
	dsubnet.Schema["availability_zone_id"] = &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	}
	// vpc_id is a computed value for Default Subnets
	dsubnet.Schema["vpc_id"] = &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	}
	// cidr_block is a computed value for Default Subnets
	dsubnet.Schema["cidr_block"] = &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	}
	// ipv6_cidr_block is a computed value for Default Subnets
	dsubnet.Schema["ipv6_cidr_block"] = &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	}
	// map_public_ip_on_launch is a computed value for Default Subnets
	dsubnet.Schema["map_public_ip_on_launch"] = &schema.Schema{
		Type:     schema.TypeBool,
		Optional: true,
		Computed: true,
	}
	// assign_ipv6_address_on_creation is a computed value for Default Subnets
	dsubnet.Schema["assign_ipv6_address_on_creation"] = &schema.Schema{
		Type:     schema.TypeBool,
		Computed: true,
	}

	return dsubnet
}

func resourceAwsDefaultSubnetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	req := &ec2.DescribeSubnetsInput{}
	req.Filters = buildEC2AttributeFilterList(
		map[string]string{
			"availabilityZone": d.Get("availability_zone").(string),
			"defaultForAz":     "true",
		},
	)

	log.Printf("[DEBUG] Reading Default Subnet: %s", req)
	resp, err := conn.DescribeSubnets(req)
	if err != nil {
		return err
	}
	if len(resp.Subnets) != 1 || resp.Subnets[0] == nil {
		return fmt.Errorf("Default subnet not found")
	}

	d.SetId(aws.StringValue(resp.Subnets[0].SubnetId))
	return resourceAwsSubnetUpdate(d, meta)
}

func resourceAwsDefaultSubnetDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[WARN] Cannot destroy Default Subnet. Terraform will remove this resource from the state file, however resources may remain.")
	return nil
}
