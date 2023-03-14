// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	v1 "k8s.io/api/authentication/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Flatteners

func flattenTokenRequestSpec(in v1.TokenRequestSpec, d *schema.ResourceData, meta interface{}) ([]interface{}, error) {
	att := make(map[string]interface{})

	att["audiences"] = in.Audiences

	if in.BoundObjectRef != nil {
		bndObjRef, err := flattenBoundObjectReference(*in.BoundObjectRef, d, meta)
		if err != nil {
			return nil, err
		}
		att["boundobjectref"] = bndObjRef
	}

	att["expirationseconds"] = int(*in.ExpirationSeconds)

	return []interface{}{att}, nil
}

func flattenBoundObjectReference(in v1.BoundObjectReference, d *schema.ResourceData, meta interface{}) ([]interface{}, error) {
	att := make(map[string]interface{})

	att["apiversion"] = in.APIVersion

	att["kind"] = in.Kind

	att["name"] = in.Name

	att["uid"] = in.UID

	return []interface{}{att}, nil
}

// Expanders

func expandTokenRequestSpec(p []interface{}) *v1.TokenRequestSpec {
	obj := &v1.TokenRequestSpec{}
	if len(p) == 0 || p[0] == nil {
		return obj
	}
	in := p[0].(map[string]interface{})

	if v, ok := in["audiences"].([]interface{}); ok && len(v) > 0 {
		obj.Audiences = expandStringSlice(v)
	}

	bdObjRef, err := expandBoundObjectReference(in["bound_object_ref"].([]interface{}))
	if err != nil {
		return obj
	}
	obj.BoundObjectRef = bdObjRef

	if v, ok := in["expiration_seconds"].(int); ok {
		if v != 0 {
			obj.ExpirationSeconds = ptrToInt64(int64(v))
		}
	}

	return obj
}

func expandBoundObjectReference(p []interface{}) (*v1.BoundObjectReference, error) {
	obj := &v1.BoundObjectReference{}
	if len(p) == 0 || p[0] == nil {
		return nil, nil
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

	return obj, nil
}
