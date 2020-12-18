package kubernetes

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceKubernetesPodV0() *schema.Resource {
	schemaV1 := resourceKubernetesPodSchemaV1()
	schemaV0 := patchPodSpecWithResourcesFieldV0(schemaV1)
	return &schema.Resource{Schema: schemaV0}
}

func resourceKubernetesPodUpgradeV0(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	return upgradePodSpecWithResourcesFieldV0(ctx, rawState, meta)
}
