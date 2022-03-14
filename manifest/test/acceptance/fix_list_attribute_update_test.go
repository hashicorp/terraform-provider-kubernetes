//go:build acceptance
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

func TestKubernetesManifest_FixListAttributeUpdate(t *testing.T) {
	ctx := context.Background()

	reattachInfo, err := provider.ServeTest(ctx, hclog.Default(), t)
	if err != nil {
		t.Errorf("Failed to create provider instance: %q", err)
	}

	name := randName()
	namespace := randName()

	k8shelper.CreateNamespace(t, namespace)
	defer k8shelper.DeleteResource(t, namespace, kubernetes.NewGroupVersionResource("v1", "namespaces"))

	tfvars := TFVARS{
		"namespace": namespace,
		"name":      name,
	}

	tf := tfhelper.RequireNewWorkingDir(t)
	tf.SetReattachInfo(reattachInfo)
	defer func() {
		tf.RequireDestroy(t)
		tf.Close()
		k8shelper.AssertNamespacedResourceDoesNotExist(t, "apps/v1", "deployments", namespace, name)
	}()

	tfconfig1 := loadTerraformConfig(t, "FixListAttributeUpdate/step1.tf", tfvars)
	tf.RequireSetConfig(t, tfconfig1)
	tf.RequireInit(t)
	tf.RequireApply(t)

	k8shelper.AssertNamespacedResourceExists(t, "apps/v1", "deployments", namespace, name)

	tfstate := tfstatehelper.NewHelper(tf.RequireState(t))
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.namespace":                    namespace,
		"kubernetes_manifest.test.object.metadata.name":                         name,
		"kubernetes_manifest.test.object.spec.template.spec.containers.0.name":  "ping",
		"kubernetes_manifest.test.object.spec.template.spec.containers.0.image": "alpine:latest",
	})

	tfstate.AssertAttributeEmpty(t, "kubernetes_manifest.test.object.spec.template.spec.tolerations")

	tfconfig2 := loadTerraformConfig(t, "FixListAttributeUpdate/step2.tf", tfvars)
	tf.RequireSetConfig(t, tfconfig2)
	tf.RequireApply(t)

	tfstate = tfstatehelper.NewHelper(tf.RequireState(t))

	tfstate.AssertAttributeNotEmpty(t, "kubernetes_manifest.test.object.spec.template.spec.tolerations")
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.spec.template.spec.tolerations.0.effect":   "NoSchedule",
		"kubernetes_manifest.test.object.spec.template.spec.tolerations.0.key":      "nvidia.com/gpu",
		"kubernetes_manifest.test.object.spec.template.spec.tolerations.0.operator": "Exists",
	})
}
