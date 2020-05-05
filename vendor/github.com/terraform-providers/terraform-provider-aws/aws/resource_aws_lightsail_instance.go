package aws

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/lightsail"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func resourceAwsLightsailInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsLightsailInstanceCreate,
		Read:   resourceAwsLightsailInstanceRead,
		Update: resourceAwsLightsailInstanceUpdate,
		Delete: resourceAwsLightsailInstanceDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.All(
					validation.StringLenBetween(2, 255),
					validation.StringMatch(regexp.MustCompile(`^[a-zA-Z]`), "must begin with an alphabetic character"),
					validation.StringMatch(regexp.MustCompile(`^[a-zA-Z0-9_\-.]+[^._\-]$`), "must contain only alphanumeric characters, underscores, hyphens, and dots"),
				),
			},
			"availability_zone": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"blueprint_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"bundle_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			// Optional attributes
			"key_pair_name": {
				// Not compatible with aws_key_pair (yet)
				// We'll need a new aws_lightsail_key_pair resource
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if old == "LightsailDefaultKeyPair" && new == "" {
						return true
					}
					return false
				},
			},

			// cannot be retrieved from the API
			"user_data": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			// additional info returned from the API
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"created_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"cpu_count": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"ram_size": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"ipv6_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"is_static_ip": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"private_ip_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"public_ip_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"username": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags": tagsSchema(),
		},
	}
}

func resourceAwsLightsailInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lightsailconn

	iName := d.Get("name").(string)

	req := lightsail.CreateInstancesInput{
		AvailabilityZone: aws.String(d.Get("availability_zone").(string)),
		BlueprintId:      aws.String(d.Get("blueprint_id").(string)),
		BundleId:         aws.String(d.Get("bundle_id").(string)),
		InstanceNames:    aws.StringSlice([]string{iName}),
	}

	if v, ok := d.GetOk("key_pair_name"); ok {
		req.KeyPairName = aws.String(v.(string))
	}
	if v, ok := d.GetOk("user_data"); ok {
		req.UserData = aws.String(v.(string))
	}

	tags := tagsFromMapLightsail(d.Get("tags").(map[string]interface{}))

	if len(tags) != 0 {
		req.Tags = tags
	}

	resp, err := conn.CreateInstances(&req)
	if err != nil {
		return err
	}

	if len(resp.Operations) == 0 {
		return fmt.Errorf("No operations found for CreateInstance request")
	}

	op := resp.Operations[0]
	d.SetId(d.Get("name").(string))

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"Started"},
		Target:     []string{"Completed", "Succeeded"},
		Refresh:    resourceAwsLightsailOperationRefreshFunc(op.Id, meta),
		Timeout:    10 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		// We don't return an error here because the Create call succeeded
		log.Printf("[ERR] Error waiting for instance (%s) to become ready: %s", d.Id(), err)
	}

	return resourceAwsLightsailInstanceRead(d, meta)
}

func resourceAwsLightsailInstanceRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lightsailconn
	resp, err := conn.GetInstance(&lightsail.GetInstanceInput{
		InstanceName: aws.String(d.Id()),
	})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "NotFoundException" {
				log.Printf("[WARN] Lightsail Instance (%s) not found, removing from state", d.Id())
				d.SetId("")
				return nil
			}
			return err
		}
		return err
	}

	if resp == nil {
		log.Printf("[WARN] Lightsail Instance (%s) not found, nil response from server, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	i := resp.Instance

	d.Set("availability_zone", i.Location.AvailabilityZone)
	d.Set("blueprint_id", i.BlueprintId)
	d.Set("bundle_id", i.BundleId)
	d.Set("key_pair_name", i.SshKeyName)
	d.Set("name", i.Name)

	// additional attributes
	d.Set("arn", i.Arn)
	d.Set("username", i.Username)
	d.Set("created_at", i.CreatedAt.Format(time.RFC3339))
	d.Set("cpu_count", i.Hardware.CpuCount)
	d.Set("ram_size", strconv.FormatFloat(*i.Hardware.RamSizeInGb, 'f', 0, 64))
	d.Set("ipv6_address", i.Ipv6Address)
	d.Set("is_static_ip", i.IsStaticIp)
	d.Set("private_ip_address", i.PrivateIpAddress)
	d.Set("public_ip_address", i.PublicIpAddress)

	if err := d.Set("tags", tagsToMapLightsail(i.Tags)); err != nil {
		return fmt.Errorf("Error setting tags: %s", err)
	}

	return nil
}

func resourceAwsLightsailInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lightsailconn
	resp, err := conn.DeleteInstance(&lightsail.DeleteInstanceInput{
		InstanceName: aws.String(d.Id()),
	})

	if err != nil {
		return err
	}

	op := resp.Operations[0]

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"Started"},
		Target:     []string{"Completed", "Succeeded"},
		Refresh:    resourceAwsLightsailOperationRefreshFunc(op.Id, meta),
		Timeout:    10 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for instance (%s) to become destroyed: %s",
			d.Id(), err)
	}

	return nil
}

func resourceAwsLightsailInstanceUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lightsailconn

	if d.HasChange("tags") {
		if err := setTagsLightsail(conn, d); err != nil {
			return err
		}
		d.SetPartial("tags")
	}

	return resourceAwsLightsailInstanceRead(d, meta)
}

// method to check the status of an Operation, which is returned from
// Create/Delete methods.
// Status's are an aws.OperationStatus enum:
// - NotStarted
// - Started
// - Failed
// - Completed
// - Succeeded (not documented?)
func resourceAwsLightsailOperationRefreshFunc(
	oid *string, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		conn := meta.(*AWSClient).lightsailconn
		log.Printf("[DEBUG] Checking if Lightsail Operation (%s) is Completed", *oid)
		o, err := conn.GetOperation(&lightsail.GetOperationInput{
			OperationId: oid,
		})
		if err != nil {
			return o, "FAILED", err
		}

		if o.Operation == nil {
			return nil, "Failed", fmt.Errorf("Error retrieving Operation info for operation (%s)", *oid)
		}

		log.Printf("[DEBUG] Lightsail Operation (%s) is currently %q", *oid, *o.Operation.Status)
		return o, *o.Operation.Status, nil
	}
}
