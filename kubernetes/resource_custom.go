package kubernetes

import (
	"fmt"
	"log"
	"strings"
	"time"

	flat "github.com/hashicorp/terraform/flatmap"
	"github.com/hashicorp/terraform/helper/schema"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/ghodss/yaml"
)

const (
	customResourceEditAddPrefix    = "transitent-add."
	customResourceEditUpdatePrefix = "transitent-update."
	customResourceEditDeletePrefix = "transitent-delete."
)

func resourceKubernetesCustom() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesCustomCreate,
		Read:   resourceKubernetesCustomRead,
		Update: resourceKubernetesCustomUpdate,
		Delete: resourceKubernetesCustomDelete,
		Exists: resourceKubernetesCustomExists,

		CustomizeDiff: resourceKubernetesCustomDiff,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"api_version": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "API Version of the custom resource",
			},
			"kind": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The formal type of the custom resource",
			},
			"metadata": &schema.Schema{
				Type:        schema.TypeList,
				Description: fmt.Sprintf("Standard GenericResource's metadata. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#metadata"),
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: metadataFields("GenericResource"),
				},
			},
			"yaml": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Resource spec as raw yaml, this configuration is merged with spec map.",
			},
			"spec": {
				Type:        schema.TypeMap,
				Description: "Specification of the desired behavior of the custom resource.",
				Computed:    true,
			},
		},
	}
}

func getCustomResourceFromKubernetes(id, kind string, meta interface{}) (*GenericKubernetesObject, error) {
	conn := meta.(ExtendedClientset)

	genericID := genericObjectID(id, kind)
	if genericID == nil {
		return nil, nil
	}

	if genericID.Namespace == "" {
		log.Printf("[INFO] Checking custom %s\n", genericID.Name)
	} else {
		log.Printf("[INFO] Checking custom %s in %s\n", genericID.Name, genericID.Namespace)
	}

	result, err := conn.CustomResource().Get(genericID)
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return nil, err
	}
	return result, err
}

func resourceKubernetesCustomCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(ExtendedClientset)

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	genericID := genericObjectID(d.Get("api_version").(string), d.Get("kind").(string))

	base := map[string]interface{}{
		"kind":       genericID.Kind,
		"apiVersion": genericID.ApiVersion(),
		"metadata": map[string]interface{}{
			"name": metadata.Name,
		},
	}

	spec := map[string]interface{}{}
	base["spec"] = spec

	yamlSpec := d.Get("yaml").(string)
	if err := yaml.Unmarshal([]byte(yamlSpec), &spec); err != nil {
		return fmt.Errorf("---> %v %s", err, yamlSpec)
	}

	result, err := conn.CustomResource().Create(genericID, base)
	if err != nil {
		return err
	}

	d.SetId(result.ID())

	err = d.Set("spec", result.FlattenedSpec())
	if err != nil {
		return err
	}

	err = d.Set("metadata", flattenMetadata(*result.ObjectMeta))
	if err != nil {
		return err
	}

	log.Printf("[INFO] Created new custom: %#v", d)
	return nil
}

func resourceKubernetesCustomDiff(d *schema.ResourceDiff, meta interface{}) error {
	resource, err := getCustomResourceFromKubernetes(d.Id(), d.Get("kind").(string), meta)
	if resource == nil {
		return nil
	} else if err != nil {
		return err
	}

	fullSpec := map[string]interface{}{}
	err = yaml.Unmarshal([]byte(d.Get("yaml").(string)), &fullSpec)
	if err != nil {
		return err
	}

	desiredSpec := flat.Flatten(fullSpec)
	kubernetesSpec := resource.FlattenedSpec()

	deletedSpec := map[string]string{}
	for key, value := range kubernetesSpec {
		if _, ok := desiredSpec[key]; !ok {
			deletedSpec[key] = fmt.Sprintf("'%s' --> ''", value)
		}
	}

	for k, v := range deletedSpec {
		kubernetesSpec[fmt.Sprintf("%s%s", customResourceEditDeletePrefix, k)] = v
		delete(kubernetesSpec, k)
	}

	for key, value := range desiredSpec {
		if kube, ok := kubernetesSpec[key]; !ok {
			kubernetesSpec[key] = value
			kubernetesSpec[fmt.Sprintf("%s%s", customResourceEditAddPrefix, key)] = value
		} else if value != kube {
			kubernetesSpec[key] = value
			kubernetesSpec[fmt.Sprintf("%s%s", customResourceEditUpdatePrefix, key)] = fmt.Sprintf("'%s' --> '%s'", kube, value)
		}
	}

	err = d.SetNew("spec", kubernetesSpec)
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesCustomUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(ExtendedClientset)

	genericID := genericObjectID(d.Id(), d.Get("kind").(string))
	ops := patchMetadata("metadata.0.", "/metadata/", d)

	if d.HasChange("spec") {
		spec := d.Get("spec").(map[string]interface{})
		for k := range spec {
			if strings.HasPrefix(k, customResourceEditAddPrefix) {
				key := strings.Replace(k, customResourceEditAddPrefix, "", 1)
				ops = append(ops, &AddOperation{
					Path:  fmt.Sprintf("/spec/%s", strings.Replace(key, ".", "/", -1)),
					Value: spec[key],
				})
			} else if strings.HasPrefix(k, customResourceEditUpdatePrefix) {
				key := strings.Replace(k, customResourceEditUpdatePrefix, "", 1)
				ops = append(ops, &ReplaceOperation{
					Path:  fmt.Sprintf("/spec/%s", strings.Replace(key, ".", "/", -1)),
					Value: spec[key],
				})
			} else if strings.HasPrefix(k, customResourceEditDeletePrefix) {
				ops = append(ops, &RemoveOperation{
					Path: strings.Replace(strings.Replace(k, customResourceEditDeletePrefix, "/spec/", 1), ".", "/", -1),
				})
			}
		}
	}

	result, err := conn.CustomResource().Update(genericID, ops)
	if err != nil {
		return err
	}

	d.SetId(result.ID())

	err = d.Set("spec", result.FlattenedSpec())
	if err != nil {
		return err
	}

	err = d.Set("metadata", flattenMetadata(*result.ObjectMeta))
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesCustomRead(d *schema.ResourceData, meta interface{}) error {
	result, err := getCustomResourceFromKubernetes(d.Id(), d.Get("kind").(string), meta)
	if err != nil {
		return err
	}

	d.SetId(result.ID())

	err = d.Set("metadata", flattenMetadata(*result.ObjectMeta))
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesCustomDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(ExtendedClientset)

	genericID := genericObjectID(d.Id(), d.Get("kind").(string))
	err := conn.CustomResource().Delete(genericID)
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}

	d.SetId("")
	log.Printf("[INFO] Deleted %s", genericID.ID())

	return nil
}

func resourceKubernetesCustomExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(ExtendedClientset)

	genericID := genericObjectID(d.Id(), d.Get("kind").(string))
	if genericID == nil {
		return false, nil
	}

	if genericID.Namespace == "" {
		log.Printf("[INFO] Checking custom %s\n", genericID.Name)
	} else {
		log.Printf("[INFO] Checking custom %s in %s\n", genericID.Name, genericID.Namespace)
	}

	_, err := conn.CustomResource().Get(genericID)
	if err != nil {
		if statusErr, ok := err.(*apiErrors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
