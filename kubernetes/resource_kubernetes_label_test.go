package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccKubernetesLabel_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_label.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_label.test",
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesLabelDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesLabelConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLabelExists(resourceName),
					resource.TestCheckResourceAttr("kubernetes_label.test", "label_value", "foobar"),
				),
			},
			// {
			// 	ResourceName:            resourceName,
			// 	ImportState:             true,
			// 	ImportStateVerify:       true,
			// 	ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			// },
			// {
			// 	Config: testAccKubernetesLabelConfig_metaModified(name),
			// 	Check: resource.ComposeAggregateTestCheckFunc(
			// 		testAccCheckKubernetesLabelExists("kubernetes_label.test", &namespaceConf),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "metadata.0.annotations.%", "2"),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "metadata.0.annotations.TestAnnotationOne", "one"),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
			// 		//testAccCheckMetaAnnotations(&namespaceConf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "TestAnnotationTwo": "two"}),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "metadata.0.labels.%", "3"),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "metadata.0.labels.TestLabelOne", "one"),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "metadata.0.labels.TestLabelTwo", "two"),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "metadata.0.labels.TestLabelThree", "three"),
			// 		//testAccCheckMetaLabels(&namespaceConf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelTwo": "two", "TestLabelThree": "three"}),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "metadata.0.name", name),
			// 		resource.TestCheckResourceAttrSet("kubernetes_label.test", "metadata.0.generation"),
			// 		resource.TestCheckResourceAttrSet("kubernetes_label.test", "metadata.0.resource_version"),
			// 		resource.TestCheckResourceAttrSet("kubernetes_label.test", "metadata.0.uid"),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.#", "1"),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.default.%", "2"),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.default.cpu", "200m"),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.default.memory", "512M"),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.default_request.%", "2"),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.default_request.cpu", "100m"),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.default_request.memory", "256M"),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.type", "Container"),
			// 	),
			// },
			// {
			// 	Config: testAccKubernetesLabelConfig_specModified(name),
			// 	Check: resource.ComposeAggregateTestCheckFunc(
			// 		testAccCheckKubernetesLabelExists("kubernetes_label.test", &namespaceConf),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "metadata.0.annotations.%", "0"),
			// 		//testAccCheckMetaAnnotations(&namespaceConf.ObjectMeta, map[string]string{}),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "metadata.0.labels.%", "0"),
			// 		//testAccCheckMetaLabels(&namespaceConf.ObjectMeta, map[string]string{}),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "metadata.0.name", name),
			// 		resource.TestCheckResourceAttrSet("kubernetes_label.test", "metadata.0.generation"),
			// 		resource.TestCheckResourceAttrSet("kubernetes_label.test", "metadata.0.resource_version"),
			// 		resource.TestCheckResourceAttrSet("kubernetes_label.test", "metadata.0.uid"),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.#", "1"),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.default.%", "2"),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.default.cpu", "200m"),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.default.memory", "1024M"),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.default_request.%", "2"),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.default_request.cpu", "100m"),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.default_request.memory", "256M"),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.max.%", "1"),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.max.cpu", "500m"),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.min.%", "2"),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.min.cpu", "10m"),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.min.memory", "10M"),
			// 		resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.type", "Container"),
			// 	),
			// },
		},
	})
}

// func TestAccKubernetesLabel_generatedName(t *testing.T) {
// 	var namespaceConf api.Namespace
// 	prefix := "tf-acc-test-"

// 	resource.Test(t, resource.TestCase{
// 		PreCheck:          func() { testAccPreCheck(t) },
// 		IDRefreshName:     "kubernetes_label.test",
// 		ProviderFactories: testAccProviderFactories,
// 		CheckDestroy:      testAccCheckKubernetesLabelDestroy,
// 		Steps: []resource.TestStep{
// 			{
// 				Config: testAccKubernetesLabelConfig_generatedName(prefix),
// 				Check: resource.ComposeAggregateTestCheckFunc(
// 					testAccCheckKubernetesLabelExists("kubernetes_label.test", &namespaceConf),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "metadata.0.annotations.%", "0"),
// 					//testAccCheckMetaAnnotations(&namespaceConf.ObjectMeta, map[string]string{}),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "metadata.0.labels.%", "0"),
// 					//testAccCheckMetaLabels(&namespaceConf.ObjectMeta, map[string]string{}),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "metadata.0.generate_name", prefix),
// 					resource.TestCheckResourceAttrSet("kubernetes_label.test", "metadata.0.generation"),
// 					resource.TestCheckResourceAttrSet("kubernetes_label.test", "metadata.0.resource_version"),
// 					resource.TestCheckResourceAttrSet("kubernetes_label.test", "metadata.0.uid"),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.#", "1"),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.type", "Pod"),
// 				),
// 			},
// 		},
// 	})
// }

// func TestAccKubernetesLabel_typeChange(t *testing.T) {
// 	var namespaceConf api.Namespace
// 	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

// 	resource.Test(t, resource.TestCase{
// 		PreCheck:          func() { testAccPreCheck(t) },
// 		IDRefreshName:     "kubernetes_label.test",
// 		ProviderFactories: testAccProviderFactories,
// 		CheckDestroy:      testAccCheckKubernetesLabelDestroy,
// 		Steps: []resource.TestStep{
// 			{
// 				Config: testAccKubernetesLabelConfig_typeChange(name),
// 				Check: resource.ComposeAggregateTestCheckFunc(
// 					testAccCheckKubernetesLabelExists("kubernetes_label.test", &namespaceConf),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "metadata.0.annotations.%", "0"),
// 					//testAccCheckMetaAnnotations(&namespaceConf.ObjectMeta, map[string]string{}),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "metadata.0.labels.%", "0"),
// 					//testAccCheckMetaLabels(&namespaceConf.ObjectMeta, map[string]string{}),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "metadata.0.name", name),
// 					resource.TestCheckResourceAttrSet("kubernetes_label.test", "metadata.0.generation"),
// 					resource.TestCheckResourceAttrSet("kubernetes_label.test", "metadata.0.resource_version"),
// 					resource.TestCheckResourceAttrSet("kubernetes_label.test", "metadata.0.uid"),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.#", "1"),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.default.%", "2"),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.default.cpu", "200m"),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.default.memory", "1024M"),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.type", "Container"),
// 				),
// 			},
// 			{
// 				Config: testAccKubernetesLabelConfig_typeChangeModified(name),
// 				Check: resource.ComposeAggregateTestCheckFunc(
// 					testAccCheckKubernetesLabelExists("kubernetes_label.test", &namespaceConf),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "metadata.0.annotations.%", "0"),
// 					//testAccCheckMetaAnnotations(&namespaceConf.ObjectMeta, map[string]string{}),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "metadata.0.labels.%", "0"),
// 					//testAccCheckMetaLabels(&namespaceConf.ObjectMeta, map[string]string{}),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "metadata.0.name", name),
// 					resource.TestCheckResourceAttrSet("kubernetes_label.test", "metadata.0.generation"),
// 					resource.TestCheckResourceAttrSet("kubernetes_label.test", "metadata.0.resource_version"),
// 					resource.TestCheckResourceAttrSet("kubernetes_label.test", "metadata.0.uid"),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.#", "1"),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.min.%", "2"),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.min.cpu", "200m"),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.min.memory", "1024M"),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.type", "Pod"),
// 				),
// 			},
// 		},
// 	})
// }

// func TestAccKubernetesLabel_multipleLimits(t *testing.T) {
// 	var namespaceConf api.Namespace
// 	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

// 	resource.Test(t, resource.TestCase{
// 		PreCheck:          func() { testAccPreCheck(t) },
// 		IDRefreshName:     "kubernetes_label.test",
// 		ProviderFactories: testAccProviderFactories,
// 		CheckDestroy:      testAccCheckKubernetesLabelDestroy,
// 		Steps: []resource.TestStep{
// 			{
// 				Config: testAccKubernetesLabelConfig_multipleLimits(name),
// 				Check: resource.ComposeAggregateTestCheckFunc(
// 					testAccCheckKubernetesLabelExists("kubernetes_label.test", &namespaceConf),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "metadata.0.annotations.%", "0"),
// 					//testAccCheckMetaAnnotations(&namespaceConf.ObjectMeta, map[string]string{}),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "metadata.0.labels.%", "0"),
// 					//testAccCheckMetaLabels(&namespaceConf.ObjectMeta, map[string]string{}),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "metadata.0.name", name),
// 					resource.TestCheckResourceAttrSet("kubernetes_label.test", "metadata.0.generation"),
// 					resource.TestCheckResourceAttrSet("kubernetes_label.test", "metadata.0.resource_version"),
// 					resource.TestCheckResourceAttrSet("kubernetes_label.test", "metadata.0.uid"),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.#", "3"),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.max.%", "2"),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.max.cpu", "200m"),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.max.memory", "1024M"),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.0.type", "Pod"),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.1.min.%", "1"),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.1.min.storage", "24M"),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.1.type", "PersistentVolumeClaim"),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.2.default.%", "2"),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.2.default.cpu", "50m"),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.2.default.memory", "24M"),
// 					resource.TestCheckResourceAttr("kubernetes_label.test", "spec.0.limit.2.type", "Container"),
// 				),
// 			},
// 		},
// 	})
// }

func testAccCheckKubernetesLabelDestroy(s *terraform.State) error {
	client, err := newDynamicClientFromMeta(testAccProvider.Meta())
	if err != nil {
		return err
	}

	ctx := context.TODO()

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

func testAccKubernetesLabelConfig_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace" "test" {
		metadata {
		  name = "test"
		}
	  
		lifecycle {
		  ignore_changes = [
			metadata[0].labels["test"],
		  ]
		}
	  }
	  
	  resource "kubernetes_label" "this" {
		  api_version      = "v1"
		  kind             = "Namespaces"
		  namespace_scoped = false
		  namespace        = null
		  name             = kubernetes_namespace.test.metadata[0].name
		  label_key        = "%s"
		  label_value      = "foobar"
	  }`, name)
}

// func testAccKubernetesLabelConfig_metaModified(name string) string {
// 	return fmt.Sprintf(`resource "kubernetes_limit_range" "test" {
//   metadata {
//     annotations = {
//       TestAnnotationOne = "one"
//       TestAnnotationTwo = "two"
//     }

//     labels = {
//       TestLabelOne   = "one"
//       TestLabelTwo   = "two"
//       TestLabelThree = "three"
//     }

//     name = "%s"
//   }

//   spec {
//     limit {
//       type = "Container"

//       default = {
//         cpu    = "200m"
//         memory = "512M"
//       }

//       default_request = {
//         cpu    = "100m"
//         memory = "256M"
//       }
//     }
//   }
// }
// `, name)
// }

// func testAccKubernetesLabelConfig_specModified(name string) string {
// 	return fmt.Sprintf(`resource "kubernetes_limit_range" "test" {
//   metadata {
//     name = "%s"
//   }

//   spec {
//     limit {
//       type = "Container"

//       default = {
//         cpu    = "200m"
//         memory = "1024M"
//       }

//       max = {
//         cpu = "500m"
//       }

//       min = {
//         cpu    = "10m"
//         memory = "10M"
//       }
//     }
//   }
// }
// `, name)
// }

// func testAccKubernetesLabelConfig_generatedName(prefix string) string {
// 	return fmt.Sprintf(`resource "kubernetes_limit_range" "test" {
//   metadata {
//     generate_name = "%s"
//   }

//   spec {
//     limit {
//       type = "Pod"
//     }
//   }
// }
// `, prefix)
// }

// func testAccKubernetesLabelConfig_typeChange(name string) string {
// 	return fmt.Sprintf(`resource "kubernetes_limit_range" "test" {
//   metadata {
//     name = "%s"
//   }

//   spec {
//     limit {
//       type = "Container"

//       default = {
//         cpu    = "200m"
//         memory = "1024M"
//       }
//     }
//   }
// }
// `, name)
// }

// func testAccKubernetesLabelConfig_typeChangeModified(name string) string {
// 	return fmt.Sprintf(`resource "kubernetes_limit_range" "test" {
//   metadata {
//     name = "%s"
//   }

//   spec {
//     limit {
//       type = "Pod"

//       min = {
//         cpu    = "200m"
//         memory = "1024M"
//       }
//     }
//   }
// }
// `, name)
// }

// func testAccKubernetesLabelConfig_multipleLimits(name string) string {
// 	return fmt.Sprintf(`resource "kubernetes_limit_range" "test" {
//   metadata {
//     name = "%s"
//   }

//   spec {
//     limit {
//       type = "Pod"

//       max = {
//         cpu    = "200m"
//         memory = "1024M"
//       }
//     }

//     limit {
//       type = "PersistentVolumeClaim"

//       min = {
//         storage = "24M"
//       }
//     }

//     limit {
//       type = "Container"

//       default = {
//         cpu    = "50m"
//         memory = "24M"
//       }
//     }
//   }
// }
// `, name)
// }
