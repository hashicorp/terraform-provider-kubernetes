package kubernetes

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dataSourceKubernetesPod() *schema.Resource {
	podSpecFields := podSpecFields(false, false, false)
	// Setting this default to false prevents a perpetual diff caused by volume_mounts
	// being mutated on the server side as Kubernetes automatically adds a mount
	// for the service account token
	return &schema.Resource{
		Read: dataSourceKubernetesPodRead,

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("pod", true),
			"spec": {
				Type:        schema.TypeList,
				Description: "Specification of the desired behavior of the pod.",
				Computed:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: podSpecFields,
				},
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceKubernetesPodRead(d *schema.ResourceData, meta interface{}) error {

	metadata := expandMetadata(d.Get("metadata").([]interface{}))

	om := meta_v1.ObjectMeta{
		Namespace: metadata.Namespace,
		Name:      metadata.Name,
	}
	d.SetId(buildId(om))

	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading pod %s", metadata.Name)
	pod, err := conn.CoreV1().Pods(metadata.Namespace).Get(metadata.Name, meta_v1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received pod: %#v", pod)

	err = d.Set("metadata", flattenMetadata(pod.ObjectMeta, d))
	if err != nil {
		return err
	}

	podSpec, err := flattenPodSpec(pod.Spec, true)
	if err != nil {
		return err
	}

	err = d.Set("spec", podSpec)
	if err != nil {
		return err
	}
	statusPhase := fmt.Sprintf("%v", pod.Status.Phase)
	d.Set("status", statusPhase)

	return nil

}
