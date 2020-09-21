package kubernetes

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kubernetes "k8s.io/client-go/kubernetes"
	polymorphichelpers "k8s.io/kubectl/pkg/polymorphichelpers"
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
		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("stateful set", true),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec defines the desired identities of pods in this set.",
				Required:    true,
				MaxItems:    1,
				MinItems:    1,
				ForceNew:    true,
				Elem: &schema.Resource{
					Schema: statefulSetSpecFields(false),
				},
			},
			"wait_for_rollout": {
				Type:        schema.TypeBool,
				Description: "Wait for the rollout of the stateful set to complete. Defaults to false.",
				Default:     false,
				Optional:    true,
			},
		},
	}
}

func resourceKubernetesStatefulSetCreate(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandStatefulSetSpec(d.Get("spec").([]interface{}))
	if err != nil {
		return err
	}
	statefulSet := appsv1.StatefulSet{
		ObjectMeta: metadata,
		Spec:       *spec,
	}
	log.Printf("[INFO] Creating new StatefulSet: %#v", statefulSet)

	out, err := conn.AppsV1().StatefulSets(metadata.Namespace).Create(ctx, &statefulSet, metav1.CreateOptions{})

	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted new StatefulSet: %#v", out)

	id := buildId(out.ObjectMeta)
	d.SetId(id)

	log.Printf("[INFO] StatefulSet %s created", id)

	if d.Get("wait_for_rollout").(bool) {
		log.Printf("[INFO] Waiting for StatefulSet %s to rollout", id)
		namespace := out.ObjectMeta.Namespace
		name := out.ObjectMeta.Name
		return resource.Retry(d.Timeout(schema.TimeoutCreate),
			retryUntilStatefulSetRolloutComplete(ctx, conn, namespace, name))
	}

	return resourceKubernetesStatefulSetRead(d, meta)
}

func resourceKubernetesStatefulSetExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}
	ctx := context.TODO()

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking StatefulSet %s", name)
	_, err = conn.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

func resourceKubernetesStatefulSetRead(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	id := d.Id()
	namespace, name, err := idParts(id)
	if err != nil {
		return fmt.Errorf("Error parsing resource ID: %#v", err)
	}
	log.Printf("[INFO] Reading stateful set %s", id)
	statefulSet, err := conn.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		switch {
		case errors.IsNotFound(err):
			log.Printf("[DEBUG] Stateful Set %q was not found in Namespace %q - removing from state!", namespace, name)
			d.SetId("")
			return nil
		default:
			log.Printf("[DEBUG] Error reading stateful set: %#v", err)
			return err
		}
	}
	log.Printf("[INFO] Received stateful set: %#v", statefulSet)
	if d.Set("metadata", flattenMetadata(statefulSet.ObjectMeta, d)) != nil {
		return fmt.Errorf("Error setting `metadata`: %+v", err)
	}
	sss, err := flattenStatefulSetSpec(statefulSet.Spec, d)
	if err != nil {
		return fmt.Errorf("Error flattening `spec`: %+v", err)
	}
	err = d.Set("spec", sss)
	if err != nil {
		return fmt.Errorf("Error setting `spec`: %+v", err)
	}
	return nil
}

func resourceKubernetesStatefulSetUpdate(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return fmt.Errorf("Error parsing resource ID: %#v", err)
	}
	ops := patchMetadata("metadata.0.", "/metadata/", d)

	if d.HasChange("spec") {
		log.Println("[TRACE] StatefulSet.Spec has changes")
		specPatch, err := patchStatefulSetSpec(d)
		if err != nil {
			return err
		}
		ops = append(ops, specPatch...)
	}

	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations for StatefulSet: %s", err)
	}
	log.Printf("[INFO] Updating StatefulSet %q: %v", name, string(data))
	out, err := conn.AppsV1().StatefulSets(namespace).Patch(ctx, name, types.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("Failed to update StatefulSet: %s", err)
	}
	log.Printf("[INFO] Submitted updated StatefulSet: %#v", out)

	if d.Get("wait_for_rollout").(bool) {
		log.Printf("[INFO] Waiting for StatefulSet %s to rollout", d.Id())
		return resource.Retry(d.Timeout(schema.TimeoutCreate),
			retryUntilStatefulSetRolloutComplete(ctx, conn, namespace, name))
	}

	return resourceKubernetesStatefulSetRead(d, meta)
}

func resourceKubernetesStatefulSetDelete(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return fmt.Errorf("Error parsing resource ID: %#v", err)
	}
	log.Printf("[INFO] Deleting StatefulSet: %#v", name)
	err = conn.AppsV1().StatefulSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	err = resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		out, err := conn.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			switch {
			case errors.IsNotFound(err):
				return nil
			default:
				return resource.NonRetryableError(err)
			}
		}

		log.Printf("[DEBUG] Current state of StatefulSet: %#v", out.Status.Conditions)
		e := fmt.Errorf("StatefulSet %s still exists %#v", name, out.Status.Conditions)
		return resource.RetryableError(e)
	})
	if err != nil {
		return err
	}

	log.Printf("[INFO] StatefulSet %s deleted", name)

	return nil
}

// retryUntilStatefulSetRolloutComplete checks if a given job finished its execution and is either in 'Complete' or 'Failed' state.
func retryUntilStatefulSetRolloutComplete(ctx context.Context, conn *kubernetes.Clientset, ns, name string) resource.RetryFunc {
	return func() *resource.RetryError {
		res, err := conn.AppsV1().StatefulSets(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return resource.NonRetryableError(err)
		}

		// NOTE: This is what kubectl uses to determine if a rollout is done.
		// We are using this here because the logic for determining if a StatefulSet
		// is done is gnarly and we don't want to duplicate it in the provider.
		gvk := appsv1.SchemeGroupVersion.WithKind("StatefulSet")
		gk := gvk.GroupKind()
		statusViewer, err := polymorphichelpers.StatusViewerFor(gk)
		if err != nil {
			return resource.NonRetryableError(err)
		}

		obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(res)
		if err != nil {
			return resource.NonRetryableError(err)
		}

		// NOTE: For some reason, the Kind and apiVersion get lost when converting to unstructured.
		obj["apiVersion"] = gvk.GroupVersion().String()
		obj["kind"] = gvk.Kind
		u := unstructured.Unstructured{Object: obj}

		// NOTE: the revision parameter of the Status function below is not actually used.
		// for StatefulSet so it is set to 0 here
		_, done, err := statusViewer.Status(&u, 0)
		if err != nil {
			return resource.NonRetryableError(err)
		}

		if done {
			return nil
		}

		return resource.RetryableError(fmt.Errorf("StatefulSet %s/%s is not finished rolling out", ns, name))
	}
}
