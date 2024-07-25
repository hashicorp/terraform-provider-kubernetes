// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/openapi"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GVRFromUnstructured extracts a canonical schema.GroupVersionResource out of the resource's
// metadata by checking it against the discovery API via a RESTMapper
func GVRFromUnstructured(o *unstructured.Unstructured, m meta.RESTMapper) (schema.GroupVersionResource, error) {
	apv := o.GetAPIVersion()
	kind := o.GetKind()
	gv, err := schema.ParseGroupVersion(apv)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}
	mapping, err := m.RESTMapping(gv.WithKind(kind).GroupKind(), gv.Version)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}
	return mapping.Resource, err
}

// GVKFromTftypesObject extracts a canonical schema.GroupVersionKind out of the resource's
// metadata by checking it against the discovery API via a RESTMapper
func GVKFromTftypesObject(in *tftypes.Value, m meta.RESTMapper) (schema.GroupVersionKind, error) {
	var obj map[string]tftypes.Value
	err := in.As(&obj)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	var apv string
	var kind string
	err = obj["apiVersion"].As(&apv)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	err = obj["kind"].As(&kind)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	gv, err := schema.ParseGroupVersion(apv)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	mappings, err := m.RESTMappings(gv.WithKind(kind).GroupKind())
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	for _, m := range mappings {
		if m.GroupVersionKind.GroupVersion().String() == apv {
			return m.GroupVersionKind, nil
		}
	}
	return schema.GroupVersionKind{}, errors.New("cannot select exact GV from REST mapper")
}

// IsResourceNamespaced determines if a resource is namespaced or cluster-level
// by querying the Kubernetes discovery API
func IsResourceNamespaced(gvk schema.GroupVersionKind, m meta.RESTMapper) (bool, error) {
	rm, err := m.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return false, err
	}
	if rm.Scope.Name() == meta.RESTScopeNameNamespace {
		return true, nil
	}
	return false, nil
}

// TFTypeFromOpenAPI generates a tftypes.Type representation of a Kubernetes resource
// designated by the supplied GroupVersionKind resource id
func (ps *RawProviderServer) TFTypeFromOpenAPI(ctx context.Context, gvk schema.GroupVersionKind, status bool) (tftypes.Type, map[string]string, error) {
	var tsch tftypes.Type
	var hints map[string]string

	oapi, err := ps.getOAPIv2Foundry()
	if err != nil {
		return nil, hints, fmt.Errorf("cannot get OpenAPI foundry: %s", err)
	}
	// check if GVK is from a CRD
	crdSchema, err := ps.lookUpGVKinCRDs(ctx, gvk)
	if err != nil {
		return nil, hints, fmt.Errorf("failed to look up GVK [%s] among available CRDs: %s", gvk.String(), err)
	}
	if crdSchema != nil {
		js, err := json.Marshal(openapi.SchemaToSpec("", crdSchema.(map[string]interface{})))
		if err != nil {
			return nil, hints, fmt.Errorf("CRD schema fails to marshal into JSON: %s", err)
		}
		oapiv3, err := openapi.NewFoundryFromSpecV3(js)
		if err != nil {
			return nil, hints, err
		}
		tsch, hints, err = oapiv3.GetTypeByGVK(gvk)
		if err != nil {
			return nil, hints, fmt.Errorf("failed to generate tftypes for GVK [%s] from CRD schema: %s", gvk.String(), err)
		}
	}
	if tsch == nil {
		// Not a CRD type - look GVK up in cluster OpenAPI spec
		tsch, hints, err = oapi.GetTypeByGVK(gvk)
		if err != nil {
			return nil, hints, fmt.Errorf("cannot get resource type from OpenAPI (%s): %s", gvk.String(), err)
		}
	}
	// remove "status" attribute from resource type
	if tsch.Is(tftypes.Object{}) && !status {
		ot := tsch.(tftypes.Object)
		atts := make(map[string]tftypes.Type)
		for k, t := range ot.AttributeTypes {
			if k != "status" {
				atts[k] = t
			}
		}
		// types from CRDs only contain specific attributes
		// we need to backfill metadata and apiVersion/kind attributes
		if _, ok := atts["apiVersion"]; !ok {
			atts["apiVersion"] = tftypes.String
		}
		if _, ok := atts["kind"]; !ok {
			atts["kind"] = tftypes.String
		}
		metaType, _, err := oapi.GetTypeByGVK(openapi.ObjectMetaGVK)
		if err != nil {
			return nil, hints, fmt.Errorf("failed to generate tftypes for v1.ObjectMeta: %s", err)
		}
		atts["metadata"] = metaType.(tftypes.Object)

		tsch = tftypes.Object{AttributeTypes: atts}
	}

	return tsch, hints, nil
}

func mapRemoveNulls(in map[string]interface{}) map[string]interface{} {
	for k, v := range in {
		switch tv := v.(type) {
		case []interface{}:
			in[k] = sliceRemoveNulls(tv)
		case map[string]interface{}:
			in[k] = mapRemoveNulls(tv)
		default:
			if v == nil {
				delete(in, k)
			}
		}
	}
	return in
}

func sliceRemoveNulls(in []interface{}) []interface{} {
	s := []interface{}{}
	for _, v := range in {
		switch tv := v.(type) {
		case []interface{}:
			s = append(s, sliceRemoveNulls(tv))
		case map[string]interface{}:
			s = append(s, mapRemoveNulls(tv))
		default:
			if v != nil {
				s = append(s, v)
			}
		}
	}
	return s
}

// RemoveServerSideFields removes certain fields which get added to the
// resource after creation which would cause a perpetual diff
func RemoveServerSideFields(in map[string]interface{}) map[string]interface{} {
	// Remove "status" attribute
	delete(in, "status")

	meta := in["metadata"].(map[string]interface{})

	// Remove "uid", "creationTimestamp", "resourceVersion" as
	// they change with most resource operations
	delete(meta, "uid")
	delete(meta, "creationTimestamp")
	delete(meta, "resourceVersion")
	delete(meta, "generation")
	delete(meta, "selfLink")

	// TODO: we should be filtering API responses based on the contents of 'managedFields'
	// and only retain the attributes for which the manager is Terraform
	delete(meta, "managedFields")

	return in
}

func (ps *RawProviderServer) lookUpGVKinCRDs(ctx context.Context, gvk schema.GroupVersionKind) (interface{}, error) {
	c, err := ps.getDynamicClient()
	if err != nil {
		return nil, err
	}
	m, err := ps.getRestMapper()
	if err != nil {
		return nil, err
	}

	crd := schema.GroupKind{Group: "apiextensions.k8s.io", Kind: "CustomResourceDefinition"}
	crms, err := m.RESTMappings(crd)
	if err != nil {
		return nil, fmt.Errorf("could not extract resource version mappings for apiextensions.k8s.io.CustomResourceDefinition: %s", err)
	}
	// check  CRD versions
	for _, crm := range crms {
		crdRes, err := c.Resource(crm.Resource).List(ctx, v1.ListOptions{})
		if err != nil {
			return nil, err
		}

		for _, r := range crdRes.Items {
			spec := r.Object["spec"].(map[string]interface{})
			if spec == nil {
				continue
			}
			grp := spec["group"].(string)
			if grp != gvk.Group {
				continue
			}
			names := spec["names"]
			if names == nil {
				continue
			}
			kind := names.(map[string]interface{})["kind"]
			if kind != gvk.Kind {
				continue
			}
			ver := spec["versions"]
			if ver == nil {
				ver = spec["version"]
				if ver == nil {
					continue
				}
			}
			for _, rv := range ver.([]interface{}) {
				if rv == nil {
					continue
				}
				v := rv.(map[string]interface{})
				if v["name"] == gvk.Version {
					s, ok := v["schema"].(map[string]interface{})
					if !ok {
						return nil, nil // non-structural CRD
					}
					return s["openAPIV3Schema"], nil
				}
			}
		}
	}
	return nil, nil
}

// privateStateSchema describes the structure of the private state payload that
// Terraform can store along with the "regular" resource state state.
var privateStateSchema tftypes.Object = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"IsImported": tftypes.Bool,
}}

func getPrivateStateValue(p []byte) (ps map[string]tftypes.Value, err error) {
	if p == nil {
		err = errors.New("private state value is nil")
		return
	}
	pv, err := tftypes.ValueFromMsgPack(p, privateStateSchema)
	if err != nil {
		return
	}
	err = pv.As(&ps)
	return
}
