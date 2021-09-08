package provider

import (
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestParseImportID(t *testing.T) {
	samples := []struct {
		ID        string
		GVK       schema.GroupVersionKind
		Name      string
		Namespace string
		Err       error
	}{
		{
			ID:        "apiVersion=v1,kind=ConfigMap,namespace=default,name=test",
			GVK:       schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"},
			Name:      "test",
			Namespace: "default",
			Err:       nil,
		},
		{
			ID:        "apiVersion=rbac.authorization.k8s.io/v1,kind=ClusterRole,name=test",
			GVK:       schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"},
			Name:      "test",
			Namespace: "",
			Err:       nil,
		},
		{
			ID:        "apiVersion=apps/v1,kind=Deployment,namespace=foo,name=bar",
			GVK:       schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"},
			Name:      "bar",
			Namespace: "foo",
			Err:       nil,
		},
		{
			ID:  "foobar",
			Err: fmt.Errorf("invalid format for import ID [%s]\nExpected format is: apiVersion=<value>,kind=<value>,name=<value>[,namespace=<value>]", "foobar"),
		},
	}
	for _, expected := range samples {
		actualGvk, actualName, actualNamespace, actualErr := parseImportID(expected.ID)
		if actualErr != nil {
			if actualErr.Error() == expected.Err.Error() {
				continue
			}
			t.Fatal(actualErr.Error())
		}
		if expected.GVK != actualGvk {
			t.Log("GVK (actual / wanted):", actualGvk, expected.GVK)
			t.Fail()
		}
		if expected.Name != actualName {
			t.Log("Name (actual / wanted):", actualName, expected.Name)
			t.Fail()
		}
		if expected.Namespace != actualNamespace {
			t.Log("Namespace (actual / wanted):", actualNamespace, expected.Namespace)
			t.Fail()
		}
	}
}
