package client

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
)

func ResourceInterface(clientGetter KubernetesClientGetter, kind, apiVersion, namespace string) (dynamic.ResourceInterface, error) {
	client, err := clientGetter.DynamicClient()
	if err != nil {
		return nil, err
	}
	discoveryClient, err := clientGetter.DiscoveryClient()
	if err != nil {
		return nil, err
	}
	agr, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		return nil, err
	}
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return nil, err
	}
	restMapper := restmapper.NewDiscoveryRESTMapper(agr)
	mapping, err := restMapper.RESTMapping(gv.WithKind(kind).GroupKind(), gv.Version)
	if err != nil {
		return nil, err
	}

	if namespace == "" {
		return client.Resource(mapping.Resource), nil
	}
	return client.Resource(mapping.Resource).Namespace(namespace), nil
}
