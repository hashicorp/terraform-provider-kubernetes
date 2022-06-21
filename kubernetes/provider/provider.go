package provider

import (
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	aggregator "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
)

type KubeClientsets interface {
	MainClientset() (*kubernetes.Clientset, error)
	AggregatorClientset() (*aggregator.Clientset, error)
	DynamicClient() (dynamic.Interface, error)
	DiscoveryClient() (discovery.DiscoveryInterface, error)
}

type Meta interface {
	KubeClientsets

	IgnoredAnnotations() []string
	IgnoredLabels() []string
}
