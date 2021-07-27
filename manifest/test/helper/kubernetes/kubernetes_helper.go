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
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	k8sretry "k8s.io/client-go/util/retry"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Helper is a Kubernetes dynamic client wrapped with a set of helper functions
// for making assertions about API resources
type Helper struct {
	client dynamic.Interface
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

	// print API server version to log output
	// also serves as validation for client config
	logAPIVersion(config)

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		//lintignore:R009
		panic(err)
	}

	return &Helper{
		client: client,
	}
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
	gvr := createGroupVersionResource("v1", "namespaces")
	_, err := k.client.Resource(gvr).Create(context.TODO(), namespace, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create namespace %q: %v", name, err)
	}
}

// DeleteNamespace deletes a namespace
func (k *Helper) DeleteNamespace(t *testing.T, name string) {
	t.Helper()

	gvr := createGroupVersionResource("v1", "namespaces")
	err := k.client.Resource(gvr).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Failed to delete namespace %q: %v", name, err)
	}
}

func createGroupVersionResource(gv, resource string) schema.GroupVersionResource {
	gvr, _ := schema.ParseGroupVersion(gv)
	return gvr.WithResource(resource)
}

// AssertNamespacedResourceExists will fail the current test if the resource doesn't exist in the
// specified namespace
func (k *Helper) AssertNamespacedResourceExists(t *testing.T, gv, resource, namespace, name string) {
	t.Helper()

	gvr := createGroupVersionResource(gv, resource)

	op := func() error {
		_, operr := k.client.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
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

	gvr := createGroupVersionResource(gv, resource)

	op := func() error {
		_, operr := k.client.Resource(gvr).Get(context.TODO(), name, metav1.GetOptions{})
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

	gvr := createGroupVersionResource(gv, resource)

	op := func() error {
		var res *unstructured.Unstructured
		var operr error
		if namespace != "" {
			res, operr = k.client.Resource(gvr).Namespace(namespace).Get(
				context.TODO(), name, metav1.GetOptions{})
		} else {
			res, operr = k.client.Resource(gvr).Get(context.TODO(),
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

	gvr := createGroupVersionResource(gv, resource)

	op := func() error {
		_, operr := k.client.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
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

	gvr := createGroupVersionResource(gv, resource)

	op := func() error {
		_, operr := k.client.Resource(gvr).Get(context.TODO(), name, metav1.GetOptions{})
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

func logAPIVersion(config *rest.Config) {
	codec := runtime.NoopEncoder{Decoder: scheme.Codecs.UniversalDecoder()}
	config.NegotiatedSerializer = serializer.NegotiatedSerializerWrapper(runtime.SerializerInfo{Serializer: codec})
	rc, err := rest.UnversionedRESTClientFor(config)
	if err != nil {
		//lintignore:R009
		panic(err)
	}
	apiVer, err := rc.Get().AbsPath("/version").DoRaw(context.Background())
	if err != nil {
		log.Printf("API version check responded with error: %s", err)
		return
	}
	var vInfo version.Info
	err = json.Unmarshal(apiVer, &vInfo)
	if err != nil {
		log.Printf("Failed to decode API version block: %s", err)
		return
	}
	log.Printf("Testing against Kubernetes API version: %s", vInfo.String())

}
