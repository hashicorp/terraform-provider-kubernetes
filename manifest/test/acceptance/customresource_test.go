// Copyright IBM Corp. 2017, 2026
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

func TestKubernetesManifest_CustomResource(t *testing.T) {
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

	step1 := tfhelper.RequireNewWorkingDir(ctx, t)
	step1.SetReattachInfo(ctx, reattachInfo)
	defer func() {
		step1.Destroy(ctx)
		step1.Close()
		k8shelper.AssertResourceDoesNotExist(t, "apiextensions.k8s.io/v1", "customresourcedefinitions", crd)
	}()

	tfconfig := loadTerraformConfig(t, "CustomResourceDefinition/customresourcedefinition.tf", tfvars)
	step1.SetConfig(ctx, string(tfconfig))
	step1.Init(ctx)
	step1.Apply(ctx)
	k8shelper.AssertResourceExists(t, "apiextensions.k8s.io/v1", "customresourcedefinitions", crd)

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
		k8shelper.AssertResourceDoesNotExist(t, groupVersion, kind, name)
	}()

	tfconfig = loadTerraformConfig(t, "CustomResource/custom_resource.tf", tfvars)
	step2.SetConfig(ctx, string(tfconfig))
	step2.Init(ctx)
	step2.Apply(ctx)

	s2, err := step2.State(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve terraform state: %q", err)
	}
	tfstate := tfstatehelper.NewHelper(s2)
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.name":      name,
		"kubernetes_manifest.test.object.metadata.namespace": namespace,
		"kubernetes_manifest.test.object.data":               "this is a test",
		"kubernetes_manifest.test.object.refs":               json.Number("98.765"),
		"kubernetes_manifest.test.object.stuff.0":            map[string]interface{}{"foo": interface{}(nil)},
		"kubernetes_manifest.test.object.limits": map[string]interface{}{
			"foo": interface{}("bar"),
			"baz": interface{}("42"),
		},
		"kubernetes_manifest.test.object.limits.foo": "bar",
		"kubernetes_manifest.test.object.limits.baz": "42",
	})
}
