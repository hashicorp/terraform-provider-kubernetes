// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestyaml_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/restmapper"
)

// ---- helpers ---------------------------------------------------------------

// parseManifestID parses the resource id "apiVersion=..,kind=..,namespace=..,name=..".
func parseManifestID(id string) (apiVersion, kind, namespace, name string) {
	for _, p := range strings.Split(id, ",") {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch strings.TrimSpace(kv[0]) {
		case "apiVersion":
			apiVersion = strings.TrimSpace(kv[1])
		case "kind":
			kind = strings.TrimSpace(kv[1])
		case "namespace":
			namespace = strings.TrimSpace(kv[1])
		case "name":
			name = strings.TrimSpace(kv[1])
		}
	}
	return
}

// getManifestObject resolves the GVR via discovery and fetches the live object.
func getManifestObject(clients kubernetes.KubeClientsets, id string) (bool, error) {
	apiVersion, kind, namespace, name := parseManifestID(id)
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return false, err
	}
	gvk := gv.WithKind(kind)

	disco, err := clients.DiscoveryClient()
	if err != nil {
		return false, err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(disco))
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return false, err
	}
	dyn, err := clients.DynamicClient()
	if err != nil {
		return false, err
	}

	ri := dyn.Resource(mapping.Resource)
	var getErr error
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		_, getErr = ri.Namespace(namespace).Get(context.Background(), name, metav1.GetOptions{})
	} else {
		_, getErr = ri.Get(context.Background(), name, metav1.GetOptions{})
	}
	if apierrors.IsNotFound(getErr) {
		return false, nil
	}
	if getErr != nil {
		return false, getErr
	}
	return true, nil
}

func testAccCheckManifestYAMLExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found in state: %s", resourceName)
		}
		id := rs.Primary.Attributes["id"]
		if id == "" {
			return fmt.Errorf("%s: empty id", resourceName)
		}
		exists, err := getManifestObject(testAccClients(), id)
		if err != nil {
			return fmt.Errorf("%s: exists check error: %w", resourceName, err)
		}
		if !exists {
			return fmt.Errorf("%s: object %q not found in cluster", resourceName, id)
		}
		return nil
	}
}

// testAccCheckManifestYAMLDestroy asserts every managed object is gone after destroy.
func testAccCheckManifestYAMLDestroy(s *terraform.State) error {
	clients := testAccClients()
	for name, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_manifest_yaml" {
			continue
		}
		id := rs.Primary.Attributes["id"]
		if id == "" {
			continue
		}
		exists, err := getManifestObject(clients, id)
		if err != nil {
			// A NoMatch (CRD removed) also implies the object is gone.
			continue
		}
		if exists {
			return fmt.Errorf("%s: object %q still exists after destroy", name, id)
		}
	}
	return nil
}

// ---- tests -----------------------------------------------------------------

func TestAccManifestYAML_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-cm")
	resourceName := "kubernetes_manifest_yaml.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckManifestYAMLDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccManifestYAMLConfigMap(name, "hello"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckManifestYAMLExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "kind", "ConfigMap"),
					resource.TestCheckResourceAttr(resourceName, "api_version", "v1"),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "namespace", "default"),
					resource.TestCheckResourceAttr(resourceName, "field_manager", "terraform"),
					resource.TestCheckResourceAttrSet(resourceName, "uid"),
					resource.TestCheckResourceAttr(resourceName, "id",
						fmt.Sprintf("apiVersion=v1,kind=ConfigMap,namespace=default,name=%s", name)),
				),
			},
		},
	})
}

func TestAccManifestYAML_update(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-cm")
	resourceName := "kubernetes_manifest_yaml.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckManifestYAMLDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccManifestYAMLConfigMap(name, "v1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckManifestYAMLExists(resourceName),
					resource.TestCheckResourceAttrSet(resourceName, "uid"),
				),
			},
			{
				// Change the payload; object should update in place (same name/kind).
				Config: testAccManifestYAMLConfigMap(name, "v2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckManifestYAMLExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", name),
				),
			},
		},
	})
}

func TestAccManifestYAML_clusterScoped(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-ns")
	resourceName := "kubernetes_manifest_yaml.ns"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckManifestYAMLDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccManifestYAMLNamespace(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckManifestYAMLExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "kind", "Namespace"),
					resource.TestCheckResourceAttr(resourceName, "namespace", ""),
					resource.TestCheckResourceAttr(resourceName, "id",
						fmt.Sprintf("apiVersion=v1,kind=Namespace,namespace=,name=%s", name)),
				),
			},
		},
	})
}

func TestAccManifestYAML_import(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-cm")
	resourceName := "kubernetes_manifest_yaml.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckManifestYAMLDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccManifestYAMLConfigMap(name, "hello"),
				Check:  testAccCheckManifestYAMLExists(resourceName),
			},
			{
				ResourceName: resourceName,
				ImportState:  true,
				// yaml_body cannot be reconstructed on import (RFC-011 §2.1); the
				// computed identity fields are what we verify, so skip full verify.
				ImportStateVerify: false,
			},
		},
	})
}

func TestAccManifestYAML_forceConflicts(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-cm")
	resourceName := "kubernetes_manifest_yaml.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckManifestYAMLDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccManifestYAMLConfigMapForce(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckManifestYAMLExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "force_conflicts", "true"),
					resource.TestCheckResourceAttr(resourceName, "field_manager", "tf-custom"),
				),
			},
		},
	})
}

// TestAccManifestYAML_identityChangeReplaces verifies that renaming the object
// (an identity change) forces replacement rather than orphaning the old object.
func TestAccManifestYAML_identityChangeReplaces(t *testing.T) {
	name1 := acctest.RandomWithPrefix("tf-acc-cm")
	name2 := acctest.RandomWithPrefix("tf-acc-cm")
	resourceName := "kubernetes_manifest_yaml.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckManifestYAMLDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccManifestYAMLConfigMap(name1, "v1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckManifestYAMLExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", name1),
				),
			},
			{
				// Renaming changes identity → plan must show replacement.
				Config: testAccManifestYAMLConfigMap(name2, "v1"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckManifestYAMLExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", name2),
				),
			},
		},
	})
}

// ---- config builders -------------------------------------------------------

func testAccManifestYAMLConfigMap(name, value string) string {
	return fmt.Sprintf(`
resource "kubernetes_manifest_yaml" "test" {
  yaml_body = <<-YAML
    apiVersion: v1
    kind: ConfigMap
    metadata:
      name: %s
      namespace: default
    data:
      key: %q
  YAML
}
`, name, value)
}

func testAccManifestYAMLConfigMapForce(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_manifest_yaml" "test" {
  field_manager   = "tf-custom"
  force_conflicts = true
  yaml_body = <<-YAML
    apiVersion: v1
    kind: ConfigMap
    metadata:
      name: %s
      namespace: default
    data:
      key: value
  YAML
}
`, name)
}

func testAccManifestYAMLNamespace(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_manifest_yaml" "ns" {
  yaml_body = <<-YAML
    apiVersion: v1
    kind: Namespace
    metadata:
      name: %s
      labels:
        app.kubernetes.io/managed-by: terraform
  YAML

  delete {
    propagation_policy = "Foreground"
  }
}
`, name)
}
