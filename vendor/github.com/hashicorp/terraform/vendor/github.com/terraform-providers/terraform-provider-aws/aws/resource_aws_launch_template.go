package aws

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsLaunchTemplate() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsLaunchTemplateCreate,
		Read:   resourceAwsLaunchTemplateRead,
		Update: resourceAwsLaunchTemplateUpdate,
		Delete: resourceAwsLaunchTemplateDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
				ValidateFunc:  validateLaunchTemplateName,
			},

			"name_prefix": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name"},
				ValidateFunc:  validateLaunchTemplateName,
			},

			"description": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringLenBetween(0, 255),
			},

			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"default_version": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"latest_version": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"block_device_mappings": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"device_name": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"no_device": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"virtual_name": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"ebs": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"delete_on_termination": {
										Type:     schema.TypeBool,
										Optional: true,
									},
									"encrypted": {
										Type:     schema.TypeBool,
										Optional: true,
									},
									"iops": {
										Type:     schema.TypeInt,
										Computed: true,
										Optional: true,
									},
									"kms_key_id": {
										Type:         schema.TypeString,
										Optional:     true,
										ValidateFunc: validateArn,
									},
									"snapshot_id": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"volume_size": {
										Type:     schema.TypeInt,
										Optional: true,
										Computed: true,
									},
									"volume_type": {
										Type:     schema.TypeString,
										Optional: true,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},

			"credit_specification": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cpu_credits": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"disable_api_termination": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"ebs_optimized": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"elastic_gpu_specifications": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},

			"iam_instance_profile": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"arn": {
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{"iam_instance_profile.0.name"},
							ValidateFunc:  validateArn,
						},
						"name": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"image_id": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"instance_initiated_shutdown_behavior": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: validation.StringInSlice([]string{
					ec2.ShutdownBehaviorStop,
					ec2.ShutdownBehaviorTerminate,
				}, false),
			},

			"instance_market_options": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"market_type": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validation.StringInSlice([]string{ec2.MarketTypeSpot}, false),
						},
						"spot_options": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"block_duration_minutes": {
										Type:     schema.TypeInt,
										Optional: true,
									},
									"instance_interruption_behavior": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"max_price": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"spot_instance_type": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"valid_until": {
										Type:         schema.TypeString,
										Optional:     true,
										Computed:     true,
										ValidateFunc: validation.ValidateRFC3339TimeString,
									},
								},
							},
						},
					},
				},
			},

			"instance_type": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"kernel_id": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"key_name": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"monitoring": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},

			"network_interfaces": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"associate_public_ip_address": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"delete_on_termination": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"description": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"device_index": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"security_groups": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"ipv6_address_count": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"ipv6_addresses": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"network_interface_id": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"private_ip_address": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"ipv4_addresses": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"ipv4_address_count": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"subnet_id": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"placement": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"affinity": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"availability_zone": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"group_name": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"host_id": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"spread_domain": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"tenancy": {
							Type:     schema.TypeString,
							Optional: true,
							ValidateFunc: validation.StringInSlice([]string{
								ec2.TenancyDedicated,
								ec2.TenancyDefault,
								ec2.TenancyHost,
							}, false),
						},
					},
				},
			},

			"ram_disk_id": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"security_group_names": {
				Type:          schema.TypeSet,
				Optional:      true,
				Elem:          &schema.Schema{Type: schema.TypeString},
				ConflictsWith: []string{"vpc_security_group_ids"},
			},

			"vpc_security_group_ids": {
				Type:          schema.TypeSet,
				Optional:      true,
				Elem:          &schema.Schema{Type: schema.TypeString},
				ConflictsWith: []string{"security_group_names"},
			},

			"tag_specifications": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"resource_type": {
							Type:     schema.TypeString,
							Optional: true,
							ValidateFunc: validation.StringInSlice([]string{
								"instance",
								"volume",
							}, false),
						},
						"tags": tagsSchema(),
					},
				},
			},

			"user_data": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsLaunchTemplateCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	var ltName string
	if v, ok := d.GetOk("name"); ok {
		ltName = v.(string)
	} else if v, ok := d.GetOk("name_prefix"); ok {
		ltName = resource.PrefixedUniqueId(v.(string))
	} else {
		ltName = resource.UniqueId()
	}

	launchTemplateData, err := buildLaunchTemplateData(d, meta)
	if err != nil {
		return err
	}

	launchTemplateOpts := &ec2.CreateLaunchTemplateInput{
		ClientToken:        aws.String(resource.UniqueId()),
		LaunchTemplateName: aws.String(ltName),
		LaunchTemplateData: launchTemplateData,
	}

	resp, err := conn.CreateLaunchTemplate(launchTemplateOpts)
	if err != nil {
		return err
	}

	launchTemplate := resp.LaunchTemplate
	d.SetId(*launchTemplate.LaunchTemplateId)

	log.Printf("[DEBUG] Launch Template created: %q (version %d)",
		*launchTemplate.LaunchTemplateId, *launchTemplate.LatestVersionNumber)

	return resourceAwsLaunchTemplateUpdate(d, meta)
}

func resourceAwsLaunchTemplateRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	log.Printf("[DEBUG] Reading launch template %s", d.Id())

	dlt, err := conn.DescribeLaunchTemplates(&ec2.DescribeLaunchTemplatesInput{
		LaunchTemplateIds: []*string{aws.String(d.Id())},
	})
	if err != nil {
		if isAWSErr(err, ec2.LaunchTemplateErrorCodeLaunchTemplateIdDoesNotExist, "") {
			log.Printf("[WARN] launch template (%s) not found - removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error getting launch template: %s", err)
	}
	if len(dlt.LaunchTemplates) == 0 {
		log.Printf("[WARN] launch template (%s) not found - removing from state", d.Id())
		d.SetId("")
		return nil
	}
	if *dlt.LaunchTemplates[0].LaunchTemplateId != d.Id() {
		return fmt.Errorf("Unable to find launch template: %#v", dlt.LaunchTemplates)
	}

	log.Printf("[DEBUG] Found launch template %s", d.Id())

	lt := dlt.LaunchTemplates[0]
	d.Set("name", lt.LaunchTemplateName)
	d.Set("latest_version", lt.LatestVersionNumber)
	d.Set("default_version", lt.DefaultVersionNumber)
	d.Set("tags", tagsToMap(lt.Tags))

	arn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Service:   "ec2",
		Region:    meta.(*AWSClient).region,
		AccountID: meta.(*AWSClient).accountid,
		Resource:  fmt.Sprintf("launch-template/%s", d.Id()),
	}.String()
	d.Set("arn", arn)

	version := strconv.Itoa(int(*lt.LatestVersionNumber))
	dltv, err := conn.DescribeLaunchTemplateVersions(&ec2.DescribeLaunchTemplateVersionsInput{
		LaunchTemplateId: aws.String(d.Id()),
		Versions:         []*string{aws.String(version)},
	})
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Received launch template version %q (version %d)", d.Id(), *lt.LatestVersionNumber)

	ltData := dltv.LaunchTemplateVersions[0].LaunchTemplateData

	d.Set("disable_api_termination", ltData.DisableApiTermination)
	d.Set("ebs_optimized", ltData.EbsOptimized)
	d.Set("image_id", ltData.ImageId)
	d.Set("instance_initiated_shutdown_behavior", ltData.InstanceInitiatedShutdownBehavior)
	d.Set("instance_type", ltData.InstanceType)
	d.Set("kernel_id", ltData.KernelId)
	d.Set("key_name", ltData.KeyName)
	d.Set("ram_disk_id", ltData.RamDiskId)
	d.Set("security_group_names", aws.StringValueSlice(ltData.SecurityGroups))
	d.Set("user_data", ltData.UserData)
	d.Set("vpc_security_group_ids", aws.StringValueSlice(ltData.SecurityGroupIds))

	if err := d.Set("block_device_mappings", getBlockDeviceMappings(ltData.BlockDeviceMappings)); err != nil {
		return err
	}

	if strings.HasPrefix(aws.StringValue(ltData.InstanceType), "t2") {
		if err := d.Set("credit_specification", getCreditSpecification(ltData.CreditSpecification)); err != nil {
			return err
		}
	}

	if err := d.Set("elastic_gpu_specifications", getElasticGpuSpecifications(ltData.ElasticGpuSpecifications)); err != nil {
		return err
	}

	if err := d.Set("iam_instance_profile", getIamInstanceProfile(ltData.IamInstanceProfile)); err != nil {
		return err
	}

	if err := d.Set("instance_market_options", getInstanceMarketOptions(ltData.InstanceMarketOptions)); err != nil {
		return err
	}

	if err := d.Set("monitoring", getMonitoring(ltData.Monitoring)); err != nil {
		return err
	}

	if err := d.Set("network_interfaces", getNetworkInterfaces(ltData.NetworkInterfaces)); err != nil {
		return err
	}

	if err := d.Set("placement", getPlacement(ltData.Placement)); err != nil {
		return err
	}

	if err := d.Set("tag_specifications", getTagSpecifications(ltData.TagSpecifications)); err != nil {
		return err
	}

	return nil
}

func resourceAwsLaunchTemplateUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	if !d.IsNewResource() {
		launchTemplateData, err := buildLaunchTemplateData(d, meta)
		if err != nil {
			return err
		}

		launchTemplateVersionOpts := &ec2.CreateLaunchTemplateVersionInput{
			ClientToken:        aws.String(resource.UniqueId()),
			LaunchTemplateId:   aws.String(d.Id()),
			LaunchTemplateData: launchTemplateData,
		}

		_, createErr := conn.CreateLaunchTemplateVersion(launchTemplateVersionOpts)
		if createErr != nil {
			return createErr
		}
	}

	d.Partial(true)

	if err := setTags(conn, d); err != nil {
		return err
	} else {
		d.SetPartial("tags")
	}

	d.Partial(false)

	return resourceAwsLaunchTemplateRead(d, meta)
}

func resourceAwsLaunchTemplateDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	log.Printf("[DEBUG] Launch Template destroy: %v", d.Id())
	_, err := conn.DeleteLaunchTemplate(&ec2.DeleteLaunchTemplateInput{
		LaunchTemplateId: aws.String(d.Id()),
	})
	if err != nil {
		if isAWSErr(err, ec2.LaunchTemplateErrorCodeLaunchTemplateIdDoesNotExist, "") {
			return nil
		}
		return err
	}

	log.Printf("[DEBUG] Launch Template deleted: %v", d.Id())
	return nil
}

func getBlockDeviceMappings(m []*ec2.LaunchTemplateBlockDeviceMapping) []interface{} {
	s := []interface{}{}
	for _, v := range m {
		mapping := map[string]interface{}{
			"device_name":  aws.StringValue(v.DeviceName),
			"virtual_name": aws.StringValue(v.VirtualName),
		}
		if v.NoDevice != nil {
			mapping["no_device"] = *v.NoDevice
		}
		if v.Ebs != nil {
			ebs := map[string]interface{}{
				"delete_on_termination": aws.BoolValue(v.Ebs.DeleteOnTermination),
				"encrypted":             aws.BoolValue(v.Ebs.Encrypted),
				"volume_size":           int(aws.Int64Value(v.Ebs.VolumeSize)),
				"volume_type":           aws.StringValue(v.Ebs.VolumeType),
			}
			if v.Ebs.Iops != nil {
				ebs["iops"] = aws.Int64Value(v.Ebs.Iops)
			}
			if v.Ebs.KmsKeyId != nil {
				ebs["kms_key_id"] = aws.StringValue(v.Ebs.KmsKeyId)
			}
			if v.Ebs.SnapshotId != nil {
				ebs["snapshot_id"] = aws.StringValue(v.Ebs.SnapshotId)
			}

			mapping["ebs"] = []interface{}{ebs}
		}
		s = append(s, mapping)
	}
	return s
}

func getCreditSpecification(cs *ec2.CreditSpecification) []interface{} {
	s := []interface{}{}
	if cs != nil {
		s = append(s, map[string]interface{}{
			"cpu_credits": aws.StringValue(cs.CpuCredits),
		})
	}
	return s
}

func getElasticGpuSpecifications(e []*ec2.ElasticGpuSpecificationResponse) []interface{} {
	s := []interface{}{}
	for _, v := range e {
		s = append(s, map[string]interface{}{
			"type": aws.StringValue(v.Type),
		})
	}
	return s
}

func getIamInstanceProfile(i *ec2.LaunchTemplateIamInstanceProfileSpecification) []interface{} {
	s := []interface{}{}
	if i != nil {
		s = append(s, map[string]interface{}{
			"arn":  aws.StringValue(i.Arn),
			"name": aws.StringValue(i.Name),
		})
	}
	return s
}

func getInstanceMarketOptions(m *ec2.LaunchTemplateInstanceMarketOptions) []interface{} {
	s := []interface{}{}
	if m != nil {
		mo := map[string]interface{}{
			"market_type": aws.StringValue(m.MarketType),
		}
		spot := []interface{}{}
		so := m.SpotOptions
		if so != nil {
			spot = append(spot, map[string]interface{}{
				"block_duration_minutes":         aws.Int64Value(so.BlockDurationMinutes),
				"instance_interruption_behavior": aws.StringValue(so.InstanceInterruptionBehavior),
				"max_price":                      aws.StringValue(so.MaxPrice),
				"spot_instance_type":             aws.StringValue(so.SpotInstanceType),
				"valid_until":                    aws.TimeValue(so.ValidUntil).Format(time.RFC3339),
			})
			mo["spot_options"] = spot
		}
		s = append(s, mo)
	}
	return s
}

func getMonitoring(m *ec2.LaunchTemplatesMonitoring) []interface{} {
	s := []interface{}{}
	if m != nil {
		mo := map[string]interface{}{
			"enabled": aws.BoolValue(m.Enabled),
		}
		s = append(s, mo)
	}
	return s
}

func getNetworkInterfaces(n []*ec2.LaunchTemplateInstanceNetworkInterfaceSpecification) []interface{} {
	s := []interface{}{}
	for _, v := range n {
		var ipv6Addresses []string
		var ipv4Addresses []string

		networkInterface := map[string]interface{}{
			"associate_public_ip_address": aws.BoolValue(v.AssociatePublicIpAddress),
			"delete_on_termination":       aws.BoolValue(v.DeleteOnTermination),
			"description":                 aws.StringValue(v.Description),
			"device_index":                aws.Int64Value(v.DeviceIndex),
			"ipv4_address_count":          aws.Int64Value(v.SecondaryPrivateIpAddressCount),
			"ipv6_address_count":          aws.Int64Value(v.Ipv6AddressCount),
			"network_interface_id":        aws.StringValue(v.NetworkInterfaceId),
			"private_ip_address":          aws.StringValue(v.PrivateIpAddress),
			"subnet_id":                   aws.StringValue(v.SubnetId),
		}

		for _, address := range v.Ipv6Addresses {
			ipv6Addresses = append(ipv6Addresses, aws.StringValue(address.Ipv6Address))
		}
		if len(ipv6Addresses) > 0 {
			networkInterface["ipv6_addresses"] = ipv6Addresses
		}

		for _, address := range v.PrivateIpAddresses {
			ipv4Addresses = append(ipv4Addresses, aws.StringValue(address.PrivateIpAddress))
		}
		if len(ipv4Addresses) > 0 {
			networkInterface["ipv4_addresses"] = ipv4Addresses
		}

		if len(v.Groups) > 0 {
			raw, ok := networkInterface["security_groups"]
			if !ok {
				raw = schema.NewSet(schema.HashString, nil)
			}
			list := raw.(*schema.Set)

			for _, group := range v.Groups {
				list.Add(aws.StringValue(group))
			}

			networkInterface["security_groups"] = list
		}

		s = append(s, networkInterface)
	}
	return s
}

func getPlacement(p *ec2.LaunchTemplatePlacement) []interface{} {
	s := []interface{}{}
	if p != nil {
		s = append(s, map[string]interface{}{
			"affinity":          aws.StringValue(p.Affinity),
			"availability_zone": aws.StringValue(p.AvailabilityZone),
			"group_name":        aws.StringValue(p.GroupName),
			"host_id":           aws.StringValue(p.HostId),
			"spread_domain":     aws.StringValue(p.SpreadDomain),
			"tenancy":           aws.StringValue(p.Tenancy),
		})
	}
	return s
}

func getTagSpecifications(t []*ec2.LaunchTemplateTagSpecification) []interface{} {
	s := []interface{}{}
	for _, v := range t {
		s = append(s, map[string]interface{}{
			"resource_type": aws.StringValue(v.ResourceType),
			"tags":          tagsToMap(v.Tags),
		})
	}
	return s
}

func buildLaunchTemplateData(d *schema.ResourceData, meta interface{}) (*ec2.RequestLaunchTemplateData, error) {
	opts := &ec2.RequestLaunchTemplateData{
		UserData: aws.String(d.Get("user_data").(string)),
	}

	if v, ok := d.GetOk("image_id"); ok {
		opts.ImageId = aws.String(v.(string))
	}

	if v, ok := d.GetOk("instance_initiated_shutdown_behavior"); ok {
		opts.InstanceInitiatedShutdownBehavior = aws.String(v.(string))
	}

	instanceType := d.Get("instance_type").(string)
	if instanceType != "" {
		opts.InstanceType = aws.String(instanceType)
	}

	if v, ok := d.GetOk("kernel_id"); ok {
		opts.KernelId = aws.String(v.(string))
	}

	if v, ok := d.GetOk("key_name"); ok {
		opts.KeyName = aws.String(v.(string))
	}

	if v, ok := d.GetOk("ram_disk_id"); ok {
		opts.RamDiskId = aws.String(v.(string))
	}

	if v, ok := d.GetOk("disable_api_termination"); ok {
		opts.DisableApiTermination = aws.Bool(v.(bool))
	}

	if v, ok := d.GetOk("ebs_optimized"); ok {
		opts.EbsOptimized = aws.Bool(v.(bool))
	}

	if v, ok := d.GetOk("security_group_names"); ok {
		opts.SecurityGroups = expandStringList(v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("vpc_security_group_ids"); ok {
		opts.SecurityGroupIds = expandStringList(v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("block_device_mappings"); ok {
		var blockDeviceMappings []*ec2.LaunchTemplateBlockDeviceMappingRequest
		bdms := v.([]interface{})

		for _, bdm := range bdms {
			blockDeviceMappings = append(blockDeviceMappings, readBlockDeviceMappingFromConfig(bdm.(map[string]interface{})))
		}
		opts.BlockDeviceMappings = blockDeviceMappings
	}

	if v, ok := d.GetOk("credit_specification"); ok && strings.HasPrefix(instanceType, "t2") {
		cs := v.([]interface{})

		if len(cs) > 0 {
			opts.CreditSpecification = readCreditSpecificationFromConfig(cs[0].(map[string]interface{}))
		}
	}

	if v, ok := d.GetOk("elastic_gpu_specifications"); ok {
		var elasticGpuSpecifications []*ec2.ElasticGpuSpecification
		egsList := v.([]interface{})

		for _, egs := range egsList {
			elasticGpuSpecifications = append(elasticGpuSpecifications, readElasticGpuSpecificationsFromConfig(egs.(map[string]interface{})))
		}
		opts.ElasticGpuSpecifications = elasticGpuSpecifications
	}

	if v, ok := d.GetOk("iam_instance_profile"); ok {
		iip := v.([]interface{})

		if len(iip) > 0 {
			opts.IamInstanceProfile = readIamInstanceProfileFromConfig(iip[0].(map[string]interface{}))
		}
	}

	if v, ok := d.GetOk("instance_market_options"); ok {
		imo := v.([]interface{})

		if len(imo) > 0 {
			instanceMarketOptions, err := readInstanceMarketOptionsFromConfig(imo[0].(map[string]interface{}))
			if err != nil {
				return nil, err
			}
			opts.InstanceMarketOptions = instanceMarketOptions
		}
	}

	if v, ok := d.GetOk("monitoring"); ok {
		m := v.([]interface{})
		if len(m) > 0 {
			mData := m[0].(map[string]interface{})
			monitoring := &ec2.LaunchTemplatesMonitoringRequest{
				Enabled: aws.Bool(mData["enabled"].(bool)),
			}
			opts.Monitoring = monitoring
		}
	}

	if v, ok := d.GetOk("network_interfaces"); ok {
		var networkInterfaces []*ec2.LaunchTemplateInstanceNetworkInterfaceSpecificationRequest
		niList := v.([]interface{})

		for _, ni := range niList {
			niData := ni.(map[string]interface{})
			networkInterface := readNetworkInterfacesFromConfig(niData)
			networkInterfaces = append(networkInterfaces, networkInterface)
		}
		opts.NetworkInterfaces = networkInterfaces
	}

	if v, ok := d.GetOk("placement"); ok {
		p := v.([]interface{})

		if len(p) > 0 {
			opts.Placement = readPlacementFromConfig(p[0].(map[string]interface{}))
		}
	}

	if v, ok := d.GetOk("tag_specifications"); ok {
		var tagSpecifications []*ec2.LaunchTemplateTagSpecificationRequest
		t := v.([]interface{})

		for _, ts := range t {
			tsData := ts.(map[string]interface{})
			tags := tagsFromMap(tsData["tags"].(map[string]interface{}))
			tagSpecification := &ec2.LaunchTemplateTagSpecificationRequest{
				ResourceType: aws.String(tsData["resource_type"].(string)),
				Tags:         tags,
			}
			tagSpecifications = append(tagSpecifications, tagSpecification)
		}
		opts.TagSpecifications = tagSpecifications
	}

	return opts, nil
}

func readBlockDeviceMappingFromConfig(bdm map[string]interface{}) *ec2.LaunchTemplateBlockDeviceMappingRequest {
	blockDeviceMapping := &ec2.LaunchTemplateBlockDeviceMappingRequest{}

	if v := bdm["device_name"].(string); v != "" {
		blockDeviceMapping.DeviceName = aws.String(v)
	}

	if v := bdm["no_device"].(string); v != "" {
		blockDeviceMapping.NoDevice = aws.String(v)
	}

	if v := bdm["virtual_name"].(string); v != "" {
		blockDeviceMapping.VirtualName = aws.String(v)
	}

	if v := bdm["ebs"]; len(v.([]interface{})) > 0 {
		ebs := v.([]interface{})
		if len(ebs) > 0 {
			ebsData := ebs[0]
			blockDeviceMapping.Ebs = readEbsBlockDeviceFromConfig(ebsData.(map[string]interface{}))
		}
	}

	return blockDeviceMapping
}

func readEbsBlockDeviceFromConfig(ebs map[string]interface{}) *ec2.LaunchTemplateEbsBlockDeviceRequest {
	ebsDevice := &ec2.LaunchTemplateEbsBlockDeviceRequest{}

	if v := ebs["delete_on_termination"]; v != nil {
		ebsDevice.DeleteOnTermination = aws.Bool(v.(bool))
	}

	if v := ebs["encrypted"]; v != nil {
		ebsDevice.Encrypted = aws.Bool(v.(bool))
	}

	if v := ebs["iops"].(int); v > 0 {
		ebsDevice.Iops = aws.Int64(int64(v))
	}

	if v := ebs["kms_key_id"].(string); v != "" {
		ebsDevice.KmsKeyId = aws.String(v)
	}

	if v := ebs["snapshot_id"].(string); v != "" {
		ebsDevice.SnapshotId = aws.String(v)
	}

	if v := ebs["volume_size"]; v != nil {
		ebsDevice.VolumeSize = aws.Int64(int64(v.(int)))
	}

	if v := ebs["volume_type"].(string); v != "" {
		ebsDevice.VolumeType = aws.String(v)
	}

	return ebsDevice
}

func readNetworkInterfacesFromConfig(ni map[string]interface{}) *ec2.LaunchTemplateInstanceNetworkInterfaceSpecificationRequest {
	var ipv4Addresses []*ec2.PrivateIpAddressSpecification
	var ipv6Addresses []*ec2.InstanceIpv6AddressRequest
	var privateIpAddress string
	networkInterface := &ec2.LaunchTemplateInstanceNetworkInterfaceSpecificationRequest{
		AssociatePublicIpAddress: aws.Bool(ni["associate_public_ip_address"].(bool)),
		DeleteOnTermination:      aws.Bool(ni["delete_on_termination"].(bool)),
	}

	if v, ok := ni["description"].(string); ok && v != "" {
		networkInterface.Description = aws.String(v)
	}

	if v, ok := ni["device_index"].(int); ok {
		networkInterface.DeviceIndex = aws.Int64(int64(v))
	}

	if v, ok := ni["network_interface_id"].(string); ok && v != "" {
		networkInterface.NetworkInterfaceId = aws.String(v)
	}

	if v, ok := ni["private_ip_address"].(string); ok && v != "" {
		privateIpAddress = v
		networkInterface.PrivateIpAddress = aws.String(v)
	}

	if v, ok := ni["subnet_id"].(string); ok && v != "" {
		networkInterface.SubnetId = aws.String(v)
	}

	if v := ni["security_groups"].(*schema.Set); v.Len() > 0 {
		for _, v := range v.List() {
			networkInterface.Groups = append(networkInterface.Groups, aws.String(v.(string)))
		}
	}

	ipv6AddressList := ni["ipv6_addresses"].(*schema.Set).List()
	for _, address := range ipv6AddressList {
		ipv6Addresses = append(ipv6Addresses, &ec2.InstanceIpv6AddressRequest{
			Ipv6Address: aws.String(address.(string)),
		})
	}
	networkInterface.Ipv6Addresses = ipv6Addresses

	if v := ni["ipv6_address_count"].(int); v > 0 {
		networkInterface.Ipv6AddressCount = aws.Int64(int64(v))
	}

	ipv4AddressList := ni["ipv4_addresses"].(*schema.Set).List()
	for _, address := range ipv4AddressList {
		privateIp := &ec2.PrivateIpAddressSpecification{
			Primary:          aws.Bool(address.(string) == privateIpAddress),
			PrivateIpAddress: aws.String(address.(string)),
		}
		ipv4Addresses = append(ipv4Addresses, privateIp)
	}
	networkInterface.PrivateIpAddresses = ipv4Addresses

	if v := ni["ipv4_address_count"].(int); v > 0 {
		networkInterface.SecondaryPrivateIpAddressCount = aws.Int64(int64(v))
	}

	return networkInterface
}

func readIamInstanceProfileFromConfig(iip map[string]interface{}) *ec2.LaunchTemplateIamInstanceProfileSpecificationRequest {
	iamInstanceProfile := &ec2.LaunchTemplateIamInstanceProfileSpecificationRequest{}

	if v, ok := iip["arn"].(string); ok && v != "" {
		iamInstanceProfile.Arn = aws.String(v)
	}

	if v, ok := iip["name"].(string); ok && v != "" {
		iamInstanceProfile.Name = aws.String(v)
	}

	return iamInstanceProfile
}

func readCreditSpecificationFromConfig(cs map[string]interface{}) *ec2.CreditSpecificationRequest {
	creditSpecification := &ec2.CreditSpecificationRequest{}

	if v, ok := cs["cpu_credits"].(string); ok && v != "" {
		creditSpecification.CpuCredits = aws.String(v)
	}

	return creditSpecification
}

func readElasticGpuSpecificationsFromConfig(egs map[string]interface{}) *ec2.ElasticGpuSpecification {
	elasticGpuSpecification := &ec2.ElasticGpuSpecification{}

	if v, ok := egs["type"].(string); ok && v != "" {
		elasticGpuSpecification.Type = aws.String(v)
	}

	return elasticGpuSpecification
}

func readInstanceMarketOptionsFromConfig(imo map[string]interface{}) (*ec2.LaunchTemplateInstanceMarketOptionsRequest, error) {
	instanceMarketOptions := &ec2.LaunchTemplateInstanceMarketOptionsRequest{}
	spotOptions := &ec2.LaunchTemplateSpotMarketOptionsRequest{}

	if v, ok := imo["market_type"].(string); ok && v != "" {
		instanceMarketOptions.MarketType = aws.String(v)
	}

	if v, ok := imo["spot_options"]; ok {
		vL := v.([]interface{})
		for _, v := range vL {
			so := v.(map[string]interface{})

			if v, ok := so["block_duration_minutes"].(int); ok && v != 0 {
				spotOptions.BlockDurationMinutes = aws.Int64(int64(v))
			}

			if v, ok := so["instance_interruption_behavior"].(string); ok && v != "" {
				spotOptions.InstanceInterruptionBehavior = aws.String(v)
			}

			if v, ok := so["max_price"].(string); ok && v != "" {
				spotOptions.MaxPrice = aws.String(v)
			}

			if v, ok := so["spot_instance_type"].(string); ok && v != "" {
				spotOptions.SpotInstanceType = aws.String(v)
			}

			if v, ok := so["valid_until"].(string); ok && v != "" {
				t, err := time.Parse(time.RFC3339, v)
				if err != nil {
					return nil, fmt.Errorf("Error Parsing Launch Template Spot Options valid until: %s", err.Error())
				}
				spotOptions.ValidUntil = aws.Time(t)
			}
		}
		instanceMarketOptions.SpotOptions = spotOptions
	}

	return instanceMarketOptions, nil
}

func readPlacementFromConfig(p map[string]interface{}) *ec2.LaunchTemplatePlacementRequest {
	placement := &ec2.LaunchTemplatePlacementRequest{}

	if v, ok := p["affinity"].(string); ok && v != "" {
		placement.Affinity = aws.String(v)
	}

	if v, ok := p["availability_zone"].(string); ok && v != "" {
		placement.AvailabilityZone = aws.String(v)
	}

	if v, ok := p["group_name"].(string); ok && v != "" {
		placement.GroupName = aws.String(v)
	}

	if v, ok := p["host_id"].(string); ok && v != "" {
		placement.HostId = aws.String(v)
	}

	if v, ok := p["spread_domain"].(string); ok && v != "" {
		placement.SpreadDomain = aws.String(v)
	}

	if v, ok := p["tenancy"].(string); ok && v != "" {
		placement.Tenancy = aws.String(v)
	}

	return placement
}
