// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	authv1 "k8s.io/api/authentication/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
)

// Flatteners

func flattenTokenRequestV1Spec(in authv1.TokenRequestSpec, d *schema.ResourceData, meta interface{}) ([]interface{}, error) {
	att := make(map[string]interface{})

	att["audiences"] = in.Audiences

	if in.BoundObjectRef != nil {
		att["bound_object_ref"] = flattenBoundObjectReference(*in.BoundObjectRef, d, meta)
	}

	if in.ExpirationSeconds != nil {
		att["expiration_seconds"] = int(*in.ExpirationSeconds)
	}

	return []interface{}{att}, nil
}

func flattenBoundObjectReference(in authv1.BoundObjectReference, d *schema.ResourceData, meta interface{}) []interface{} {
	att := make(map[string]interface{})

	att["api_version"] = in.APIVersion

	att["kind"] = in.Kind

	att["name"] = in.Name

	att["uid"] = in.UID

	return []interface{}{att}
}

// Expanders

func expandTokenRequestV1Spec(p []interface{}) *authv1.TokenRequestSpec {
	obj := &authv1.TokenRequestSpec{}
	if len(p) == 0 || p[0] == nil {
		return obj
	}
	in := p[0].(map[string]interface{})

	if v, ok := in["audiences"].([]interface{}); ok && len(v) > 0 {
		obj.Audiences = expandStringSlice(v)
	}

	obj.BoundObjectRef = expandBoundObjectReference(in["bound_object_ref"].([]interface{}))

	if v, ok := in["expiration_seconds"].(int); v != 0 && ok {
		obj.ExpirationSeconds = ptr.To(int64(v))
	}

	return obj
}

func expandBoundObjectReference(p []interface{}) *authv1.BoundObjectReference {
	obj := &authv1.BoundObjectReference{}
	if len(p) == 0 || p[0] == nil {
		return nil
	}
	in := p[0].(map[string]interface{})

	if v, ok := in["api_version"]; ok {
		obj.APIVersion = v.(string)
	}

	if v, ok := in["kind"]; ok {
		obj.Kind = v.(string)
	}

	if v, ok := in["name"]; ok {
		obj.Name = v.(string)
	}

	if v, ok := in["uid"]; ok {
		obj.UID = types.UID(v.(string))
	}

	return obj
}
