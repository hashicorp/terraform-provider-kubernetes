package v1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	providerschema "github.com/hashicorp/terraform-provider-kubernetes/kubernetes/schema"
)

func resourceKubernetesReplicationControllerV0() *schema.Resource {
	schemaV1 := resourceKubernetesReplicationControllerSchemaV1()
	schemaV0 := providerschema.PatchTemplatePodSpecWithResourcesFieldV0(schemaV1)
	return &schema.Resource{Schema: schemaV0}
}

func resourceKubernetesReplicationControllerUpgradeV0(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	return providerschema.UpgradeTemplatePodSpecWithResourcesFieldV0(ctx, rawState, meta)
}
