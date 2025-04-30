// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dataSourceKubernetesPodV1() *schema.Resource {
	podSpecFields := podSpecFields(false, false)
	// Setting this default to false prevents a perpetual diff caused by volume_mounts
	// being mutated on the server side as Kubernetes automatically adds a mount
	// for the service account token
	return &schema.Resource{
		Description: "A pod is a group of one or more containers, the shared storage for those containers, and options about how to run the containers. Pods are always co-located and co-scheduled, and run in a shared context. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod/.",
		ReadContext: dataSourceKubernetesPodV1Read,

		Schema: map[string]*schema.Schema{
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
		},
	}
}

func dataSourceKubernetesPodV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var pod v1.Pod
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))

	om := metav1.ObjectMeta{
		Namespace: metadata.Namespace,
		Name:      metadata.Name,
	}
	d.SetId(buildId(om))

	pods := conn.CoreV1().Pods(metadata.Namespace)

	if metadata.Name != "" {
		log.Printf("[INFO] Getting pod %s", metadata.Name)
		podResult, getErr := pods.Get(ctx, metadata.Name, metav1.GetOptions{})
		if getErr != nil {
			err = getErr
		} else {
			pod = *podResult
		}
	} else {
		log.Printf("[INFO] Listing pods")
		listOptions := metav1.ListOptions{
			LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{MatchLabels: metadata.Labels}),
		}
		podList, listErr := pods.List(ctx, listOptions)
		if listErr != nil {
			err = listErr
		} else {
			if len(podList.Items) == 0 {
				return diag.Errorf("No pods found")
			}
			pod = podList.Items[0]
		}
	}

	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Received pod: %#v", pod)

	err = d.Set("metadata", flattenMetadataFields(pod.ObjectMeta))
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
	err = d.Set("status", pod.Status.Phase)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil

}
