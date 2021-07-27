// +build acceptance

package acceptance

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform-provider-kubernetes-alpha/provider"
	tfstatehelper "github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/state"
)

func TestKubernetesManifest_CustomResource(t *testing.T) {
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

	step1 := tfhelper.RequireNewWorkingDir(t)
	step1.SetReattachInfo(reattachInfo)
	defer func() {
		step1.RequireDestroy(t)
		step1.Close()
		k8shelper.AssertResourceDoesNotExist(t, "apiextensions.k8s.io/v1", "customresourcedefinitions", crd)
	}()

	tfconfig := loadTerraformConfig(t, "CustomResourceDefinition/customresourcedefinition.tf", tfvars)
	step1.RequireSetConfig(t, string(tfconfig))
	step1.RequireInit(t)
	step1.RequireApply(t)
	k8shelper.AssertResourceExists(t, "apiextensions.k8s.io/v1", "customresourcedefinitions", crd)

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

	tfconfig = loadTerraformConfig(t, "CustomResource/custom_resource.tf", tfvars)
	step2.RequireSetConfig(t, string(tfconfig))
	step2.RequireInit(t)
	step2.RequireApply(t)

	tfstate := tfstatehelper.NewHelper(step2.RequireState(t))
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.name":      name,
		"kubernetes_manifest.test.object.metadata.namespace": namespace,
		"kubernetes_manifest.test.object.data":               "this is a test",
	})
}
