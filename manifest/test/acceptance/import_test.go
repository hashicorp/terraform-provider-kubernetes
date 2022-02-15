//go:build acceptance
// +build acceptance

package acceptance

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/kubernetes"
	tfstatehelper "github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/state"
)

func TestKubernetesManifest_Import(t *testing.T) {
	ctx := context.Background()

	reattachInfo, err := provider.ServeTest(ctx, hclog.Default(), t)
	if err != nil {
		t.Errorf("Failed to create provider instance: %q", err)
	}

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

	k8shelper.AssertNamespacedResourceDoesNotExist(t, "v1", "configmaps", namespace, name)

	k8shelper.CreateConfigMap(t, name, namespace,
		map[string]interface{}{
			"foo": "bar",
		})

	tfvars := TFVARS{
		"namespace": namespace,
		"name":      name,
	}
	tfconfig := loadTerraformConfig(t, "Import/import.tf", tfvars)
	tf.RequireSetConfig(t, tfconfig)
	tf.RequireInit(t)

	importId := fmt.Sprintf("apiVersion=%s,kind=%s,namespace=%s,name=%s", "v1", "ConfigMap", namespace, name)

	tf.RequireImport(t, "kubernetes_manifest.test", importId)
	k8shelper.AssertNamespacedResourceExists(t, "v1", "configmaps", namespace, name)

	tfstate := tfstatehelper.NewHelper(tf.RequireState(t))
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.namespace": namespace,
		"kubernetes_manifest.test.object.metadata.name":      name,
		"kubernetes_manifest.test.object.data.foo":           "bar",
	})
	tfstate.AssertAttributeDoesNotExist(t, "kubernetes_manifest.test.data.fizz")

	tf.RequireApply(t)

	tfstate = tfstatehelper.NewHelper(tf.RequireState(t))
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.namespace": namespace,
		"kubernetes_manifest.test.object.metadata.name":      name,
		"kubernetes_manifest.test.object.data.foo":           "bar",
		"kubernetes_manifest.test.object.data.fizz":          "buzz",
	})
}
