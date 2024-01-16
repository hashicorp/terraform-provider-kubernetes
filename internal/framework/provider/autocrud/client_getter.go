package autocrud

import (
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
)

type KubernetesClientGetter interface {
	DynamicClient() (dynamic.Interface, error)
	DiscoveryClient() (discovery.DiscoveryInterface, error)
}
