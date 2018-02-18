package kubernetes

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
	kubernetes "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/apis/apps/v1beta1"
)

func resourceKubernetesStatefulSet() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesStatefulSetCreate,
		Read:   resourceKubernetesStatefulSetRead,
		Update: resourceKubernetesStatefulSetUpdate,
		Delete: resourceKubernetesStatefulSetDelete,
		Exists: resourceKubernetesStatefulSetExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		SchemaVersion: 1,
		MigrateState:  resourceKubernetesStatefulSetStateUpgrader,
		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("statefulset", true),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec defines the specification of the desired behavior of the StatefulSet. More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#spec-and-status",
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"pod_management_policy": {
							Type:        schema.TypeString,
							Description: "Controls how pods are created during initial scale up, when replacing pods on nodes, or when scaling down. The default policy is OrderedReady, where pods are created in increasing order (pod-0, then pod-1, etc) and the controller will wait until each pod is ready before continuing. When scaling down, the pods are removed in the opposite order. The alternative policy is Parallel which will create pods in parallel to match the desired scale without waiting, and on scale down will delete all pods at once.",
							Optional:    true,
							Default:     "OrderedReady",
						},
						"replicas": {
							Type:        schema.TypeInt,
							Description: "The number of desired replicas. Defaults to 1. More info: http://kubernetes.io/docs/user-guide/replication-controller#what-is-a-replication-controller",
							Optional:    true,
							Default:     1,
						},
						"revision_history_limit": {
							Type:        schema.TypeInt,
							Description: "revisionHistoryLimit is the maximum number of revisions that will be maintained in the StatefulSet's revision history. The revision history consists of all revisions not represented by a currently applied StatefulSetSpec version. The default value is 10.",
							Optional:    true,
							Default:     10,
						},
						"selector": {
							Type:        schema.TypeMap,
							Description: "A label query over pods that should match the Replicas count. More info: http://kubernetes.io/docs/user-guide/labels#label-selectors",
							Required:    true,
						},
						"service_name": {
							Type:        schema.TypeString,
							Description: "The name of the service that governs this StatefulSet. This service must exist before the StatefulSet, and is responsible for the network identity of the set. Pods get DNS/hostnames that follow the pattern: pod-specific-string.serviceName.default.svc.cluster.local where \"pod-specific-string\" is managed by the StatefulSet controller.",
							Required:    true,
						},
						"template": {
							Type:        schema.TypeList,
							Description: "Template describes the pods that will be created.",
							Required:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"metadata": metadataSchema("statefulsetSpec", true),
									"spec": &schema.Schema{
										Type:        schema.TypeList,
										Description: "Template describes the pods that will be created.",
										Required:    true,
										MaxItems:    1,
										Elem: &schema.Resource{
											Schema: podSpecFields(true),
										},
									},
									"active_deadline_seconds":          relocatedAttribute("active_deadline_seconds"),
									"container":                        relocatedAttribute("container"),
									"dns_policy":                       relocatedAttribute("dns_policy"),
									"host_ipc":                         relocatedAttribute("host_ipc"),
									"host_network":                     relocatedAttribute("host_network"),
									"host_pid":                         relocatedAttribute("host_pid"),
									"hostname":                         relocatedAttribute("hostname"),
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
						"update_strategy": {
							Type:        schema.TypeList,
							Optional:    true,
							Computed:    true,
							Description: "updateStrategy indicates the StatefulSetUpdateStrategy that will be employed to update Pods in the StatefulSet when a revision is made to Template.",
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:        schema.TypeString,
										Optional:    true,
										Default:     "RollingUpdate",
										Description: "Type indicates the type of the StatefulSetUpdateStrategy. Default is RollingUpdate.",
									},
									"rolling_update": {
										Type:        schema.TypeList,
										Description: "RollingUpdate is used to communicate parameters when Type is RollingUpdateStatefulSetStrategyType.",
										Optional:    true,
										Computed:    true,
										MaxItems:    1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"partition": {
													Type:        schema.TypeInt,
													Description: "Partition indicates the ordinal at which the StatefulSet should be partitioned. Default value is 0.",
													Optional:    true,
													Default:     0,
												},
											},
										},
									},
								},
							},
						},
						"volume_claim_templates": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "volumeClaimTemplates is a list of claims that pods are allowed to reference. The StatefulSet controller is responsible for mapping network identities to claims in a way that maintains the identity of a pod. Every claim in this list must have at least one matching (by name) volumeMount in one container in the template. A claim in this list takes precedence over any volumes in the template, with the same name.",
							Elem: &schema.Resource{
								Schema: persistentVolumeClaimSpecFields(true),
							},
						},
					},
				},
			},
		},
	}
}

func resourceKubernetesStatefulSetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandStatefulSetSpec(d.Get("spec").([]interface{}))
	if err != nil {
		return err
	}

	//use name as label and selector if not set
	if metadata.Namespace == "" {
		metadata.Namespace = "default"
	}

	statefulSet := v1beta1.StatefulSet{
		ObjectMeta: metadata,
		Spec:       spec,
	}

	log.Printf("[INFO] Creating new Stateful Set: %#v", statefulSet)
	out, err := conn.AppsV1beta1().StatefulSets(metadata.Namespace).Create(&statefulSet)
	if err != nil {
		return fmt.Errorf("Failed to create Stateful Set: %s", err)
	}

	d.SetId(buildId(out.ObjectMeta))

	log.Printf("[DEBUG] Waiting for Stateful Set %s to schedule %d replicas",
		d.Id(), *out.Spec.Replicas)
	// 10 mins should be sufficient for scheduling ~10k replicas
	err = resource.Retry(d.Timeout(schema.TimeoutCreate),
		waitForStatefulSetReplicasFunc(conn, out.GetNamespace(), out.GetName()))
	if err != nil {
		return err
	}
	// We could wait for all pods to actually reach Ready state
	// but that means checking each pod status separately (which can be expensive at scale)
	// as there's no aggregate data available from the API

	log.Printf("[INFO] Submitted new statefulSet: %#v", out)

	return resourceKubernetesStatefulSetRead(d, meta)
}

func resourceKubernetesStatefulSetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	log.Printf("[INFO] Reading statefulSet %s", name)
	statefulSet, err := conn.AppsV1beta1().StatefulSets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received statefulSet: %#v", statefulSet)

	err = d.Set("metadata", flattenMetadata(statefulSet.ObjectMeta, d))
	if err != nil {
		return err
	}

	spec, err := flattenStatefulSetSpec(statefulSet.Spec, d)
	if err != nil {
		return err
	}

	err = d.Set("spec", spec)
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesStatefulSetUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())

	ops := patchMetadata("metadata.0.", "/metadata/", d)

	if d.HasChange("spec") {
		spec, err := expandStatefulSetSpec(d.Get("spec").([]interface{}))
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
	log.Printf("[INFO] Updating statefulSet %q: %v", name, string(data))
	out, err := conn.AppsV1beta1().StatefulSets(namespace).Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return fmt.Errorf("Failed to update statefulSet: %s", err)
	}
	log.Printf("[INFO] Submitted updated statefulSet: %#v", out)

	err = resource.Retry(d.Timeout(schema.TimeoutUpdate),
		waitForStatefulSetReplicasFunc(conn, namespace, name))
	if err != nil {
		return err
	}

	return resourceKubernetesStatefulSetRead(d, meta)
}

func resourceKubernetesStatefulSetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	log.Printf("[INFO] Deleting statefulSet: %#v", name)

	// Drain all replicas before deleting
	var ops PatchOperations
	ops = append(ops, &ReplaceOperation{
		Path:  "/spec/replicas",
		Value: 0,
	})
	data, err := ops.MarshalJSON()
	if err != nil {
		return err
	}
	_, err = conn.AppsV1beta1().StatefulSets(namespace).Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return err
	}

	// Wait until all replicas are gone
	err = resource.Retry(d.Timeout(schema.TimeoutDelete),
		waitForStatefulSetReplicasFunc(conn, namespace, name))
	if err != nil {
		return err
	}

	err = conn.AppsV1beta1().StatefulSets(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	log.Printf("[INFO] StatefulSet %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesStatefulSetExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking statefulSet %s", name)
	_, err = conn.AppsV1beta1().StatefulSets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

func waitForStatefulSetReplicasFunc(conn *kubernetes.Clientset, ns, name string) resource.RetryFunc {
	return func() *resource.RetryError {
		statefulSet, err := conn.AppsV1beta1().StatefulSets(ns).Get(name, metav1.GetOptions{})
		if err != nil {
			return resource.NonRetryableError(err)
		}

		desiredReplicas := *statefulSet.Spec.Replicas
		log.Printf("[DEBUG] Current number of labelled replicas of %q: %d (of %d)\n",
			statefulSet.GetName(), statefulSet.Status.Replicas, desiredReplicas)

		if statefulSet.Status.Replicas == desiredReplicas {
			return nil
		}

		return resource.RetryableError(fmt.Errorf("Waiting for %d replicas of %q to be scheduled (%d)",
			desiredReplicas, statefulSet.GetName(), statefulSet.Status.Replicas))
	}
}

func resourceKubernetesStatefulSetStateUpgrader(
	v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	if is.Empty() {
		log.Println("[DEBUG] Empty InstanceState; nothing to migrate.")
		return is, nil
	}

	var err error

	switch v {
	case 0:
		log.Println("[INFO] Found Kubernetes StatefulSet State schema v0; migrating to v1")
		is, err = migrateStateV0toV1(is)
		if err != nil {
			return is, err
		}

	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}

	return is, err
}
