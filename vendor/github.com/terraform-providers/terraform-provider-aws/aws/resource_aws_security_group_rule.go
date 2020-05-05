package aws

import (
	"bytes"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/helper/hashcode"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func resourceAwsSecurityGroupRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSecurityGroupRuleCreate,
		Read:   resourceAwsSecurityGroupRuleRead,
		Update: resourceAwsSecurityGroupRuleUpdate,
		Delete: resourceAwsSecurityGroupRuleDelete,
		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				importParts, err := validateSecurityGroupRuleImportString(d.Id())
				if err != nil {
					return nil, err
				}
				if err := populateSecurityGroupRuleFromImport(d, importParts); err != nil {
					return nil, err
				}
				return []*schema.ResourceData{d}, nil
			},
		},

		SchemaVersion: 2,
		MigrateState:  resourceAwsSecurityGroupRuleMigrateState,

		Schema: map[string]*schema.Schema{
			"type": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Type of rule, ingress (inbound) or egress (outbound).",
				ValidateFunc: validation.StringInSlice([]string{
					"ingress",
					"egress",
				}, false),
			},

			"from_port": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
				// Support existing configurations that have non-zero from_port and to_port defined with all protocols
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					protocol := protocolForValue(d.Get("protocol").(string))
					if protocol == "-1" && old == "0" {
						return true
					}
					return false
				},
			},

			"to_port": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
				// Support existing configurations that have non-zero from_port and to_port defined with all protocols
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					protocol := protocolForValue(d.Get("protocol").(string))
					if protocol == "-1" && old == "0" {
						return true
					}
					return false
				},
			},

			"protocol": {
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				StateFunc: protocolStateFunc,
			},

			"cidr_blocks": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validateCIDRNetworkAddress,
				},
			},

			"ipv6_cidr_blocks": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validateCIDRNetworkAddress,
				},
			},

			"prefix_list_ids": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"security_group_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"source_security_group_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				Computed:      true,
				ConflictsWith: []string{"cidr_blocks", "self"},
			},

			"self": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},

			"description": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateSecurityGroupRuleDescription,
			},
		},
	}
}

func resourceAwsSecurityGroupRuleCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	sg_id := d.Get("security_group_id").(string)

	awsMutexKV.Lock(sg_id)
	defer awsMutexKV.Unlock(sg_id)

	sg, err := findResourceSecurityGroup(conn, sg_id)
	if err != nil {
		return err
	}

	perm, err := expandIPPerm(d, sg)
	if err != nil {
		return err
	}

	// Verify that either 'cidr_blocks', 'self', or 'source_security_group_id' is set
	// If they are not set the AWS API will silently fail. This causes TF to hit a timeout
	// at 5-minutes waiting for the security group rule to appear, when it was never actually
	// created.
	if err := validateAwsSecurityGroupRule(d); err != nil {
		return err
	}

	ruleType := d.Get("type").(string)
	isVPC := sg.VpcId != nil && *sg.VpcId != ""

	var autherr error
	switch ruleType {
	case "ingress":
		log.Printf("[DEBUG] Authorizing security group %s %s rule: %s",
			sg_id, "Ingress", perm)

		req := &ec2.AuthorizeSecurityGroupIngressInput{
			GroupId:       sg.GroupId,
			IpPermissions: []*ec2.IpPermission{perm},
		}

		if !isVPC {
			req.GroupId = nil
			req.GroupName = sg.GroupName
		}

		_, autherr = conn.AuthorizeSecurityGroupIngress(req)

	case "egress":
		log.Printf("[DEBUG] Authorizing security group %s %s rule: %#v",
			sg_id, "Egress", perm)

		req := &ec2.AuthorizeSecurityGroupEgressInput{
			GroupId:       sg.GroupId,
			IpPermissions: []*ec2.IpPermission{perm},
		}

		_, autherr = conn.AuthorizeSecurityGroupEgress(req)

	default:
		return fmt.Errorf("Security Group Rule must be type 'ingress' or type 'egress'")
	}

	if autherr != nil {
		if awsErr, ok := autherr.(awserr.Error); ok {
			if awsErr.Code() == "InvalidPermission.Duplicate" {
				return fmt.Errorf(`[WARN] A duplicate Security Group rule was found on (%s). This may be
a side effect of a now-fixed Terraform issue causing two security groups with
identical attributes but different source_security_group_ids to overwrite each
other in the state. See https://github.com/hashicorp/terraform/pull/2376 for more
information and instructions for recovery. Error message: %s`, sg_id, awsErr.Message())
			}
		}

		return fmt.Errorf(
			"Error authorizing security group rule type %s: %s",
			ruleType, autherr)
	}

	var rules []*ec2.IpPermission
	id := ipPermissionIDHash(sg_id, ruleType, perm)
	log.Printf("[DEBUG] Computed group rule ID %s", id)

	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		sg, err := findResourceSecurityGroup(conn, sg_id)

		if err != nil {
			log.Printf("[DEBUG] Error finding Security Group (%s) for Rule (%s): %s", sg_id, id, err)
			return resource.NonRetryableError(err)
		}

		switch ruleType {
		case "ingress":
			rules = sg.IpPermissions
		default:
			rules = sg.IpPermissionsEgress
		}

		rule := findRuleMatch(perm, rules, isVPC)
		if rule == nil {
			log.Printf("[DEBUG] Unable to find matching %s Security Group Rule (%s) for Group %s",
				ruleType, id, sg_id)
			return resource.RetryableError(fmt.Errorf("No match found"))
		}

		log.Printf("[DEBUG] Found rule for Security Group Rule (%s): %s", id, rule)
		return nil
	})
	if isResourceTimeoutError(err) {
		sg, err := findResourceSecurityGroup(conn, sg_id)
		if err != nil {
			return fmt.Errorf("Error finding security group: %s", err)
		}

		switch ruleType {
		case "ingress":
			rules = sg.IpPermissions
		default:
			rules = sg.IpPermissionsEgress
		}

		rule := findRuleMatch(perm, rules, isVPC)
		if rule == nil {
			return fmt.Errorf("Error finding matching security group rule: %s", err)
		}
	}
	if err != nil {
		return fmt.Errorf("Error finding matching %s Security Group Rule (%s) for Group %s", ruleType, id, sg_id)
	}

	d.SetId(id)
	return nil
}

func resourceAwsSecurityGroupRuleRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	sg_id := d.Get("security_group_id").(string)
	sg, err := findResourceSecurityGroup(conn, sg_id)
	if _, notFound := err.(securityGroupNotFound); notFound {
		// The security group containing this rule no longer exists.
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("Error finding security group (%s) for rule (%s): %s", sg_id, d.Id(), err)
	}

	isVPC := sg.VpcId != nil && *sg.VpcId != ""

	var rule *ec2.IpPermission
	var rules []*ec2.IpPermission
	ruleType := d.Get("type").(string)
	switch ruleType {
	case "ingress":
		rules = sg.IpPermissions
	default:
		rules = sg.IpPermissionsEgress
	}
	log.Printf("[DEBUG] Rules %v", rules)

	p, err := expandIPPerm(d, sg)
	if err != nil {
		return err
	}

	if len(rules) == 0 {
		log.Printf("[WARN] No %s rules were found for Security Group (%s) looking for Security Group Rule (%s)",
			ruleType, *sg.GroupName, d.Id())
		d.SetId("")
		return nil
	}

	rule = findRuleMatch(p, rules, isVPC)

	if rule == nil {
		log.Printf("[DEBUG] Unable to find matching %s Security Group Rule (%s) for Group %s",
			ruleType, d.Id(), sg_id)
		d.SetId("")
		return nil
	}

	log.Printf("[DEBUG] Found rule for Security Group Rule (%s): %s", d.Id(), rule)

	d.Set("type", ruleType)
	if err := setFromIPPerm(d, sg, p); err != nil {
		return fmt.Errorf("Error setting IP Permission for Security Group Rule: %s", err)
	}

	d.Set("description", descriptionFromIPPerm(d, rule))

	if strings.Contains(d.Id(), "_") {
		// import so fix the id
		id := ipPermissionIDHash(sg_id, ruleType, p)
		d.SetId(id)
	}

	return nil
}

func resourceAwsSecurityGroupRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	if d.HasChange("description") {
		if err := resourceSecurityGroupRuleDescriptionUpdate(conn, d); err != nil {
			return err
		}
	}

	return resourceAwsSecurityGroupRuleRead(d, meta)
}

func resourceAwsSecurityGroupRuleDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	sg_id := d.Get("security_group_id").(string)

	awsMutexKV.Lock(sg_id)
	defer awsMutexKV.Unlock(sg_id)

	sg, err := findResourceSecurityGroup(conn, sg_id)
	if err != nil {
		return err
	}

	perm, err := expandIPPerm(d, sg)
	if err != nil {
		return err
	}
	ruleType := d.Get("type").(string)
	switch ruleType {
	case "ingress":
		log.Printf("[DEBUG] Revoking rule (%s) from security group %s:\n%s",
			"ingress", sg_id, perm)
		req := &ec2.RevokeSecurityGroupIngressInput{
			GroupId:       sg.GroupId,
			IpPermissions: []*ec2.IpPermission{perm},
		}

		_, err = conn.RevokeSecurityGroupIngress(req)

		if err != nil {
			return fmt.Errorf(
				"Error revoking security group %s rules: %s",
				sg_id, err)
		}
	case "egress":

		log.Printf("[DEBUG] Revoking security group %#v %s rule: %#v",
			sg_id, "egress", perm)
		req := &ec2.RevokeSecurityGroupEgressInput{
			GroupId:       sg.GroupId,
			IpPermissions: []*ec2.IpPermission{perm},
		}

		_, err = conn.RevokeSecurityGroupEgress(req)

		if err != nil {
			return fmt.Errorf(
				"Error revoking security group %s rules: %s",
				sg_id, err)
		}
	}

	return nil
}

func findResourceSecurityGroup(conn *ec2.EC2, id string) (*ec2.SecurityGroup, error) {
	req := &ec2.DescribeSecurityGroupsInput{
		GroupIds: []*string{aws.String(id)},
	}
	resp, err := conn.DescribeSecurityGroups(req)
	if err, ok := err.(awserr.Error); ok && err.Code() == "InvalidGroup.NotFound" {
		return nil, securityGroupNotFound{id, nil}
	}
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, securityGroupNotFound{id, nil}
	}
	if len(resp.SecurityGroups) != 1 || resp.SecurityGroups[0] == nil {
		return nil, securityGroupNotFound{id, resp.SecurityGroups}
	}

	return resp.SecurityGroups[0], nil
}

type securityGroupNotFound struct {
	id             string
	securityGroups []*ec2.SecurityGroup
}

func (err securityGroupNotFound) Error() string {
	if err.securityGroups == nil {
		return fmt.Sprintf("No security group with ID %q", err.id)
	}
	return fmt.Sprintf("Expected to find one security group with ID %q, got: %#v",
		err.id, err.securityGroups)
}

// ByGroupPair implements sort.Interface for []*ec2.UserIDGroupPairs based on
// GroupID or GroupName field (only one should be set).
type ByGroupPair []*ec2.UserIdGroupPair

func (b ByGroupPair) Len() int      { return len(b) }
func (b ByGroupPair) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b ByGroupPair) Less(i, j int) bool {
	if b[i].GroupId != nil && b[j].GroupId != nil {
		return *b[i].GroupId < *b[j].GroupId
	}
	if b[i].GroupName != nil && b[j].GroupName != nil {
		return *b[i].GroupName < *b[j].GroupName
	}

	panic("mismatched security group rules, may be a terraform bug")
}

func findRuleMatch(p *ec2.IpPermission, rules []*ec2.IpPermission, isVPC bool) *ec2.IpPermission {
	var rule *ec2.IpPermission
	for _, r := range rules {
		if p.ToPort != nil && r.ToPort != nil && *p.ToPort != *r.ToPort {
			continue
		}

		if p.FromPort != nil && r.FromPort != nil && *p.FromPort != *r.FromPort {
			continue
		}

		if p.IpProtocol != nil && r.IpProtocol != nil && *p.IpProtocol != *r.IpProtocol {
			continue
		}

		remaining := len(p.IpRanges)
		for _, ip := range p.IpRanges {
			for _, rip := range r.IpRanges {
				if ip.CidrIp == nil || rip.CidrIp == nil {
					continue
				}
				if *ip.CidrIp == *rip.CidrIp {
					remaining--
				}
			}
		}

		if remaining > 0 {
			continue
		}

		remaining = len(p.Ipv6Ranges)
		for _, ipv6 := range p.Ipv6Ranges {
			for _, ipv6ip := range r.Ipv6Ranges {
				if ipv6.CidrIpv6 == nil || ipv6ip.CidrIpv6 == nil {
					continue
				}
				if *ipv6.CidrIpv6 == *ipv6ip.CidrIpv6 {
					remaining--
				}
			}
		}

		if remaining > 0 {
			continue
		}

		remaining = len(p.PrefixListIds)
		for _, pl := range p.PrefixListIds {
			for _, rpl := range r.PrefixListIds {
				if pl.PrefixListId == nil || rpl.PrefixListId == nil {
					continue
				}
				if *pl.PrefixListId == *rpl.PrefixListId {
					remaining--
				}
			}
		}

		if remaining > 0 {
			continue
		}

		remaining = len(p.UserIdGroupPairs)
		for _, ip := range p.UserIdGroupPairs {
			for _, rip := range r.UserIdGroupPairs {
				if isVPC {
					if ip.GroupId == nil || rip.GroupId == nil {
						continue
					}
					if *ip.GroupId == *rip.GroupId {
						remaining--
					}
				} else {
					if ip.GroupName == nil || rip.GroupName == nil {
						continue
					}
					if *ip.GroupName == *rip.GroupName {
						remaining--
					}
				}
			}
		}

		if remaining > 0 {
			continue
		}

		rule = r
	}
	return rule
}

func ipPermissionIDHash(sg_id, ruleType string, ip *ec2.IpPermission) string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("%s-", sg_id))
	if ip.FromPort != nil && *ip.FromPort > 0 {
		buf.WriteString(fmt.Sprintf("%d-", *ip.FromPort))
	}
	if ip.ToPort != nil && *ip.ToPort > 0 {
		buf.WriteString(fmt.Sprintf("%d-", *ip.ToPort))
	}
	buf.WriteString(fmt.Sprintf("%s-", *ip.IpProtocol))
	buf.WriteString(fmt.Sprintf("%s-", ruleType))

	// We need to make sure to sort the strings below so that we always
	// generate the same hash code no matter what is in the set.
	if len(ip.IpRanges) > 0 {
		s := make([]string, len(ip.IpRanges))
		for i, r := range ip.IpRanges {
			s[i] = *r.CidrIp
		}
		sort.Strings(s)

		for _, v := range s {
			buf.WriteString(fmt.Sprintf("%s-", v))
		}
	}

	if len(ip.Ipv6Ranges) > 0 {
		s := make([]string, len(ip.Ipv6Ranges))
		for i, r := range ip.Ipv6Ranges {
			s[i] = *r.CidrIpv6
		}
		sort.Strings(s)

		for _, v := range s {
			buf.WriteString(fmt.Sprintf("%s-", v))
		}
	}

	if len(ip.PrefixListIds) > 0 {
		s := make([]string, len(ip.PrefixListIds))
		for i, pl := range ip.PrefixListIds {
			s[i] = *pl.PrefixListId
		}
		sort.Strings(s)

		for _, v := range s {
			buf.WriteString(fmt.Sprintf("%s-", v))
		}
	}

	if len(ip.UserIdGroupPairs) > 0 {
		sort.Sort(ByGroupPair(ip.UserIdGroupPairs))
		for _, pair := range ip.UserIdGroupPairs {
			if pair.GroupId != nil {
				buf.WriteString(fmt.Sprintf("%s-", *pair.GroupId))
			} else {
				buf.WriteString("-")
			}
			if pair.GroupName != nil {
				buf.WriteString(fmt.Sprintf("%s-", *pair.GroupName))
			} else {
				buf.WriteString("-")
			}
		}
	}

	return fmt.Sprintf("sgrule-%d", hashcode.String(buf.String()))
}

func expandIPPerm(d *schema.ResourceData, sg *ec2.SecurityGroup) (*ec2.IpPermission, error) {
	var perm ec2.IpPermission

	protocol := protocolForValue(d.Get("protocol").(string))
	perm.IpProtocol = aws.String(protocol)

	// InvalidParameterValue: When protocol is ALL, you cannot specify from-port.
	if protocol != "-1" {
		perm.FromPort = aws.Int64(int64(d.Get("from_port").(int)))
		perm.ToPort = aws.Int64(int64(d.Get("to_port").(int)))
	}

	// build a group map that behaves like a set
	groups := make(map[string]bool)
	if raw, ok := d.GetOk("source_security_group_id"); ok {
		groups[raw.(string)] = true
	}

	if v, ok := d.GetOk("self"); ok && v.(bool) {
		if sg.VpcId != nil && *sg.VpcId != "" {
			groups[*sg.GroupId] = true
		} else {
			groups[*sg.GroupName] = true
		}
	}

	description := d.Get("description").(string)

	if len(groups) > 0 {
		perm.UserIdGroupPairs = make([]*ec2.UserIdGroupPair, len(groups))
		// build string list of group name/ids
		var gl []string
		for k := range groups {
			gl = append(gl, k)
		}

		for i, name := range gl {
			ownerId, id := "", name
			if items := strings.Split(id, "/"); len(items) > 1 {
				ownerId, id = items[0], items[1]
			}

			perm.UserIdGroupPairs[i] = &ec2.UserIdGroupPair{
				GroupId: aws.String(id),
				UserId:  aws.String(ownerId),
			}

			if sg.VpcId == nil || *sg.VpcId == "" {
				perm.UserIdGroupPairs[i].GroupId = nil
				perm.UserIdGroupPairs[i].GroupName = aws.String(id)
				perm.UserIdGroupPairs[i].UserId = nil
			}

			if description != "" {
				perm.UserIdGroupPairs[i].Description = aws.String(description)
			}
		}
	}

	if raw, ok := d.GetOk("cidr_blocks"); ok {
		list := raw.([]interface{})
		perm.IpRanges = make([]*ec2.IpRange, len(list))
		for i, v := range list {
			cidrIP, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("empty element found in cidr_blocks - consider using the compact function")
			}
			perm.IpRanges[i] = &ec2.IpRange{CidrIp: aws.String(cidrIP)}

			if description != "" {
				perm.IpRanges[i].Description = aws.String(description)
			}
		}
	}

	if raw, ok := d.GetOk("ipv6_cidr_blocks"); ok {
		list := raw.([]interface{})
		perm.Ipv6Ranges = make([]*ec2.Ipv6Range, len(list))
		for i, v := range list {
			cidrIP, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("empty element found in ipv6_cidr_blocks - consider using the compact function")
			}
			perm.Ipv6Ranges[i] = &ec2.Ipv6Range{CidrIpv6: aws.String(cidrIP)}

			if description != "" {
				perm.Ipv6Ranges[i].Description = aws.String(description)
			}
		}
	}

	if raw, ok := d.GetOk("prefix_list_ids"); ok {
		list := raw.([]interface{})
		perm.PrefixListIds = make([]*ec2.PrefixListId, len(list))
		for i, v := range list {
			prefixListID, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("empty element found in prefix_list_ids - consider using the compact function")
			}
			perm.PrefixListIds[i] = &ec2.PrefixListId{PrefixListId: aws.String(prefixListID)}

			if description != "" {
				perm.PrefixListIds[i].Description = aws.String(description)
			}
		}
	}

	return &perm, nil
}

func setFromIPPerm(d *schema.ResourceData, sg *ec2.SecurityGroup, rule *ec2.IpPermission) error {
	isVPC := sg.VpcId != nil && *sg.VpcId != ""

	d.Set("from_port", rule.FromPort)
	d.Set("to_port", rule.ToPort)
	d.Set("protocol", rule.IpProtocol)

	var cb []string
	for _, c := range rule.IpRanges {
		cb = append(cb, *c.CidrIp)
	}
	d.Set("cidr_blocks", cb)

	var ipv6 []string
	for _, ip := range rule.Ipv6Ranges {
		ipv6 = append(ipv6, *ip.CidrIpv6)
	}
	d.Set("ipv6_cidr_blocks", ipv6)

	var pl []string
	for _, p := range rule.PrefixListIds {
		pl = append(pl, *p.PrefixListId)
	}
	d.Set("prefix_list_ids", pl)

	if len(rule.UserIdGroupPairs) > 0 {
		s := rule.UserIdGroupPairs[0]

		if isVPC {
			d.Set("source_security_group_id", *s.GroupId)
		} else {
			d.Set("source_security_group_id", *s.GroupName)
		}
	}

	return nil
}

func descriptionFromIPPerm(d *schema.ResourceData, rule *ec2.IpPermission) string {
	// probe IpRanges
	cidrIps := make(map[string]bool)
	if raw, ok := d.GetOk("cidr_blocks"); ok {
		for _, v := range raw.([]interface{}) {
			cidrIps[v.(string)] = true
		}
	}

	if len(cidrIps) > 0 {
		for _, c := range rule.IpRanges {
			if _, ok := cidrIps[*c.CidrIp]; !ok {
				continue
			}

			if desc := aws.StringValue(c.Description); desc != "" {
				return desc
			}
		}
	}

	// probe Ipv6Ranges
	cidrIpv6s := make(map[string]bool)
	if raw, ok := d.GetOk("ipv6_cidr_blocks"); ok {
		for _, v := range raw.([]interface{}) {
			cidrIpv6s[v.(string)] = true
		}
	}

	if len(cidrIpv6s) > 0 {
		for _, ip := range rule.Ipv6Ranges {
			if _, ok := cidrIpv6s[*ip.CidrIpv6]; !ok {
				continue
			}

			if desc := aws.StringValue(ip.Description); desc != "" {
				return desc
			}
		}
	}

	// probe PrefixListIds
	listIds := make(map[string]bool)
	if raw, ok := d.GetOk("prefix_list_ids"); ok {
		for _, v := range raw.([]interface{}) {
			listIds[v.(string)] = true
		}
	}

	if len(listIds) > 0 {
		for _, p := range rule.PrefixListIds {
			if _, ok := listIds[*p.PrefixListId]; !ok {
				continue
			}

			if desc := aws.StringValue(p.Description); desc != "" {
				return desc
			}
		}
	}

	// probe UserIdGroupPairs
	groupIds := make(map[string]bool)
	if raw, ok := d.GetOk("source_security_group_id"); ok {
		groupIds[raw.(string)] = true
	}

	if len(groupIds) > 0 {
		for _, gp := range rule.UserIdGroupPairs {
			if _, ok := groupIds[*gp.GroupId]; !ok {
				continue
			}

			if desc := aws.StringValue(gp.Description); desc != "" {
				return desc
			}
		}
	}

	return ""
}

// Validates that either 'cidr_blocks', 'ipv6_cidr_blocks', 'self', or 'source_security_group_id' is set
func validateAwsSecurityGroupRule(d *schema.ResourceData) error {
	blocks, blocksOk := d.GetOk("cidr_blocks")
	self, selfOk := d.GetOk("self")
	if blocksOk && self.(bool) {
		return fmt.Errorf("'self': conflicts with 'cidr_blocks' (%#v)", blocks)
	}

	_, ipv6Ok := d.GetOk("ipv6_cidr_blocks")
	_, sourceOk := d.GetOk("source_security_group_id")
	_, prefixOk := d.GetOk("prefix_list_ids")
	if !blocksOk && !sourceOk && !selfOk && !prefixOk && !ipv6Ok {
		return fmt.Errorf(
			"One of ['cidr_blocks', 'ipv6_cidr_blocks', 'self', 'source_security_group_id', 'prefix_list_ids'] must be set to create an AWS Security Group Rule")
	}
	return nil
}

func resourceSecurityGroupRuleDescriptionUpdate(conn *ec2.EC2, d *schema.ResourceData) error {
	sg_id := d.Get("security_group_id").(string)

	awsMutexKV.Lock(sg_id)
	defer awsMutexKV.Unlock(sg_id)

	sg, err := findResourceSecurityGroup(conn, sg_id)
	if err != nil {
		return err
	}

	perm, err := expandIPPerm(d, sg)
	if err != nil {
		return err
	}
	ruleType := d.Get("type").(string)
	switch ruleType {
	case "ingress":
		req := &ec2.UpdateSecurityGroupRuleDescriptionsIngressInput{
			GroupId:       sg.GroupId,
			IpPermissions: []*ec2.IpPermission{perm},
		}

		_, err = conn.UpdateSecurityGroupRuleDescriptionsIngress(req)

		if err != nil {
			return fmt.Errorf(
				"Error updating security group %s rule description: %s",
				sg_id, err)
		}
	case "egress":
		req := &ec2.UpdateSecurityGroupRuleDescriptionsEgressInput{
			GroupId:       sg.GroupId,
			IpPermissions: []*ec2.IpPermission{perm},
		}

		_, err = conn.UpdateSecurityGroupRuleDescriptionsEgress(req)

		if err != nil {
			return fmt.Errorf(
				"Error updating security group %s rule description: %s",
				sg_id, err)
		}
	}

	return nil
}

// validateSecurityGroupRuleImportString does minimal validation of import string without going to AWS
func validateSecurityGroupRuleImportString(importStr string) ([]string, error) {
	// example: sg-09a093729ef9382a6_ingress_tcp_8000_8000_10.0.3.0/24
	// example: sg-09a093729ef9382a6_ingress_92_0_65536_10.0.3.0/24_10.0.4.0/24
	// example: sg-09a093729ef9382a6_egress_tcp_8000_8000_10.0.3.0/24
	// example: sg-09a093729ef9382a6_egress_tcp_8000_8000_pl-34800000
	// example: sg-09a093729ef9382a6_ingress_all_0_65536_sg-08123412342323
	// example: sg-09a093729ef9382a6_ingress_tcp_100_121_10.1.0.0/16_2001:db8::/48_10.2.0.0/16_2002:db8::/48

	log.Printf("[DEBUG] Validating import string %s", importStr)

	importParts := strings.Split(strings.ToLower(importStr), "_")
	errStr := "unexpected format of import string (%q), expected SECURITYGROUPID_TYPE_PROTOCOL_FROMPORT_TOPORT_SOURCE[_SOURCE]*: %s"
	if len(importParts) < 6 {
		return nil, fmt.Errorf(errStr, importStr, "too few parts")
	}

	sgID := importParts[0]
	ruleType := importParts[1]
	protocol := importParts[2]
	fromPort := importParts[3]
	toPort := importParts[4]
	sources := importParts[5:]

	if !strings.HasPrefix(sgID, "sg-") {
		return nil, fmt.Errorf(errStr, importStr, "invalid security group ID")
	}

	if ruleType != "ingress" && ruleType != "egress" {
		return nil, fmt.Errorf(errStr, importStr, "expecting 'ingress' or 'egress'")
	}

	if _, ok := sgProtocolIntegers()[protocol]; !ok {
		if _, err := strconv.Atoi(protocol); err != nil {
			return nil, fmt.Errorf(errStr, importStr, "protocol must be tcp/udp/icmp/all or a number")
		}
	}

	if p1, err := strconv.Atoi(fromPort); err != nil {
		return nil, fmt.Errorf(errStr, importStr, "invalid port")
	} else if p2, err := strconv.Atoi(toPort); err != nil || p2 < p1 {
		return nil, fmt.Errorf(errStr, importStr, "invalid port")
	}

	for _, source := range sources {
		// will be properly validated later
		if source != "self" && !strings.Contains(source, "sg-") && !strings.Contains(source, "pl-") && !strings.Contains(source, ":") && !strings.Contains(source, ".") {
			return nil, fmt.Errorf(errStr, importStr, "source must be cidr, ipv6cidr, prefix list, 'self', or a sg ID")
		}
	}

	log.Printf("[DEBUG] Validated import string %s", importStr)
	return importParts, nil
}

func populateSecurityGroupRuleFromImport(d *schema.ResourceData, importParts []string) error {
	log.Printf("[DEBUG] Populating resource data on import: %v", importParts)

	sgID := importParts[0]
	ruleType := importParts[1]
	protocol := importParts[2]
	fromPort, _ := strconv.Atoi(importParts[3])
	toPort, _ := strconv.Atoi(importParts[4])
	sources := importParts[5:]

	d.Set("security_group_id", sgID)

	if ruleType == "ingress" {
		d.Set("type", ruleType)
	} else {
		d.Set("type", "egress")
	}

	d.Set("protocol", protocolForValue(protocol))
	d.Set("from_port", fromPort)
	d.Set("to_port", toPort)

	d.Set("self", false)
	var cidrs []string
	var prefixList []string
	var ipv6cidrs []string
	for _, source := range sources {
		if source == "self" {
			d.Set("self", true)
		} else if strings.Contains(source, "sg-") {
			d.Set("source_security_group_id", source)
		} else if strings.Contains(source, "pl-") {
			prefixList = append(prefixList, source)
		} else if strings.Contains(source, ":") {
			ipv6cidrs = append(ipv6cidrs, source)
		} else {
			cidrs = append(cidrs, source)
		}
	}
	d.Set("ipv6_cidr_blocks", ipv6cidrs)
	d.Set("cidr_blocks", cidrs)
	d.Set("prefix_list_ids", prefixList)

	return nil
}
