package client

import (
	"fmt"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

type kubernetesClientGetter struct {
	config            *rest.Config
	dynamicClient     dynamic.Interface
	discoveryClient   discovery.DiscoveryInterface
	ignoreLabels      []string
	ignoreAnnotations []string
}

func NewKubernetesClientGetter(config *rest.Config, ignoreLabels, ignoreAnnotations []string) KubernetesClientGetter {
	return &kubernetesClientGetter{
		config:            config,
		ignoreLabels:      ignoreLabels,
		ignoreAnnotations: ignoreAnnotations,
	}
}

func (k kubernetesClientGetter) DynamicClient() (dynamic.Interface, error) {
	if k.dynamicClient != nil {
		return k.dynamicClient, nil
	}

	if k.config != nil {
		kc, err := dynamic.NewForConfig(k.config)
		if err != nil {
			return nil, fmt.Errorf("failed to configure dynamic client: %s", err)
		}
		k.dynamicClient = kc
	}
	return k.dynamicClient, nil
}

func (k kubernetesClientGetter) DiscoveryClient() (discovery.DiscoveryInterface, error) {
	if k.discoveryClient != nil {
		return k.discoveryClient, nil
	}

	if k.config != nil {
		kc, err := discovery.NewDiscoveryClientForConfig(k.config)
		if err != nil {
			return nil, fmt.Errorf("failed to configure discovery client: %s", err)
		}
		k.discoveryClient = kc
	}
	return k.discoveryClient, nil
}

func (k kubernetesClientGetter) IgnoreLabels() []string {
	return k.ignoreLabels
}

func (k kubernetesClientGetter) IgnoreAnnotations() []string {
	return k.ignoreAnnotations
}
