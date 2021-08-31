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
	for _, s := range samples {
		gotGvk, gotName, gotNamespace, gotErr := parseImportID(s.ID)
		if gotErr != nil {
			if gotErr.Error() == s.Err.Error() {
				continue
			}
			t.Fatal(gotErr.Error())
		}
		if s.GVK != gotGvk {
			t.Log("GVK (got / wanted):", gotGvk, s.GVK)
			t.Fail()
		}
		if s.Name != gotName {
			t.Log("Name (got / wanted):", gotName, s.Name)
			t.Fail()
		}
		if s.Namespace != gotNamespace {
			t.Log("Namespace (got / wanted):", gotNamespace, s.Namespace)
			t.Fail()
		}
	}
}
