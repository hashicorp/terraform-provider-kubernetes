package kubernetes

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceKubernetesStatefulSetV0() *schema.Resource {
	schemaV1 := resourceKubernetesStatefulSetSchemaV1()
	schemaV0 := patchTemplatePodSpecWithResourcesFieldV0(schemaV1)
	return &schema.Resource{Schema: schemaV0}
}

func resourceKubernetesStatefulSetUpgradeV0(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	return upgradeTemplatePodSpecWithResourcesFieldV0(ctx, rawState, meta)
}
