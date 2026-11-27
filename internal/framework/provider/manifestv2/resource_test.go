// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestv2

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// yaml_body holds one object. Null values mean "not set".
func TestDecodeYAML(t *testing.T) {
	manifest, err := decodeYAML("# a comment\n---\napiVersion: v1\nkind: Service\nmetadata:\n  name: web\n  annotations:\nspec:\n  ports:\n  - port: 80\n    name:\n")
	want := map[string]any{"apiVersion": "v1", "kind": "Service", "metadata": map[string]any{"name": "web"},
		"spec": map[string]any{"ports": []any{map[string]any{"port": int64(80)}}}}
	if err != nil || !reflect.DeepEqual(manifest, want) {
		t.Fatalf("got %v (error %v)", manifest, err)
	}
	if _, err := decodeYAML(`{"apiVersion": "v1", "kind": "ConfigMap", "metadata": {"name": "json"}}`); err != nil {
		t.Errorf("JSON is YAML: %v", err)
	}
	for name, text := range map[string]string{
		"two objects": "kind: A\n---\nkind: B\n", "a key twice": "kind: A\nmetadata:\n  name: a\n  name: b\n", "no object": "# nothing\n", "a list": "- a\n- b\n", "not YAML": "a: b: c",
	} {
		if _, err := decodeYAML(text); err == nil {
			t.Errorf("%s must be an error", name)
		}
	}
}

// A part of the identity that is not known yet, such as all of metadata, is
// not an error while planning.
func TestManifestIdentityUnknownMetadata(t *testing.T) {
	metadata := map[string]attr.Type{"name": types.StringType}
	manifest := types.DynamicValue(types.ObjectValueMust(
		map[string]attr.Type{"apiVersion": types.StringType, "kind": types.StringType, "metadata": types.ObjectType{AttrTypes: metadata}},
		map[string]attr.Value{"apiVersion": types.StringValue("v1"), "kind": types.StringValue("ConfigMap"), "metadata": types.ObjectUnknown(metadata)}))
	if _, known, err := manifestIdentity(manifest); known || err != nil {
		t.Fatalf("known %t, error %v", known, err)
	}
}
