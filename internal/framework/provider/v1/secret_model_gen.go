package v1

import "github.com/hashicorp/terraform-plugin-framework/types"

type SecretModel struct {
	ID         types.String            `tfsdk:"id" manifest:"id"`
	ApiVersion types.String            `tfsdk:"api_version" manifest:"apiVersion"`
	Data       map[string]types.String `tfsdk:"data" manifest:"data"`
	Immutable  types.Bool              `tfsdk:"immutable" manifest:"immutable"`
	Kind       types.String            `tfsdk:"kind" manifest:"kind"`
	Metadata   struct {
		Annotations                map[string]types.String `tfsdk:"annotations" manifest:"annotations"`
		CreationTimestamp          types.String            `tfsdk:"creation_timestamp" manifest:"creationTimestamp"`
		DeletionGracePeriodSeconds types.Int64             `tfsdk:"deletion_grace_period_seconds" manifest:"deletionGracePeriodSeconds"`
		DeletionTimestamp          types.String            `tfsdk:"deletion_timestamp" manifest:"deletionTimestamp"`
		Finalizers                 []types.String          `tfsdk:"finalizers" manifest:"finalizers"`
		GenerateName               types.String            `tfsdk:"generate_name" manifest:"generateName"`
		Generation                 types.Int64             `tfsdk:"generation" manifest:"generation"`
		Labels                     map[string]types.String `tfsdk:"labels" manifest:"labels"`
		Name                       types.String            `tfsdk:"name" manifest:"name"`
		Namespace                  types.String            `tfsdk:"namespace" manifest:"namespace"`
		OwnerReferences            []struct {
			ApiVersion         types.String `tfsdk:"api_version" manifest:"apiVersion"`
			BlockOwnerDeletion types.Bool   `tfsdk:"block_owner_deletion" manifest:"blockOwnerDeletion"`
			Controller         types.Bool   `tfsdk:"controller" manifest:"controller"`
			Kind               types.String `tfsdk:"kind" manifest:"kind"`
			Name               types.String `tfsdk:"name" manifest:"name"`
			Uid                types.String `tfsdk:"uid" manifest:"uid"`
		} `tfsdk:"owner_references" manifest:"ownerReferences"`
		ResourceVersion types.String `tfsdk:"resource_version" manifest:"resourceVersion"`
		SelfLink        types.String `tfsdk:"self_link" manifest:"selfLink"`
		Uid             types.String `tfsdk:"uid" manifest:"uid"`
	} `tfsdk:"metadata" manifest:"metadata"`
	StringData map[string]types.String `tfsdk:"string_data" manifest:"stringData"`
	Type       types.String            `tfsdk:"type" manifest:"type"`
	Name       types.String            `tfsdk:"name" manifest:"name"`
	Namespace  types.String            `tfsdk:"namespace" manifest:"namespace"`
	Pretty     types.String            `tfsdk:"pretty" manifest:"pretty"`
}
