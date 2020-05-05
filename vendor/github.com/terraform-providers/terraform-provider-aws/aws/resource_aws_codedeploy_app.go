package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/codedeploy"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func resourceAwsCodeDeployApp() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCodeDeployAppCreate,
		Read:   resourceAwsCodeDeployAppRead,
		Update: resourceAwsCodeDeployUpdate,
		Delete: resourceAwsCodeDeployAppDelete,
		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				idParts := strings.Split(d.Id(), ":")

				if len(idParts) == 2 {
					return []*schema.ResourceData{d}, nil
				}

				applicationName := d.Id()
				conn := meta.(*AWSClient).codedeployconn

				input := &codedeploy.GetApplicationInput{
					ApplicationName: aws.String(applicationName),
				}

				log.Printf("[DEBUG] Reading CodeDeploy Application: %s", input)
				output, err := conn.GetApplication(input)

				if err != nil {
					return []*schema.ResourceData{}, err
				}

				if output == nil || output.Application == nil {
					return []*schema.ResourceData{}, fmt.Errorf("error reading CodeDeploy Application (%s): empty response", applicationName)
				}

				d.SetId(fmt.Sprintf("%s:%s", aws.StringValue(output.Application.ApplicationId), applicationName))
				d.Set("name", applicationName)

				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"compute_platform": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					codedeploy.ComputePlatformEcs,
					codedeploy.ComputePlatformLambda,
					codedeploy.ComputePlatformServer,
				}, false),
				Default: codedeploy.ComputePlatformServer,
			},

			// The unique ID is set by AWS on create.
			"unique_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceAwsCodeDeployAppCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codedeployconn

	application := d.Get("name").(string)
	computePlatform := d.Get("compute_platform").(string)
	log.Printf("[DEBUG] Creating CodeDeploy application %s", application)

	resp, err := conn.CreateApplication(&codedeploy.CreateApplicationInput{
		ApplicationName: aws.String(application),
		ComputePlatform: aws.String(computePlatform),
	})
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] CodeDeploy application %s created", *resp.ApplicationId)

	// Despite giving the application a unique ID, AWS doesn't actually use
	// it in API calls. Use it and the app name to identify the resource in
	// the state file. This allows us to reliably detect both when the TF
	// config file changes and when the user deletes the app without removing
	// it first from the TF config.
	d.SetId(fmt.Sprintf("%s:%s", *resp.ApplicationId, application))

	return resourceAwsCodeDeployAppRead(d, meta)
}

func resourceAwsCodeDeployAppRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codedeployconn

	application := resourceAwsCodeDeployAppParseId(d.Id())
	log.Printf("[DEBUG] Reading CodeDeploy application %s", application)
	resp, err := conn.GetApplication(&codedeploy.GetApplicationInput{
		ApplicationName: aws.String(application),
	})
	if err != nil {
		if codedeployerr, ok := err.(awserr.Error); ok && codedeployerr.Code() == "ApplicationDoesNotExistException" {
			d.SetId("")
			return nil
		} else {
			log.Printf("[ERROR] Error finding CodeDeploy application: %s", err)
			return err
		}
	}

	d.Set("compute_platform", resp.Application.ComputePlatform)
	d.Set("name", resp.Application.ApplicationName)

	return nil
}

func resourceAwsCodeDeployUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codedeployconn

	o, n := d.GetChange("name")

	_, err := conn.UpdateApplication(&codedeploy.UpdateApplicationInput{
		ApplicationName:    aws.String(o.(string)),
		NewApplicationName: aws.String(n.(string)),
	})
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] CodeDeploy application %s updated", n)

	d.Set("name", n)

	return nil
}

func resourceAwsCodeDeployAppDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).codedeployconn

	_, err := conn.DeleteApplication(&codedeploy.DeleteApplicationInput{
		ApplicationName: aws.String(d.Get("name").(string)),
	})
	if err != nil {
		if cderr, ok := err.(awserr.Error); ok && cderr.Code() == "InvalidApplicationNameException" {
			return nil
		} else {
			log.Printf("[ERROR] Error deleting CodeDeploy application: %s", err)
			return err
		}
	}

	return nil
}

func resourceAwsCodeDeployAppParseId(id string) string {
	parts := strings.SplitN(id, ":", 2)
	// We currently omit the application ID as it is not currently used anywhere
	return parts[1]
}
