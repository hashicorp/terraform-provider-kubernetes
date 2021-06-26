// +build acceptance

package acceptance

import (
	"testing"

	tfstatehelper "github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/state"
)

// This test case tests a ConfigMap but also is a demonstration of some the assert functions
// available in the test helper
func TestKubernetesManifest_ConfigMap(t *testing.T) {
	name := randName()
	namespace := randName()

	tf := tfhelper.RequireNewWorkingDir(t)
	tf.SetReattachInfo(reattachInfo)
	defer func() {
		tf.RequireDestroy(t)
		tf.Close()
		k8shelper.AssertNamespacedResourceDoesNotExist(t, "v1", "configmaps", namespace, name)
	}()

	k8shelper.CreateNamespace(t, namespace)
	defer k8shelper.DeleteNamespace(t, namespace)

	tfvars := TFVARS{
		"namespace": namespace,
		"name":      name,
	}
	tfconfig := loadTerraformConfig(t, "ConfigMap/configmap.tf", tfvars)
	tf.RequireSetConfig(t, tfconfig)
	tf.RequireInit(t)
	tf.RequireApply(t)

	k8shelper.AssertNamespacedResourceExists(t, "v1", "configmaps", namespace, name)

	tfstate := tfstatehelper.NewHelper(tf.RequireState(t))
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.namespace": namespace,
		"kubernetes_manifest.test.object.metadata.name":      name,
		"kubernetes_manifest.test.object.data.foo":           "bar",
	})

	tfconfigModified := loadTerraformConfig(t, "ConfigMap/configmap_modified.tf", tfvars)
	tf.RequireSetConfig(t, tfconfigModified)
	tf.RequireApply(t)

	tfstate = tfstatehelper.NewHelper(tf.RequireState(t))
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.namespace":        namespace,
		"kubernetes_manifest.test.object.metadata.name":             name,
		"kubernetes_manifest.test.object.metadata.annotations.test": "1",
		"kubernetes_manifest.test.object.metadata.labels.test":      "2",
		"kubernetes_manifest.test.object.data.foo":                  "bar",
	})

	tfstate.AssertAttributeEqual(t, "kubernetes_manifest.test.object.data.fizz", "buzz")

	tfstate.AssertAttributeLen(t, "kubernetes_manifest.test.object.metadata.labels", 1)
	tfstate.AssertAttributeLen(t, "kubernetes_manifest.test.object.metadata.annotations", 1)

	tfstate.AssertAttributeNotEmpty(t, "kubernetes_manifest.test.object.metadata.labels.test")

	tfstate.AssertAttributeDoesNotExist(t, "kubernetes_manifest.test.spec")
}
