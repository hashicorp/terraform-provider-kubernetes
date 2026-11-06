// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance
// +build acceptance

package acceptance

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/go-hclog"
	version "github.com/hashicorp/go-version"

	"github.com/hashicorp/terraform-provider-kubernetes/manifest/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/kubernetes"
	tfstatehelper "github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/state"
)

func TestKubernetesManifest_BootstrapConnection(t *testing.T) {
	ctx := context.Background()

	kubeconfig := os.Getenv("KUBE_CONFIG_PATH")
	if kubeconfig == "" {
		t.Skip("KUBE_CONFIG_PATH must be set for the bootstrap_connection acceptance test")
	}

	reattachInfo, err := provider.ServeTest(ctx, hclog.Default(), t)
	if err != nil {
		t.Errorf("Failed to create provider instance: %q", err)
	}

	name := randName()
	namespace := randName()

	tf := tfhelper.RequireNewWorkingDir(ctx, t)
	tf.SetReattachInfo(ctx, reattachInfo)

	tfversion, err := tf.Version(ctx)
	if err != nil {
		tf.Close()
		t.Fatalf("Failed to retrieve terraform version: %v", err)
	}
	if constraint, _ := version.NewConstraint(">= 1.4.0"); !constraint.Check(tfversion) {
		tf.Close()
		t.Skipf("Skipping: terraform %s predates terraform_data (1.4.0), which this fixture uses to make the connection unknown at plan time", tfversion)
	}

	defer func() {
		tf.Destroy(ctx)
		tf.Close()
		k8shelper.AssertNamespacedResourceDoesNotExist(t, "v1", "configmaps", namespace, name)
	}()

	k8shelper.CreateNamespace(t, namespace)
	defer k8shelper.DeleteResource(t, namespace, kubernetes.NewGroupVersionResource("v1", "namespaces"))

	tfvars := TFVARS{
		"namespace":  namespace,
		"name":       name,
		"kubeconfig": kubeconfig,
	}
	tfconfig := loadTerraformConfig(t, "BootstrapConnection/bootstrap_connection.tf", tfvars)
	tf.SetConfig(ctx, tfconfig)
	tf.Init(ctx)

	// The connection is not knowable at plan time, so planning must succeed with
	// "object" unknown rather than failing to reach a cluster.
	if err := tf.CreatePlan(ctx); err != nil {
		t.Fatalf("Failed to create plan: %q", err)
	}
	plan, err := tf.SavedPlan(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve saved plan: %q", err)
	}

	plannedIdx := -1
	for i, rc := range plan.ResourceChanges {
		if rc.Address == "kubernetes_manifest.test" {
			plannedIdx = i
			break
		}
	}
	if plannedIdx < 0 {
		t.Fatal("kubernetes_manifest.test not found in plan")
	}
	planned := plan.ResourceChanges[plannedIdx]
	if len(planned.Change.Actions) != 1 || planned.Change.Actions[0] != "create" {
		t.Fatalf("expected a create action, planned for: %q", planned.Change.Actions)
	}
	afterUnknown, ok := planned.Change.AfterUnknown.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected AfterUnknown shape: %T", planned.Change.AfterUnknown)
	}
	if unknown, found := afterUnknown["object"]; !found || unknown == false {
		t.Error(`expected "object" to be unknown at plan time while the connection is unknown`)
	}

	// The apply re-plan resolves the connection and does the cluster-dependent work.
	tf.Apply(ctx)

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

	tfconfig = loadTerraformConfig(t, "BootstrapConnection/bootstrap_connection_modified.tf", tfvars)
	tf.SetConfig(ctx, tfconfig)

	if err := tf.CreatePlan(ctx); err != nil {
		t.Fatalf("Failed to create plan after editing the connection: %q", err)
	}
	plan, err = tf.SavedPlan(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve saved plan: %q", err)
	}

	for _, rc := range plan.ResourceChanges {
		if rc.Address != "kubernetes_manifest.test" {
			continue
		}
		if len(rc.Change.Actions) != 1 || rc.Change.Actions[0] != "no-op" {
			t.Errorf("expected editing the connection to plan nothing, planned for: %q", rc.Change.Actions)
		}
	}
}
