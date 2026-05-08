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

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/provider"
)

func TestKubernetesManifest_CustomResource_Multiversion(t *testing.T) {
	ctx := context.Background()

	reattachInfo, err := provider.ServeTest(ctx, hclog.Default(), t)
	if err != nil {
		t.Errorf("Failed to create provider instance: %q", err)
	}

	kind1 := strings.Title(randString(8))
	plural1 := strings.ToLower(kind1) + "s"
	group1 := "terraform.io"
	version1 := "v1"
	groupVersion1 := group1 + "/" + version1
	crd1 := fmt.Sprintf("%s.%s", plural1, group1)

	kind2 := strings.Title(randString(8))
	plural2 := strings.ToLower(kind2) + "s"
	group2 := "terraform.io"
	version2 := "v1"
	groupVersion2 := group2 + "/" + version2
	crd2 := fmt.Sprintf("%s.%s", plural2, group2)

	tfvars := TFVARS{
		"kind1":          kind1,
		"plural1":        plural1,
		"group1":         group1,
		"group_version1": groupVersion1,
		"cr_version1":    version1,

		"kind2":          kind2,
		"plural2":        plural2,
		"group2":         group2,
		"group_version2": groupVersion2,
		"cr_version2":    version2,
	}

	step1 := tfhelper.RequireNewWorkingDir(ctx, t)
	step1.SetReattachInfo(ctx, reattachInfo)
	defer func() {
		step1.Destroy(ctx)
		step1.Close()
		k8shelper.AssertResourceDoesNotExist(t, "apiextensions.k8s.io/v1", "customresourcedefinitions", crd1)
		k8shelper.AssertResourceDoesNotExist(t, "apiextensions.k8s.io/v1", "customresourcedefinitions", crd2)
	}()

	tfconfig := loadTerraformConfig(t, "CustomResourceDefinition-multiversion/customresourcedefinition.tf", tfvars)
	step1.SetConfig(ctx, string(tfconfig))
	step1.Init(ctx)
	step1.Apply(ctx)
	k8shelper.AssertResourceExists(t, "apiextensions.k8s.io/v1", "customresourcedefinitions", crd1)
	k8shelper.AssertResourceExists(t, "apiextensions.k8s.io/v1", "customresourcedefinitions", crd2)

}
