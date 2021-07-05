// +build acceptance

package acceptance

import (
	"testing"

	tfstatehelper "github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/state"
)

func TestKubernetesManifest_Job(t *testing.T) {
	name := randName()
	namespace := randName()

	tf := tfhelper.RequireNewWorkingDir(t)
	tf.SetReattachInfo(reattachInfo)
	defer func() {
		tf.RequireDestroy(t)
		tf.Close()
		k8shelper.AssertNamespacedResourceDoesNotExist(t, "batch/v1", "jobs", namespace, name)
	}()

	k8shelper.CreateNamespace(t, namespace)
	defer k8shelper.DeleteNamespace(t, namespace)

	tfvars := TFVARS{
		"namespace": namespace,
		"name":      name,
	}
	tfconfig := loadTerraformConfig(t, "Job/job.tf", tfvars)
	tf.RequireSetConfig(t, tfconfig)
	tf.RequireInit(t)
	tf.RequireApply(t)

	k8shelper.AssertNamespacedResourceExists(t, "batch/v1", "jobs", namespace, name)

	tfstate := tfstatehelper.NewHelper(tf.RequireState(t))
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.namespace":                      namespace,
		"kubernetes_manifest.test.object.metadata.name":                           name,
		"kubernetes_manifest.test.object.spec.template.spec.containers.0.name":    "busybox",
		"kubernetes_manifest.test.object.spec.template.spec.containers.0.image":   "busybox",
		"kubernetes_manifest.test.object.spec.template.spec.containers.0.command": []interface{}{"sleep", "30"},
		"kubernetes_manifest.test.object.spec.template.spec.restartPolicy":        "Never",
	})
}
