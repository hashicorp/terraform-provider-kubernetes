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

	tf := tfhelper.RequireNewWorkingDir(ctx, t)
	tf.SetReattachInfo(ctx, reattachInfo)
	defer func() {
		tf.Destroy(ctx)
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
	tf.SetConfig(ctx, tfconfig)
	tf.Init(ctx)

	importId := fmt.Sprintf("apiVersion=%s,kind=%s,namespace=%s,name=%s", "v1", "ConfigMap", namespace, name)

	tf.Import(ctx, "kubernetes_manifest.test", importId)
	k8shelper.AssertNamespacedResourceExists(t, "v1", "configmaps", namespace, name)

	s, err := tf.State(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve terraform state: %q", err)
	}
	tfstate := tfstatehelper.NewHelper(s)
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.namespace": namespace,
		"kubernetes_manifest.test.object.metadata.name":      name,
		"kubernetes_manifest.test.object.data.foo":           "bar",
	})
	tfstate.AssertAttributeDoesNotExist(t, "kubernetes_manifest.test.data.fizz")

	err = tf.CreatePlan(ctx)
	if err != nil {
		t.Fatalf("Failed to create plan: %q", err)
	}
	plan, err := tf.SavedPlan(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve saved plan: %q", err)
	}

	if len(plan.ResourceChanges) != 1 || plan.ResourceChanges[0].Address != "kubernetes_manifest.test" {
		t.Fatalf("Failed to find resource in plan data: %q", plan.ResourceChanges[0].Address)
	}
	if len(plan.ResourceChanges[0].Change.Actions) != 1 || plan.ResourceChanges[0].Change.Actions[0] != "update" {
		t.Fatalf("Failed to plan for resource update - in fact, planned for: %q", plan.ResourceChanges[0].Change.Actions[0])
	}

	tf.Apply(ctx)

	s, err = tf.State(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve terraform state: %q", err)
	}
	tfstate = tfstatehelper.NewHelper(s)
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.namespace": namespace,
		"kubernetes_manifest.test.object.metadata.name":      name,
		"kubernetes_manifest.test.object.data.foo":           "bar",
		"kubernetes_manifest.test.object.data.fizz":          "buzz",
	})
}
