// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	certificates "k8s.io/api/certificates/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kretry "k8s.io/client-go/util/retry"
)

func resourceKubernetesCertificateSigningRequestV1() *schema.Resource {
	apiDoc := certificates.CertificateSigningRequest{}.SwaggerDoc()
	apiDocSpec := certificates.CertificateSigningRequestSpec{}.SwaggerDoc()
	apiDocStatus := certificates.CertificateSigningRequestStatus{}.SwaggerDoc()

	return &schema.Resource{
		Description:   "Use this resource to generate TLS certificates using Kubernetes. This is a *logical resource*, so it contributes only to the current Terraform state and does not persist any external managed resources. This resource enables automation of [X.509](https://www.itu.int/rec/T-REC-X.509) credential provisioning (including TLS/SSL certificates). It does this by creating a CertificateSigningRequest using the Kubernetes API, which generates a certificate from the Certificate Authority (CA) configured in the Kubernetes cluster. The CSR can be approved automatically by Terraform, or it can be approved by a custom controller running in Kubernetes. See [Kubernetes reference](https://kubernetes.io/docs/reference/access-authn-authz/certificate-signing-requests/) for all available options pertaining to CertificateSigningRequests.",
		CreateContext: resourceKubernetesCertificateSigningRequestV1Create,
		ReadContext:   resourceKubernetesCertificateSigningRequestV1Read,
		DeleteContext: resourceKubernetesCertificateSigningRequestV1Delete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			"auto_approve": {
				Type:        schema.TypeBool,
				Description: "Automatically approve the CertificateSigningRequest",
				Optional:    true,
				ForceNew:    true,
				Default:     true,
			},
			"certificate": {
				Type:        schema.TypeString,
				Description: apiDocStatus["certificate"],
				Computed:    true,
			},
			"metadata": metadataSchemaForceNew(metadataSchema("certificate signing request", true)),
			"spec": {
				ForceNew:    true,
				Type:        schema.TypeList,
				Description: apiDoc[""],
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"request": {
							Type:        schema.TypeString,
							Description: apiDocSpec["request"],
							Required:    true,
							ForceNew:    true,
						},
						"signer_name": {
							Type:        schema.TypeString,
							Description: apiDocSpec["signerName"],
							Required:    true,
							ForceNew:    true,
						},
						"usages": {
							Type:        schema.TypeSet,
							Description: apiDocSpec["usages"],
							Set:         schema.HashString,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Optional:    true,
							ForceNew:    true,
						},
					},
				},
			},
		},
	}
}

const TerraformAutoApproveReason = "TerraformAutoApprove"
const TerraformAutoApproveMessage = "This CertificateSigningRequest was auto-approved by Terraform"

func resourceKubernetesCertificateSigningRequestV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec := expandCertificateSigningRequestV1Spec(d.Get("spec").([]interface{}))

	csr := certificates.CertificateSigningRequest{
		ObjectMeta: metadata,
		Spec:       *spec,
	}
	log.Printf("[INFO] Creating new certificate signing request: %#v", csr)
	newCSR, err := conn.CertificatesV1().CertificateSigningRequests().Create(ctx, &csr, metav1.CreateOptions{})
	if err != nil {
		return diag.Errorf("Failed to create certificate signing request: %s", err)
	}

	// Get the name, since it might have been randomly generated during create.
	csrName := newCSR.ObjectMeta.Name

	// Delete the remote CSR resource when this function exits, or when errors are encountered.
	defer conn.CertificatesV1().CertificateSigningRequests().Delete(ctx, csrName, metav1.DeleteOptions{})

	if d.Get("auto_approve").(bool) {
		retryErr := kretry.RetryOnConflict(kretry.DefaultRetry, func() error {
			pendingCSR, getErr := conn.CertificatesV1().CertificateSigningRequests().Get(
				ctx, csrName, metav1.GetOptions{})
			if getErr != nil {
				return getErr
			}
			approval := certificates.CertificateSigningRequestCondition{
				Status:  v1.ConditionTrue,
				Type:    certificates.CertificateApproved,
				Reason:  TerraformAutoApproveReason,
				Message: TerraformAutoApproveMessage,
			}
			pendingCSR.Status.Certificate = []byte{}
			pendingCSR.Status.Conditions = append(pendingCSR.Status.Conditions, approval)
			_, err := conn.CertificatesV1().CertificateSigningRequests().UpdateApproval(
				ctx, csrName, pendingCSR, metav1.UpdateOptions{})
			return err
		})
		if retryErr != nil {
			return diag.Errorf("CSR auto-approve update failed: %v", retryErr)
		}
		log.Printf("[DEBUG] Auto approve succeeded")
	}

	log.Printf("[DEBUG] Waiting for certificate to be issued")
	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutCreate), func() *retry.RetryError {
		out, err := conn.CertificatesV1().CertificateSigningRequests().Get(ctx, csrName, metav1.GetOptions{})
		if err != nil {
			log.Printf("[ERROR] Received error: %v", err)
			return retry.NonRetryableError(err)
		}

		// Check to see if a certificate has been issued, and update status accordingly,
		// since 'Issued' is not a state ever populated in the Status Conditions.
		for _, condition := range out.Status.Conditions {
			log.Printf("[DEBUG] Found Status.Condition.Type: %v", condition.Type)
			if condition.Type == certificates.CertificateApproved &&
				len(out.Status.Certificate) > 0 {
				log.Printf("[DEBUG] Found non-empty Certificate field in Status")
				return nil

			}
		}
		log.Printf("[DEBUG] CertificateSigningRequest %s status received: %#v", csrName, out.Status)
		return retry.RetryableError(errors.New("Waiting for certificate to be issued"))
	})
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Certificate issued for request: %s", csrName)

	issued, err := conn.CertificatesV1().CertificateSigningRequests().Get(ctx, csrName, metav1.GetOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(csrName)
	d.Set("certificate", string(issued.Status.Certificate))

	return resourceKubernetesCertificateSigningRequestV1Read(ctx, d, meta)
}

// resourceKubernetesCertificateSigningRequestV1Read does not return any data, because Read functions exist to
// sync the local state with the remote state. Since this data is local-only, there is nothing to read.
func resourceKubernetesCertificateSigningRequestV1Read(ctx context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
	return diag.Diagnostics{}
}

func resourceKubernetesCertificateSigningRequestV1Delete(ctx context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
	d.SetId("")
	return diag.Diagnostics{}
}
