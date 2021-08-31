//go:build acceptance
// +build acceptance

package acceptance

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/kubernetes"
)

func TestKubernetesManifest_alreadyExists(t *testing.T) {
	name := randName()
	namespace := randName()

	tf := tfhelper.RequireNewWorkingDir(t)
	tf.SetReattachInfo(reattachInfo)
	defer func() {
		tf.RequireDestroy(t)
		tf.Close()
		k8shelper.AssertNamespacedResourceDoesNotExist(t,
			"v1", "configmaps", namespace, name)
	}()

	k8shelper.CreateNamespace(t, namespace)
	defer k8shelper.DeleteResource(t, namespace, kubernetes.NewGroupVersionResource("v1", "namespaces"))

	tfvars := TFVARS{
		"namespace": namespace,
		"name":      name,
	}
	tfconfig := loadTerraformConfig(t, "alreadyExists/configmap.tf", tfvars)
	tf.RequireSetConfig(t, tfconfig)
	tf.RequireInit(t)
	tf.RequireApply(t)

	k8shelper.AssertNamespacedResourceExists(t,
		"v1", "configmaps", namespace, name)

	// Make a new working dir and apply again
	tf2 := tfhelper.RequireNewWorkingDir(t)
	tf2.SetReattachInfo(reattachInfo)
	tfconfigModified := loadTerraformConfig(t, "alreadyExists/configmap.tf", tfvars)
	tf2.RequireSetConfig(t, tfconfigModified)
	err := tf2.Apply()

	if err == nil {
		t.Fatal("Creating a resource that already exists should cause an error")
	}

	errMsg := "Error: Cannot create resource that already exists"
	if err != nil && !strings.Contains(err.Error(), errMsg) {
		t.Errorf("Expected error to contain %q. Actual error:", errMsg)
		t.Log(err)
	}
}
