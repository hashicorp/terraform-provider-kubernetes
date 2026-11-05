// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance
// +build acceptance

package acceptance

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/provider"
	tfstatehelper "github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/state"
)

func TestKubernetesManifest_CustomResource_x_preserve_unknown_fields_unknown_value(t *testing.T) {
	ctx := context.Background()

	reattachInfo, err := provider.ServeTest(ctx, hclog.Default(), t)
	if err != nil {
		t.Fatalf("Failed to create provider instance: %q", err)
	}

	rawKind := randString(8)
	kind := strings.ToUpper(rawKind[:1]) + rawKind[1:]
	plural := strings.ToLower(kind) + "s"
	group := "terraform.io"
	crVersion := "v1"
	groupVersion := group + "/" + crVersion
	crd := fmt.Sprintf("%s.%s", plural, group)

	name := strings.ToLower(randName())
	namespace := "default"

	tfvars := TFVARS{
		"name":          name,
		"namespace":     namespace,
		"kind":          kind,
		"plural":        plural,
		"group":         group,
		"group_version": groupVersion,
		"cr_version":    crVersion,
	}

	crdStep := tfhelper.RequireNewWorkingDir(ctx, t)
	crdStep.SetReattachInfo(ctx, reattachInfo)
	defer func() {
		crdStep.Destroy(ctx)
		crdStep.Close()
		k8shelper.AssertResourceDoesNotExist(t, "apiextensions.k8s.io/v1", "customresourcedefinitions", crd)
	}()

	crdStep.SetConfig(ctx, loadTerraformConfig(t, "x-kubernetes-preserve-unknown-fields-unknown/crd/test.tf", tfvars))
	if err := crdStep.Init(ctx); err != nil {
		t.Fatalf("CRD init: %s", err)
	}
	if err := crdStep.Apply(ctx); err != nil {
		t.Fatalf("CRD apply: %s", err)
	}
	k8shelper.AssertResourceExists(t, "apiextensions.k8s.io/v1", "customresourcedefinitions", crd)

	// wait for API to finish ingesting the CRD
	time.Sleep(5 * time.Second) //lintignore:R018

	reattachInfo2, err := provider.ServeTest(ctx, hclog.Default(), t)
	if err != nil {
		t.Fatalf("Failed to create additional provider instance: %q", err)
	}

	step := tfhelper.RequireNewWorkingDir(ctx, t)
	step.SetReattachInfo(ctx, reattachInfo2)
	defer func() {
		step.Destroy(ctx)
		step.Close()
		k8shelper.AssertNamespacedResourceDoesNotExist(t, groupVersion, plural, namespace, name)
	}()

	step.SetConfig(ctx, loadTerraformConfig(t, "x-kubernetes-preserve-unknown-fields-unknown/test.tf", tfvars))
	if err := step.Init(ctx); err != nil {
		t.Fatalf("init: %s", err)
	}
	if err := step.CreatePlan(ctx); err != nil {
		t.Fatalf("plan: %s", err)
	}
	if err := step.Apply(ctx); err != nil {
		t.Fatalf("apply: %s", err)
	}
	k8shelper.AssertNamespacedResourceExists(t, groupVersion, plural, namespace, name)

	s, err := step.State(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve terraform state: %q", err)
	}
	tfstate := tfstatehelper.NewHelper(s)
	clientID := tfstate.GetAttributeValue(t, "terraform_data.client.id")
	tfstate.AssertAttributeNotEmpty(t, "terraform_data.client.id")
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.name":      name,
		"kubernetes_manifest.test.object.metadata.namespace": namespace,
		"kubernetes_manifest.test.object.spec.defaultValues": map[string]interface{}{
			"known":     "already",
			"client_id": clientID,
		},
	})
}
