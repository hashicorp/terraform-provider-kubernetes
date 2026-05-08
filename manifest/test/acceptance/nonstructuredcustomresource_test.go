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

	"github.com/Masterminds/semver"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/provider"
	tfstatehelper "github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/state"
)

func TestKubernetesManifest_NonStructuredCustomResource(t *testing.T) {
	ctx := context.Background()

	reattachInfo, err := provider.ServeTest(ctx, hclog.Default(), t)
	if err != nil {
		t.Errorf("Failed to create provider instance: %q", err)
	}

	cv, err := semver.NewVersion(k8shelper.ClusterVersion().String())
	if err != nil {
		t.Skip("cannot determine cluster version")
	}
	mv, err := semver.NewConstraint(">= 1.22.0")
	if err != nil {
		t.Skip("cannot establish cluster version constraint")
	}
	if mv.Check(cv) {
		t.Skip("only applicable to cluster versions < 1.22")
	}
	kind := randName()
	plural := strings.ToLower(kind) + "s"
	group := "k8s.terraform.io"
	version := "v1"
	groupVersion := group + "/" + version
	crd := fmt.Sprintf("%s.%s", plural, group)

	name := strings.ToLower(randName())
	namespace := randName()

	k8shelper.CreateNamespace(t, namespace)

	tfvars := TFVARS{
		"name":          name,
		"namespace":     namespace,
		"kind":          kind,
		"plural":        plural,
		"group":         group,
		"group_version": groupVersion,
		"cr_version":    version,
		"testdata":      "hello world",
	}

	step1 := tfhelper.RequireNewWorkingDir(ctx, t)
	step1.SetReattachInfo(ctx, reattachInfo)
	defer func() {
		step1.Destroy(ctx)
		step1.Close()
		k8shelper.AssertResourceDoesNotExist(t, "apiextensions.k8s.io/v1beta1", "customresourcedefinitions", crd)
	}()

	// create the CRD for the non-structured resource
	tfconfig := loadTerraformConfig(t, "NonStructuredCustomResourceDefinition/customresourcedefinition.tf", tfvars)
	step1.SetConfig(ctx, string(tfconfig))
	step1.Init(ctx)
	step1.Apply(ctx)
	k8shelper.AssertResourceExists(t, "apiextensions.k8s.io/v1beta1", "customresourcedefinitions", crd)

	// wait for API to finish ingesting the CRD
	time.Sleep(5 * time.Second) //lintignore:R018

	reattachInfo2, err := provider.ServeTest(ctx, hclog.Default(), t)
	if err != nil {
		t.Errorf("Failed to create additional provider instance: %q", err)
	}
	step2 := tfhelper.RequireNewWorkingDir(ctx, t)
	step2.SetReattachInfo(ctx, reattachInfo2)
	defer func() {
		step2.Destroy(ctx)
		step2.Close()
		k8shelper.AssertResourceDoesNotExist(t, groupVersion, plural, name)
	}()

	// create non-structured resource
	tfconfig = loadTerraformConfig(t, "NonStructuredCustomResource/custom_resource.tf", tfvars)
	step2.SetConfig(ctx, string(tfconfig))
	step2.Init(ctx)
	step2.Apply(ctx)

	s2, err := step2.State(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve terraform state: %q", err)
	}
	tfstate := tfstatehelper.NewHelper(s2)
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.name":        name,
		"kubernetes_manifest.test.object.metadata.namespace":   namespace,
		"kubernetes_manifest.test.object.data.nested.testdata": "hello world",
	})

	// update the non-structured resource
	tfvars["testdata"] = "updated"
	tfconfig = loadTerraformConfig(t, "NonStructuredCustomResource/custom_resource.tf", tfvars)
	step2.SetConfig(ctx, string(tfconfig))
	step2.Init(ctx)
	step2.Apply(ctx)

	// updating a non-structured custom resource should force a replacement
	// so the generation should be 1
	k8shelper.AssertResourceGeneration(t, groupVersion, plural, namespace, name, 1)

	s2, err = step2.State(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve terraform state: %q", err)
	}
	tfstate = tfstatehelper.NewHelper(s2)
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.name":        name,
		"kubernetes_manifest.test.object.metadata.namespace":   namespace,
		"kubernetes_manifest.test.object.data.nested.testdata": "updated",
	})
}
