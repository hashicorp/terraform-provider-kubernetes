//go:build acceptance
// +build acceptance

package acceptance

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/kubernetes"
	tfstatehelper "github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/state"
)

// This test case tests a Service but also is a demonstration of some the assert functions
// available in the test helper
func TestKubernetesManifest_Service_NodePort(t *testing.T) {
	name := randName()
	namespace := randName()

	tf := tfhelper.RequireNewWorkingDir(t)
	tf.SetReattachInfo(reattachInfo)
	defer func() {
		tf.RequireDestroy(t)
		tf.Close()
		k8shelper.AssertNamespacedResourceDoesNotExist(t, "v1", "services", namespace, name)
	}()

	k8shelper.CreateNamespace(t, namespace)
	defer k8shelper.DeleteResource(t, namespace, kubernetes.NewGroupVersionResource("v1", "namespaces"))

	tfvars := TFVARS{
		"namespace": namespace,
		"name":      name,
	}
	tfconfig := loadTerraformConfig(t, "Service_NodePort/service.tf", tfvars)
	tf.RequireSetConfig(t, tfconfig)
	tf.RequireInit(t)
	tf.RequireApply(t)

	k8shelper.AssertNamespacedResourceExists(t, "v1", "services", namespace, name)

	tfstate := tfstatehelper.NewHelper(tf.RequireState(t))
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.namespace":      namespace,
		"kubernetes_manifest.test.object.metadata.name":           name,
		"kubernetes_manifest.test.object.spec.ports.0.name":       "http",
		"kubernetes_manifest.test.object.spec.ports.0.port":       json.Number("80"),
		"kubernetes_manifest.test.object.spec.ports.0.targetPort": json.Number("8080"),
		"kubernetes_manifest.test.object.spec.ports.0.protocol":   "TCP",
		"kubernetes_manifest.test.object.spec.selector.app":       "test",
		"kubernetes_manifest.test.object.spec.type":               "NodePort",
	})

	tfstate.AssertAttributeNotEmpty(t, "kubernetes_manifest.test.object.spec.ports.0.nodePort")

	tfconfigModified := loadTerraformConfig(t, "Service_NodePort/service_modified.tf", tfvars)
	tf.RequireSetConfig(t, tfconfigModified)
	tf.RequireApply(t)

	tfstate = tfstatehelper.NewHelper(tf.RequireState(t))
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.namespace":        namespace,
		"kubernetes_manifest.test.object.metadata.name":             name,
		"kubernetes_manifest.test.object.metadata.annotations.test": "1",
		"kubernetes_manifest.test.object.metadata.labels.test":      "2",
		"kubernetes_manifest.test.object.spec.ports.0.name":         "https",
		"kubernetes_manifest.test.object.spec.ports.0.port":         json.Number("443"),
		"kubernetes_manifest.test.object.spec.ports.0.targetPort":   json.Number("8443"),
		"kubernetes_manifest.test.object.spec.ports.0.nodePort":     json.Number("32767"),
		"kubernetes_manifest.test.object.spec.ports.0.protocol":     "TCP",
		"kubernetes_manifest.test.object.spec.selector.app":         "test",
		"kubernetes_manifest.test.object.spec.type":                 "NodePort",
	})

	tfstate.AssertAttributeLen(t, "kubernetes_manifest.test.object.metadata.labels", 1)
	tfstate.AssertAttributeLen(t, "kubernetes_manifest.test.object.metadata.annotations", 1)

	tfstate.AssertAttributeNotEmpty(t, "kubernetes_manifest.test.object.metadata.labels.test")

	tfstate.AssertAttributeDoesNotExist(t, "kubernetes_manifest.test.spec")
}
