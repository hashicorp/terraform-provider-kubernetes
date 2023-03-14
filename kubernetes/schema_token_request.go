// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	apiv1 "k8s.io/api/authentication/v1"
)

func tokenRequestSpecFields() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"audiences": {
			Type:        schema.TypeList,
			Optional:    true,
			ForceNew:    true,
			Description: "Audiences are the intendend audiences of the token. A recipient of a token must identify themself with an identifier in the list of audiences of the token, and otherwise should reject the token. A token issued for multiple audiences may be used to authenticate against any of the audiences listed but implies a high degree of trust between the target audiences.",
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"bound_object_ref": {
			Type:        schema.TypeList,
			Optional:    true,
			ForceNew:    true,
			MaxItems:    1,
			Description: apiv1.TokenRequest{}.Spec.SwaggerDoc()["boundObjectRef"],
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"api_version": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "API version of the referent.",
					},
					"kind": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "Kind of the referent. Valid kinds are 'Pod' and 'Secret'.",
					},
					"name": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "Name of the referent.",
					},
					"uid": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "UID of the referent.",
					},
				},
			},
		},
		"expiration_seconds": {
			Type:        schema.TypeInt,
			Optional:    true,
			Description: "expiration_seconds is the requested duration of validity of the request. The token issuer may return a token with a different validity duration so a client needs to check the 'expiration' field in a response. The expiration can't be less than 10 minutes.",
			ValidateFunc: func(value interface{}, key string) (ws []string, es []error) {
				v := value.(int)
				if v < 600 {
					es = append(es, fmt.Errorf("%s can't be less than 600", key))
				}
				if v > 4294967296 {
					es = append(es, fmt.Errorf("%s can't be greater than 4294967296", key))
				}
				return
			},
		},
	}
	return s
}
