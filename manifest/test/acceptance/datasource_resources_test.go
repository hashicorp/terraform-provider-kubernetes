// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance
// +build acceptance

package acceptance

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/kubernetes"
	tfstatehelper "github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/state"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func TestDataSourceKubernetesResources_DiffTuples(t *testing.T) {
	ctx := context.Background()

	reattachInfo, err := provider.ServeTest(ctx, hclog.Default(), t)
	if err != nil {
		t.Errorf("Failed to create provider instance: %q", err)
	}

	name := randName()
	namespace := randName()

	tf := tfhelper.RequireNewWorkingDir(ctx, t)
	tf.SetReattachInfo(ctx, reattachInfo)
	defer func() {
		tf.Destroy(ctx)
		tf.Close()
		k8shelper.AssertResourceDoesNotExist(t, "v1", "namespaces", name)
	}()

	// Create a Namespace to provision the rest of the resources in it.
	k8shelper.CreateNamespace(t, namespace)
	defer k8shelper.DeleteResource(t, namespace, kubernetes.NewGroupVersionResource("v1", "namespaces"))

	// Create Pods to use a data source.
	// The only difference between the two pods is the number of VolumeMounts(tuples) that will be created.
	// This is necessary to ensure that DeepUnknown doesn't mutate object type when iterates over items.
	// kubernetes_manifest failed to create pods with volumes, thus create them manually.
	k8shelper.AssertNamespacedResourceDoesNotExist(t, "v1", "pods", namespace, name)

	volumes := []corev1.Volume{
		{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
	podSpecs := []corev1.PodSpec{
		{
			TerminationGracePeriodSeconds: ptr.To(int64(1)),
			Containers: []corev1.Container{
				{
					Name:    "this",
					Image:   "busybox",
					Command: []string{"sleep", "infinity"},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "config",
							MountPath: "/config-a",
						},
						{
							Name:      "config",
							MountPath: "/config-b",
						},
					},
				},
			},
			Volumes: volumes,
		},
		{
			TerminationGracePeriodSeconds: ptr.To(int64(1)),
			Containers: []corev1.Container{
				{
					Name:    "this",
					Image:   "busybox",
					Command: []string{"sleep", "infinity"},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "config",
							MountPath: "/config-a",
						},
						{
							Name:      "config",
							MountPath: "/config-b",
						},
						{
							Name:      "config",
							MountPath: "/config-c",
						},
					},
				},
			},
			Volumes: volumes,
		},
	}

	for i, ps := range podSpecs {
		k8shelper.CreatePod(t, fmt.Sprintf("%s-%d", name, i), namespace, ps)
	}

	// Get pods
	tfvars := TFVARS{
		"namespace": namespace,
	}
	tfconfig := loadTerraformConfig(t, "DataSourceResources/pods_data_source.tf", tfvars)
	tf.SetConfig(ctx, tfconfig)
	tf.Init(ctx)
	err = tf.Apply(ctx)
	if err != nil {
		t.Fatal(err.Error())
	}

	st, err := tf.State(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve terraform state: %q", err)
	}
	tfstate := tfstatehelper.NewHelper(st)

	// check the data source
	tfstate.AssertAttributeLen(t, "data.kubernetes_resources.pods.objects", len(podSpecs))
}
