// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/corev1"
)

func TestConfigMapV1DataSourceSchema(t *testing.T) {
	var resp datasource.SchemaResponse
	corev1.NewConfigMapV1DataSource().Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	metadata, ok := resp.Schema.Blocks["metadata"].(datasourceschema.ListNestedBlock)
	if !ok {
		t.Fatalf("metadata block has type %T, want ListNestedBlock", resp.Schema.Blocks["metadata"])
	}

	attributes := metadata.NestedObject.Attributes
	if len(metadata.Validators) == 0 {
		t.Fatal("metadata block has no required/cardinality validators")
	}
	if _, ok := attributes["generate_name"]; ok {
		t.Fatal("metadata unexpectedly exposes generate_name")
	}

	name, ok := attributes["name"].(datasourceschema.StringAttribute)
	if !ok || !name.Required {
		t.Fatalf("metadata.name = %#v, want a required string attribute", attributes["name"])
	}

	namespace, ok := attributes["namespace"].(datasourceschema.StringAttribute)
	if !ok || !namespace.Optional || !namespace.Computed {
		t.Fatalf("metadata.namespace = %#v, want optional+computed string attribute", attributes["namespace"])
	}

	for _, name := range []string{"annotations", "labels"} {
		attribute, ok := attributes[name].(datasourceschema.MapAttribute)
		if !ok || !attribute.Optional || !attribute.Computed {
			t.Errorf("metadata.%s = %#v, want optional+computed map attribute", name, attributes[name])
		}
	}
}
