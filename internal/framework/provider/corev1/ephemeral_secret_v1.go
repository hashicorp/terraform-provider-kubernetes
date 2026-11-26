// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"encoding/base64"
	"fmt"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	_ ephemeral.EphemeralResource              = (*SecretV1EphemeralResource)(nil)
	_ ephemeral.EphemeralResourceWithConfigure = (*SecretV1EphemeralResource)(nil)
)

// SecretV1EphemeralResource reads a Secret for use during one Terraform run.
// Its values are stored in neither the plan nor the state.
type SecretV1EphemeralResource struct {
	SDKv2Meta func() any
}

type secretV1EphemeralModel struct {
	Metadata   *secretV1EphemeralMetadata `tfsdk:"metadata"`
	Data       types.Map                  `tfsdk:"data"`
	BinaryData types.Map                  `tfsdk:"binary_data"`
	Type       types.String               `tfsdk:"type"`
	Immutable  types.Bool                 `tfsdk:"immutable"`
}

type secretV1EphemeralMetadata struct {
	Name      types.String `tfsdk:"name"`
	Namespace types.String `tfsdk:"namespace"`
}

func NewSecretV1EphemeralResource() ephemeral.EphemeralResource {
	return &SecretV1EphemeralResource{}
}

func (r *SecretV1EphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_v1"
}

func (r *SecretV1EphemeralResource) Configure(_ context.Context, req ephemeral.ConfigureRequest, _ *ephemeral.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.SDKv2Meta = req.ProviderData.(func() any)
}

func (r *SecretV1EphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads a Secret without storing its values in the Terraform plan or state.",
		Attributes: map[string]schema.Attribute{
			"data": schema.MapAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Sensitive:   true,
				Description: "The values of the Secret that are valid UTF-8 text, by key. A value that is not text is only in `binary_data`.",
			},
			"binary_data": schema.MapAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Sensitive:   true,
				Description: "Every value of the Secret in base64 encoding, by key.",
			},
			"type": schema.StringAttribute{
				Computed:    true,
				Description: "The type of the Secret, such as `Opaque` or `kubernetes.io/tls`.",
			},
			"immutable": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the Secret is marked immutable.",
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": schema.SingleNestedBlock{
				Description: "The Secret to read.",
				Validators:  []validator.Object{objectvalidator.IsRequired()},
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required:    true,
						Description: "Name of the Secret.",
					},
					"namespace": schema.StringAttribute{
						Optional:    true,
						Description: "Namespace of the Secret. Defaults to `default`.",
					},
				},
			},
		},
	}
}

func (r *SecretV1EphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var data secretV1EphemeralModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	clients, _, diags := common.ResolveSDKv2Meta(r.SDKv2Meta)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Cannot initialize the Kubernetes client", err.Error())
		return
	}

	name := data.Metadata.Name.ValueString()
	namespace := data.Metadata.Namespace.ValueString()
	if namespace == "" {
		namespace = metav1.NamespaceDefault
	}
	secret, err := conn.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		// Terraform reads during the plan whenever the name is known then.
		resp.Diagnostics.AddError("Secret not found", fmt.Sprintf(
			"Secret %s/%s does not exist. To read a Secret that the same configuration creates, build metadata.name from the id of that resource, which is not known until the Secret exists.", namespace, name))
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Cannot read Secret", fmt.Sprintf("%s/%s: %s", namespace, name, err))
		return
	}

	// A Terraform string holds text only, so a value that is not valid UTF-8
	// would not survive in data.
	text := map[string]string{}
	encoded := make(map[string]string, len(secret.Data))
	for key, value := range secret.Data {
		encoded[key] = base64.StdEncoding.EncodeToString(value)
		if utf8.Valid(value) {
			text[key] = string(value)
		}
	}
	data.Data, diags = types.MapValueFrom(ctx, types.StringType, text)
	resp.Diagnostics.Append(diags...)
	data.BinaryData, diags = types.MapValueFrom(ctx, types.StringType, encoded)
	resp.Diagnostics.Append(diags...)
	data.Type = types.StringValue(string(secret.Type))
	data.Immutable = types.BoolValue(secret.Immutable != nil && *secret.Immutable)

	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}
