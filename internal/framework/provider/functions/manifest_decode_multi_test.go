package functions_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"
)

func TestManifestDecodeMulti(t *testing.T) {
	t.Parallel()

	outputName := "test"

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testManifestDecodeMultiConfig("testdata/decode_single.yaml"),
				Check: resource.ComposeAggregateTestCheckFunc(
					// FIXME: terraform-plugin-testing doesn't support dynamic yet
					func(s *terraform.State) error {
						ms := s.RootModule()
						rs, ok := ms.Outputs[outputName]
						if !ok {
							return fmt.Errorf("no output value for %q", outputName)
						}
						expectedOutput := []interface{}{map[string]any{
							"apiVersion": "v1",
							"data": map[string]any{
								"configfile": "---\ntest: document\n",
							},
							"kind": "ConfigMap",
							"metadata": map[string]any{
								"labels": map[string]any{
									"test": "test---label",
								},
								"name": "test-configmap",
							},
						}}
						assert.Equal(t, expectedOutput, rs.Value)
						return nil
					},
				),
			},
			{
				Config: testManifestDecodeMultiConfig("testdata/decode_multi.yaml"),
				Check: resource.ComposeAggregateTestCheckFunc(
					// FIXME: terraform-plugin-testing doesn't support dynamic yet
					func(s *terraform.State) error {
						ms := s.RootModule()
						rs, ok := ms.Outputs[outputName]
						if !ok {
							return fmt.Errorf("no output value for %q", outputName)
						}
						expectedOutput := []any{
							map[string]any{
								"apiVersion": "apps/v1",
								"kind":       "DaemonSet",
								"metadata": map[string]any{
									"labels": map[string]any{
										"k8s-app": "fluentd-logging",
									},
									"name":      "fluentd-elasticsearch",
									"namespace": "kube-system",
								},
								"spec": map[string]any{
									"selector": map[string]any{
										"matchLabels": map[string]any{
											"name": "fluentd-elasticsearch",
										},
									},
									"template": map[string]any{
										"metadata": map[string]any{
											"labels": map[string]any{
												"name": "fluentd-elasticsearch",
											},
										},
										"spec": map[string]any{
											"containers": []any{map[string]any{
												"image": "quay.io/fluentd_elasticsearch/fluentd:v2.5.2",
												"name":  "fluentd-elasticsearch",
												"resources": map[string]any{
													"limits": map[string]any{
														"cpu":    json.Number("1.5"),
														"memory": "200Mi",
													},
													"requests": map[string]any{
														"cpu":    json.Number("1"),
														"memory": "200Mi",
													},
												},
												"volumeMounts": []any{map[string]any{
													"mountPath": "/var/log",
													"name":      "varlog",
												}},
											}},
											"terminationGracePeriodSeconds": json.Number("30"),
											"tolerations": []any{
												map[string]any{
													"effect":   "NoSchedule",
													"key":      "node-role.kubernetes.io/control-plane",
													"operator": "Exists",
												},
												map[string]any{
													"effect":   "NoSchedule",
													"key":      "node-role.kubernetes.io/master",
													"operator": "Exists",
												},
											},
											"volumes": []any{map[string]any{
												"hostPath": map[string]any{
													"path": "/var/log",
												},
												"name": "varlog",
											}},
										},
									},
								},
							},
							map[string]any{
								"apiVersion": "v1",
								"data": map[string]any{
									"configfile": "---\ntest: document",
									"immutable":  false,
								},
								"kind": "ConfigMap",
								"metadata": map[string]any{
									"labels": map[string]any{
										"test": "test---label",
									},
									"name":      "test-configmap",
									"namespace": "kube-system",
								},
							},
							map[string]any{
								"apiVersion": "apps/v1",
								"kind":       "DaemonSet",
								"metadata": map[string]any{
									"labels": map[string]any{
										"k8s-app": "fluentd-logging",
									},
									"name":      "fluentd-elasticsearch2",
									"namespace": "kube-system",
								},
								"spec": map[string]any{
									"selector": map[string]any{
										"matchLabels": map[string]any{
											"name": "fluentd-elasticsearch",
										},
									},
									"template": map[string]any{
										"metadata": map[string]any{
											"labels": map[string]any{
												"name":      "fluentd-elasticsearch",
												"something": "helloworld",
											},
										},
										"spec": map[string]any{
											"containers": []any{map[string]any{
												"image": "quay.io/fluentd_elasticsearch/fluentd:v2.5.2",
												"name":  "fluentd-elasticsearch",
												"resources": map[string]any{
													"limits": map[string]any{
														"memory": "200Mi",
													},
													"requests": map[string]any{
														"cpu":    "100m",
														"memory": "200Mi",
													},
												},
												"volumeMounts": []any{map[string]any{
													"mountPath": "/var/log",
													"name":      "varlog",
												}},
											}},
											"terminationGracePeriodSeconds": json.Number("30"),
											"tolerations": []any{
												map[string]any{
													"effect":   "NoSchedule",
													"key":      "node-role.kubernetes.io/control-plane",
													"operator": "Exists",
												},
												map[string]any{
													"effect":   "NoSchedule",
													"key":      "node-role.kubernetes.io/master",
													"operator": "Exists",
												},
											},
											"volumes": []any{
												map[string]any{
													"hostPath": map[string]any{
														"path": "/var/log",
													},
													"name": "varlog",
												},
											},
										},
									},
								},
							},
						}

						assert.Equal(t, expectedOutput, rs.Value)
						return nil
					},
				),
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
