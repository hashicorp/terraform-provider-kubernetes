//go:build acceptance
// +build acceptance

package acceptance

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"

	"github.com/hashicorp/terraform-provider-kubernetes/manifest/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/kubernetes"
	tfstatehelper "github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/state"
)

func TestKubernetesManifest_WaitFields_Pod(t *testing.T) {
	ctx := context.Background()

	name := randName()
	namespace := randName()

	reattachInfo, err := provider.ServeTest(ctx, hclog.Default(), t)
	if err != nil {
		t.Errorf("Failed to create provider instance: %q", err)
	}

	tf := tfhelper.RequireNewWorkingDir(ctx, t)
	tf.SetReattachInfo(ctx, reattachInfo)
	defer func() {
		tf.Destroy(ctx)
		tf.Close()
		k8shelper.AssertNamespacedResourceDoesNotExist(t, "v1", "pods", namespace, name)
	}()

	k8shelper.CreateNamespace(t, namespace)
	defer k8shelper.DeleteResource(t, namespace, kubernetes.NewGroupVersionResource("v1", "namespaces"))

	tfvars := TFVARS{
		"namespace": namespace,
		"name":      name,
	}
	tfconfig := loadTerraformConfig(t, "Wait/wait_for_fields_pod.tf", tfvars)
	tf.SetConfig(ctx, tfconfig)
	tf.Init(ctx)

	startTime := time.Now()
	tf.Apply(ctx)

	k8shelper.AssertNamespacedResourceExists(t, "v1", "pods", namespace, name)

	// NOTE We set a readinessProbe in the fixture with a delay of 10s
	// so the apply should take at least 10 seconds to complete.
	minDuration := time.Duration(5) * time.Second
	applyDuration := time.Since(startTime)
	if applyDuration < minDuration {
		t.Fatalf("the apply should have taken at least %s", minDuration)
	}

	st, err := tf.State(ctx)
	if err != nil {
		t.Fatalf("Failed to obtain state: %q", err)
	}
	tfstate := tfstatehelper.NewHelper(st)
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.wait.0.fields": map[string]interface{}{
			"metadata.annotations[\"test.terraform.io\"]": "test",
			"status.containerStatuses[0].ready":           "true",
			"status.containerStatuses[0].restartCount":    "0",
			"status.podIP": "^(\\d+(\\.|$)){4}",
			"status.phase": "Running",
		},
	})
}

func TestKubernetesManifest_WaitRollout_Deployment(t *testing.T) {
	ctx := context.Background()

	name := randName()
	namespace := randName()

	reattachInfo, err := provider.ServeTest(ctx, hclog.Default(), t)
	if err != nil {
		t.Errorf("Failed to create provider instance: %q", err)
	}

	tf := tfhelper.RequireNewWorkingDir(ctx, t)
	tf.SetReattachInfo(ctx, reattachInfo)
	defer func() {
		tf.Destroy(ctx)
		tf.Close()
		k8shelper.AssertNamespacedResourceDoesNotExist(t, "apps/v1", "deployments", namespace, name)
	}()

	k8shelper.CreateNamespace(t, namespace)
	defer k8shelper.DeleteResource(t, namespace, kubernetes.NewGroupVersionResource("v1", "namespaces"))

	tfvars := TFVARS{
		"namespace": namespace,
		"name":      name,
	}
	tfconfig := loadTerraformConfig(t, "Wait/wait_for_rollout.tf", tfvars)
	tf.SetConfig(ctx, tfconfig)
	tf.Init(ctx)

	startTime := time.Now()
	tf.Apply(ctx)

	k8shelper.AssertNamespacedResourceExists(t, "apps/v1", "deployments", namespace, name)

	// NOTE We set a readinessProbe in the fixture with a delay of 10s
	// so the apply should take at least 10 seconds to complete.
	minDuration := time.Duration(5) * time.Second
	applyDuration := time.Since(startTime)
	if applyDuration < minDuration {
		t.Fatalf("the apply should have taken at least %s", minDuration)
	}

	st, err := tf.State(ctx)
	if err != nil {
		t.Fatalf("Failed to get state: %q", err)
	}
	tfstate := tfstatehelper.NewHelper(st)
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.wait_for_rollout.wait.0.rollout": true,
	})
}

func TestKubernetesManifest_WaitCondition_Pod(t *testing.T) {
	ctx := context.Background()

	name := randName()
	namespace := randName()

	reattachInfo, err := provider.ServeTest(ctx, hclog.Default(), t)
	if err != nil {
		t.Errorf("Failed to create provider instance: %q", err)
	}

	tf := tfhelper.RequireNewWorkingDir(ctx, t)
	tf.SetReattachInfo(ctx, reattachInfo)
	defer func() {
		tf.Destroy(ctx)
		tf.Close()
		k8shelper.AssertNamespacedResourceDoesNotExist(t, "v1", "pods", namespace, name)
	}()

	k8shelper.CreateNamespace(t, namespace)
	defer k8shelper.DeleteResource(t, namespace, kubernetes.NewGroupVersionResource("v1", "namespaces"))

	tfvars := TFVARS{
		"namespace": namespace,
		"name":      name,
	}
	tfconfig := loadTerraformConfig(t, "Wait/wait_for_conditions.tf", tfvars)
	tf.SetConfig(ctx, tfconfig)
	tf.Init(ctx)

	startTime := time.Now()
	err = tf.Apply(ctx)
	if err != nil {
		t.Fatalf("Failed to apply: %q", err)
	}

	k8shelper.AssertNamespacedResourceExists(t, "v1", "pods", namespace, name)

	// NOTE We set a readinessProbe in the fixture with a delay of 10s
	// so the apply should take at least 10 seconds to complete.
	minDuration := time.Duration(10) * time.Second
	applyDuration := time.Since(startTime)
	if applyDuration < minDuration {
		t.Fatalf("the apply should have taken at least %s", minDuration)
	}

	st, err := tf.State(ctx)
	if err != nil {
		t.Fatalf("Failed to get state: %q", err)
	}
	tfstate := tfstatehelper.NewHelper(st)
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.wait.0.condition.0.type":   "Ready",
		"kubernetes_manifest.test.wait.0.condition.0.status": "True",
		"kubernetes_manifest.test.wait.0.condition.1.type":   "ContainersReady",
		"kubernetes_manifest.test.wait.0.condition.1.status": "True",
	})
}
