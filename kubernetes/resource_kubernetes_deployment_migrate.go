package kubernetes

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceKubernetesDeploymentV0() *schema.Resource {
	schemaV1 := resourceKubernetesDeploymentSchemaV1()
	schemaV0 := patchTemplatePodSpecWithResourcesFieldV0(schemaV1)
	return &schema.Resource{Schema: schemaV0}
}

func resourceKubernetesDeploymentUpgradeV0(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	return upgradeTemplatePodSpecWithResourcesFieldV0(ctx, rawState, meta)
}
