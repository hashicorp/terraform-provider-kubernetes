package kubernetes

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

func resourceKubernetesNamespace() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesNamespaceCreate,
		Read:   resourceKubernetesNamespaceRead,
		Update: resourceKubernetesNamespaceUpdate,
		Delete: resourceKubernetesNamespaceDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("namespace", true),
		},
		Timeouts: &schema.ResourceTimeout{
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},
	}
}

func resourceKubernetesNamespaceCreate(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	namespace := api.Namespace{
		ObjectMeta: metadata,
	}
	log.Printf("[INFO] Creating new namespace: %#v", namespace)
	out, err := conn.CoreV1().Namespaces().Create(ctx, &namespace, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted new namespace: %#v", out)
	d.SetId(out.Name)

	return resourceKubernetesNamespaceRead(d, meta)
}

func resourceKubernetesNamespaceRead(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	name := d.Id()
	log.Printf("[INFO] Reading namespace %s", name)
	namespace, err := conn.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received namespace: %#v", namespace)
	err = d.Set("metadata", flattenMetadata(namespace.ObjectMeta, d))
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesNamespaceUpdate(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}

	log.Printf("[INFO] Updating namespace: %s", ops)
	out, err := conn.CoreV1().Namespaces().Patch(ctx, d.Id(), pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted updated namespace: %#v", out)
	d.SetId(out.Name)

	return resourceKubernetesNamespaceRead(d, meta)
}

func resourceKubernetesNamespaceDelete(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	name := d.Id()
	log.Printf("[INFO] Deleting namespace: %#v", name)
	err = conn.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Target:  []string{},
		Pending: []string{"Terminating"},
		Timeout: d.Timeout(schema.TimeoutDelete),
		Refresh: func() (interface{}, string, error) {
			out, err := conn.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
					return nil, "", nil
				}
				log.Printf("[ERROR] Received error: %#v", err)
				return out, "Error", err
			}

			statusPhase := fmt.Sprintf("%v", out.Status.Phase)
			log.Printf("[DEBUG] Namespace %s status received: %#v", out.Name, statusPhase)
			return out, statusPhase, nil
		},
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}
	log.Printf("[INFO] Namespace %s deleted", name)

	d.SetId("")
	return nil
}
