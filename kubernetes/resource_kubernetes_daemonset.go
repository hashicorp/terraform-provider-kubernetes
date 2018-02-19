package kubernetes

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
	kubernetes "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
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
		SchemaVersion: 1,
		MigrateState:  resourceKubernetesDaemonSetStateUpgrader,

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
									"rolling_update": {
										Type:        schema.TypeList,
										Description: "rolling update",
										Optional:    true,
										Computed:    true,
										MaxItems:    1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"max_surge": {
													Type:        schema.TypeString,
													Description: "max surge",
													Optional:    true,
													Default:     1,
												},
												"max_unavailable": {
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
							Description: "Template describes the pods that will be created.",
							Required:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"metadata": metadataSchema("daemonsetSpec", true),
									"spec": &schema.Schema{
										Type:        schema.TypeList,
										Description: "Spec describes the pods that will be created.",
										Required:    true,
										MaxItems:    1,
										Elem: &schema.Resource{
											Schema: podSpecFields(false),
										},
									},
									"active_deadline_seconds":          relocatedAttribute("active_deadline_seconds"),
									"container":                        relocatedAttribute("container"),
									"dns_policy":                       relocatedAttribute("dns_policy"),
									"host_ipc":                         relocatedAttribute("host_ipc"),
									"host_network":                     relocatedAttribute("host_network"),
									"host_pid":                         relocatedAttribute("host_pid"),
									"hostname":                         relocatedAttribute("hostname"),
									"init_container":                   relocatedAttribute("init_container"),
									"node_name":                        relocatedAttribute("node_name"),
									"node_selector":                    relocatedAttribute("node_selector"),
									"restart_policy":                   relocatedAttribute("restart_policy"),
									"security_context":                 relocatedAttribute("security_context"),
									"service_account_name":             relocatedAttribute("service_account_name"),
									"automount_service_account_token":  relocatedAttribute("automount_service_account_token"),
									"subdomain":                        relocatedAttribute("subdomain"),
									"termination_grace_period_seconds": relocatedAttribute("termination_grace_period_seconds"),
									"volume": relocatedAttribute("volume"),
								},
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
	if metadata.Namespace == "" {
		metadata.Namespace = "default"
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

	namespace, name, err := idParts(d.Id())
	log.Printf("[INFO] Reading daemonset %s", name)
	daemonset, err := conn.DaemonSets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received daemonset: %#v", daemonset)

	err = d.Set("metadata", flattenMetadata(daemonset.ObjectMeta, d))
	if err != nil {
		return err
	}

	spec, err := flattenDaemonSetSpec(daemonset.Spec, d)
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

	namespace, name, err := idParts(d.Id())

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

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}
	log.Printf("[INFO] Deleting daemonset: %#v", name)

	falseVar := false
	conn.DaemonSets(namespace).Delete(name, &metav1.DeleteOptions{OrphanDependents: &falseVar})

	log.Printf("[INFO] DaemonSet %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesDaemonSetExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	log.Printf("[INFO] Checking daemonset %s", name)
	_, err = conn.DaemonSets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

func resourceKubernetesDaemonSetStateUpgrader(
	v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	if is.Empty() {
		log.Println("[DEBUG] Empty InstanceState; nothing to migrate.")
		return is, nil
	}

	var err error

	switch v {
	case 0:
		log.Println("[INFO] Found Kubernetes DaemonSet State v0; migrating to v1")
		is, err = migrateDaemonSetStateV0toV1(is)
		if err != nil {
			return is, err
		}

	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}

	return is, err
}

// This deployment resource originally had the podSpec directly below spec.template level
// This migration moves the state to spec.template.spec match the Kubernetes documented structure
func migrateDaemonSetStateV0toV1(is *terraform.InstanceState) (*terraform.InstanceState, error) {
	log.Printf("[DEBUG] Attributes before migration: %#v", is.Attributes)

	newTemplate := make(map[string]string)

	for k, v := range is.Attributes {
		log.Println("[DEBUG] - checking attribute for state upgrade: ", k, v)
		if strings.HasPrefix(k, "name") {
			// don't clobber an existing metadata.0.name value
			if _, ok := is.Attributes["metadata.0.name"]; ok {
				continue
			}

			newK := "metadata.0.name"

			newTemplate[newK] = v
			log.Printf("[DEBUG] moved attribute %s -> %s ", k, newK)
			delete(is.Attributes, k)

		} else if !strings.HasPrefix(k, "spec.0.template") {
			continue

		} else if strings.HasPrefix(k, "spec.0.template.0.spec") || strings.HasPrefix(k, "spec.0.template.0.metadata") {
			continue

		} else {
			newK := strings.Replace(k, "spec.0.template.0", "spec.0.template.0.spec.0", 1)

			newTemplate[newK] = v
			log.Printf("[DEBUG] moved attribute %s -> %s ", k, newK)
			delete(is.Attributes, k)
		}
	}

	for k, v := range newTemplate {
		is.Attributes[k] = v
	}

	log.Printf("[DEBUG] Attributes after migration: %#v", is.Attributes)
	return is, nil
}
