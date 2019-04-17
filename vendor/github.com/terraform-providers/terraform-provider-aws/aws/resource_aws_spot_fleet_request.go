package aws

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsSpotFleetRequest() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSpotFleetRequestCreate,
		Read:   resourceAwsSpotFleetRequestRead,
		Delete: resourceAwsSpotFleetRequestDelete,
		Update: resourceAwsSpotFleetRequestUpdate,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},

		SchemaVersion: 1,
		MigrateState:  resourceAwsSpotFleetRequestMigrateState,

		Schema: map[string]*schema.Schema{
			"iam_fleet_role": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},
			"replace_unhealthy_instances": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  false,
			},
			"wait_for_fulfillment": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: false,
				Default:  false,
			},
			// http://docs.aws.amazon.com/sdk-for-go/api/service/ec2.html#type-SpotFleetLaunchSpecification
			// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_SpotFleetLaunchSpecification.html
			"launch_specification": {
				Type:     schema.TypeSet,
				Required: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"vpc_security_group_ids": {
							Type:     schema.TypeSet,
							Optional: true,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
						"associate_public_ip_address": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"ebs_block_device": {
							Type:     schema.TypeSet,
							Optional: true,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"delete_on_termination": {
										Type:     schema.TypeBool,
										Optional: true,
										Default:  true,
										ForceNew: true,
									},
									"device_name": {
										Type:     schema.TypeString,
										Required: true,
										ForceNew: true,
									},
									"encrypted": {
										Type:     schema.TypeBool,
										Optional: true,
										Computed: true,
										ForceNew: true,
									},
									"iops": {
										Type:     schema.TypeInt,
										Optional: true,
										Computed: true,
										ForceNew: true,
									},
									"snapshot_id": {
										Type:     schema.TypeString,
										Optional: true,
										Computed: true,
										ForceNew: true,
									},
									"volume_size": {
										Type:     schema.TypeInt,
										Optional: true,
										Computed: true,
										ForceNew: true,
									},
									"volume_type": {
										Type:     schema.TypeString,
										Optional: true,
										Computed: true,
										ForceNew: true,
									},
								},
							},
							Set: hashEbsBlockDevice,
						},
						"ephemeral_block_device": {
							Type:     schema.TypeSet,
							Optional: true,
							Computed: true,
							ForceNew: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"device_name": {
										Type:     schema.TypeString,
										Required: true,
									},
									"virtual_name": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
							Set: hashEphemeralBlockDevice,
						},
						"root_block_device": {
							// TODO: This is a set because we don't support singleton
							//       sub-resources today. We'll enforce that the set only ever has
							//       length zero or one below. When TF gains support for
							//       sub-resources this can be converted.
							Type:     schema.TypeSet,
							Optional: true,
							Computed: true,
							Elem: &schema.Resource{
								// "You can only modify the volume size, volume type, and Delete on
								// Termination flag on the block device mapping entry for the root
								// device volume." - bit.ly/ec2bdmap
								Schema: map[string]*schema.Schema{
									"delete_on_termination": {
										Type:     schema.TypeBool,
										Optional: true,
										Default:  true,
										ForceNew: true,
									},
									"iops": {
										Type:     schema.TypeInt,
										Optional: true,
										Computed: true,
										ForceNew: true,
									},
									"volume_size": {
										Type:     schema.TypeInt,
										Optional: true,
										Computed: true,
										ForceNew: true,
									},
									"volume_type": {
										Type:     schema.TypeString,
										Optional: true,
										Computed: true,
										ForceNew: true,
									},
								},
							},
							Set: hashRootBlockDevice,
						},
						"ebs_optimized": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"iam_instance_profile": {
							Type:     schema.TypeString,
							ForceNew: true,
							Optional: true,
						},
						"iam_instance_profile_arn": {
							Type:     schema.TypeString,
							ForceNew: true,
							Optional: true,
						},
						"ami": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"instance_type": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"key_name": {
							Type:         schema.TypeString,
							Optional:     true,
							ForceNew:     true,
							Computed:     true,
							ValidateFunc: validation.NoZeroValues,
						},
						"monitoring": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"placement_group": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},
						"placement_tenancy": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"spot_price": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"user_data": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							StateFunc: func(v interface{}) string {
								switch v.(type) {
								case string:
									return userDataHashSum(v.(string))
								default:
									return ""
								}
							},
						},
						"weighted_capacity": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"subnet_id": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},
						"availability_zone": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},
						"tags": {
							Type:     schema.TypeMap,
							Optional: true,
							ForceNew: true,
						},
					},
				},
				Set: hashLaunchSpecification,
			},
			// Everything on a spot fleet is ForceNew except target_capacity
			"target_capacity": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: false,
			},
			"allocation_strategy": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "lowestPrice",
				ForceNew: true,
			},
			"instance_pools_to_use_count": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  1,
				ForceNew: true,
			},
			"excess_capacity_termination_policy": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Default",
				ForceNew: false,
			},
			"instance_interruption_behaviour": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "terminate",
				ForceNew: true,
			},
			"spot_price": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"terminate_instances_with_expiration": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
			"valid_from": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.ValidateRFC3339TimeString,
			},
			"valid_until": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.ValidateRFC3339TimeString,
			},
			"fleet_type": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  ec2.FleetTypeMaintain,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					ec2.FleetTypeMaintain,
					ec2.FleetTypeRequest,
				}, false),
			},
			"spot_request_state": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"client_token": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"load_balancers": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"target_group_arns": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func buildSpotFleetLaunchSpecification(d map[string]interface{}, meta interface{}) (*ec2.SpotFleetLaunchSpecification, error) {
	conn := meta.(*AWSClient).ec2conn

	opts := &ec2.SpotFleetLaunchSpecification{
		ImageId:      aws.String(d["ami"].(string)),
		InstanceType: aws.String(d["instance_type"].(string)),
		SpotPrice:    aws.String(d["spot_price"].(string)),
	}

	placement := new(ec2.SpotPlacement)
	if v, ok := d["availability_zone"]; ok {
		placement.AvailabilityZone = aws.String(v.(string))
		opts.Placement = placement
	}

	if v, ok := d["placement_tenancy"]; ok {
		placement.Tenancy = aws.String(v.(string))
		opts.Placement = placement
	}

	if v, ok := d["ebs_optimized"]; ok {
		opts.EbsOptimized = aws.Bool(v.(bool))
	}

	if v, ok := d["monitoring"]; ok {
		opts.Monitoring = &ec2.SpotFleetMonitoring{
			Enabled: aws.Bool(v.(bool)),
		}
	}

	if v, ok := d["iam_instance_profile"]; ok {
		opts.IamInstanceProfile = &ec2.IamInstanceProfileSpecification{
			Name: aws.String(v.(string)),
		}
	}

	if v, ok := d["iam_instance_profile_arn"]; ok && v.(string) != "" {
		opts.IamInstanceProfile = &ec2.IamInstanceProfileSpecification{
			Arn: aws.String(v.(string)),
		}
	}

	if v, ok := d["user_data"]; ok {
		opts.UserData = aws.String(base64Encode([]byte(v.(string))))
	}

	if v, ok := d["key_name"]; ok && v != "" {
		opts.KeyName = aws.String(v.(string))
	}

	if v, ok := d["weighted_capacity"]; ok && v != "" {
		wc, err := strconv.ParseFloat(v.(string), 64)
		if err != nil {
			return nil, err
		}
		opts.WeightedCapacity = aws.Float64(wc)
	}

	var securityGroupIds []*string
	if v, ok := d["vpc_security_group_ids"]; ok {
		if s := v.(*schema.Set); s.Len() > 0 {
			for _, v := range s.List() {
				securityGroupIds = append(securityGroupIds, aws.String(v.(string)))
			}
		}
	}

	if m, ok := d["tags"].(map[string]interface{}); ok && len(m) > 0 {
		tagsSpec := make([]*ec2.SpotFleetTagSpecification, 0)

		tags := tagsFromMap(m)

		spec := &ec2.SpotFleetTagSpecification{
			ResourceType: aws.String("instance"),
			Tags:         tags,
		}

		tagsSpec = append(tagsSpec, spec)

		opts.TagSpecifications = tagsSpec
	}

	subnetId, hasSubnetId := d["subnet_id"]
	if hasSubnetId {
		opts.SubnetId = aws.String(subnetId.(string))
	}

	associatePublicIpAddress, hasPublicIpAddress := d["associate_public_ip_address"]
	if hasPublicIpAddress && associatePublicIpAddress.(bool) && hasSubnetId {

		// If we have a non-default VPC / Subnet specified, we can flag
		// AssociatePublicIpAddress to get a Public IP assigned. By default these are not provided.
		// You cannot specify both SubnetId and the NetworkInterface.0.* parameters though, otherwise
		// you get: Network interfaces and an instance-level subnet ID may not be specified on the same request
		// You also need to attach Security Groups to the NetworkInterface instead of the instance,
		// to avoid: Network interfaces and an instance-level security groups may not be specified on
		// the same request
		ni := &ec2.InstanceNetworkInterfaceSpecification{
			AssociatePublicIpAddress: aws.Bool(true),
			DeleteOnTermination:      aws.Bool(true),
			DeviceIndex:              aws.Int64(int64(0)),
			SubnetId:                 aws.String(subnetId.(string)),
			Groups:                   securityGroupIds,
		}

		opts.NetworkInterfaces = []*ec2.InstanceNetworkInterfaceSpecification{ni}
		opts.SubnetId = aws.String("")
	} else {
		for _, id := range securityGroupIds {
			opts.SecurityGroups = append(opts.SecurityGroups, &ec2.GroupIdentifier{GroupId: id})
		}
	}

	blockDevices, err := readSpotFleetBlockDeviceMappingsFromConfig(d, conn)
	if err != nil {
		return nil, err
	}
	if len(blockDevices) > 0 {
		opts.BlockDeviceMappings = blockDevices
	}

	return opts, nil
}

func readSpotFleetBlockDeviceMappingsFromConfig(
	d map[string]interface{}, conn *ec2.EC2) ([]*ec2.BlockDeviceMapping, error) {
	blockDevices := make([]*ec2.BlockDeviceMapping, 0)

	if v, ok := d["ebs_block_device"]; ok {
		vL := v.(*schema.Set).List()
		for _, v := range vL {
			bd := v.(map[string]interface{})
			ebs := &ec2.EbsBlockDevice{
				DeleteOnTermination: aws.Bool(bd["delete_on_termination"].(bool)),
			}

			if v, ok := bd["snapshot_id"].(string); ok && v != "" {
				ebs.SnapshotId = aws.String(v)
			}

			if v, ok := bd["encrypted"].(bool); ok && v {
				ebs.Encrypted = aws.Bool(v)
			}

			if v, ok := bd["volume_size"].(int); ok && v != 0 {
				ebs.VolumeSize = aws.Int64(int64(v))
			}

			if v, ok := bd["volume_type"].(string); ok && v != "" {
				ebs.VolumeType = aws.String(v)
			}

			if v, ok := bd["iops"].(int); ok && v > 0 {
				ebs.Iops = aws.Int64(int64(v))
			}

			blockDevices = append(blockDevices, &ec2.BlockDeviceMapping{
				DeviceName: aws.String(bd["device_name"].(string)),
				Ebs:        ebs,
			})
		}
	}

	if v, ok := d["ephemeral_block_device"]; ok {
		vL := v.(*schema.Set).List()
		for _, v := range vL {
			bd := v.(map[string]interface{})
			blockDevices = append(blockDevices, &ec2.BlockDeviceMapping{
				DeviceName:  aws.String(bd["device_name"].(string)),
				VirtualName: aws.String(bd["virtual_name"].(string)),
			})
		}
	}

	if v, ok := d["root_block_device"]; ok {
		vL := v.(*schema.Set).List()
		if len(vL) > 1 {
			return nil, fmt.Errorf("Cannot specify more than one root_block_device.")
		}
		for _, v := range vL {
			bd := v.(map[string]interface{})
			ebs := &ec2.EbsBlockDevice{
				DeleteOnTermination: aws.Bool(bd["delete_on_termination"].(bool)),
			}

			if v, ok := bd["volume_size"].(int); ok && v != 0 {
				ebs.VolumeSize = aws.Int64(int64(v))
			}

			if v, ok := bd["volume_type"].(string); ok && v != "" {
				ebs.VolumeType = aws.String(v)
			}

			if v, ok := bd["iops"].(int); ok && v > 0 {
				ebs.Iops = aws.Int64(int64(v))
			}

			if dn, err := fetchRootDeviceName(d["ami"].(string), conn); err == nil {
				if dn == nil {
					return nil, fmt.Errorf(
						"Expected 1 AMI for ID: %s, got none",
						d["ami"].(string))
				}

				blockDevices = append(blockDevices, &ec2.BlockDeviceMapping{
					DeviceName: dn,
					Ebs:        ebs,
				})
			} else {
				return nil, err
			}
		}
	}

	return blockDevices, nil
}

func buildAwsSpotFleetLaunchSpecifications(
	d *schema.ResourceData, meta interface{}) ([]*ec2.SpotFleetLaunchSpecification, error) {

	user_specs := d.Get("launch_specification").(*schema.Set).List()
	specs := make([]*ec2.SpotFleetLaunchSpecification, len(user_specs))
	for i, user_spec := range user_specs {
		user_spec_map := user_spec.(map[string]interface{})
		// panic: interface conversion: interface {} is map[string]interface {}, not *schema.ResourceData
		opts, err := buildSpotFleetLaunchSpecification(user_spec_map, meta)
		if err != nil {
			return nil, err
		}
		specs[i] = opts
	}

	return specs, nil
}

func resourceAwsSpotFleetRequestCreate(d *schema.ResourceData, meta interface{}) error {
	// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_RequestSpotFleet.html
	conn := meta.(*AWSClient).ec2conn

	launch_specs, err := buildAwsSpotFleetLaunchSpecifications(d, meta)
	if err != nil {
		return err
	}

	// http://docs.aws.amazon.com/sdk-for-go/api/service/ec2.html#type-SpotFleetRequestConfigData
	spotFleetConfig := &ec2.SpotFleetRequestConfigData{
		IamFleetRole:                     aws.String(d.Get("iam_fleet_role").(string)),
		LaunchSpecifications:             launch_specs,
		TargetCapacity:                   aws.Int64(int64(d.Get("target_capacity").(int))),
		ClientToken:                      aws.String(resource.UniqueId()),
		TerminateInstancesWithExpiration: aws.Bool(d.Get("terminate_instances_with_expiration").(bool)),
		ReplaceUnhealthyInstances:        aws.Bool(d.Get("replace_unhealthy_instances").(bool)),
		InstanceInterruptionBehavior:     aws.String(d.Get("instance_interruption_behaviour").(string)),
		Type:                             aws.String(d.Get("fleet_type").(string)),
	}

	if v, ok := d.GetOk("excess_capacity_termination_policy"); ok {
		spotFleetConfig.ExcessCapacityTerminationPolicy = aws.String(v.(string))
	}

	if v, ok := d.GetOk("allocation_strategy"); ok {
		spotFleetConfig.AllocationStrategy = aws.String(v.(string))
	} else {
		spotFleetConfig.AllocationStrategy = aws.String("lowestPrice")
	}

	if v, ok := d.GetOk("instance_pools_to_use_count"); ok && v.(int) != 1 {
		spotFleetConfig.InstancePoolsToUseCount = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("spot_price"); ok && v.(string) != "" {
		spotFleetConfig.SpotPrice = aws.String(v.(string))
	}

	if v, ok := d.GetOk("valid_from"); ok {
		valid_from, err := time.Parse(time.RFC3339, v.(string))
		if err != nil {
			return err
		}
		spotFleetConfig.ValidFrom = aws.Time(valid_from)
	}

	if v, ok := d.GetOk("valid_until"); ok {
		valid_until, err := time.Parse(time.RFC3339, v.(string))
		if err != nil {
			return err
		}
		spotFleetConfig.ValidUntil = aws.Time(valid_until)
	} else {
		valid_until := time.Now().Add(24 * time.Hour)
		spotFleetConfig.ValidUntil = aws.Time(valid_until)
	}

	if v, ok := d.GetOk("load_balancers"); ok && v.(*schema.Set).Len() > 0 {
		var elbNames []*ec2.ClassicLoadBalancer
		for _, v := range v.(*schema.Set).List() {
			elbNames = append(elbNames, &ec2.ClassicLoadBalancer{
				Name: aws.String(v.(string)),
			})
		}
		if spotFleetConfig.LoadBalancersConfig == nil {
			spotFleetConfig.LoadBalancersConfig = &ec2.LoadBalancersConfig{}
		}
		spotFleetConfig.LoadBalancersConfig.ClassicLoadBalancersConfig = &ec2.ClassicLoadBalancersConfig{
			ClassicLoadBalancers: elbNames,
		}
	}

	if v, ok := d.GetOk("target_group_arns"); ok && v.(*schema.Set).Len() > 0 {
		var targetGroups []*ec2.TargetGroup
		for _, v := range v.(*schema.Set).List() {
			targetGroups = append(targetGroups, &ec2.TargetGroup{
				Arn: aws.String(v.(string)),
			})
		}
		if spotFleetConfig.LoadBalancersConfig == nil {
			spotFleetConfig.LoadBalancersConfig = &ec2.LoadBalancersConfig{}
		}
		spotFleetConfig.LoadBalancersConfig.TargetGroupsConfig = &ec2.TargetGroupsConfig{
			TargetGroups: targetGroups,
		}
	}

	// http://docs.aws.amazon.com/sdk-for-go/api/service/ec2.html#type-RequestSpotFleetInput
	spotFleetOpts := &ec2.RequestSpotFleetInput{
		SpotFleetRequestConfig: spotFleetConfig,
		DryRun:                 aws.Bool(false),
	}

	log.Printf("[DEBUG] Requesting spot fleet with these opts: %+v", spotFleetOpts)

	// Since IAM is eventually consistent, we retry creation as a newly created role may not
	// take effect immediately, resulting in an InvalidSpotFleetRequestConfig error
	var resp *ec2.RequestSpotFleetOutput
	err = resource.Retry(10*time.Minute, func() *resource.RetryError {
		var err error
		resp, err = conn.RequestSpotFleet(spotFleetOpts)

		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				// IAM is eventually consistent :/
				if awsErr.Code() == "InvalidSpotFleetRequestConfig" {
					return resource.RetryableError(
						fmt.Errorf("Error creating Spot fleet request, retrying: %s", err))
				}
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("Error requesting spot fleet: %s", err)
	}

	d.SetId(*resp.SpotFleetRequestId)

	log.Printf("[INFO] Spot Fleet Request ID: %s", d.Id())
	log.Println("[INFO] Waiting for Spot Fleet Request to be active")
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"submitted"},
		Target:     []string{"active"},
		Refresh:    resourceAwsSpotFleetRequestStateRefreshFunc(d, meta),
		Timeout:    10 * time.Minute,
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	if d.Get("wait_for_fulfillment").(bool) {
		log.Println("[INFO] Waiting for Spot Fleet Request to be fulfilled")
		spotStateConf := &resource.StateChangeConf{
			Pending:    []string{"pending_fulfillment"},
			Target:     []string{"fulfilled"},
			Refresh:    resourceAwsSpotFleetRequestFulfillmentRefreshFunc(d.Id(), meta.(*AWSClient).ec2conn),
			Timeout:    d.Timeout(schema.TimeoutCreate),
			Delay:      10 * time.Second,
			MinTimeout: 3 * time.Second,
		}

		_, err = spotStateConf.WaitForState()

		if err != nil {
			return err
		}
	}

	return resourceAwsSpotFleetRequestRead(d, meta)
}

func resourceAwsSpotFleetRequestStateRefreshFunc(d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		conn := meta.(*AWSClient).ec2conn
		req := &ec2.DescribeSpotFleetRequestsInput{
			SpotFleetRequestIds: []*string{aws.String(d.Id())},
		}
		resp, err := conn.DescribeSpotFleetRequests(req)

		if err != nil {
			log.Printf("Error on retrieving Spot Fleet Request when waiting: %s", err)
			return nil, "", nil
		}

		if resp == nil {
			return nil, "", nil
		}

		if len(resp.SpotFleetRequestConfigs) == 0 {
			return nil, "", nil
		}

		spotFleetRequest := resp.SpotFleetRequestConfigs[0]

		return spotFleetRequest, *spotFleetRequest.SpotFleetRequestState, nil
	}
}

func resourceAwsSpotFleetRequestFulfillmentRefreshFunc(id string, conn *ec2.EC2) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		req := &ec2.DescribeSpotFleetRequestsInput{
			SpotFleetRequestIds: []*string{aws.String(id)},
		}
		resp, err := conn.DescribeSpotFleetRequests(req)

		if err != nil {
			log.Printf("Error on retrieving Spot Fleet Request when waiting: %s", err)
			return nil, "", nil
		}

		if resp == nil {
			return nil, "", nil
		}

		if len(resp.SpotFleetRequestConfigs) == 0 {
			return nil, "", nil
		}

		cfg := resp.SpotFleetRequestConfigs[0]
		status := *cfg.ActivityStatus

		var fleetError error
		if status == ec2.ActivityStatusError {
			var events []*ec2.HistoryRecord

			// Query "information" events (e.g. launchSpecUnusable b/c low bid price)
			out, err := conn.DescribeSpotFleetRequestHistory(&ec2.DescribeSpotFleetRequestHistoryInput{
				EventType:          aws.String("information"),
				SpotFleetRequestId: aws.String(id),
				StartTime:          cfg.CreateTime,
			})
			if err != nil {
				log.Printf("[ERROR] Failed to get the reason of 'error' state: %s", err)
			}
			if len(out.HistoryRecords) > 0 {
				events = out.HistoryRecords
			}

			out, err = conn.DescribeSpotFleetRequestHistory(&ec2.DescribeSpotFleetRequestHistoryInput{
				EventType:          aws.String(ec2.EventTypeError),
				SpotFleetRequestId: aws.String(id),
				StartTime:          cfg.CreateTime,
			})
			if err != nil {
				log.Printf("[ERROR] Failed to get the reason of 'error' state: %s", err)
			}
			if len(out.HistoryRecords) > 0 {
				events = append(events, out.HistoryRecords...)
			}

			if len(events) > 0 {
				fleetError = fmt.Errorf("Last events: %v", events)
			}
		}

		return cfg, status, fleetError
	}
}

func resourceAwsSpotFleetRequestRead(d *schema.ResourceData, meta interface{}) error {
	// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeSpotFleetRequests.html
	conn := meta.(*AWSClient).ec2conn

	req := &ec2.DescribeSpotFleetRequestsInput{
		SpotFleetRequestIds: []*string{aws.String(d.Id())},
	}
	resp, err := conn.DescribeSpotFleetRequests(req)

	if err != nil {
		// If the spot request was not found, return nil so that we can show
		// that it is gone.
		ec2err, ok := err.(awserr.Error)
		if ok && ec2err.Code() == "InvalidSpotFleetRequestId.NotFound" {
			d.SetId("")
			return nil
		}

		// Some other error, report it
		return err
	}

	sfr := resp.SpotFleetRequestConfigs[0]

	// if the request is cancelled, then it is gone
	cancelledStates := map[string]bool{
		"cancelled":             true,
		"cancelled_running":     true,
		"cancelled_terminating": true,
	}
	if _, ok := cancelledStates[*sfr.SpotFleetRequestState]; ok {
		d.SetId("")
		return nil
	}

	d.SetId(*sfr.SpotFleetRequestId)
	d.Set("spot_request_state", aws.StringValue(sfr.SpotFleetRequestState))

	config := sfr.SpotFleetRequestConfig

	if config.AllocationStrategy != nil {
		d.Set("allocation_strategy", aws.StringValue(config.AllocationStrategy))
	}

	if config.InstancePoolsToUseCount != nil {
		d.Set("instance_pools_to_use_count", aws.Int64Value(config.InstancePoolsToUseCount))
	}

	if config.ClientToken != nil {
		d.Set("client_token", aws.StringValue(config.ClientToken))
	}

	if config.ExcessCapacityTerminationPolicy != nil {
		d.Set("excess_capacity_termination_policy",
			aws.StringValue(config.ExcessCapacityTerminationPolicy))
	}

	if config.IamFleetRole != nil {
		d.Set("iam_fleet_role", aws.StringValue(config.IamFleetRole))
	}

	if config.SpotPrice != nil {
		d.Set("spot_price", aws.StringValue(config.SpotPrice))
	}

	if config.TargetCapacity != nil {
		d.Set("target_capacity", aws.Int64Value(config.TargetCapacity))
	}

	if config.TerminateInstancesWithExpiration != nil {
		d.Set("terminate_instances_with_expiration",
			aws.BoolValue(config.TerminateInstancesWithExpiration))
	}

	if config.ValidFrom != nil {
		d.Set("valid_from",
			aws.TimeValue(config.ValidFrom).Format(time.RFC3339))
	}

	if config.ValidUntil != nil {
		d.Set("valid_until",
			aws.TimeValue(config.ValidUntil).Format(time.RFC3339))
	}

	d.Set("replace_unhealthy_instances", config.ReplaceUnhealthyInstances)
	d.Set("instance_interruption_behaviour", config.InstanceInterruptionBehavior)
	d.Set("fleet_type", config.Type)
	d.Set("launch_specification", launchSpecsToSet(config.LaunchSpecifications, conn))

	return nil
}

func launchSpecsToSet(launchSpecs []*ec2.SpotFleetLaunchSpecification, conn *ec2.EC2) *schema.Set {
	specSet := &schema.Set{F: hashLaunchSpecification}
	for _, spec := range launchSpecs {
		rootDeviceName, err := fetchRootDeviceName(aws.StringValue(spec.ImageId), conn)
		if err != nil {
			log.Panic(err)
		}

		specSet.Add(launchSpecToMap(spec, rootDeviceName))
	}
	return specSet
}

func launchSpecToMap(l *ec2.SpotFleetLaunchSpecification, rootDevName *string) map[string]interface{} {
	m := make(map[string]interface{})

	m["root_block_device"] = rootBlockDeviceToSet(l.BlockDeviceMappings, rootDevName)
	m["ebs_block_device"] = ebsBlockDevicesToSet(l.BlockDeviceMappings, rootDevName)
	m["ephemeral_block_device"] = ephemeralBlockDevicesToSet(l.BlockDeviceMappings)

	if l.ImageId != nil {
		m["ami"] = aws.StringValue(l.ImageId)
	}

	if l.InstanceType != nil {
		m["instance_type"] = aws.StringValue(l.InstanceType)
	}

	if l.SpotPrice != nil {
		m["spot_price"] = aws.StringValue(l.SpotPrice)
	}

	if l.EbsOptimized != nil {
		m["ebs_optimized"] = aws.BoolValue(l.EbsOptimized)
	}

	if l.Monitoring != nil && l.Monitoring.Enabled != nil {
		m["monitoring"] = aws.BoolValue(l.Monitoring.Enabled)
	}

	if l.IamInstanceProfile != nil && l.IamInstanceProfile.Name != nil {
		m["iam_instance_profile"] = aws.StringValue(l.IamInstanceProfile.Name)
	}

	if l.IamInstanceProfile != nil && l.IamInstanceProfile.Arn != nil {
		m["iam_instance_profile_arn"] = aws.StringValue(l.IamInstanceProfile.Arn)
	}

	if l.UserData != nil {
		m["user_data"] = userDataHashSum(aws.StringValue(l.UserData))
	}

	if l.KeyName != nil {
		m["key_name"] = aws.StringValue(l.KeyName)
	}

	if l.Placement != nil {
		m["availability_zone"] = aws.StringValue(l.Placement.AvailabilityZone)
	}

	if l.SubnetId != nil {
		m["subnet_id"] = aws.StringValue(l.SubnetId)
	}

	securityGroupIds := &schema.Set{F: schema.HashString}
	if len(l.NetworkInterfaces) > 0 {
		m["associate_public_ip_address"] = aws.BoolValue(l.NetworkInterfaces[0].AssociatePublicIpAddress)
		m["subnet_id"] = aws.StringValue(l.NetworkInterfaces[0].SubnetId)

		for _, group := range l.NetworkInterfaces[0].Groups {
			securityGroupIds.Add(aws.StringValue(group))
		}
	} else {
		for _, group := range l.SecurityGroups {
			securityGroupIds.Add(aws.StringValue(group.GroupId))
		}
	}
	m["vpc_security_group_ids"] = securityGroupIds

	if l.WeightedCapacity != nil {
		m["weighted_capacity"] = strconv.FormatFloat(*l.WeightedCapacity, 'f', 0, 64)
	}

	if l.TagSpecifications != nil {
		for _, tagSpecs := range l.TagSpecifications {
			// only "instance" tags are currently supported: http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_SpotFleetTagSpecification.html
			if *(tagSpecs.ResourceType) == "instance" {
				m["tags"] = tagsToMap(tagSpecs.Tags)
			}
		}
	}

	return m
}

func ebsBlockDevicesToSet(bdm []*ec2.BlockDeviceMapping, rootDevName *string) *schema.Set {
	set := &schema.Set{F: hashEbsBlockDevice}

	for _, val := range bdm {
		if val.Ebs != nil {
			m := make(map[string]interface{})

			ebs := val.Ebs

			if val.DeviceName != nil {
				if aws.StringValue(rootDevName) == aws.StringValue(val.DeviceName) {
					continue
				}

				m["device_name"] = aws.StringValue(val.DeviceName)
			}

			if ebs.DeleteOnTermination != nil {
				m["delete_on_termination"] = aws.BoolValue(ebs.DeleteOnTermination)
			}

			if ebs.SnapshotId != nil {
				m["snapshot_id"] = aws.StringValue(ebs.SnapshotId)
			}

			if ebs.Encrypted != nil {
				m["encrypted"] = aws.BoolValue(ebs.Encrypted)
			}

			if ebs.VolumeSize != nil {
				m["volume_size"] = aws.Int64Value(ebs.VolumeSize)
			}

			if ebs.VolumeType != nil {
				m["volume_type"] = aws.StringValue(ebs.VolumeType)
			}

			if ebs.Iops != nil {
				m["iops"] = aws.Int64Value(ebs.Iops)
			}

			set.Add(m)
		}
	}

	return set
}

func ephemeralBlockDevicesToSet(bdm []*ec2.BlockDeviceMapping) *schema.Set {
	set := &schema.Set{F: hashEphemeralBlockDevice}

	for _, val := range bdm {
		if val.VirtualName != nil {
			m := make(map[string]interface{})
			m["virtual_name"] = aws.StringValue(val.VirtualName)

			if val.DeviceName != nil {
				m["device_name"] = aws.StringValue(val.DeviceName)
			}

			set.Add(m)
		}
	}

	return set
}

func rootBlockDeviceToSet(
	bdm []*ec2.BlockDeviceMapping,
	rootDevName *string,
) *schema.Set {
	set := &schema.Set{F: hashRootBlockDevice}

	if rootDevName != nil {
		for _, val := range bdm {
			if aws.StringValue(val.DeviceName) == aws.StringValue(rootDevName) {
				m := make(map[string]interface{})
				if val.Ebs.DeleteOnTermination != nil {
					m["delete_on_termination"] = aws.BoolValue(val.Ebs.DeleteOnTermination)
				}

				if val.Ebs.VolumeSize != nil {
					m["volume_size"] = aws.Int64Value(val.Ebs.VolumeSize)
				}

				if val.Ebs.VolumeType != nil {
					m["volume_type"] = aws.StringValue(val.Ebs.VolumeType)
				}

				if val.Ebs.Iops != nil {
					m["iops"] = aws.Int64Value(val.Ebs.Iops)
				}

				set.Add(m)
			}
		}
	}

	return set
}

func resourceAwsSpotFleetRequestUpdate(d *schema.ResourceData, meta interface{}) error {
	// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_ModifySpotFleetRequest.html
	conn := meta.(*AWSClient).ec2conn

	d.Partial(true)

	req := &ec2.ModifySpotFleetRequestInput{
		SpotFleetRequestId: aws.String(d.Id()),
	}

	if val, ok := d.GetOk("target_capacity"); ok {
		req.TargetCapacity = aws.Int64(int64(val.(int)))
	}

	if val, ok := d.GetOk("excess_capacity_termination_policy"); ok {
		req.ExcessCapacityTerminationPolicy = aws.String(val.(string))
	}

	resp, err := conn.ModifySpotFleetRequest(req)
	if err == nil && aws.BoolValue(resp.Return) {
		// TODO: rollback to old values?
	}

	return nil
}

func resourceAwsSpotFleetRequestDelete(d *schema.ResourceData, meta interface{}) error {
	// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_CancelSpotFleetRequests.html
	conn := meta.(*AWSClient).ec2conn
	terminateInstances := d.Get("terminate_instances_with_expiration").(bool)

	log.Printf("[INFO] Cancelling spot fleet request: %s", d.Id())
	err := deleteSpotFleetRequest(d.Id(), terminateInstances, d.Timeout(schema.TimeoutDelete), conn)
	if err != nil {
		return fmt.Errorf("error deleting spot request (%s): %s", d.Id(), err)
	}

	return nil
}

func deleteSpotFleetRequest(spotFleetRequestID string, terminateInstances bool, timeout time.Duration, conn *ec2.EC2) error {
	_, err := conn.CancelSpotFleetRequests(&ec2.CancelSpotFleetRequestsInput{
		SpotFleetRequestIds: []*string{aws.String(spotFleetRequestID)},
		TerminateInstances:  aws.Bool(terminateInstances),
	})
	if err != nil {
		return err
	}

	// Only wait for instance termination if requested
	if !terminateInstances {
		return nil
	}

	return resource.Retry(timeout, func() *resource.RetryError {
		resp, err := conn.DescribeSpotFleetInstances(&ec2.DescribeSpotFleetInstancesInput{
			SpotFleetRequestId: aws.String(spotFleetRequestID),
		})
		if err != nil {
			return resource.NonRetryableError(err)
		}

		if len(resp.ActiveInstances) == 0 {
			log.Printf("[DEBUG] Active instance count is 0 for Spot Fleet Request (%s), removing", spotFleetRequestID)
			return nil
		}

		log.Printf("[DEBUG] Active instance count in Spot Fleet Request (%s): %d", spotFleetRequestID, len(resp.ActiveInstances))

		return resource.RetryableError(
			fmt.Errorf("fleet still has (%d) running instances", len(resp.ActiveInstances)))
	})
}

func hashEphemeralBlockDevice(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["device_name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["virtual_name"].(string)))
	return hashcode.String(buf.String())
}

func hashRootBlockDevice(v interface{}) int {
	// there can be only one root device; no need to hash anything
	return 0
}

func hashLaunchSpecification(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["ami"].(string)))
	if m["availability_zone"] != "" {
		buf.WriteString(fmt.Sprintf("%s-", m["availability_zone"].(string)))
	}
	if m["subnet_id"] != "" {
		buf.WriteString(fmt.Sprintf("%s-", m["subnet_id"].(string)))
	}
	buf.WriteString(fmt.Sprintf("%s-", m["instance_type"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["spot_price"].(string)))
	return hashcode.String(buf.String())
}

func hashEbsBlockDevice(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	if name, ok := m["device_name"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", name.(string)))
	}
	if id, ok := m["snapshot_id"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", id.(string)))
	}
	return hashcode.String(buf.String())
}
