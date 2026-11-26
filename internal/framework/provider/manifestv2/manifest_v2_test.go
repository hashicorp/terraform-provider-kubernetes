// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestv2_test

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/kubetest"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const resourceName = "kubernetes_manifest_v2.test"

func randomName() string {
	return fmt.Sprintf("tf-acc-test-%s", strings.ToLower(acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)))
}

func expectAction(action plancheck.ResourceActionType) resource.ConfigPlanChecks {
	return resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, action)}}
}

func checkConfigMapDestroyed(name string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		conn, err := kubetest.Clientset()
		if err != nil {
			return err
		}
		if _, err = conn.CoreV1().ConfigMaps("default").Get(context.Background(), name, metav1.GetOptions{}); apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("ConfigMap %s still exists (error: %v)", name, err)
	}
}

func testAccConfigMap(name, data string) string {
	return fmt.Sprintf(`resource "kubernetes_manifest_v2" "test" {
  manifest = {
    apiVersion = "v1"
    kind       = "ConfigMap"
    metadata   = { name = %q }
    data       = { %s }
  }
}
`, name, data)
}

func TestAccKubernetesManifestV2_basic(t *testing.T) {
	name := randomName()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t) },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		CheckDestroy:             checkConfigMapDestroyed(name),
		Steps: []resource.TestStep{
			{
				Config: testAccConfigMap(name, `one = "1", two = "2"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "kind", "ConfigMap"),
					resource.TestCheckResourceAttr(resourceName, "namespace", "default"),
					resource.TestCheckResourceAttrSet(resourceName, "uid"),
				),
				// object holds what the cluster added, such as the namespace.
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("object").AtMapKey("metadata").AtMapKey("namespace"), knownvalue.StringExact("default")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("object").AtMapKey("data").AtMapKey("two"), knownvalue.StringExact("2")),
				},
			},
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateId:                        fmt.Sprintf("apiVersion=v1,kind=ConfigMap,namespace=default,name=%s", name),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uid",
				// The imported manifest names the namespace the object is in.
				ImportStateVerifyIgnore: []string{"manifest"},
			},
			// A removed key leaves the object: the field manager owned it.
			// References to object stay known while the update is planned.
			{
				Config: testAccConfigMap(name, `one = "changed"`),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					plancheck.ExpectKnownValue(resourceName, tfjsonpath.New("object").AtMapKey("metadata").AtMapKey("namespace"), knownvalue.StringExact("default")),
				}},
				Check: func(*terraform.State) error {
					conn, _ := kubetest.Clientset()
					cm, err := conn.CoreV1().ConfigMaps("default").Get(context.Background(), name, metav1.GetOptions{})
					if err != nil || len(cm.Data) != 1 || cm.Data["one"] != "changed" {
						return fmt.Errorf("unexpected data %v (error: %v)", cm.Data, err)
					}
					return nil
				},
			},
			{Config: testAccConfigMap(name, `one = "changed"`), ConfigPlanChecks: expectAction(plancheck.ResourceActionNoop)},
			{Config: testAccConfigMap(name+"-renamed", `one = "changed"`), ConfigPlanChecks: expectAction(plancheck.ResourceActionDestroyBeforeCreate)},
		},
	})
}

func TestAccKubernetesManifestV2_invalidComputedField(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t) },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      strings.Replace(testAccConfigMap("unused", `one = "1"`), "manifest = {", "computed_fields = [\"data[\"]\n  manifest = {", 1),
			ExpectError: regexp.MustCompile(`"data\[" is not a field path`),
		}},
	})
}

func TestAccKubernetesManifestV2_alreadyExists(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t) },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `resource "kubernetes_manifest_v2" "test" {
  manifest = { apiVersion = "v1", kind = "Namespace", metadata = { name = "kube-system" } }
}
`,
			ExpectError: regexp.MustCompile(`already exists in the cluster(.|\n)*apiVersion=v1,kind=Namespace,name=kube-system`),
		}},
	})
}

func testAccDeployment(name, extra string) string {
	return fmt.Sprintf(`resource "kubernetes_manifest_v2" "test" {
  manifest = provider::kubernetes::manifest_decode(<<-YAML
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: %[1]s
      namespace: default
      labels: {app: %[1]s, tier: front}
    spec:
      replicas: 1
      selector: {matchLabels: {app: %[1]s}}
      template:
        metadata: {labels: {app: %[1]s}}
        spec:
          terminationGracePeriodSeconds: 1
          containers:
          - name: main
            image: %[2]s
            command: ["sleep", "infinity"]
            resources: {requests: {cpu: 0.01, memory: 8192Ki}}
    YAML
  )
%[3]s
}
`, name, kubetest.BusyboxImage, extra)
}

// Another client scales the Deployment, removes a label and adds an
// annotation. Terraform reports the first two, which it owns, and restores
// them; the annotation is the other client's.
func TestAccKubernetesManifestV2_drift(t *testing.T) {
	name := randomName()
	ctx := context.Background()
	const forced = "  field_manager {\n    force_conflicts = true\n  }"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t) },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_8_0)},
		Steps: []resource.TestStep{
			{
				Config: testAccDeployment(name, `  wait {
    rollout = true
    fields  = { "status.readyReplicas" = "1" }
    condition {
      type   = "Available"
      status = "True"
    }
  }`),
				Check: resource.TestCheckResourceAttr(resourceName, "name", name),
				// The object is read again after the wait.
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("object").AtMapKey("status").AtMapKey("readyReplicas"), knownvalue.Int64Exact(1)),
				},
			},
			// The server's spelling of the quantities is not a change.
			{Config: testAccDeployment(name, ""), ConfigPlanChecks: expectAction(plancheck.ResourceActionUpdate)},
			{Config: testAccDeployment(name, ""), ConfigPlanChecks: expectAction(plancheck.ResourceActionNoop)},
			{
				PreConfig: func() {
					conn, _ := kubetest.Clientset()
					patch := []byte(`{"metadata":{"labels":{"tier":null},"annotations":{"other":"yes"}},"spec":{"replicas":2}}`)
					if _, err := conn.AppsV1().Deployments("default").Patch(ctx, name, types.MergePatchType, patch, metav1.PatchOptions{FieldManager: "tf-acc-other"}); err != nil {
						t.Fatal(err)
					}
				},
				Config:      testAccDeployment(name, ""),
				ExpectError: regexp.MustCompile(`Field ownership conflict(.|\n)*tf-acc-other(.|\n)*\.spec\.replicas`),
			},
			{
				Config:           testAccDeployment(name, forced),
				ConfigPlanChecks: expectAction(plancheck.ResourceActionUpdate),
				Check: func(*terraform.State) error {
					conn, _ := kubetest.Clientset()
					d, err := conn.AppsV1().Deployments("default").Get(ctx, name, metav1.GetOptions{})
					if err != nil || *d.Spec.Replicas != 1 || d.Labels["tier"] != "front" || d.Annotations["other"] != "yes" {
						return fmt.Errorf("unexpected Deployment: replicas %v, labels %v, annotations %v (error: %v)", d.Spec.Replicas, d.Labels, d.Annotations, err)
					}
					return nil
				},
			},
			{Config: testAccDeployment(name, forced), ConfigPlanChecks: expectAction(plancheck.ResourceActionNoop)},
			// A computed field is left to the client that changes it.
			{Config: testAccDeployment(name, forced+"\n  computed_fields = [\"spec.replicas\"]")},
			{
				PreConfig: func() {
					conn, _ := kubetest.Clientset()
					if _, err := conn.AppsV1().Deployments("default").Patch(ctx, name, types.MergePatchType, []byte(`{"spec":{"replicas":2}}`), metav1.PatchOptions{FieldManager: "tf-acc-other"}); err != nil {
						t.Fatal(err)
					}
				},
				Config:           testAccDeployment(name, forced+"\n  computed_fields = [\"spec.replicas\"]"),
				ConfigPlanChecks: expectAction(plancheck.ResourceActionNoop),
			},
		},
	})
}

// A CustomResourceDefinition and a custom resource of it are planned before
// either exists and created in one apply, in no particular order. The test
// creates no Namespace, which would disturb the tests of the all-namespaces
// data source, and uses no for_each, which the test framework does not support.
func TestAccKubernetesManifestV2_bundle(t *testing.T) {
	name := randomName()
	group := strings.ToLower(acctest.RandStringFromCharSet(8, acctest.CharSetAlpha)) + ".example.com"
	config := fmt.Sprintf(`locals {
  objects = provider::kubernetes::manifest_decode_map(<<-YAML
    apiVersion: %[2]s/v1
    kind: Widget
    metadata:
      name: %[1]s
    spec:
      size: 3
    ---
    apiVersion: apiextensions.k8s.io/v1
    kind: CustomResourceDefinition
    metadata:
      name: widgets.%[2]s
    spec:
      group: %[2]s
      names: {kind: Widget, plural: widgets}
      scope: Namespaced
      versions:
      - name: v1
        served: true
        storage: true
        schema:
          openAPIV3Schema:
            type: object
            properties:
              spec:
                type: object
                properties:
                  size: {type: integer}
    YAML
  )
}
resource "kubernetes_manifest_v2" "test" {
  manifest = local.objects["Widget.%[2]s/%[1]s"]
}
resource "kubernetes_manifest_v2" "definition" {
  manifest = local.objects["CustomResourceDefinition.apiextensions.k8s.io/widgets.%[2]s"]
}
`, name, group)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t) },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_8_0)},
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "namespace", "default"),
					resource.TestCheckResourceAttr("kubernetes_manifest_v2.definition", "namespace", ""),
					resource.TestCheckResourceAttrSet("kubernetes_manifest_v2.definition", "uid"),
				),
			},
			{Config: config, PlanOnly: true},
		},
	})
}

// kubernetes_manifest applies with the same field manager and takes the same
// arguments, so the moved object plans no change.
func TestAccKubernetesManifestV2_movedFromManifest(t *testing.T) {
	name := randomName()
	manifest := func(resourceType, value string) string {
		return fmt.Sprintf(`resource %q "test" {
  manifest = {
    apiVersion = "v1"
    kind       = "ConfigMap"
    metadata   = { name = %q, namespace = "default" }
    data       = { one = %q }
  }
  computed_fields = ["metadata.annotations"]
  field_manager {
    force_conflicts = true
  }
  wait {
    fields = { "metadata.name" = %q }
  }
  timeouts {
    create = "2m"
  }
}
`, resourceType, name, value, name)
	}
	moved := "moved {\n  from = kubernetes_manifest.test\n  to   = kubernetes_manifest_v2.test\n}\n"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t) },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_8_0)},
		CheckDestroy:             checkConfigMapDestroyed(name),
		Steps: []resource.TestStep{
			{Config: manifest("kubernetes_manifest", "1")},
			{Config: moved + manifest("kubernetes_manifest_v2", "1"), ConfigPlanChecks: expectAction(plancheck.ResourceActionNoop)},
			{
				Config:           moved + manifest("kubernetes_manifest_v2", "2"),
				ConfigPlanChecks: expectAction(plancheck.ResourceActionUpdate),
				Check:            resource.TestCheckResourceAttr(resourceName, "namespace", "default"),
			},
		},
	})
}

// The manifest as text. A change by another client is listed in drift, the
// plan restores it, and the list is empty again.
func TestAccKubernetesManifestV2_yamlBody(t *testing.T) {
	name := randomName()
	ctx := context.Background()
	config := func(name string, port int) string {
		return fmt.Sprintf(`resource "kubernetes_manifest_v2" "test" {
  yaml_body = <<-YAML
    apiVersion: v1
    kind: Service
    metadata:
      name: %[1]s
      namespace: default
      annotations:
      labels:
        app.kubernetes.io/name: %[1]s
    spec:
      type: ClusterIP
      selector:
        app.kubernetes.io/name: %[1]s
      ports:
      - name: grpc
        port: %[2]d
        targetPort: 4245
    YAML
  field_manager {
    force_conflicts = true
  }
}
`, name, port)
	}
	drift := func(paths ...knownvalue.Check) statecheck.StateCheck {
		return statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("drift"), knownvalue.ListExact(paths))
	}
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t) },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(name, 80),
				Check:  resource.TestCheckResourceAttr(resourceName, "kind", "Service"),
				ConfigStateChecks: []statecheck.StateCheck{
					drift(),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("object").AtMapKey("spec").AtMapKey("clusterIP"), knownvalue.NotNull()),
				},
			},
			{Config: config(name, 80), ConfigPlanChecks: expectAction(plancheck.ResourceActionNoop)},
			{
				PreConfig: func() {
					conn, _ := kubetest.Clientset()
					patch := []byte(`{"metadata":{"labels":{"app.kubernetes.io/name":"other"}}}`)
					if _, err := conn.CoreV1().Services("default").Patch(ctx, name, types.MergePatchType, patch, metav1.PatchOptions{FieldManager: "tf-acc-other"}); err != nil {
						t.Fatal(err)
					}
				},
				RefreshState:       true,
				Check:              resource.TestCheckResourceAttr(resourceName, "drift.0", "metadata.labels.app.kubernetes.io/name"),
				ExpectNonEmptyPlan: true,
			},
			{
				Config:            config(name, 80),
				ConfigPlanChecks:  expectAction(plancheck.ResourceActionUpdate),
				ConfigStateChecks: []statecheck.StateCheck{drift()},
				Check: func(*terraform.State) error {
					conn, _ := kubetest.Clientset()
					service, err := conn.CoreV1().Services("default").Get(ctx, name, metav1.GetOptions{})
					if err != nil || service.Labels["app.kubernetes.io/name"] != name {
						return fmt.Errorf("label not restored: %v (error: %v)", service.Labels, err)
					}
					return nil
				},
			},
			{Config: config(name, 81), ConfigPlanChecks: expectAction(plancheck.ResourceActionUpdate)},
			// object shows what the update changed once it is refreshed.
			{
				Config:           config(name, 81),
				ConfigPlanChecks: expectAction(plancheck.ResourceActionNoop),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("object").AtMapKey("spec").AtMapKey("ports").AtSliceIndex(0).AtMapKey("port"), knownvalue.Int64Exact(81)),
				},
			},
			{Config: config(name+"-renamed", 81), ConfigPlanChecks: expectAction(plancheck.ResourceActionDestroyBeforeCreate)},
		},
	})
}
