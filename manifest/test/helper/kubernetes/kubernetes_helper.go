// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance
// +build acceptance

package kubernetes

import (
	"context"
	"encoding/json"
	"log"
	"testing"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	k8sretry "k8s.io/client-go/util/retry"
	"k8s.io/kubectl/pkg/scheme"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Helper is a Kubernetes dynamic client wrapped with a set of helper functions
// for making assertions about API resources
type Helper struct {
	dynClient  dynamic.Interface
	restClient *rest.RESTClient
}

type apiVersionResponse struct {
}

// NewHelper initializes a new Kubernetes client
func NewHelper() *Helper {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{}
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
	if err != nil {
		//lintignore:R009
		panic(err)
	}

	if config == nil {
		config = &rest.Config{}
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		//lintignore:R009
		panic(err)
	}

	codec := runtime.NoopEncoder{Decoder: scheme.Codecs.UniversalDecoder()}
	config.NegotiatedSerializer = serializer.NegotiatedSerializerWrapper(runtime.SerializerInfo{Serializer: codec})

	rc, err := rest.UnversionedRESTClientFor(config)
	if err != nil {
		//lintignore:R009
		panic(err)
	}

	h := Helper{
		dynClient:  client,
		restClient: rc,
	}
	// print API server version to log output
	// also serves as validation for client config
	h.logAPIVersion()

	return &h
}

// CreateNamespace creates a new namespace
func (k *Helper) CreateNamespace(t *testing.T, name string) {
	t.Helper()

	namespace := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name": name,
			},
		},
	}
	gvr := NewGroupVersionResource("v1", "namespaces")
	_, err := k.dynClient.Resource(gvr).Create(context.TODO(), namespace, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create namespace %q: %v", name, err)
	}
}

// CreateConfigMap creates a new ConfigMap
func (k *Helper) CreateConfigMap(t *testing.T, name string, namespace string, data map[string]interface{}) {
	t.Helper()

	cfgmap := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"data": data,
		},
	}
	gvr := NewGroupVersionResource("v1", "configmaps")
	_, err := k.dynClient.Resource(gvr).Namespace(namespace).Create(context.TODO(), cfgmap, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create configmap %q/%q: %v", namespace, name, err)
	}
}

// DeleteResource deletes a resource referred to by the name and GVK
func (k *Helper) DeleteResource(t *testing.T, name string, gvr schema.GroupVersionResource) {
	t.Helper()

	err := k.dynClient.Resource(gvr).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Failed to delete resource %q: %v", name, err)
	}
}

// DeleteResource deletes a namespaced resource referred to by the name and GVK
func (k *Helper) DeleteNamespacedResource(t *testing.T, name string, namespace string, gvr schema.GroupVersionResource) {
	t.Helper()

	err := k.dynClient.Resource(gvr).Namespace(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Failed to delete resource \"%s/%s\": %v", namespace, name, err)
	}
}

func NewGroupVersionResource(gv, resource string) schema.GroupVersionResource {
	gvr, _ := schema.ParseGroupVersion(gv)
	return gvr.WithResource(resource)
}

// AssertNamespacedResourceExists will fail the current test if the resource doesn't exist in the
// specified namespace
func (k *Helper) AssertNamespacedResourceExists(t *testing.T, gv, resource, namespace, name string) {
	t.Helper()

	gvr := NewGroupVersionResource(gv, resource)

	op := func() error {
		_, operr := k.dynClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		return operr
	}

	err := k8sretry.OnError(k8sretry.DefaultBackoff, func(e error) bool {
		if !isErrorRetriable(e) {
			t.Logf("Error not retriable: %s", e)
			return false
		}
		return !errors.IsNotFound(e)
	}, op)

	if err != nil {
		t.Errorf("Error when trying to get resource %s/%s: %v", namespace, name, err)
	}
}

// AssertResourceExists will fail the current test if the resource doesn't exist
func (k *Helper) AssertResourceExists(t *testing.T, gv, resource, name string) {
	t.Helper()

	gvr := NewGroupVersionResource(gv, resource)

	op := func() error {
		_, operr := k.dynClient.Resource(gvr).Get(context.TODO(), name, metav1.GetOptions{})
		return operr
	}
	err := k8sretry.OnError(k8sretry.DefaultBackoff, func(e error) bool {
		if !isErrorRetriable(e) {
			t.Logf("Error not retriable: %s", e)
			return false
		}
		return !errors.IsNotFound(e)
	}, op)
	if err != nil {
		t.Errorf("Error when trying to get resource %s: %v", name, err)
	}
}

// AssertResourceGeneration will fail if the generation does not match
func (k *Helper) AssertResourceGeneration(t *testing.T, gv, resource, namespace, name string, generation int64) {
	t.Helper()

	gvr := NewGroupVersionResource(gv, resource)

	op := func() error {
		var res *unstructured.Unstructured
		var operr error
		if namespace != "" {
			res, operr = k.dynClient.Resource(gvr).Namespace(namespace).Get(
				context.TODO(), name, metav1.GetOptions{})
		} else {
			res, operr = k.dynClient.Resource(gvr).Get(context.TODO(),
				name, metav1.GetOptions{})
		}

		if operr != nil {
			return operr
		}

		g := res.GetGeneration()
		if g != generation {
			t.Errorf("Expected generation to be %v actual %v", generation, g)
		}
		return nil
	}
	err := k8sretry.OnError(k8sretry.DefaultBackoff, func(e error) bool {
		if !isErrorRetriable(e) {
			t.Logf("Error not retriable: %s", e)
			return false
		}
		return !errors.IsNotFound(e)
	}, op)
	if err != nil {
		t.Errorf("Error when trying to get resource %s: %v", name, err)
	}

}

// AssertNamespacedResourceDoesNotExist fails the test if the resource still exists in the namespace specified
func (k *Helper) AssertNamespacedResourceDoesNotExist(t *testing.T, gv, resource, namespace, name string) {
	t.Helper()

	gvr := NewGroupVersionResource(gv, resource)

	op := func() error {
		_, operr := k.dynClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if errors.IsNotFound(operr) {
			return nil
		}
		return operr
	}
	err := k8sretry.OnError(k8sretry.DefaultBackoff, func(e error) bool {
		if !isErrorRetriable(e) {
			t.Logf("Error not retriable: %s", e)
			return false
		}
		return e != nil
	}, op)

	if err != nil {
		t.Errorf("Resource %s/%s still exists", namespace, name)
	}
}

// AssertResourceDoesNotExist fails the test if the resource still exists
func (k *Helper) AssertResourceDoesNotExist(t *testing.T, gv, resource, name string) {
	t.Helper()

	gvr := NewGroupVersionResource(gv, resource)

	op := func() error {
		_, operr := k.dynClient.Resource(gvr).Get(context.TODO(), name, metav1.GetOptions{})
		if errors.IsNotFound(operr) {
			return nil
		}
		return operr
	}
	err := k8sretry.OnError(k8sretry.DefaultBackoff, func(e error) bool {
		if !isErrorRetriable(e) {
			t.Logf("Error not retriable: %s", e)
			return false
		}
		return e != nil
	}, op)
	if err != nil {
		t.Errorf("Resource %s still exists", name)
	}
}

func isErrorRetriable(e error) bool {
	if errors.IsBadRequest(e) ||
		errors.IsForbidden(e) ||
		errors.IsTimeout(e) ||
		errors.IsInvalid(e) ||
		errors.IsUnauthorized(e) ||
		errors.IsServiceUnavailable(e) ||
		errors.IsInternalError(e) {
		return false
	}
	return true
}

func (k *Helper) logAPIVersion() {
	log.Printf("Testing against Kubernetes API version: %s", k.ClusterVersion().String())
}

func (k *Helper) ClusterVersion() (vInfo version.Info) {
	apiVer, err := k.restClient.Get().AbsPath("/version").DoRaw(context.Background())
	if err != nil {
		log.Printf("API version check responded with error: %s", err)
		return
	}
	err = json.Unmarshal(apiVer, &vInfo)
	if err != nil {
		log.Printf("Failed to decode API version block: %s", err)
	}
	return
}
