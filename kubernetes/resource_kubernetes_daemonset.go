package kubernetes

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/apis/extensions/v1beta1"
	kubernetes "k8s.io/kubernetes/pkg/client/clientset_generated/clientset"
)

func resourceKubernetesDaemonSet() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesDaemonSetCreate,
		Read:   resourceKubernetesDaemonSetRead,
		Exists: resourceKubernetesDaemonSetExists,
		Update: resourceKubernetesDaemonSetUpdate,
		Delete: resourceKubernetesDaemonSetDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("daemonset", true),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec defines the specification of the desired behavior of the daemonset. More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#spec-and-status",
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"min_ready_seconds": {
							Type:        schema.TypeInt,
							Description: "Minimum number of seconds for which a newly created pod should be ready without any of its container crashing, for it to be considered available. Defaults to 0 (pod will be considered available as soon as it is ready)",
							Optional:    true,
							Default:     0,
						},
						"selector": {
							Type:        schema.TypeMap,
							Description: "A label query over pods that should match the Replicas count. If Selector is empty, it is defaulted to the labels present on the Pod template. Label keys and values that must match in order to be controlled by this deployment, if empty defaulted to labels on Pod template. More info: http://kubernetes.io/docs/user-guide/labels#label-selectors",
							Required:    true,
						},
						"strategy": {
							Type:        schema.TypeList,
							Optional:    true,
							Computed:    true,
							Description: "Update strategy. One of RollingUpdate, Destroy. Defaults to RollingDate",
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:        schema.TypeString,
										Optional:    true,
										Computed:    true,
										Description: "Update strategy",
									},
									"rollingUpdate": {
										Type:        schema.TypeList,
										Description: "rolling update",
										Optional:    true,
										Computed:    true,
										MaxItems:    1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"maxSurge": {
													Type:        schema.TypeString,
													Description: "max surge",
													Optional:    true,
													Default:     1,
												},
												"maxUnavailable": {
													Type:        schema.TypeString,
													Description: "max unavailable",
													Optional:    true,
													Default:     1,
												},
											},
										},
									},
								},
							},
						},
						"template": {
							Type:        schema.TypeList,
							Description: "Describes the pod that will be created if insufficient replicas are detected. This takes precedence over a TemplateRef. More info: http://kubernetes.io/docs/user-guide/replication-controller#pod-template",
							Required:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: podSpecFields(false),
							},
						},
					},
				},
			},
		},
	}
}

func resourceKubernetesDaemonSetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandDaemonSetSpec(d.Get("spec").([]interface{}))
	if err != nil {
		return err
	}

	daemonset := v1beta1.DaemonSet{
		ObjectMeta: metadata,
		Spec:       spec,
	}

	log.Printf("[INFO] Creating new daemonset: %#v", daemonset)
	out, err := conn.DaemonSets(metadata.Namespace).Create(&daemonset)
	if err != nil {
		return fmt.Errorf("Failed to create daemonset: %s", err)
	}

	d.SetId(buildId(out.ObjectMeta))

	log.Printf("[INFO] Submitted new daemonset: %#v", out)

	return resourceKubernetesDaemonSetRead(d, meta)
}

func resourceKubernetesDaemonSetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Reading daemonset %s", name)
	daemonset, err := conn.DaemonSets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received daemonset: %#v", daemonset)

	err = d.Set("metadata", flattenMetadata(daemonset.ObjectMeta))
	if err != nil {
		return err
	}

	spec, err := flattenDaemonSetSpec(daemonset.Spec)
	if err != nil {
		return err
	}

	err = d.Set("spec", spec)
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesDaemonSetUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())

	ops := patchMetadata("metadata.0.", "/metadata/", d)

	if d.HasChange("spec") {
		spec, err := expandDaemonSetSpec(d.Get("spec").([]interface{}))
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
	log.Printf("[INFO] Updating daemonset %q: %v", name, string(data))
	out, err := conn.DaemonSets(namespace).Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return fmt.Errorf("Failed to update daemonset: %s", err)
	}
	log.Printf("[INFO] Submitted updated daemonset: %#v", out)

	return resourceKubernetesDaemonSetRead(d, meta)
}

func resourceKubernetesDaemonSetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Deleting daemonset: %#v", name)

	falseVar := false
	conn.DaemonSets(namespace).Delete(name, &metav1.DeleteOptions{OrphanDependents: &falseVar})

	log.Printf("[INFO] Replication controller %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesDaemonSetExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Checking daemonset %s", name)
	_, err := conn.DaemonSets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
