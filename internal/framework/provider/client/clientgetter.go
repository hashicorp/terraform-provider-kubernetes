package client

import (
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
)

// KubernetesClientGetter is an interface for getting client-go interfaces
type KubernetesClientGetter interface {
	DynamicClient() (dynamic.Interface, error)
	DiscoveryClient() (discovery.DiscoveryInterface, error)
}
