// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestpatch

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	apitypes "k8s.io/apimachinery/pkg/types"
)

func TestBuildAndParseID(t *testing.T) {
	m := manifestPatchModel{
		APIVersion:   types.StringValue("apps/v1"),
		Kind:         types.StringValue("Deployment"),
		Name:         types.StringValue("app"),
		Namespace:    types.StringValue("default"),
		FieldManager: types.StringValue("terraform-patch"),
	}
	id := buildID(&m)
	want := "apiVersion=apps/v1,kind=Deployment,namespace=default,name=app,fieldManager=terraform-patch"
	if id != want {
		t.Fatalf("buildID = %q, want %q", id, want)
	}
	pid, err := parsePatchID(id)
	if err != nil {
		t.Fatalf("parsePatchID: %v", err)
	}
	if pid.APIVersion != "apps/v1" || pid.Kind != "Deployment" || pid.Name != "app" ||
		pid.Namespace != "default" || pid.FieldManager != "terraform-patch" {
		t.Fatalf("round-trip mismatch: %+v", pid)
	}
}

func TestParseID_requiresCore(t *testing.T) {
	if _, err := parsePatchID("kind=Deployment,name=app"); err == nil {
		t.Fatal("missing apiVersion should error")
	}
	if _, err := parsePatchID("apiVersion=v1,kind=ConfigMap,name=x,bogus=y"); err == nil {
		t.Fatal("unknown key should error")
	}
}

func TestDefaults(t *testing.T) {
	var m manifestPatchModel
	if fieldManagerOf(&m) != "terraform-patch" {
		t.Errorf("default field manager wrong: %s", fieldManagerOf(&m))
	}
	if destroyBehaviorOf(&m) != "relinquish" {
		t.Errorf("default destroy behavior wrong: %s", destroyBehaviorOf(&m))
	}
}

func TestJSONPatchType(t *testing.T) {
	cases := map[string]apitypes.PatchType{
		"json":      apitypes.JSONPatchType,
		"merge":     apitypes.MergePatchType,
		"strategic": apitypes.StrategicMergePatchType,
	}
	for in, want := range cases {
		got, err := jsonPatchType(in)
		if err != nil || got != want {
			t.Errorf("jsonPatchType(%q) = %v,%v want %v", in, got, err, want)
		}
	}
	if _, err := jsonPatchType("apply"); err == nil {
		t.Error("apply is not a json patch type")
	}
}

// TestSSABody verifies the identity fields are merged with the patch and that a null
// leaf is preserved as JSON null (SSA field removal).
func TestSSABody(t *testing.T) {
	ctx := context.Background()
	ann := types.ObjectValueMust(
		map[string]attr.Type{"gone": types.StringType},
		map[string]attr.Value{"gone": types.StringNull()},
	)
	md := types.ObjectValueMust(
		map[string]attr.Type{"annotations": ann.Type(ctx)},
		map[string]attr.Value{"annotations": ann},
	)
	spec := types.ObjectValueMust(
		map[string]attr.Type{"replicas": types.NumberType},
		map[string]attr.Value{"replicas": types.NumberValue(big.NewFloat(3))},
	)
	patchObj := types.ObjectValueMust(
		map[string]attr.Type{"spec": spec.Type(ctx), "metadata": md.Type(ctx)},
		map[string]attr.Value{"spec": spec, "metadata": md},
	)
	m := manifestPatchModel{
		APIVersion: types.StringValue("apps/v1"),
		Kind:       types.StringValue("Deployment"),
		Name:       types.StringValue("app"),
		Namespace:  types.StringValue("default"),
		Patch:      types.DynamicValue(patchObj),
	}
	body, err := ssaBody(ctx, &m)
	if err != nil {
		t.Fatalf("ssaBody: %v", err)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if got["apiVersion"] != "apps/v1" || got["kind"] != "Deployment" {
		t.Fatalf("identity missing: %v", got)
	}
	meta := got["metadata"].(map[string]interface{})
	if meta["name"] != "app" || meta["namespace"] != "default" {
		t.Fatalf("metadata identity missing: %v", meta)
	}
	// null leaf must be PRUNED (omitted) so SSA removes the owned field — a JSON null
	// would be coerced to "" for string maps like annotations.
	anns := meta["annotations"].(map[string]interface{})
	if _, present := anns["gone"]; present {
		t.Fatalf("null leaf must be pruned (omitted), but it is present: %v", anns)
	}
	if got["spec"].(map[string]interface{})["replicas"].(float64) != 3 {
		t.Fatalf("spec.replicas wrong: %v", got["spec"])
	}
}
