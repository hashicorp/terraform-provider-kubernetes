package corev1

import "github.com/hashicorp/terraform-plugin-framework/types"

type SecretModel struct {
	ID        types.String            `tfsdk:"id" manifest:""`
	Data      map[string]types.String `tfsdk:"data" manifest:"data"`
	Immutable types.Bool              `tfsdk:"immutable" manifest:"immutable"`
	Metadata  struct {
		Annotations     map[string]types.String `tfsdk:"annotations" manifest:"annotations"`
		GenerateName    types.String            `tfsdk:"generate_name" manifest:"generateName"`
		Generation      types.Int64             `tfsdk:"generation" manifest:"generation"`
		Labels          map[string]types.String `tfsdk:"labels" manifest:"labels"`
		Name            types.String            `tfsdk:"name" manifest:"name"`
		Namespace       types.String            `tfsdk:"namespace" manifest:"namespace"`
		ResourceVersion types.String            `tfsdk:"resource_version" manifest:"resourceVersion"`
		UID             types.String            `tfsdk:"uid" manifest:"uid"`
	} `tfsdk:"metadata" manifest:"metadata"`
	StringData map[string]types.String `tfsdk:"string_data" manifest:"stringData"`
	Type       types.String            `tfsdk:"type" manifest:"type"`
}
