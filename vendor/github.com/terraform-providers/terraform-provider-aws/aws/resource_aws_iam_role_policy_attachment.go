package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceAwsIamRolePolicyAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIamRolePolicyAttachmentCreate,
		Read:   resourceAwsIamRolePolicyAttachmentRead,
		Delete: resourceAwsIamRolePolicyAttachmentDelete,
		Importer: &schema.ResourceImporter{
			State: resourceAwsIamRolePolicyAttachmentImport,
		},

		Schema: map[string]*schema.Schema{
			"role": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"policy_arn": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsIamRolePolicyAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn

	role := d.Get("role").(string)
	arn := d.Get("policy_arn").(string)

	err := attachPolicyToRole(conn, role, arn)
	if err != nil {
		return fmt.Errorf("Error attaching policy %s to IAM Role %s: %v", arn, role, err)
	}

	d.SetId(resource.PrefixedUniqueId(fmt.Sprintf("%s-", role)))
	return resourceAwsIamRolePolicyAttachmentRead(d, meta)
}

func resourceAwsIamRolePolicyAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn
	role := d.Get("role").(string)
	policyARN := d.Get("policy_arn").(string)

	hasPolicyAttachment, err := iamRoleHasPolicyARNAttachment(conn, role, policyARN)

	if isAWSErr(err, iam.ErrCodeNoSuchEntityException, "") {
		log.Printf("[WARN] IAM Role (%s) not found, removing from state", role)
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error finding IAM Role (%s) Policy Attachment (%s): %s", role, policyARN, err)
	}

	if !hasPolicyAttachment {
		log.Printf("[WARN] IAM Role (%s) Policy Attachment (%s) not found, removing from state", role, policyARN)
		d.SetId("")
		return nil
	}

	d.Set("policy_arn", policyARN)
	d.Set("role", role)

	return nil
}

func resourceAwsIamRolePolicyAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iamconn
	role := d.Get("role").(string)
	arn := d.Get("policy_arn").(string)

	err := detachPolicyFromRole(conn, role, arn)

	if isAWSErr(err, iam.ErrCodeNoSuchEntityException, "") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("Error removing policy %s from IAM Role %s: %v", arn, role, err)
	}
	return nil
}

func resourceAwsIamRolePolicyAttachmentImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	idParts := strings.SplitN(d.Id(), "/", 2)
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		return nil, fmt.Errorf("unexpected format of ID (%q), expected <role-name>/<policy_arn>", d.Id())
	}

	roleName := idParts[0]
	policyARN := idParts[1]

	d.Set("role", roleName)
	d.Set("policy_arn", policyARN)
	d.SetId(fmt.Sprintf("%s-%s", roleName, policyARN))

	return []*schema.ResourceData{d}, nil
}

func attachPolicyToRole(conn *iam.IAM, role string, arn string) error {
	_, err := conn.AttachRolePolicy(&iam.AttachRolePolicyInput{
		RoleName:  aws.String(role),
		PolicyArn: aws.String(arn),
	})
	return err
}

func detachPolicyFromRole(conn *iam.IAM, role string, arn string) error {
	_, err := conn.DetachRolePolicy(&iam.DetachRolePolicyInput{
		RoleName:  aws.String(role),
		PolicyArn: aws.String(arn),
	})
	return err
}

func iamRoleHasPolicyARNAttachment(conn *iam.IAM, role string, policyARN string) (bool, error) {
	hasPolicyAttachment := false
	input := &iam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(role),
	}

	err := conn.ListAttachedRolePoliciesPages(input, func(page *iam.ListAttachedRolePoliciesOutput, lastPage bool) bool {
		for _, p := range page.AttachedPolicies {
			if aws.StringValue(p.PolicyArn) == policyARN {
				hasPolicyAttachment = true
				return false
			}
		}

		return !lastPage
	})

	return hasPolicyAttachment, err
}
