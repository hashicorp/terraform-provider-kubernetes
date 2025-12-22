// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance
// +build acceptance

package acceptance

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/provider"
	tfstatehelper "github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/state"
)

func TestKubernetesManifest_CustomResource_x_preserve_unknown_fields(t *testing.T) {
	ctx := context.Background()

	reattachInfo, err := provider.ServeTest(ctx, hclog.Default(), t)
	if err != nil {
		t.Errorf("Failed to create provider instance: %q", err)
	}

	kind := strings.Title(randString(8))
	plural := strings.ToLower(kind) + "s"
	group := "terraform.io"
	version := "v1"
	groupVersion := group + "/" + version
	crd := fmt.Sprintf("%s.%s", plural, group)

	name := strings.ToLower(randName())
	namespace := "default" //randName()

	tfvars := TFVARS{
		"name":          name,
		"namespace":     namespace,
		"kind":          kind,
		"plural":        plural,
		"group":         group,
		"group_version": groupVersion,
		"cr_version":    version,
	}

	crdStep := tfhelper.RequireNewWorkingDir(ctx, t)
	crdStep.SetReattachInfo(ctx, reattachInfo)
	defer func() {
		crdStep.Destroy(ctx)
		crdStep.Close()
		k8shelper.AssertResourceDoesNotExist(t, "apiextensions.k8s.io/v1", "customresourcedefinitions", crd)
	}()

	tfconfig := loadTerraformConfig(t, "x-kubernetes-preserve-unknown-fields/crd/test.tf", tfvars)
	crdStep.SetConfig(ctx, string(tfconfig))
	crdStep.Init(ctx)
	crdStep.Apply(ctx)
	k8shelper.AssertResourceExists(t, "apiextensions.k8s.io/v1", "customresourcedefinitions", crd)

	// wait for API to finish ingesting the CRD
	time.Sleep(5 * time.Second) //lintignore:R018

	reattachInfo2, err := provider.ServeTest(ctx, hclog.Default(), t)
	if err != nil {
		t.Errorf("Failed to create additional provider instance: %q", err)
	}

	step1 := tfhelper.RequireNewWorkingDir(ctx, t)
	step1.SetReattachInfo(ctx, reattachInfo2)
	defer func() {
		step1.Destroy(ctx)
		step1.Close()
		k8shelper.AssertResourceDoesNotExist(t, groupVersion, kind, name)
	}()

	tfconfig = loadTerraformConfig(t, "x-kubernetes-preserve-unknown-fields/test-cr-1.tf", tfvars)
	step1.SetConfig(ctx, string(tfconfig))
	step1.Init(ctx)
	step1.Apply(ctx)

	s1, err := step1.State(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve terraform state: %q", err)
	}
	tfstate := tfstatehelper.NewHelper(s1)
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.name":      name,
		"kubernetes_manifest.test.object.metadata.namespace": namespace,
		"kubernetes_manifest.test.object.spec.count":         json.Number("100"),
		"kubernetes_manifest.test.object.spec.resources": map[string]interface{}{
			"foo": interface{}("bar"),
		},
	})

	tfconfig = loadTerraformConfig(t, "x-kubernetes-preserve-unknown-fields/test-cr-2.tf", tfvars)
	step1.SetConfig(ctx, string(tfconfig))
	step1.Apply(ctx)

	s2, err := step1.State(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve terraform state: %q", err)
	}
	tfstate2 := tfstatehelper.NewHelper(s2)
	tfstate2.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.name":      name,
		"kubernetes_manifest.test.object.metadata.namespace": namespace,
		"kubernetes_manifest.test.object.spec.count":         json.Number("100"),
		"kubernetes_manifest.test.object.spec.resources": map[string]interface{}{
			"foo": interface{}("bar"),
			"baz": interface{}("42"),
		},
	})

	tfconfig = loadTerraformConfig(t, "x-kubernetes-preserve-unknown-fields/test-cr-3.tf", tfvars)
	step1.SetConfig(ctx, string(tfconfig))
	step1.Apply(ctx)

	s3, err := step1.State(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve terraform state: %q", err)
	}
	tfstate3 := tfstatehelper.NewHelper(s3)
	tfstate3.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.name":      name,
		"kubernetes_manifest.test.object.metadata.namespace": namespace,
		"kubernetes_manifest.test.object.spec.count":         json.Number("100"),
		"kubernetes_manifest.test.object.spec.resources": map[string]interface{}{
			"foo": interface{}([]interface{}{"bar"}),
			"baz": interface{}("42"),
		},
	})

	tfconfig = loadTerraformConfig(t, "x-kubernetes-preserve-unknown-fields/test-cr-4.tf", tfvars)
	step1.SetConfig(ctx, string(tfconfig))
	step1.Apply(ctx)

	s4, err := step1.State(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve terraform state: %q", err)
	}
	tfstate4 := tfstatehelper.NewHelper(s4)
	tfstate4.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.name":      name,
		"kubernetes_manifest.test.object.metadata.namespace": namespace,
		"kubernetes_manifest.test.object.spec.count":         json.Number("100"),
		"kubernetes_manifest.test.object.spec.resources":     false,
	})
}

// TestKubernetesManifest_CustomResource_x_preserve_unknown_fields_empty_object
// is a regression test for the DynamicPseudoType preservation fix.
// Without the fix, creating a resource with an empty object ({}) in a field
// marked with x-kubernetes-preserve-unknown-fields would fail with:
//
//	"Provider produced inconsistent result after apply"
//	"wrong final value type: incorrect object attributes"
//
// This happens because the schema uses DynamicPseudoType for such fields,
// but the value conversion pipeline was using the actual value's type (Object{})
// instead of preserving the schema type (DynamicPseudoType).
func TestKubernetesManifest_CustomResource_x_preserve_unknown_fields_empty_object(t *testing.T) {
	ctx := context.Background()

	reattachInfo, err := provider.ServeTest(ctx, hclog.Default(), t)
	if err != nil {
		t.Errorf("Failed to create provider instance: %q", err)
	}

	kind := strings.Title(randString(8))
	plural := strings.ToLower(kind) + "s"
	group := "terraform.io"
	version := "v1"
	groupVersion := group + "/" + version
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
		"cr_version":    version,
	}

	// Step 1: Create the CRD
	crdStep := tfhelper.RequireNewWorkingDir(ctx, t)
	crdStep.SetReattachInfo(ctx, reattachInfo)
	defer func() {
		crdStep.Destroy(ctx)
		crdStep.Close()
		k8shelper.AssertResourceDoesNotExist(t, "apiextensions.k8s.io/v1", "customresourcedefinitions", crd)
	}()

	tfconfig := loadTerraformConfig(t, "x-kubernetes-preserve-unknown-fields/crd/test.tf", tfvars)
	crdStep.SetConfig(ctx, string(tfconfig))
	crdStep.Init(ctx)
	crdStep.Apply(ctx)
	k8shelper.AssertResourceExists(t, "apiextensions.k8s.io/v1", "customresourcedefinitions", crd)

	// wait for API to finish ingesting the CRD
	time.Sleep(5 * time.Second) //lintignore:R018

	// Step 2: Create a CR with an empty object in a DynamicPseudoType field
	// This is the regression test - without the fix, this Apply would fail
	reattachInfo2, err := provider.ServeTest(ctx, hclog.Default(), t)
	if err != nil {
		t.Errorf("Failed to create additional provider instance: %q", err)
	}

	crStep := tfhelper.RequireNewWorkingDir(ctx, t)
	crStep.SetReattachInfo(ctx, reattachInfo2)
	defer func() {
		crStep.Destroy(ctx)
		crStep.Close()
		k8shelper.AssertResourceDoesNotExist(t, groupVersion, kind, name)
	}()

	// Apply config with empty object - this would fail without the DynamicPseudoType fix
	tfconfig = loadTerraformConfig(t, "x-kubernetes-preserve-unknown-fields/test-cr-empty.tf", tfvars)
	crStep.SetConfig(ctx, string(tfconfig))
	crStep.Init(ctx)
	crStep.Apply(ctx)

	// Verify the resource was created with the empty object
	s1, err := crStep.State(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve terraform state: %q", err)
	}
	tfstate := tfstatehelper.NewHelper(s1)
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.name":      name,
		"kubernetes_manifest.test.object.metadata.namespace": namespace,
		"kubernetes_manifest.test.object.spec.count":         json.Number("100"),
		"kubernetes_manifest.test.object.spec.resources":     map[string]interface{}{},
	})

	// Step 3: Verify we can update from empty object to object with data
	reattachInfo3, err := provider.ServeTest(ctx, hclog.Default(), t)
	if err != nil {
		t.Errorf("Failed to create additional provider instance: %q", err)
	}
	crStep.SetReattachInfo(ctx, reattachInfo3)

	tfconfig = loadTerraformConfig(t, "x-kubernetes-preserve-unknown-fields/test-cr-1.tf", tfvars)
	crStep.SetConfig(ctx, string(tfconfig))
	crStep.Apply(ctx)

	s2, err := crStep.State(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve terraform state: %q", err)
	}
	tfstate2 := tfstatehelper.NewHelper(s2)
	tfstate2.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.name":      name,
		"kubernetes_manifest.test.object.metadata.namespace": namespace,
		"kubernetes_manifest.test.object.spec.count":         json.Number("100"),
		"kubernetes_manifest.test.object.spec.resources": map[string]interface{}{
			"foo": interface{}("bar"),
		},
	})
}
