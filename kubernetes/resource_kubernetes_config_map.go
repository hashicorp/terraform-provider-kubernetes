package kubernetes

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

func resourceKubernetesConfigMap() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesConfigMapCreate,
		Read:   resourceKubernetesConfigMapRead,
		Exists: resourceKubernetesConfigMapExists,
		Update: resourceKubernetesConfigMapUpdate,
		Delete: resourceKubernetesConfigMapDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("config map", true),
			"binary_data": {
				Type:         schema.TypeMap,
				Description:  "BinaryData contains the binary data. Each key must consist of alphanumeric characters, '-', '_' or '.'. BinaryData can contain byte sequences that are not in the UTF-8 range. The keys stored in BinaryData must not overlap with the ones in the Data field, this is enforced during validation process. Using this field will require 1.10+ apiserver and kubelet. This field only accepts base64-encoded payloads that will be decoded/encoded before being sent/received to/from the apiserver.",
				Optional:     true,
				ValidateFunc: validateBase64EncodedMap,
			},
			"data": {
				Type:        schema.TypeMap,
				Description: "Data contains the configuration data. Each key must consist of alphanumeric characters, '-', '_' or '.'. Values with non-UTF-8 byte sequences must use the BinaryData field. The keys stored in Data must not overlap with the keys in the BinaryData field, this is enforced during validation process.",
				Optional:    true,
			},
		},
	}
}

func resourceKubernetesConfigMapCreate(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	cfgMap := api.ConfigMap{
		ObjectMeta: metadata,
		BinaryData: expandBase64MapToByteMap(d.Get("binary_data").(map[string]interface{})),
		Data:       expandStringMap(d.Get("data").(map[string]interface{})),
	}
	log.Printf("[INFO] Creating new config map: %#v", cfgMap)
	out, err := conn.CoreV1().ConfigMaps(metadata.Namespace).Create(&cfgMap)
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted new config map: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesConfigMapRead(d, meta)
}

func resourceKubernetesConfigMapRead(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}
	log.Printf("[INFO] Reading config map %s", name)
	cfgMap, err := conn.CoreV1().ConfigMaps(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received config map: %#v", cfgMap)
	err = d.Set("metadata", flattenMetadata(cfgMap.ObjectMeta, d))
	if err != nil {
		return err
	}

	d.Set("binary_data", flattenByteMapToBase64Map(cfgMap.BinaryData))
	d.Set("data", cfgMap.Data)

	return nil
}

func resourceKubernetesConfigMapUpdate(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("binary_data") {
		oldV, newV := d.GetChange("binary_data")
		diffOps := diffStringMap("/binaryData/", oldV.(map[string]interface{}), newV.(map[string]interface{}))
		ops = append(ops, diffOps...)
	}

	if d.HasChange("data") {
		oldV, newV := d.GetChange("data")
		diffOps := diffStringMap("/data/", oldV.(map[string]interface{}), newV.(map[string]interface{}))
		ops = append(ops, diffOps...)
	}

	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}

	log.Printf("[INFO] Updating config map %q: %v", name, string(data))
	out, err := conn.CoreV1().ConfigMaps(namespace).Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return fmt.Errorf("Failed to update Config Map: %s", err)
	}
	log.Printf("[INFO] Submitted updated config map: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesConfigMapRead(d, meta)
}

func resourceKubernetesConfigMapDelete(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}
	log.Printf("[INFO] Deleting config map: %#v", name)
	err = conn.CoreV1().ConfigMaps(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Config map %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesConfigMapExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking config map %s", name)
	_, err = conn.CoreV1().ConfigMaps(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
