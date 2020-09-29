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
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

func resourceKubernetesPersistentVolumeClaim() *schema.Resource {
	fields := persistentVolumeClaimFields()
	// The 'wait_until_bound' control attribute only makes sense in stand-alone PVCs,
	// so adding it on top of the standard PVC fields which are re-usable for other resources.
	fields["wait_until_bound"] = &schema.Schema{
		Type:        schema.TypeBool,
		Description: "Whether to wait for the claim to reach `Bound` state (to find volume in which to claim the space)",
		Optional:    true,
		Default:     true,
	}
	return &schema.Resource{
		Create: resourceKubernetesPersistentVolumeClaimCreate,
		Read:   resourceKubernetesPersistentVolumeClaimRead,
		Exists: resourceKubernetesPersistentVolumeClaimExists,
		Update: resourceKubernetesPersistentVolumeClaimUpdate,
		Delete: resourceKubernetesPersistentVolumeClaimDelete,
		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				d.Set("wait_until_bound", true)
				return []*schema.ResourceData{d}, nil
			},
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: fields,

		// All fields of Spec are immutable after creation, except for resources.requests.storage.
		// Storage can only be increased in place. A new object will be created when the storage is decreased.
		CustomizeDiff: func(diff *schema.ResourceDiff, meta interface{}) error {
			// Skip custom logic for resource creation.
			if diff.Id() == "" {
				return nil
			}
			key := "spec.0.resources.0.requests"
			subKeyStorage := "spec.0.resources.0.requests.storage"
			subKeyLimits := "spec.0.resources.0.limits"
			if diff.HasChange(subKeyLimits) {
				err := diff.ForceNew(subKeyLimits)
				if err != nil {
					return err
				}
				return nil
			}
			if diff.HasChange(key) {
				old, new := diff.GetChange(subKeyStorage)
				oldStorageQuantity, err := k8sresource.ParseQuantity(old.(string))
				if err != nil {
					return err
				}
				newStorageQuantity, err := k8sresource.ParseQuantity(new.(string))
				if err != nil {
					return err
				}
				if newStorageQuantity.Cmp(oldStorageQuantity) == -1 {
					log.Printf("[DEBUG] CustomizeDiff spec.resources.requests.storage: field can not be less than previous value")
					log.Printf("[DEBUG] CustomizeDiff creating new PVC with size: %v", new)
					err := diff.ForceNew(key)
					if err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
}

func resourceKubernetesPersistentVolumeClaimCreate(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	claim, err := expandPersistentVolumeClaim(map[string]interface{}{
		"metadata": d.Get("metadata"),
		"spec":     d.Get("spec"),
	})
	if err != nil {
		return err
	}
	log.Printf("[INFO] Creating new persistent volume claim: %#v", claim)
	out, err := conn.CoreV1().PersistentVolumeClaims(claim.Namespace).Create(ctx, claim, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted new persistent volume claim: %#v", out)

	d.SetId(buildId(out.ObjectMeta))
	name := out.ObjectMeta.Name

	if d.Get("wait_until_bound").(bool) {
		stateConf := &resource.StateChangeConf{
			Target:  []string{"Bound"},
			Pending: []string{"Pending"},
			Timeout: d.Timeout(schema.TimeoutCreate),
			Refresh: func() (interface{}, string, error) {
				out, err := conn.CoreV1().PersistentVolumeClaims(claim.Namespace).Get(ctx, name, metav1.GetOptions{})
				if err != nil {
					log.Printf("[ERROR] Received error: %#v", err)
					return out, "", err
				}

				statusPhase := fmt.Sprintf("%v", out.Status.Phase)
				log.Printf("[DEBUG] Persistent volume claim %s status received: %#v", out.Name, statusPhase)
				return out, statusPhase, nil
			},
		}
		_, err = stateConf.WaitForState()
		if err != nil {
			var lastWarnings []api.Event
			var wErr error

			lastWarnings, wErr = getLastWarningsForObject(ctx, conn, out.ObjectMeta, "PersistentVolumeClaim", 3)
			if wErr != nil {
				return wErr
			}

			if len(lastWarnings) == 0 {
				lastWarnings, wErr = getLastWarningsForObject(ctx, conn, metav1.ObjectMeta{
					Name: out.Spec.VolumeName,
				}, "PersistentVolume", 3)
				if wErr != nil {
					return wErr
				}
			}

			return fmt.Errorf("%s%s", err, stringifyEvents(lastWarnings))
		}
	}
	log.Printf("[INFO] Persistent volume claim %s created", out.Name)

	return resourceKubernetesPersistentVolumeClaimRead(d, meta)
}

func resourceKubernetesPersistentVolumeClaimRead(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading persistent volume claim %s", name)
	claim, err := conn.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received persistent volume claim: %#v", claim)
	err = d.Set("metadata", flattenMetadata(claim.ObjectMeta, d))
	if err != nil {
		return err
	}
	err = d.Set("spec", flattenPersistentVolumeClaimSpec(claim.Spec))
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesPersistentVolumeClaimUpdate(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	// spec.resources.requests is the only editable field in Spec.
	if d.HasChange("spec.0.resources.0.requests") {
		r := d.Get("spec.0.resources.0.requests").(map[string]interface{})
		requests, err := expandMapToResourceList(r)
		if err != nil {
			return err
		}
		ops = append(ops, &ReplaceOperation{
			Path:  "/spec/resources/requests",
			Value: requests,
		})
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}

	log.Printf("[INFO] Updating persistent volume claim: %s", ops)
	out, err := conn.CoreV1().PersistentVolumeClaims(namespace).Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted updated persistent volume claim: %#v", out)

	return resourceKubernetesPersistentVolumeClaimRead(d, meta)
}

func resourceKubernetesPersistentVolumeClaimDelete(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Deleting persistent volume claim: %#v", name)
	err = conn.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	err = resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		out, err := conn.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return resource.NonRetryableError(err)
		}

		log.Printf("[DEBUG] Current state of persistent volume claim finalizers: %#v", out.Finalizers)
		e := fmt.Errorf("Persistent volume claim %s still exists with finalizers: %v", name, out.Finalizers)
		return resource.RetryableError(e)
	})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Persistent volume claim %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesPersistentVolumeClaimExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}
	ctx := context.TODO()

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking persistent volume claim %s", name)
	_, err = conn.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
