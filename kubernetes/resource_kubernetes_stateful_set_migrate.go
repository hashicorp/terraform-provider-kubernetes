package kubernetes

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	providerschema "github.com/hashicorp/terraform-provider-kubernetes/kubernetes/schema"
)

func resourceKubernetesStatefulSetV0() *schema.Resource {
	schemaV1 := resourceKubernetesStatefulSetSchemaV1()
	schemaV0 := providerschema.PatchTemplatePodSpecWithResourcesFieldV0(schemaV1)
	return &schema.Resource{Schema: schemaV0}
}

func resourceKubernetesStatefulSetUpgradeV0(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	return providerschema.UpgradeTemplatePodSpecWithResourcesFieldV0(ctx, rawState, meta)
}
