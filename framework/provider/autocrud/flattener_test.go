package autocrud

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestManifestFlattener(t *testing.T) {
	var model TestModel
	manifest := map[string]any{
		"apiVersion": "v1",
		"immutable":  true,
		"binaryData": map[string]interface{}{
			"test": "test",
		},
		"data": map[string]interface{}{
			"test": "test",
		},
		"kind": "ConfigMap",
		"metadata": map[string]any{
			"name":         "test",
			"generateName": "test",
			"namespace":    "default",
			"generation":   int64(1),
			"labels": map[string]any{
				"test": "test",
			},
			"annotations": map[string]any{
				"test": "test",
			},
			"resourceVersion": "test",
			"uid":             "test",
		},
		"test_list": []interface{}{"test", "test"},
		"nested_list": []interface{}{
			map[string]interface{}{
				"name": "test",
			},
			map[string]interface{}{
				"name": "test",
			},
		},
	}

	FlattenManifest(manifest, &model)
	expectedResult := TestModel{
		APIVersion: types.StringValue("v1"),
		BinaryData: map[string]types.String{
			"test": types.StringValue("test"),
		},
		Data: map[string]types.String{
			"test": types.StringValue("test"),
		},
		Immutable: types.BoolValue(true),
		Kind:      types.StringValue("ConfigMap"),
		Metadata: struct {
			Annotations     map[string]types.String `tfsdk:"annotations" manifest:"annotations"`
			GenerateName    types.String            `tfsdk:"generate_name" manifest:"generateName"`
			Generation      types.Int64             `tfsdk:"generation" manifest:"generation"`
			Labels          map[string]types.String `tfsdk:"labels" manifest:"labels"`
			Name            types.String            `tfsdk:"name" manifest:"name"`
			Namespace       types.String            `tfsdk:"namespace" manifest:"namespace"`
			ResourceVersion types.String            `tfsdk:"resource_version" manifest:"resourceVersion"`
			UID             types.String            `tfsdk:"uid" manifest:"uid"`
		}{
			Annotations: map[string]types.String{
				"test": types.StringValue("test"),
			},
			GenerateName: types.StringValue("test"),
			Generation:   types.Int64Value(1),
			Labels: map[string]types.String{
				"test": types.StringValue("test"),
			},
			Name:            types.StringValue("test"),
			Namespace:       types.StringValue("default"),
			ResourceVersion: types.StringValue("test"),
			UID:             types.StringValue("test"),
		},
		List: []types.String{
			types.StringValue("test"),
			types.StringValue("test"),
		},
		NestedList: []struct {
			Name types.String `tfsdk:"name" manifest:"name"`
		}{
			{Name: types.StringValue("test")},
			{Name: types.StringValue("test")},
		},
	}

	assert.Equal(t, expectedResult, model)
}
