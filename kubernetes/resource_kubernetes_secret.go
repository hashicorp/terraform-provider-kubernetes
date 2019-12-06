package kubernetes

import (
	"encoding/base64"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

func resourceKubernetesSecret() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesSecretCreate,
		Read:   resourceKubernetesSecretRead,
		Exists: resourceKubernetesSecretExists,
		Update: resourceKubernetesSecretUpdate,
		Delete: resourceKubernetesSecretDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("secret", true),
			"data": {
				Type:        schema.TypeMap,
				Description: "A map of the secret data.",
				Optional:    true,
				Sensitive:   true,
			},
			"base64data": {
				Type:        schema.TypeMap,
				Description: "A map of the base64-encoded secret data.",
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
				Sensitive:   true,
			},
			"type": {
				Type:        schema.TypeString,
				Description: "Type of secret",
				Default:     "Opaque",
				Optional:    true,
				ForceNew:    true,
			},
		},
	}
}

func decodeBase64Value(value interface{}) ([]byte, error) {
	enc, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("base64data cannot decode type %T", value)
	}
	return base64.StdEncoding.DecodeString(enc)
}

func resourceKubernetesSecretCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*KubeClientsets).MainClientset

	// Merge data and base64-encoded data into a single data map
	dataMap := d.Get("data").(map[string]interface{})
	for key, value := range d.Get("base64data").(map[string]interface{}) {
		// Decode Terraform's base64 representation to avoid double-encoding in Kubernetes.
		decodedValue, err := decodeBase64Value(value)
		if err != nil {
			return err
		}
		dataMap[key] = string(decodedValue)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	secret := api.Secret{
		ObjectMeta: metadata,
		Data:       expandStringMapToByteMap(dataMap),
	}

	if v, ok := d.GetOk("type"); ok {
		secret.Type = api.SecretType(v.(string))
	}

	log.Printf("[INFO] Creating new secret: %#v", secret)
	out, err := conn.CoreV1().Secrets(metadata.Namespace).Create(&secret)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Submitting new secret: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesSecretRead(d, meta)
}

func resourceKubernetesSecretRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*KubeClientsets).MainClientset

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading secret %s", name)
	secret, err := conn.CoreV1().Secrets(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Received secret: %#v", secret)
	err = d.Set("metadata", flattenMetadata(secret.ObjectMeta, d))
	if err != nil {
		return err
	}

	d.Set("type", secret.Type)

	secretData := flattenByteMapToStringMap(secret.Data)
	// Remove base64data keys from the payload before setting the data key on the resource. If
	// these keys are not removed, they will always show in the diff at update.
	for key, value := range d.Get("base64data").(map[string]interface{}) {
		if _, err := decodeBase64Value(value); err != nil {
			continue
		}
		delete(secretData, key)
	}
	d.Set("data", secretData)

	return nil
}

func resourceKubernetesSecretUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*KubeClientsets).MainClientset

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("data") || d.HasChange("base64data") {
		oldV, newV := d.GetChange("data")
		oldV = base64EncodeStringMap(oldV.(map[string]interface{}))
		newV = base64EncodeStringMap(newV.(map[string]interface{}))

		oldVB64, newVB64 := d.GetChange("base64data")
		for key, value := range oldVB64.(map[string]interface{}) {
			oldV.(map[string]interface{})[key] = value
		}
		for key, value := range newVB64.(map[string]interface{}) {
			newV.(map[string]interface{})[key] = value
		}

		diffOps := diffStringMap("/data/", oldV.(map[string]interface{}), newV.(map[string]interface{}))

		ops = append(ops, diffOps...)
	}

	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}

	log.Printf("[INFO] Updating secret %q: %v", name, data)
	out, err := conn.CoreV1().Secrets(namespace).Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return fmt.Errorf("Failed to update secret: %s", err)
	}

	log.Printf("[INFO] Submitting updated secret: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesSecretRead(d, meta)
}

func resourceKubernetesSecretDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*KubeClientsets).MainClientset

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Deleting secret: %q", name)
	err = conn.CoreV1().Secrets(namespace).Delete(name, &meta_v1.DeleteOptions{})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Secret %s deleted", name)

	d.SetId("")

	return nil
}

func resourceKubernetesSecretExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*KubeClientsets).MainClientset

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking secret %s", name)
	_, err = conn.CoreV1().Secrets(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}

	return true, err
}
