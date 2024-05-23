// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance
// +build acceptance

package acceptance

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/provider"
	tfstatehelper "github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/state"
)

func TestKubernetesManifest_ComputedFields(t *testing.T) {
	ctx := context.Background()

	reattachInfo, err := provider.ServeTest(ctx, hclog.Default(), t)
	if err != nil {
		t.Errorf("Failed to create provider instance: %q", err)
	}

	name := strings.ToLower(randName())
	namespace := strings.ToLower(randName())
	webhook_image := "tf-k8s-acc-webhook"

	tfvars := TFVARS{
		"name":          name,
		"namespace":     namespace,
		"webhook_image": webhook_image,
	}

	// Step 1: install a mutating webhook that annotates resources.
	// We will later check for this annotation on the test subject resource.
	step1 := tfhelper.RequireNewWorkingDir(ctx, t)
	step1.SetReattachInfo(ctx, reattachInfo)
	defer func() {
		step1.Destroy(ctx)
		step1.Close()
		k8shelper.AssertNamespacedResourceDoesNotExist(t, "v1", "secrets", namespace, name)
		k8shelper.AssertNamespacedResourceDoesNotExist(t, "v1", "services", namespace, name)
		k8shelper.AssertNamespacedResourceDoesNotExist(t, "v1", "deployments", namespace, name)
		k8shelper.AssertResourceDoesNotExist(t, "admissionregistration.k8s.io", "mutatingwebhookconfigurations", name)
	}()

	tfconfig := loadTerraformConfig(t, "ComputedFields/webhook/deploy/webhook.tf", tfvars)
	step1.SetConfig(ctx, string(tfconfig))
	step1.Init(ctx)
	step1.Apply(ctx)
	k8shelper.AssertNamespacedResourceExists(t, "v1", "secrets", namespace, name)
	k8shelper.AssertNamespacedResourceExists(t, "v1", "services", namespace, name)
	k8shelper.AssertNamespacedResourceExists(t, "apps/v1", "deployments", namespace, name)
	k8shelper.AssertResourceExists(t, "admissionregistration.k8s.io/v1", "mutatingwebhookconfigurations", name)

	// wait for API to finish installing the webhook
	time.Sleep(10 * time.Second) //lintignore:R018

	// Step 2: deploy the test subject resource and check for the annotation set by our webhook.
	step2 := tfhelper.RequireNewWorkingDir(ctx, t)
	step2.SetReattachInfo(ctx, reattachInfo)
	defer func() {
		step2.Destroy(ctx)
		step2.Close()
		k8shelper.AssertNamespacedResourceDoesNotExist(t, "v1", "configmaps", namespace, name)
	}()

	tfconfig = loadTerraformConfig(t, "ComputedFields/computed.tf", tfvars)
	step2.SetConfig(ctx, string(tfconfig))
	step2.Init(ctx)
	step2.Apply(ctx)
	k8shelper.AssertNamespacedResourceExists(t, "v1", "configmaps", namespace, name)

	s2, err := step2.State(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve terraform state: %q", err)
	}
	tfstate := tfstatehelper.NewHelper(s2)
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.name":                   name,
		"kubernetes_manifest.test.object.metadata.namespace":              namespace,
		"kubernetes_manifest.test.object.metadata.annotations.tf-k8s-acc": "true",
		"kubernetes_manifest.test.object.metadata.annotations.mutated":    "true",
	})
}
