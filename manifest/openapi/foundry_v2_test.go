package openapi

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type testSample struct {
	gvk  schema.GroupVersionKind
	want tftypes.Type
}

type testSamples map[string]testSample

var objectMetaType = tftypes.Object{
	AttributeTypes: map[string]tftypes.Type{
		"annotations":                tftypes.Map{AttributeType: tftypes.String},
		"clusterName":                tftypes.String,
		"creationTimestamp":          tftypes.String,
		"deletionGracePeriodSeconds": tftypes.Number,
		"deletionTimestamp":          tftypes.String,
		"finalizers":                 tftypes.List{ElementType: tftypes.String},
		"generateName":               tftypes.String,
		"generation":                 tftypes.Number,
		"labels":                     tftypes.Map{AttributeType: tftypes.String},
		"managedFields": tftypes.List{
			ElementType: tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"apiVersion": tftypes.String,
					"fieldsType": tftypes.String,
					"fieldsV1":   tftypes.DynamicPseudoType,
					"manager":    tftypes.String,
					"operation":  tftypes.String,
					"time":       tftypes.String,
				},
			},
		},
		"name":      tftypes.String,
		"namespace": tftypes.String,
		"ownerReferences": tftypes.List{
			ElementType: tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"apiVersion":         tftypes.String,
					"blockOwnerDeletion": tftypes.Bool,
					"controller":         tftypes.Bool,
					"kind":               tftypes.String,
					"name":               tftypes.String,
					"uid":                tftypes.String,
				},
			},
		},
		"resourceVersion": tftypes.String,
		"selfLink":        tftypes.String,
		"uid":             tftypes.String,
	},
}

var samples = testSamples{
	"core.v1/ConfigMap": {
		gvk: schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"},
		want: tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"apiVersion": tftypes.String,
				"kind":       tftypes.String,
				"metadata":   objectMetaType,
				"immutable":  tftypes.Bool,
				"data":       tftypes.Map{AttributeType: tftypes.String},
				"binaryData": tftypes.Map{AttributeType: tftypes.String},
			},
		},
	},
	"core.v1/Service": {
		gvk: schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Service"},
		want: tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"apiVersion": tftypes.String,
				"kind":       tftypes.String,
				"metadata":   objectMetaType,
				"spec": tftypes.Object{
					AttributeTypes: map[string]tftypes.Type{
						"clusterIP":                tftypes.String,
						"externalIPs":              tftypes.List{ElementType: tftypes.String},
						"externalName":             tftypes.String,
						"externalTrafficPolicy":    tftypes.String,
						"healthCheckNodePort":      tftypes.Number,
						"ipFamily":                 tftypes.String,
						"loadBalancerIP":           tftypes.String,
						"loadBalancerSourceRanges": tftypes.List{ElementType: tftypes.String},
						"ports": tftypes.List{
							ElementType: tftypes.Object{
								AttributeTypes: map[string]tftypes.Type{
									"appProtocol": tftypes.String,
									"name":        tftypes.String,
									"nodePort":    tftypes.Number,
									"port":        tftypes.Number,
									"protocol":    tftypes.String,
									"targetPort":  tftypes.DynamicPseudoType,
								},
							},
						},
						"publishNotReadyAddresses": tftypes.Bool,
						"selector":                 tftypes.Map{AttributeType: tftypes.String},
						"sessionAffinity":          tftypes.String,
						"sessionAffinityConfig": tftypes.Object{AttributeTypes: map[string]tftypes.Type{
							"clientIP": tftypes.Object{
								AttributeTypes: map[string]tftypes.Type{
									"timeoutSeconds": tftypes.Number,
								},
							},
						}},
						"topologyKeys": tftypes.List{ElementType: tftypes.String},
						"type":         tftypes.String,
					},
				},
				"status": tftypes.Object{AttributeTypes: map[string]tftypes.Type{
					"loadBalancer": tftypes.Object{AttributeTypes: map[string]tftypes.Type{
						"ingress": tftypes.List{ElementType: tftypes.Object{
							AttributeTypes: map[string]tftypes.Type{
								"hostname": tftypes.String,
								"ip":       tftypes.String,
							},
						}},
					}},
				}},
			},
		},
	},
	"apiextensions.v1/CustomResourceDefinition": {
		gvk: schema.GroupVersionKind{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"},
		want: tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"apiVersion": tftypes.String,
				"kind":       tftypes.String,
				"metadata":   objectMetaType,
				"spec": tftypes.Object{AttributeTypes: map[string]tftypes.Type{
					"conversion": tftypes.Object{AttributeTypes: map[string]tftypes.Type{
						"strategy": tftypes.String,
						"webhook": tftypes.Object{AttributeTypes: map[string]tftypes.Type{
							"clientConfig": tftypes.Object{AttributeTypes: map[string]tftypes.Type{
								"caBundle": tftypes.String,
								"service": tftypes.Object{AttributeTypes: map[string]tftypes.Type{
									"name":      tftypes.String,
									"namespace": tftypes.String,
									"path":      tftypes.String,
									"port":      tftypes.Number,
								}},
								"url": tftypes.String,
							}},
							"conversionReviewVersions": tftypes.List{ElementType: tftypes.String},
						}},
					}},
					"group": tftypes.String,
					"names": tftypes.Object{AttributeTypes: map[string]tftypes.Type{
						"categories": tftypes.List{ElementType: tftypes.String},
						"kind":       tftypes.String,
						"listKind":   tftypes.String,
						"plural":     tftypes.String,
						"shortNames": tftypes.List{ElementType: tftypes.String},
						"singular":   tftypes.String,
					}},
					"preserveUnknownFields": tftypes.Bool,
					"scope":                 tftypes.String,
					"versions": tftypes.Tuple{ElementTypes: []tftypes.Type{
						tftypes.Object{
							AttributeTypes: map[string]tftypes.Type{
								"additionalPrinterColumns": tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
									"description": tftypes.String,
									"format":      tftypes.String,
									"jsonPath":    tftypes.String,
									"name":        tftypes.String,
									"priority":    tftypes.Number,
									"type":        tftypes.String,
								}}},
								"deprecated":         tftypes.Bool,
								"deprecationWarning": tftypes.String,
								"name":               tftypes.String,
								"schema": tftypes.Object{AttributeTypes: map[string]tftypes.Type{
									"openAPIV3Schema": tftypes.DynamicPseudoType,
								}},
								"served":  tftypes.Bool,
								"storage": tftypes.Bool,
								"subresources": tftypes.Object{AttributeTypes: map[string]tftypes.Type{
									"scale": tftypes.Object{AttributeTypes: map[string]tftypes.Type{
										"labelSelectorPath":  tftypes.String,
										"specReplicasPath":   tftypes.String,
										"statusReplicasPath": tftypes.String,
									}},
									"status": tftypes.Map{AttributeType: tftypes.String},
								}},
							}}},
					},
				}},
				"status": tftypes.Object{AttributeTypes: map[string]tftypes.Type{
					"acceptedNames": tftypes.Object{AttributeTypes: map[string]tftypes.Type{
						"categories": tftypes.List{ElementType: tftypes.String},
						"kind":       tftypes.String,
						"listKind":   tftypes.String,
						"plural":     tftypes.String,
						"shortNames": tftypes.List{ElementType: tftypes.String},
						"singular":   tftypes.String,
					}},
					"conditions": tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
						"lastTransitionTime": tftypes.String,
						"message":            tftypes.String,
						"reason":             tftypes.String,
						"status":             tftypes.String,
						"type":               tftypes.String,
					}}},
					"storedVersions": tftypes.List{ElementType: tftypes.String},
				}},
			},
		},
	},
}

func TestGetType(t *testing.T) {
	tf, err := buildFixtureFoundry()
	if err != nil {
		t.Skip()
	}
	for name, s := range samples {
		t.Run(name,
			func(t *testing.T) {
				rt, err := tf.GetTypeByGVK(s.gvk)
				if err != nil {
					t.Fatal(fmt.Errorf("GetTypeByID() failed: %s", err))
				}
				if !rt.Is(s.want) {
					t.Fatalf("\nRETURNED %s\nEXPECTED: %s", spew.Sdump(rt), spew.Sdump(s.want))
				}
			})
	}
}

func buildFixtureFoundry() (Foundry, error) {
	sfile := filepath.Join("testdata", "k8s-swagger.json")

	input, err := ioutil.ReadFile(sfile)
	if err != nil {
		return nil, fmt.Errorf("failed to load definition file: %s : %s", sfile, err)
	}

	tf, err := NewFoundryFromSpecV2(input)

	if err != nil {
		return nil, err
	}

	if tf == nil {
		return nil, fmt.Errorf("constructed foundry is nil")
	}

	return tf, nil
}

func TestFoundryOAPIv2(t *testing.T) {
	_, err := buildFixtureFoundry()
	if err != nil {
		t.Error(err)
	}
}

func TestOpenAPIPathFromGVK(t *testing.T) {
	samples := []struct {
		gvk schema.GroupVersionKind
		id  string
	}{
		{
			gvk: schema.GroupVersionKind{
				Group:   "apiextensions.k8s.io",
				Version: "v1beta1",
				Kind:    "CustomResourceDefinition",
			},
			id: "io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1beta1.CustomResourceDefinition",
		},
		{
			gvk: schema.GroupVersionKind{
				Group:   "storage.k8s.io",
				Version: "v1beta1",
				Kind:    "StorageClass",
			},
			id: "io.k8s.api.storage.v1beta1.StorageClass",
		},
		{
			gvk: schema.GroupVersionKind{
				Group:   "apiregistration.k8s.io",
				Version: "v1",
				Kind:    "APIService",
			},
			id: "io.k8s.kube-aggregator.pkg.apis.apiregistration.v1.APIService",
		},
		{
			gvk: schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "Namespace",
			},
			id: "io.k8s.api.core.v1.Namespace",
		},
	}

	tf, err := buildFixtureFoundry()
	if err != nil {
		t.Skip()
	}
	for _, s := range samples {
		id, ok := (tf.(*foapiv2)).gkvIndex.Load(s.gvk)
		if !ok {
			t.Fatal(err)
		}
		if strings.Compare(id.(string), s.id) != 0 {
			t.Fatalf("IDs don't match\n\tWant:\t%s\n\tGot:\t%s", s.id, id)
		}
	}
}
