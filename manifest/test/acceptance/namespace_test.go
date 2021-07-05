// +build acceptance

package acceptance

import (
	"testing"

	tfstatehelper "github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/state"
)

func TestKubernetesManifest_Namespace(t *testing.T) {
	name := randName()

	tf := tfhelper.RequireNewWorkingDir(t)
	tf.SetReattachInfo(reattachInfo)
	defer func() {
		tf.RequireDestroy(t)
		tf.Close()
		k8shelper.AssertResourceDoesNotExist(t, "v1", "namespaces", name)
	}()

	tfvars := TFVARS{
		"name": name,
	}
	tfconfig := loadTerraformConfig(t, "Namespace/namespace.tf", tfvars)
	tf.RequireSetConfig(t, tfconfig)
	tf.RequireInit(t)
	tf.RequireApply(t)

	k8shelper.AssertResourceExists(t, "v1", "namespaces", name)

	tfstate := tfstatehelper.NewHelper(tf.RequireState(t))
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.name": name,
	})

	tfconfigModified := loadTerraformConfig(t, "Namespace/namespace_modified.tf", tfvars)
	tf.RequireSetConfig(t, tfconfigModified)
	tf.RequireApply(t)

	tfstate = tfstatehelper.NewHelper(tf.RequireState(t))
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.name":        name,
		"kubernetes_manifest.test.object.metadata.labels.test": "test",
	})
}
