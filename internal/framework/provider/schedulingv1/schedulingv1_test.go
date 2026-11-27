// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package schedulingv1_test

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	sdkv2 "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func sdkv2providerMeta() func() any {
	p := kubernetes.Provider()
	p.Configure(context.Background(), sdkv2.NewResourceConfigRaw(nil))
	return p.Meta
}

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": providerserver.NewProtocol6WithError(provider.New("test", sdkv2providerMeta())),
}

// testAccMuxProtoV6ProviderFactories serves the full muxed provider, so a provider block in
// the test config (ignore_annotations, ignore_labels) reaches the SDKv2 meta the framework
// resource reads. The factory above configures that meta from the environment only.
var testAccMuxProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": func() (tfprotov6.ProviderServer, error) {
		return mux.MuxServer(context.Background(), "test")
	},
}

const priorityClassV1Address = "kubernetes_priority_class_v1.test"

// State paths compared across steps with statecheck.CompareValue: the same uid means the
// object was not replaced, and the same resource_version means it was not modified. It does
// not show that no request was sent: a PATCH that changes nothing leaves resource_version as
// it was.
var (
	uidPath             = tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("uid")
	resourceVersionPath = tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("resource_version")
)

// testAccCheckPriorityClassV1Destroy matches both type names: upgrade and move tests start
// under the SDKv2 types, and a failure before the framework step leaves state there.
func testAccCheckPriorityClassV1Destroy(s *terraform.State) error {
	conn, err := sdkv2providerMeta()().(kubernetes.KubeClientsets).MainClientset()
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_priority_class_v1" && rs.Type != "kubernetes_priority_class" {
			continue
		}
		_, err := conn.SchedulingV1().PriorityClasses().Get(context.Background(), rs.Primary.ID, metav1.GetOptions{})
		if err == nil {
			return fmt.Errorf("PriorityClass still exists: %s", rs.Primary.ID)
		}
		if !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}
