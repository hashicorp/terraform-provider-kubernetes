package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/xray"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func resourceAwsXraySamplingRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsXraySamplingRuleCreate,
		Read:   resourceAwsXraySamplingRuleRead,
		Update: resourceAwsXraySamplingRuleUpdate,
		Delete: resourceAwsXraySamplingRuleDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"rule_name": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringLenBetween(1, 128),
			},
			"resource_arn": {
				Type:     schema.TypeString,
				Required: true,
			},
			"priority": {
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validation.IntBetween(1, 9999),
			},
			"fixed_rate": {
				Type:     schema.TypeFloat,
				Required: true,
			},
			"reservoir_size": {
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validation.IntAtLeast(0),
			},
			"service_name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringLenBetween(0, 64),
			},
			"service_type": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringLenBetween(0, 64),
			},
			"host": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringLenBetween(0, 64),
			},
			"http_method": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringLenBetween(0, 10),
			},
			"url_path": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringLenBetween(0, 128),
			},
			"version": {
				Type:         schema.TypeInt,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IntAtLeast(1),
			},
			"attributes": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringLenBetween(1, 32),
				},
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsXraySamplingRuleCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).xrayconn
	samplingRule := &xray.SamplingRule{
		RuleName:      aws.String(d.Get("rule_name").(string)),
		ResourceARN:   aws.String(d.Get("resource_arn").(string)),
		Priority:      aws.Int64(int64(d.Get("priority").(int))),
		FixedRate:     aws.Float64(d.Get("fixed_rate").(float64)),
		ReservoirSize: aws.Int64(int64(d.Get("reservoir_size").(int))),
		ServiceName:   aws.String(d.Get("service_name").(string)),
		ServiceType:   aws.String(d.Get("service_type").(string)),
		Host:          aws.String(d.Get("host").(string)),
		HTTPMethod:    aws.String(d.Get("http_method").(string)),
		URLPath:       aws.String(d.Get("url_path").(string)),
		Version:       aws.Int64(int64(d.Get("version").(int))),
	}

	if v, ok := d.GetOk("attributes"); ok {
		samplingRule.Attributes = stringMapToPointers(v.(map[string]interface{}))
	}

	params := &xray.CreateSamplingRuleInput{
		SamplingRule: samplingRule,
	}

	out, err := conn.CreateSamplingRule(params)
	if err != nil {
		return fmt.Errorf("error creating XRay Sampling Rule: %s", err)
	}

	d.SetId(*out.SamplingRuleRecord.SamplingRule.RuleName)

	return resourceAwsXraySamplingRuleRead(d, meta)
}

func resourceAwsXraySamplingRuleRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).xrayconn

	samplingRule, err := getXraySamplingRule(conn, d.Id())

	if err != nil {
		return fmt.Errorf("error reading XRay Sampling Rule (%s): %s", d.Id(), err)
	}

	if samplingRule == nil {
		log.Printf("[WARN] XRay Sampling Rule (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("rule_name", samplingRule.RuleName)
	d.Set("resource_arn", samplingRule.ResourceARN)
	d.Set("priority", samplingRule.Priority)
	d.Set("fixed_rate", samplingRule.FixedRate)
	d.Set("reservoir_size", samplingRule.ReservoirSize)
	d.Set("service_name", samplingRule.ServiceName)
	d.Set("service_type", samplingRule.ServiceType)
	d.Set("host", samplingRule.Host)
	d.Set("http_method", samplingRule.HTTPMethod)
	d.Set("url_path", samplingRule.URLPath)
	d.Set("version", samplingRule.Version)
	d.Set("attributes", aws.StringValueMap(samplingRule.Attributes))
	d.Set("arn", samplingRule.RuleARN)

	return nil
}

func resourceAwsXraySamplingRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).xrayconn
	samplingRuleUpdate := &xray.SamplingRuleUpdate{
		RuleName:      aws.String(d.Id()),
		Priority:      aws.Int64(int64(d.Get("priority").(int))),
		FixedRate:     aws.Float64(d.Get("fixed_rate").(float64)),
		ReservoirSize: aws.Int64(int64(d.Get("reservoir_size").(int))),
		ServiceName:   aws.String(d.Get("service_name").(string)),
		ServiceType:   aws.String(d.Get("service_type").(string)),
		Host:          aws.String(d.Get("host").(string)),
		HTTPMethod:    aws.String(d.Get("http_method").(string)),
		URLPath:       aws.String(d.Get("url_path").(string)),
	}

	if d.HasChange("attributes") {
		attributes := map[string]*string{}
		if v, ok := d.GetOk("attributes"); ok {
			if m, ok := v.(map[string]interface{}); ok {
				attributes = stringMapToPointers(m)
			}
		}
		samplingRuleUpdate.Attributes = attributes
	}

	params := &xray.UpdateSamplingRuleInput{
		SamplingRuleUpdate: samplingRuleUpdate,
	}

	_, err := conn.UpdateSamplingRule(params)
	if err != nil {
		return fmt.Errorf("error updating XRay Sampling Rule (%s): %s", d.Id(), err)
	}

	return resourceAwsXraySamplingRuleRead(d, meta)
}

func resourceAwsXraySamplingRuleDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).xrayconn

	log.Printf("[INFO] Deleting XRay Sampling Rule: %s", d.Id())

	params := &xray.DeleteSamplingRuleInput{
		RuleName: aws.String(d.Id()),
	}
	_, err := conn.DeleteSamplingRule(params)
	if err != nil {
		return fmt.Errorf("error deleting XRay Sampling Rule: %s", d.Id())
	}

	return nil
}

func getXraySamplingRule(conn *xray.XRay, ruleName string) (*xray.SamplingRule, error) {
	params := &xray.GetSamplingRulesInput{}
	for {
		out, err := conn.GetSamplingRules(params)
		if err != nil {
			return nil, err
		}
		for _, samplingRuleRecord := range out.SamplingRuleRecords {
			samplingRule := samplingRuleRecord.SamplingRule
			if aws.StringValue(samplingRule.RuleName) == ruleName {
				return samplingRule, nil
			}
		}
		if aws.StringValue(out.NextToken) == "" {
			break
		}
		params.NextToken = out.NextToken
	}
	return nil, nil
}
