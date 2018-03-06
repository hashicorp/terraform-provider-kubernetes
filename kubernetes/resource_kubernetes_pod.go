package kubernetes

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
	kubernetes "k8s.io/client-go/kubernetes"
	api "k8s.io/client-go/pkg/api/v1"
)

func resourceKubernetesPod() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesPodCreate,
		Read:   resourceKubernetesPodRead,
		Update: resourceKubernetesPodUpdate,
		Delete: resourceKubernetesPodDelete,
		Exists: resourceKubernetesPodExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("pod", true),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec of the pod owned by the cluster",
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: podSpecFields(false),
				},
			},
		},
	}
}
func resourceKubernetesPodCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandPodSpec(d.Get("spec").([]interface{}))
	if err != nil {
		return err
	}

	pod := api.Pod{
		ObjectMeta: metadata,
		Spec:       spec,
	}

	log.Printf("[INFO] Creating new pod: %#v", pod)
	out, err := conn.CoreV1().Pods(metadata.Namespace).Create(&pod)

	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted new pod: %#v", out)

	d.SetId(buildId(out.ObjectMeta))

	stateConf := &resource.StateChangeConf{
		Target:  []string{"Running"},
		Pending: []string{"Pending"},
		Timeout: 5 * time.Minute,
		Refresh: func() (interface{}, string, error) {
			out, err := conn.CoreV1().Pods(metadata.Namespace).Get(metadata.Name, metav1.GetOptions{})
			if err != nil {
				log.Printf("[ERROR] Received error: %#v", err)
				return out, "Error", err
			}

			statusPhase := fmt.Sprintf("%v", out.Status.Phase)
			log.Printf("[DEBUG] Pods %s status received: %#v", out.Name, statusPhase)
			return out, statusPhase, nil
		},
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		lastWarnings, wErr := getLastWarningsForObject(conn, out.ObjectMeta, "Pod", 3)
		if wErr != nil {
			return wErr
		}
		return fmt.Errorf("%s%s", err, stringifyEvents(lastWarnings))
	}
	log.Printf("[INFO] Pod %s created", out.Name)

	return resourceKubernetesPodRead(d, meta)
}

func resourceKubernetesPodUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("spec") {
		specOps, err := patchPodSpec("/spec", "spec.0.", d)
		if err != nil {
			return err
		}
		ops = append(ops, specOps...)
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}

	log.Printf("[INFO] Updating pod %s: %s", d.Id(), ops)

	out, err := conn.CoreV1().Pods(namespace).Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted updated pod: %#v", out)

	d.SetId(buildId(out.ObjectMeta))
	return resourceKubernetesPodRead(d, meta)
}

func resourceKubernetesPodRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading pod %s", name)
	pod, err := conn.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received pod: %#v", pod)

	err = d.Set("metadata", flattenMetadata(pod.ObjectMeta, d))
	if err != nil {
		return err
	}

	// if the Pod has automount_service_account_token = true, then the backend
	// will automatically add an extra Volume + VolumeMount to the pod for the
	// service account token
	// We remove it here before flattening the PodSpec to avoid a TF diff
	volumeConfig, _ := expandVolumes(d.Get("spec.0.volume").([]interface{}))
	if *pod.Spec.AutomountServiceAccountToken && len(volumeConfig) != len(pod.Spec.Volumes) {
		tokenVolumeName := ""
		vIndex := -1
		// there is a mismatch, loop over real volumes to find the token
		for i, v := range pod.Spec.Volumes {
			if strings.Contains(v.Name, fmt.Sprintf("%s-token-", pod.Spec.ServiceAccountName)) {
				log.Printf("[INFO] found automounted token Volume [%s]", v.Name)
				tokenVolumeName = v.Name
				vIndex = i
			}
		}
		if vIndex > -1 {
			// remove Volume
			log.Print("[INFO] removing automounted token Volume from spec")
			pod.Spec.Volumes = append(pod.Spec.Volumes[:vIndex], pod.Spec.Volumes[vIndex+1:]...)
		}

		for ci, c := range pod.Spec.Containers {
			pod.Spec.Containers[ci] = removeTokenVolumeMount(c, tokenVolumeName)
		}
		for ci, c := range pod.Spec.InitContainers {
			pod.Spec.Containers[ci] = removeTokenVolumeMount(c, tokenVolumeName)
		}
	}

	podSpec, err := flattenPodSpec(pod.Spec)
	if err != nil {
		return err
	}

	err = d.Set("spec", podSpec)
	if err != nil {
		return err
	}
	return nil

}

func resourceKubernetesPodDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Deleting pod: %#v", name)
	err = conn.CoreV1().Pods(namespace).Delete(name, nil)
	if err != nil {
		return err
	}

	err = resource.Retry(1*time.Minute, func() *resource.RetryError {
		out, err := conn.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
				return nil
			}
			return resource.NonRetryableError(err)
		}

		log.Printf("[DEBUG] Current state of pod: %#v", out.Status.Phase)
		e := fmt.Errorf("Pod %s still exists (%s)", name, out.Status.Phase)
		return resource.RetryableError(e)
	})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Pod %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesPodExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*kubernetes.Clientset)

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking pod %s", name)
	_, err = conn.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

func removeTokenVolumeMount(c api.Container, tokenName string) api.Container {
	vmIndex := -1
	for vi, v := range c.VolumeMounts {
		if v.Name == tokenName {
			vmIndex = vi
		}
	}
	if vmIndex > -1 {
		// remove VolumeMount from container
		log.Print("[INFO] removing automounted token VolumeMount from spec")
		c.VolumeMounts = append(c.VolumeMounts[:vmIndex], c.VolumeMounts[vmIndex+1:]...)
	}
	return c
}
