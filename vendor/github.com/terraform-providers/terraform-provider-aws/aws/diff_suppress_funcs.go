package aws

import (
	"bytes"
	"encoding/json"
	"log"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/jen20/awspolicyequivalence"
)

func suppressEquivalentAwsPolicyDiffs(k, old, new string, d *schema.ResourceData) bool {
	equivalent, err := awspolicy.PoliciesAreEquivalent(old, new)
	if err != nil {
		return false
	}

	return equivalent
}

// suppressEquivalentTypeStringBoolean provides custom difference suppression for TypeString booleans
// Some arguments require three values: true, false, and "" (unspecified), but
// confusing behavior exists when converting bare true/false values with state.
func suppressEquivalentTypeStringBoolean(k, old, new string, d *schema.ResourceData) bool {
	if old == "false" && new == "0" {
		return true
	}
	if old == "true" && new == "1" {
		return true
	}
	return false
}

// suppressMissingOptionalConfigurationBlock handles configuration block attributes in the following scenario:
//  * The resource schema includes an optional configuration block with defaults
//  * The API response includes those defaults to refresh into the Terraform state
//  * The operator's configuration omits the optional configuration block
func suppressMissingOptionalConfigurationBlock(k, old, new string, d *schema.ResourceData) bool {
	return old == "1" && new == "0"
}

// Suppresses minor version changes to the db_instance engine_version attribute
func suppressAwsDbEngineVersionDiffs(k, old, new string, d *schema.ResourceData) bool {
	// First check if the old/new values are nil.
	// If both are nil, we have no state to compare the values with, so register a diff.
	// This populates the attribute field during a plan/apply with fresh state, allowing
	// the attribute to still be used in future resources.
	// See https://github.com/hashicorp/terraform/issues/11881
	if old == "" && new == "" {
		return false
	}

	if v, ok := d.GetOk("auto_minor_version_upgrade"); ok {
		if v.(bool) {
			// If we're set to auto upgrade minor versions
			// ignore a minor version diff between versions
			if strings.HasPrefix(old, new) {
				log.Printf("[DEBUG] Ignoring minor version diff")
				return true
			}
		}
	}

	// Throw a diff by default
	return false
}

func suppressEquivalentJsonDiffs(k, old, new string, d *schema.ResourceData) bool {
	ob := bytes.NewBufferString("")
	if err := json.Compact(ob, []byte(old)); err != nil {
		return false
	}

	nb := bytes.NewBufferString("")
	if err := json.Compact(nb, []byte(new)); err != nil {
		return false
	}

	return jsonBytesEqual(ob.Bytes(), nb.Bytes())
}

func suppressOpenIdURL(k, old, new string, d *schema.ResourceData) bool {
	oldUrl, err := url.Parse(old)
	if err != nil {
		return false
	}

	newUrl, err := url.Parse(new)
	if err != nil {
		return false
	}

	oldUrl.Scheme = "https"

	return oldUrl.String() == newUrl.String()
}

func suppressAutoscalingGroupAvailabilityZoneDiffs(k, old, new string, d *schema.ResourceData) bool {
	// If VPC zone identifiers are provided then there is no need to explicitly
	// specify availability zones.
	if _, ok := d.GetOk("vpc_zone_identifier"); ok {
		return true
	}

	return false
}

func suppressCloudFormationTemplateBodyDiffs(k, old, new string, d *schema.ResourceData) bool {
	normalizedOld, err := normalizeCloudFormationTemplate(old)

	if err != nil {
		log.Printf("[WARN] Unable to normalize Terraform state CloudFormation template body: %s", err)
		return false
	}

	normalizedNew, err := normalizeCloudFormationTemplate(new)

	if err != nil {
		log.Printf("[WARN] Unable to normalize Terraform configuration CloudFormation template body: %s", err)
		return false
	}

	return normalizedOld == normalizedNew
}

func suppressRoute53ZoneNameWithTrailingDot(k, old, new string, d *schema.ResourceData) bool {
	// "." is different from "".
	if old == "." || new == "." {
		return old == new
	}
	return strings.TrimSuffix(old, ".") == strings.TrimSuffix(new, ".")
}
