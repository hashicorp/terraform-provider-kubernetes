package kubernetes

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	providerschema "github.com/hashicorp/terraform-provider-kubernetes/kubernetes/schema"
)

func resourceKubernetesCronJobV0() *schema.Resource {
	schemaV1 := resourceKubernetesCronJobSchemaV1()
	schemaV0 := providerschema.PatchJobTemplatePodSpecWithResourcesFieldV0(schemaV1)
	return &schema.Resource{Schema: schemaV0}
}

func resourceKubernetesCronJobUpgradeV0(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	return providerschema.UpgradeJobTemplatePodSpecWithResourcesFieldV0(ctx, rawState, meta)
}
