package authenticationv1

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

var (
	_ ephemeral.EphemeralResource              = (*TokenRequestEphemeralResource)(nil)
	_ ephemeral.EphemeralResourceWithConfigure = (*TokenRequestEphemeralResource)(nil)
)

type TokenRequestEphemeralResource struct {
	SDKv2Meta func() any
}

type TokenRequestMetadata struct {
	Name      types.String `tfsdk:"name"`
	Namespace types.String `tfsdk:"namespace"`
}

type BoundObjectReference struct {
	APIVersion types.String `tfsdk:"api_version"`
	Kind       types.String `tfsdk:"kind"`
	Name       types.String `tfsdk:"name"`
	UID        types.String `tfsdk:"uid"`
}

type TokenRequestSpec struct {
	Audiences         []types.String        `tfsdk:"audiences"`
	BoundObjecRef     *BoundObjectReference `tfsdk:"bound_object_ref"`
	ExpirationSeconds types.Int64           `tfsdk:"expiration_seconds"`
}

type TokenRequestModel struct {
	Metadata TokenRequestMetadata `tfsdk:"metadata"`
	Spec     *TokenRequestSpec    `tfsdk:"spec"`

	Token               types.String `tfsdk:"token"`
	ExpirationTimestamp types.String `tfsdk:"expiration_timestamp"`
}

func NewTokenRequestEphemeralResource() ephemeral.EphemeralResource {
	return &TokenRequestEphemeralResource{}
}

func (r *TokenRequestEphemeralResource) Configure(ctx context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.SDKv2Meta = req.ProviderData.(func() any)
}

func (r *TokenRequestEphemeralResource) Metadata(ctx context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_token_request_v1"
}

func (r *TokenRequestEphemeralResource) Schema(ctx context.Context, req ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	objectMetaOpenAPI := metav1.ObjectMeta{}.SwaggerDoc()

	tokenreqOpenAPISpec := authv1.TokenRequestSpec{}.SwaggerDoc()
	tokenreqOpenAPIStatus := authv1.TokenRequestStatus{}.SwaggerDoc()
	tokenreqOpenAPIBoundObjRef := authv1.BoundObjectReference{}.SwaggerDoc()

	resp.Schema = schema.Schema{
		Description: "TokenRequest requests a token for a given service account.",
		Attributes: map[string]schema.Attribute{
			"token": schema.StringAttribute{
				Computed:    true,
				Optional:    true,
				Description: tokenreqOpenAPIStatus["token"],
			},
			"expiration_timestamp": schema.StringAttribute{
				Computed:    true,
				Optional:    true,
				Description: tokenreqOpenAPIStatus["expirationTimestamp"],
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": schema.SingleNestedBlock{
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required:    true,
						Description: objectMetaOpenAPI["name"],
					},
					"namespace": schema.StringAttribute{
						Required:    true,
						Description: objectMetaOpenAPI["namespace"],
					},
				},
			},
			"spec": schema.SingleNestedBlock{
				Description: tokenreqOpenAPISpec[""],
				Attributes: map[string]schema.Attribute{
					"audiences": schema.ListAttribute{
						Optional:    true,
						Computed:    true,
						ElementType: types.StringType,
						Description: tokenreqOpenAPISpec["audiences"],
					},
					"expiration_seconds": schema.Int64Attribute{
						Optional:    true,
						Computed:    true,
						Description: tokenreqOpenAPISpec["expirationSeconds"],
					},
				},
				Blocks: map[string]schema.Block{
					"bound_object_ref": schema.SingleNestedBlock{
						Attributes: map[string]schema.Attribute{
							"api_version": schema.StringAttribute{
								Optional:    true,
								Description: tokenreqOpenAPIBoundObjRef["apiVersion"],
							},
							"kind": schema.StringAttribute{
								Optional:    true,
								Description: tokenreqOpenAPIBoundObjRef["kind"],
							},
							"name": schema.StringAttribute{
								Optional:    true,
								Description: tokenreqOpenAPIBoundObjRef["name"],
							},
							"uid": schema.StringAttribute{
								Optional:    true,
								Description: tokenreqOpenAPIBoundObjRef["uid"],
							},
						},
					},
				},
			},
		},
	}
}

func expandStringSlice(s []types.String) []string {
	ss := make([]string, len(s))
	for i := 0; i < len(s); i++ {
		ss[i] = s[i].ValueString()
	}
	return ss
}

func (r *TokenRequestEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var data TokenRequestModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Metadata.Name.ValueString()
	namespace := data.Metadata.Namespace.ValueString()
	if namespace == "" {
		namespace = "default"
	}

	conn, err := r.SDKv2Meta().(kubernetes.KubeClientsets).MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("error initializing kubernetes client", err.Error())
		return
	}

	tokenRequest := authv1.TokenRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	if data.Spec != nil {
		tokenRequest.Spec = authv1.TokenRequestSpec{
			Audiences:         expandStringSlice(data.Spec.Audiences),
			ExpirationSeconds: data.Spec.ExpirationSeconds.ValueInt64Pointer(),
		}

		if data.Spec.BoundObjecRef != nil {
			tokenRequest.Spec.BoundObjectRef = &authv1.BoundObjectReference{
				Kind:       data.Spec.BoundObjecRef.Kind.ValueString(),
				APIVersion: data.Spec.BoundObjecRef.APIVersion.ValueString(),
				Name:       data.Spec.BoundObjecRef.Name.ValueString(),
				UID:        k8stypes.UID(data.Spec.BoundObjecRef.UID.ValueString()),
			}
		}
	}

	res, err := conn.CoreV1().ServiceAccounts(namespace).CreateToken(ctx, name, &tokenRequest, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError("error creating token request", err.Error())
		return
	}

	data.ExpirationTimestamp = types.StringValue(res.Status.ExpirationTimestamp.Format(time.RFC3339))
	data.Token = types.StringValue(res.Status.Token)

	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
