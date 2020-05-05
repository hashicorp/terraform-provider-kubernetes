package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceAwsEbsSnapshot() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEbsSnapshotCreate,
		Read:   resourceAwsEbsSnapshotRead,
		Delete: resourceAwsEbsSnapshotDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"volume_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"owner_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"owner_alias": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"encrypted": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"volume_size": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"kms_key_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"data_encryption_key_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsEbsSnapshotCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	request := &ec2.CreateSnapshotInput{
		VolumeId: aws.String(d.Get("volume_id").(string)),
	}
	if v, ok := d.GetOk("description"); ok {
		request.Description = aws.String(v.(string))
	}

	var res *ec2.Snapshot
	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		var err error
		res, err = conn.CreateSnapshot(request)

		if isAWSErr(err, "SnapshotCreationPerVolumeRateExceeded", "The maximum per volume CreateSnapshot request rate has been exceeded") {
			return resource.RetryableError(err)
		}

		if err != nil {
			return resource.NonRetryableError(err)
		}

		return nil
	})
	if isResourceTimeoutError(err) {
		res, err = conn.CreateSnapshot(request)
	}
	if err != nil {
		return fmt.Errorf("error creating EC2 EBS Snapshot: %s", err)
	}

	d.SetId(*res.SnapshotId)

	err = resourceAwsEbsSnapshotWaitForAvailable(d, conn)
	if err != nil {
		return err
	}

	if err := setTags(conn, d); err != nil {
		log.Printf("[WARN] error setting tags: %s", err)
	}

	return resourceAwsEbsSnapshotRead(d, meta)
}

func resourceAwsEbsSnapshotRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	req := &ec2.DescribeSnapshotsInput{
		SnapshotIds: []*string{aws.String(d.Id())},
	}
	res, err := conn.DescribeSnapshots(req)
	if err != nil {
		if isAWSErr(err, "InvalidSnapshot.NotFound", "") {
			log.Printf("[WARN] Snapshot %q Not found - removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	if len(res.Snapshots) == 0 {
		d.SetId("")
		return nil
	}

	snapshot := res.Snapshots[0]

	d.Set("description", snapshot.Description)
	d.Set("owner_id", snapshot.OwnerId)
	d.Set("encrypted", snapshot.Encrypted)
	d.Set("owner_alias", snapshot.OwnerAlias)
	d.Set("volume_id", snapshot.VolumeId)
	d.Set("data_encryption_key_id", snapshot.DataEncryptionKeyId)
	d.Set("kms_key_id", snapshot.KmsKeyId)
	d.Set("volume_size", snapshot.VolumeSize)

	if err := d.Set("tags", tagsToMap(snapshot.Tags)); err != nil {
		log.Printf("[WARN] error saving tags to state: %s", err)
	}

	return nil
}

func resourceAwsEbsSnapshotDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	input := &ec2.DeleteSnapshotInput{
		SnapshotId: aws.String(d.Id()),
	}
	err := resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		_, err := conn.DeleteSnapshot(input)
		if err == nil {
			return nil
		}
		if isAWSErr(err, "SnapshotInUse", "") {
			return resource.RetryableError(fmt.Errorf("EBS SnapshotInUse - trying again while it detaches"))
		}
		return resource.NonRetryableError(err)
	})
	if isResourceTimeoutError(err) {
		_, err = conn.DeleteSnapshot(input)
	}
	if err != nil {
		return fmt.Errorf("Error deleting EBS snapshot: %s", err)
	}
	return nil
}

func resourceAwsEbsSnapshotWaitForAvailable(d *schema.ResourceData, conn *ec2.EC2) error {
	log.Printf("Waiting for Snapshot %s to become available...", d.Id())
	input := &ec2.DescribeSnapshotsInput{
		SnapshotIds: []*string{aws.String(d.Id())},
	}
	err := resource.Retry(d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		err := conn.WaitUntilSnapshotCompleted(input)
		if err == nil {
			return nil
		}
		if isAWSErr(err, "ResourceNotReady", "") {
			return resource.RetryableError(fmt.Errorf("EBS CreatingSnapshot - waiting for snapshot to become available"))
		}
		return resource.NonRetryableError(err)
	})
	if isResourceTimeoutError(err) {
		err = conn.WaitUntilSnapshotCompleted(input)
	}
	if err != nil {
		return fmt.Errorf("Error waiting for EBS snapshot to complete: %s", err)
	}
	return nil
}
