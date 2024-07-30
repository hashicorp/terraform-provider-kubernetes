package rbacv1

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type RoleModel struct {
	Timeouts timeouts.Value `tfsdk:"timeouts"`

	ID       types.String `tfsdk:"id" manifest:""`
	Metadata struct {
		Annotations     map[string]types.String `tfsdk:"annotations" manifest:"annotations"`
		GenerateName    types.String            `tfsdk:"generate_name" manifest:"generateName"`
		Generation      types.Int64             `tfsdk:"generation" manifest:"generation"`
		Labels          map[string]types.String `tfsdk:"labels" manifest:"labels"`
		Name            types.String            `tfsdk:"name" manifest:"name"`
		ResourceVersion types.String            `tfsdk:"resource_version" manifest:"resourceVersion"`
		UID             types.String            `tfsdk:"uid" manifest:"uid"`
	} `tfsdk:"metadata" manifest:"metadata"`
	Rules []struct {
		ApiGroups       []types.String `tfsdk:"api_groups" manifest:"apiGroups"`
		NonResourceUrls []types.String `tfsdk:"non_resource_urls" manifest:"nonResourceUrls"`
		ResourceNames   []types.String `tfsdk:"resource_names" manifest:"resourceNames"`
		Resources       []types.String `tfsdk:"resources" manifest:"resources"`
		Verbs           []types.String `tfsdk:"verbs" manifest:"verbs"`
	} `tfsdk:"rules" manifest:"rules"`
}
