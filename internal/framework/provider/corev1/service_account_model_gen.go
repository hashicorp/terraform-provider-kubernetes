package corev1

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ServiceAccountModel struct {
	Timeouts timeouts.Value `tfsdk:"timeouts"`

	ID                           types.String `tfsdk:"id" manifest:""`
	AutomountServiceAccountToken types.Bool   `tfsdk:"automount_service_account_token" manifest:"automountServiceAccountToken"`
	ImagePullSecrets             []struct {
		Name types.String `tfsdk:"name" manifest:"name"`
	} `tfsdk:"image_pull_secrets" manifest:"imagePullSecrets"`
	Metadata struct {
		Annotations     map[string]types.String `tfsdk:"annotations" manifest:"annotations"`
		GenerateName    types.String            `tfsdk:"generate_name" manifest:"generateName"`
		Generation      types.Int64             `tfsdk:"generation" manifest:"generation"`
		Labels          map[string]types.String `tfsdk:"labels" manifest:"labels"`
		Name            types.String            `tfsdk:"name" manifest:"name"`
		Namespace       types.String            `tfsdk:"namespace" manifest:"namespace"`
		ResourceVersion types.String            `tfsdk:"resource_version" manifest:"resourceVersion"`
		UID             types.String            `tfsdk:"uid" manifest:"uid"`
	} `tfsdk:"metadata" manifest:"metadata"`
	Secrets []struct {
		APIVersion      types.String `tfsdk:"api_version" manifest:"apiVersion"`
		FieldPath       types.String `tfsdk:"field_path" manifest:"fieldPath"`
		Kind            types.String `tfsdk:"kind" manifest:"kind"`
		Name            types.String `tfsdk:"name" manifest:"name"`
		Namespace       types.String `tfsdk:"namespace" manifest:"namespace"`
		ResourceVersion types.String `tfsdk:"resource_version" manifest:"resourceVersion"`
		UID             types.String `tfsdk:"uid" manifest:"uid"`
	} `tfsdk:"secrets" manifest:"secrets"`
}
