// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestv2_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/kubetest"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestAccKubernetesManifestPatch_overlappingManagers(t *testing.T) {
	name := randomName()
	first := testAccPatchConfigMap(name, "", `mode = "one"`, "")
	// Ordering makes this a collision on a single owned field, not a race
	// between two patches of disjoint fields.
	second := `
resource "kubernetes_manifest_patch" "second" {
  manifest = {
    apiVersion = "v1"
    kind       = "ConfigMap"
    metadata   = { name = kubernetes_manifest_patch.test.name }
    data       = { mode = "two" }
  }
}`
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t); othersConfigMap(t, name) },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		Steps: []resource.TestStep{
			{Config: first},
			{Config: first + second, ExpectError: regexp.MustCompile(`Field\s+manager\s+in\s+use`)},
			{Config: first, Check: expectConfigMap(name, map[string]string{"owner": "1", "mode": "one"}, nil, "owner", "terraform-patch")},
		},
	})
}

func TestAccKubernetesManifestPatch_recreatedWithSameValue(t *testing.T) {
	name := randomName()
	config := testAccPatchConfigMap(name, "", `mode = "one"`, "")
	var replacedUID types.UID
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t); othersConfigMap(t, name) },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config},
			{
				Config: config,
				PreConfig: func() {
					conn, err := kubetest.Clientset()
					if err != nil {
						t.Fatal(err)
					}
					ctx := context.Background()
					cms := conn.CoreV1().ConfigMaps("default")
					old, err := cms.Get(ctx, name, metav1.GetOptions{})
					if err != nil {
						t.Fatal(err)
					}
					if err := cms.Delete(ctx, name, metav1.DeleteOptions{Preconditions: &metav1.Preconditions{UID: &old.UID}}); err != nil {
						t.Fatal(err)
					}
					cm, err := cms.Create(ctx, &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{Name: name},
						Data:       map[string]string{"owner": "replacement", "mode": "one"},
					}, metav1.CreateOptions{FieldManager: "replacement-controller"})
					if err != nil {
						t.Fatal(err)
					}
					replacedUID = cm.UID
				},
				Check: func(*terraform.State) error {
					conn, err := kubetest.Clientset()
					if err != nil {
						return err
					}
					cm, err := conn.CoreV1().ConfigMaps("default").Get(context.Background(), name, metav1.GetOptions{})
					if err != nil {
						return err
					}
					if cm.UID != replacedUID || cm.Data["owner"] != "replacement" || cm.Data["mode"] != "one" {
						return fmt.Errorf("replacement unexpectedly changed: uid=%s data=%v", cm.UID, cm.Data)
					}
					for _, entry := range cm.ManagedFields {
						if entry.Manager == "terraform-patch" && entry.Operation == metav1.ManagedFieldsOperationApply {
							return nil
						}
					}
					return fmt.Errorf("patch did not acquire ownership on replacement UID %s", cm.UID)
				},
			},
		},
	})
}
