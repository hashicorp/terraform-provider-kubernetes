package autocrud

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
)

func Create(ctx context.Context, clientGetter KubernetesClientGetter, apiVersion, kind string, model any) error {
	client, err := clientGetter.DynamicClient()
	if err != nil {
		return err
	}
	discoveryClient, err := clientGetter.DiscoveryClient()
	if err != nil {
		return err
	}
	agr, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		return err
	}
	gvk := k8sschema.FromAPIVersionAndKind(apiVersion, kind)
	restMapper := restmapper.NewDiscoveryRESTMapper(agr)
	mapping, err := restMapper.RESTMapping(gvk.GroupKind(), apiVersion)
	if err != nil {
		return err
	}

	manifest := ExpandModel(model)

	var resourceInterface dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		metadata := manifest["metadata"].(map[string]interface{})
		namespace := "default"
		if v, ok := metadata["namespace"].(string); ok && v != "" {
			namespace = v
		}
		resourceInterface = client.Resource(mapping.Resource).Namespace(namespace)
	} else {
		resourceInterface = client.Resource(mapping.Resource)
	}

	obj := unstructured.Unstructured{}
	obj.SetUnstructuredContent(manifest)
	res, err := resourceInterface.Create(ctx, &obj,
		v1.CreateOptions{
			// FIXME this should be configurable
			FieldManager: "terraform",
		},
	)
	if err != nil {
		return err
	}

	responseManifest := res.UnstructuredContent()
	responseManifest["id"] = createID(responseManifest)
	FlattenManifest(responseManifest, model)
	return nil
}
