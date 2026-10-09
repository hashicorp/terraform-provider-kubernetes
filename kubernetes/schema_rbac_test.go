// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestRbacRoleRefSchemaApiGroupAcceptsOpenShift(t *testing.T) {
	roleRef := rbacRoleRefSchema()
	apiGroup := roleRef["api_group"]
	if apiGroup == nil {
		t.Fatal("api_group schema missing")
	}
	if !apiGroup.Required {
		t.Error("api_group should remain Required")
	}
	if !apiGroup.ForceNew {
		t.Error("api_group should remain ForceNew")
	}

	value := "roles.authorization.openshift.io"
	if apiGroup.ValidateFunc != nil {
		_, errs := apiGroup.ValidateFunc(value, "api_group")
		if len(errs) > 0 {
			t.Fatalf("ValidateFunc rejected %q: %v", value, errs)
		}
	}

	resource := &schema.Resource{Schema: roleRef}
	if err := resource.InternalValidate(resource.Schema, true); err != nil {
		t.Fatalf("schema InternalValidate failed: %s", err)
	}
}
