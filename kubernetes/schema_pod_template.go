package kubernetes

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

func podTemplateFields(owner string) map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"metadata": metadataSchema(owner, true),
		"spec": {
			Type:        schema.TypeList,
			Description: fmt.Sprintf("Spec of the pods owned by the %s", owner),
			Optional:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: podSpecFields(true, false, false),
			},
		},
	}
	return s
}
