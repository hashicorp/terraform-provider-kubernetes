package certificatesv1

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kwait "k8s.io/apimachinery/pkg/util/wait"
	kretry "k8s.io/client-go/util/retry"
)

var (
	_ ephemeral.EphemeralResource              = (*CertificateSigningRequestEphemeralResource)(nil)
	_ ephemeral.EphemeralResourceWithConfigure = (*CertificateSigningRequestEphemeralResource)(nil)
)

type CertificateSigningRequestEphemeralResource struct {
	SDKv2Meta func() any
}

type CertificateSigningRequestMetadata struct {
	Name      types.String `tfsdk:"name"`
	Namespace types.String `tfsdk:"namespace"`
}

type CertificateSigningRequestSpec struct {
	ExpirationSeconds types.Int32    `tfsdk:"expiration_seconds"`
	Request           types.String   `tfsdk:"request"`
	SignerName        types.String   `tfsdk:"signer_name"`
	Usages            []types.String `tfsdk:"usages"`
}

type CertificateSigningRequestModel struct {
	Metadata CertificateSigningRequestMetadata `tfsdk:"metadata"`
	Spec     *CertificateSigningRequestSpec    `tfsdk:"spec"`

	AutoApprove types.Bool   `tfsdk:"auto_approve"`
	Certificate types.String `tfsdk:"certificate"`
}

func NewCertificateSigningRequestEphemeralResource() ephemeral.EphemeralResource {
	return &CertificateSigningRequestEphemeralResource{}
}

func (r *CertificateSigningRequestEphemeralResource) Configure(ctx context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.SDKv2Meta = req.ProviderData.(func() any)
}

func (r *CertificateSigningRequestEphemeralResource) Metadata(ctx context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_certificate_signing_request_v1"
}

func (r *CertificateSigningRequestEphemeralResource) Schema(ctx context.Context, req ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	openapi := certificatesv1.CertificateSigningRequest{}.SwaggerDoc()
	openapiSpec := certificatesv1.CertificateSigningRequestSpec{}.SwaggerDoc()
	openapiStatus := certificatesv1.CertificateSigningRequestStatus{}.SwaggerDoc()

	resp.Schema = schema.Schema{
		Description: "TokenRequest requests a token for a given service account.",
		Attributes: map[string]schema.Attribute{
			"auto_approve": schema.BoolAttribute{
				Description: "Automatically approve the Certificate Signing Request",
				Optional:    true,
			},
			"certificate": schema.StringAttribute{
				Computed:    true,
				Optional:    true,
				Description: openapiStatus["certificate"],
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": schema.SingleNestedBlock{
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required: true,
					},
					"namespace": schema.StringAttribute{
						Required: true,
					},
				},
			},
			"spec": schema.SingleNestedBlock{
				Description: openapi[""],
				Attributes: map[string]schema.Attribute{
					"usages": schema.ListAttribute{
						Description: openapiSpec["usages"],
						Optional:    true,
						ElementType: types.StringType,
					},
					"expiration_seconds": schema.Int32Attribute{
						Description: openapiSpec["expirationSeconds"],
						Optional:    true,
					},
					"request": schema.StringAttribute{
						Description: openapiSpec["request"],
						Required:    true,
					},
					"signer_name": schema.StringAttribute{
						Description: openapiSpec["signerName"],
						Required:    true,
					},
				},
			},
		},
	}
}

func expandUsages(s []types.String) []certificatesv1.KeyUsage {
	ss := make([]certificatesv1.KeyUsage, len(s))
	for i := 0; i < len(s); i++ {
		ss[i] = certificatesv1.KeyUsage(s[i].ValueString())
	}
	return ss
}

const (
	TerraformAutoApproveReason  = "TerraformAutoApprove"
	TerraformAutoApproveMessage = "This CertificateSigningRequest was auto-approved by Terraform"
)

func (r *CertificateSigningRequestEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var data CertificateSigningRequestModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Metadata.Name.ValueString()
	namespace := data.Metadata.Namespace.ValueString()

	conn, err := r.SDKv2Meta().(kubernetes.KubeClientsets).MainClientset()
	if err != nil {
		panic(err.Error())
	}

	csr := certificatesv1.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: certificatesv1.CertificateSigningRequestSpec{
			ExpirationSeconds: data.Spec.ExpirationSeconds.ValueInt32Pointer(),
			Request:           []byte(data.Spec.Request.ValueString()),
			SignerName:        data.Spec.SignerName.ValueString(),
			Usages:            expandUsages(data.Spec.Usages),
		},
	}
	newcsr, err := conn.CertificatesV1().CertificateSigningRequests().Create(
		ctx, &csr, metav1.CreateOptions{})
	if err != nil {
		panic(err.Error())
	}
	defer conn.CertificatesV1().CertificateSigningRequests().Delete(
		ctx, csr.GetName(), metav1.DeleteOptions{})

	// auto approve the certificate
	if data.AutoApprove.IsNull() || data.AutoApprove.ValueBool() {
		err := kretry.RetryOnConflict(kretry.DefaultRetry, func() error {
			pendingCSR, err := conn.CertificatesV1().CertificateSigningRequests().Get(
				ctx, newcsr.GetName(), metav1.GetOptions{})
			if err != nil {
				return err
			}
			approval := certificatesv1.CertificateSigningRequestCondition{
				Status:  corev1.ConditionTrue,
				Type:    certificatesv1.CertificateApproved,
				Reason:  TerraformAutoApproveReason,
				Message: TerraformAutoApproveMessage,
			}
			pendingCSR.Status.Certificate = []byte{}
			pendingCSR.Status.Conditions = append(pendingCSR.Status.Conditions, approval)
			_, err = conn.CertificatesV1().CertificateSigningRequests().UpdateApproval(
				ctx, newcsr.GetName(), pendingCSR, metav1.UpdateOptions{})
			return err
		})
		if err != nil {
			panic(fmt.Sprintf("CSR auto-approve update failed: %v", err))
		}
	}

	// wait for the certificate to be issued
	waitingErr := fmt.Errorf("waiting")
	waitForIssue := kwait.Backoff{
		Steps:    5,
		Duration: 5 * time.Minute,
		Factor:   1.0,
		Jitter:   0.1,
	}
	err = kretry.OnError(waitForIssue, func(e error) bool { return true }, func() error {
		out, err := conn.CertificatesV1().CertificateSigningRequests().Get(ctx,
			newcsr.GetName(), metav1.GetOptions{})
		if err != nil {
			return err
		}

		for _, condition := range out.Status.Conditions {
			if condition.Type == certificatesv1.CertificateApproved &&
				len(out.Status.Certificate) > 0 {
				return nil
			}
		}
		return waitingErr
	})
	if err != nil {
		panic(err.Error())
	}

	issued, err := conn.CertificatesV1().CertificateSigningRequests().Get(ctx, newcsr.GetName(), metav1.GetOptions{})
	if err != nil {
		panic(err.Error())
	}
	data.Certificate = types.StringValue(string(issued.Status.Certificate))

	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
