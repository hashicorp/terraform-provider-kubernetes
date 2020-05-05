package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/keyvaluetags"
)

func resourceAwsVpc() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsVpcCreate,
		Read:   resourceAwsVpcRead,
		Update: resourceAwsVpcUpdate,
		Delete: resourceAwsVpcDelete,
		Importer: &schema.ResourceImporter{
			State: resourceAwsVpcInstanceImport,
		},
		CustomizeDiff: resourceAwsVpcCustomizeDiff,

		SchemaVersion: 1,
		MigrateState:  resourceAwsVpcMigrateState,

		Schema: map[string]*schema.Schema{
			"cidr_block": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.CIDRNetwork(16, 28),
			},

			"instance_tenancy": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      ec2.TenancyDefault,
				ValidateFunc: validation.StringInSlice([]string{ec2.TenancyDefault, ec2.TenancyDedicated}, false),
			},

			"enable_dns_hostnames": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"enable_dns_support": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"enable_classiclink": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"enable_classiclink_dns_support": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"assign_generated_ipv6_cidr_block": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"main_route_table_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"default_network_acl_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"dhcp_options_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"default_security_group_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"default_route_table_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"ipv6_association_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"ipv6_cidr_block": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsSchema(),

			"owner_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsVpcCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	// Create the VPC
	createOpts := &ec2.CreateVpcInput{
		CidrBlock:                   aws.String(d.Get("cidr_block").(string)),
		InstanceTenancy:             aws.String(d.Get("instance_tenancy").(string)),
		AmazonProvidedIpv6CidrBlock: aws.Bool(d.Get("assign_generated_ipv6_cidr_block").(bool)),
	}

	log.Printf("[DEBUG] VPC create config: %#v", *createOpts)
	vpcResp, err := conn.CreateVpc(createOpts)
	if err != nil {
		return fmt.Errorf("Error creating VPC: %s", err)
	}

	// Get the ID and store it
	vpc := vpcResp.Vpc
	d.SetId(*vpc.VpcId)
	log.Printf("[INFO] VPC ID: %s", d.Id())

	// Set partial mode and say that we setup the cidr block
	d.Partial(true)
	d.SetPartial("cidr_block")

	// Wait for the VPC to become available
	log.Printf(
		"[DEBUG] Waiting for VPC (%s) to become available",
		d.Id())
	stateConf := &resource.StateChangeConf{
		Pending: []string{"pending"},
		Target:  []string{"available"},
		Refresh: VPCStateRefreshFunc(conn, d.Id()),
		Timeout: 10 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf(
			"Error waiting for VPC (%s) to become available: %s",
			d.Id(), err)
	}

	if len(vpc.Ipv6CidrBlockAssociationSet) > 0 && vpc.Ipv6CidrBlockAssociationSet[0] != nil {
		log.Printf("[DEBUG] Waiting for EC2 VPC (%s) IPv6 CIDR to become associated", d.Id())
		if err := waitForEc2VpcIpv6CidrBlockAssociationCreate(conn, d.Id(), aws.StringValue(vpcResp.Vpc.Ipv6CidrBlockAssociationSet[0].AssociationId)); err != nil {
			return fmt.Errorf("error waiting for EC2 VPC (%s) IPv6 CIDR to become associated: %s", d.Id(), err)
		}
	}

	// You cannot modify the DNS resolution and DNS hostnames attributes in the same request. Use separate requests for each attribute.
	// Reference: https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_ModifyVpcAttribute.html

	if d.Get("enable_dns_hostnames").(bool) {
		input := &ec2.ModifyVpcAttributeInput{
			EnableDnsHostnames: &ec2.AttributeBooleanValue{
				Value: aws.Bool(true),
			},
			VpcId: aws.String(d.Id()),
		}

		if _, err := conn.ModifyVpcAttribute(input); err != nil {
			return fmt.Errorf("error enabling VPC (%s) DNS hostnames: %s", d.Id(), err)
		}

		d.SetPartial("enable_dns_hostnames")
	}

	// By default, only the enableDnsSupport attribute is set to true in a VPC created any other way.
	// Reference: https://docs.aws.amazon.com/vpc/latest/userguide/vpc-dns.html#vpc-dns-support

	if !d.Get("enable_dns_support").(bool) {
		input := &ec2.ModifyVpcAttributeInput{
			EnableDnsSupport: &ec2.AttributeBooleanValue{
				Value: aws.Bool(false),
			},
			VpcId: aws.String(d.Id()),
		}

		if _, err := conn.ModifyVpcAttribute(input); err != nil {
			return fmt.Errorf("error disabling VPC (%s) DNS support: %s", d.Id(), err)
		}

		d.SetPartial("enable_dns_support")
	}

	if d.Get("enable_classiclink").(bool) {
		input := &ec2.EnableVpcClassicLinkInput{
			VpcId: aws.String(d.Id()),
		}

		if _, err := conn.EnableVpcClassicLink(input); err != nil {
			return fmt.Errorf("error enabling VPC (%s) ClassicLink: %s", d.Id(), err)
		}

		d.SetPartial("enable_classiclink")
	}

	if d.Get("enable_classiclink_dns_support").(bool) {
		input := &ec2.EnableVpcClassicLinkDnsSupportInput{
			VpcId: aws.String(d.Id()),
		}

		if _, err := conn.EnableVpcClassicLinkDnsSupport(input); err != nil {
			return fmt.Errorf("error enabling VPC (%s) ClassicLink DNS support: %s", d.Id(), err)
		}

		d.SetPartial("enable_classiclink_dns_support")
	}

	if v := d.Get("tags").(map[string]interface{}); len(v) > 0 {
		// Handle EC2 eventual consistency on creation
		err := resource.Retry(5*time.Minute, func() *resource.RetryError {
			err := keyvaluetags.Ec2UpdateTags(conn, d.Id(), nil, v)

			if isAWSErr(err, "InvalidVpcID.NotFound", "") {
				return resource.RetryableError(err)
			}

			if err != nil {
				return resource.NonRetryableError(err)
			}

			return nil
		})

		if isResourceTimeoutError(err) {
			err = keyvaluetags.Ec2UpdateTags(conn, d.Id(), nil, v)
		}

		if err != nil {
			return fmt.Errorf("error adding tags: %s", err)
		}

		d.SetPartial("tags")
	}

	d.Partial(false)

	return resourceAwsVpcRead(d, meta)
}

func resourceAwsVpcRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	// Refresh the VPC state
	vpcRaw, _, err := VPCStateRefreshFunc(conn, d.Id())()
	if err != nil {
		return err
	}
	if vpcRaw == nil {
		d.SetId("")
		return nil
	}

	// VPC stuff
	vpc := vpcRaw.(*ec2.Vpc)
	vpcid := d.Id()
	d.Set("cidr_block", vpc.CidrBlock)
	d.Set("dhcp_options_id", vpc.DhcpOptionsId)
	d.Set("instance_tenancy", vpc.InstanceTenancy)

	// ARN
	arn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Service:   "ec2",
		Region:    meta.(*AWSClient).region,
		AccountID: meta.(*AWSClient).accountid,
		Resource:  fmt.Sprintf("vpc/%s", d.Id()),
	}.String()
	d.Set("arn", arn)

	if err := d.Set("tags", keyvaluetags.Ec2KeyValueTags(vpc.Tags).IgnoreAws().Map()); err != nil {
		return fmt.Errorf("error setting tags: %s", err)
	}

	d.Set("owner_id", vpc.OwnerId)

	// Make sure those values are set, if an IPv6 block exists it'll be set in the loop
	d.Set("assign_generated_ipv6_cidr_block", false)
	d.Set("ipv6_association_id", "")
	d.Set("ipv6_cidr_block", "")

	for _, a := range vpc.Ipv6CidrBlockAssociationSet {
		if aws.StringValue(a.Ipv6CidrBlockState.State) == ec2.VpcCidrBlockStateCodeAssociated { //we can only ever have 1 IPv6 block associated at once
			d.Set("assign_generated_ipv6_cidr_block", true)
			d.Set("ipv6_association_id", a.AssociationId)
			d.Set("ipv6_cidr_block", a.Ipv6CidrBlock)
		}
	}

	resp, err := awsVpcDescribeVpcAttribute("enableDnsSupport", vpcid, conn)
	if err != nil {
		return err
	}
	d.Set("enable_dns_support", resp.EnableDnsSupport.Value)

	resp, err = awsVpcDescribeVpcAttribute("enableDnsHostnames", vpcid, conn)
	if err != nil {
		return err
	}
	d.Set("enable_dns_hostnames", resp.EnableDnsHostnames.Value)

	describeClassiclinkOpts := &ec2.DescribeVpcClassicLinkInput{
		VpcIds: []*string{&vpcid},
	}

	// Classic Link is only available in regions that support EC2 Classic
	respClassiclink, err := conn.DescribeVpcClassicLink(describeClassiclinkOpts)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "UnsupportedOperation" {
			log.Printf("[WARN] VPC Classic Link is not supported in this region")
		} else {
			return err
		}
	} else {
		classiclink_enabled := false
		for _, v := range respClassiclink.Vpcs {
			if *v.VpcId == vpcid {
				if v.ClassicLinkEnabled != nil {
					classiclink_enabled = *v.ClassicLinkEnabled
				}
				break
			}
		}
		d.Set("enable_classiclink", classiclink_enabled)
	}

	describeClassiclinkDnsOpts := &ec2.DescribeVpcClassicLinkDnsSupportInput{
		VpcIds: []*string{&vpcid},
	}

	respClassiclinkDnsSupport, err := conn.DescribeVpcClassicLinkDnsSupport(describeClassiclinkDnsOpts)
	if err != nil {
		if isAWSErr(err, "UnsupportedOperation", "The functionality you requested is not available in this region") ||
			isAWSErr(err, "AuthFailure", "This request has been administratively disabled") {
			log.Printf("[WARN] VPC Classic Link DNS Support is not supported in this region")
		} else {
			return err
		}
	} else {
		classiclinkdns_enabled := false
		for _, v := range respClassiclinkDnsSupport.Vpcs {
			if *v.VpcId == vpcid {
				if v.ClassicLinkDnsSupported != nil {
					classiclinkdns_enabled = *v.ClassicLinkDnsSupported
				}
				break
			}
		}
		d.Set("enable_classiclink_dns_support", classiclinkdns_enabled)
	}

	routeTableId, err := resourceAwsVpcSetMainRouteTable(conn, vpcid)
	if err != nil {
		log.Printf("[WARN] Unable to set Main Route Table: %s", err)
	}
	d.Set("main_route_table_id", routeTableId)

	if err := resourceAwsVpcSetDefaultNetworkAcl(conn, d); err != nil {
		log.Printf("[WARN] Unable to set Default Network ACL: %s", err)
	}
	if err := resourceAwsVpcSetDefaultSecurityGroup(conn, d); err != nil {
		log.Printf("[WARN] Unable to set Default Security Group: %s", err)
	}
	if err := resourceAwsVpcSetDefaultRouteTable(conn, d); err != nil {
		log.Printf("[WARN] Unable to set Default Route Table: %s", err)
	}

	return nil
}

func resourceAwsVpcUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	// Turn on partial mode
	d.Partial(true)
	vpcid := d.Id()
	if d.HasChange("enable_dns_hostnames") {
		val := d.Get("enable_dns_hostnames").(bool)
		modifyOpts := &ec2.ModifyVpcAttributeInput{
			VpcId: &vpcid,
			EnableDnsHostnames: &ec2.AttributeBooleanValue{
				Value: &val,
			},
		}

		log.Printf(
			"[INFO] Modifying enable_dns_hostnames vpc attribute for %s: %s",
			d.Id(), modifyOpts)
		if _, err := conn.ModifyVpcAttribute(modifyOpts); err != nil {
			return err
		}

		d.SetPartial("enable_dns_hostnames")
	}

	_, hasEnableDnsSupportOption := d.GetOk("enable_dns_support")

	if !hasEnableDnsSupportOption || d.HasChange("enable_dns_support") {
		val := d.Get("enable_dns_support").(bool)
		modifyOpts := &ec2.ModifyVpcAttributeInput{
			VpcId: &vpcid,
			EnableDnsSupport: &ec2.AttributeBooleanValue{
				Value: &val,
			},
		}

		log.Printf(
			"[INFO] Modifying enable_dns_support vpc attribute for %s: %s",
			d.Id(), modifyOpts)
		if _, err := conn.ModifyVpcAttribute(modifyOpts); err != nil {
			return err
		}

		d.SetPartial("enable_dns_support")
	}

	if d.HasChange("enable_classiclink") {
		val := d.Get("enable_classiclink").(bool)
		if val {
			modifyOpts := &ec2.EnableVpcClassicLinkInput{
				VpcId: &vpcid,
			}
			log.Printf(
				"[INFO] Modifying enable_classiclink vpc attribute for %s: %#v",
				d.Id(), modifyOpts)
			if _, err := conn.EnableVpcClassicLink(modifyOpts); err != nil {
				return err
			}
		} else {
			modifyOpts := &ec2.DisableVpcClassicLinkInput{
				VpcId: &vpcid,
			}
			log.Printf(
				"[INFO] Modifying enable_classiclink vpc attribute for %s: %#v",
				d.Id(), modifyOpts)
			if _, err := conn.DisableVpcClassicLink(modifyOpts); err != nil {
				return err
			}
		}

		d.SetPartial("enable_classiclink")
	}

	if d.HasChange("enable_classiclink_dns_support") {
		val := d.Get("enable_classiclink_dns_support").(bool)
		if val {
			modifyOpts := &ec2.EnableVpcClassicLinkDnsSupportInput{
				VpcId: &vpcid,
			}
			log.Printf(
				"[INFO] Modifying enable_classiclink_dns_support vpc attribute for %s: %#v",
				d.Id(), modifyOpts)
			if _, err := conn.EnableVpcClassicLinkDnsSupport(modifyOpts); err != nil {
				return err
			}
		} else {
			modifyOpts := &ec2.DisableVpcClassicLinkDnsSupportInput{
				VpcId: &vpcid,
			}
			log.Printf(
				"[INFO] Modifying enable_classiclink_dns_support vpc attribute for %s: %#v",
				d.Id(), modifyOpts)
			if _, err := conn.DisableVpcClassicLinkDnsSupport(modifyOpts); err != nil {
				return err
			}
		}

		d.SetPartial("enable_classiclink_dns_support")
	}

	if d.HasChange("assign_generated_ipv6_cidr_block") {
		toAssign := d.Get("assign_generated_ipv6_cidr_block").(bool)

		log.Printf("[INFO] Modifying assign_generated_ipv6_cidr_block to %#v", toAssign)

		if toAssign {
			modifyOpts := &ec2.AssociateVpcCidrBlockInput{
				VpcId:                       &vpcid,
				AmazonProvidedIpv6CidrBlock: aws.Bool(toAssign),
			}
			log.Printf("[INFO] Enabling assign_generated_ipv6_cidr_block vpc attribute for %s: %#v",
				d.Id(), modifyOpts)
			resp, err := conn.AssociateVpcCidrBlock(modifyOpts)
			if err != nil {
				return err
			}

			log.Printf("[DEBUG] Waiting for EC2 VPC (%s) IPv6 CIDR to become associated", d.Id())
			if err := waitForEc2VpcIpv6CidrBlockAssociationCreate(conn, d.Id(), aws.StringValue(resp.Ipv6CidrBlockAssociation.AssociationId)); err != nil {
				return fmt.Errorf("error waiting for EC2 VPC (%s) IPv6 CIDR to become associated: %s", d.Id(), err)
			}
		} else {
			associationID := d.Get("ipv6_association_id").(string)
			modifyOpts := &ec2.DisassociateVpcCidrBlockInput{
				AssociationId: aws.String(associationID),
			}
			log.Printf("[INFO] Disabling assign_generated_ipv6_cidr_block vpc attribute for %s: %#v",
				d.Id(), modifyOpts)
			if _, err := conn.DisassociateVpcCidrBlock(modifyOpts); err != nil {
				return err
			}

			log.Printf("[DEBUG] Waiting for EC2 VPC (%s) IPv6 CIDR to become disassociated", d.Id())
			if err := waitForEc2VpcIpv6CidrBlockAssociationDelete(conn, d.Id(), associationID); err != nil {
				return fmt.Errorf("error waiting for EC2 VPC (%s) IPv6 CIDR to become disassociated: %s", d.Id(), err)
			}
		}

		d.SetPartial("assign_generated_ipv6_cidr_block")
	}

	if d.HasChange("instance_tenancy") {
		modifyOpts := &ec2.ModifyVpcTenancyInput{
			VpcId:           aws.String(vpcid),
			InstanceTenancy: aws.String(d.Get("instance_tenancy").(string)),
		}
		log.Printf(
			"[INFO] Modifying instance_tenancy vpc attribute for %s: %#v",
			d.Id(), modifyOpts)
		if _, err := conn.ModifyVpcTenancy(modifyOpts); err != nil {
			return err
		}

		d.SetPartial("instance_tenancy")
	}

	if d.HasChange("tags") {
		o, n := d.GetChange("tags")

		if err := keyvaluetags.Ec2UpdateTags(conn, d.Id(), o, n); err != nil {
			return fmt.Errorf("error updating tags: %s", err)
		}

		d.SetPartial("tags")
	}

	d.Partial(false)
	return resourceAwsVpcRead(d, meta)
}

func resourceAwsVpcDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	vpcID := d.Id()
	deleteVpcOpts := &ec2.DeleteVpcInput{
		VpcId: &vpcID,
	}
	log.Printf("[INFO] Deleting VPC: %s", d.Id())

	err := resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteVpc(deleteVpcOpts)
		if err == nil {
			return nil
		}

		if isAWSErr(err, "InvalidVpcID.NotFound", "") {
			return nil
		}
		if isAWSErr(err, "DependencyViolation", "") {
			return resource.RetryableError(err)
		}
		return resource.NonRetryableError(fmt.Errorf("Error deleting VPC: %s", err))
	})
	if isResourceTimeoutError(err) {
		_, err = conn.DeleteVpc(deleteVpcOpts)
		if isAWSErr(err, "InvalidVpcID.NotFound", "") {
			return nil
		}
	}

	if err != nil {
		return fmt.Errorf("Error deleting VPC: %s", err)
	}
	return nil
}

func resourceAwsVpcCustomizeDiff(diff *schema.ResourceDiff, v interface{}) error {
	if diff.HasChange("assign_generated_ipv6_cidr_block") {
		if err := diff.SetNewComputed("ipv6_association_id"); err != nil {
			return fmt.Errorf("error setting ipv6_association_id to computed: %s", err)
		}
		if err := diff.SetNewComputed("ipv6_cidr_block"); err != nil {
			return fmt.Errorf("error setting ipv6_cidr_block to computed: %s", err)
		}
	}
	if diff.HasChange("instance_tenancy") {
		old, new := diff.GetChange("instance_tenancy")
		if old.(string) != ec2.TenancyDedicated || new.(string) != ec2.TenancyDefault {
			diff.ForceNew("instance_tenancy")
		}
	}

	return nil
}

// VPCStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// a VPC.
func VPCStateRefreshFunc(conn *ec2.EC2, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		describeVpcOpts := &ec2.DescribeVpcsInput{
			VpcIds: []*string{aws.String(id)},
		}
		resp, err := conn.DescribeVpcs(describeVpcOpts)
		if err != nil {
			if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidVpcID.NotFound" {
				resp = nil
			} else {
				log.Printf("Error on VPCStateRefresh: %s", err)
				return nil, "", err
			}
		}

		if resp == nil {
			// Sometimes AWS just has consistency issues and doesn't see
			// our instance yet. Return an empty state.
			return nil, "", nil
		}

		vpc := resp.Vpcs[0]
		return vpc, *vpc.State, nil
	}
}

func Ipv6CidrStateRefreshFunc(conn *ec2.EC2, id string, associationId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		describeVpcOpts := &ec2.DescribeVpcsInput{
			VpcIds: []*string{aws.String(id)},
		}
		resp, err := conn.DescribeVpcs(describeVpcOpts)

		if isAWSErr(err, "InvalidVpcID.NotFound", "") {
			return nil, "", nil
		}

		if err != nil {
			return nil, "", err
		}

		if resp == nil || len(resp.Vpcs) == 0 || resp.Vpcs[0] == nil || resp.Vpcs[0].Ipv6CidrBlockAssociationSet == nil {
			// Sometimes AWS just has consistency issues and doesn't see
			// our instance yet. Return an empty state.
			return nil, "", nil
		}

		for _, association := range resp.Vpcs[0].Ipv6CidrBlockAssociationSet {
			if aws.StringValue(association.AssociationId) == associationId {
				return association, aws.StringValue(association.Ipv6CidrBlockState.State), nil
			}
		}

		return nil, "", nil
	}
}

func resourceAwsVpcSetDefaultNetworkAcl(conn *ec2.EC2, d *schema.ResourceData) error {
	filter1 := &ec2.Filter{
		Name:   aws.String("default"),
		Values: []*string{aws.String("true")},
	}
	filter2 := &ec2.Filter{
		Name:   aws.String("vpc-id"),
		Values: []*string{aws.String(d.Id())},
	}
	describeNetworkACLOpts := &ec2.DescribeNetworkAclsInput{
		Filters: []*ec2.Filter{filter1, filter2},
	}
	networkAclResp, err := conn.DescribeNetworkAcls(describeNetworkACLOpts)

	if err != nil {
		return err
	}
	if v := networkAclResp.NetworkAcls; len(v) > 0 {
		d.Set("default_network_acl_id", v[0].NetworkAclId)
	}

	return nil
}

func resourceAwsVpcSetDefaultSecurityGroup(conn *ec2.EC2, d *schema.ResourceData) error {
	filter1 := &ec2.Filter{
		Name:   aws.String("group-name"),
		Values: []*string{aws.String("default")},
	}
	filter2 := &ec2.Filter{
		Name:   aws.String("vpc-id"),
		Values: []*string{aws.String(d.Id())},
	}
	describeSgOpts := &ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{filter1, filter2},
	}
	securityGroupResp, err := conn.DescribeSecurityGroups(describeSgOpts)

	if err != nil {
		return err
	}
	if v := securityGroupResp.SecurityGroups; len(v) > 0 {
		d.Set("default_security_group_id", v[0].GroupId)
	}

	return nil
}

func resourceAwsVpcSetDefaultRouteTable(conn *ec2.EC2, d *schema.ResourceData) error {
	filter1 := &ec2.Filter{
		Name:   aws.String("association.main"),
		Values: []*string{aws.String("true")},
	}
	filter2 := &ec2.Filter{
		Name:   aws.String("vpc-id"),
		Values: []*string{aws.String(d.Id())},
	}

	findOpts := &ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{filter1, filter2},
	}

	resp, err := conn.DescribeRouteTables(findOpts)
	if err != nil {
		return err
	}

	if len(resp.RouteTables) < 1 || resp.RouteTables[0] == nil {
		return fmt.Errorf("Default Route table not found")
	}

	// There Can Be Only 1 ... Default Route Table
	d.Set("default_route_table_id", resp.RouteTables[0].RouteTableId)

	return nil
}

func resourceAwsVpcSetMainRouteTable(conn *ec2.EC2, vpcid string) (string, error) {
	filter1 := &ec2.Filter{
		Name:   aws.String("association.main"),
		Values: []*string{aws.String("true")},
	}
	filter2 := &ec2.Filter{
		Name:   aws.String("vpc-id"),
		Values: []*string{aws.String(vpcid)},
	}

	findOpts := &ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{filter1, filter2},
	}

	resp, err := conn.DescribeRouteTables(findOpts)
	if err != nil {
		return "", err
	}

	if len(resp.RouteTables) < 1 || resp.RouteTables[0] == nil {
		return "", fmt.Errorf("Main Route table not found")
	}

	// There Can Be Only 1 Main Route Table for a VPC
	return aws.StringValue(resp.RouteTables[0].RouteTableId), nil
}

func resourceAwsVpcInstanceImport(
	d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	d.Set("assign_generated_ipv6_cidr_block", false)
	return []*schema.ResourceData{d}, nil
}

func awsVpcDescribeVpcAttribute(attribute string, vpcId string, conn *ec2.EC2) (*ec2.DescribeVpcAttributeOutput, error) {
	describeAttrOpts := &ec2.DescribeVpcAttributeInput{
		Attribute: aws.String(attribute),
		VpcId:     aws.String(vpcId),
	}
	resp, err := conn.DescribeVpcAttribute(describeAttrOpts)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// vpcDescribe returns EC2 API information about the specified VPC.
// If the VPC doesn't exist, return nil.
func vpcDescribe(conn *ec2.EC2, vpcId string) (*ec2.Vpc, error) {
	resp, err := conn.DescribeVpcs(&ec2.DescribeVpcsInput{
		VpcIds: aws.StringSlice([]string{vpcId}),
	})
	if err != nil {
		if !isAWSErr(err, "InvalidVpcID.NotFound", "") {
			return nil, err
		}
		resp = nil
	}

	if resp == nil {
		return nil, nil
	}

	n := len(resp.Vpcs)
	switch n {
	case 0:
		return nil, nil

	case 1:
		return resp.Vpcs[0], nil

	default:
		return nil, fmt.Errorf("Found %d VPCs for %s, expected 1", n, vpcId)
	}
}

func waitForEc2VpcIpv6CidrBlockAssociationCreate(conn *ec2.EC2, vpcID, associationID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{
			ec2.VpcCidrBlockStateCodeAssociating,
			ec2.VpcCidrBlockStateCodeDisassociated,
		},
		Target:  []string{ec2.VpcCidrBlockStateCodeAssociated},
		Refresh: Ipv6CidrStateRefreshFunc(conn, vpcID, associationID),
		Timeout: 1 * time.Minute,
	}
	_, err := stateConf.WaitForState()

	return err
}

func waitForEc2VpcIpv6CidrBlockAssociationDelete(conn *ec2.EC2, vpcID, associationID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{
			ec2.VpcCidrBlockStateCodeAssociated,
			ec2.VpcCidrBlockStateCodeDisassociating,
		},
		Target:         []string{ec2.VpcCidrBlockStateCodeDisassociated},
		Refresh:        Ipv6CidrStateRefreshFunc(conn, vpcID, associationID),
		Timeout:        1 * time.Minute,
		NotFoundChecks: 1,
	}
	_, err := stateConf.WaitForState()

	return err
}
