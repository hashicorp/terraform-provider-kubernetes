// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestv2_test

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/kubetest"
	corev1 "k8s.io/api/core/v1"
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

// An object that kubectl applied client-side is imported. The first apply
// takes over the fields the configuration sets, changes them without a
// conflict, and leaves the other fields in place.
func TestAccKubernetesManifestV2_importClientSide(t *testing.T) {
	name := randomName()
	ctx := context.Background()
	config := func(value string) string {
		return fmt.Sprintf(`import {
  to = kubernetes_manifest_v2.test
  id = "apiVersion=v1,kind=ConfigMap,namespace=default,name=%s"
}
%s`, name, testAccConfigMap(name, fmt.Sprintf("one = %q", value)))
	}
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t) },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_5_0)},
		CheckDestroy:             checkConfigMapDestroyed(name),
		Steps: []resource.TestStep{{
			PreConfig: func() {
				conn, _ := kubetest.Clientset()
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default", Labels: map[string]string{"team": "a"}},
					Data:       map[string]string{"one": "1", "two": "2"},
				}
				if _, err := conn.CoreV1().ConfigMaps("default").Create(ctx, cm, metav1.CreateOptions{FieldManager: "kubectl-client-side-apply"}); err != nil {
					t.Fatal(err)
				}
			},
			Config: config("changed"),
			Check: func(*terraform.State) error {
				conn, _ := kubetest.Clientset()
				cm, err := conn.CoreV1().ConfigMaps("default").Get(ctx, name, metav1.GetOptions{})
				if err != nil || cm.Data["one"] != "changed" || cm.Data["two"] != "2" || cm.Labels["team"] != "a" {
					return fmt.Errorf("unexpected ConfigMap: data %v, labels %v (error: %v)", cm.Data, cm.Labels, err)
				}
				return nil
			},
		}},
	})
}

// Renaming the field manager moves the fields to the new name, also when a
// value changes in the same apply, and leaves another manager's fields alone.
func TestAccKubernetesManifestV2_renameFieldManager(t *testing.T) {
	name := randomName()
	ctx := context.Background()
	config := func(manager, value string) string {
		return strings.Replace(testAccConfigMap(name, fmt.Sprintf("one = %q", value)), "manifest = {",
			fmt.Sprintf("field_manager {\n    name = %q\n  }\n  manifest = {", manager), 1)
	}
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t) },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		CheckDestroy:             checkConfigMapDestroyed(name),
		Steps: []resource.TestStep{
			{Config: testAccConfigMap(name, `one = "1"`)},
			{
				PreConfig: func() {
					conn, _ := kubetest.Clientset()
					patch := []byte(fmt.Sprintf(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":%q},"data":{"other":"x"}}`, name))
					if _, err := conn.CoreV1().ConfigMaps("default").Patch(ctx, name, types.ApplyPatchType, patch, metav1.PatchOptions{FieldManager: "tf-acc-other"}); err != nil {
						t.Fatal(err)
					}
				},
				Config: config("tf-acc-renamed", "2"),
				Check: func(*terraform.State) error {
					conn, _ := kubetest.Clientset()
					cm, err := conn.CoreV1().ConfigMaps("default").Get(ctx, name, metav1.GetOptions{})
					if err != nil || cm.Data["one"] != "2" || cm.Data["other"] != "x" {
						return fmt.Errorf("unexpected data %v (error: %v)", cm.Data, err)
					}
					for _, entry := range cm.ManagedFields {
						if entry.Manager == "Terraform" {
							return fmt.Errorf("the previous field manager still owns fields: %s", entry.FieldsV1.GetRawString())
						}
					}
					return nil
				},
			},
			{Config: testAccConfigMap(name, `one = "2"`), ConfigPlanChecks: expectAction(plancheck.ResourceActionUpdate)},
		},
	})
}

// HCL numbers and bools are applied as strings where the kind has strings.
func TestAccKubernetesManifestV2_stringValues(t *testing.T) {
	name := randomName()
	config := strings.Replace(testAccConfigMap(name, `PORT = 8080, DEBUG = true`), "}\n    data", ", labels = { version = 2 } }\n    data", 1)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t) },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		CheckDestroy:             checkConfigMapDestroyed(name),
		Steps: []resource.TestStep{
			{
				Config: config,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("object").AtMapKey("data").AtMapKey("PORT"), knownvalue.StringExact("8080")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("object").AtMapKey("metadata").AtMapKey("labels").AtMapKey("version"), knownvalue.StringExact("2")),
				},
			},
			{Config: config, ConfigPlanChecks: expectAction(plancheck.ResourceActionNoop)},
		},
	})
}

// A Job that deletes itself when it completes still satisfies the wait: the
// watch sees its last state. Its next plan creates it again.
func TestAccKubernetesManifestV2_selfDeletingJob(t *testing.T) {
	name := randomName()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t) },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		Steps: []resource.TestStep{{
			Config: fmt.Sprintf(`resource "kubernetes_manifest_v2" "test" {
  manifest = {
    apiVersion = "batch/v1"
    kind       = "Job"
    metadata   = { name = %q }
    spec = {
      ttlSecondsAfterFinished = 0
      template = { spec = {
        restartPolicy = "Never"
        containers    = [{ name = "main", image = %q, command = ["true"] }]
      } }
    }
  }
  wait {
    condition { type = "Complete" }
  }
}
`, name, kubetest.BusyboxImage),
			ExpectNonEmptyPlan: true,
		}},
	})
}

// An update whose rollout never completes fails. The next plan applies again,
// also when the configuration goes back to the image that worked.
func TestAccKubernetesManifestV2_failedUpdateWait(t *testing.T) {
	name := randomName()
	ctx := context.Background()
	const wait = "  wait {\n    rollout = true\n  }\n  timeouts {\n    update = \"20s\"\n  }"
	broken := strings.Replace(testAccDeployment(name, wait), kubetest.BusyboxImage, "registry.invalid/busybox:none", 1)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t) },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_8_0)},
		Steps: []resource.TestStep{
			{Config: testAccDeployment(name, wait)},
			{Config: broken, ExpectError: regexp.MustCompile(`Object applied but not ready(.|\n)*timed out waiting for(.|\n)*rollout`)},
			{
				Config:           testAccDeployment(name, wait),
				ConfigPlanChecks: expectAction(plancheck.ResourceActionUpdate),
				Check: func(*terraform.State) error {
					conn, _ := kubetest.Clientset()
					d, err := conn.AppsV1().Deployments("default").Get(ctx, name, metav1.GetOptions{})
					if err != nil || d.Spec.Template.Spec.Containers[0].Image != kubetest.BusyboxImage {
						return fmt.Errorf("the image was not rolled back: %v (error: %v)", d.Spec.Template.Spec.Containers, err)
					}
					return nil
				},
			},
			{Config: testAccDeployment(name, wait), ConfigPlanChecks: expectAction(plancheck.ResourceActionNoop)},
		},
	})
}

// ready waits by the rules of the kind; Orphan leaves the ReplicaSet when the
// Deployment is destroyed.
func TestAccKubernetesManifestV2_readyAndOrphan(t *testing.T) {
	name := randomName()
	ctx := context.Background()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t) },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_8_0)},
		CheckDestroy: func(*terraform.State) error {
			conn, _ := kubetest.Clientset()
			if _, err := conn.AppsV1().Deployments("default").Get(ctx, name, metav1.GetOptions{}); !apierrors.IsNotFound(err) {
				return fmt.Errorf("Deployment %s still exists (error: %v)", name, err)
			}
			sets, err := conn.AppsV1().ReplicaSets("default").List(ctx, metav1.ListOptions{LabelSelector: "app=" + name})
			if err != nil || len(sets.Items) != 1 {
				return fmt.Errorf("want the orphaned ReplicaSet, got %v (error: %v)", sets, err)
			}
			return conn.AppsV1().ReplicaSets("default").Delete(ctx, sets.Items[0].Name, metav1.DeleteOptions{})
		},
		Steps: []resource.TestStep{{
			Config: testAccDeployment(name, "  deletion_propagation = \"Orphan\"\n  wait {\n    ready = true\n  }"),
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("object").AtMapKey("status").AtMapKey("readyReplicas"), knownvalue.Int64Exact(1)),
			},
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
	config := func(name string, port int, extra ...string) string {
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
%[3]s
}
`, name, port, strings.Join(extra, "\n"))
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
				Check:              resource.TestCheckResourceAttr(resourceName, "drift.0", `metadata.labels["app.kubernetes.io/name"]`),
				ExpectNonEmptyPlan: true,
			},
			// lifecycle.ignore_changes leaves the change in place.
			{
				Config:           config(name, 80, "  lifecycle {\n    ignore_changes = [yaml_body]\n  }"),
				ConfigPlanChecks: expectAction(plancheck.ResourceActionNoop),
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

// A move from kubectl_manifest keeps, in private state, the field manager the
// object was applied with, so that the first update takes over its fields
// without a conflict. Providers without a field_manager argument apply as
// kubectl.
func TestManifestV2MoveFromKubectlManifest(t *testing.T) {
	ctx := context.Background()
	server, err := kubetest.ProviderFactories["kubernetes"]()
	if err != nil {
		t.Fatal(err)
	}
	const body = `"yaml_body":"apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: c\n  namespace: prod\n","namespace":"prod","uid":"u-1"`
	for name, tc := range map[string]struct{ state, manager string }{
		"server-side apply without field_manager": {`{` + body + `,"server_side_apply":true,"force_conflicts":false}`, "kubectl"},
		"server-side apply with field_manager":    {`{` + body + `,"server_side_apply":true,"field_manager":"tf-kubectl"}`, "tf-kubectl"},
	} {
		resp, err := server.MoveResourceState(ctx, &tfprotov6.MoveResourceStateRequest{
			SourceProviderAddress: "registry.example.com/acme/kubectl", SourceTypeName: "kubectl_manifest", SourceSchemaVersion: 1,
			SourceState: &tfprotov6.RawState{JSON: []byte(tc.state)}, TargetTypeName: "kubernetes_manifest_v2",
		})
		if err != nil || len(resp.Diagnostics) > 0 {
			t.Fatalf("%s: %v %v", name, err, resp.Diagnostics)
		}
		var private map[string][]byte
		var manager string
		if err := json.Unmarshal(resp.TargetPrivate, &private); err != nil || json.Unmarshal(private["manager"], &manager) != nil || manager != tc.manager {
			t.Errorf("%s: previous field manager %q (private state %s), want %q", name, manager, resp.TargetPrivate, tc.manager)
		}
	}
}
