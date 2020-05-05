package aws

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/hashcode"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53resolver"
)

const (
	route53ResolverRuleStatusDeleted = "DELETED"
)

func resourceAwsRoute53ResolverRule() *schema.Resource {
	return &schema.Resource{
		Create:        resourceAwsRoute53ResolverRuleCreate,
		Read:          resourceAwsRoute53ResolverRuleRead,
		Update:        resourceAwsRoute53ResolverRuleUpdate,
		Delete:        resourceAwsRoute53ResolverRuleDelete,
		CustomizeDiff: resourceAwsRoute53ResolverRuleCustomizeDiff,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"domain_name": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				DiffSuppressFunc: suppressRoute53ZoneNameWithTrailingDot,
				ValidateFunc:     validation.StringLenBetween(1, 256),
			},

			"rule_type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					route53resolver.RuleTypeOptionForward,
					route53resolver.RuleTypeOptionSystem,
					route53resolver.RuleTypeOptionRecursive,
				}, false),
			},

			"name": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRoute53ResolverName,
			},

			"resolver_endpoint_id": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"target_ip": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ip": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.SingleIP(),
						},
						"port": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      53,
							ValidateFunc: validation.IntBetween(1, 65535),
						},
					},
				},
				Set: route53ResolverRuleHashTargetIp,
			},

			"tags": tagsSchema(),

			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"owner_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"share_status": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsRoute53ResolverRuleCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).route53resolverconn

	req := &route53resolver.CreateResolverRuleInput{
		CreatorRequestId: aws.String(resource.PrefixedUniqueId("tf-r53-resolver-rule-")),
		DomainName:       aws.String(d.Get("domain_name").(string)),
		RuleType:         aws.String(d.Get("rule_type").(string)),
	}
	if v, ok := d.GetOk("name"); ok {
		req.Name = aws.String(v.(string))
	}
	if v, ok := d.GetOk("resolver_endpoint_id"); ok {
		req.ResolverEndpointId = aws.String(v.(string))
	}
	if v, ok := d.GetOk("target_ip"); ok {
		req.TargetIps = expandRoute53ResolverRuleTargetIps(v.(*schema.Set))
	}
	if v, ok := d.GetOk("tags"); ok && len(v.(map[string]interface{})) > 0 {
		req.Tags = tagsFromMapRoute53Resolver(v.(map[string]interface{}))
	}

	log.Printf("[DEBUG] Creating Route 53 Resolver rule: %s", req)
	resp, err := conn.CreateResolverRule(req)
	if err != nil {
		return fmt.Errorf("error creating Route 53 Resolver rule: %s", err)
	}

	d.SetId(aws.StringValue(resp.ResolverRule.Id))

	err = route53ResolverRuleWaitUntilTargetState(conn, d.Id(), d.Timeout(schema.TimeoutCreate),
		[]string{}, // Should go straight to COMPLETE
		[]string{route53resolver.ResolverRuleStatusComplete})
	if err != nil {
		return err
	}

	return resourceAwsRoute53ResolverRuleRead(d, meta)
}

func resourceAwsRoute53ResolverRuleRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).route53resolverconn

	ruleRaw, state, err := route53ResolverRuleRefresh(conn, d.Id())()
	if err != nil {
		return fmt.Errorf("error getting Route53 Resolver rule (%s): %s", d.Id(), err)
	}
	if state == route53ResolverRuleStatusDeleted {
		log.Printf("[WARN] Route53 Resolver rule (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	rule := ruleRaw.(*route53resolver.ResolverRule)
	d.Set("arn", rule.Arn)
	d.Set("domain_name", rule.DomainName)
	d.Set("name", rule.Name)
	d.Set("owner_id", rule.OwnerId)
	d.Set("resolver_endpoint_id", rule.ResolverEndpointId)
	d.Set("rule_type", rule.RuleType)
	d.Set("share_status", rule.ShareStatus)
	if err := d.Set("target_ip", schema.NewSet(route53ResolverRuleHashTargetIp, flattenRoute53ResolverRuleTargetIps(rule.TargetIps))); err != nil {
		return err
	}

	err = getTagsRoute53Resolver(conn, d)
	if err != nil {
		return fmt.Errorf("Error reading Route 53 Resolver rule tags %s: %s", d.Id(), err)
	}

	return nil
}

func resourceAwsRoute53ResolverRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).route53resolverconn

	d.Partial(true)
	if d.HasChange("name") || d.HasChange("resolver_endpoint_id") || d.HasChange("target_ip") {
		req := &route53resolver.UpdateResolverRuleInput{
			ResolverRuleId: aws.String(d.Id()),
			Config:         &route53resolver.ResolverRuleConfig{},
		}
		if v, ok := d.GetOk("name"); ok {
			req.Config.Name = aws.String(v.(string))
		}
		if v, ok := d.GetOk("resolver_endpoint_id"); ok {
			req.Config.ResolverEndpointId = aws.String(v.(string))
		}
		if v, ok := d.GetOk("target_ip"); ok {
			req.Config.TargetIps = expandRoute53ResolverRuleTargetIps(v.(*schema.Set))
		}

		log.Printf("[DEBUG] Updating Route53 Resolver rule: %#v", req)
		_, err := conn.UpdateResolverRule(req)
		if err != nil {
			return fmt.Errorf("error updating Route 53 Resolver rule (%s): %s", d.Id(), err)
		}

		err = route53ResolverRuleWaitUntilTargetState(conn, d.Id(), d.Timeout(schema.TimeoutUpdate),
			[]string{route53resolver.ResolverRuleStatusUpdating},
			[]string{route53resolver.ResolverRuleStatusComplete})
		if err != nil {
			return err
		}

		d.SetPartial("name")
		d.SetPartial("resolver_endpoint_id")
		d.SetPartial("target_ip")
	}

	if err := setTagsRoute53Resolver(conn, d); err != nil {
		return fmt.Errorf("error setting Route53 Resolver rule (%s) tags: %s", d.Id(), err)
	}
	d.SetPartial("tags")

	d.Partial(false)
	return resourceAwsRoute53ResolverRuleRead(d, meta)
}

func resourceAwsRoute53ResolverRuleDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).route53resolverconn

	log.Printf("[DEBUG] Deleting Route53 Resolver rule: %s", d.Id())
	_, err := conn.DeleteResolverRule(&route53resolver.DeleteResolverRuleInput{
		ResolverRuleId: aws.String(d.Id()),
	})
	if isAWSErr(err, route53resolver.ErrCodeResourceNotFoundException, "") {
		return nil
	}
	if err != nil {
		return fmt.Errorf("error deleting Route 53 Resolver rule (%s): %s", d.Id(), err)
	}

	err = route53ResolverRuleWaitUntilTargetState(conn, d.Id(), d.Timeout(schema.TimeoutDelete),
		[]string{route53resolver.ResolverRuleStatusDeleting},
		[]string{route53ResolverRuleStatusDeleted})
	if err != nil {
		return err
	}

	return nil
}

func resourceAwsRoute53ResolverRuleCustomizeDiff(diff *schema.ResourceDiff, v interface{}) error {
	if diff.Id() != "" {
		if diff.HasChange("resolver_endpoint_id") {
			if _, n := diff.GetChange("resolver_endpoint_id"); n.(string) == "" {
				if err := diff.ForceNew("resolver_endpoint_id"); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func route53ResolverRuleRefresh(conn *route53resolver.Route53Resolver, ruleId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.GetResolverRule(&route53resolver.GetResolverRuleInput{
			ResolverRuleId: aws.String(ruleId),
		})
		if isAWSErr(err, route53resolver.ErrCodeResourceNotFoundException, "") {
			return "", route53ResolverRuleStatusDeleted, nil
		}
		if err != nil {
			return nil, "", err
		}

		if statusMessage := aws.StringValue(resp.ResolverRule.StatusMessage); statusMessage != "" {
			log.Printf("[INFO] Route 53 Resolver rule (%s) status message: %s", ruleId, statusMessage)
		}

		return resp.ResolverRule, aws.StringValue(resp.ResolverRule.Status), nil
	}
}

func route53ResolverRuleWaitUntilTargetState(conn *route53resolver.Route53Resolver, ruleId string, timeout time.Duration, pending, target []string) error {
	stateConf := &resource.StateChangeConf{
		Pending:    pending,
		Target:     target,
		Refresh:    route53ResolverRuleRefresh(conn, ruleId),
		Timeout:    timeout,
		Delay:      10 * time.Second,
		MinTimeout: 5 * time.Second,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("error waiting for Route53 Resolver rule (%s) to reach target state: %s", ruleId, err)
	}

	return nil
}

func route53ResolverRuleHashTargetIp(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-%d-", m["ip"].(string), m["port"].(int)))
	return hashcode.String(buf.String())
}
