package kubernetes

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

func podTemplateFields(controller string) map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"metadata": metadataSchema(controller, true),
		"spec": {
			Type:        schema.TypeList,
			Description: fmt.Sprintf("Spec of the pods owned by the %s", controller),
			Optional:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: podSpecFields(true, false, false),
			},
		},
	}
	return s
}
