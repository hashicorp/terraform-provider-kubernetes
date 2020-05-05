package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

const emptyBasePathMappingValue = "(none)"

func resourceAwsApiGatewayBasePathMapping() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsApiGatewayBasePathMappingCreate,
		Read:   resourceAwsApiGatewayBasePathMappingRead,
		Delete: resourceAwsApiGatewayBasePathMappingDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"api_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"base_path": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"stage_name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"domain_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsApiGatewayBasePathMappingCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway
	input := &apigateway.CreateBasePathMappingInput{
		RestApiId:  aws.String(d.Get("api_id").(string)),
		DomainName: aws.String(d.Get("domain_name").(string)),
		BasePath:   aws.String(d.Get("base_path").(string)),
		Stage:      aws.String(d.Get("stage_name").(string)),
	}

	err := resource.Retry(30*time.Second, func() *resource.RetryError {
		_, err := conn.CreateBasePathMapping(input)

		if err != nil {
			if err, ok := err.(awserr.Error); ok && err.Code() != "BadRequestException" {
				return resource.NonRetryableError(err)
			}

			return resource.RetryableError(
				fmt.Errorf("Error creating Gateway base path mapping: %s", err),
			)
		}

		return nil
	})

	if isResourceTimeoutError(err) {
		_, err = conn.CreateBasePathMapping(input)
	}

	if err != nil {
		return fmt.Errorf("Error creating Gateway base path mapping: %s", err)
	}

	id := fmt.Sprintf("%s/%s", d.Get("domain_name").(string), d.Get("base_path").(string))
	d.SetId(id)

	return resourceAwsApiGatewayBasePathMappingRead(d, meta)
}

func resourceAwsApiGatewayBasePathMappingRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	domainName, basePath, err := decodeApiGatewayBasePathMappingId(d.Id())
	if err != nil {
		return err
	}

	mapping, err := conn.GetBasePathMapping(&apigateway.GetBasePathMappingInput{
		DomainName: aws.String(domainName),
		BasePath:   aws.String(basePath),
	})
	if err != nil {
		if err, ok := err.(awserr.Error); ok && err.Code() == "NotFoundException" {
			log.Printf("[WARN] API Gateway Base Path Mapping (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error reading Gateway base path mapping: %s", err)
	}

	mappingBasePath := *mapping.BasePath

	if mappingBasePath == emptyBasePathMappingValue {
		mappingBasePath = ""
	}

	d.Set("base_path", mappingBasePath)
	d.Set("domain_name", domainName)
	d.Set("api_id", mapping.RestApiId)
	d.Set("stage_name", mapping.Stage)

	return nil
}

func resourceAwsApiGatewayBasePathMappingDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	domainName, basePath, err := decodeApiGatewayBasePathMappingId(d.Id())
	if err != nil {
		return err
	}

	_, err = conn.DeleteBasePathMapping(&apigateway.DeleteBasePathMappingInput{
		DomainName: aws.String(domainName),
		BasePath:   aws.String(basePath),
	})

	if err != nil {
		if err, ok := err.(awserr.Error); ok && err.Code() == "NotFoundException" {
			return nil
		}

		return err
	}

	return nil
}

func decodeApiGatewayBasePathMappingId(id string) (string, string, error) {
	idFormatErr := fmt.Errorf("Unexpected format of ID (%q), expected DOMAIN/BASEPATH", id)

	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return "", "", idFormatErr
	}

	domainName := parts[0]
	basePath := parts[1]

	if domainName == "" {
		return "", "", idFormatErr
	}

	if basePath == "" {
		basePath = emptyBasePathMappingValue
	}

	return domainName, basePath, nil
}
