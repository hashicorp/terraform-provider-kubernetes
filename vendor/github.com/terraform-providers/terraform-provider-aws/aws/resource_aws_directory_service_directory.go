package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/directoryservice"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsDirectoryServiceDirectory() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDirectoryServiceDirectoryCreate,
		Read:   resourceAwsDirectoryServiceDirectoryRead,
		Update: resourceAwsDirectoryServiceDirectoryUpdate,
		Delete: resourceAwsDirectoryServiceDirectoryDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"password": {
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				Sensitive: true,
			},
			"size": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					directoryservice.DirectorySizeLarge,
					directoryservice.DirectorySizeSmall,
				}, false),
			},
			"alias": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"short_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"tags": tagsSchema(),
			"vpc_settings": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"subnet_ids": {
							Type:     schema.TypeSet,
							Required: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
						"vpc_id": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
					},
				},
			},
			"connect_settings": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"customer_username": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"customer_dns_ips": {
							Type:     schema.TypeSet,
							Required: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
						"subnet_ids": {
							Type:     schema.TypeSet,
							Required: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
						"vpc_id": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
					},
				},
			},
			"enable_sso": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"access_url": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"dns_ip_addresses": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
				Computed: true,
			},
			"security_group_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"type": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  directoryservice.DirectoryTypeSimpleAd,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					directoryservice.DirectoryTypeAdconnector,
					directoryservice.DirectoryTypeMicrosoftAd,
					directoryservice.DirectoryTypeSimpleAd,
				}, false),
			},
			"edition": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					directoryservice.DirectoryEditionEnterprise,
					directoryservice.DirectoryEditionStandard,
				}, false),
			},
		},
	}
}

func buildVpcSettings(d *schema.ResourceData) (vpcSettings *directoryservice.DirectoryVpcSettings, err error) {
	v, ok := d.GetOk("vpc_settings")
	if !ok {
		return nil, fmt.Errorf("vpc_settings is required for type = SimpleAD or MicrosoftAD")
	}
	settings := v.([]interface{})
	s := settings[0].(map[string]interface{})
	var subnetIds []*string
	for _, id := range s["subnet_ids"].(*schema.Set).List() {
		subnetIds = append(subnetIds, aws.String(id.(string)))
	}

	vpcSettings = &directoryservice.DirectoryVpcSettings{
		SubnetIds: subnetIds,
		VpcId:     aws.String(s["vpc_id"].(string)),
	}

	return vpcSettings, nil
}

func buildConnectSettings(d *schema.ResourceData) (connectSettings *directoryservice.DirectoryConnectSettings, err error) {
	v, ok := d.GetOk("connect_settings")
	if !ok {
		return nil, fmt.Errorf("connect_settings is required for type = ADConnector")
	}
	settings := v.([]interface{})
	s := settings[0].(map[string]interface{})

	var subnetIds []*string
	for _, id := range s["subnet_ids"].(*schema.Set).List() {
		subnetIds = append(subnetIds, aws.String(id.(string)))
	}

	var customerDnsIps []*string
	for _, id := range s["customer_dns_ips"].(*schema.Set).List() {
		customerDnsIps = append(customerDnsIps, aws.String(id.(string)))
	}

	connectSettings = &directoryservice.DirectoryConnectSettings{
		CustomerDnsIps:   customerDnsIps,
		CustomerUserName: aws.String(s["customer_username"].(string)),
		SubnetIds:        subnetIds,
		VpcId:            aws.String(s["vpc_id"].(string)),
	}

	return connectSettings, nil
}

func createDirectoryConnector(dsconn *directoryservice.DirectoryService, d *schema.ResourceData) (directoryId string, err error) {
	input := directoryservice.ConnectDirectoryInput{
		Name:     aws.String(d.Get("name").(string)),
		Password: aws.String(d.Get("password").(string)),
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}
	if v, ok := d.GetOk("size"); ok {
		input.Size = aws.String(v.(string))
	} else {
		// Matching previous behavior of Default: "Large" for Size attribute
		input.Size = aws.String(directoryservice.DirectorySizeLarge)
	}
	if v, ok := d.GetOk("short_name"); ok {
		input.ShortName = aws.String(v.(string))
	}

	input.ConnectSettings, err = buildConnectSettings(d)
	if err != nil {
		return "", err
	}

	log.Printf("[DEBUG] Creating Directory Connector: %s", input)
	out, err := dsconn.ConnectDirectory(&input)
	if err != nil {
		return "", err
	}
	log.Printf("[DEBUG] Directory Connector created: %s", out)

	return *out.DirectoryId, nil
}

func createSimpleDirectoryService(dsconn *directoryservice.DirectoryService, d *schema.ResourceData) (directoryId string, err error) {
	input := directoryservice.CreateDirectoryInput{
		Name:     aws.String(d.Get("name").(string)),
		Password: aws.String(d.Get("password").(string)),
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}
	if v, ok := d.GetOk("size"); ok {
		input.Size = aws.String(v.(string))
	} else {
		// Matching previous behavior of Default: "Large" for Size attribute
		input.Size = aws.String(directoryservice.DirectorySizeLarge)
	}
	if v, ok := d.GetOk("short_name"); ok {
		input.ShortName = aws.String(v.(string))
	}

	input.VpcSettings, err = buildVpcSettings(d)
	if err != nil {
		return "", err
	}

	log.Printf("[DEBUG] Creating Simple Directory Service: %s", input)
	out, err := dsconn.CreateDirectory(&input)
	if err != nil {
		return "", err
	}
	log.Printf("[DEBUG] Simple Directory Service created: %s", out)

	return *out.DirectoryId, nil
}

func createActiveDirectoryService(dsconn *directoryservice.DirectoryService, d *schema.ResourceData) (directoryId string, err error) {
	input := directoryservice.CreateMicrosoftADInput{
		Name:     aws.String(d.Get("name").(string)),
		Password: aws.String(d.Get("password").(string)),
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}
	if v, ok := d.GetOk("short_name"); ok {
		input.ShortName = aws.String(v.(string))
	}
	if v, ok := d.GetOk("edition"); ok {
		input.Edition = aws.String(v.(string))
	}

	input.VpcSettings, err = buildVpcSettings(d)
	if err != nil {
		return "", err
	}

	log.Printf("[DEBUG] Creating Microsoft AD Directory Service: %s", input)
	out, err := dsconn.CreateMicrosoftAD(&input)
	if err != nil {
		return "", err
	}
	log.Printf("[DEBUG] Microsoft AD Directory Service created: %s", out)

	return *out.DirectoryId, nil
}

func resourceAwsDirectoryServiceDirectoryCreate(d *schema.ResourceData, meta interface{}) error {
	dsconn := meta.(*AWSClient).dsconn

	var directoryId string
	var err error
	directoryType := d.Get("type").(string)

	if directoryType == directoryservice.DirectoryTypeAdconnector {
		directoryId, err = createDirectoryConnector(dsconn, d)
	} else if directoryType == directoryservice.DirectoryTypeMicrosoftAd {
		directoryId, err = createActiveDirectoryService(dsconn, d)
	} else if directoryType == directoryservice.DirectoryTypeSimpleAd {
		directoryId, err = createSimpleDirectoryService(dsconn, d)
	}

	if err != nil {
		return err
	}

	d.SetId(directoryId)

	// Wait for creation
	log.Printf("[DEBUG] Waiting for DS (%q) to become available", d.Id())
	stateConf := &resource.StateChangeConf{
		Pending: []string{
			directoryservice.DirectoryStageRequested,
			directoryservice.DirectoryStageCreating,
			directoryservice.DirectoryStageCreated,
		},
		Target: []string{directoryservice.DirectoryStageActive},
		Refresh: func() (interface{}, string, error) {
			resp, err := dsconn.DescribeDirectories(&directoryservice.DescribeDirectoriesInput{
				DirectoryIds: []*string{aws.String(d.Id())},
			})
			if err != nil {
				log.Printf("Error during creation of DS: %q", err.Error())
				return nil, "", err
			}

			ds := resp.DirectoryDescriptions[0]
			log.Printf("[DEBUG] Creation of DS %q is in following stage: %q.",
				d.Id(), *ds.Stage)
			return ds, *ds.Stage, nil
		},
		Timeout: 60 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf(
			"Error waiting for Directory Service (%s) to become available: %s",
			d.Id(), err)
	}

	if v, ok := d.GetOk("alias"); ok {
		d.SetPartial("alias")

		input := directoryservice.CreateAliasInput{
			DirectoryId: aws.String(d.Id()),
			Alias:       aws.String(v.(string)),
		}

		log.Printf("[DEBUG] Assigning alias %q to DS directory %q",
			v.(string), d.Id())
		out, err := dsconn.CreateAlias(&input)
		if err != nil {
			return err
		}
		log.Printf("[DEBUG] Alias %q assigned to DS directory %q",
			*out.Alias, *out.DirectoryId)
	}

	return resourceAwsDirectoryServiceDirectoryUpdate(d, meta)
}

func resourceAwsDirectoryServiceDirectoryUpdate(d *schema.ResourceData, meta interface{}) error {
	dsconn := meta.(*AWSClient).dsconn

	if d.HasChange("enable_sso") {
		d.SetPartial("enable_sso")
		var err error

		if v, ok := d.GetOk("enable_sso"); ok && v.(bool) {
			log.Printf("[DEBUG] Enabling SSO for DS directory %q", d.Id())
			_, err = dsconn.EnableSso(&directoryservice.EnableSsoInput{
				DirectoryId: aws.String(d.Id()),
			})
		} else {
			log.Printf("[DEBUG] Disabling SSO for DS directory %q", d.Id())
			_, err = dsconn.DisableSso(&directoryservice.DisableSsoInput{
				DirectoryId: aws.String(d.Id()),
			})
		}

		if err != nil {
			return err
		}
	}

	if err := setTagsDS(dsconn, d, d.Id()); err != nil {
		return err
	}

	return resourceAwsDirectoryServiceDirectoryRead(d, meta)
}

func resourceAwsDirectoryServiceDirectoryRead(d *schema.ResourceData, meta interface{}) error {
	dsconn := meta.(*AWSClient).dsconn

	input := directoryservice.DescribeDirectoriesInput{
		DirectoryIds: []*string{aws.String(d.Id())},
	}
	out, err := dsconn.DescribeDirectories(&input)
	if err != nil {
		return err

	}

	if len(out.DirectoryDescriptions) == 0 {
		log.Printf("[WARN] Directory %s not found", d.Id())
		d.SetId("")
		return nil
	}

	dir := out.DirectoryDescriptions[0]
	log.Printf("[DEBUG] Received DS directory: %s", dir)

	d.Set("access_url", dir.AccessUrl)
	d.Set("alias", dir.Alias)
	d.Set("description", dir.Description)

	if *dir.Type == directoryservice.DirectoryTypeAdconnector {
		d.Set("dns_ip_addresses", schema.NewSet(schema.HashString, flattenStringList(dir.ConnectSettings.ConnectIps)))
	} else {
		d.Set("dns_ip_addresses", schema.NewSet(schema.HashString, flattenStringList(dir.DnsIpAddrs)))
	}
	d.Set("name", dir.Name)
	d.Set("short_name", dir.ShortName)
	d.Set("size", dir.Size)
	d.Set("edition", dir.Edition)
	d.Set("type", dir.Type)
	d.Set("vpc_settings", flattenDSVpcSettings(dir.VpcSettings))
	d.Set("connect_settings", flattenDSConnectSettings(dir.DnsIpAddrs, dir.ConnectSettings))
	d.Set("enable_sso", dir.SsoEnabled)

	if aws.StringValue(dir.Type) == directoryservice.DirectoryTypeAdconnector {
		d.Set("security_group_id", aws.StringValue(dir.ConnectSettings.SecurityGroupId))
	} else {
		d.Set("security_group_id", aws.StringValue(dir.VpcSettings.SecurityGroupId))
	}

	tagList, err := dsconn.ListTagsForResource(&directoryservice.ListTagsForResourceInput{
		ResourceId: aws.String(d.Id()),
	})
	if err != nil {
		return fmt.Errorf("Failed to get Directory service tags (id: %s): %s", d.Id(), err)
	}
	d.Set("tags", tagsToMapDS(tagList.Tags))

	return nil
}

func resourceAwsDirectoryServiceDirectoryDelete(d *schema.ResourceData, meta interface{}) error {
	dsconn := meta.(*AWSClient).dsconn

	input := directoryservice.DeleteDirectoryInput{
		DirectoryId: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Deleting Directory Service Directory: %s", input)
	_, err := dsconn.DeleteDirectory(&input)
	if err != nil {
		return fmt.Errorf("error deleting Directory Service Directory (%s): %s", d.Id(), err)
	}

	log.Printf("[DEBUG] Waiting for Directory Service Directory (%q) to be deleted", d.Id())
	err = waitForDirectoryServiceDirectoryDeletion(dsconn, d.Id())
	if err != nil {
		return fmt.Errorf("error waiting for Directory Service (%s) to be deleted: %s", d.Id(), err)
	}

	return nil
}

func waitForDirectoryServiceDirectoryDeletion(conn *directoryservice.DirectoryService, directoryID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{
			directoryservice.DirectoryStageActive,
			directoryservice.DirectoryStageDeleting,
		},
		Target: []string{directoryservice.DirectoryStageDeleted},
		Refresh: func() (interface{}, string, error) {
			resp, err := conn.DescribeDirectories(&directoryservice.DescribeDirectoriesInput{
				DirectoryIds: []*string{aws.String(directoryID)},
			})
			if err != nil {
				if isAWSErr(err, directoryservice.ErrCodeEntityDoesNotExistException, "") {
					return 42, directoryservice.DirectoryStageDeleted, nil
				}
				return nil, "error", err
			}

			if len(resp.DirectoryDescriptions) == 0 || resp.DirectoryDescriptions[0] == nil {
				return 42, directoryservice.DirectoryStageDeleted, nil
			}

			ds := resp.DirectoryDescriptions[0]
			log.Printf("[DEBUG] Deletion of Directory Service Directory %q is in following stage: %q.", directoryID, aws.StringValue(ds.Stage))
			return ds, aws.StringValue(ds.Stage), nil
		},
		Timeout: 60 * time.Minute,
	}
	_, err := stateConf.WaitForState()

	return err
}
