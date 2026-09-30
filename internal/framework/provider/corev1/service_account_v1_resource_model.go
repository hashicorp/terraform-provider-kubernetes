// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	serviceAccountKind       = "ServiceAccount"
	serviceAccountAPIVersion = "v1"
)

type ServiceAccountV1Model struct {
	ID                           types.String                     `tfsdk:"id"`
	Metadata                     []common.NamespacedMetadataModel `tfsdk:"metadata"`
	Secret                       types.Set                        `tfsdk:"secret"`
	ImagePullSecret              types.Set                        `tfsdk:"image_pull_secret"`
	AutomountServiceAccountToken types.Bool                       `tfsdk:"automount_service_account_token"`
	DefaultSecretName            types.String                     `tfsdk:"default_secret_name"`
	Timeouts                     timeouts.Value                   `tfsdk:"timeouts"`
}

type DefaultServiceAccountV1Model struct {
	ID                           types.String                    `tfsdk:"id"`
	Metadata                     []common.NamespacedMetadataBase `tfsdk:"metadata"`
	Secret                       types.Set                       `tfsdk:"secret"`
	ImagePullSecret              types.Set                       `tfsdk:"image_pull_secret"`
	AutomountServiceAccountToken types.Bool                      `tfsdk:"automount_service_account_token"`
	DefaultSecretName            types.String                    `tfsdk:"default_secret_name"`
	Timeouts                     timeouts.Value                  `tfsdk:"timeouts"`
}

type serviceAccountModelReader interface {
	Get(context.Context, any) diag.Diagnostics
}

type serviceAccountModelWriter interface {
	Set(context.Context, any) diag.Diagnostics
}

// Only the shared CRUD representation has generate_name. The default resource's
// Terraform model never does, since Framework requires an exact object shape.
func serviceAccountReadModel(ctx context.Context, source serviceAccountModelReader, defaultAccount bool) (ServiceAccountV1Model, diag.Diagnostics) {
	var model ServiceAccountV1Model
	if !defaultAccount {
		diags := source.Get(ctx, &model)
		return model, diags
	}
	var other DefaultServiceAccountV1Model
	diags := source.Get(ctx, &other)
	model = ServiceAccountV1Model{
		ID: other.ID, Secret: other.Secret, ImagePullSecret: other.ImagePullSecret,
		AutomountServiceAccountToken: other.AutomountServiceAccountToken,
		DefaultSecretName:            other.DefaultSecretName, Timeouts: other.Timeouts,
	}
	for _, m := range other.Metadata {
		model.Metadata = append(model.Metadata, common.NamespacedMetadataModel{
			MetadataModel: common.MetadataModel{MetadataBase: m.MetadataBase, GenerateName: types.StringNull()},
			Namespace:     m.Namespace,
		})
	}
	return model, diags
}

func serviceAccountWriteModel(ctx context.Context, target serviceAccountModelWriter, model ServiceAccountV1Model, defaultAccount bool) diag.Diagnostics {
	if !defaultAccount {
		return target.Set(ctx, &model)
	}
	other := DefaultServiceAccountV1Model{
		ID: model.ID, Secret: model.Secret, ImagePullSecret: model.ImagePullSecret,
		AutomountServiceAccountToken: model.AutomountServiceAccountToken,
		DefaultSecretName:            model.DefaultSecretName, Timeouts: model.Timeouts,
	}
	for _, m := range model.Metadata {
		other.Metadata = append(other.Metadata, common.NamespacedMetadataBase{MetadataBase: m.MetadataBase, Namespace: m.Namespace})
	}
	return target.Set(ctx, &other)
}

func serviceAccountIDParts(id string, defaultAccount bool) (string, string, error) {
	namespace, name, err := kubernetes.IdParts(id)
	if err != nil {
		return "", "", err
	}
	if namespace == "" || name == "" {
		return "", "", fmt.Errorf("expected a non-empty namespace/name identifier, got %q", id)
	}
	if defaultAccount && name != "default" {
		return "", "", fmt.Errorf("the default service account must be named \"default\", got %q", name)
	}
	return namespace, name, nil
}

func serviceAccountIdentity(meta metav1.ObjectMeta) common.NamespacedResourceIdentity {
	return common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{
			APIVersion: types.StringValue(serviceAccountAPIVersion),
			Kind:       types.StringValue(serviceAccountKind), Name: types.StringValue(meta.Name),
		},
		Namespace: types.StringValue(meta.Namespace),
	}
}

func serviceAccountSetComputed(model *ServiceAccountV1Model, account *corev1.ServiceAccount) {
	model.ID = types.StringValue(kubernetes.BuildId(account.ObjectMeta))
	model.Metadata[0].Name = types.StringValue(account.Name)
	model.Metadata[0].Namespace = types.StringValue(account.Namespace)
	model.Metadata[0].UID = types.StringValue(string(account.UID))
	model.Metadata[0].Generation = types.Int64Value(account.Generation)
	model.Metadata[0].ResourceVersion = types.StringValue(account.ResourceVersion)
}

var serviceAccountReferenceType = types.ObjectType{AttrTypes: map[string]attr.Type{"name": types.StringType}}

type serviceAccountReferenceModel struct {
	Name types.String `tfsdk:"name"`
}

func serviceAccountReferenceNames(ctx context.Context, value types.Set) ([]string, diag.Diagnostics) {
	if value.IsNull() {
		return []string{}, nil
	}
	var references []serviceAccountReferenceModel
	diags := value.ElementsAs(ctx, &references, false)
	names := make([]string, 0, len(references))
	for _, reference := range references {
		if reference.Name.IsUnknown() {
			diags.AddError("Unknown service account secret", "Secret names must be known before writing the service account.")
			continue
		}
		names = append(names, reference.Name.ValueString())
	}
	return names, diags
}

func serviceAccountReferenceSet(ctx context.Context, names []string, prior types.Set) (types.Set, diag.Diagnostics) {
	if len(names) == 0 && prior.IsNull() {
		return types.SetNull(serviceAccountReferenceType), nil
	}
	references := make([]serviceAccountReferenceModel, 0, len(names))
	for _, name := range names {
		references = append(references, serviceAccountReferenceModel{Name: types.StringValue(name)})
	}
	return types.SetValueFrom(ctx, serviceAccountReferenceType, references)
}

func serviceAccountFlatten(ctx context.Context, model *ServiceAccountV1Model, account *corev1.ServiceAccount, filters kubernetes.MetadataFilters) diag.Diagnostics {
	metadata, diags := common.FlattenNamespacedMetadata(ctx, account.ObjectMeta, model.Metadata, filters.GetIgnoreAnnotations(), filters.GetIgnoreLabels())
	if len(metadata) == 1 {
		metadata[0].GenerateName = types.StringValue(account.GenerateName)
	}
	model.Metadata = metadata
	model.ID = types.StringValue(kubernetes.BuildId(account.ObjectMeta))
	// Unlike the schema's default, an omitted API field historically reads false.
	model.AutomountServiceAccountToken = types.BoolValue(account.AutomountServiceAccountToken != nil && *account.AutomountServiceAccountToken)
	var names []string
	for _, reference := range account.Secrets {
		if reference.Name != model.DefaultSecretName.ValueString() {
			names = append(names, reference.Name)
		}
	}
	value, d := serviceAccountReferenceSet(ctx, names, model.Secret)
	diags.Append(d...)
	model.Secret = value
	names = nil
	for _, reference := range account.ImagePullSecrets {
		names = append(names, reference.Name)
	}
	value, d = serviceAccountReferenceSet(ctx, names, model.ImagePullSecret)
	diags.Append(d...)
	model.ImagePullSecret = value
	return diags
}
