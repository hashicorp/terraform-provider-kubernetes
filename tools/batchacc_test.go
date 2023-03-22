// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestTokenizeCamelCase(t *testing.T) {
	samples := map[string][]string{
		"HelloWorld":                                       {"Hello", "World"},
		"hello-world":                                      {"hello-world"},
		"TestAccKubernetesIngress_TLS":                     {"Test", "Acc", "Kubernetes", "Ingress", "TLS"},
		"TestAccKubernetesCSIDriver_basic":                 {"Test", "Acc", "Kubernetes", "CSIDriver", "basic"},
		"TestAccKubernetesAPIService_basic":                {"Test", "Acc", "Kubernetes", "APIService", "basic"},
		"TestAccKubernetesCertificateSigningRequest_basic": {"Test", "Acc", "Kubernetes", "Certificate", "Signing", "Request", "basic"},
		"TestAccKubernetesPod_with_node_affinity_with_required_during_scheduling_ignored_during_execution": {"Test", "Acc", "Kubernetes", "Pod", "with", "node", "affinity", "with", "required", "during", "scheduling", "ignored", "during", "execution"},
	}
	for s, r := range samples {
		t.Run(s, func(t *testing.T) {
			res := tokenizeCamelCase(s)
			fail := len(res) != len(r)
			for ri := range res {
				if fail {
					break
				}
				fail = res[ri] != r[ri]
			}
			if fail {
				t.Errorf("Sample '%s' failed.\n\tWanted:\t%s\n\tActual:\t%v\n", s, r, res)
			}
		})
	}
}

func TestAddString(t *testing.T) {
	samples := []struct {
		In  []string
		Out PrefixTree
	}{
		{
			In: []string{
				"HelloWorld",
				"HelloWonderfulWorld",
				"HelloWorldWonder",
				"GoodbyeCruelWorld",
			},
			Out: PrefixTree{
				"Hello": {
					"World": {
						"Wonder": {},
					},
					"Wonderful": {
						"World": {},
					},
				},
				"Goodbye": {
					"Cruel": {
						"World": {},
					},
				},
			},
		},
		{
			In: []string{
				"TestAccKubernetesCSIDriver_basic",
				"TestAccKubernetesAPIService_basic",
				"TestAccKubernetesCertificateSigningRequest_basic",
				"TestAccKubernetesClusterRole_basic",
			},
			Out: PrefixTree{
				"Test": {
					"Acc": {
						"Kubernetes": {
							"CSIDriver": {
								"basic": {},
							},
							"APIService": {
								"basic": {},
							},
							"Certificate": {
								"Signing": {
									"Request": {
										"basic": {},
									},
								},
							},
							"Cluster": {
								"Role": {
									"basic": {},
								},
							},
						},
					},
				},
			},
		},
	}
	for k, s := range samples {
		tt := PrefixTree{}
		for _, w := range s.In {
			tt.addString(w)
		}
		if !cmp.Equal(s.Out, tt) {
			t.Errorf("Sample %d failed.\n\tWanted:\t%v\n\tActual:\t%v\n", k, s.Out, tt)
		}

	}
}

func TestPrefixesToDepth(t *testing.T) {
	type depthSample struct {
		D int
		P []string
	}
	samples := []struct {
		In  []string
		Out []depthSample
	}{
		{
			In: []string{
				"HelloWorld",
				"HelloWonderfulWorld",
				"HelloWorldWonder",
				"GoodbyeCruelWorld",
			},
			Out: []depthSample{
				{
					D: 1,
					P: []string{"HelloWorld", "HelloWonderful", "GoodbyeCruel"},
				},
			},
		},
		{
			In: []string{
				"TestAccKubernetesCSIDriver_basic",
				"TestAccKubernetesAPIService_basic",
				"TestAccKubernetesCertificateSigningRequest_basic",
				"TestAccKubernetesClusterRole_basic",
			},
			Out: []depthSample{
				{
					D: 3,
					P: []string{"TestAccKubernetesCSIDriver", "TestAccKubernetesAPIService", "TestAccKubernetesCertificate", "TestAccKubernetesCluster"},
				},
			},
		},
	}
	for _, s := range samples {
		tt := PrefixTree{}
		for _, w := range s.In {
			tt.addString(w)
		}
		for _, ds := range s.Out {
			out := tt.prefixesToDepth(ds.D)
			sort.Strings(out)
			sort.Strings(ds.P)
			if !cmp.Equal(out, ds.P) {
				t.Errorf("Sample for depth %d failed!\n\tWanted:%v\n\tActual:%v", ds.D, ds.P, out)
			}
		}
	}
}
