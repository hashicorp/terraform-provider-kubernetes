// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package functions_test

import (
	"fmt"
	"math/big"
	"os"
	"path"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

func TestManifestDecodeMulti(t *testing.T) {
	t.Parallel()

	outputName := "test"

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testManifestDecodeMultiConfig("testdata/decode_single.yaml"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue(outputName, knownvalue.ListExact([]knownvalue.Check{
						knownvalue.ObjectExact(map[string]knownvalue.Check{
							"apiVersion": knownvalue.StringExact("v1"),
							"data": knownvalue.ObjectExact(map[string]knownvalue.Check{
								"configfile": knownvalue.StringExact("---\ntest: document\n"),
							}),
							"kind": knownvalue.StringExact("ConfigMap"),
							"metadata": knownvalue.ObjectExact(map[string]knownvalue.Check{
								"annotations": knownvalue.Null(),
								"labels": knownvalue.ObjectExact(map[string]knownvalue.Check{
									"test": knownvalue.StringExact("test---label"),
								}),
								"name": knownvalue.StringExact("test-configmap"),
							}),
						}),
					})),
				},
			},
			{
				Config: testManifestDecodeMultiConfig("testdata/decode_multi.yaml"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue(outputName, knownvalue.ListExact([]knownvalue.Check{
						knownvalue.ObjectExact(map[string]knownvalue.Check{
							"apiVersion": knownvalue.StringExact("apps/v1"),
							"kind":       knownvalue.StringExact("DaemonSet"),
							"metadata": knownvalue.ObjectExact(map[string]knownvalue.Check{
								"labels": knownvalue.ObjectExact(map[string]knownvalue.Check{
									"k8s-app": knownvalue.StringExact("fluentd-logging"),
								}),
								"name":      knownvalue.StringExact("fluentd-elasticsearch"),
								"namespace": knownvalue.StringExact("kube-system"),
							}),
							"spec": knownvalue.ObjectExact(map[string]knownvalue.Check{
								"selector": knownvalue.ObjectExact(map[string]knownvalue.Check{
									"matchLabels": knownvalue.ObjectExact(map[string]knownvalue.Check{
										"name": knownvalue.StringExact("fluentd-elasticsearch"),
									}),
								}),
								"template": knownvalue.ObjectExact(map[string]knownvalue.Check{
									"metadata": knownvalue.ObjectExact(map[string]knownvalue.Check{
										"labels": knownvalue.ObjectExact(map[string]knownvalue.Check{
											"name": knownvalue.StringExact("fluentd-elasticsearch"),
										}),
									}),
									"spec": knownvalue.ObjectExact(map[string]knownvalue.Check{
										"containers": knownvalue.ListExact([]knownvalue.Check{knownvalue.ObjectExact(map[string]knownvalue.Check{
											"image": knownvalue.StringExact("quay.io/fluentd_elasticsearch/fluentd:v2.5.2"),
											"name":  knownvalue.StringExact("fluentd-elasticsearch"),
											"resources": knownvalue.ObjectExact(map[string]knownvalue.Check{
												"limits": knownvalue.ObjectExact(map[string]knownvalue.Check{
													"cpu":    knownvalue.NumberExact(big.NewFloat(1.5)),
													"memory": knownvalue.StringExact("200Mi"),
												}),
												"requests": knownvalue.ObjectExact(map[string]knownvalue.Check{
													"cpu":    knownvalue.Int64Exact(1),
													"memory": knownvalue.StringExact("200Mi"),
												}),
											}),
											"volumeMounts": knownvalue.ListExact([]knownvalue.Check{knownvalue.ObjectExact(map[string]knownvalue.Check{
												"mountPath": knownvalue.StringExact("/var/log"),
												"name":      knownvalue.StringExact("varlog"),
											})}),
										})}),
										"terminationGracePeriodSeconds": knownvalue.Int64Exact(30),
										"tolerations": knownvalue.ListExact([]knownvalue.Check{
											knownvalue.ObjectExact(map[string]knownvalue.Check{
												"effect":   knownvalue.StringExact("NoSchedule"),
												"key":      knownvalue.StringExact("node-role.kubernetes.io/control-plane"),
												"operator": knownvalue.StringExact("Exists"),
											}),
											knownvalue.ObjectExact(map[string]knownvalue.Check{
												"effect":   knownvalue.StringExact("NoSchedule"),
												"key":      knownvalue.StringExact("node-role.kubernetes.io/master"),
												"operator": knownvalue.StringExact("Exists"),
											}),
										}),
										"volumes": knownvalue.ListExact([]knownvalue.Check{
											knownvalue.ObjectExact(map[string]knownvalue.Check{
												"hostPath": knownvalue.ObjectExact(map[string]knownvalue.Check{
													"path": knownvalue.StringExact("/var/log"),
												}),
												"name": knownvalue.StringExact("varlog"),
											}),
										}),
									}),
								}),
							}),
						}),
						knownvalue.ObjectExact(map[string]knownvalue.Check{
							"apiVersion": knownvalue.StringExact("v1"),
							"data": knownvalue.ObjectExact(map[string]knownvalue.Check{
								"configfile": knownvalue.StringExact("---\ntest: document"),
								"immutable":  knownvalue.Bool(false),
							}),
							"kind": knownvalue.StringExact("ConfigMap"),
							"metadata": knownvalue.ObjectExact(map[string]knownvalue.Check{
								"labels": knownvalue.ObjectExact(map[string]knownvalue.Check{
									"test": knownvalue.StringExact("test---label"),
								}),
								"name":      knownvalue.StringExact("test-configmap"),
								"namespace": knownvalue.StringExact("kube-system"),
							}),
						}),
						knownvalue.ObjectExact(map[string]knownvalue.Check{
							"apiVersion": knownvalue.StringExact("apps/v1"),
							"kind":       knownvalue.StringExact("DaemonSet"),
							"metadata": knownvalue.ObjectExact(map[string]knownvalue.Check{
								"labels": knownvalue.ObjectExact(map[string]knownvalue.Check{
									"k8s-app": knownvalue.StringExact("fluentd-logging"),
								}),
								"name":      knownvalue.StringExact("fluentd-elasticsearch2"),
								"namespace": knownvalue.StringExact("kube-system"),
							}),
							"spec": knownvalue.ObjectExact(map[string]knownvalue.Check{
								"selector": knownvalue.ObjectExact(map[string]knownvalue.Check{
									"matchLabels": knownvalue.ObjectExact(map[string]knownvalue.Check{
										"name": knownvalue.StringExact("fluentd-elasticsearch"),
									}),
								}),
								"template": knownvalue.ObjectExact(map[string]knownvalue.Check{
									"metadata": knownvalue.ObjectExact(map[string]knownvalue.Check{
										"labels": knownvalue.ObjectExact(map[string]knownvalue.Check{
											"name":      knownvalue.StringExact("fluentd-elasticsearch"),
											"something": knownvalue.StringExact("helloworld"),
										}),
									}),
									"spec": knownvalue.ObjectExact(map[string]knownvalue.Check{
										"containers": knownvalue.ListExact([]knownvalue.Check{knownvalue.ObjectExact(map[string]knownvalue.Check{
											"image": knownvalue.StringExact("quay.io/fluentd_elasticsearch/fluentd:v2.5.2"),
											"name":  knownvalue.StringExact("fluentd-elasticsearch"),
											"resources": knownvalue.ObjectExact(map[string]knownvalue.Check{
												"limits": knownvalue.ObjectExact(map[string]knownvalue.Check{
													"memory": knownvalue.StringExact("200Mi"),
												}),
												"requests": knownvalue.ObjectExact(map[string]knownvalue.Check{
													"cpu":    knownvalue.StringExact("100m"),
													"memory": knownvalue.StringExact("200Mi"),
												}),
											}),
											"volumeMounts": knownvalue.ListExact([]knownvalue.Check{knownvalue.ObjectExact(map[string]knownvalue.Check{
												"mountPath": knownvalue.StringExact("/var/log"),
												"name":      knownvalue.StringExact("varlog"),
											})}),
										})}),
										"terminationGracePeriodSeconds": knownvalue.Int64Exact(30),
										"tolerations": knownvalue.ListExact([]knownvalue.Check{
											knownvalue.ObjectExact(map[string]knownvalue.Check{
												"effect":   knownvalue.StringExact("NoSchedule"),
												"key":      knownvalue.StringExact("node-role.kubernetes.io/control-plane"),
												"operator": knownvalue.StringExact("Exists"),
											}),
											knownvalue.ObjectExact(map[string]knownvalue.Check{
												"effect":   knownvalue.StringExact("NoSchedule"),
												"key":      knownvalue.StringExact("node-role.kubernetes.io/master"),
												"operator": knownvalue.StringExact("Exists"),
											}),
										}),
										"volumes": knownvalue.ListExact([]knownvalue.Check{
											knownvalue.ObjectExact(map[string]knownvalue.Check{
												"hostPath": knownvalue.ObjectExact(map[string]knownvalue.Check{
													"path": knownvalue.StringExact("/var/log"),
												}),
												"name": knownvalue.StringExact("varlog"),
											}),
										}),
									}),
								}),
							}),
						}),
					})),
				},
			},
			{
				Config:      testManifestDecodeMultiConfig("testdata/decode_manifest_invalid.yaml"),
				ExpectError: regexp.MustCompile(`Invalid\s+Kubernetes\s+manifest`),
			},
			{
				Config:      testManifestDecodeMultiConfig("testdata/decode_manifest_invalid_syntax.yaml"),
				ExpectError: regexp.MustCompile(`Invalid\s+YAML\s+document`),
			},
		},
	})
}

func testManifestDecodeMultiConfig(filename string) string {
	cwd, _ := os.Getwd()
	return fmt.Sprintf(`
output "test" {
  value = provider::kubernetes::manifest_decode_multi(file(%q))
}`, path.Join(cwd, filename))
}
