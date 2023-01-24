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

func TestDataSourceKubernetesResource_ConfigMap(t *testing.T) {
	ctx := context.Background()

	reattachInfo, err := provider.ServeTest(ctx, hclog.Default(), t)
	if err != nil {
		t.Errorf("Failed to create provider instance: %q", err)
	}

	name := randName()
	name2 := randName()
	namespace := randName()

	k8shelper.CreateNamespace(t, namespace)
	defer k8shelper.DeleteResource(t, namespace, kubernetes.NewGroupVersionResource("v1", "namespaces"))

	// STEP 1: Create a ConfigMap to use as a data source
	tf := tfhelper.RequireNewWorkingDir(ctx, t)
	tf.SetReattachInfo(ctx, reattachInfo)
	defer func() {
		tf.Destroy(ctx)
		tf.Close()
		k8shelper.AssertNamespacedResourceDoesNotExist(t, "v1", "configmaps", namespace, name)
	}()

	tfvars := TFVARS{
		"name":      name,
		"name2":     name2,
		"namespace": namespace,
	}
	tfconfig := loadTerraformConfig(t, "datasource/step1.tf", tfvars)
	tf.SetConfig(ctx, tfconfig)
	tf.Init(ctx)
	tf.Apply(ctx)

	k8shelper.AssertNamespacedResourceExists(t, "v1", "configmaps", namespace, name)

	// STEP 2: Create another ConfigMap using the ConfigMap from step 1 as a data source
	reattachInfo2, err := provider.ServeTest(ctx, hclog.Default(), t)
	if err != nil {
		t.Errorf("Failed to create additional provider instance: %q", err)
	}
	step2 := tfhelper.RequireNewWorkingDir(ctx, t)
	step2.SetReattachInfo(ctx, reattachInfo2)
	defer func() {
		step2.Destroy(ctx)
		step2.Close()
		k8shelper.AssertNamespacedResourceDoesNotExist(t, "v1", "configmaps", namespace, name2)
	}()

	tfconfig = loadTerraformConfig(t, "datasource/step2.tf", tfvars)
	step2.SetConfig(ctx, string(tfconfig))
	step2.Init(ctx)
	step2.Apply(ctx)

	s2, err := step2.State(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve terraform state: %q", err)
	}
	tfstate := tfstatehelper.NewHelper(s2)

	// check the data source
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"data.kubernetes_resource.test_config.object.data.TEST": "hello world",
	})
	// check the resource was created with the correct value
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test_config2.object.data.TEST": "hello world",
	})
}

func TestDataSourceKubernetesResources_Namespaces(t *testing.T) {
	ctx := context.Background()

	reattachInfo, err := provider.ServeTest(ctx, hclog.Default(), t)
	if err != nil {
		t.Errorf("Failed to create provider instance: %q", err)
	}

	namespace := randName()

	// STEP 1: Create Namespaces for use with label selector
	k8shelper.CreateNamespace(t, namespace)
	defer k8shelper.DeleteResource(t, namespace, kubernetes.NewGroupVersionResource("v1", "namespaces"))

	// STEP 2: Create three ConfigMap to use as a data source
	// First ConfigMap
	configMap1 := tfhelper.RequireNewWorkingDir(ctx, t)
	configMap1.SetReattachInfo(ctx, reattachInfo)
	name := randName()

	defer func() {
		configMap1.Destroy(ctx)
		configMap1.Close()
		k8shelper.AssertNamespacedResourceDoesNotExist(t, "v1", "configmaps", namespace, name)
	}()

	cmVars1 := TFVARS{
		"name":      name,
		"namespace": namespace,
	}
	cmConfig1 := loadTerraformConfig(t, "datasource_plural/step1.tf", cmVars1)
	configMap1.SetConfig(ctx, cmConfig1)
	configMap1.Init(ctx)
	configMap1.Apply(ctx)

	k8shelper.AssertNamespacedResourceExists(t, "v1", "configmaps", namespace, name)

	// Second ConfigMap
	configMap2 := tfhelper.RequireNewWorkingDir(ctx, t)
	configMap2.SetReattachInfo(ctx, reattachInfo)
	name2 := randName()

	defer func() {
		configMap2.Destroy(ctx)
		configMap2.Close()
		k8shelper.AssertNamespacedResourceDoesNotExist(t, "v1", "configmaps", namespace, name2)
	}()

	cmVars2 := TFVARS{
		"name":      name2,
		"namespace": namespace,
	}
	cmConfig2 := loadTerraformConfig(t, "datasource_plural/step1.tf", cmVars2)
	configMap2.SetConfig(ctx, cmConfig2)
	configMap2.Init(ctx)
	configMap2.Apply(ctx)

	k8shelper.AssertNamespacedResourceExists(t, "v1", "configmaps", namespace, name2)

	// Third ConfigMap
	configMap3 := tfhelper.RequireNewWorkingDir(ctx, t)
	configMap3.SetReattachInfo(ctx, reattachInfo)
	name3 := randName()

	defer func() {
		configMap3.Destroy(ctx)
		configMap3.Close()
		k8shelper.AssertNamespacedResourceDoesNotExist(t, "v1", "configmaps", namespace, name3)
	}()

	cmVars3 := TFVARS{
		"name":      name3,
		"namespace": namespace,
	}
	cmConfig3 := loadTerraformConfig(t, "datasource_plural/step1.tf", cmVars3)
	configMap3.SetConfig(ctx, cmConfig3)
	configMap3.Init(ctx)
	configMap3.Apply(ctx)

	k8shelper.AssertNamespacedResourceExists(t, "v1", "configmaps", namespace, name3)

	//TODO create 3 config maps
	// filter

	filter := tfhelper.RequireNewWorkingDir(ctx, t)
	filter.SetReattachInfo(ctx, reattachInfo)

	defer func() {
		filter.Destroy(ctx)
		filter.Close()
	}()

	// Step 3: filter using label_selector

	filterVars := TFVARS{
		"label_selector": "kubernetes.io/metadata.name!=terraform",
		"limit":          2,
	}
	filterConfig := loadTerraformConfig(t, "datasource_plural/resources.tf", filterVars)
	filter.SetConfig(ctx, filterConfig)
	filter.Init(ctx)
	filter.Apply(ctx)

	tfState, err := filter.State(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve terraform state: %q", err)
	}
	state := tfstatehelper.NewHelper(tfState)

	// check the data source
	state.AssertAttributeLen(t, "data.kubernetes_resources.example.objects", 2)
}
