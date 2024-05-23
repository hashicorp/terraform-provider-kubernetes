// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceKubernetesCronJobV0() *schema.Resource {
	schemaV1 := resourceKubernetesCronJobSchemaV1Beta1()
	schemaV0 := patchJobTemplatePodSpecWithResourcesFieldV0(schemaV1)
	return &schema.Resource{Schema: schemaV0}
}

func resourceKubernetesCronJobUpgradeV0(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	// Return a nil error here to satisfy StateUpgradeFunc signature
	return upgradeJobTemplatePodSpecWithResourcesFieldV0(ctx, rawState, meta), nil
}
