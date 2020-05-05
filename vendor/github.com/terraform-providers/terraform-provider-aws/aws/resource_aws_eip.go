package aws

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceAwsEip() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEipCreate,
		Read:   resourceAwsEipRead,
		Update: resourceAwsEipUpdate,
		Delete: resourceAwsEipDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Read:   schema.DefaultTimeout(15 * time.Minute),
			Update: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(3 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"vpc": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"instance": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"network_interface": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"allocation_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"association_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"domain": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"public_ip": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"public_dns": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"private_ip": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"private_dns": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"associate_with_private_ip": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"public_ipv4_pool": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsEipCreate(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	// By default, we're not in a VPC
	domainOpt := ""
	if v := d.Get("vpc"); v != nil && v.(bool) {
		domainOpt = "vpc"
	}

	allocOpts := &ec2.AllocateAddressInput{
		Domain: aws.String(domainOpt),
	}

	if v, ok := d.GetOk("public_ipv4_pool"); ok {
		allocOpts.PublicIpv4Pool = aws.String(v.(string))
	}

	log.Printf("[DEBUG] EIP create configuration: %#v", allocOpts)
	allocResp, err := ec2conn.AllocateAddress(allocOpts)
	if err != nil {
		return fmt.Errorf("Error creating EIP: %s", err)
	}

	// The domain tells us if we're in a VPC or not
	d.Set("domain", allocResp.Domain)

	// Assign the eips (unique) allocation id for use later
	// the EIP api has a conditional unique ID (really), so
	// if we're in a VPC we need to save the ID as such, otherwise
	// it defaults to using the public IP
	log.Printf("[DEBUG] EIP Allocate: %#v", allocResp)
	if d.Get("domain").(string) == "vpc" {
		d.SetId(*allocResp.AllocationId)
	} else {
		d.SetId(*allocResp.PublicIp)
	}

	log.Printf("[INFO] EIP ID: %s (domain: %v)", d.Id(), *allocResp.Domain)

	if _, ok := d.GetOk("tags"); ok {
		if err := setTags(ec2conn, d); err != nil {
			return fmt.Errorf("Error creating EIP tags: %s", err)
		}
	}

	return resourceAwsEipUpdate(d, meta)
}

func resourceAwsEipRead(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	domain := resourceAwsEipDomain(d)
	id := d.Id()

	req := &ec2.DescribeAddressesInput{}

	if domain == "vpc" {
		req.AllocationIds = []*string{aws.String(id)}
	} else {
		req.PublicIps = []*string{aws.String(id)}
	}

	log.Printf(
		"[DEBUG] EIP describe configuration: %s (domain: %s)",
		req, domain)

	var err error
	var describeAddresses *ec2.DescribeAddressesOutput

	if d.IsNewResource() {
		err := resource.Retry(d.Timeout(schema.TimeoutRead), func() *resource.RetryError {
			describeAddresses, err = ec2conn.DescribeAddresses(req)
			if err != nil {
				awsErr, ok := err.(awserr.Error)
				if ok && (awsErr.Code() == "InvalidAllocationID.NotFound" ||
					awsErr.Code() == "InvalidAddress.NotFound") {
					return resource.RetryableError(err)
				}

				return resource.NonRetryableError(err)
			}
			return nil
		})
		if isResourceTimeoutError(err) {
			describeAddresses, err = ec2conn.DescribeAddresses(req)
		}
		if err != nil {
			return fmt.Errorf("Error retrieving EIP: %s", err)
		}
	} else {
		describeAddresses, err = ec2conn.DescribeAddresses(req)
		if err != nil {
			awsErr, ok := err.(awserr.Error)
			if ok && (awsErr.Code() == "InvalidAllocationID.NotFound" ||
				awsErr.Code() == "InvalidAddress.NotFound") {
				log.Printf("[WARN] EIP not found, removing from state: %s", req)
				d.SetId("")
				return nil
			}
			return err
		}
	}

	var address *ec2.Address

	// In the case that AWS returns more EIPs than we intend it to, we loop
	// over the returned addresses to see if it's in the list of results
	for _, addr := range describeAddresses.Addresses {
		if (domain == "vpc" && aws.StringValue(addr.AllocationId) == id) || aws.StringValue(addr.PublicIp) == id {
			address = addr
			break
		}
	}

	if address == nil {
		log.Printf("[WARN] EIP %q not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("association_id", address.AssociationId)
	if address.InstanceId != nil {
		d.Set("instance", address.InstanceId)
	} else {
		d.Set("instance", "")
	}
	if address.NetworkInterfaceId != nil {
		d.Set("network_interface", address.NetworkInterfaceId)
	} else {
		d.Set("network_interface", "")
	}

	region := *ec2conn.Config.Region
	d.Set("private_ip", address.PrivateIpAddress)
	if address.PrivateIpAddress != nil {
		dashIP := strings.Replace(*address.PrivateIpAddress, ".", "-", -1)

		if region == "us-east-1" {
			d.Set("private_dns", fmt.Sprintf("ip-%s.ec2.internal", dashIP))
		} else {
			d.Set("private_dns", fmt.Sprintf("ip-%s.%s.compute.internal", dashIP, region))
		}
	}
	d.Set("public_ip", address.PublicIp)
	if address.PublicIp != nil {
		dashIP := strings.Replace(*address.PublicIp, ".", "-", -1)

		if region == "us-east-1" {
			d.Set("public_dns", fmt.Sprintf("ec2-%s.compute-1.amazonaws.com", dashIP))
		} else {
			d.Set("public_dns", fmt.Sprintf("ec2-%s.%s.compute.amazonaws.com", dashIP, region))
		}
	}
	d.Set("public_ipv4_pool", address.PublicIpv4Pool)

	// On import (domain never set, which it must've been if we created),
	// set the 'vpc' attribute depending on if we're in a VPC.
	if address.Domain != nil {
		d.Set("vpc", *address.Domain == "vpc")
	}

	d.Set("domain", address.Domain)

	// Force ID to be an Allocation ID if we're on a VPC
	// This allows users to import the EIP based on the IP if they are in a VPC
	if *address.Domain == "vpc" && net.ParseIP(id) != nil {
		log.Printf("[DEBUG] Re-assigning EIP ID (%s) to it's Allocation ID (%s)", d.Id(), *address.AllocationId)
		d.SetId(*address.AllocationId)
	}

	d.Set("tags", tagsToMap(address.Tags))

	return nil
}

func resourceAwsEipUpdate(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	domain := resourceAwsEipDomain(d)

	// If we are updating an EIP that is not newly created, and we are attached to
	// an instance or interface, detach first.
	disassociate := false
	if !d.IsNewResource() {
		if d.HasChange("instance") && d.Get("instance").(string) != "" {
			disassociate = true
		} else if (d.HasChange("network_interface") || d.HasChange("associate_with_private_ip")) && d.Get("association_id").(string) != "" {
			disassociate = true
		}
	}
	if disassociate {
		if err := disassociateEip(d, meta); err != nil {
			return err
		}
	}

	// Associate to instance or interface if specified
	associate := false
	v_instance, ok_instance := d.GetOk("instance")
	v_interface, ok_interface := d.GetOk("network_interface")

	if d.HasChange("instance") && ok_instance {
		associate = true
	} else if (d.HasChange("network_interface") || d.HasChange("associate_with_private_ip")) && ok_interface {
		associate = true
	}
	if associate {
		instanceId := v_instance.(string)
		networkInterfaceId := v_interface.(string)

		assocOpts := &ec2.AssociateAddressInput{
			InstanceId: aws.String(instanceId),
			PublicIp:   aws.String(d.Id()),
		}

		// more unique ID conditionals
		if domain == "vpc" {
			var privateIpAddress *string
			if v := d.Get("associate_with_private_ip").(string); v != "" {
				privateIpAddress = aws.String(v)
			}
			assocOpts = &ec2.AssociateAddressInput{
				NetworkInterfaceId: aws.String(networkInterfaceId),
				InstanceId:         aws.String(instanceId),
				AllocationId:       aws.String(d.Id()),
				PrivateIpAddress:   privateIpAddress,
			}
		}

		log.Printf("[DEBUG] EIP associate configuration: %s (domain: %s)", assocOpts, domain)

		err := resource.Retry(d.Timeout(schema.TimeoutUpdate), func() *resource.RetryError {
			_, err := ec2conn.AssociateAddress(assocOpts)
			if err != nil {
				if isAWSErr(err, "InvalidAllocationID.NotFound", "") {
					return resource.RetryableError(err)
				}
				return resource.NonRetryableError(err)
			}
			return nil
		})
		if isResourceTimeoutError(err) {
			_, err = ec2conn.AssociateAddress(assocOpts)
		}
		if err != nil {
			// Prevent saving instance if association failed
			// e.g. missing internet gateway in VPC
			d.Set("instance", "")
			d.Set("network_interface", "")
			return fmt.Errorf("Failure associating EIP: %s", err)
		}
	}

	if _, ok := d.GetOk("tags"); ok {
		if err := setTags(ec2conn, d); err != nil {
			return fmt.Errorf("Error updating EIP tags: %s", err)
		}
	}

	return resourceAwsEipRead(d, meta)
}

func resourceAwsEipDelete(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	if err := resourceAwsEipRead(d, meta); err != nil {
		return err
	}
	if d.Id() == "" {
		// This might happen from the read
		return nil
	}

	// If we are attached to an instance or interface, detach first.
	if d.Get("instance").(string) != "" || d.Get("association_id").(string) != "" {
		if err := disassociateEip(d, meta); err != nil {
			return err
		}
	}

	domain := resourceAwsEipDomain(d)

	var input *ec2.ReleaseAddressInput
	switch domain {
	case "vpc":
		log.Printf("[DEBUG] EIP release (destroy) address allocation: %v", d.Id())
		input = &ec2.ReleaseAddressInput{
			AllocationId: aws.String(d.Id()),
		}
	case "standard":
		log.Printf("[DEBUG] EIP release (destroy) address: %v", d.Id())
		input = &ec2.ReleaseAddressInput{
			PublicIp: aws.String(d.Id()),
		}
	}

	err := resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		var err error
		_, err = ec2conn.ReleaseAddress(input)

		if err == nil {
			return nil
		}
		if _, ok := err.(awserr.Error); !ok {
			return resource.NonRetryableError(err)
		}

		return resource.RetryableError(err)
	})
	if isResourceTimeoutError(err) {
		_, err = ec2conn.ReleaseAddress(input)
	}
	if err != nil {
		return fmt.Errorf("Error releasing EIP address: %s", err)
	}
	return nil
}

func resourceAwsEipDomain(d *schema.ResourceData) string {
	if v, ok := d.GetOk("domain"); ok {
		return v.(string)
	} else if strings.Contains(d.Id(), "eipalloc") {
		// We have to do this for backwards compatibility since TF 0.1
		// didn't have the "domain" computed attribute.
		return "vpc"
	}

	return "standard"
}

func disassociateEip(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn
	log.Printf("[DEBUG] Disassociating EIP: %s", d.Id())
	var err error
	switch resourceAwsEipDomain(d) {
	case "vpc":
		associationID := d.Get("association_id").(string)
		if associationID == "" {
			// If assiciationID is empty, it means there's no association.
			// Hence this disassociation can be skipped.
			return nil
		}
		_, err = ec2conn.DisassociateAddress(&ec2.DisassociateAddressInput{
			AssociationId: aws.String(associationID),
		})
	case "standard":
		_, err = ec2conn.DisassociateAddress(&ec2.DisassociateAddressInput{
			PublicIp: aws.String(d.Get("public_ip").(string)),
		})
	}

	// First check if the association ID is not found. If this
	// is the case, then it was already disassociated somehow,
	// and that is okay. The most commmon reason for this is that
	// the instance or ENI it was attached it was destroyed.
	if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidAssociationID.NotFound" {
		err = nil
	}
	return err
}
