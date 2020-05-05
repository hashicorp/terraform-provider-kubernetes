package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceAwsIamUserPolicyAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIamUserPolicyAttachmentCreate,
		Read:   resourceAwsIamUserPolicyAttachmentRead,
		Delete: resourceAwsIamUserPolicyAttachmentDelete,
		Importer: &schema.ResourceImporter{
			State: resourceAwsIamUserPolicyAttachmentImport,
		},

		Schema: map[string]*schema.Schema{
			"user": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"policy_arn": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsIamUserPolicyAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn

	user := d.Get("user").(string)
	arn := d.Get("policy_arn").(string)

	err := attachPolicyToUser(conn, user, arn)
	if err != nil {
		return fmt.Errorf("Error attaching policy %s to IAM User %s: %v", arn, user, err)
	}

	d.SetId(resource.PrefixedUniqueId(fmt.Sprintf("%s-", user)))
	return resourceAwsIamUserPolicyAttachmentRead(d, meta)
}

func resourceAwsIamUserPolicyAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn
	user := d.Get("user").(string)
	arn := d.Get("policy_arn").(string)

	_, err := conn.GetUser(&iam.GetUserInput{
		UserName: aws.String(user),
	})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "NoSuchEntity" {
				log.Printf("[WARN] No such entity found for Policy Attachment (%s)", user)
				d.SetId("")
				return nil
			}
		}
		return err
	}

	attachedPolicies, err := conn.ListAttachedUserPolicies(&iam.ListAttachedUserPoliciesInput{
		UserName: aws.String(user),
	})
	if err != nil {
		return err
	}

	var policy string
	for _, p := range attachedPolicies.AttachedPolicies {
		if *p.PolicyArn == arn {
			policy = *p.PolicyArn
		}
	}

	if policy == "" {
		log.Printf("[WARN] No such User found for Policy Attachment (%s)", user)
		d.SetId("")
	}
	return nil
}

func resourceAwsIamUserPolicyAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn
	user := d.Get("user").(string)
	arn := d.Get("policy_arn").(string)

	err := detachPolicyFromUser(conn, user, arn)
	if err != nil {
		return fmt.Errorf("Error removing policy %s from IAM User %s: %v", arn, user, err)
	}
	return nil
}

func resourceAwsIamUserPolicyAttachmentImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	idParts := strings.SplitN(d.Id(), "/", 2)
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		return nil, fmt.Errorf("unexpected format of ID (%q), expected <user-name>/<policy_arn>", d.Id())
	}

	userName := idParts[0]
	policyARN := idParts[1]

	d.Set("user", userName)
	d.Set("policy_arn", policyARN)
	d.SetId(fmt.Sprintf("%s-%s", userName, policyARN))

	return []*schema.ResourceData{d}, nil
}

func attachPolicyToUser(conn *iam.IAM, user string, arn string) error {
	_, err := conn.AttachUserPolicy(&iam.AttachUserPolicyInput{
		UserName:  aws.String(user),
		PolicyArn: aws.String(arn),
	})
	return err
}

func detachPolicyFromUser(conn *iam.IAM, user string, arn string) error {
	_, err := conn.DetachUserPolicy(&iam.DetachUserPolicyInput{
		UserName:  aws.String(user),
		PolicyArn: aws.String(arn),
	})
	return err
}
