// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func podTemplateFields(owner string) map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"metadata": namespacedMetadataSchemaIsTemplate(owner, true, true),
		"spec": {
			Type:        schema.TypeList,
			Description: fmt.Sprintf("Spec of the pods owned by the %s", owner),
			Optional:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: podSpecFields(true, false),
			},
		},
	}
	return s
}
