package rbacv1

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type RoleBindingModel struct {
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
	RoleRef struct {
		ApiGroup types.String `tfsdk:"api_group" manifest:"apiGroup"`
		Kind     types.String `tfsdk:"kind" manifest:"kind"`
		Name     types.String `tfsdk:"name" manifest:"name"`
	} `tfsdk:"role_ref" manifest:"roleRef"`
	Subjects []struct {
		ApiGroup  types.String `tfsdk:"api_group" manifest:"apiGroup"`
		Kind      types.String `tfsdk:"kind" manifest:"kind"`
		Name      types.String `tfsdk:"name" manifest:"name"`
		Namespace types.String `tfsdk:"namespace" manifest:"namespace"`
	} `tfsdk:"subjects" manifest:"subjects"`
}
