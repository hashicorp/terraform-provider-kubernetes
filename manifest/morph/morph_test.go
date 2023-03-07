// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package morph

import (
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestMorphValueToType(t *testing.T) {
	type sampleInType struct {
		V tftypes.Value
		T tftypes.Type
	}
	samples := map[string]struct {
		In      sampleInType
		Out     tftypes.Value
		WantErr bool
	}{
		"string->string": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.String, "hello"),
				T: tftypes.String,
			},
			Out: tftypes.NewValue(tftypes.String, "hello"),
		},
		"string->number": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.String, "12.4"),
				T: tftypes.Number,
			},
			Out: tftypes.NewValue(tftypes.Number, new(big.Float).SetFloat64(12.4)),
		},
		"string->bool": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.String, "true"),
				T: tftypes.Bool,
			},
			Out: tftypes.NewValue(tftypes.Bool, true),
		},
		"number->number": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Number, new(big.Float).SetFloat64(12.4)),
				T: tftypes.Number,
			},
			Out: tftypes.NewValue(tftypes.Number, new(big.Float).SetFloat64(12.4)),
		},
		"number->string": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Number, new(big.Float).SetFloat64(12.4)),
				T: tftypes.String,
			},
			Out: tftypes.NewValue(tftypes.String, "12.4"),
		},
		"bool->bool": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Bool, true),
				T: tftypes.Bool,
			},
			Out: tftypes.NewValue(tftypes.Bool, true),
		},
		"bool->string": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Bool, true),
				T: tftypes.String,
			},
			Out: tftypes.NewValue(tftypes.String, "true"),
		},
		"list->list": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
					tftypes.NewValue(tftypes.String, "foo"),
					tftypes.NewValue(tftypes.String, "bar"),
					tftypes.NewValue(tftypes.String, "baz"),
				}),
				T: tftypes.List{ElementType: tftypes.String},
			},
			Out: tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "foo"),
				tftypes.NewValue(tftypes.String, "bar"),
				tftypes.NewValue(tftypes.String, "baz"),
			}),
		},
		"list->tuple": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
					tftypes.NewValue(tftypes.String, "foo"),
					tftypes.NewValue(tftypes.String, "bar"),
					tftypes.NewValue(tftypes.String, "baz"),
				}),
				T: tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String, tftypes.String, tftypes.String}},
			},
			Out: tftypes.NewValue(tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String, tftypes.String, tftypes.String}}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "foo"),
				tftypes.NewValue(tftypes.String, "bar"),
				tftypes.NewValue(tftypes.String, "baz"),
			}),
		},
		"list->set": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
					tftypes.NewValue(tftypes.String, "10"),
					tftypes.NewValue(tftypes.String, "11.9"),
					tftypes.NewValue(tftypes.String, "42"),
				}),
				T: tftypes.Set{ElementType: tftypes.Number},
			},
			Out: tftypes.NewValue(tftypes.Set{ElementType: tftypes.Number}, []tftypes.Value{
				tftypes.NewValue(tftypes.Number, new(big.Float).SetFloat64(10)),
				tftypes.NewValue(tftypes.Number, new(big.Float).SetFloat64(11.9)),
				tftypes.NewValue(tftypes.Number, new(big.Float).SetFloat64(42)),
			}),
		},
		"tuple->tuple": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String, tftypes.String, tftypes.String}}, []tftypes.Value{
					tftypes.NewValue(tftypes.String, "foo"),
					tftypes.NewValue(tftypes.String, "bar"),
					tftypes.NewValue(tftypes.String, "baz"),
				}),
				T: tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String, tftypes.String, tftypes.String}},
			},
			Out: tftypes.NewValue(tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String, tftypes.String, tftypes.String}}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "foo"),
				tftypes.NewValue(tftypes.String, "bar"),
				tftypes.NewValue(tftypes.String, "baz"),
			}),
		},
		// This covers the case were we need to represent lists that contain dynamicPseudoType sub-elements
		// because the dynamicPseudoType might hold heterogenous types
		"tuple(single)->tuple": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String, tftypes.String, tftypes.String}}, []tftypes.Value{
					tftypes.NewValue(tftypes.String, "foo"),
					tftypes.NewValue(tftypes.String, "bar"),
					tftypes.NewValue(tftypes.String, "baz"),
				}),
				T: tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.DynamicPseudoType}},
			},
			Out: tftypes.NewValue(tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String, tftypes.String, tftypes.String}}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "foo"),
				tftypes.NewValue(tftypes.String, "bar"),
				tftypes.NewValue(tftypes.String, "baz"),
			}),
		},
		"tuple->list": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String, tftypes.String, tftypes.String}}, []tftypes.Value{
					tftypes.NewValue(tftypes.String, "foo"),
					tftypes.NewValue(tftypes.String, "bar"),
					tftypes.NewValue(tftypes.String, "baz"),
				}),
				T: tftypes.List{ElementType: tftypes.String},
			},
			Out: tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "foo"),
				tftypes.NewValue(tftypes.String, "bar"),
				tftypes.NewValue(tftypes.String, "baz"),
			}),
		},
		"tuple->set": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String, tftypes.String, tftypes.String}}, []tftypes.Value{
					tftypes.NewValue(tftypes.String, "foo"),
					tftypes.NewValue(tftypes.String, "bar"),
					tftypes.NewValue(tftypes.String, "baz"),
				}),
				T: tftypes.Set{ElementType: tftypes.String},
			},
			Out: tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "foo"),
				tftypes.NewValue(tftypes.String, "bar"),
				tftypes.NewValue(tftypes.String, "baz"),
			}),
		},
		"tuple(object)->list(object)": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Tuple{ElementTypes: []tftypes.Type{
					tftypes.Object{AttributeTypes: map[string]tftypes.Type{"foo": tftypes.String}},
					tftypes.Object{AttributeTypes: map[string]tftypes.Type{"foo": tftypes.DynamicPseudoType}},
					tftypes.Object{AttributeTypes: map[string]tftypes.Type{"foo": tftypes.String}},
				}},
					[]tftypes.Value{
						tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{"foo": tftypes.String}},
							map[string]tftypes.Value{"foo": tftypes.NewValue(tftypes.String, "foo")}),
						tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{"foo": tftypes.DynamicPseudoType}},
							map[string]tftypes.Value{"foo": tftypes.NewValue(tftypes.DynamicPseudoType, nil)}),
						tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{"foo": tftypes.String}},
							map[string]tftypes.Value{"foo": tftypes.NewValue(tftypes.String, "baz")}),
					}),
				T: tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{"foo": tftypes.String}}},
			},
			Out: tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{"foo": tftypes.String}}}, []tftypes.Value{
				tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{"foo": tftypes.String}},
					map[string]tftypes.Value{"foo": tftypes.NewValue(tftypes.String, "foo")}),
				tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{"foo": tftypes.String}},
					map[string]tftypes.Value{"foo": tftypes.NewValue(tftypes.String, nil)}),
				tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{"foo": tftypes.String}},
					map[string]tftypes.Value{"foo": tftypes.NewValue(tftypes.String, "baz")}),
			}),
		},
		"tuple(object)->tuple(object)": {
			In: sampleInType{
				V: tftypes.NewValue(
					tftypes.Tuple{ElementTypes: []tftypes.Type{
						tftypes.Object{AttributeTypes: map[string]tftypes.Type{"first": tftypes.String}},
						tftypes.Object{AttributeTypes: map[string]tftypes.Type{"second": tftypes.DynamicPseudoType}},
						tftypes.Object{AttributeTypes: map[string]tftypes.Type{"third": tftypes.Tuple{ElementTypes: []tftypes.Type{
							tftypes.Object{AttributeTypes: map[string]tftypes.Type{"foo": tftypes.String}},
							tftypes.Object{AttributeTypes: map[string]tftypes.Type{"bar": tftypes.String}},
						}},
						}},
					}},
					[]tftypes.Value{
						tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{"first": tftypes.String}},
							map[string]tftypes.Value{"first": tftypes.NewValue(tftypes.String, "foo")}),

						tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{"second": tftypes.DynamicPseudoType}},
							map[string]tftypes.Value{"second": tftypes.NewValue(tftypes.DynamicPseudoType, nil)}),

						tftypes.NewValue(
							tftypes.Object{AttributeTypes: map[string]tftypes.Type{
								"third": tftypes.Tuple{ElementTypes: []tftypes.Type{
									tftypes.Object{AttributeTypes: map[string]tftypes.Type{"foo": tftypes.String}},
									tftypes.Object{AttributeTypes: map[string]tftypes.Type{"bar": tftypes.String}},
								}},
							}},
							map[string]tftypes.Value{
								"third": tftypes.NewValue(tftypes.Tuple{ElementTypes: []tftypes.Type{
									tftypes.Object{AttributeTypes: map[string]tftypes.Type{"foo": tftypes.String}},
									tftypes.Object{AttributeTypes: map[string]tftypes.Type{"bar": tftypes.String}},
								}}, []tftypes.Value{
									tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{"foo": tftypes.String}}, map[string]tftypes.Value{"foo": tftypes.NewValue(tftypes.String, "some")}),
									tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{"bar": tftypes.String}}, map[string]tftypes.Value{"bar": tftypes.NewValue(tftypes.String, "other")}),
								}),
							},
						),
					},
				),
				T: tftypes.Tuple{ElementTypes: []tftypes.Type{
					tftypes.Object{AttributeTypes: map[string]tftypes.Type{"first": tftypes.String}},
					tftypes.Object{AttributeTypes: map[string]tftypes.Type{"second": tftypes.DynamicPseudoType}},
					tftypes.Object{AttributeTypes: map[string]tftypes.Type{"third": tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.Object{AttributeTypes: map[string]tftypes.Type{"foo": tftypes.String, "bar": tftypes.String}}}}}},
				}},
			},

			Out: tftypes.NewValue(tftypes.Tuple{ElementTypes: []tftypes.Type{
				tftypes.Object{AttributeTypes: map[string]tftypes.Type{"first": tftypes.String}},
				tftypes.Object{AttributeTypes: map[string]tftypes.Type{"second": tftypes.DynamicPseudoType}},
				tftypes.Object{AttributeTypes: map[string]tftypes.Type{"third": tftypes.Tuple{ElementTypes: []tftypes.Type{
					tftypes.Object{AttributeTypes: map[string]tftypes.Type{"foo": tftypes.String, "bar": tftypes.String}},
					tftypes.Object{AttributeTypes: map[string]tftypes.Type{"foo": tftypes.String, "bar": tftypes.String}},
				}}}},
			}},
				[]tftypes.Value{
					tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{"first": tftypes.String}},
						map[string]tftypes.Value{"first": tftypes.NewValue(tftypes.String, "foo")}),

					tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{"second": tftypes.DynamicPseudoType}},
						map[string]tftypes.Value{"second": tftypes.NewValue(tftypes.String, nil)}),

					tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{"third": tftypes.Tuple{ElementTypes: []tftypes.Type{
						tftypes.Object{AttributeTypes: map[string]tftypes.Type{"foo": tftypes.String, "bar": tftypes.String}},
						tftypes.Object{AttributeTypes: map[string]tftypes.Type{"foo": tftypes.String, "bar": tftypes.String}},
					}}}},
						map[string]tftypes.Value{"third": tftypes.NewValue(tftypes.Tuple{ElementTypes: []tftypes.Type{
							tftypes.Object{AttributeTypes: map[string]tftypes.Type{"foo": tftypes.String, "bar": tftypes.String}},
							tftypes.Object{AttributeTypes: map[string]tftypes.Type{"foo": tftypes.String, "bar": tftypes.String}},
						}},
							[]tftypes.Value{
								tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{"foo": tftypes.String, "bar": tftypes.String}},
									map[string]tftypes.Value{
										"foo": tftypes.NewValue(tftypes.String, "some"),
										"bar": tftypes.NewValue(tftypes.String, nil),
									}),
								tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{"foo": tftypes.String, "bar": tftypes.String}},
									map[string]tftypes.Value{
										"foo": tftypes.NewValue(tftypes.String, nil),
										"bar": tftypes.NewValue(tftypes.String, "other"),
									}),
							},
						)},
					),
				}),
		},
		"set->tuple": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{
					tftypes.NewValue(tftypes.String, "foo"),
					tftypes.NewValue(tftypes.String, "bar"),
					tftypes.NewValue(tftypes.String, "baz"),
				}),
				T: tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String, tftypes.String, tftypes.String}},
			},
			Out: tftypes.NewValue(tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String, tftypes.String, tftypes.String}}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "foo"),
				tftypes.NewValue(tftypes.String, "bar"),
				tftypes.NewValue(tftypes.String, "baz"),
			}),
		},
		"set->list": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{
					tftypes.NewValue(tftypes.String, "foo"),
					tftypes.NewValue(tftypes.String, "bar"),
					tftypes.NewValue(tftypes.String, "baz"),
				}),
				T: tftypes.List{ElementType: tftypes.String},
			},
			Out: tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "foo"),
				tftypes.NewValue(tftypes.String, "bar"),
				tftypes.NewValue(tftypes.String, "baz"),
			}),
		},
		"map->object": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{
					"one":   tftypes.NewValue(tftypes.String, "foo"),
					"two":   tftypes.NewValue(tftypes.String, "bar"),
					"three": tftypes.NewValue(tftypes.String, "baz"),
				}),
				T: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
					"one":   tftypes.String,
					"two":   tftypes.String,
					"three": tftypes.String,
				}},
			},
			Out: tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"one":   tftypes.String,
				"two":   tftypes.String,
				"three": tftypes.String,
			}}, map[string]tftypes.Value{
				"one":   tftypes.NewValue(tftypes.String, "foo"),
				"two":   tftypes.NewValue(tftypes.String, "bar"),
				"three": tftypes.NewValue(tftypes.String, "baz"),
			}),
		},
		"map->map": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{
					"one":   tftypes.NewValue(tftypes.String, "foo"),
					"two":   tftypes.NewValue(tftypes.String, "bar"),
					"three": tftypes.NewValue(tftypes.String, "baz"),
				}),
				T: tftypes.Map{ElementType: tftypes.String},
			},
			Out: tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{
				"one":   tftypes.NewValue(tftypes.String, "foo"),
				"two":   tftypes.NewValue(tftypes.String, "bar"),
				"three": tftypes.NewValue(tftypes.String, "baz"),
			}),
		},
		"object->map": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{
					"one":   tftypes.String,
					"two":   tftypes.String,
					"three": tftypes.String,
				}}, map[string]tftypes.Value{
					"one":   tftypes.NewValue(tftypes.String, "foo"),
					"two":   tftypes.NewValue(tftypes.String, "bar"),
					"three": tftypes.NewValue(tftypes.String, "baz"),
				}),
				T: tftypes.Map{ElementType: tftypes.String},
			},
			Out: tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{
				"one":   tftypes.NewValue(tftypes.String, "foo"),
				"two":   tftypes.NewValue(tftypes.String, "bar"),
				"three": tftypes.NewValue(tftypes.String, "baz"),
			}),
		},
		"object->object": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{
					"one":   tftypes.String,
					"two":   tftypes.String,
					"three": tftypes.String,
				}}, map[string]tftypes.Value{
					"one":   tftypes.NewValue(tftypes.String, "foo"),
					"two":   tftypes.NewValue(tftypes.String, "bar"),
					"three": tftypes.NewValue(tftypes.String, "baz"),
				}),
				T: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
					"one":   tftypes.String,
					"two":   tftypes.String,
					"three": tftypes.String,
				}},
			},
			Out: tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"one":   tftypes.String,
				"two":   tftypes.String,
				"three": tftypes.String,
			}}, map[string]tftypes.Value{
				"one":   tftypes.NewValue(tftypes.String, "foo"),
				"two":   tftypes.NewValue(tftypes.String, "bar"),
				"three": tftypes.NewValue(tftypes.String, "baz"),
			}),
		},

		// Testcases to demonstrate https://github.com/hashicorp/terraform-provider-kubernetes-alpha/issues/190
		"string(unknown value)->string": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
				T: tftypes.String,
			},
			Out: tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		},
		"number(unkown value)->number": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Number, tftypes.UnknownValue),
				T: tftypes.Number,
			},
			Out: tftypes.NewValue(tftypes.Number, tftypes.UnknownValue),
		},
		"bool(unkown value)->bool": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue),
				T: tftypes.Bool,
			},
			Out: tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue),
		},

		// Translations that won't work without the values.
		"number(unkown value)->string": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Number, tftypes.UnknownValue),
				T: tftypes.String,
			},
			WantErr: true,
		},
		"string(unkown value)->number": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
				T: tftypes.Number,
			},
			WantErr: true,
		},
		"bool(unkown value)->string": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue),
				T: tftypes.String,
			},
			WantErr: true,
		},
		"string(unkown value)->bool": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
				T: tftypes.Bool,
			},
			WantErr: true,
		},
		"object -> object": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{
					"two":   tftypes.String,
					"three": tftypes.String,
				}}, map[string]tftypes.Value{
					"two":   tftypes.NewValue(tftypes.String, "bar"),
					"three": tftypes.NewValue(tftypes.String, "baz"),
				}),
				T: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
					"one":   tftypes.String,
					"two":   tftypes.String,
					"three": tftypes.String,
				}},
			},
			Out: tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"one":   tftypes.String,
				"two":   tftypes.String,
				"three": tftypes.String,
			}}, map[string]tftypes.Value{
				"one":   tftypes.NewValue(tftypes.String, nil),
				"two":   tftypes.NewValue(tftypes.String, "bar"),
				"three": tftypes.NewValue(tftypes.String, "baz"),
			}),
		},
		// morphing to tuple attributes to "template tuples" (containing dynamic) should result in the same number of elements as the input
		"object(dynamic) -> object": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{
					"one": tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String, tftypes.String}},
				}}, map[string]tftypes.Value{
					"one": tftypes.NewValue(tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String, tftypes.String}},
						[]tftypes.Value{
							tftypes.NewValue(tftypes.String, "bar"),
							tftypes.NewValue(tftypes.String, "baz"),
						}),
				}),
				T: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
					"one": tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.DynamicPseudoType}},
				}},
			},
			Out: tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"one": tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String, tftypes.String}},
			}}, map[string]tftypes.Value{
				"one": tftypes.NewValue(tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String, tftypes.String}},
					[]tftypes.Value{
						tftypes.NewValue(tftypes.String, "bar"),
						tftypes.NewValue(tftypes.String, "baz"),
					}),
			}),
		},
	}
	for n, s := range samples {
		t.Run(n, func(t *testing.T) {
			r, err := ValueToType(s.In.V, s.In.T, tftypes.NewAttributePath())
			if len(err) > 0 {
				if !s.WantErr {
					t.Logf("Failed type-morphing for sample '%s'", n)
					for i := range err {
						t.Logf("[%s] %s\n%s\n", err[i].Severity.String(), err[i].Summary, err[i].Detail)
					}
					t.FailNow()
				}
				return
			}
			if !cmp.Equal(r, s.Out, cmp.Exporter(func(t reflect.Type) bool { return true })) {
				t.Logf("Result doesn't match expectation for sample '%s'", n)
				t.Logf("\t Sample:\t%#v", s.In)
				t.Logf("\t Expected:\t%#v", s.Out)
				t.Logf("\t Received:\t%#v", r)
				t.Fail()
			}
		})
	}
}

func TestMorphValueToTypeDiagnostics(t *testing.T) {
	type sampleInType struct {
		V tftypes.Value
		T tftypes.Type
	}
	samples := map[string]struct {
		In        sampleInType
		Attribute *tftypes.AttributePath
		Diags     []*tfprotov5.Diagnostic
	}{
		"string->number": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.String, "12-4"),
				T: tftypes.Number,
			},
			Attribute: tftypes.NewAttributePath().WithAttributeName("foo").WithElementKeyString("bar"),
			Diags: []*tfprotov5.Diagnostic{
				{
					Severity:  tfprotov5.DiagnosticSeverityError,
					Summary:   "String value doesn't parse as Number",
					Detail:    "Error: strconv.ParseFloat: parsing \"12-4\": invalid syntax\n...at attribute:\nfoo[bar]",
					Attribute: tftypes.NewAttributePath().WithAttributeName("foo").WithElementKeyString("bar"),
				},
			},
		},
		"string->bool": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.String, "meh"),
				T: tftypes.Bool,
			},
			Attribute: tftypes.NewAttributePath().WithAttributeName("foo").WithElementKeyInt(42),
			Diags: []*tfprotov5.Diagnostic{
				{
					Severity:  tfprotov5.DiagnosticSeverityError,
					Summary:   "String value doesn't parse as Boolean",
					Detail:    "Error: strconv.ParseBool: parsing \"meh\": invalid syntax\n...at attribute:\nfoo[42]",
					Attribute: tftypes.NewAttributePath().WithAttributeName("foo").WithElementKeyInt(42),
				},
			},
		},
		"bool->number": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Bool, true),
				T: tftypes.Number,
			},
			Diags: []*tfprotov5.Diagnostic{
				{
					Severity: tfprotov5.DiagnosticSeverityError,
					Summary:  "Value incompatible with expected type",
					Detail:   "Cannot convert Bool values into type Number\n ...at attribute\n",
				},
			},
		},
		"number->set[number]": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Number, new(big.Float).SetFloat64(12.4)),
				T: tftypes.Set{ElementType: tftypes.Number},
			},
			Diags: []*tfprotov5.Diagnostic{
				{
					Severity: tfprotov5.DiagnosticSeverityError,
					Summary:  "Value incompatible with expected type",
					Detail:   "Cannot convert Number values into type Set[Number]\n ...at attribute\n",
				},
			},
		},
		"list[string]->list[number]": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
					tftypes.NewValue(tftypes.String, "10.3"),
					tftypes.NewValue(tftypes.String, "-42"),
					tftypes.NewValue(tftypes.String, "baz"),
				}),
				T: tftypes.List{ElementType: tftypes.Number},
			},
			Diags: []*tfprotov5.Diagnostic{
				{
					Severity:  tfprotov5.DiagnosticSeverityError,
					Summary:   "String value doesn't parse as Number",
					Detail:    "Error: strconv.ParseFloat: parsing \"baz\": invalid syntax\n...at attribute:\n[2]",
					Attribute: tftypes.NewAttributePath().WithElementKeyInt(2),
				},
				{
					Severity:  tfprotov5.DiagnosticSeverityError,
					Summary:   "Invalid List value element",
					Detail:    "Error at attribute:\n[2]",
					Attribute: tftypes.NewAttributePath().WithElementKeyInt(2),
				},
			},
		},
		"object -> object": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{
					"two":   tftypes.String,
					"three": tftypes.String,
				}}, map[string]tftypes.Value{
					"two":   tftypes.NewValue(tftypes.String, "bar"),
					"three": tftypes.NewValue(tftypes.String, "fourtytwo"),
				}),
				T: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
					"one":   tftypes.Number,
					"two":   tftypes.String,
					"three": tftypes.Number,
				}},
			},
			Diags: []*tfprotov5.Diagnostic{
				{
					Severity:  tfprotov5.DiagnosticSeverityError,
					Summary:   "String value doesn't parse as Number",
					Detail:    "Error: strconv.ParseFloat: parsing \"fourtytwo\": invalid syntax\n...at attribute:\nthree",
					Attribute: tftypes.NewAttributePath().WithAttributeName("three"),
				},
				{
					Severity:  tfprotov5.DiagnosticSeverityError,
					Summary:   "Failed to transform Object element into Object element type",
					Detail:    "Error (see above) at attribute:\nthree",
					Attribute: tftypes.NewAttributePath(),
				},
			},
		},
		"object -> object (deep)": {
			In: sampleInType{
				V: tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{
					"two": tftypes.String,
					"three": tftypes.Object{AttributeTypes: map[string]tftypes.Type{
						"foo": tftypes.String,
						"bar": tftypes.Number,
					}},
				}}, map[string]tftypes.Value{
					"two": tftypes.NewValue(tftypes.String, "stuff"),
					"three": tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{
						"foo": tftypes.String,
						"bar": tftypes.Number,
					}}, map[string]tftypes.Value{
						"foo": tftypes.NewValue(tftypes.String, "fourtytwo"),
						"bar": tftypes.NewValue(tftypes.Number, 42),
					}),
				}),
				T: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
					"one":   tftypes.Number,
					"two":   tftypes.String,
					"three": tftypes.Object{AttributeTypes: map[string]tftypes.Type{"foo": tftypes.String}},
				}},
			},
			Diags: []*tfprotov5.Diagnostic{
				{
					Severity:  tfprotov5.DiagnosticSeverityWarning,
					Summary:   "Attribute not found in schema",
					Detail:    "Unable to find schema type for attribute:\nthree.bar",
					Attribute: tftypes.NewAttributePath().WithAttributeName("three"),
				},
			},
		},
	}
	for n, s := range samples {
		t.Run(n, func(t *testing.T) {
			_, err := ValueToType(s.In.V, s.In.T, s.Attribute)
			if len(err) != len(s.Diags) {
				t.Logf("Unexpected type-morphing diagnostics for sample '%s'", n)
				for i := range err {
					t.Logf("[%s] %s\n%s\n", err[i].Severity.String(), err[i].Summary, err[i].Detail)
				}
				t.FailNow()
				return
			}
			if !cmp.Equal(err, s.Diags, cmp.Exporter(func(t reflect.Type) bool { return true })) {
				t.Logf("Result doesn't match expectation for sample '%s'", n)
				t.Logf("\t Sample:\t%#v", s.In)
				t.Logf("\t Received:\t%#+v", formatDiagnostics(err))
				t.Fail()
			}
		})
	}
}

func formatDiagnostics(diags []*tfprotov5.Diagnostic) string {
	var b strings.Builder
	for _, d := range diags {
		b.WriteString(fmt.Sprintf("<Severity> [%s] <Summary> [%s] <Detail> [%s] <Attribute> [%s]\n", d.Severity, d.Summary, d.Detail, d.Attribute))
	}
	return b.String()
}
