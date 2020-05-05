package aws

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/efs"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func resourceAwsEfsFileSystem() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEfsFileSystemCreate,
		Read:   resourceAwsEfsFileSystemRead,
		Update: resourceAwsEfsFileSystemUpdate,
		Delete: resourceAwsEfsFileSystemDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"creation_token": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringLenBetween(0, 64),
			},

			"reference_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				Removed:  "Use `creation_token` argument instead",
			},

			"performance_mode": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					efs.PerformanceModeGeneralPurpose,
					efs.PerformanceModeMaxIo,
				}, false),
			},

			"encrypted": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"kms_key_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},

			"dns_name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"provisioned_throughput_in_mibps": {
				Type:     schema.TypeFloat,
				Optional: true,
			},

			"tags": tagsSchema(),

			"throughput_mode": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  efs.ThroughputModeBursting,
				ValidateFunc: validation.StringInSlice([]string{
					efs.ThroughputModeBursting,
					efs.ThroughputModeProvisioned,
				}, false),
			},

			"lifecycle_policy": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"transition_to_ia": {
							Type:     schema.TypeString,
							Optional: true,
							ValidateFunc: validation.StringInSlice([]string{
								efs.TransitionToIARulesAfter14Days,
								efs.TransitionToIARulesAfter30Days,
								efs.TransitionToIARulesAfter60Days,
								efs.TransitionToIARulesAfter90Days,
							}, false),
						},
					},
				},
			},
		},
	}
}

func resourceAwsEfsFileSystemCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).efsconn

	creationToken := ""
	if v, ok := d.GetOk("creation_token"); ok {
		creationToken = v.(string)
	} else {
		creationToken = resource.UniqueId()
	}
	throughputMode := d.Get("throughput_mode").(string)

	createOpts := &efs.CreateFileSystemInput{
		CreationToken:  aws.String(creationToken),
		ThroughputMode: aws.String(throughputMode),
		Tags:           tagsFromMapEFS(d.Get("tags").(map[string]interface{})),
	}

	if v, ok := d.GetOk("performance_mode"); ok {
		createOpts.PerformanceMode = aws.String(v.(string))
	}

	if throughputMode == efs.ThroughputModeProvisioned {
		createOpts.ProvisionedThroughputInMibps = aws.Float64(d.Get("provisioned_throughput_in_mibps").(float64))
	}

	encrypted, hasEncrypted := d.GetOk("encrypted")
	kmsKeyId, hasKmsKeyId := d.GetOk("kms_key_id")

	if hasEncrypted {
		createOpts.Encrypted = aws.Bool(encrypted.(bool))
	}

	if hasKmsKeyId {
		createOpts.KmsKeyId = aws.String(kmsKeyId.(string))
	}

	if encrypted == false && hasKmsKeyId {
		return errors.New("encrypted must be set to true when kms_key_id is specified")
	}

	log.Printf("[DEBUG] EFS file system create options: %#v", *createOpts)
	fs, err := conn.CreateFileSystem(createOpts)
	if err != nil {
		return fmt.Errorf("Error creating EFS file system: %s", err)
	}

	d.SetId(*fs.FileSystemId)
	log.Printf("[INFO] EFS file system ID: %s", d.Id())

	stateConf := &resource.StateChangeConf{
		Pending:    []string{efs.LifeCycleStateCreating},
		Target:     []string{efs.LifeCycleStateAvailable},
		Refresh:    resourceEfsFileSystemCreateUpdateRefreshFunc(d.Id(), conn),
		Timeout:    10 * time.Minute,
		Delay:      2 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for EFS file system (%q) to create: %s", d.Id(), err)
	}
	log.Printf("[DEBUG] EFS file system %q created.", d.Id())

	_, hasLifecyclePolicy := d.GetOk("lifecycle_policy")
	if hasLifecyclePolicy {
		_, err := conn.PutLifecycleConfiguration(&efs.PutLifecycleConfigurationInput{
			FileSystemId:      aws.String(d.Id()),
			LifecyclePolicies: resourceAwsEfsFileSystemLifecyclePolicy(d.Get("lifecycle_policy").([]interface{})),
		})
		if err != nil {
			return fmt.Errorf("Error creating lifecycle policy for EFS file system %q: %s",
				d.Id(), err.Error())
		}
	}

	return resourceAwsEfsFileSystemRead(d, meta)
}

func resourceAwsEfsFileSystemUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).efsconn

	if d.HasChange("provisioned_throughput_in_mibps") || d.HasChange("throughput_mode") {
		throughputMode := d.Get("throughput_mode").(string)

		input := &efs.UpdateFileSystemInput{
			FileSystemId:   aws.String(d.Id()),
			ThroughputMode: aws.String(throughputMode),
		}

		if throughputMode == efs.ThroughputModeProvisioned {
			input.ProvisionedThroughputInMibps = aws.Float64(d.Get("provisioned_throughput_in_mibps").(float64))
		}

		_, err := conn.UpdateFileSystem(input)
		if err != nil {
			return fmt.Errorf("error updating EFS File System %q: %s", d.Id(), err)
		}

		stateConf := &resource.StateChangeConf{
			Pending:    []string{efs.LifeCycleStateUpdating},
			Target:     []string{efs.LifeCycleStateAvailable},
			Refresh:    resourceEfsFileSystemCreateUpdateRefreshFunc(d.Id(), conn),
			Timeout:    10 * time.Minute,
			Delay:      2 * time.Second,
			MinTimeout: 3 * time.Second,
		}

		_, err = stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf("error waiting for EFS file system (%q) to update: %s", d.Id(), err)
		}
	}

	if d.HasChange("lifecycle_policy") {
		_, err := conn.PutLifecycleConfiguration(&efs.PutLifecycleConfigurationInput{
			FileSystemId:      aws.String(d.Id()),
			LifecyclePolicies: resourceAwsEfsFileSystemLifecyclePolicy(d.Get("lifecycle_policy").([]interface{})),
		})
		if err != nil {
			return fmt.Errorf("Error updating lifecycle policy for EFS file system %q: %s",
				d.Id(), err.Error())
		}
	}

	if d.HasChange("tags") {
		err := setTagsEFS(conn, d)
		if err != nil {
			return fmt.Errorf("Error setting EC2 tags for EFS file system (%q): %s",
				d.Id(), err.Error())
		}
	}

	return resourceAwsEfsFileSystemRead(d, meta)
}

func resourceAwsEfsFileSystemRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).efsconn

	resp, err := conn.DescribeFileSystems(&efs.DescribeFileSystemsInput{
		FileSystemId: aws.String(d.Id()),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "FileSystemNotFound" {
			log.Printf("[WARN] EFS file system (%s) could not be found.", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	if hasEmptyFileSystems(resp) {
		return fmt.Errorf("EFS file system %q could not be found.", d.Id())
	}

	tags := make([]*efs.Tag, 0)
	var marker string
	for {
		params := &efs.DescribeTagsInput{
			FileSystemId: aws.String(d.Id()),
		}
		if marker != "" {
			params.Marker = aws.String(marker)
		}

		tagsResp, err := conn.DescribeTags(params)
		if err != nil {
			return fmt.Errorf("Error retrieving EC2 tags for EFS file system (%q): %s",
				d.Id(), err.Error())
		}

		tags = append(tags, tagsResp.Tags...)

		if tagsResp.NextMarker != nil {
			marker = *tagsResp.NextMarker
		} else {
			break
		}
	}

	err = d.Set("tags", tagsToMapEFS(tags))
	if err != nil {
		return err
	}

	var fs *efs.FileSystemDescription
	for _, f := range resp.FileSystems {
		if d.Id() == *f.FileSystemId {
			fs = f
			break
		}
	}
	if fs == nil {
		log.Printf("[WARN] EFS (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	fsARN := arn.ARN{
		AccountID: meta.(*AWSClient).accountid,
		Partition: meta.(*AWSClient).partition,
		Region:    meta.(*AWSClient).region,
		Resource:  fmt.Sprintf("file-system/%s", aws.StringValue(fs.FileSystemId)),
		Service:   "elasticfilesystem",
	}.String()

	d.Set("arn", fsARN)
	d.Set("creation_token", fs.CreationToken)
	d.Set("encrypted", fs.Encrypted)
	d.Set("kms_key_id", fs.KmsKeyId)
	d.Set("performance_mode", fs.PerformanceMode)
	d.Set("provisioned_throughput_in_mibps", fs.ProvisionedThroughputInMibps)
	d.Set("throughput_mode", fs.ThroughputMode)

	region := meta.(*AWSClient).region
	if err := d.Set("dns_name", resourceAwsEfsDnsName(aws.StringValue(fs.FileSystemId), region)); err != nil {
		return fmt.Errorf("error setting dns_name: %s", err)
	}

	res, err := conn.DescribeLifecycleConfiguration(&efs.DescribeLifecycleConfigurationInput{
		FileSystemId: fs.FileSystemId,
	})
	if err != nil {
		return fmt.Errorf("Error describing lifecycle configuration for EFS file system (%s): %s",
			aws.StringValue(fs.FileSystemId), err)
	}
	if err := resourceAwsEfsFileSystemSetLifecyclePolicy(d, res.LifecyclePolicies); err != nil {
		return err
	}

	return nil
}

func resourceAwsEfsFileSystemDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).efsconn

	log.Printf("[DEBUG] Deleting EFS file system: %s", d.Id())
	_, err := conn.DeleteFileSystem(&efs.DeleteFileSystemInput{
		FileSystemId: aws.String(d.Id()),
	})
	if err != nil {
		return fmt.Errorf("Error delete file system: %s with err %s", d.Id(), err.Error())
	}
	stateConf := &resource.StateChangeConf{
		Pending: []string{"available", "deleting"},
		Target:  []string{},
		Refresh: func() (interface{}, string, error) {
			resp, err := conn.DescribeFileSystems(&efs.DescribeFileSystemsInput{
				FileSystemId: aws.String(d.Id()),
			})
			if err != nil {
				efsErr, ok := err.(awserr.Error)
				if ok && efsErr.Code() == "FileSystemNotFound" {
					return nil, "", nil
				}
				return nil, "error", err
			}

			if hasEmptyFileSystems(resp) {
				return nil, "", nil
			}

			fs := resp.FileSystems[0]
			log.Printf("[DEBUG] current status of %q: %q", *fs.FileSystemId, *fs.LifeCycleState)
			return fs, *fs.LifeCycleState, nil
		},
		Timeout:    10 * time.Minute,
		Delay:      2 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for EFS file system (%q) to delete: %s",
			d.Id(), err.Error())
	}

	log.Printf("[DEBUG] EFS file system %q deleted.", d.Id())

	return nil
}

func hasEmptyFileSystems(fs *efs.DescribeFileSystemsOutput) bool {
	if fs != nil && len(fs.FileSystems) > 0 {
		return false
	}
	return true
}

func resourceAwsEfsDnsName(fileSystemId, region string) string {
	return fmt.Sprintf("%s.efs.%s.amazonaws.com", fileSystemId, region)
}

func resourceEfsFileSystemCreateUpdateRefreshFunc(id string, conn *efs.EFS) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeFileSystems(&efs.DescribeFileSystemsInput{
			FileSystemId: aws.String(id),
		})
		if err != nil {
			return nil, "error", err
		}

		if hasEmptyFileSystems(resp) {
			return nil, "not-found", fmt.Errorf("EFS file system %q could not be found.", id)
		}

		fs := resp.FileSystems[0]
		state := aws.StringValue(fs.LifeCycleState)
		log.Printf("[DEBUG] current status of %q: %q", id, state)
		return fs, state, nil
	}
}

func resourceAwsEfsFileSystemSetLifecyclePolicy(d *schema.ResourceData, lp []*efs.LifecyclePolicy) error {
	log.Printf("[DEBUG] lifecycle pols: %s %d", lp, len(lp))
	if len(lp) == 0 {
		d.Set("lifecycle_policy", nil)
		return nil
	}
	newLP := make([]*map[string]interface{}, len(lp))

	for i := 0; i < len(lp); i++ {
		config := lp[i]
		data := make(map[string]interface{})
		newLP[i] = &data
		if config.TransitionToIA != nil {
			data["transition_to_ia"] = *config.TransitionToIA
		}
		log.Printf("[DEBUG] lp: %s", data)
	}

	if err := d.Set("lifecycle_policy", newLP); err != nil {
		return fmt.Errorf("error setting lifecycle_policy: %s", err)
	}
	return nil
}

func resourceAwsEfsFileSystemLifecyclePolicy(lcPol []interface{}) []*efs.LifecyclePolicy {
	result := make([]*efs.LifecyclePolicy, len(lcPol))

	for i := 0; i < len(lcPol); i++ {
		lp := lcPol[i].(map[string]interface{})
		result[i] = &efs.LifecyclePolicy{TransitionToIA: aws.String(lp["transition_to_ia"].(string))}
	}
	return result
}
