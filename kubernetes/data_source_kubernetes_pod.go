package kubernetes

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dataSourceKubernetesPod() *schema.Resource {
	return &schema.Resource{
		ReadContext:   dataSourceKubernetesPodRead,
		Schema:        dataSourceKubernetesPodSchemaV1(),
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Version: 0,
				Type:    dataSourceKubernetesPodV0().CoreConfigSchema().ImpliedType(),
				Upgrade: dataSourceKubernetesPodUpgradeV0,
			},
		},
	}
}

func dataSourceKubernetesPodSchemaV1() map[string]*schema.Schema {
	podSpecFields := podSpecFields(false, false, false)

	return map[string]*schema.Schema{
		"metadata": namespacedMetadataSchema("pod", true),
		"spec": {
			Type:        schema.TypeList,
			Description: "Specification of the desired behavior of the pod.",
			Computed:    true,
			Elem: &schema.Resource{
				Schema: podSpecFields,
			},
		},
		"status": {
			Type:     schema.TypeString,
			Computed: true,
		},
	}
}

func dataSourceKubernetesPodRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))

	om := meta_v1.ObjectMeta{
		Namespace: metadata.Namespace,
		Name:      metadata.Name,
	}
	d.SetId(buildId(om))

	log.Printf("[INFO] Reading pod %s", metadata.Name)
	pod, err := conn.CoreV1().Pods(metadata.Namespace).Get(ctx, metadata.Name, meta_v1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Received pod: %#v", pod)

	err = d.Set("metadata", flattenMetadata(pod.ObjectMeta, d))
	if err != nil {
		return diag.FromErr(err)
	}

	podSpec, err := flattenPodSpec(pod.Spec)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("spec", podSpec)
	if err != nil {
		return diag.FromErr(err)
	}
	statusPhase := fmt.Sprintf("%v", pod.Status.Phase)
	d.Set("status", statusPhase)

	return nil

}
