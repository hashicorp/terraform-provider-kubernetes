// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func serviceAccountImport(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse, callback func() any, defaultAccount bool) {
	id := req.ID
	if id == "" {
		if req.Identity == nil {
			resp.Diagnostics.AddError("Missing service account identifier", "Supply a namespace/name identifier or a resource identity.")
			return
		}
		var identity common.NamespacedResourceIdentity
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if identity.APIVersion.ValueString() != serviceAccountAPIVersion || identity.Kind.ValueString() != serviceAccountKind {
			resp.Diagnostics.AddError("Invalid service account identity", "Expected api_version \"v1\" and kind \"ServiceAccount\".")
			return
		}
		namespace := identity.Namespace.ValueString()
		if identity.Namespace.IsNull() {
			namespace = "default"
		}
		id = namespace + "/" + identity.Name.ValueString()
	}
	namespace, name, err := serviceAccountIDParts(id, defaultAccount)
	if err != nil {
		resp.Diagnostics.AddError("Invalid service account identifier", err.Error())
		return
	}
	conn, filters, diags := serviceAccountClient(callback)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	account, err := conn.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Error importing service account", err.Error())
		return
	}
	secret, diags := serviceAccountDiscoverDefaultSecret(ctx, conn, account)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Import has no configuration. Empty maps support both explicit {} and
	// omission through the metadata plan modifier, without an import-only update.
	metadata := common.NamespacedMetadataModel{}
	metadata.Annotations = types.MapValueMust(types.StringType, map[string]attr.Value{})
	metadata.Labels = types.MapValueMust(types.StringType, map[string]attr.Value{})
	model := ServiceAccountV1Model{
		ID:                types.StringValue(id),
		Metadata:          []common.NamespacedMetadataModel{metadata},
		DefaultSecretName: types.StringValue(secret),
		Secret:            types.SetNull(serviceAccountReferenceType),
		ImagePullSecret:   types.SetNull(serviceAccountReferenceType),
		Timeouts:          timeouts.Value{Object: types.ObjectNull(timeoutsAttributeTypes(serviceAccountSchema(ctx, defaultAccount)))},
	}
	resp.Diagnostics.Append(serviceAccountFlatten(ctx, &model, account, filters)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(serviceAccountWriteModel(ctx, &resp.State, model, defaultAccount)...)
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, serviceAccountIdentity(account.ObjectMeta))...)
	}
}

// Frozen schema-0 JSON, independent of the target schema. Pointers and nil
// collections retain the distinction between null and explicit zero/empty.
type serviceAccountSDKStateV0 struct {
	ID                string                         `json:"id"`
	Metadata          []serviceAccountSDKMetadataV0  `json:"metadata"`
	Secret            []serviceAccountSDKReferenceV0 `json:"secret"`
	ImagePullSecret   []serviceAccountSDKReferenceV0 `json:"image_pull_secret"`
	Automount         *bool                          `json:"automount_service_account_token"`
	DefaultSecretName *string                        `json:"default_secret_name"`
	Timeouts          *serviceAccountSDKTimeoutsV0   `json:"timeouts"`
}

type serviceAccountSDKMetadataV0 struct {
	Annotations     map[string]string `json:"annotations"`
	Labels          map[string]string `json:"labels"`
	Name            string            `json:"name"`
	Namespace       string            `json:"namespace"`
	GenerateName    json.RawMessage   `json:"generate_name"`
	Generation      int64             `json:"generation"`
	ResourceVersion string            `json:"resource_version"`
	UID             string            `json:"uid"`
}

type serviceAccountSDKReferenceV0 struct {
	Name *string `json:"name"`
}

type serviceAccountSDKTimeoutsV0 struct {
	Create *string `json:"create"`
}

func serviceAccountMoveState(ctx context.Context, defaultAccount bool) []resource.StateMover {
	sourceType := "kubernetes_service_account"
	if defaultAccount {
		sourceType = "kubernetes_default_service_account"
	}
	timeoutTypes := timeoutsAttributeTypes(serviceAccountSchema(ctx, defaultAccount))
	return []resource.StateMover{{StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
		if req.SourceTypeName != sourceType || req.SourceSchemaVersion != 0 ||
			!strings.HasSuffix(req.SourceProviderAddress, sdkv2ProviderAddressSuffix) {
			tflog.Debug(ctx, "MoveState: unsupported ServiceAccount source, skipping", map[string]any{
				"source_type_name":        req.SourceTypeName,
				"source_schema_version":   req.SourceSchemaVersion,
				"source_provider_address": req.SourceProviderAddress,
			})
			return
		}
		summary := "Unable to move " + sourceType + " state"
		if req.SourceIdentitySchemaVersion < 0 || req.SourceIdentitySchemaVersion > common.IdentitySchemaVersion {
			resp.Diagnostics.AddError(summary, fmt.Sprintf("Unsupported source identity schema version %d.", req.SourceIdentitySchemaVersion))
			return
		}
		if req.SourceRawState == nil || len(req.SourceRawState.JSON) == 0 {
			resp.Diagnostics.AddError(summary, "The source state has no JSON data; legacy flatmap state is not supported.")
			return
		}
		var prior serviceAccountSDKStateV0
		if err := json.Unmarshal(req.SourceRawState.JSON, &prior); err != nil {
			resp.Diagnostics.AddError(summary, fmt.Sprintf("Could not decode the source state: %s", err))
			return
		}
		namespace, name, err := serviceAccountIDParts(prior.ID, defaultAccount)
		if err != nil {
			resp.Diagnostics.AddError(summary, err.Error())
			return
		}
		if len(prior.Metadata) != 1 {
			resp.Diagnostics.AddError(summary, "Expected exactly one metadata element in the source state.")
			return
		}
		m := prior.Metadata[0]
		if m.Name != name || m.Namespace != namespace {
			resp.Diagnostics.AddError(summary, "The source metadata name and namespace do not match its identifier.")
			return
		}
		if req.SourceIdentity != nil && len(req.SourceIdentity.JSON) > 0 &&
			strings.TrimSpace(string(req.SourceIdentity.JSON)) != "null" {
			var identity struct {
				Name       string `json:"name"`
				Namespace  string `json:"namespace"`
				Kind       string `json:"kind"`
				APIVersion string `json:"api_version"`
			}
			if err := json.Unmarshal(req.SourceIdentity.JSON, &identity); err != nil {
				resp.Diagnostics.AddError(summary, fmt.Sprintf("Could not decode the source identity: %s", err))
				return
			}
			if identity.Name != name || identity.Namespace != namespace ||
				(identity.Kind != "" && identity.Kind != serviceAccountKind) ||
				(identity.APIVersion != "" && identity.APIVersion != serviceAccountAPIVersion) ||
				(req.SourceIdentitySchemaVersion == common.IdentitySchemaVersion &&
					(identity.Kind == "" || identity.APIVersion == "")) {
				resp.Diagnostics.AddError(summary, "The source identity does not match its ServiceAccount state.")
				return
			}
		}
		if defaultAccount && len(m.GenerateName) != 0 {
			resp.Diagnostics.AddError(summary, "Default service account state must not contain generate_name.")
			return
		}
		metadata := common.NamespacedMetadataModel{
			MetadataModel: common.MetadataModel{
				MetadataBase: common.MetadataBase{
					Name: types.StringValue(m.Name), Generation: types.Int64Value(m.Generation),
					UID: types.StringValue(m.UID), ResourceVersion: types.StringValue(m.ResourceVersion),
				},
				GenerateName: types.StringNull(),
			},
			Namespace: types.StringValue(m.Namespace),
		}
		if len(m.GenerateName) != 0 {
			var generateName *string
			if err := json.Unmarshal(m.GenerateName, &generateName); err != nil {
				resp.Diagnostics.AddError(summary, fmt.Sprintf("Could not decode generate_name: %s", err))
				return
			}
			if generateName != nil {
				metadata.GenerateName = types.StringValue(*generateName)
			}
		}
		labels, diags := sdkv2MapToFramework(ctx, m.Labels)
		resp.Diagnostics.Append(diags...)
		metadata.Labels = labels
		annotations, diags := sdkv2MapToFramework(ctx, m.Annotations)
		resp.Diagnostics.Append(diags...)
		metadata.Annotations = annotations
		model := ServiceAccountV1Model{
			ID:                           types.StringValue(prior.ID),
			Metadata:                     []common.NamespacedMetadataModel{metadata},
			AutomountServiceAccountToken: types.BoolPointerValue(prior.Automount),
			DefaultSecretName:            types.StringPointerValue(prior.DefaultSecretName),
			Timeouts:                     timeouts.Value{Object: types.ObjectNull(timeoutTypes)},
		}
		model.Secret, diags = serviceAccountSDKReferences(ctx, prior.Secret)
		resp.Diagnostics.Append(diags...)
		model.ImagePullSecret, diags = serviceAccountSDKReferences(ctx, prior.ImagePullSecret)
		resp.Diagnostics.Append(diags...)
		if prior.Timeouts != nil {
			value, d := types.ObjectValue(timeoutTypes, map[string]attr.Value{"create": types.StringPointerValue(prior.Timeouts.Create)})
			resp.Diagnostics.Append(d...)
			model.Timeouts = timeouts.Value{Object: value}
		}
		if resp.Diagnostics.HasError() {
			return
		}
		resp.Diagnostics.Append(serviceAccountWriteModel(ctx, &resp.TargetState, model, defaultAccount)...)
		if !resp.Diagnostics.HasError() && resp.TargetIdentity != nil {
			resp.Diagnostics.Append(resp.TargetIdentity.Set(ctx, serviceAccountIdentity(metav1.ObjectMeta{Name: name, Namespace: namespace}))...)
		}
	}}}
}

func serviceAccountSDKReferences(ctx context.Context, prior []serviceAccountSDKReferenceV0) (types.Set, diag.Diagnostics) {
	if prior == nil {
		return types.SetNull(serviceAccountReferenceType), nil
	}
	references := make([]serviceAccountReferenceModel, 0, len(prior))
	for _, reference := range prior {
		references = append(references, serviceAccountReferenceModel{Name: types.StringPointerValue(reference.Name)})
	}
	return types.SetValueFrom(ctx, serviceAccountReferenceType, references)
}
