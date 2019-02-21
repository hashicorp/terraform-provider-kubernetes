package kubernetes

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	api "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
	kubernetes "k8s.io/client-go/kubernetes"
)

// Use generated swagger docs from kubernetes' client-go to avoid copy/pasting them here
var (
	podDisruptionBudgetSpecDoc               = api.PodDisruptionBudget{}.SwaggerDoc()["spec"]
	podDisruptionBudgetSpecMaxUnavailableDoc = api.PodDisruptionBudget{}.SwaggerDoc()["maxUnavailable"]
	podDisruptionBudgetSpecMinAvailableDoc   = api.PodDisruptionBudget{}.SwaggerDoc()["minAvailable"]
	podDisruptionBudgetSpecSelectorDoc       = api.PodDisruptionBudget{}.SwaggerDoc()["selector"]
)

func resourceKubernetesPodDisruptionBudget() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesPodDisruptionBudgetCreate,
		Read:   resourceKubernetesPodDisruptionBudgetRead,
		Update: resourceKubernetesPodDisruptionBudgetUpdate,
		Delete: resourceKubernetesPodDisruptionBudgetDelete,
		Exists: resourceKubernetesPodDisruptionBudgetExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("pod disruption budget", true),
			// Updates to spec not allowed; have to delete and recreate
			// https://github.com/kubernetes/kubernetes/issues/45398
			"spec": {
				Type:        schema.TypeList,
				Description: podDisruptionBudgetSpecDoc,
				Required:    true,
				ForceNew:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"max_unavailable": {
							Type:         schema.TypeString,
							Description:  podDisruptionBudgetSpecMaxUnavailableDoc,
							Optional:     true,
							ForceNew:     true,
							ValidateFunc: validateTypeStringNullableIntOrPercent,
						},
						"min_available": {
							Type:         schema.TypeString,
							Description:  podDisruptionBudgetSpecMinAvailableDoc,
							Optional:     true,
							ForceNew:     true,
							ValidateFunc: validateTypeStringNullableIntOrPercent,
						},
						"selector": {
							Type:        schema.TypeList,
							Description: podDisruptionBudgetSpecSelectorDoc,
							Optional:    true,
							ForceNew:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: labelSelectorFields(false),
							},
						},
					},
				},
			},
		},
	}
}

func resourceKubernetesPodDisruptionBudgetUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("spec") {
		specOps, err := patchPodDisruptionBudgetSpec("spec.0.", "/spec", d)
		if err != nil {
			return err
		}
		ops = append(ops, *specOps...)
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}

	log.Printf("[INFO] Updating pod disruption budget %s: %s", d.Id(), ops)
	out, err := conn.PolicyV1beta1().PodDisruptionBudgets(namespace).Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Submitted updated pod disruption budget: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesPodDisruptionBudgetRead(d, meta)
}

func resourceKubernetesPodDisruptionBudgetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandPodDisruptionBudgetSpec(d.Get("spec").([]interface{}))
	if err != nil {
		return err
	}
	pdb := api.PodDisruptionBudget{
		ObjectMeta: metadata,
		Spec:       *spec,
	}

	log.Printf("[INFO] Creating new pod disruption budget: %#v", pdb)
	out, err := conn.PolicyV1beta1().PodDisruptionBudgets(metadata.Namespace).Create(&pdb)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Submitted new pod disruption budget: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesPodDisruptionBudgetRead(d, meta)
}

func resourceKubernetesPodDisruptionBudgetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading pod disruption budget %s", name)
	pdb, err := conn.PolicyV1beta1().PodDisruptionBudgets(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}

	log.Printf("[INFO] Received pod disruption budget: %#v", pdb)
	err = d.Set("metadata", flattenMetadata(pdb.ObjectMeta))
	if err != nil {
		return err
	}

	err = d.Set("spec", flattenPodDisruptionBudgetSpec(pdb.Spec))
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesPodDisruptionBudgetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Deleting pod disruption budget %#v", name)
	err = conn.PolicyV1beta1().PodDisruptionBudgets(namespace).Delete(name, &meta_v1.DeleteOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}

	log.Printf("[INFO] Pod disruption budget %#v deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesPodDisruptionBudgetExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking pod disruption budget %s", name)
	_, err = conn.PolicyV1beta1().PodDisruptionBudgets(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
