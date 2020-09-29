package kubernetes

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"k8s.io/api/certificates/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"log"
	"reflect"
	"time"
)

func resourceKubernetesCertificateSigningRequest() *schema.Resource {
	apiDoc := v1beta1.CertificateSigningRequest{}.SwaggerDoc()
	apiDocSpec := v1beta1.CertificateSigningRequestSpec{}.SwaggerDoc()
	apiDocStatus := v1beta1.CertificateSigningRequestStatus{}.SwaggerDoc()

	return &schema.Resource{
		Create: resourceKubernetesCertificateSigningRequestCreate,
		Read:   resourceKubernetesCertificateSigningRequestRead,
		Delete: resourceKubernetesCertificateSigningRequestDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
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
							Type: schema.TypeString,
							// no swagger doc available for signerName
							Description: "Requested signer for the request. It is a qualified name in the form: `scope-hostname.io/name`." +
								"If empty, it will be defaulted: 1. If it's a kubelet client certificate, it is assigned `kubernetes.io/kube-apiserver-client-kubelet`." +
								"2. If it's a kubelet serving certificate, it is assigned `kubernetes.io/kubelet-serving`." +
								"3. Otherwise, it is assigned `kubernetes.io/legacy-unknown`. Distribution of trust for signers happens out of band." +
								"You can select on this field using `spec.signerName`.",
							Optional: true,
							ForceNew: true,
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

func resourceKubernetesCertificateSigningRequestCreate(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandCertificateSigningRequestSpec(d.Get("spec").([]interface{}))
	if err != nil {
		return err
	}

	csr := v1beta1.CertificateSigningRequest{
		ObjectMeta: metadata,
		Spec:       *spec,
	}
	log.Printf("[INFO] Creating new certificate signing request: %#v", csr)
	newCSR, createErr := conn.CertificatesV1beta1().CertificateSigningRequests().Create(ctx, &csr, metav1.CreateOptions{})
	if createErr != nil {
		return fmt.Errorf("Failed to create certificate signing request: %s", err)
	}

	// Get the name, since it might have been randomly generated during create.
	csrName := newCSR.ObjectMeta.Name

	// Delete the remote CSR resource when this function exits, or when errors are encountered.
	defer conn.CertificatesV1beta1().CertificateSigningRequests().Delete(ctx, csrName, metav1.DeleteOptions{})

	if d.Get("auto_approve").(bool) {
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			pendingCSR, getErr := conn.CertificatesV1beta1().CertificateSigningRequests().Get(ctx, csrName, metav1.GetOptions{})
			if getErr != nil {
				return getErr
			}
			approval := v1beta1.CertificateSigningRequestCondition{
				Type:    "Approved",
				Reason:  "TerraformAutoApprove",
				Message: "This CSR was approved by Terraform auto_approve.",
			}
			pendingCSR.Status.Certificate = []byte{}
			pendingCSR.Status.Conditions = append(pendingCSR.Status.Conditions, approval)
			_, updateErr := conn.CertificatesV1beta1().CertificateSigningRequests().UpdateApproval(ctx, pendingCSR, metav1.UpdateOptions{})
			return updateErr
		})
		if retryErr != nil {
			panic(fmt.Errorf("CSR auto-approve update failed: %v", retryErr))
		}
		fmt.Println("CSR auto-approve update succeeded")
	}

	log.Printf("[DEBUG] Waiting for certificate to be issued")
	stateConf := &resource.StateChangeConf{
		Target:  []string{"Issued"},
		Pending: []string{"", "Approved"},
		Timeout: d.Timeout(schema.TimeoutCreate),
		Refresh: func() (interface{}, string, error) {
			out, refreshErr := conn.CertificatesV1beta1().CertificateSigningRequests().Get(ctx, csrName, metav1.GetOptions{})
			if refreshErr != nil {
				log.Printf("[ERROR] Received error: %v", refreshErr)
				return out, "Error", refreshErr
			}
			var csrStatus string
			emptyStatus := v1beta1.CertificateSigningRequestStatus{}
			emptyCSR := v1beta1.CertificateSigningRequest{}

			// If the CSR is empty, check again later.
			if reflect.DeepEqual(emptyCSR, out) {
				return out, csrStatus, nil
			}

			// If the status is empty, check again later.
			if reflect.DeepEqual(emptyStatus, out.Status) {
				return out, csrStatus, nil
			}

			// Check to see if a certificate has been issued, and update status accordingly,
			// since 'Issued' is not a state ever populated in the Status Conditions.
			for _, condition := range out.Status.Conditions {
				log.Printf("[DEBUG] Found Status.Condition.Type: %v", condition.Type)
				if string(condition.Type) == "Approved" {
					if string(out.Status.Certificate) != "" {
						log.Printf("[DEBUG] Found non-empty Certificate field in Status")
						csrStatus = "Issued"
					}
				}
			}
			log.Printf("[DEBUG] CertificateSigningRequest %s status received: %#v", csrName, csrStatus)
			return out, csrStatus, nil
		},
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}
	log.Printf("[INFO] Certificate issued for request: %s", csrName)

	issued, err := conn.CertificatesV1beta1().CertificateSigningRequests().Get(ctx, csrName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	d.SetId(csrName)
	d.Set("certificate", string(issued.Status.Certificate))

	return resourceKubernetesCertificateSigningRequestRead(d, meta)
}

// resourceKubernetesCertificateSigningRequestRead does not return any data, because Read functions exist to
// sync the local state with the remote state. Since this data is local-only, there is nothing to read.
func resourceKubernetesCertificateSigningRequestRead(_ *schema.ResourceData, _ interface{}) error {
	return nil
}

func resourceKubernetesCertificateSigningRequestDelete(d *schema.ResourceData, _ interface{}) error {
	d.SetId("")
	return nil
}
