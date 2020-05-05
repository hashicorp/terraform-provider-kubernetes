package aws

import (
	"fmt"
	"log"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/helper/hashcode"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func dataSourceAwsVpcEndpointService() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsVpcEndpointServiceRead,

		Schema: map[string]*schema.Schema{
			"acceptance_required": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"availability_zones": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
				Set:      schema.HashString,
			},
			"base_endpoint_dns_names": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
				Set:      schema.HashString,
			},
			"manages_vpc_endpoints": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"owner": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"private_dns_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"service": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"service_name"},
			},
			"service_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"service_name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ConflictsWith: []string{"service"},
			},
			"service_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags": tagsSchemaComputed(),
			"vpc_endpoint_policy_supported": {
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsVpcEndpointServiceRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	var serviceName string
	if v, ok := d.GetOk("service_name"); ok {
		serviceName = v.(string)
	} else if v, ok := d.GetOk("service"); ok {
		serviceName = fmt.Sprintf("com.amazonaws.%s.%s", meta.(*AWSClient).region, v.(string))
	} else {
		return fmt.Errorf(
			"One of ['service', 'service_name'] must be set to query VPC Endpoint Services")
	}

	req := &ec2.DescribeVpcEndpointServicesInput{
		ServiceNames: aws.StringSlice([]string{serviceName}),
	}

	log.Printf("[DEBUG] Reading VPC Endpoint Service: %s", req)
	resp, err := conn.DescribeVpcEndpointServices(req)
	if err != nil {
		return fmt.Errorf("error reading VPC Endpoint Service (%s): %s", serviceName, err)
	}

	if resp == nil || (len(resp.ServiceNames) == 0 && len(resp.ServiceDetails) == 0) {
		return fmt.Errorf("no matching VPC Endpoint Service found")
	}

	// Note: AWS Commercial now returns a response with `ServiceNames` and
	// `ServiceDetails`, but GovCloud responses only include `ServiceNames`
	if len(resp.ServiceDetails) == 0 {
		// GovCloud doesn't respect the filter.
		names := aws.StringValueSlice(resp.ServiceNames)
		for _, name := range names {
			if name == serviceName {
				d.SetId(strconv.Itoa(hashcode.String(name)))
				d.Set("service_name", name)
				return nil
			}
		}

		return fmt.Errorf("no matching VPC Endpoint Service found")
	}

	if len(resp.ServiceDetails) > 1 {
		return fmt.Errorf("multiple VPC Endpoint Services matched; use additional constraints to reduce matches to a single VPC Endpoint Service")
	}

	sd := resp.ServiceDetails[0]
	serviceName = aws.StringValue(sd.ServiceName)
	d.SetId(strconv.Itoa(hashcode.String(serviceName)))
	d.Set("service_name", serviceName)
	d.Set("acceptance_required", sd.AcceptanceRequired)
	err = d.Set("availability_zones", flattenStringSet(sd.AvailabilityZones))
	if err != nil {
		return fmt.Errorf("error setting availability_zones: %s", err)
	}
	err = d.Set("base_endpoint_dns_names", flattenStringSet(sd.BaseEndpointDnsNames))
	if err != nil {
		return fmt.Errorf("error setting base_endpoint_dns_names: %s", err)
	}
	d.Set("manages_vpc_endpoints", sd.ManagesVpcEndpoints)
	d.Set("owner", sd.Owner)
	d.Set("private_dns_name", sd.PrivateDnsName)
	d.Set("service_id", sd.ServiceId)
	d.Set("service_type", sd.ServiceType[0].ServiceType)
	err = d.Set("tags", tagsToMap(sd.Tags))
	if err != nil {
		return fmt.Errorf("error setting tags: %s", err)
	}
	d.Set("vpc_endpoint_policy_supported", sd.VpcEndpointPolicySupported)

	return nil
}
