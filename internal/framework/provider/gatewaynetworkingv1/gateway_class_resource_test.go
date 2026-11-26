// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package gatewaynetworkingv1_test

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	sdkv2 "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
)

// TestAccGatewayClassV1_lifecycle creates and imports a class, then updates it
// in place twice: changing description and adding parameters_ref, then removing
// both. Each apply is checked against the live object and the data source, with
// the same UID and an empty follow-up plan. The last step fails unless Update
// sends nulls for the removed fields.
func TestAccGatewayClassV1_lifecycle(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-gateway-class")
	resourceName := "kubernetes_gateway_class_v1.test"
	testCase, client := testAccGatewayTestCase(t)
	sameUID := statecheck.CompareValue(compare.ValuesSame())
	check := func(description, parameterName string) resource.TestCheckFunc {
		return func(state *terraform.State) error {
			conn, err := client()
			if err != nil {
				return err
			}
			object, err := conn.GatewayV1().GatewayClasses().Get(context.Background(), name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			if string(object.UID) != state.RootModule().Resources[resourceName].Primary.Attributes["metadata.0.uid"] {
				return fmt.Errorf("GatewayClass API and state UIDs differ")
			}
			if description == "" && object.Spec.Description != nil || description != "" && (object.Spec.Description == nil || *object.Spec.Description != description) {
				return fmt.Errorf("unexpected GatewayClass API description: %v", object.Spec.Description)
			}
			if parameterName == "" && object.Spec.ParametersRef != nil || parameterName != "" && (object.Spec.ParametersRef == nil || object.Spec.ParametersRef.Name != parameterName) {
				return fmt.Errorf("unexpected GatewayClass API parametersRef: %#v", object.Spec.ParametersRef)
			}
			return nil
		}
	}
	for i, extra := range []string{
		`description = "initial"`,
		`description = "updated"
parameters_ref {
  group = "example.com"
  kind  = "GatewaySettings"
  name  = "settings"
}`,
		"",
	} {
		description, parameterName := "initial", ""
		switch i {
		case 1:
			description, parameterName = "updated", "settings"
		case 2:
			description = ""
		}
		checks := []resource.TestCheckFunc{
			check(description, parameterName),
			resource.TestCheckResourceAttr("data.kubernetes_gateway_class_v1.test", "metadata.0.name", name),
			resource.TestCheckResourceAttrPair(resourceName, "metadata.0.uid", "data.kubernetes_gateway_class_v1.test", "metadata.0.uid"),
		}
		if description != "" {
			checks = append(checks, resource.TestCheckResourceAttr(resourceName, "spec.0.description", description),
				resource.TestCheckResourceAttr("data.kubernetes_gateway_class_v1.test", "spec.0.description", description))
		} else {
			checks = append(checks, resource.TestCheckNoResourceAttr(resourceName, "spec.0.description"),
				resource.TestCheckResourceAttr("data.kubernetes_gateway_class_v1.test", "spec.0.parameters_ref.#", "0"))
		}
		if parameterName != "" {
			checks = append(checks, resource.TestCheckResourceAttr("data.kubernetes_gateway_class_v1.test", "spec.0.parameters_ref.0.name", parameterName))
		}
		step := resource.TestStep{
			Config: gatewayClassConfig(name, extra),
			Check:  resource.ComposeAggregateTestCheckFunc(checks...),
			ConfigStateChecks: []statecheck.StateCheck{
				sameUID.AddStateValue(resourceName, tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("uid")),
			},
			ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
		}
		if i > 0 {
			step.ConfigPlanChecks.PreApply = []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)}
		}
		testCase.Steps = append(testCase.Steps, step)
		if i == 0 {
			testCase.Steps = append(testCase.Steps, resource.TestStep{ResourceName: resourceName, ImportState: true, ImportStateVerify: true})
		}
	}
	resource.ParallelTest(t, testCase)
}

// TestAccGatewayClassV1_disappears deletes the object outside Terraform. With
// the API still served, Read must drop it from state, so the next plan creates
// it again and the new object has a different UID.
func TestAccGatewayClassV1_disappears(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-gateway-class")
	testCase, client := testAccGatewayTestCase(t)
	var oldUID string
	// Do not look up a deliberately absent object through a data source.
	config := gatewayClassResourceConfig(name, "")
	testCase.Steps = []resource.TestStep{
		{Config: config, Check: func(*terraform.State) error {
			conn, err := client()
			if err != nil {
				return err
			}
			object, err := conn.GatewayV1().GatewayClasses().Get(context.Background(), name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			oldUID = string(object.UID)
			return conn.GatewayV1().GatewayClasses().Delete(context.Background(), name, metav1.DeleteOptions{})
		}, ExpectNonEmptyPlan: true},
		{Config: config, Check: func(state *terraform.State) error {
			if uid := state.RootModule().Resources["kubernetes_gateway_class_v1.test"].Primary.Attributes["metadata.0.uid"]; uid == "" || uid == oldUID {
				return fmt.Errorf("GatewayClass was not recreated with a new UID")
			}
			return nil
		}, ConfigPlanChecks: resource.ConfigPlanChecks{
			PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("kubernetes_gateway_class_v1.test", plancheck.ResourceActionCreate)},
		}},
	}
	resource.ParallelTest(t, testCase)
}

// TestAccGatewayClassV1_controllerNameReplacement changes controller_name, which
// the API makes immutable. The plan must replace the object, giving it a new
// UID, rather than attempt an update the API would reject.
func TestAccGatewayClassV1_controllerNameReplacement(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-gateway-class")
	testCase, _ := testAccGatewayTestCase(t)
	differentUID := statecheck.CompareValue(compare.ValuesDiffer())
	resourceName := "kubernetes_gateway_class_v1.test"
	config := gatewayClassConfig(name, "")
	testCase.Steps = []resource.TestStep{
		{Config: config, ConfigStateChecks: []statecheck.StateCheck{
			differentUID.AddStateValue(resourceName, tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("uid")),
		}},
		{
			Config: strings.Replace(config, "terraform-provider-kubernetes.example/"+name, "terraform-provider-kubernetes.example/"+name+"-updated", 1),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionDestroyBeforeCreate)},
				PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
			},
			ConfigStateChecks: []statecheck.StateCheck{
				differentUID.AddStateValue(resourceName, tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("uid")),
			},
			Check: resource.TestCheckResourceAttr("data.kubernetes_gateway_class_v1.test", "spec.0.controller_name", "terraform-provider-kubernetes.example/"+name+"-updated"),
		},
	}
	resource.ParallelTest(t, testCase)
}

// TestAccGatewayClassV1_generatedName names the object with generate_name. The
// server-assigned name must reach id and metadata.0.name, survive import and
// leave an empty plan.
func TestAccGatewayClassV1_generatedName(t *testing.T) {
	prefix := acctest.RandomWithPrefix("tf-gateway-class") + "-"
	testCase, _ := testAccGatewayTestCase(t)
	resourceName := "kubernetes_gateway_class_v1.test"
	config := strings.Replace(gatewayClassConfig(prefix, ""), "name = ", "generate_name = ", 1)
	testCase.Steps = []resource.TestStep{
		{Config: config, Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestMatchResourceAttr(resourceName, "metadata.0.name", regexp.MustCompile("^"+regexp.QuoteMeta(prefix)+".+")),
			resource.TestCheckResourceAttrPair(resourceName, "id", "data.kubernetes_gateway_class_v1.test", "metadata.0.name"),
		)},
		{ResourceName: resourceName, ImportState: true, ImportStateVerify: true},
		{Config: config, PlanOnly: true},
	}
	resource.ParallelTest(t, testCase)
}

// TestAccGatewayClassV1_identity checks the identity written on create, then
// imports by identity alone and expects an empty plan. Resource identity needs
// Terraform 1.12 or later.
func TestAccGatewayClassV1_identity(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-gateway-class")
	testCase, _ := testAccGatewayTestCase(t)
	resourceName := "kubernetes_gateway_class_v1.test"
	config := gatewayClassConfig(name, "")
	testCase.TerraformVersionChecks = []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)}
	testCase.Steps = []resource.TestStep{
		{Config: config, ConfigStateChecks: []statecheck.StateCheck{
			statecheck.ExpectIdentity(resourceName, map[string]knownvalue.Check{
				"name": knownvalue.StringExact(name), "kind": knownvalue.StringExact("GatewayClass"),
				"api_version": knownvalue.StringExact("gateway.networking.k8s.io/v1"),
			}),
		}},
		{ResourceName: resourceName, ImportState: true, ImportStateKind: resource.ImportBlockWithResourceIdentity},
		{Config: config, PlanOnly: true},
	}
	resource.ParallelTest(t, testCase)
}

// TestAccGatewayClassV1_invalidConfig expects a schema validator to reject an
// invalid controller_name at plan time. Terraform wraps long diagnostics, so
// the pattern uses \s+ between words rather than single spaces.
func TestAccGatewayClassV1_invalidConfig(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-gateway-class")
	testCase, _ := testAccGatewayTestCase(t)
	testCase.Steps = []resource.TestStep{{
		Config:   strings.Replace(gatewayClassResourceConfig(name, ""), "terraform-provider-kubernetes.example/"+name, "invalid-controller", 1),
		PlanOnly: true, ExpectError: regexp.MustCompile(`must\s+be\s+a\s+domain-prefixed\s+controller\s+identifier`),
	}}
	resource.ParallelTest(t, testCase)
}

// TestAccGatewayClassV1_dataSourceNotFound looks up a name that does not exist.
// It must fail with the API's not-found error, not return empty values or
// report the Gateway API as unavailable.
func TestAccGatewayClassV1_dataSourceNotFound(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-gateway-class-missing")
	testCase, _ := testAccGatewayTestCase(t)
	testCase.Steps = []resource.TestStep{{
		Config: fmt.Sprintf(`
data "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = %q
  }
}
`, name),
		PlanOnly: true, ExpectError: regexp.MustCompile(`gatewayclasses\.gateway\.networking\.k8s\.io\s+"` + regexp.QuoteMeta(name) + `"\s+not\s+found`),
	}}
	resource.ParallelTest(t, testCase)
}

// testAccPreCheck mirrors the corev1 and rbacv1 checks. The cluster must also
// serve the Gateway API v1.6.3 Standard CRDs.
func testAccPreCheck(t *testing.T) {
	ctx := context.TODO()
	hasFileCfg := (os.Getenv("KUBE_CTX_AUTH_INFO") != "" && os.Getenv("KUBE_CTX_CLUSTER") != "") ||
		os.Getenv("KUBE_CTX") != "" ||
		os.Getenv("KUBE_CONFIG_PATH") != ""
	hasUserCredentials := os.Getenv("KUBE_USER") != "" && os.Getenv("KUBE_PASSWORD") != ""
	hasClientCert := os.Getenv("KUBE_CLIENT_CERT_DATA") != "" && os.Getenv("KUBE_CLIENT_KEY_DATA") != ""
	hasStaticCfg := (os.Getenv("KUBE_HOST") != "" &&
		os.Getenv("KUBE_CLUSTER_CA_CERT_DATA") != "") &&
		(hasUserCredentials || hasClientCert || os.Getenv("KUBE_TOKEN") != "")

	if !hasFileCfg && !hasStaticCfg && !hasUserCredentials {
		t.Fatalf("File config (KUBE_CTX_AUTH_INFO and KUBE_CTX_CLUSTER) or static configuration"+
			"(%s) or (%s) must be set for acceptance tests",
			strings.Join([]string{
				"KUBE_HOST",
				"KUBE_USER",
				"KUBE_PASSWORD",
				"KUBE_CLUSTER_CA_CERT_DATA",
			}, ", "),
			strings.Join([]string{
				"KUBE_HOST",
				"KUBE_CLIENT_CERT_DATA",
				"KUBE_CLIENT_KEY_DATA",
				"KUBE_CLUSTER_CA_CERT_DATA",
			}, ", "),
		)
	}

	diags := kubernetes.Provider().Configure(ctx, sdkv2.NewResourceConfigRaw(nil))
	if diags.HasError() {
		t.Fatal(diags[0].Summary)
	}
}

// testAccGatewayTestCase returns a TestCase and a client for checking the live
// API. Each test gets its own provider; API checks use the same credentials as
// Terraform. The client is a function because the provider is only configured
// once a step runs. CheckDestroy skips data sources, which share the type name
// but own nothing.
func testAccGatewayTestCase(t *testing.T) (resource.TestCase, func() (gatewayclient.Interface, error)) {
	t.Helper()
	p := kubernetes.Provider()
	client := func() (gatewayclient.Interface, error) {
		meta, ok := p.Meta().(kubernetes.GatewayClientsets)
		if !ok {
			return nil, fmt.Errorf("Gateway provider metadata is unavailable")
		}
		return meta.GatewayClientset()
	}
	return resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"kubernetes": func() (tfprotov6.ProviderServer, error) {
				return mux.MuxServerWithProvider(context.Background(), "test", p)
			},
		},
		CheckDestroy: func(state *terraform.State) error {
			for address, rs := range state.RootModule().Resources {
				if strings.HasPrefix(address, "data.") || rs.Type != "kubernetes_gateway_class_v1" {
					continue
				}
				if rs.Primary == nil {
					return fmt.Errorf("missing primary state for %s", rs.Type)
				}
				conn, err := client()
				if err != nil {
					return err
				}
				_, err = conn.GatewayV1().GatewayClasses().Get(context.Background(), rs.Primary.ID, metav1.GetOptions{})
				if apierrors.IsNotFound(err) {
					continue
				}
				if err != nil {
					return err
				}
				return fmt.Errorf("%s %s still exists", rs.Type, rs.Primary.ID)
			}
			return nil
		},
	}, client
}

// gatewayClassConfig adds a data source that looks up the managed class by its
// applied name, so Terraform reads it only after the write and the tests can
// compare both views of the same object.
func gatewayClassConfig(name, extra string) string {
	return gatewayClassResourceConfig(name, extra) + `
data "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = kubernetes_gateway_class_v1.test.metadata[0].name
  }
}
`
}

// gatewayClassResourceConfig puts controller_name under the reserved .example
// domain, so no installed controller claims the class and status keeps its CRD
// default. Tests rewrite that value with strings.Replace; extra is raw HCL
// placed inside spec.
func gatewayClassResourceConfig(name, extra string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = %q
  }
  spec {
    controller_name = %q
    %s
  }
}
`, name, "terraform-provider-kubernetes.example/"+name, extra)
}
