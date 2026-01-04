// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"

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

	if oapiV3, err := ps.getOAPIv3Foundry(gvk.GroupVersion()); err == nil {
		return getTypeByGVK(gvk, status, oapiV3, "OpenAPI v3")
	}

	var tfo tftypes.Object
	var hints map[string]string

	// check if GVK is from a CRD
	oapiV2, err := ps.getOAPIv2Foundry()
	if err != nil {
		return nil, hints, fmt.Errorf("cannot get OpenAPI foundry: %w", err)
	}

	if crdSchema, err := ps.lookUpGVKinCRDs(ctx, gvk); err != nil {
		return nil, hints, fmt.Errorf("failed to look up GVK (%v) among available CRDs: %w", gvk, err)
	} else if crdSchema != nil {
		js, err := json.Marshal(openapi.CRDSchemaToSpec(gvk, crdSchema.(map[string]interface{})))
		if err != nil {
			return nil, hints, fmt.Errorf("CRD schema fails to marshal into JSON: %w", err)
		}
		oapiV3, err := openapi.NewFoundryFromSpecV3(js)
		if err != nil {
			return nil, hints, err
		}
		tfo, hints, err = getTypeByGVK(gvk, status, oapiV3, "CRD schema")
		if err != nil {
			return nil, hints, err
		}
	} else {
		// Not a CRD type - look GVK up in cluster OpenAPI v2 spec
		tfo, hints, err = getTypeByGVK(gvk, status, oapiV2, "OpenAPI v2")
		if err != nil {
			return nil, hints, err
		}
	}

	// types from CRDs only contain specific attributes
	// we need to backfill metadata and apiVersion/kind attributes
	atts := maps.Clone(tfo.AttributeTypes)
	if _, ok := atts["apiVersion"]; !ok {
		atts["apiVersion"] = tftypes.String
	}
	if _, ok := atts["kind"]; !ok {
		atts["kind"] = tftypes.String
	}
	metaType, _, err := oapiV2.GetTypeByGVK(openapi.ObjectMetaGVK)
	if err != nil {
		return nil, hints, fmt.Errorf("failed to generate tftypes for v1.ObjectMeta: %w", err)
	}
	atts["metadata"] = metaType.(tftypes.Object)
	tfo.AttributeTypes = atts

	return tfo, hints, nil
}

// getTypeByGVK retrieves a terraform type from an OpenAPI fondry given a GVK, verifies it's an Object and
// optionally removes the "status" attribute.
func getTypeByGVK(gvk schema.GroupVersionKind, status bool, oapi openapi.Foundry, source string) (tftypes.Object, map[string]string, error) {
	tft, hints, err := oapi.GetTypeByGVK(gvk)
	if err != nil {
		return tftypes.Object{}, hints, fmt.Errorf("cannot get resource type from %s (%v): %w", source, gvk, err)
	}
	if !tft.Is(tftypes.Object{}) {
		return tftypes.Object{}, hints, fmt.Errorf("did not resolve into an object type (%v)", gvk)
	}
	tfo := tft.(tftypes.Object)
	if !status {
		if _, present := tfo.AttributeTypes["status"]; present {
			tfo.AttributeTypes = maps.Clone(tfo.AttributeTypes)
			delete(tfo.AttributeTypes, "status")
		}
	}

	return tfo, hints, err
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
	// check CRD versions
	crds, err := ps.fetchCRDs(ctx)
	if err != nil {
		return nil, err
	}

	for _, r := range crds {
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
	return nil, nil
}

func (ps *RawProviderServer) fetchCRDs(ctx context.Context) ([]unstructured.Unstructured, error) {
	return ps.crds.Get(func() ([]unstructured.Unstructured, error) {
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

		var crds []unstructured.Unstructured
		for _, crm := range crms {
			crdRes, err := c.Resource(crm.Resource).List(ctx, v1.ListOptions{})
			if err != nil {
				return nil, err
			}

			crds = append(crds, crdRes.Items...)
		}

		return crds, nil
	})
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
