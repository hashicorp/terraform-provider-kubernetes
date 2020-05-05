package aws

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/hashicorp/terraform-plugin-sdk/helper/hashcode"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func resourceAwsAppmeshVirtualRouter() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAppmeshVirtualRouterCreate,
		Read:   resourceAwsAppmeshVirtualRouterRead,
		Update: resourceAwsAppmeshVirtualRouterUpdate,
		Delete: resourceAwsAppmeshVirtualRouterDelete,
		Importer: &schema.ResourceImporter{
			State: resourceAwsAppmeshVirtualRouterImport,
		},

		SchemaVersion: 1,
		MigrateState:  resourceAwsAppmeshVirtualRouterMigrateState,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringLenBetween(1, 255),
			},

			"mesh_name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringLenBetween(1, 255),
			},

			"spec": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"service_names": {
							Type:     schema.TypeSet,
							Removed:  "Use `aws_appmesh_virtual_service` resources instead",
							Optional: true,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},

						"listener": {
							Type:     schema.TypeSet,
							Required: true,
							MinItems: 1,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"port_mapping": {
										Type:     schema.TypeList,
										Required: true,
										MinItems: 1,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"port": {
													Type:         schema.TypeInt,
													Required:     true,
													ValidateFunc: validation.IntBetween(1, 65535),
												},

												"protocol": {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.StringInSlice([]string{
														appmesh.PortProtocolHttp,
														appmesh.PortProtocolTcp,
													}, false),
												},
											},
										},
									},
								},
							},
							Set: appmeshVirtualRouterListenerHash,
						},
					},
				},
			},

			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"created_date": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"last_updated_date": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsAppmeshVirtualRouterCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appmeshconn

	req := &appmesh.CreateVirtualRouterInput{
		MeshName:          aws.String(d.Get("mesh_name").(string)),
		VirtualRouterName: aws.String(d.Get("name").(string)),
		Spec:              expandAppmeshVirtualRouterSpec(d.Get("spec").([]interface{})),
		Tags:              tagsFromMapAppmesh(d.Get("tags").(map[string]interface{})),
	}

	log.Printf("[DEBUG] Creating App Mesh virtual router: %#v", req)
	resp, err := conn.CreateVirtualRouter(req)
	if err != nil {
		return fmt.Errorf("error creating App Mesh virtual router: %s", err)
	}

	d.SetId(aws.StringValue(resp.VirtualRouter.Metadata.Uid))

	return resourceAwsAppmeshVirtualRouterRead(d, meta)
}

func resourceAwsAppmeshVirtualRouterRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appmeshconn

	resp, err := conn.DescribeVirtualRouter(&appmesh.DescribeVirtualRouterInput{
		MeshName:          aws.String(d.Get("mesh_name").(string)),
		VirtualRouterName: aws.String(d.Get("name").(string)),
	})
	if isAWSErr(err, appmesh.ErrCodeNotFoundException, "") {
		log.Printf("[WARN] App Mesh virtual router (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("error reading App Mesh virtual router: %s", err)
	}
	if aws.StringValue(resp.VirtualRouter.Status.Status) == appmesh.VirtualRouterStatusCodeDeleted {
		log.Printf("[WARN] App Mesh virtual router (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("name", resp.VirtualRouter.VirtualRouterName)
	d.Set("mesh_name", resp.VirtualRouter.MeshName)
	d.Set("arn", resp.VirtualRouter.Metadata.Arn)
	d.Set("created_date", resp.VirtualRouter.Metadata.CreatedAt.Format(time.RFC3339))
	d.Set("last_updated_date", resp.VirtualRouter.Metadata.LastUpdatedAt.Format(time.RFC3339))
	err = d.Set("spec", flattenAppmeshVirtualRouterSpec(resp.VirtualRouter.Spec))
	if err != nil {
		return fmt.Errorf("error setting spec: %s", err)
	}

	err = saveTagsAppmesh(conn, d, aws.StringValue(resp.VirtualRouter.Metadata.Arn))
	if isAWSErr(err, appmesh.ErrCodeNotFoundException, "") {
		log.Printf("[WARN] App Mesh virtual router (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("error saving tags: %s", err)
	}

	return nil
}

func resourceAwsAppmeshVirtualRouterUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appmeshconn

	if d.HasChange("spec") {
		_, v := d.GetChange("spec")
		req := &appmesh.UpdateVirtualRouterInput{
			MeshName:          aws.String(d.Get("mesh_name").(string)),
			VirtualRouterName: aws.String(d.Get("name").(string)),
			Spec:              expandAppmeshVirtualRouterSpec(v.([]interface{})),
		}

		log.Printf("[DEBUG] Updating App Mesh virtual router: %#v", req)
		_, err := conn.UpdateVirtualRouter(req)
		if err != nil {
			return fmt.Errorf("error updating App Mesh virtual router: %s", err)
		}
	}

	err := setTagsAppmesh(conn, d, d.Get("arn").(string))
	if isAWSErr(err, appmesh.ErrCodeNotFoundException, "") {
		log.Printf("[WARN] App Mesh virtual router (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("error setting tags: %s", err)
	}

	return resourceAwsAppmeshVirtualRouterRead(d, meta)
}

func resourceAwsAppmeshVirtualRouterDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appmeshconn

	log.Printf("[DEBUG] Deleting App Mesh virtual router: %s", d.Id())
	_, err := conn.DeleteVirtualRouter(&appmesh.DeleteVirtualRouterInput{
		MeshName:          aws.String(d.Get("mesh_name").(string)),
		VirtualRouterName: aws.String(d.Get("name").(string)),
	})
	if isAWSErr(err, appmesh.ErrCodeNotFoundException, "") {
		return nil
	}
	if err != nil {
		return fmt.Errorf("error deleting App Mesh virtual router: %s", err)
	}

	return nil
}

func resourceAwsAppmeshVirtualRouterImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	parts := strings.Split(d.Id(), "/")
	if len(parts) != 2 {
		return []*schema.ResourceData{}, fmt.Errorf("Wrong format of resource: %s. Please follow 'mesh-name/virtual-router-name'", d.Id())
	}

	mesh := parts[0]
	name := parts[1]
	log.Printf("[DEBUG] Importing App Mesh virtual router %s from mesh %s", name, mesh)

	conn := meta.(*AWSClient).appmeshconn

	resp, err := conn.DescribeVirtualRouter(&appmesh.DescribeVirtualRouterInput{
		MeshName:          aws.String(mesh),
		VirtualRouterName: aws.String(name),
	})
	if err != nil {
		return nil, err
	}

	d.SetId(aws.StringValue(resp.VirtualRouter.Metadata.Uid))
	d.Set("name", resp.VirtualRouter.VirtualRouterName)
	d.Set("mesh_name", resp.VirtualRouter.MeshName)

	return []*schema.ResourceData{d}, nil
}

func appmeshVirtualRouterListenerHash(vListener interface{}) int {
	var buf bytes.Buffer
	mListener := vListener.(map[string]interface{})
	if vPortMapping, ok := mListener["port_mapping"].([]interface{}); ok && len(vPortMapping) > 0 && vPortMapping[0] != nil {
		mPortMapping := vPortMapping[0].(map[string]interface{})
		if v, ok := mPortMapping["port"].(int); ok {
			buf.WriteString(fmt.Sprintf("%d-", v))
		}
		if v, ok := mPortMapping["protocol"].(string); ok {
			buf.WriteString(fmt.Sprintf("%s-", v))
		}
	}
	return hashcode.String(buf.String())
}
