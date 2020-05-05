package aws

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dax"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func resourceAwsDaxCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDaxClusterCreate,
		Read:   resourceAwsDaxClusterRead,
		Update: resourceAwsDaxClusterUpdate,
		Delete: resourceAwsDaxClusterDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(45 * time.Minute),
			Delete: schema.DefaultTimeout(45 * time.Minute),
			Update: schema.DefaultTimeout(90 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"cluster_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				StateFunc: func(val interface{}) string {
					return strings.ToLower(val.(string))
				},
				ValidateFunc: validation.All(
					validation.StringLenBetween(1, 20),
					validation.StringMatch(regexp.MustCompile(`^[0-9a-z-]+$`), "must contain only lowercase alphanumeric characters and hyphens"),
					validation.StringMatch(regexp.MustCompile(`^[a-z]`), "must begin with a lowercase letter"),
					validateStringNotMatch(regexp.MustCompile(`--`), "cannot contain two consecutive hyphens"),
					validateStringNotMatch(regexp.MustCompile(`-$`), "cannot end with a hyphen"),
				),
			},
			"iam_role_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},
			"node_type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"replication_factor": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"availability_zones": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"notification_topic_arn": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"parameter_group_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"maintenance_window": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				StateFunc: func(val interface{}) string {
					return strings.ToLower(val.(string))
				},
				ValidateFunc: validateOnceAWeekWindowFormat,
			},
			"security_group_ids": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"server_side_encryption": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if old == "1" && new == "0" {
						return true
					}
					return false
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
							ForceNew: true,
						},
					},
				},
			},
			"subnet_group_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"tags": tagsSchema(),
			"port": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"configuration_endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"cluster_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"nodes": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"address": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"port": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"availability_zone": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func resourceAwsDaxClusterCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).daxconn

	clusterName := d.Get("cluster_name").(string)
	iamRoleArn := d.Get("iam_role_arn").(string)
	nodeType := d.Get("node_type").(string)
	numNodes := int64(d.Get("replication_factor").(int))
	subnetGroupName := d.Get("subnet_group_name").(string)
	securityIdSet := d.Get("security_group_ids").(*schema.Set)

	securityIds := expandStringList(securityIdSet.List())
	tags := tagsFromMapDax(d.Get("tags").(map[string]interface{}))

	req := &dax.CreateClusterInput{
		ClusterName:       aws.String(clusterName),
		IamRoleArn:        aws.String(iamRoleArn),
		NodeType:          aws.String(nodeType),
		ReplicationFactor: aws.Int64(numNodes),
		SecurityGroupIds:  securityIds,
		SubnetGroupName:   aws.String(subnetGroupName),
		Tags:              tags,
	}

	// optionals can be defaulted by AWS
	if v, ok := d.GetOk("description"); ok {
		req.Description = aws.String(v.(string))
	}

	if v, ok := d.GetOk("parameter_group_name"); ok {
		req.ParameterGroupName = aws.String(v.(string))
	}

	if v, ok := d.GetOk("maintenance_window"); ok {
		req.PreferredMaintenanceWindow = aws.String(v.(string))
	}

	if v, ok := d.GetOk("notification_topic_arn"); ok {
		req.NotificationTopicArn = aws.String(v.(string))
	}

	preferred_azs := d.Get("availability_zones").(*schema.Set).List()
	if len(preferred_azs) > 0 {
		azs := expandStringList(preferred_azs)
		req.AvailabilityZones = azs
	}

	if v, ok := d.GetOk("server_side_encryption"); ok && len(v.([]interface{})) > 0 {
		options := v.([]interface{})
		s := options[0].(map[string]interface{})
		req.SSESpecification = expandDaxEncryptAtRestOptions(s)
	}

	// IAM roles take some time to propagate
	var resp *dax.CreateClusterOutput
	err := resource.Retry(30*time.Second, func() *resource.RetryError {
		var err error
		resp, err = conn.CreateCluster(req)
		if err != nil {
			if isAWSErr(err, dax.ErrCodeInvalidParameterValueException, "No permission to assume role") {
				log.Print("[DEBUG] Retrying create of DAX cluster")
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if isResourceTimeoutError(err) {
		resp, err = conn.CreateCluster(req)
	}
	if err != nil {
		return fmt.Errorf("Error creating DAX cluster: %s", err)
	}

	// Assign the cluster id as the resource ID
	// DAX always retains the id in lower case, so we have to
	// mimic that or else we won't be able to refresh a resource whose
	// name contained uppercase characters.
	d.SetId(strings.ToLower(*resp.Cluster.ClusterName))

	pending := []string{"creating", "modifying"}
	stateConf := &resource.StateChangeConf{
		Pending:    pending,
		Target:     []string{"available"},
		Refresh:    daxClusterStateRefreshFunc(conn, d.Id(), "available", pending),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second,
	}

	log.Printf("[DEBUG] Waiting for state to become available: %v", d.Id())
	_, sterr := stateConf.WaitForState()
	if sterr != nil {
		return fmt.Errorf("Error waiting for DAX cluster (%s) to be created: %s", d.Id(), sterr)
	}

	return resourceAwsDaxClusterRead(d, meta)
}

func resourceAwsDaxClusterRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).daxconn
	req := &dax.DescribeClustersInput{
		ClusterNames: []*string{aws.String(d.Id())},
	}

	res, err := conn.DescribeClusters(req)
	if err != nil {
		if isAWSErr(err, dax.ErrCodeClusterNotFoundFault, "") {
			log.Printf("[WARN] DAX cluster (%s) not found", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	if len(res.Clusters) == 0 {
		log.Printf("[WARN] DAX cluster (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	c := res.Clusters[0]
	d.Set("arn", c.ClusterArn)
	d.Set("cluster_name", c.ClusterName)
	d.Set("description", c.Description)
	d.Set("iam_role_arn", c.IamRoleArn)
	d.Set("node_type", c.NodeType)
	d.Set("replication_factor", c.TotalNodes)

	if c.ClusterDiscoveryEndpoint != nil {
		d.Set("port", aws.Int64Value(c.ClusterDiscoveryEndpoint.Port))
		d.Set("configuration_endpoint", fmt.Sprintf("%s:%d", aws.StringValue(c.ClusterDiscoveryEndpoint.Address), aws.Int64Value(c.ClusterDiscoveryEndpoint.Port)))
		d.Set("cluster_address", aws.StringValue(c.ClusterDiscoveryEndpoint.Address))
	}

	d.Set("subnet_group_name", c.SubnetGroup)
	d.Set("security_group_ids", flattenDaxSecurityGroupIds(c.SecurityGroups))

	if c.ParameterGroup != nil {
		d.Set("parameter_group_name", c.ParameterGroup.ParameterGroupName)
	}

	d.Set("maintenance_window", c.PreferredMaintenanceWindow)

	if c.NotificationConfiguration != nil {
		if *c.NotificationConfiguration.TopicStatus == "active" {
			d.Set("notification_topic_arn", c.NotificationConfiguration.TopicArn)
		}
	}

	if err := setDaxClusterNodeData(d, c); err != nil {
		return err
	}

	if err := d.Set("server_side_encryption", flattenDaxEncryptAtRestOptions(c.SSEDescription)); err != nil {
		return fmt.Errorf("error setting server_side_encryption: %s", err)
	}

	// list tags for resource
	resp, err := conn.ListTags(&dax.ListTagsInput{
		ResourceName: aws.String(aws.StringValue(c.ClusterArn)),
	})

	if err != nil {
		log.Printf("[DEBUG] Error retrieving tags for ARN: %s", aws.StringValue(c.ClusterArn))
	}

	var dt []*dax.Tag
	if len(resp.Tags) > 0 {
		dt = resp.Tags
	}
	d.Set("tags", tagsToMapDax(dt))

	return nil
}

func resourceAwsDaxClusterUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).daxconn

	if err := setTagsDax(conn, d, d.Get("arn").(string)); err != nil {
		return err
	}

	req := &dax.UpdateClusterInput{
		ClusterName: aws.String(d.Id()),
	}

	requestUpdate := false
	awaitUpdate := false
	if d.HasChange("description") {
		req.Description = aws.String(d.Get("description").(string))
		requestUpdate = true
	}

	if d.HasChange("security_group_ids") {
		if attr := d.Get("security_group_ids").(*schema.Set); attr.Len() > 0 {
			req.SecurityGroupIds = expandStringList(attr.List())
			requestUpdate = true
		}
	}

	if d.HasChange("parameter_group_name") {
		req.ParameterGroupName = aws.String(d.Get("parameter_group_name").(string))
		requestUpdate = true
	}

	if d.HasChange("maintenance_window") {
		req.PreferredMaintenanceWindow = aws.String(d.Get("maintenance_window").(string))
		requestUpdate = true
	}

	if d.HasChange("notification_topic_arn") {
		v := d.Get("notification_topic_arn").(string)
		req.NotificationTopicArn = aws.String(v)
		if v == "" {
			inactive := "inactive"
			req.NotificationTopicStatus = &inactive
		}
		requestUpdate = true
	}

	if requestUpdate {
		log.Printf("[DEBUG] Modifying DAX Cluster (%s), opts:\n%s", d.Id(), req)
		_, err := conn.UpdateCluster(req)
		if err != nil {
			return fmt.Errorf("Error updating DAX cluster (%s), error: %s", d.Id(), err)
		}
		awaitUpdate = true
	}

	if d.HasChange("replication_factor") {
		oraw, nraw := d.GetChange("replication_factor")
		o := oraw.(int)
		n := nraw.(int)
		if n < o {
			log.Printf("[INFO] Decreasing nodes in DAX cluster %s from %d to %d", d.Id(), o, n)
			_, err := conn.DecreaseReplicationFactor(&dax.DecreaseReplicationFactorInput{
				ClusterName:          aws.String(d.Id()),
				NewReplicationFactor: aws.Int64(int64(nraw.(int))),
			})
			if err != nil {
				return fmt.Errorf("Error increasing nodes in DAX cluster %s, error: %s", d.Id(), err)
			}
			awaitUpdate = true
		}
		if n > o {
			log.Printf("[INFO] Increasing nodes in DAX cluster %s from %d to %d", d.Id(), o, n)
			_, err := conn.IncreaseReplicationFactor(&dax.IncreaseReplicationFactorInput{
				ClusterName:          aws.String(d.Id()),
				NewReplicationFactor: aws.Int64(int64(nraw.(int))),
			})
			if err != nil {
				return fmt.Errorf("Error increasing nodes in DAX cluster %s, error: %s", d.Id(), err)
			}
			awaitUpdate = true
		}
	}

	if awaitUpdate {
		log.Printf("[DEBUG] Waiting for update: %s", d.Id())
		pending := []string{"modifying"}
		stateConf := &resource.StateChangeConf{
			Pending:    pending,
			Target:     []string{"available"},
			Refresh:    daxClusterStateRefreshFunc(conn, d.Id(), "available", pending),
			Timeout:    d.Timeout(schema.TimeoutUpdate),
			MinTimeout: 10 * time.Second,
			Delay:      30 * time.Second,
		}

		_, sterr := stateConf.WaitForState()
		if sterr != nil {
			return fmt.Errorf("Error waiting for DAX (%s) to update: %s", d.Id(), sterr)
		}
	}

	return resourceAwsDaxClusterRead(d, meta)
}

func setDaxClusterNodeData(d *schema.ResourceData, c *dax.Cluster) error {
	sortedNodes := make([]*dax.Node, len(c.Nodes))
	copy(sortedNodes, c.Nodes)
	sort.Sort(byNodeId(sortedNodes))

	nodeDate := make([]map[string]interface{}, 0, len(sortedNodes))

	for _, node := range sortedNodes {
		if node.NodeId == nil || node.Endpoint == nil || node.Endpoint.Address == nil || node.Endpoint.Port == nil || node.AvailabilityZone == nil {
			return fmt.Errorf("Unexpected nil pointer in: %s", node)
		}
		nodeDate = append(nodeDate, map[string]interface{}{
			"id":                *node.NodeId,
			"address":           *node.Endpoint.Address,
			"port":              int(*node.Endpoint.Port),
			"availability_zone": *node.AvailabilityZone,
		})
	}

	return d.Set("nodes", nodeDate)
}

type byNodeId []*dax.Node

func (b byNodeId) Len() int      { return len(b) }
func (b byNodeId) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b byNodeId) Less(i, j int) bool {
	return b[i].NodeId != nil && b[j].NodeId != nil &&
		*b[i].NodeId < *b[j].NodeId
}

func resourceAwsDaxClusterDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).daxconn

	req := &dax.DeleteClusterInput{
		ClusterName: aws.String(d.Id()),
	}
	err := resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteCluster(req)
		if err != nil {
			if isAWSErr(err, dax.ErrCodeInvalidClusterStateFault, "") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if isResourceTimeoutError(err) {
		_, err = conn.DeleteCluster(req)
	}
	if err != nil {
		return fmt.Errorf("Error deleting DAX cluster: %s", err)
	}

	log.Printf("[DEBUG] Waiting for deletion: %v", d.Id())
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"creating", "available", "deleting", "incompatible-parameters", "incompatible-network"},
		Target:     []string{},
		Refresh:    daxClusterStateRefreshFunc(conn, d.Id(), "", []string{}),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second,
	}

	_, sterr := stateConf.WaitForState()
	if sterr != nil {
		return fmt.Errorf("Error waiting for DAX (%s) to delete: %s", d.Id(), sterr)
	}

	return nil
}

func daxClusterStateRefreshFunc(conn *dax.DAX, clusterID, givenState string, pending []string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeClusters(&dax.DescribeClustersInput{
			ClusterNames: []*string{aws.String(clusterID)},
		})
		if err != nil {
			if isAWSErr(err, dax.ErrCodeClusterNotFoundFault, "") {
				log.Printf("[DEBUG] Detect deletion")
				return nil, "", nil
			}

			log.Printf("[ERROR] daxClusterStateRefreshFunc: %s", err)
			return nil, "", err
		}

		if len(resp.Clusters) == 0 {
			return nil, "", fmt.Errorf("Error: no DAX clusters found for id (%s)", clusterID)
		}

		var c *dax.Cluster
		for _, cluster := range resp.Clusters {
			if *cluster.ClusterName == clusterID {
				log.Printf("[DEBUG] Found matching DAX cluster: %s", *cluster.ClusterName)
				c = cluster
			}
		}

		if c == nil {
			return nil, "", fmt.Errorf("Error: no matching DAX cluster for id (%s)", clusterID)
		}

		// DescribeCluster returns a response without status late on in the
		// deletion process - assume cluster is still deleting until we
		// get ClusterNotFoundFault
		if c.Status == nil {
			log.Printf("[DEBUG] DAX Cluster %s has no status attribute set - assume status is deleting", clusterID)
			return c, "deleting", nil
		}

		log.Printf("[DEBUG] DAX Cluster (%s) status: %v", clusterID, *c.Status)

		// return the current state if it's in the pending array
		for _, p := range pending {
			log.Printf("[DEBUG] DAX: checking pending state (%s) for cluster (%s), cluster status: %s", pending, clusterID, *c.Status)
			s := *c.Status
			if p == s {
				log.Printf("[DEBUG] Return with status: %v", *c.Status)
				return c, p, nil
			}
		}

		// return given state if it's not in pending
		if givenState != "" {
			log.Printf("[DEBUG] DAX: checking given state (%s) of cluster (%s) against cluster status (%s)", givenState, clusterID, *c.Status)
			// check to make sure we have the node count we're expecting
			if int64(len(c.Nodes)) != *c.TotalNodes {
				log.Printf("[DEBUG] Node count is not what is expected: %d found, %d expected", len(c.Nodes), *c.TotalNodes)
				return nil, "creating", nil
			}

			log.Printf("[DEBUG] Node count matched (%d)", len(c.Nodes))
			// loop the nodes and check their status as well
			for _, n := range c.Nodes {
				log.Printf("[DEBUG] Checking cache node for status: %s", n)
				if n.NodeStatus != nil && *n.NodeStatus != "available" {
					log.Printf("[DEBUG] Node (%s) is not yet available, status: %s", *n.NodeId, *n.NodeStatus)
					return nil, "creating", nil
				}
				log.Printf("[DEBUG] Cache node not in expected state")
			}
			log.Printf("[DEBUG] DAX returning given state (%s), cluster: %s", givenState, c)
			return c, givenState, nil
		}
		log.Printf("[DEBUG] current status: %v", *c.Status)
		return c, *c.Status, nil
	}
}
