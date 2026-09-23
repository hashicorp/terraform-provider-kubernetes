package kubernetes

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestDataSourceKubernetesServiceV1_schema(t *testing.T) {
	ds := dataSourceKubernetesServiceV1("")
	if ds == nil {
		t.Fatal("Expected non-nil data source")
	}

	if _, ok := ds.Schema["label_selector"]; !ok {
		t.Fatal("label_selector field missing from schema")
	}
	if _, ok := ds.Schema["field_selector"]; !ok {
		t.Fatal("field_selector field missing from schema")
	}
}

func TestDataSourceKubernetesServiceV1_labelSelector(t *testing.T) {
	ds := dataSourceKubernetesServiceV1("")
	if ds == nil {
		t.Fatal("Expected non-nil data source")
	}

	ls := ds.Schema["label_selector"]
	if ls.Type != schema.TypeString {
		t.Fatalf("Expected label_selector to be TypeString, got %T", ls.Type)
	}
	if !ls.Optional {
		t.Fatal("Expected label_selector to be optional")
	}
}

func TestDataSourceKubernetesServiceV1_fieldSelector(t *testing.T) {
	ds := dataSourceKubernetesServiceV1("")
	if ds == nil {
		t.Fatal("Expected non-nil data source")
	}

	fs := ds.Schema["field_selector"]
	if fs.Type != schema.TypeString {
		t.Fatalf("Expected field_selector to be TypeString, got %T", fs.Type)
	}
	if !fs.Optional {
		t.Fatal("Expected field_selector to be optional")
	}
}
