package kubernetes

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"

	copier "github.com/jinzhu/copier"
)

func resourceKubernetesMutatingWebhookConfiguration() *schema.Resource {
	apiDoc := admissionregistrationv1.MutatingWebhookConfiguration{}.SwaggerDoc()
	webhookDoc := admissionregistrationv1.MutatingWebhook{}.SwaggerDoc()
	return &schema.Resource{
		Create: resourceKubernetesMutatingWebhookConfigurationCreate,
		Read:   resourceKubernetesMutatingWebhookConfigurationRead,
		Exists: resourceKubernetesMutatingWebhookConfigurationExists,
		Update: resourceKubernetesMutatingWebhookConfigurationUpdate,
		Delete: resourceKubernetesMutatingWebhookConfigurationDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("validating webhook configuration", true),
			"webhook": {
				Type:        schema.TypeList,
				Description: apiDoc["webhooks"],
				Required:    true,
				MinItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"admission_review_versions": {
							Type:        schema.TypeList,
							Description: webhookDoc["admissionReviewVersions"],
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"client_config": {
							Type:        schema.TypeList,
							Description: webhookDoc["clientConfig"],
							Required:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: webhookClientConfigFields(),
							},
						},
						"failure_policy": {
							Type:        schema.TypeString,
							Description: webhookDoc["failurePolicy"],
							Optional:    true,
							Default:     "Fail",
						},
						"match_policy": {
							Type:        schema.TypeString,
							Description: webhookDoc["matchPolicy"],
							Optional:    true,
							Default:     "Equivalent",
						},
						"name": {
							Type:        schema.TypeString,
							Description: webhookDoc["name"],
							Required:    true,
						},
						"namespace_selector": {
							Type:        schema.TypeList,
							Description: webhookDoc["namespaceSelector"],
							Optional:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: labelSelectorFields(true),
							},
						},
						"object_selector": {
							Type:        schema.TypeList,
							Description: webhookDoc["objectSelector"],
							Optional:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: labelSelectorFields(true),
							},
						},
						"reinvocation_policy": {
							Type:        schema.TypeString,
							Description: webhookDoc["reinvocationPolicy"],
							Optional:    true,
							Default:     "Never",
						},
						"rule": {
							Type:        schema.TypeList,
							Description: webhookDoc["rules"],
							Required:    true,
							MinItems:    1,
							Elem: &schema.Resource{
								Schema: ruleWithOperationsFields(),
							},
						},
						"side_effects": {
							Type:        schema.TypeString,
							Description: webhookDoc["sideEffects"],
							Optional:    true,
						},
						"timeout_seconds": {
							Type:        schema.TypeInt,
							Description: webhookDoc["timeoutSeconds"],
							Default:     10,
							Optional:    true,
						},
					},
				},
			},
		},
	}
}

func resourceKubernetesMutatingWebhookConfigurationCreate(d *schema.ResourceData, meta interface{}) error {
	var err error

	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}

	cfg := admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: expandMetadata(d.Get("metadata").([]interface{})),
		Webhooks:   expandMutatingWebhooks(d.Get("webhook").([]interface{})),
	}

	log.Printf("[INFO] Creating new MutatingWebhookConfiguration: %#v", cfg)

	res := &admissionregistrationv1.MutatingWebhookConfiguration{}

	usev1beta1, err := useAdmissionregistrationV1beta1(conn)
	if err != nil {
		return err
	}
	if usev1beta1 {
		resv1beta1 := &admissionregistrationv1beta1.MutatingWebhookConfiguration{}
		reqv1beta1 := &admissionregistrationv1beta1.MutatingWebhookConfiguration{}
		copier.Copy(reqv1beta1, cfg)
		resv1beta1, err = conn.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Create(reqv1beta1)
		copier.Copy(res, resv1beta1)
	} else {
		res, err = conn.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(&cfg)
	}

	if err != nil {
		return err
	}

	log.Printf("[INFO] Submitted new MutatingWebhookConfiguration: %#v", res)

	d.SetId(res.Name)

	return resourceKubernetesMutatingWebhookConfigurationRead(d, meta)
}

func resourceKubernetesMutatingWebhookConfigurationRead(d *schema.ResourceData, meta interface{}) error {
	var err error

	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}

	name := d.Id()

	cfg := &admissionregistrationv1.MutatingWebhookConfiguration{}

	log.Printf("[INFO] Reading MutatingWebhookConfiguration %s", name)
	usev1beta1, err := useAdmissionregistrationV1beta1(conn)
	if err != nil {
		return err
	}
	if usev1beta1 {
		cfgv1beta1 := &admissionregistrationv1beta1.MutatingWebhookConfiguration{}
		cfgv1beta1, err = conn.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Get(name, metav1.GetOptions{})
		copier.Copy(cfg, cfgv1beta1)
	} else {
		cfg, err = conn.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(name, metav1.GetOptions{})
	}
	if err != nil {
		return err
	}

	err = d.Set("metadata", flattenMetadata(cfg.ObjectMeta, d))
	if err != nil {
		return nil
	}

	log.Printf("[DEBUG] Setting webhook to: %#v", cfg.Webhooks)

	err = d.Set("webhook", flattenMutatingWebhooks(cfg.Webhooks))
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesMutatingWebhookConfigurationUpdate(d *schema.ResourceData, meta interface{}) error {
	var err error

	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)

	if d.HasChange("webhook") {
		op := &ReplaceOperation{
			Path: "/webhooks",
		}

		patch := expandMutatingWebhooks(d.Get("webhook").([]interface{}))

		usev1beta1, err := useAdmissionregistrationV1beta1(conn)
		if err != nil {
			return err
		}
		if usev1beta1 {
			patchv1beta1 := []admissionregistrationv1beta1.MutatingWebhook{}
			copier.Copy(&patchv1beta1, &patch)
			op.Value = patchv1beta1
		} else {
			op.Value = patch
		}

		ops = append(ops, op)
	}

	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}

	name := d.Id()
	log.Printf("[INFO] Updating MutatingWebhookCoßnfiguration %q: %v", name, string(data))

	res := &admissionregistrationv1.MutatingWebhookConfiguration{}

	usev1beta1, err := useAdmissionregistrationV1beta1(conn)
	if err != nil {
		return err
	}
	if usev1beta1 {
		resv1beta1 := &admissionregistrationv1beta1.MutatingWebhookConfiguration{}
		resv1beta1, err = conn.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Patch(name, types.JSONPatchType, data)
		copier.Copy(res, resv1beta1)
	} else {
		res, err = conn.AdmissionregistrationV1().MutatingWebhookConfigurations().Patch(name, types.JSONPatchType, data)
	}
	if err != nil {
		return fmt.Errorf("Failed to update MutatingWebhookConfiguration: %s", err)
	}

	log.Printf("[INFO] Submitted updated MutatingWebhookConfiguration: %#v", res)

	return resourceKubernetesMutatingWebhookConfigurationRead(d, meta)
}

func resourceKubernetesMutatingWebhookConfigurationDelete(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}

	name := d.Id()

	log.Printf("[INFO] Deleting MutatingWebhookConfiguration: %#v", name)
	usev1beta1, err := useAdmissionregistrationV1beta1(conn)
	if err != nil {
		return err
	}
	if usev1beta1 {
		err = conn.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Delete(name, &metav1.DeleteOptions{})
	} else {
		err = conn.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(name, &metav1.DeleteOptions{})
	}
	if err != nil {
		return err
	}

	log.Printf("[INFO] MutatingWebhookConfiguration %#v is deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesMutatingWebhookConfigurationExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	name := d.Id()

	log.Printf("[INFO] Checking MutatingWebhookConfiguration %s", name)

	usev1beta1, err := useAdmissionregistrationV1beta1(conn)
	if err != nil {
		return false, err
	}
	if usev1beta1 {
		_, err = conn.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Get(name, metav1.GetOptions{})
	} else {
		_, err = conn.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(name, metav1.GetOptions{})
	}

	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
