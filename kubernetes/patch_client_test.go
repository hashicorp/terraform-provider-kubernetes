package kubernetes

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/dynamic/fake"
)

func TestNewPatchClient(t *testing.T) {
	scheme := runtime.NewScheme()
	namespace := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name":   "foobar",
				"labels": map[string]interface{}{},
			},
		},
	}
	client := k8sfake.NewSimpleDynamicClient(scheme, namespace)
	foo := map[string]interface{}{
		"api_version":      "v1",
		"kind":             "Namespaces",
		"namespace_scoped": false,
		"namespace":        "",
		"name":             "foobar",
	}
	getFn := func(key string) interface{} {
		return foo[key]
	}

	p, err := newPatchClient(getFn, client, "/metadata/labels/foo", "foo", "bar")
	if err != nil {
		t.Fatalf("Unable to create patchClient: %v", err)
	}

	ctx := context.TODO()
	res, err := p.ReadResource(ctx)
	if err != nil {
		t.Fatalf("Unable to read resource: %v", err)
	}
	t.Logf("Object before add: %v", res.Object)

	_, err = p.Create(ctx)
	if err != nil {
		t.Fatalf("Unable to add label: %v", err)
	}

	res, err = p.ReadResource(ctx)
	if err != nil {
		t.Fatalf("Unable to read resource after add: %v", err)
	}
	t.Logf("Object after add: %v", res.Object)

	t.Fail()

	// fakeInput := map[string]string{}
	// getFn := func(key string) interface{} {
	// 	return fakeInput[key]
	// }

	// newPatchClient(getFn, client, "/metadata/labels/baz", "foo", "bar")
}
