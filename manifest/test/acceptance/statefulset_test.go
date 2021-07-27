// +build acceptance

package acceptance

import (
	"testing"

	tfstatehelper "github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/state"
)

func TestKubernetesManifest_StatefulSet(t *testing.T) {
	name := randName()
	namespace := randName()

	tf := tfhelper.RequireNewWorkingDir(t)
	tf.SetReattachInfo(reattachInfo)
	defer func() {
		tf.RequireDestroy(t)
		tf.Close()
		k8shelper.AssertNamespacedResourceDoesNotExist(t, "apps/v1", "statefulsets", namespace, name)
	}()

	k8shelper.CreateNamespace(t, namespace)
	defer k8shelper.DeleteNamespace(t, namespace)

	tfvars := TFVARS{
		"namespace": namespace,
		"name":      name,
	}
	tfconfig := loadTerraformConfig(t, "StatefulSet/statefulset.tf", tfvars)
	tf.RequireSetConfig(t, tfconfig)
	tf.RequireInit(t)
	tf.RequireApply(t)

	k8shelper.AssertNamespacedResourceExists(t, "apps/v1", "statefulsets", namespace, name)

	tfstate := tfstatehelper.NewHelper(tf.RequireState(t))
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.namespace": namespace,
		"kubernetes_manifest.test.object.metadata.name":      name,

		"kubernetes_manifest.test.object.spec.replicas":                 "2",
		"kubernetes_manifest.test.object.spec.selector.matchLabels.app": "nginx",

		"kubernetes_manifest.test.object.spec.template.spec.containers.0.name":                     "nginx",
		"kubernetes_manifest.test.object.spec.template.spec.containers.0.image":                    "nginx:1",
		"kubernetes_manifest.test.object.spec.template.spec.containers.0.ports.0.containerPort":    "80",
		"kubernetes_manifest.test.object.spec.template.spec.containers.0.ports.0.name":             "web",
		"kubernetes_manifest.test.object.spec.template.spec.containers.0.volumeMounts.0.name":      "www",
		"kubernetes_manifest.test.object.spec.template.spec.containers.0.volumeMounts.0.mountPath": "/usr/share/nginx/html",

		"kubernetes_manifest.test.object.spec.volumeClaimTemplates.0.metadata.name":                   "www",
		"kubernetes_manifest.test.object.spec.volumeClaimTemplates.0.spec.accessModes.0":              "ReadWriteOnce",
		"kubernetes_manifest.test.object.spec.volumeClaimTemplates.0.spec.resources.requests.storage": "1Gi",
	})

	tfstate.AssertAttributeExists(t, "kubernetes_manifest.test.object.spec.serviceName")
}
