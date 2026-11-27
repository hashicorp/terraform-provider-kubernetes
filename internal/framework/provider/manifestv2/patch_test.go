// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestv2

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// The patch takes attributes from the schema of kubernetes_manifest_v2. This
// is the type of its state: a change to it, for whichever resource it was
// made, needs a schema version and a state upgrader of the patch's own.
func TestPatchStateType(t *testing.T) {
	var resp resource.SchemaResponse
	(&ManifestPatch{}).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	const want = `types.ObjectType["api_version":basetypes.StringType, "computed_fields":types.ListType[basetypes.StringType], ` +
		`"destroy_behavior":basetypes.StringType, "drift":types.ListType[basetypes.StringType], ` +
		`"field_manager":types.ObjectType["force_conflicts":basetypes.BoolType, "name":basetypes.StringType], ` +
		`"kind":basetypes.StringType, "manifest":basetypes.DynamicType, "name":basetypes.StringType, "namespace":basetypes.StringType, ` +
		`"show_drift_values":basetypes.BoolType, "timeouts":timeouts.Type, "uid":basetypes.StringType, ` +
		`"wait":types.ObjectType["condition":types.ListType[types.ObjectType["status":basetypes.StringType, "type":basetypes.StringType]], ` +
		`"fields":types.MapType[basetypes.StringType], "ready":basetypes.BoolType, "rollout":basetypes.BoolType], "yaml_body":basetypes.StringType]`
	if got := resp.Schema.Type().String(); got != want || resp.Schema.Version != 0 {
		t.Errorf("schema version %d, state type:\n%s\nwant version 0 and:\n%s", resp.Schema.Version, got, want)
	}
}
