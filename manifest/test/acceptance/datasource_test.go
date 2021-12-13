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

func TestDataSourceKubernetesResource_ConfigMap(t *testing.T) {
	name := randName()
	name2 := randName()
	namespace := randName()

	k8shelper.CreateNamespace(t, namespace)
	defer k8shelper.DeleteResource(t, namespace, kubernetes.NewGroupVersionResource("v1", "namespaces"))

	// STEP 1: Create a ConfigMap to use as a data source
	tf := tfhelper.RequireNewWorkingDir(t)
	tf.SetReattachInfo(reattachInfo)
	defer func() {
		tf.RequireDestroy(t)
		tf.Close()
		k8shelper.AssertNamespacedResourceDoesNotExist(t, "v1", "configmaps", namespace, name)
	}()

	tfvars := TFVARS{
		"name":      name,
		"name2":     name2,
		"namespace": namespace,
	}
	tfconfig := loadTerraformConfig(t, "datasource/step1.tf", tfvars)
	tf.RequireSetConfig(t, tfconfig)
	tf.RequireInit(t)
	tf.RequireApply(t)

	k8shelper.AssertNamespacedResourceExists(t, "v1", "configmaps", namespace, name)

	// STEP 2: Create another ConfigMap using the ConfigMap from step 1 as a data source
	reattachInfo2, err := provider.ServeTest(context.TODO(), hclog.Default())
	if err != nil {
		t.Errorf("Failed to create additional provider instance: %q", err)
	}
	step2 := tfhelper.RequireNewWorkingDir(t)
	step2.SetReattachInfo(reattachInfo2)
	defer func() {
		step2.RequireDestroy(t)
		step2.Close()
		k8shelper.AssertNamespacedResourceDoesNotExist(t, "v1", "configmaps", namespace, name2)
	}()

	tfconfig = loadTerraformConfig(t, "datasource/step2.tf", tfvars)
	step2.RequireSetConfig(t, string(tfconfig))
	step2.RequireInit(t)
	step2.RequireApply(t)

	tfstate := tfstatehelper.NewHelper(step2.RequireState(t))

	// check the data source
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"data.kubernetes_resource.test_config.object.data.TEST": "hello world",
	})
	// check the resource was created with the correct value
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test_config2.object.data.TEST": "hello world",
	})
}
