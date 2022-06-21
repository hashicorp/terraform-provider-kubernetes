package schema

import (
	"context"
	"reflect"
	"testing"
)

func TestUpgradeTemplatePodSpecWithResourcesFieldV0(t *testing.T) {
	v0 := map[string]interface{}{
		"spec": []interface{}{map[string]interface{}{
			"template": []interface{}{map[string]interface{}{
				"spec": []interface{}{map[string]interface{}{
					"init_container": []interface{}{
						map[string]interface{}{
							"resources": []interface{}{map[string]interface{}{
								"requests": []interface{}{map[string]interface{}{
									"cpu":    "500m",
									"memory": "56Mi",
								}},
								"limits": []interface{}{map[string]interface{}{
									"cpu":    "750m",
									"memory": "128Mi",
								}},
							}},
						},
						map[string]interface{}{
							"resources": []interface{}{map[string]interface{}{
								"requests": []interface{}{map[string]interface{}{
									"cpu":    "200m",
									"memory": "16Mi",
								}},
								"limits": []interface{}{map[string]interface{}{
									"cpu":    "300m",
									"memory": "32Mi",
								}},
							}},
						},
					},
					"container": []interface{}{
						map[string]interface{}{
							"resources": []interface{}{map[string]interface{}{
								"requests": []interface{}{map[string]interface{}{
									"cpu":    "500m",
									"memory": "56Mi",
								}},
								"limits": []interface{}{map[string]interface{}{
									"cpu":    "750m",
									"memory": "128Mi",
								}},
							}},
						},
						map[string]interface{}{
							"resources": []interface{}{map[string]interface{}{
								"requests": []interface{}{map[string]interface{}{
									"cpu":    "200m",
									"memory": "16Mi",
								}},
								"limits": []interface{}{map[string]interface{}{
									"cpu":    "300m",
									"memory": "32Mi",
								}},
							}},
						},
					},
				}},
			}},
		}},
	}

	v1 := map[string]interface{}{
		"spec": []interface{}{map[string]interface{}{
			"template": []interface{}{map[string]interface{}{
				"spec": []interface{}{map[string]interface{}{
					"init_container": []interface{}{
						map[string]interface{}{
							"resources": []interface{}{map[string]interface{}{
								"requests": map[string]interface{}{
									"cpu":    "500m",
									"memory": "56Mi",
								},
								"limits": map[string]interface{}{
									"cpu":    "750m",
									"memory": "128Mi",
								},
							}},
						},
						map[string]interface{}{
							"resources": []interface{}{map[string]interface{}{
								"requests": map[string]interface{}{
									"cpu":    "200m",
									"memory": "16Mi",
								},
								"limits": map[string]interface{}{
									"cpu":    "300m",
									"memory": "32Mi",
								},
							}},
						},
					},
					"container": []interface{}{
						map[string]interface{}{
							"resources": []interface{}{map[string]interface{}{
								"requests": map[string]interface{}{
									"cpu":    "500m",
									"memory": "56Mi",
								},
								"limits": map[string]interface{}{
									"cpu":    "750m",
									"memory": "128Mi",
								},
							}},
						},
						map[string]interface{}{
							"resources": []interface{}{map[string]interface{}{
								"requests": map[string]interface{}{
									"cpu":    "200m",
									"memory": "16Mi",
								},
								"limits": map[string]interface{}{
									"cpu":    "300m",
									"memory": "32Mi",
								},
							}},
						},
					},
				}},
			}},
		}},
	}

	actual, _ := UpgradeTemplatePodSpecWithResourcesFieldV0(context.TODO(), v0, nil)

	if !reflect.DeepEqual(v1, actual) {
		t.Fatalf("\n\nexpected:\n\n%#v\n\ngot:\n\n%#v\n\n", v1, actual)
	}
}

func TestUpgradeTemplatePodSpecWithResourcesFieldV0_empty(t *testing.T) {
	v0 := map[string]interface{}{
		"spec": []interface{}{map[string]interface{}{
			"template": []interface{}{map[string]interface{}{
				"spec": []interface{}{map[string]interface{}{
					"init_container": []interface{}{
						map[string]interface{}{
							"resources": []interface{}{map[string]interface{}{
								"requests": []interface{}{},
								"limits":   []interface{}{},
							}},
						},
					},
					"container": []interface{}{
						map[string]interface{}{
							"resources": []interface{}{map[string]interface{}{
								"requests": []interface{}{},
								"limits":   []interface{}{},
							}},
						},
					},
				}},
			}},
		}},
	}

	v1 := map[string]interface{}{
		"spec": []interface{}{map[string]interface{}{
			"template": []interface{}{map[string]interface{}{
				"spec": []interface{}{map[string]interface{}{
					"init_container": []interface{}{
						map[string]interface{}{
							"resources": []interface{}{map[string]interface{}{
								"requests": map[string]interface{}{},
								"limits":   map[string]interface{}{},
							}},
						},
					},
					"container": []interface{}{
						map[string]interface{}{
							"resources": []interface{}{map[string]interface{}{
								"requests": map[string]interface{}{},
								"limits":   map[string]interface{}{},
							}},
						},
					},
				}},
			}},
		}},
	}

	actual, _ := UpgradeTemplatePodSpecWithResourcesFieldV0(context.TODO(), v0, nil)

	if !reflect.DeepEqual(v1, actual) {
		t.Fatalf("\n\nexpected:\n\n%#v\n\ngot:\n\n%#v\n\n", v1, actual)
	}
}
