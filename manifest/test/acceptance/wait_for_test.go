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

func TestKubernetesManifest_WaitForFields_Pod(t *testing.T) {
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
		k8shelper.AssertNamespacedResourceDoesNotExist(t, "v1", "pods", namespace, name)
	}()

	k8shelper.CreateNamespace(t, namespace)
	defer k8shelper.DeleteResource(t, namespace, kubernetes.NewGroupVersionResource("v1", "namespaces"))

	tfvars := TFVARS{
		"namespace": namespace,
		"name":      name,
	}
	tfconfig := loadTerraformConfig(t, "WaitFor/wait_for_fields_pod.tf", tfvars)
	tf.RequireSetConfig(t, tfconfig)
	tf.RequireInit(t)

	startTime := time.Now()
	tf.RequireApply(t)

	k8shelper.AssertNamespacedResourceExists(t, "v1", "pods", namespace, name)

	// NOTE We set a readinessProbe in the fixture with a delay of 10s
	// so the apply should take at least 10 seconds to complete.
	minDuration := time.Duration(5) * time.Second
	applyDuration := time.Since(startTime)
	if applyDuration < minDuration {
		t.Fatalf("the apply should have taken at least %s", minDuration)
	}

	tfstate := tfstatehelper.NewHelper(tf.RequireState(t))
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.wait_for": map[string]interface{}{
			"fields": map[string]interface{}{
				"metadata.annotations[\"test.terraform.io\"]": "test",

				"status.containerStatuses[0].ready":        "true",
				"status.containerStatuses[0].restartCount": "0",

				"status.podIP": "^(\\d+(\\.|$)){4}",
				"status.phase": "Running",
			},
		},
	})
}
