package kubernetes

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	api "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

func resourceKubernetesCustomResourceDefinition() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesCustomResourceDefinitionCreate,
		Read:   resourceKubernetesCustomResourceDefinitionRead,
		Exists: resourceKubernetesCustomResourceDefinitionExists,
		Update: resourceKubernetesCustomResourceDefinitionUpdate,
		Delete: resourceKubernetesCustomResourceDefinitionDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("custom resource definition", true),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec describes how the user wants the resources to appear",
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: customResourceDefinitionSpecFields(),
				},
			},
		},
	}
}

func resourceKubernetesCustomResourceDefinitionCreate(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).ApiextensionsClientset()
	if err != nil {
		return err
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandCustomResourceDefinitionSpec(d.Get("spec").([]interface{}))
	if err != nil {
		return err
	}

	customResourceDefinition := api.CustomResourceDefinition{
		ObjectMeta: metadata,
		Spec:       *spec,
	}
	log.Printf("[INFO] Creating new custom resource definition: %#v", customResourceDefinition)
	out, err := conn.ApiextensionsV1beta1().CustomResourceDefinitions().Create(&customResourceDefinition)
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted new custom resource definition: %#v", out)
	d.SetId(out.Name)

	return resourceKubernetesCustomResourceDefinitionRead(d, meta)
}

func resourceKubernetesCustomResourceDefinitionRead(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).ApiextensionsClientset()
	if err != nil {
		return err
	}

	name := d.Id()
	log.Printf("[INFO] Reading custom resource definition %s", name)
	customResourceDefinition, err := conn.ApiextensionsV1beta1().CustomResourceDefinitions().Get(name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received custom resource definition: %#v", customResourceDefinition)
	err = d.Set("metadata", flattenMetadata(customResourceDefinition.ObjectMeta, d))
	if err != nil {
		return err
	}

	spec := flattenCustomResourceDefinitionSpec(customResourceDefinition.Spec)

	err = d.Set("spec", spec)
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesCustomResourceDefinitionUpdate(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).ApiextensionsClientset()
	if err != nil {
		return err
	}

	name := d.Id()

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("spec") {
		spec, err := expandCustomResourceDefinitionSpec(d.Get("spec").([]interface{}))
		if err != nil {
			return err
		}
		ops = append(ops, &ReplaceOperation{
			Path:  "/spec",
			Value: spec,
		})
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating custom resource definition %q: %v", name, string(data))
	out, err := conn.ApiextensionsV1beta1().CustomResourceDefinitions().Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return fmt.Errorf("Failed to update Custom Resource Definition: %s", err)
	}
	log.Printf("[INFO] Submitted updated custom resource definition: %#v", out)
	d.SetId(out.ObjectMeta.Name)

	return resourceKubernetesCustomResourceDefinitionRead(d, meta)
}

func resourceKubernetesCustomResourceDefinitionDelete(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).ApiextensionsClientset()
	if err != nil {
		return err
	}

	name := d.Id()
	log.Printf("[INFO] Deleting custom resource definition: %#v", name)
	err = conn.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Custom resource definition %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesCustomResourceDefinitionExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).ApiextensionsClientset()
	if err != nil {
		return false, err
	}

	name := d.Id()

	log.Printf("[INFO] Checking custom resource definition %s", name)
	_, err = conn.ApiextensionsV1beta1().CustomResourceDefinitions().Get(name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
