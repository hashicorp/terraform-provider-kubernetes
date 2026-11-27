// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestv2_test

import (
	"context"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/kubetest"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const patchName = "kubernetes_manifest_patch.test"

// othersConfigMap creates a ConfigMap as another client would, with a key of
// its own, for the length of the test.
func othersConfigMap(t *testing.T, name string) {
	t.Helper()
	conn, err := kubetest.Clientset()
	if err != nil {
		t.Fatal(err)
	}
	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: name}, Data: map[string]string{"owner": "1"}}
	if _, err := conn.CoreV1().ConfigMaps("default").Create(context.Background(), cm, metav1.CreateOptions{FieldManager: "owner"}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = conn.CoreV1().ConfigMaps("default").Delete(context.Background(), name, metav1.DeleteOptions{})
	})
}

// expectConfigMap checks the ConfigMap's data and labels, and the field
// managers that own its fields.
func expectConfigMap(name string, data, labels map[string]string, managers ...string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		conn, err := kubetest.Clientset()
		if err != nil {
			return err
		}
		cm, err := conn.CoreV1().ConfigMaps("default").Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		var owners []string
		for _, entry := range cm.ManagedFields {
			owners = append(owners, entry.Manager)
		}
		slices.Sort(owners)
		if !maps.Equal(cm.Data, data) || !maps.Equal(cm.Labels, labels) || !slices.Equal(owners, managers) {
			return fmt.Errorf("data %v, labels %v, field managers %v; want %v, %v, %v", cm.Data, cm.Labels, owners, data, labels, managers)
		}
		return nil
	}
}

func testAccPatchConfigMap(name, metadata, data, extra string) string {
	return fmt.Sprintf(`resource "kubernetes_manifest_patch" "test" {
  manifest = {
    apiVersion = "v1"
    kind       = "ConfigMap"
    metadata   = { name = %q%s }
    data       = { %s }
  }
%s}
`, name, metadata, data, extra)
}

func TestAccKubernetesManifestPatch_basic(t *testing.T) {
	name := randomName()
	force := "  field_manager {\n    force_conflicts = true\n  }\n"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t); othersConfigMap(t, name) },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		// Destroying the patch keeps the object and the values, without an owner.
		CheckDestroy: expectConfigMap(name, map[string]string{"owner": "1", "patched": "b"}, nil, "owner"),
		Steps: []resource.TestStep{
			{
				Config: testAccPatchConfigMap(name, `, labels = { team = "a" }`, `patched = "a"`, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					expectConfigMap(name, map[string]string{"owner": "1", "patched": "a"}, map[string]string{"team": "a"}, "owner", "terraform-patch"),
					resource.TestCheckResourceAttr(patchName, "namespace", "default"),
					resource.TestCheckResourceAttrSet(patchName, "uid"),
				),
			},
			{
				ResourceName:                         patchName,
				ImportState:                          true,
				ImportStateId:                        fmt.Sprintf("apiVersion=v1,kind=ConfigMap,namespace=default,name=%s", name),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uid",
			},
			// A field removed from the manifest leaves the object.
			{
				Config: testAccPatchConfigMap(name, "", `patched = "b"`, force),
				Check:  expectConfigMap(name, map[string]string{"owner": "1", "patched": "b"}, nil, "owner", "terraform-patch"),
			},
			// A field another client changed is restored.
			{
				PreConfig: func() {
					conn, _ := kubetest.Clientset()
					cm, err := conn.CoreV1().ConfigMaps("default").Get(context.Background(), name, metav1.GetOptions{})
					if err == nil {
						cm.Data["patched"] = "changed"
						_, err = conn.CoreV1().ConfigMaps("default").Update(context.Background(), cm, metav1.UpdateOptions{FieldManager: "other"})
					}
					if err != nil {
						t.Fatal(err)
					}
				},
				Config: testAccPatchConfigMap(name, "", `patched = "b"`, force),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(patchName, plancheck.ResourceActionUpdate),
				}},
				Check: expectConfigMap(name, map[string]string{"owner": "1", "patched": "b"}, nil, "owner", "terraform-patch"),
			},
			{
				Config: testAccPatchConfigMap(name, "", `patched = "b"`, force),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(patchName, plancheck.ResourceActionNoop),
				}},
			},
		},
	})
}

func TestAccKubernetesManifestPatch_removeFields(t *testing.T) {
	name := randomName()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t); othersConfigMap(t, name) },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		CheckDestroy:             expectConfigMap(name, map[string]string{"owner": "1"}, nil, "owner"),
		Steps: []resource.TestStep{{
			Config: testAccPatchConfigMap(name, "", `patched = "a"`, "  destroy_behavior = \"remove_fields\"\n"),
			Check:  expectConfigMap(name, map[string]string{"owner": "1", "patched": "a"}, nil, "owner", "terraform-patch"),
		}},
	})
}

func TestAccKubernetesManifestPatch_errors(t *testing.T) {
	name := randomName()
	// A second patch of the object, under the default field manager name or
	// its own. Nothing orders the two: they are applied in parallel.
	other := func(manager string) string {
		return testAccPatchConfigMap(name, "", `a = "1"`, "") + fmt.Sprintf(`
resource "kubernetes_manifest_patch" "other" {
  manifest = {
    apiVersion = "v1"
    kind       = "ConfigMap"
    metadata   = { name = %q }
    data       = { b = "1" }
  }
%s}
`, name, manager)
	}
	own := "  field_manager {\n    name = \"second\"\n  }\n"
	both := map[string]string{"owner": "1", "a": "1", "b": "1"}
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t); othersConfigMap(t, name) },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccPatchConfigMap(name+"-missing", "", `a = "1"`, ""),
				ExpectError: regexp.MustCompile(`Object not found`),
			},
			// Two patches of one object under one field manager name would
			// remove each other's fields: whichever applies second is refused.
			{
				Config:      other(""),
				ExpectError: regexp.MustCompile(`Field manager in use(.|\n)*\.data\.[ab]`),
			},
			{
				Config: other(own),
				Check:  expectConfigMap(name, both, nil, "owner", "second", "terraform-patch"),
			},
			// Renaming onto a name in use is refused before any field moves.
			{
				Config:      other(""),
				ExpectError: regexp.MustCompile(`Field manager in use(.|\n)*\.data\.a`),
			},
			{
				Config: other(own),
				Check:  expectConfigMap(name, both, nil, "owner", "second", "terraform-patch"),
			},
		},
	})
}

// A new patch waits for an object that arrives a moment later, as the default
// ServiceAccount of a new namespace does.
func TestAccKubernetesManifestPatch_waitsForObject(t *testing.T) {
	name := randomName()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t) },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		Steps: []resource.TestStep{{
			PreConfig: func() {
				conn, err := kubetest.Clientset()
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					_ = conn.CoreV1().ConfigMaps("default").Delete(context.Background(), name, metav1.DeleteOptions{})
				})
				go func() {
					time.Sleep(4 * time.Second)
					cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: name}, Data: map[string]string{"owner": "1"}}
					if _, err := conn.CoreV1().ConfigMaps("default").Create(context.Background(), cm, metav1.CreateOptions{FieldManager: "owner"}); err != nil {
						t.Error(err)
					}
				}()
			},
			Config: testAccPatchConfigMap(name, "", `patched = "a"`, ""),
			Check:  expectConfigMap(name, map[string]string{"owner": "1", "patched": "a"}, nil, "owner", "terraform-patch"),
		}},
	})
}

// The environment of another client's container gains a variable, as a
// patch of an add-on's DaemonSet adds one.
func TestAccKubernetesManifestPatch_containerEnv(t *testing.T) {
	name := randomName()
	expectEnv := func(want ...string) resource.TestCheckFunc {
		return func(*terraform.State) error {
			conn, _ := kubetest.Clientset()
			d, err := conn.AppsV1().Deployments("default").Get(context.Background(), name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			var env []string
			for _, v := range d.Spec.Template.Spec.Containers[0].Env {
				env = append(env, v.Name+"="+v.Value)
			}
			if !slices.Equal(env, want) {
				return fmt.Errorf("env %v, want %v", env, want)
			}
			return nil
		}
	}
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			kubetest.PreCheck(t)
			conn, err := kubetest.Clientset()
			if err != nil {
				t.Fatal(err)
			}
			labels := map[string]string{"app": name}
			d := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: name},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{MatchLabels: labels},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{Labels: labels},
						Spec: corev1.PodSpec{Containers: []corev1.Container{{
							Name: "main", Image: kubetest.BusyboxImage, Command: []string{"sleep", "3600"},
							Env: []corev1.EnvVar{{Name: "OWNER", Value: "1"}},
						}}},
					},
				},
			}
			if _, err := conn.AppsV1().Deployments("default").Create(context.Background(), d, metav1.CreateOptions{FieldManager: "owner"}); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				_ = conn.AppsV1().Deployments("default").Delete(context.Background(), name, metav1.DeleteOptions{})
			})
		},
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		CheckDestroy:             expectEnv("OWNER=1", "PATCHED=1"),
		Steps: []resource.TestStep{{
			Config: fmt.Sprintf(`resource "kubernetes_manifest_patch" "test" {
  manifest = {
    apiVersion = "apps/v1"
    kind       = "Deployment"
    metadata   = { name = %q }
    spec = { template = { spec = { containers = [{
      name = "main"
      env  = [{ name = "PATCHED", value = "1" }]
    }] } } }
  }
  wait {
    rollout = true
  }
}
`, name),
			Check: expectEnv("OWNER=1", "PATCHED=1"),
		}},
	})
}
