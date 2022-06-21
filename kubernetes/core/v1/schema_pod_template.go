package v1

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	providermetav1 "github.com/hashicorp/terraform-provider-kubernetes/kubernetes/meta/v1"
)

func PodTemplateFields(owner string) map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"metadata": providermetav1.MetadataSchema(owner, true),
		"spec": {
			Type:        schema.TypeList,
			Description: fmt.Sprintf("Spec of the pods owned by the %s", owner),
			Optional:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: PodSpecFields(true, false),
			},
		},
	}
	return s
}
