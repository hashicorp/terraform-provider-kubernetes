// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/morph"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/payload"
)

func TestBackfillComputedFieldsOmitsManifestAbsentComputedAnnotation(t *testing.T) {
	objectType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"apiVersion": tftypes.String,
		"kind":       tftypes.String,
		"metadata": tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"name":        tftypes.String,
			"namespace":   tftypes.String,
			"annotations": tftypes.Map{ElementType: tftypes.String},
		}},
		"data": tftypes.Map{ElementType: tftypes.String},
	}}
	metadataType := objectType.AttributeTypes["metadata"]

	manifest := tftypes.NewValue(objectType, map[string]tftypes.Value{
		"apiVersion": tftypes.NewValue(tftypes.String, "v1"),
		"kind":       tftypes.NewValue(tftypes.String, "ConfigMap"),
		"metadata": tftypes.NewValue(metadataType, map[string]tftypes.Value{
			"name":      tftypes.NewValue(tftypes.String, "example"),
			"namespace": tftypes.NewValue(tftypes.String, "default"),
			"annotations": tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{
				"example.com/managed": tftypes.NewValue(tftypes.String, "terraform"),
			}),
		}),
		"data": tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{
			"value": tftypes.NewValue(tftypes.String, "v2"),
		}),
	})
	plannedObject := tftypes.NewValue(objectType, map[string]tftypes.Value{
		"apiVersion": tftypes.NewValue(tftypes.String, "v1"),
		"kind":       tftypes.NewValue(tftypes.String, "ConfigMap"),
		"metadata": tftypes.NewValue(metadataType, map[string]tftypes.Value{
			"name":      tftypes.NewValue(tftypes.String, "example"),
			"namespace": tftypes.NewValue(tftypes.String, "default"),
			"annotations": tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{
				"controller.example.com/runtime": tftypes.NewValue(tftypes.String, "stale-runtime-value"),
				"example.com/managed":            tftypes.NewValue(tftypes.String, "terraform"),
			}),
		}),
		"data": tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{
			"value": tftypes.NewValue(tftypes.String, "v2"),
		}),
	})

	annotationsPath := tftypes.NewAttributePath().WithAttributeName("metadata").WithAttributeName("annotations")
	runtimeAnnotationPath := tftypes.NewAttributePath().WithAttributeName("metadata").WithAttributeName("annotations").WithElementKeyString("controller.example.com/runtime")
	computedFields := map[string]*tftypes.AttributePath{
		annotationsPath.String():       annotationsPath,
		runtimeAnnotationPath.String(): runtimeAnnotationPath,
	}

	backfilled, diagnostics, err := backfillComputedFields(plannedObject, manifest, computedFields)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(diagnostics) > 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}

	pu, err := payload.FromTFValue(morph.UnknownToNull(backfilled), nil, tftypes.NewAttributePath())
	if err != nil {
		t.Fatalf("unexpected payload conversion error: %s", err)
	}
	rqObj := mapRemoveNulls(pu.(map[string]interface{}))

	annotations := rqObj["metadata"].(map[string]interface{})["annotations"].(map[string]interface{})
	if _, ok := annotations["controller.example.com/runtime"]; ok {
		t.Fatalf("runtime annotation should not be included in apply payload: %#v", annotations)
	}
	if got := annotations["example.com/managed"]; got != "terraform" {
		t.Fatalf("managed annotation mismatch: got %#v", got)
	}
}
