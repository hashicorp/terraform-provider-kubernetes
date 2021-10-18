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

func TestKubernetesManifest_EmptyBlocks(t *testing.T) {
	kind := strings.Title(randString(8))
	plural := strings.ToLower(kind) + "s"
	group := "terraform.io"
	version := "v1"
	groupVersion := group + "/" + version
	name := fmt.Sprintf("%s.%s", plural, group)
	namespace := "default"

	tf := tfhelper.RequireNewWorkingDir(t)
	tf.SetReattachInfo(reattachInfo)
	defer func() {
		tf.RequireDestroy(t)
		tf.Close()
		k8shelper.AssertResourceDoesNotExist(t, "apiextensions.k8s.io/v1", "customresourcedefinitions", name)
	}()

	tfvars := TFVARS{
		"name":          name,
		"namespace":     namespace,
		"kind":          kind,
		"plural":        plural,
		"group":         group,
		"group_version": groupVersion,
		"cr_version":    version,
	}
	tfconfig := loadTerraformConfig(t, "EmptyBlock/step1.tf", tfvars)
	tf.RequireSetConfig(t, tfconfig)
	tf.RequireInit(t)
	tf.RequireApply(t)

	k8shelper.AssertResourceExists(t, "apiextensions.k8s.io/v1", "customresourcedefinitions", name)

	// wait for API to finish ingesting the CRD
	time.Sleep(5 * time.Second) //lintignore:R018

	reattachInfo2, err := provider.ServeTest(context.TODO(), hclog.Default())
	if err != nil {
		t.Errorf("Failed to create additional provider instance: %q", err)
	}
	step2 := tfhelper.RequireNewWorkingDir(t)
	step2.SetReattachInfo(reattachInfo2)
	defer func() {
		step2.RequireDestroy(t)
		step2.Close()
		k8shelper.AssertResourceDoesNotExist(t, groupVersion, kind, name)
	}()

	tfconfig = loadTerraformConfig(t, "EmptyBlock/step2.tf", tfvars)
	step2.RequireSetConfig(t, string(tfconfig))
	step2.RequireInit(t)
	step2.RequireApply(t)

	tfstate := tfstatehelper.NewHelper(step2.RequireState(t))
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.kind":               kind,
		"kubernetes_manifest.test.object.apiVersion":         groupVersion,
		"kubernetes_manifest.test.object.metadata.name":      name,
		"kubernetes_manifest.test.object.metadata.namespace": namespace,
	})
	tfstate.AssertAttributeExists(t, "kubernetes_manifest.test.object.spec.selfSigned")
}
