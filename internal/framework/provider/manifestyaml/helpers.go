// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestyaml

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/restmapper"
	"sigs.k8s.io/yaml"
)

// decodeYAML parses a single-document YAML string into an unstructured object.
func decodeYAML(y string) (*unstructured.Unstructured, error) {
	j, err := yaml.YAMLToJSON([]byte(y))
	if err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}
	obj := &unstructured.Unstructured{}
	if err := obj.UnmarshalJSON(j); err != nil {
		return nil, fmt.Errorf("cannot decode manifest object: %w", err)
	}
	if obj.GetKind() == "" || obj.GetAPIVersion() == "" {
		return nil, fmt.Errorf("manifest must set both apiVersion and kind")
	}
	if obj.GetName() == "" {
		return nil, fmt.Errorf("manifest must set metadata.name (generateName is not supported)")
	}
	return obj, nil
}

// resolveGVR maps the object's GVK to a GroupVersionResource via cluster discovery.
// A DeferredDiscoveryRESTMapper is used so a CRD registered earlier in the same
// apply can be discovered on retry (RFC-011 §6.2).
func resolveGVR(clients kubernetes.KubeClientsets, obj *unstructured.Unstructured) (schema.GroupVersionResource, bool, error) {
	disco, err := clients.DiscoveryClient()
	if err != nil {
		return schema.GroupVersionResource{}, false, err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(disco))
	gvk := obj.GroupVersionKind()

	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if meta.IsNoMatchError(err) {
		// The kind may have just been installed (e.g. a CRD in this apply): reset the
		// discovery cache and try once more.
		mapper.Reset()
		mapping, err = mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	}
	if err != nil {
		return schema.GroupVersionResource{}, false, err
	}
	namespaced := mapping.Scope.Name() == meta.RESTScopeNameNamespace
	return mapping.Resource, namespaced, nil
}

// buildID returns the stable resource ID: apiVersion=<>,kind=<>,namespace=<>,name=<>.
func buildID(obj *unstructured.Unstructured) string {
	return fmt.Sprintf("apiVersion=%s,kind=%s,namespace=%s,name=%s",
		obj.GetAPIVersion(), obj.GetKind(), obj.GetNamespace(), obj.GetName())
}

// importIdentity is the parsed key=value import ID.
type importIdentity struct {
	APIVersion string
	Kind       string
	Namespace  string
	Name       string
}

// parseImportID parses "apiVersion=apps/v1,kind=Deployment,namespace=default,name=web".
// apiVersion may itself contain "/", so a positional ID is ambiguous; key=value is required.
func parseImportID(id string) (importIdentity, error) {
	var out importIdentity
	for _, part := range strings.Split(id, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return out, fmt.Errorf("expected key=value pairs, got %q", part)
		}
		key, val := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
		switch key {
		case "apiVersion":
			out.APIVersion = val
		case "kind":
			out.Kind = val
		case "namespace":
			out.Namespace = val
		case "name":
			out.Name = val
		default:
			return out, fmt.Errorf("unknown import key %q (allowed: apiVersion, kind, namespace, name)", key)
		}
	}
	if out.APIVersion == "" || out.Kind == "" || out.Name == "" {
		return out, fmt.Errorf("import ID must include apiVersion, kind and name")
	}
	return out, nil
}

// unstructuredForImport builds a minimal object from the parsed import identity so
// the GVR can be resolved and a Get performed.
func unstructuredForImport(id importIdentity) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(id.APIVersion)
	obj.SetKind(id.Kind)
	obj.SetNamespace(id.Namespace)
	obj.SetName(id.Name)
	return obj
}
