package kubernetes

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesSecret_basic(t *testing.T) {
	var conf1, conf2 api.Secret
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_secret.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesSecretDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesSecretConfig_emptyData(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretExists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					//testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "TestAnnotationTwo": "two"}),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					//testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelTwo": "two", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "data.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "type", "Opaque"),
					resource.TestCheckResourceAttr(resourceName, "immutable", "false"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesSecretConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretExists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					//testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "TestAnnotationTwo": "two"}),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					//testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelTwo": "two", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "data.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "data.one", "first"),
					resource.TestCheckResourceAttr(resourceName, "data.two", "second"),
					resource.TestCheckResourceAttr(resourceName, "type", "Opaque"),
					testAccCheckSecretData(&conf2, map[string]string{"one": "first", "two": "second"}),
					testAccCheckSecretRecreated(&conf1, &conf2, false),
				),
			},
			{
				Config: testAccKubernetesSecretConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretExists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.Different", "1234"),
					//testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "Different": "1234"}),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					//testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "data.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "data.one", "first"),
					resource.TestCheckResourceAttr(resourceName, "data.two", "second"),
					resource.TestCheckResourceAttr(resourceName, "data.nine", "ninth"),
					resource.TestCheckResourceAttr(resourceName, "type", "Opaque"),
					testAccCheckSecretData(&conf1, map[string]string{"one": "first", "two": "second", "nine": "ninth"}),
				),
			},
			{
				Config: testAccKubernetesSecretConfig_noData(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretExists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					//testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					//testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "data.%", "0"),
					testAccCheckSecretData(&conf1, map[string]string{}),
				),
			},
			{
				Config: testAccKubernetesSecretConfig_typeSpecified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretExists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					//testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					//testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "data.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "data.username", "admin"),
					resource.TestCheckResourceAttr(resourceName, "data.password", "password"),
					resource.TestCheckResourceAttr(resourceName, "type", "kubernetes.io/basic-auth"),
					testAccCheckSecretData(&conf1, map[string]string{"username": "admin", "password": "password"}),
				),
			},
		},
	})
}

func TestAccKubernetesSecret_immutable(t *testing.T) {
	var conf1 api.Secret
	var conf2 api.Secret
	var conf3 api.Secret

	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_secret.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesSecretDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesSecretConfig_immutable(name, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretExists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "immutable", "true"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesSecretConfig_immutable(name, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretExists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "immutable", "false"),
					testAccCheckSecretRecreated(&conf1, &conf2, true),
				),
			},
			{
				Config: testAccKubernetesSecretConfig_immutable(name, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretExists(resourceName, &conf3),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "immutable", "true"),
					testAccCheckSecretRecreated(&conf2, &conf3, false),
				),
			},
		},
	})
}

func TestAccKuberNetesSecret_dotInName(t *testing.T) {
	var conf api.Secret
	resourceName := "kubernetes_secret.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesSecretDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesSecretConfig_dotInSecretName,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", "dot.test"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
		},
	})
}

func TestAccKubernetesSecret_generatedName(t *testing.T) {
	var conf api.Secret
	prefix := "tf-acc-test-gen-"
	resourceName := "kubernetes_secret.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesSecretDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesSecretConfig_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					//testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					//testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr(resourceName, "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
		},
	})
}

func TestAccKubernetesSecret_binaryData(t *testing.T) {
	var conf api.Secret
	prefix := "tf-acc-test-gen-"
	resourceName := "kubernetes_secret.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesSecretDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesSecretConfig_binaryData(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "binary_data.%", "1"),
				),
			},
			{
				Config: testAccKubernetesSecretConfig_binaryData2(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "binary_data.%", "2"),
				),
			},
			{
				Config: testAccKubernetesSecretConfig_binaryDataCombined(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "data.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "binary_data.%", "2"),
				),
			},
		},
	})
}

func testAccCheckSecretData(m *api.Secret, expected map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(expected) == 0 && len(m.Data) == 0 {
			return nil
		}
		if !reflect.DeepEqual(flattenByteMapToStringMap(m.Data), expected) {
			return fmt.Errorf("%s data don't match.\nExpected: %q\nGiven: %q",
				m.Name, expected, m.Data)
		}
		return nil
	}
}

func testAccCheckSecretRecreated(sec1, sec2 *api.Secret, recreated bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		wasRecreated := sec2.CreationTimestamp.After(sec1.CreationTimestamp.Time)
		if wasRecreated != recreated {
			return fmt.Errorf("secret should have been recreated: %t, was it recreated: %t", recreated, wasRecreated)
		}

		return nil
	}
}

func testAccCheckKubernetesSecretDestroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_secret" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Secret still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesSecretExists(n string, obj *api.Secret) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
		if err != nil {
			return err
		}
		ctx := context.TODO()

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		out, err := conn.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

const testAccKubernetesSecretConfig_dotInSecretName = `
resource "kubernetes_secret" "test" {
  metadata {
    name = "dot.test"
  }
  }  
`

func testAccKubernetesSecretConfig_emptyData(name string) string {
	return fmt.Sprintf(`resource "kubernetes_secret" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    name = "%s"
  }

  data = {}
}
`, name)
}

func testAccKubernetesSecretConfig_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_secret" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    name = "%s"
  }

  data = {
    one = "first"
    two = "second"
  }
}
`, name)
}

func testAccKubernetesSecretConfig_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_secret" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      Different         = "1234"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
    }

    name = "%s"
  }

  data = {
    one  = "first"
    two  = "second"
    nine = "ninth"
  }
}
`, name)
}

func testAccKubernetesSecretConfig_noData(name string) string {
	return fmt.Sprintf(`resource "kubernetes_secret" "test" {
  metadata {
    name = "%s"
  }

  data = {}
}
`, name)
}

func testAccKubernetesSecretConfig_typeSpecified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_secret" "test" {
  metadata {
    name = "%s"
  }

  data = {
    username = "admin"
    password = "password"
  }

  type = "kubernetes.io/basic-auth"
}
`, name)
}

func testAccKubernetesSecretConfig_generatedName(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_secret" "test" {
  metadata {
    generate_name = "%s"
  }

  data = {
    one = "first"
    two = "second"
  }
}
`, prefix)
}

func testAccKubernetesSecretConfig_binaryData(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_secret" "test" {
  metadata {
    generate_name = "%s"
  }

  binary_data = {
    one = filebase64("./test-fixtures/binary.data")
  }
}
`, prefix)
}

func testAccKubernetesSecretConfig_binaryData2(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_secret" "test" {
  metadata {
    generate_name = "%s"
  }

  binary_data = {
    one = filebase64("./test-fixtures/binary.data")
    two = filebase64("./test-fixtures/binary2.data")
  }
}
`, prefix)
}

func testAccKubernetesSecretConfig_binaryDataCombined(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_secret" "test" {
  metadata {
    generate_name = "%s"
  }

  data = {
	  "HOST" = "127.0.0.1"
	  "PORT" = "80"
  }

  binary_data = {
    one = filebase64("./test-fixtures/binary.data")
    two = filebase64("./test-fixtures/binary2.data")
  }
}
`, prefix)
}

func testAccKubernetesSecretConfig_immutable(name string, immutable bool) string {
	return fmt.Sprintf(`resource "kubernetes_secret" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    name = "%s"
  }

  immutable = %t
  data      = {}
}
`, name, immutable)
}
