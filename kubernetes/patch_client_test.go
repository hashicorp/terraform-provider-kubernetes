package kubernetes

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/dynamic/fake"
)

func TestPatchClient(t *testing.T) {
	pt := testPatchClientHelper{}
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
	config := map[string]interface{}{
		"api_version":      "v1",
		"kind":             "Namespaces",
		"namespace_scoped": false,
		"namespace":        "",
		"name":             "foobar",
	}
	getFn := func(key string) interface{} {
		return config[key]
	}

	p, err := newPatchClient(getFn, client, "/metadata/labels/foo", "foo", "bar")
	pt.expectNoError(t, err)

	ctx := context.TODO()
	res, err := p.ReadResource(ctx)
	pt.expectNoError(t, err)
	pt.expectLabel(t, res, false, "", "")

	_, err = p.Create(ctx)
	pt.expectNoError(t, err)

	res, err = p.ReadResource(ctx)
	pt.expectNoError(t, err)
	pt.expectLabel(t, res, true, "foo", "bar")

	p, err = newPatchClient(getFn, client, "/metadata/labels/foo", "foo", "baz")
	pt.expectNoError(t, err)

	_, err = p.Update(ctx)
	pt.expectNoError(t, err)

	res, err = p.ReadResource(ctx)
	pt.expectNoError(t, err)
	pt.expectLabel(t, res, true, "foo", "baz")

	_, err = p.Delete(ctx)
	pt.expectNoError(t, err)

	res, err = p.ReadResource(ctx)
	pt.expectNoError(t, err)
	pt.expectLabel(t, res, false, "", "")
}

func TestPatchClientNamespaced(t *testing.T) {
	pt := testPatchClientHelper{}
	scheme := runtime.NewScheme()
	namespace := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      "foobar",
				"namespace": "ze-namespace",
				"labels":    map[string]interface{}{},
			},
		},
	}
	client := k8sfake.NewSimpleDynamicClient(scheme, namespace)
	config := map[string]interface{}{
		"api_version":      "apps/v1",
		"kind":             "Deployments",
		"namespace_scoped": true,
		"namespace":        "ze-namespace",
		"name":             "foobar",
	}
	getFn := func(key string) interface{} {
		return config[key]
	}

	p, err := newPatchClient(getFn, client, "/metadata/labels/foo", "foo", "bar")
	pt.expectNoError(t, err)

	ctx := context.TODO()
	res, err := p.ReadResource(ctx)
	pt.expectNoError(t, err)
	pt.expectLabel(t, res, false, "", "")

	_, err = p.Create(ctx)
	pt.expectNoError(t, err)

	res, err = p.ReadResource(ctx)
	pt.expectNoError(t, err)
	pt.expectLabel(t, res, true, "foo", "bar")

	p, err = newPatchClient(getFn, client, "/metadata/labels/foo", "foo", "baz")
	pt.expectNoError(t, err)

	_, err = p.Update(ctx)
	pt.expectNoError(t, err)

	res, err = p.ReadResource(ctx)
	pt.expectNoError(t, err)
	pt.expectLabel(t, res, true, "foo", "baz")

	_, err = p.Delete(ctx)
	pt.expectNoError(t, err)

	res, err = p.ReadResource(ctx)
	pt.expectNoError(t, err)
	pt.expectLabel(t, res, false, "", "")
}

func TestLabelClient(t *testing.T) {
	pt := testPatchClientHelper{}
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
	config := map[string]interface{}{
		"api_version":      "v1",
		"kind":             "Namespaces",
		"namespace_scoped": false,
		"namespace":        "",
		"name":             "foobar",
		"label_key":        "foo",
		"label_value":      "bar",
	}
	getFn := func(key string) interface{} {
		return config[key]
	}

	l, err := newLabelClient(getFn, client)
	pt.expectNoError(t, err)

	ctx := context.TODO()
	res, err := l.ReadResource(ctx)
	pt.expectNoError(t, err)
	pt.expectLabel(t, res, false, "", "")

	err = l.Create(ctx)
	pt.expectNoError(t, err)

	res, err = l.ReadResource(ctx)
	pt.expectNoError(t, err)
	pt.expectLabel(t, res, true, "foo", "bar")

	config["label_value"] = "baz"
	l, err = newLabelClient(getFn, client)
	pt.expectNoError(t, err)

	err = l.Update(ctx)
	pt.expectNoError(t, err)

	res, err = l.ReadResource(ctx)
	pt.expectNoError(t, err)
	pt.expectLabel(t, res, true, "foo", "baz")

	err = l.Delete(ctx)
	pt.expectNoError(t, err)

	res, err = l.ReadResource(ctx)
	pt.expectNoError(t, err)
	pt.expectLabel(t, res, false, "", "")
}

func TestLabelClientNamespaced(t *testing.T) {
	pt := testPatchClientHelper{}
	scheme := runtime.NewScheme()
	namespace := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      "foobar",
				"namespace": "ze-namespace",
				"labels":    map[string]interface{}{},
			},
		},
	}
	client := k8sfake.NewSimpleDynamicClient(scheme, namespace)
	config := map[string]interface{}{
		"api_version":      "apps/v1",
		"kind":             "Deployments",
		"namespace_scoped": true,
		"namespace":        "ze-namespace",
		"name":             "foobar",
		"label_key":        "foo",
		"label_value":      "bar",
	}
	getFn := func(key string) interface{} {
		return config[key]
	}

	l, err := newLabelClient(getFn, client)
	pt.expectNoError(t, err)

	ctx := context.TODO()
	res, err := l.ReadResource(ctx)
	pt.expectNoError(t, err)
	pt.expectLabel(t, res, false, "", "")

	err = l.Create(ctx)
	pt.expectNoError(t, err)

	res, err = l.ReadResource(ctx)
	pt.expectNoError(t, err)
	pt.expectLabel(t, res, true, "foo", "bar")

	config["label_value"] = "baz"
	l, err = newLabelClient(getFn, client)
	pt.expectNoError(t, err)

	err = l.Update(ctx)
	pt.expectNoError(t, err)

	res, err = l.ReadResource(ctx)
	pt.expectNoError(t, err)
	pt.expectLabel(t, res, true, "foo", "baz")

	err = l.Delete(ctx)
	pt.expectNoError(t, err)

	res, err = l.ReadResource(ctx)
	pt.expectNoError(t, err)
	pt.expectLabel(t, res, false, "", "")
}

type testPatchClientHelper struct{}

func (pt *testPatchClientHelper) expectNoError(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("received error it wasn't expected: %v", err)
	}
}

func (pt *testPatchClientHelper) expectError(t *testing.T, err error) {
	if err == nil {
		t.Fatal("received no error it was expected")
	}
}

func (pt *testPatchClientHelper) expectLabel(t *testing.T, res *unstructured.Unstructured, expectedExists bool, key string, expectedValue string) {
	t.Helper()

	labels := res.GetLabels()
	value, ok := labels[key]
	if ok != expectedExists {
		labelExists := "exist"
		if !expectedExists {
			labelExists = "not exist"
		}
		t.Fatalf("expected label to %s", labelExists)
	}

	if value != expectedValue {
		t.Fatalf("expected value %q but received %q", expectedValue, value)
	}
}
