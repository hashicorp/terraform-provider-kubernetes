// +build acceptance

package acceptance

import (
	"testing"

	tfstatehelper "github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/state"
)

func TestKubernetesManifest_HPA(t *testing.T) {
	name := randName()
	namespace := randName()

	tf := tfhelper.RequireNewWorkingDir(t)
	tf.SetReattachInfo(reattachInfo)
	defer func() {
		tf.RequireDestroy(t)
		tf.Close()
		k8shelper.AssertNamespacedResourceDoesNotExist(t,
			"autoscaling/v2beta2", "horizontalpodautoscalers", namespace, name)
	}()

	k8shelper.CreateNamespace(t, namespace)
	defer k8shelper.DeleteNamespace(t, namespace)

	tfvars := TFVARS{
		"namespace": namespace,
		"name":      name,
	}
	tfconfig := loadTerraformConfig(t, "HPA/hpa.tf", tfvars)
	tf.RequireSetConfig(t, tfconfig)
	tf.RequireInit(t)
	tf.RequireApply(t)

	k8shelper.AssertNamespacedResourceExists(t,
		"autoscaling/v2beta2", "horizontalpodautoscalers", namespace, name)

	tfstate := tfstatehelper.NewHelper(tf.RequireState(t))
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
	tf.RequireSetConfig(t, tfconfigModified)
	tf.RequireApply(t)

	tfstate = tfstatehelper.NewHelper(tf.RequireState(t))
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
