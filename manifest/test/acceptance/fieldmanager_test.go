//go:build acceptance
// +build acceptance

package acceptance

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/kubernetes"
	tfstatehelper "github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/state"
)

func TestKubernetesManifest_fieldManager(t *testing.T) {
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
	defer k8shelper.DeleteResource(t, namespace, kubernetes.NewGroupVersionResource("v1", "namespaces"))

	// 1. Create the resource
	tfvars := TFVARS{
		"namespace":     namespace,
		"name":          name,
		"field_manager": "tftest",
		"force":         false,
		"data":          "bar",
	}
	tfconfig := loadTerraformConfig(t, "FieldManager/field_manager.tf", tfvars)
	tf.RequireSetConfig(t, tfconfig)
	tf.RequireInit(t)
	tf.RequireApply(t)

	k8shelper.AssertNamespacedResourceExists(t, "v1", "configmaps", namespace, name)

	tfstate := tfstatehelper.NewHelper(tf.RequireState(t))
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.namespace":       namespace,
		"kubernetes_manifest.test.object.metadata.name":            name,
		"kubernetes_manifest.test.object.data.foo":                 "bar",
		"kubernetes_manifest.test.field_manager.0.name":            "tftest",
		"kubernetes_manifest.test.field_manager.0.force_conflicts": false,
	})

	// 2. Try to change the resource with a new field manager name, should give a conflict
	tfvars = TFVARS{
		"namespace":     namespace,
		"name":          name,
		"field_manager": "tftest-newmanager",
		"force":         false,
		"data":          "foobar",
	}
	tfconfig = loadTerraformConfig(t, "FieldManager/field_manager.tf", tfvars)
	tf.RequireSetConfig(t, tfconfig)
	tf.RequireInit(t)
	err := tf.Apply() // this should fail
	if err == nil || !strings.Contains(err.Error(), "There was a field manager conflict when trying to apply the manifest") {
		t.Log(err.Error())
		t.Fatal("Expected terraform apply to cause a field manager conflict")
	}

	// 3. Try again with force_conflicts set to true, should succeed
	tfvars = TFVARS{
		"namespace":     namespace,
		"name":          name,
		"field_manager": "tftest-newmanager",
		"force":         true,
		"data":          "foobar",
	}
	tfconfig = loadTerraformConfig(t, "FieldManager/field_manager.tf", tfvars)
	tf.RequireSetConfig(t, tfconfig)
	tf.RequireInit(t)
	tf.RequireApply(t)

	k8shelper.AssertNamespacedResourceExists(t, "v1", "configmaps", namespace, name)

	tfstate = tfstatehelper.NewHelper(tf.RequireState(t))
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.namespace":       namespace,
		"kubernetes_manifest.test.object.metadata.name":            name,
		"kubernetes_manifest.test.object.data.foo":                 "foobar",
		"kubernetes_manifest.test.field_manager.0.name":            "tftest-newmanager",
		"kubernetes_manifest.test.field_manager.0.force_conflicts": true,
	})
}
