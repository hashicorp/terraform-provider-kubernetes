// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance
// +build acceptance

package acceptance

import (
	"context"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/kubernetes"
	tfstatehelper "github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/state"
)

func TestKubernetesManifest_HPA(t *testing.T) {
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
		k8shelper.AssertNamespacedResourceDoesNotExist(t,
			"autoscaling/v2", "horizontalpodautoscalers", namespace, name)
	}()

	k8shelper.CreateNamespace(t, namespace)
	defer k8shelper.DeleteResource(t, namespace, kubernetes.NewGroupVersionResource("v1", "namespaces"))

	tfvars := TFVARS{
		"namespace": namespace,
		"name":      name,
	}
	tfconfig := loadTerraformConfig(t, "HPA/hpa.tf", tfvars)
	tf.SetConfig(ctx, tfconfig)
	tf.Init(ctx)
	tf.Apply(ctx)

	k8shelper.AssertNamespacedResourceExists(t,
		"autoscaling/v2", "horizontalpodautoscalers", namespace, name)

	s, err := tf.State(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve terraform state: %q", err)
	}
	tfstate := tfstatehelper.NewHelper(s)
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.namespace": namespace,
		"kubernetes_manifest.test.object.metadata.name":      name,

		"kubernetes_manifest.test.object.spec.scaleTargetRef.apiVersion": "apps/v1",
		"kubernetes_manifest.test.object.spec.scaleTargetRef.kind":       "Deployment",
		"kubernetes_manifest.test.object.spec.scaleTargetRef.name":       "nginx",

		"kubernetes_manifest.test.object.spec.maxReplicas": "10",
		"kubernetes_manifest.test.object.spec.minReplicas": "1",

		"kubernetes_manifest.test.object.spec.metrics.0.type":                               "Resource",
		"kubernetes_manifest.test.object.spec.metrics.0.resource.name":                      "cpu",
		"kubernetes_manifest.test.object.spec.metrics.0.resource.target.type":               "Utilization",
		"kubernetes_manifest.test.object.spec.metrics.0.resource.target.averageUtilization": "50",
	})

	tfconfigModified := loadTerraformConfig(t, "HPA/hpa_modified.tf", tfvars)
	tf.SetConfig(ctx, tfconfigModified)
	tf.Apply(ctx)

	s, err = tf.State(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve terraform state: %q", err)
	}
	tfstate = tfstatehelper.NewHelper(s)
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.namespace": namespace,
		"kubernetes_manifest.test.object.metadata.name":      name,

		"kubernetes_manifest.test.object.spec.scaleTargetRef.apiVersion": "apps/v1",
		"kubernetes_manifest.test.object.spec.scaleTargetRef.kind":       "Deployment",
		"kubernetes_manifest.test.object.spec.scaleTargetRef.name":       "nginx",

		"kubernetes_manifest.test.object.spec.maxReplicas": "20",
		"kubernetes_manifest.test.object.spec.minReplicas": "1",

		"kubernetes_manifest.test.object.spec.metrics.0.type":                               "Resource",
		"kubernetes_manifest.test.object.spec.metrics.0.resource.name":                      "cpu",
		"kubernetes_manifest.test.object.spec.metrics.0.resource.target.type":               "Utilization",
		"kubernetes_manifest.test.object.spec.metrics.0.resource.target.averageUtilization": "65",
	})
}
