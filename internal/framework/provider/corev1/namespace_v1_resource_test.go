// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	k8sv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
)

const namespaceResourceName = "kubernetes_namespace_v1.test"

// mainClientset resolves a Kubernetes client the same way the resource does at runtime:
// through the SDKv2 provider meta that the framework provider is handed at configure
// time. The SDKv2 test file uses the package-level testAccProvider, which does not exist
// in this package.
func mainClientset() (*k8sclient.Clientset, error) {
	return sdkv2providerMeta()().(kubernetes.KubeClientsets).MainClientset()
}

// nullMetadataMap asserts that a metadata map is null rather than an empty map.
//
// SDKv2 could not tell the two apart, so its tests assert `metadata.0.labels.% == 0`.
// The framework can, and an omitted map stays null — which has no "%" entry in the
// flatmap at all, so the SDKv2 form of the assertion fails with "attribute not found"
// rather than a value mismatch. Asserting null directly also catches a regression that
// turns null into an empty map.
func nullMetadataMap(attr string) statecheck.StateCheck {
	return statecheck.ExpectKnownValue(
		namespaceResourceName,
		tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey(attr),
		knownvalue.Null(),
	)
}

func TestAccKubernetesNamespaceV1_basic(t *testing.T) {
	var conf k8sv1.Namespace
	nsName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceV1Config_basic(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &conf),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.name", nsName),
					resource.TestCheckResourceAttrSet(namespaceResourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(namespaceResourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(namespaceResourceName, "metadata.0.uid"),
				),
				// The API server always adds kubernetes.io/metadata.name. These two
				// assertions are the acceptance-level proof that Read filters it out.
				ConfigStateChecks: []statecheck.StateCheck{
					nullMetadataMap("annotations"),
					nullMetadataMap("labels"),
				},
			},
			{
				ResourceName:      namespaceResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccKubernetesNamespaceV1Config_Annotations(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &conf),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.name", nsName),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					nullMetadataMap("labels"),
				},
			},
			{
				ResourceName:      namespaceResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccKubernetesNamespaceV1Config_Annotations_Labels(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &conf),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.labels.TestLabelThree", "three"),
				),
			},
			{
				ResourceName:      namespaceResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Shrinks labels and swaps an annotation key: exercises Remove and Add
				// operations in a single patch.
				Config: testAccKubernetesNamespaceV1Config_smallerLists(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &conf),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.annotations.Different", "1234"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.labels.TestLabelThree", "three"),
				),
			},
			{
				ResourceName:      namespaceResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Removing the maps entirely. Every key the practitioner declared is
				// patched away, but the server's own label must not come back into state.
				Config: testAccKubernetesNamespaceV1Config_basic(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &conf),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.name", nsName),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					nullMetadataMap("annotations"),
					nullMetadataMap("labels"),
				},
			},
			{
				ResourceName:      namespaceResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccKubernetesNamespaceV1_identity(t *testing.T) {
	nsName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		CheckDestroy: testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceV1Config_basic(nsName),
				// api_version and kind are literals because client-go's typed clients
				// clear TypeMeta on responses — the resource hardcodes them for the
				// same reason.
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(namespaceResourceName, map[string]knownvalue.Check{
						"name":        knownvalue.StringExact(nsName),
						"api_version": knownvalue.StringExact("v1"),
						"kind":        knownvalue.StringExact("Namespace"),
					}),
				},
			},
			{
				ResourceName:    namespaceResourceName,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func TestAccKubernetesNamespaceV1_default_service_account(t *testing.T) {
	var nsConf k8sv1.Namespace
	var saConf k8sv1.ServiceAccount
	nsName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceV1Config_waitForDefaultServiceAccount(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &nsConf),
					testAccCheckKubernetesDefaultServiceAccountExists(namespaceResourceName, &saConf),
					resource.TestCheckResourceAttr(namespaceResourceName, "wait_for_default_service_account", "true"),
				),
			},
			{
				ResourceName:      namespaceResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// wait_for_default_service_account is ignored here, and only here. It is
				// practitioner intent for Create and is never stored on the server, so an
				// import cannot recover `true` — ImportState seeds the schema default,
				// false. The other import tests deliberately do NOT ignore it: with an
				// omitted attribute they prove import yields false rather than null.
				ImportStateVerifyIgnore: []string{"wait_for_default_service_account"},
			},
		},
	})
}

func TestAccKubernetesNamespaceV1_complete(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		config    func(string) string
		generated bool
	}{
		{name: "name", config: testAccKubernetesNamespaceV1Config_completeName},
		{name: "generateName", config: testAccKubernetesNamespaceV1Config_completeGeneratedName, generated: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var namespace k8sv1.Namespace
			var serviceAccount k8sv1.ServiceAccount
			name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
			nameCheck := resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.name", name)
			if testCase.generated {
				name += "-"
				nameCheck = resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.generate_name", name),
					resource.TestMatchResourceAttr(namespaceResourceName, "metadata.0.name", regexp.MustCompile("^"+regexp.QuoteMeta(name)+".+$")),
				)
			}

			resource.ParallelTest(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				CheckDestroy:             testAccCheckKubernetesNamespaceV1Destroy,
				Steps: []resource.TestStep{
					{
						Config: testCase.config(name),
						Check: resource.ComposeAggregateTestCheckFunc(
							testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &namespace),
							testAccCheckKubernetesDefaultServiceAccountExists(namespaceResourceName, &serviceAccount),
							nameCheck,
							resource.TestCheckResourceAttrPair(namespaceResourceName, "id", namespaceResourceName, "metadata.0.name"),
							resource.TestCheckResourceAttrSet(namespaceResourceName, "metadata.0.uid"),
							resource.TestCheckResourceAttrSet(namespaceResourceName, "metadata.0.generation"),
							resource.TestCheckResourceAttrSet(namespaceResourceName, "metadata.0.resource_version"),
							resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.annotations.%", "2"),
							resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
							resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
							resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.labels.%", "3"),
							resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.labels.TestLabelOne", "one"),
							resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.labels.TestLabelTwo", "two"),
							resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.labels.TestLabelThree", "three"),
							resource.TestCheckResourceAttr(namespaceResourceName, "wait_for_default_service_account", "true"),
							resource.TestCheckResourceAttr(namespaceResourceName, "timeouts.delete", "10m"),
						),
					},
					{
						ResourceName:            namespaceResourceName,
						ImportState:             true,
						ImportStateVerify:       true,
						ImportStateVerifyIgnore: []string{"wait_for_default_service_account"},
					},
				},
			})
		})
	}
}

func TestAccKubernetesNamespaceV1_generatedName(t *testing.T) {
	var conf, afterUpdate k8sv1.Namespace
	prefix := "tf-acc-test-gen-"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceV1Config_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &conf),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.generate_name", prefix),
					// name is server-assigned, so this is the shape where it is unknown
					// at plan time — the case that decides whether the schema needs
					// UseStateForUnknown.
					resource.TestMatchResourceAttr(namespaceResourceName, "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet(namespaceResourceName, "metadata.0.uid"),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					nullMetadataMap("annotations"),
					nullMetadataMap("labels"),
				},
			},
			{
				ResourceName:      namespaceResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccKubernetesNamespaceV1Config_generatedNameWithLabels(prefix),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(namespaceResourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &afterUpdate),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.labels.env", "demo"),
					resource.TestMatchResourceAttr(namespaceResourceName, "metadata.0.name", regexp.MustCompile("^"+prefix)),
					testAccCheckNamespaceNotRecreated(&conf, &afterUpdate),
				),
			},
			{
				ResourceName:      namespaceResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// Keys containing "/" require escaping: it separates tokens in an RFC 6901
// pointer and must be written as "~1".
//
// The SDKv2 suite only creates such keys, and a create sends the whole object rather
// than a patch, so the escaping was never actually exercised. The second step here
// changes one escaped key's value, removes another, and adds a third, producing
// Replace, Remove, and Add operations against escaped paths.
func TestAccKubernetesNamespaceV1_withSpecialCharacters(t *testing.T) {
	var conf k8sv1.Namespace
	nsName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceV1Config_specialCharacters(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &conf),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.annotations.myhost.co.uk/any-path", "one"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.annotations.myhost.co.uk/remove-me", "1234"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.labels.myhost.co.uk/any-path", "one"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.labels.myhost.co.uk/remove-me", "three"),
				),
			},
			{
				ResourceName:      namespaceResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccKubernetesNamespaceV1Config_specialCharactersUpdated(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &conf),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.annotations.myhost.co.uk/any-path", "two"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.annotations.myhost.co.uk/new-key", "new"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckNoResourceAttr(namespaceResourceName, "metadata.0.annotations.myhost.co.uk/remove-me"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.labels.myhost.co.uk/any-path", "two"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.labels.myhost.co.uk/new-key", "new"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.labels.%", "2"),
					resource.TestCheckNoResourceAttr(namespaceResourceName, "metadata.0.labels.myhost.co.uk/remove-me"),
				),
			},
			{
				ResourceName:      namespaceResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccKubernetesNamespaceV1_deleteTimeout(t *testing.T) {
	var conf k8sv1.Namespace
	nsName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceV1Config_timeouts(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &conf),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.name", nsName),
					resource.TestCheckResourceAttr(namespaceResourceName, "timeouts.delete", "10m"),
				),
			},
		},
	})
}

func TestAccKubernetesNamespaceV1_emptyMaps(t *testing.T) {
	var before, after k8sv1.Namespace
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	empty := knownvalue.MapExact(map[string]knownvalue.Check{})
	var steps []resource.TestStep
	for stepIndex, testCase := range []struct {
		config      string
		annotations knownvalue.Check
		labels      knownvalue.Check
	}{
		{testAccKubernetesNamespaceV1Config_emptyMaps(name), empty, empty},
		{testAccKubernetesNamespaceV1Config_Annotations_Labels(name),
			knownvalue.MapExact(map[string]knownvalue.Check{
				"TestAnnotationOne": knownvalue.StringExact("one"),
				"TestAnnotationTwo": knownvalue.StringExact("two"),
			}),
			knownvalue.MapExact(map[string]knownvalue.Check{
				"TestLabelOne":   knownvalue.StringExact("one"),
				"TestLabelTwo":   knownvalue.StringExact("two"),
				"TestLabelThree": knownvalue.StringExact("three"),
			})},
		{testAccKubernetesNamespaceV1Config_emptyMaps(name), empty, empty},
		{testAccKubernetesNamespaceV1Config_basic(name), knownvalue.Null(), knownvalue.Null()},
		{testAccKubernetesNamespaceV1Config_emptyMaps(name), empty, empty},
	} {
		step := resource.TestStep{
			Config: testCase.config,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue(namespaceResourceName, tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("annotations"), testCase.annotations),
				statecheck.ExpectKnownValue(namespaceResourceName, tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("labels"), testCase.labels),
			},
		}
		if stepIndex == 0 {
			step.Check = testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &before)
		} else {
			step.ConfigPlanChecks = resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(namespaceResourceName, plancheck.ResourceActionUpdate),
				},
			}
			step.Check = resource.ComposeAggregateTestCheckFunc(
				testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &after),
				testAccCheckNamespaceNotRecreated(&before, &after),
			)
		}
		steps = append(steps, step)
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesNamespaceV1Destroy,
		Steps:                    steps,
	})
}

func TestAccKubernetesNamespaceV1_replacement(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		config func(string) string
	}{
		{name: "name", config: testAccKubernetesNamespaceV1Config_basic},
		{name: "generate_name", config: testAccKubernetesNamespaceV1Config_generatedName},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var before, after k8sv1.Namespace
			name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
			oldName, newName := name+"-old", name+"-new"
			nameCheck := resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.name", newName)
			if testCase.name == "generate_name" {
				oldName += "-"
				newName += "-"
				nameCheck = resource.TestMatchResourceAttr(namespaceResourceName, "metadata.0.name", regexp.MustCompile("^"+regexp.QuoteMeta(newName)+".+$"))
			}

			resource.ParallelTest(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				CheckDestroy:             testAccCheckKubernetesNamespaceV1Destroy,
				Steps: []resource.TestStep{
					{
						Config: testCase.config(oldName),
						Check:  testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &before),
					},
					{
						Config: testCase.config(newName),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{
								plancheck.ExpectResourceAction(namespaceResourceName, plancheck.ResourceActionDestroyBeforeCreate),
							},
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &after),
							nameCheck,
							resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0."+testCase.name, newName),
							func(*terraform.State) error {
								if after.UID == "" || after.UID == before.UID {
									return fmt.Errorf("expected replacement UID, got %q (previous %q)", after.UID, before.UID)
								}
								client, err := mainClientset()
								if err != nil {
									return err
								}
								_, err = client.CoreV1().Namespaces().Get(context.Background(), before.Name, metav1.GetOptions{})
								if apierrors.IsNotFound(err) {
									return nil
								}
								if err != nil {
									return err
								}
								return fmt.Errorf("old namespace %q still exists", before.Name)
							},
						),
					},
				},
			})
		})
	}
}

func TestAccKubernetesNamespaceV1_settingNameSameValueAfterGenerateName(t *testing.T) {
	var before, after k8sv1.Namespace
	variables := config.Variables{}
	prefix := "tf-acc-test-gen-"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceV1Config_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &before),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr(namespaceResourceName, "metadata.0.name", regexp.MustCompile("^"+prefix+".+$")),
				),
			},
			{
				PreConfig: func() {
					variables["name"] = config.StringVariable(before.Name)
				},
				Config: `variable "name" {
  type = string
}

resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = var.name
  }
}
`,
				ConfigVariables: variables,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(namespaceResourceName, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(namespaceResourceName, tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("generate_name"), knownvalue.Null()),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &after),
					func(state *terraform.State) error {
						if after.Name != before.Name {
							return fmt.Errorf("namespace name changed: %q -> %q", before.Name, after.Name)
						}
						if after.UID == "" || after.UID == before.UID {
							return fmt.Errorf("expected replacement UID, got %q (previous %q)", after.UID, before.UID)
						}
						return resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.name", before.Name)(state)
					},
				),
			},
		},
	})
}

func testAccCheckKubernetesNamespaceV1Destroy(s *terraform.State) error {
	conn, err := mainClientset()
	if err != nil {
		return err
	}
	namespaces := conn.CoreV1().Namespaces()
	ctx := context.Background()
	for _, rs := range s.RootModule().Resources {
		// Both type names: a MoveState test that fails partway leaves state under the
		// old unversioned address, and matching only the versioned one would let that
		// leak pass silently.
		if rs.Type != "kubernetes_namespace_v1" && rs.Type != "kubernetes_namespace" {
			continue
		}

		_, err := namespaces.Get(ctx, rs.Primary.ID, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("checking namespace %q destruction: %w", rs.Primary.ID, err)
		}
		return fmt.Errorf("namespace still exists: %s", rs.Primary.ID)
	}

	return nil
}

func testAccCheckKubernetesNamespaceV1Exists(n string, obj *k8sv1.Namespace) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		conn, err := mainClientset()
		if err != nil {
			return err
		}

		out, err := conn.CoreV1().Namespaces().Get(context.TODO(), rs.Primary.ID, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

// testAccCheckNamespaceNotRecreated fails if the namespace was replaced rather than
// updated in place. UID is assigned by the API server and is stable for an object's
// lifetime, so a change means delete+create.
func testAccCheckNamespaceNotRecreated(before, after *k8sv1.Namespace) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if before.UID != after.UID {
			return fmt.Errorf("namespace was recreated: uid %s -> %s", before.UID, after.UID)
		}
		return nil
	}
}

func testAccCheckKubernetesDefaultServiceAccountExists(n string, obj *k8sv1.ServiceAccount) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		conn, err := mainClientset()
		if err != nil {
			return err
		}

		// The namespace resource ID is the namespace name; the SDKv2 test routed it
		// through idParts, which is unexported and unnecessary for a cluster-scoped
		// resource.
		out, err := conn.CoreV1().ServiceAccounts(rs.Primary.ID).Get(context.TODO(), "default", metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesNamespaceV1Config_emptyMaps(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name        = "%s"
    annotations = {}
    labels      = {}
  }
}
`, name)
}

func testAccKubernetesNamespaceV1Config_smallerLists(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
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
}
`, nsName)
}

func testAccKubernetesNamespaceV1Config_specialCharacters(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    annotations = {
      "myhost.co.uk/any-path"  = "one"
      "myhost.co.uk/remove-me" = "1234"
    }

    labels = {
      "myhost.co.uk/any-path"  = "one"
      "myhost.co.uk/remove-me" = "three"
    }

    name = "%s"
  }
}
`, nsName)
}

// Replaces, removes, and adds keys whose patch paths all require ~1 escaping.
func testAccKubernetesNamespaceV1Config_specialCharactersUpdated(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    annotations = {
      "myhost.co.uk/any-path" = "two"
      "myhost.co.uk/new-key"  = "new"
    }

    labels = {
      "myhost.co.uk/any-path" = "two"
      "myhost.co.uk/new-key"  = "new"
    }

    name = "%s"
  }
}
`, nsName)
}

func testAccKubernetesNamespaceV1Config_waitForDefaultServiceAccount(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = "%s"
  }

  wait_for_default_service_account = true
}
`, nsName)
}

func testAccKubernetesNamespaceV1Config_generatedNameWithLabels(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    generate_name = "%s"

    labels = {
      env = "demo"
    }
  }
}
`, prefix)
}

// Metadata kept out of Terraform state must survive an update to the other metadata map.
//
// Update diffs each map against prior state. When state holds no keys for a map,
// DiffStringMap emits a single add of the whole object, and JSON Patch add REPLACES it,
// deleting keys owned by other controllers. common.MetadataPatchOps skips a map with no
// managed keys on either side, which is what these cases pin.
//
// Two ways a key is absent from state, both covered because they reach the same code by
// different routes:
//
//   - internal keys, filtered by common.FlattenMetadata for every resource; and
//   - keys matched by the provider's ignore_annotations / ignore_labels, which need the
//     muxed server so the HCL provider block is honoured.
//
// The assertions read the live object. Nothing shows in the plan or in state: that is the
// point, and it is why the earlier tests could not catch this. They only ever had
// kubernetes.io/metadata.name as a filtered key, which the API server re-adds on every
// write, healing the damage before any assertion ran.
func TestAccKubernetesNamespaceV1_updateDoesNotAffectIgnoredMetadata(t *testing.T) {
	muxFactories := map[string]func() (tfprotov6.ProviderServer, error){
		"kubernetes": func() (tfprotov6.ProviderServer, error) {
			return mux.MuxServer(context.Background(), "test")
		},
	}

	for _, tc := range []struct {
		name          string
		ignored       string // the map Terraform does not manage
		managed       string // the map the update changes
		key           string
		providerBlock string
		factories     map[string]func() (tfprotov6.ProviderServer, error)
	}{
		{
			name: "internal annotation survives a label update", ignored: "annotations", managed: "labels",
			key: "example.kubernetes.io/owner", factories: testAccProtoV6ProviderFactories,
		},
		{
			name: "internal label survives an annotation update", ignored: "labels", managed: "annotations",
			key: "example.kubernetes.io/owner", factories: testAccProtoV6ProviderFactories,
		},
		{
			name: "ignore_annotations survives a label update", ignored: "annotations", managed: "labels",
			key:           "outside.example/keep",
			providerBlock: "provider \"kubernetes\" {\n  ignore_annotations = [\"^outside[.]example/keep$\"]\n}\n",
			factories:     muxFactories,
		},
		{
			name: "ignore_labels survives an annotation update", ignored: "labels", managed: "annotations",
			key:           "outside.example/keep",
			providerBlock: "provider \"kubernetes\" {\n  ignore_labels = [\"^outside[.]example/keep$\"]\n}\n",
			factories:     muxFactories,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var before, after k8sv1.Namespace
			nsName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

			// ignoredMap is "" to leave the map out of the configuration, or a line such as
			// `annotations = {}` to declare it empty.
			config := func(value, ignoredMap string) string {
				return tc.providerBlock + fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    %s = {
      managed = %q
    }

    %s

    name = %q
  }
}
`, tc.managed, value, ignoredMap, nsName)
			}

			// Every step re-plans after applying: an apply that leaves state disagreeing with
			// the API would show up here rather than in the next test run.
			settles := func(preApply ...plancheck.PlanCheck) resource.ConfigPlanChecks {
				return resource.ConfigPlanChecks{
					PreApply:             preApply,
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				}
			}
			update := settles(plancheck.ExpectResourceAction(namespaceResourceName, plancheck.ResourceActionUpdate))

			// The ignored map is null in state whether it is omitted or declared empty: the
			// live key is filtered out either way.
			ignoredIsNull := []statecheck.StateCheck{nullMetadataMap(tc.ignored)}
			ignoredIsEmpty := []statecheck.StateCheck{
				statecheck.ExpectKnownValue(namespaceResourceName,
					tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey(tc.ignored),
					knownvalue.MapExact(map[string]knownvalue.Check{})),
			}

			// The ignored key is only observable on the live object; the managed value is
			// state's business.
			survived := func() resource.TestCheckFunc {
				return resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &after),
					testAccCheckNamespaceNotRecreated(&before, &after),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0."+tc.managed+".managed", "after"),
					testAccCheckIgnoredNamespaceMetadata(nsName, tc.ignored, tc.key),
				)
			}

			resource.ParallelTest(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: tc.factories,
				CheckDestroy:             testAccCheckKubernetesNamespaceV1Destroy,
				Steps: []resource.TestStep{
					{
						Config:           config("before", ""),
						ConfigPlanChecks: settles(),
						Check:            testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &before),
					},
					{
						// Another controller writes the key Terraform does not manage.
						PreConfig: func() { testAccInjectIgnoredNamespaceMetadata(t, nsName, tc.ignored, tc.key) },
						// Filtered out of state, so adopting it plans nothing.
						Config:            config("before", ""),
						ConfigPlanChecks:  settles(plancheck.ExpectEmptyPlan()),
						ConfigStateChecks: ignoredIsNull,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0."+tc.managed+".managed", "before"),
							testAccCheckIgnoredNamespaceMetadata(nsName, tc.ignored, tc.key),
						),
					},
					{
						// The update the practitioner asked for must not touch the other map.
						Config:            config("after", ""),
						ConfigPlanChecks:  update,
						ConfigStateChecks: ignoredIsNull,
						Check:             survived(),
					},
					{
						// null -> {} on the ignored map: a state-only change, since neither
						// side names a key. The live key must still survive.
						Config:            config("after", tc.ignored+" = {}"),
						ConfigPlanChecks:  update,
						ConfigStateChecks: ignoredIsEmpty,
						Check:             survived(),
					},
					{
						// and back, {} -> null.
						Config:            config("after", ""),
						ConfigPlanChecks:  update,
						ConfigStateChecks: ignoredIsNull,
						Check:             survived(),
					},
				},
			})
		})
	}
}
