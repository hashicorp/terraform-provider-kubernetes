package kubernetes

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestAccKubernetesLabel_basic_namespace(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_label.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_label.test",
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesLabelDestroy_namespace(t, name),
		Steps: []resource.TestStep{
			{
				Config:    testAccKubernetesLabelConfig_basic_namespace(name),
				PreConfig: func() { testAccKubernetesLabel_create_namespace(t, name) },
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLabelExists(resourceName),
					resource.TestCheckResourceAttr("kubernetes_label.test", "label_value", "foobar"),
				),
			},
		},
	})
}

func TestAccKubernetesLabel_basic_deployment(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_label.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_label.test",
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesLabelDestroy_deployment(t, name),
		Steps: []resource.TestStep{
			{
				Config:    testAccKubernetesLabelConfig_basic_deployment(name),
				PreConfig: func() { testAccKubernetesLabel_create_deployment(t, name) },
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLabelExists(resourceName),
					resource.TestCheckResourceAttr("kubernetes_label.test", "label_value", "foobar"),
				),
			},
		},
	})
}

func testAccKubernetesLabel_create_namespace(t *testing.T, name string) {
	t.Helper()

	client, err := newDynamicClientFromMeta(testAccProvider.Meta())
	if err != nil {
		t.Fatalf("unable to create dynamic client: %v", err)
	}

	ctx := context.TODO()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	ns := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name": name,
			},
		},
	}
	_, err = client.Resource(schema.GroupVersionResource{Version: "v1", Resource: "namespaces"}).Create(ctx, ns, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("unable to create namespace: %v", err)
	}
}

func testAccKubernetesLabel_delete_namespace(t *testing.T, ctx context.Context, name string) {
	client, err := newDynamicClientFromMeta(testAccProvider.Meta())
	if err != nil {
		t.Fatalf("unable to create dynamic client: %v", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = client.Resource(schema.GroupVersionResource{Version: "v1", Resource: "namespaces"}).Delete(ctx, name, v1.DeleteOptions{})
	if err != nil {
		t.Fatalf("unable to delete namespace: %v", err)
	}
}

func testAccKubernetesLabel_create_deployment(t *testing.T, name string) {
	t.Helper()

	testAccKubernetesLabel_create_namespace(t, name)

	ctx := context.TODO()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	client, err := newDynamicClientFromMeta(testAccProvider.Meta())
	if err != nil {
		testAccKubernetesLabel_delete_namespace(t, ctx, name)
		t.Fatalf("unable to create dynamic client: %v", err)
	}

	deploy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": name,
				"labels": map[string]interface{}{
					"a": "b",
				},
			},
			"spec": map[string]interface{}{
				"replicas": 1,
				"selector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"app": name,
					},
				},
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"app": name,
						},
					},
					"spec": map[string]interface{}{
						"containers": []map[string]interface{}{
							{
								"name":  "nginx",
								"image": "nginx",
							},
						},
					},
				},
			},
		},
	}
	_, err = client.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}).Namespace(name).Create(ctx, deploy, v1.CreateOptions{})
	if err != nil {
		testAccKubernetesLabel_delete_namespace(t, ctx, name)
		t.Fatalf("unable to create deployment: %v", err)
	}
}

func testAccKubernetesLabel_delete_deployment(t *testing.T, ctx context.Context, name string) {
	defer testAccKubernetesLabel_delete_namespace(t, ctx, name)

	client, err := newDynamicClientFromMeta(testAccProvider.Meta())
	if err != nil {
		t.Fatalf("unable to create dynamic client: %v", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = client.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}).Namespace(name).Delete(ctx, name, v1.DeleteOptions{})
	if err != nil {
		t.Fatalf("unable to delete deployment: %v", err)
	}
}

func testAccCheckKubernetesLabelDestroy_namespace(t *testing.T, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client, err := newDynamicClientFromMeta(testAccProvider.Meta())
		if err != nil {
			return err
		}

		ctx := context.TODO()
		defer testAccKubernetesLabel_delete_namespace(t, ctx, name)

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "kubernetes_label" {
				continue
			}

			getFn := func(key string) interface{} {
				return rs.Primary.Attributes[key]
			}

			lc, err := newLabelClient(getFn, client)
			if err != nil {
				return err
			}

			labelKey, ok := rs.Primary.Attributes["label_key"]
			if !ok {
				return fmt.Errorf("Unable to extract label_key from attributes of resource")
			}

			res, err := lc.ReadResource(ctx)
			if err != nil {
				return err
			}

			labels := res.GetLabels()
			_, ok = labels[labelKey]
			if ok {
				return fmt.Errorf("Label still exists")
			}
		}

		return nil
	}
}

func testAccCheckKubernetesLabelDestroy_deployment(t *testing.T, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client, err := newDynamicClientFromMeta(testAccProvider.Meta())
		if err != nil {
			return err
		}

		ctx := context.TODO()
		defer testAccKubernetesLabel_delete_deployment(t, ctx, name)

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "kubernetes_label" {
				continue
			}

			getFn := func(key string) interface{} {
				return rs.Primary.Attributes[key]
			}

			lc, err := newLabelClient(getFn, client)
			if err != nil {
				return err
			}

			labelKey, ok := rs.Primary.Attributes["label_key"]
			if !ok {
				return fmt.Errorf("Unable to extract label_key from attributes of resource")
			}

			res, err := lc.ReadResource(ctx)
			if err != nil {
				return err
			}

			labels := res.GetLabels()
			_, ok = labels[labelKey]
			if ok {
				return fmt.Errorf("Label still exists")
			}
		}

		return nil
	}
}

func testAccCheckKubernetesLabelExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		client, err := newDynamicClientFromMeta(testAccProvider.Meta())
		if err != nil {
			return err
		}

		ctx := context.TODO()
		getFn := func(key string) interface{} {
			return rs.Primary.Attributes[key]
		}

		lc, err := newLabelClient(getFn, client)
		if err != nil {
			return err
		}

		_, err = lc.Read(ctx)
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccKubernetesLabelConfig_basic_namespace(name string) string {
	return fmt.Sprintf(`resource "kubernetes_label" "test" {
		  api_version      = "v1"
		  kind             = "Namespaces"
		  namespace_scoped = false
		  namespace        = null
		  name             = "%s"
		  label_key        = "%s"
		  label_value      = "foobar"
	  }`, name, name)
}

func testAccKubernetesLabelConfig_basic_deployment(name string) string {
	return fmt.Sprintf(`resource "kubernetes_label" "test" {
		  api_version      = "apps/v1"
		  kind             = "Deployments"
		  namespace_scoped = true
		  namespace        = "%s"
		  name             = "%s"
		  label_key        = "%s"
		  label_value      = "foobar"
	  }`, name, name, name)
}
