// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package payload

import (
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var cmpCompareAllOption cmp.Option = cmp.Exporter(func(t reflect.Type) bool { return true })

type sampleInType struct {
	v interface{}
	t tftypes.Type
}

func TestToTFValue(t *testing.T) {
	samples := map[string]struct {
		In  sampleInType
		Th  map[string]string
		Out tftypes.Value
		Err error
	}{
		"string": {
			In:  sampleInType{v: "foobar", t: tftypes.String},
			Out: tftypes.NewValue(tftypes.String, "foobar"),
			Err: nil,
		},
		"string-nil": {
			In:  sampleInType{v: "foobar", t: nil},
			Out: tftypes.Value{},
			Err: tftypes.NewAttributePath().NewErrorf("[] type cannot be nil"),
		},
		"string-pseudotype": {
			In:  sampleInType{v: "foobar", t: tftypes.DynamicPseudoType},
			Out: tftypes.NewValue(tftypes.String, "foobar"),
			Err: nil,
		},
		"boolean": {
			In:  sampleInType{v: true, t: tftypes.Bool},
			Out: tftypes.NewValue(tftypes.Bool, true),
			Err: nil,
		},
		"boolean-pseudotype": {
			In:  sampleInType{v: true, t: tftypes.DynamicPseudoType},
			Out: tftypes.NewValue(tftypes.Bool, true),
			Err: nil,
		},
		"integer": {
			In:  sampleInType{v: int64(100), t: tftypes.Number},
			Out: tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(100)),
			Err: nil,
		},
		"integer-pseudotype": {
			In:  sampleInType{v: int64(100), t: tftypes.DynamicPseudoType},
			Out: tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(100)),
			Err: nil,
		},
		"string-integer": {
			In:  sampleInType{v: "42", t: tftypes.Number},
			Out: tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(42)),
			Err: nil,
		},
		"integer64": {
			In:  sampleInType{v: int64(0x100000000), t: tftypes.Number},
			Out: tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(0x100000000)),
			Err: nil,
		},
		"integer64-pseudotype": {
			In:  sampleInType{v: int64(0x100000000), t: tftypes.DynamicPseudoType},
			Out: tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(0x100000000)),
			Err: nil,
		},
		"integer32": {
			In:  sampleInType{int32(0x01000000), tftypes.Number},
			Out: tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(0x01000000)),
			Err: nil,
		},
		"integer32-pseudotype": {
			In:  sampleInType{int32(0x01000000), tftypes.DynamicPseudoType},
			Out: tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(0x01000000)),
			Err: nil,
		},
		"integer16": {
			In:  sampleInType{int16(0x0100), tftypes.Number},
			Out: tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(0x0100)),
			Err: nil,
		},
		"integer16-pseudotype": {
			In:  sampleInType{int16(0x0100), tftypes.DynamicPseudoType},
			Out: tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(0x0100)),
			Err: nil,
		},
		"float64": {
			In:  sampleInType{float64(100.0), tftypes.Number},
			Out: tftypes.NewValue(tftypes.Number, new(big.Float).SetFloat64(100)),
			Err: nil,
		},
		"float64-pseudotype": {
			In:  sampleInType{float64(100.0), tftypes.DynamicPseudoType},
			Out: tftypes.NewValue(tftypes.Number, new(big.Float).SetFloat64(100)),
			Err: nil,
		},
		"int-or-string-to-int": {
			In:  sampleInType{42, tftypes.String},
			Out: tftypes.NewValue(tftypes.String, "42"),
			Th:  map[string]string{tftypes.NewAttributePath().String(): "io.k8s.apimachinery.pkg.util.intstr.IntOrString"},
			Err: nil,
		},
		"int-or-string-to-string": {
			In:  sampleInType{"foobar", tftypes.String},
			Out: tftypes.NewValue(tftypes.String, "foobar"),
			Th:  map[string]string{tftypes.NewAttributePath().String(): "io.k8s.apimachinery.pkg.util.intstr.IntOrString"},
			Err: nil,
		},
		"list": {
			In: sampleInType{[]interface{}{"test1", "test2"}, tftypes.List{ElementType: tftypes.String}},
			Out: tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "test1"),
				tftypes.NewValue(tftypes.String, "test2"),
			}),
			Err: nil,
		},
		"list-int-or-string": {
			In: sampleInType{[]interface{}{"test1", 2}, tftypes.List{ElementType: tftypes.String}},
			Th: map[string]string{tftypes.NewAttributePath().WithElementKeyInt(-1).String(): "io.k8s.apimachinery.pkg.util.intstr.IntOrString"},
			Out: tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "test1"),
				tftypes.NewValue(tftypes.String, "2"),
			}),
			Err: nil,
		},
		"list (empty)": {
			In:  sampleInType{[]interface{}{}, tftypes.List{ElementType: tftypes.String}},
			Out: tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{}),
			Err: nil,
		},
		"set": {
			In: sampleInType{[]interface{}{"test1", "test2"}, tftypes.Set{ElementType: tftypes.String}},
			Out: tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "test1"),
				tftypes.NewValue(tftypes.String, "test2"),
			}),
			Err: nil,
		},
		"set (empty)": {
			In:  sampleInType{[]interface{}{}, tftypes.Set{ElementType: tftypes.String}},
			Out: tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{}),
			Err: nil,
		},
		"tuple": {
			In: sampleInType{[]interface{}{"test1", "test2"}, tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String, tftypes.String}}},
			Out: tftypes.NewValue(tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String, tftypes.String}}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "test1"),
				tftypes.NewValue(tftypes.String, "test2"),
			}),
			Err: nil,
		},
		"map": {
			In: sampleInType{
				v: map[string]interface{}{
					"foo": "18",
					"bar": "crawl",
				},
				t: tftypes.Map{ElementType: tftypes.String},
			},
			Out: tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{
				"foo": tftypes.NewValue(tftypes.String, "18"),
				"bar": tftypes.NewValue(tftypes.String, "crawl"),
			}),
			Err: nil,
		},
		"map-pseudotype": {
			In: sampleInType{
				v: map[string]interface{}{
					"count": 42,
					"image": "25%",
				},
				t: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
					"count": tftypes.DynamicPseudoType,
					"image": tftypes.DynamicPseudoType,
				}},
			},
			Out: tftypes.NewValue(
				tftypes.Object{AttributeTypes: map[string]tftypes.Type{
					"count": tftypes.Number,
					"image": tftypes.String,
				}},
				map[string]tftypes.Value{
					"count": tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(42)),
					"image": tftypes.NewValue(tftypes.String, "25%"),
				}),
			Err: nil,
		},
		"map-int-or-string": {
			In: sampleInType{
				v: map[string]interface{}{
					"count": 42,
					"image": "25%",
				},
				t: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
					"count": tftypes.String,
					"image": tftypes.String,
				}},
			},
			Th: map[string]string{
				tftypes.NewAttributePath().WithAttributeName("count").String(): "io.k8s.apimachinery.pkg.util.intstr.IntOrString",
				tftypes.NewAttributePath().WithAttributeName("image").String(): "io.k8s.apimachinery.pkg.util.intstr.IntOrString",
			},
			Out: tftypes.NewValue(
				tftypes.Object{AttributeTypes: map[string]tftypes.Type{
					"count": tftypes.String,
					"image": tftypes.String,
				}},
				map[string]tftypes.Value{
					"count": tftypes.NewValue(tftypes.String, "42"),
					"image": tftypes.NewValue(tftypes.String, "25%"),
				}),
			Err: nil,
		},
		"complex-map": {
			In: sampleInType{
				v: map[string]interface{}{
					"foo": []interface{}{"test1", "test2"},
					"bar": map[string]interface{}{
						"count": 1,
						"image": "nginx/latest",
					},
					"refresh": true,
				},
				t: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
					"foo": tftypes.List{ElementType: tftypes.String},
					"bar": tftypes.Object{AttributeTypes: map[string]tftypes.Type{
						"count": tftypes.Number,
						"image": tftypes.String,
					}},
					"refresh": tftypes.Bool,
				}},
			},
			Out: tftypes.NewValue(
				tftypes.Object{AttributeTypes: map[string]tftypes.Type{
					"foo": tftypes.List{ElementType: tftypes.String},
					"bar": tftypes.Object{AttributeTypes: map[string]tftypes.Type{
						"count": tftypes.Number,
						"image": tftypes.String,
					}},
					"refresh": tftypes.Bool,
				}},
				map[string]tftypes.Value{
					"foo": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
						tftypes.NewValue(tftypes.String, "test1"),
						tftypes.NewValue(tftypes.String, "test2"),
					}),
					"bar": tftypes.NewValue(
						tftypes.Object{AttributeTypes: map[string]tftypes.Type{
							"count": tftypes.Number,
							"image": tftypes.String,
						}},
						map[string]tftypes.Value{
							"count": tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(1)),
							"image": tftypes.NewValue(tftypes.String, "nginx/latest"),
						}),
					"refresh": tftypes.NewValue(tftypes.Bool, true),
				}),
		},
		"complex-map-pseudotype": {
			In: sampleInType{
				v: map[string]interface{}{
					"foo": []interface{}{"test1", "test2"},
					"bar": map[string]interface{}{
						"count": 1,
						"image": "nginx/latest",
					},
					"refresh": true,
				},
				t: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
					"foo": tftypes.List{ElementType: tftypes.String},
					"bar": tftypes.Object{AttributeTypes: map[string]tftypes.Type{
						"count": tftypes.Number,
						"image": tftypes.String,
					}},
					"refresh": tftypes.Bool,
				}},
			},
			Out: tftypes.NewValue(
				tftypes.Object{AttributeTypes: map[string]tftypes.Type{
					"foo": tftypes.List{ElementType: tftypes.String},
					"bar": tftypes.Object{AttributeTypes: map[string]tftypes.Type{
						"count": tftypes.Number,
						"image": tftypes.String,
					}},
					"refresh": tftypes.Bool,
				}},
				map[string]tftypes.Value{
					"foo": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
						tftypes.NewValue(tftypes.String, "test1"),
						tftypes.NewValue(tftypes.String, "test2"),
					}),
					"bar": tftypes.NewValue(
						tftypes.Object{AttributeTypes: map[string]tftypes.Type{
							"count": tftypes.Number,
							"image": tftypes.String,
						}},
						map[string]tftypes.Value{
							"count": tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(1)),
							"image": tftypes.NewValue(tftypes.String, "nginx/latest"),
						}),
					"refresh": tftypes.NewValue(tftypes.Bool, true),
				}),
		},
	}

	for name, s := range samples {
		t.Run(name, func(t *testing.T) {
			r, err := ToTFValue(s.In.v, s.In.t, s.Th, tftypes.NewAttributePath())
			if err != nil {
				if s.Err == nil {
					t.Logf("Unexpected error received for sample '%s': %s", name, err)
					t.FailNow()
				}
				if strings.Compare(err.Error(), s.Err.Error()) != 0 {
					t.Logf("Error does not match expectation for sample '%s': %s", name, err)
					t.FailNow()
				}
			} else {
				if !reflect.DeepEqual(s.Out, r) {
					t.Logf("Result doesn't match expectation for sample '%s'", name)
					t.Logf("\t Sample:\t%#v", s.In)
					t.Logf("\t Expected:\t%#v", s.Out)
					t.Logf("\t Received:\t%#v", r)
					t.Fail()
				}
			}
		})
	}
}

func TestSliceToTFDynamicValue(t *testing.T) {
	samples := map[string]struct {
		In    sampleInType
		Hints map[string]string
		Out   tftypes.Value
		Err   error
	}{
		"list-of-strings": {
			In: sampleInType{
				[]interface{}{"test1", "test2"},
				tftypes.DynamicPseudoType,
			},
			Hints: map[string]string{},
			Out: tftypes.NewValue(tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String, tftypes.String}}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "test1"),
				tftypes.NewValue(tftypes.String, "test2"),
			}),
			Err: nil,
		},
	}

	for name, s := range samples {
		t.Run(name, func(t *testing.T) {
			r, err := sliceToTFDynamicValue(s.In.v.([]interface{}), s.In.t, s.Hints, tftypes.NewAttributePath())
			if err != nil {
				if s.Err == nil {
					t.Logf("Unexpected error received for sample '%s': %s", name, err)
					t.FailNow()
				}
				if strings.Compare(err.Error(), s.Err.Error()) != 0 {
					t.Logf("Error does not match expectation for sample'%s': %s", name, err)
					t.FailNow()
				}
			} else {
				if !cmp.Equal(s.Out, r, cmpCompareAllOption) {
					t.Logf("Result doesn't match expectation for sample '%s'", name)
					t.Logf("\t Sample:\t%#v", s.In)
					t.Logf("\t Expected:\t%#v", s.Out)
					t.Logf("\t Received:\t%#v", r)
					t.Fail()
				}
			}
		})
	}
}

func TestSliceToTFTupleValue(t *testing.T) {
	samples := map[string]struct {
		In    sampleInType
		Hints map[string]string
		Out   tftypes.Value
		Err   error
	}{
		"list-of-strings": {
			In: sampleInType{
				[]interface{}{"test1", "test2"},
				tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String, tftypes.String}},
			},
			Hints: map[string]string{},
			Out: tftypes.NewValue(tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.String, tftypes.String}}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "test1"),
				tftypes.NewValue(tftypes.String, "test2"),
			}),
			Err: nil,
		},
	}

	for name, s := range samples {
		t.Run(name, func(t *testing.T) {
			r, err := sliceToTFTupleValue(s.In.v.([]interface{}), s.In.t, s.Hints, tftypes.NewAttributePath())
			if err != nil {
				if s.Err == nil {
					t.Logf("Unexpected error received for sample '%s': %s", name, err)
					t.FailNow()
				}
				if strings.Compare(err.Error(), s.Err.Error()) != 0 {
					t.Logf("Error does not match expectation for sample'%s': %s", name, err)
					t.FailNow()
				}
			} else {
				if !cmp.Equal(s.Out, r, cmpCompareAllOption) {
					t.Logf("Result doesn't match expectation for sample '%s'", name)
					t.Logf("\t Sample:\t%#v", s.In)
					t.Logf("\t Expected:\t%#v", s.Out)
					t.Logf("\t Received:\t%#v", r)
					t.Fail()
				}
			}
		})
	}
}

func TestSliceToTFSetValue(t *testing.T) {
	samples := map[string]struct {
		In    sampleInType
		Hints map[string]string
		Out   tftypes.Value
		Err   error
	}{
		"list-of-strings": {
			In: sampleInType{
				[]interface{}{"test1", "test2"},
				tftypes.Set{ElementType: tftypes.String},
			},
			Hints: map[string]string{},
			Out: tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "test1"),
				tftypes.NewValue(tftypes.String, "test2"),
			}),
			Err: nil,
		},
	}

	for name, s := range samples {
		t.Run(name, func(t *testing.T) {
			r, err := sliceToTFSetValue(s.In.v.([]interface{}), s.In.t, s.Hints, tftypes.NewAttributePath())
			if err != nil {
				if s.Err == nil {
					t.Logf("Unexpected error received for sample '%s': %s", name, err)
					t.FailNow()
				}
				if strings.Compare(err.Error(), s.Err.Error()) != 0 {
					t.Logf("Error does not match expectation for sample'%s': %s", name, err)
					t.FailNow()
				}
			} else {
				if !cmp.Equal(s.Out, r, cmpCompareAllOption) {
					t.Logf("Result doesn't match expectation for sample '%s'", name)
					t.Logf("\t Sample:\t%#v", s.In)
					t.Logf("\t Expected:\t%#v", s.Out)
					t.Logf("\t Received:\t%#v", r)
					t.Fail()
				}
			}
		})
	}
}

func TestSliceToTFListValue(t *testing.T) {
	samples := map[string]struct {
		In    sampleInType
		Hints map[string]string
		Out   tftypes.Value
		Err   error
	}{
		"list-of-strings": {
			In: sampleInType{
				[]interface{}{"test1", "test2"},
				tftypes.List{ElementType: tftypes.String},
			},
			Hints: map[string]string{},
			Out: tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "test1"),
				tftypes.NewValue(tftypes.String, "test2"),
			}),
			Err: nil,
		},
	}

	for name, s := range samples {
		t.Run(name, func(t *testing.T) {
			r, err := sliceToTFListValue(s.In.v.([]interface{}), s.In.t, s.Hints, tftypes.NewAttributePath())
			if err != nil {
				if s.Err == nil {
					t.Logf("Unexpected error received for sample '%s': %s", name, err)
					t.FailNow()
				}
				if strings.Compare(err.Error(), s.Err.Error()) != 0 {
					t.Logf("Error does not match expectation for sample'%s': %s", name, err)
					t.FailNow()
				}
			} else {
				if !cmp.Equal(s.Out, r, cmpCompareAllOption) {
					t.Logf("Result doesn't match expectation for sample '%s'", name)
					t.Logf("\t Sample:\t%#v", s.In)
					t.Logf("\t Expected:\t%#v", s.Out)
					t.Logf("\t Received:\t%#v", r)
					t.Fail()
				}
			}
		})
	}
}

func TestMapToTFMapValue(t *testing.T) {
	samples := map[string]struct {
		In    sampleInType
		Hints map[string]string
		Out   tftypes.Value
		Err   error
	}{
		"simple": {
			In: sampleInType{
				v: map[string]interface{}{
					"count": "42",
					"image": "nginx/latest",
				},
				t: tftypes.Map{ElementType: tftypes.String},
			},
			Hints: map[string]string{},
			Out: tftypes.NewValue(
				tftypes.Map{ElementType: tftypes.String},
				map[string]tftypes.Value{
					"count": tftypes.NewValue(tftypes.String, "42"),
					"image": tftypes.NewValue(tftypes.String, "nginx/latest"),
				}),
			Err: nil,
		},
	}

	for name, s := range samples {
		t.Run(name, func(t *testing.T) {
			r, err := mapToTFMapValue(s.In.v.(map[string]interface{}), s.In.t, s.Hints, tftypes.NewAttributePath())
			if err != nil {
				if s.Err == nil {
					t.Logf("Unexpected error received for sample '%s': %s", name, err)
					t.FailNow()
				}
				if strings.Compare(err.Error(), s.Err.Error()) != 0 {
					t.Logf("Error does not match expectation for sample'%s': %s", name, err)
					t.FailNow()
				}
			} else {
				if !cmp.Equal(s.Out, r, cmpCompareAllOption) {
					t.Logf("Result doesn't match expectation for sample '%s'", name)
					t.Logf("\t Sample:\t%#v", s.In)
					t.Logf("\t Expected:\t%#v", s.Out)
					t.Logf("\t Received:\t%#v", r)
					t.Fail()
				}
			}
		})
	}
}

func TestMapToTFObjectValue(t *testing.T) {
	samples := map[string]struct {
		In    sampleInType
		Hints map[string]string
		Out   tftypes.Value
		Err   error
	}{
		"simple": {
			In: sampleInType{
				v: map[string]interface{}{
					"count": 1,
					"image": "nginx/latest",
				},
				t: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
					"count": tftypes.Number,
					"image": tftypes.String,
				},
				},
			},
			Hints: map[string]string{},
			Out: tftypes.NewValue(
				tftypes.Object{AttributeTypes: map[string]tftypes.Type{
					"count": tftypes.Number,
					"image": tftypes.String,
				}},
				map[string]tftypes.Value{
					"count": tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(1)),
					"image": tftypes.NewValue(tftypes.String, "nginx/latest"),
				}),
			Err: nil,
		},
	}

	for name, s := range samples {
		t.Run(name, func(t *testing.T) {
			r, err := mapToTFObjectValue(s.In.v.(map[string]interface{}), s.In.t, s.Hints, tftypes.NewAttributePath())
			if err != nil {
				if s.Err == nil {
					t.Logf("Unexpected error received for sample '%s': %s", name, err)
					t.FailNow()
				}
				if strings.Compare(err.Error(), s.Err.Error()) != 0 {
					t.Logf("Error does not match expectation for sample'%s': %s", name, err)
					t.FailNow()
				}
			} else {
				if !cmp.Equal(s.Out, r, cmpCompareAllOption) {
					t.Logf("Result doesn't match expectation for sample '%s'", name)
					t.Logf("\t Sample:\t%#v", s.In)
					t.Logf("\t Expected:\t%#v", s.Out)
					t.Logf("\t Received:\t%#v", r)
					t.Fail()
				}
			}
		})
	}
}

func TestMapToTFDynamicValue(t *testing.T) {
	samples := map[string]struct {
		In    sampleInType
		Hints map[string]string
		Out   tftypes.Value
		Err   error
	}{}

	for name, s := range samples {
		t.Run(name, func(t *testing.T) {
			r, err := mapToTFDynamicValue(s.In.v.(map[string]interface{}), s.In.t, s.Hints, tftypes.NewAttributePath())
			if err != nil {
				if s.Err == nil {
					t.Logf("Unexpected error received for sample '%s': %s", name, err)
					t.FailNow()
				}
				if strings.Compare(err.Error(), s.Err.Error()) != 0 {
					t.Logf("Error does not match expectation for sample'%s': %s", name, err)
					t.FailNow()
				}
			} else {
				if !cmp.Equal(s.Out, r, cmpCompareAllOption) {
					t.Logf("Result doesn't match expectation for sample '%s'", name)
					t.Logf("\t Sample:\t%#v", s.In)
					t.Logf("\t Expected:\t%#v", s.Out)
					t.Logf("\t Received:\t%#v", r)
					t.Fail()
				}
			}
		})
	}

}
