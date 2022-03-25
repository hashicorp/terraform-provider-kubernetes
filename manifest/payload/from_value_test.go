package payload

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestFromTFValue(t *testing.T) {
	// this mimics how terraform-plugin-go decodes floats that terraform sends over msgpack without a precision marker
	fv, _, err := big.ParseFloat("98.765", 10, 512, big.ToNearestEven)
	if err != nil {
		t.Fatalf("cannot create test float value out of string: %s", err)
	}
	samples := map[string]struct {
		In  tftypes.Value
		Th  map[string]string
		Out interface{}
	}{
		"string-primitive": {
			In:  tftypes.NewValue(tftypes.String, "hello"),
			Out: "hello",
		},
		"float-primitive-native-big": {
			In:  tftypes.NewValue(tftypes.Number, big.NewFloat(98.765)),
			Out: float64(98.765),
		},
		"float-primitive-from-string": {
			In:  tftypes.NewValue(tftypes.Number, fv),
			Out: float64(98.765),
		},
		"boolean-primitive": {
			In:  tftypes.NewValue(tftypes.Bool, true),
			Out: true,
		},
		"int-or-string-into-int": {
			In:  tftypes.NewValue(tftypes.String, "100"),
			Th:  map[string]string{"": "io.k8s.apimachinery.pkg.util.intstr.IntOrString"},
			Out: 100,
		},
		"int-or-string-into-string": {
			In:  tftypes.NewValue(tftypes.String, "foobar"),
			Th:  map[string]string{"": "io.k8s.apimachinery.pkg.util.intstr.IntOrString"},
			Out: "foobar",
		},
		"list-of-int-string": {
			In: tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "foo"),
				tftypes.NewValue(tftypes.String, "42"),
			}),
			Th:  map[string]string{tftypes.NewAttributePath().WithElementKeyInt(-1).String(): "io.k8s.apimachinery.pkg.util.intstr.IntOrString"},
			Out: []interface{}{"foo", 42},
		},
		"list-of-strings": {
			In: tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "test1"),
				tftypes.NewValue(tftypes.String, "test2"),
			}),
			Out: []interface{}{"test1", "test2"},
		},
		"map-of-strings": {
			In: tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{
				"foo": tftypes.NewValue(tftypes.String, "test1"),
				"bar": tftypes.NewValue(tftypes.String, "test2"),
			}),
			Out: map[string]interface{}{
				"foo": "test1",
				"bar": "test2",
			},
		},
		"map-of-int-or-strings": {
			In: tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{
				"foo": tftypes.NewValue(tftypes.String, "test1"),
				"bar": tftypes.NewValue(tftypes.String, "42"),
			}),
			Th: map[string]string{tftypes.NewAttributePath().WithElementKeyString("#").String(): "io.k8s.apimachinery.pkg.util.intstr.IntOrString"},
			Out: map[string]interface{}{
				"foo": "test1",
				"bar": 42,
			},
		},
		"object": {
			In: tftypes.NewValue(tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"foo":    tftypes.String,
					"stuff":  tftypes.String,
					"buzz":   tftypes.Number,
					"fake":   tftypes.Bool,
					"others": tftypes.List{ElementType: tftypes.String},
				},
			}, map[string]tftypes.Value{
				"foo":   tftypes.NewValue(tftypes.String, "bar"),
				"stuff": tftypes.NewValue(tftypes.String, "42"),
				"buzz":  tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(42)),
				"fake":  tftypes.NewValue(tftypes.Bool, true),
				"others": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
					tftypes.NewValue(tftypes.String, "this"),
					tftypes.NewValue(tftypes.String, "that"),
				}),
			}),
			Th: map[string]string{tftypes.NewAttributePath().WithAttributeName("stuff").String(): "io.k8s.apimachinery.pkg.util.intstr.IntOrString"},
			Out: map[string]interface{}{
				"foo":    "bar",
				"stuff":  42,
				"buzz":   int64(42),
				"fake":   true,
				"others": []interface{}{"this", "that"},
			},
		},
	}
	for n, s := range samples {
		t.Run(n, func(t *testing.T) {
			r, err := FromTFValue(s.In, s.Th, tftypes.NewAttributePath())
			if err != nil {
				t.Logf("Conversion failed for sample '%s': %s", n, err)
				t.FailNow()
			}
			if !reflect.DeepEqual(s.Out, r) {
				t.Logf("Result doesn't match expectation for sample '%s'", n)
				t.Logf("\tSample:\t%#v", s.In)
				t.Logf("\tExpected:\t%#v", s.Out)
				t.Logf("\tReceived:\t%#v", r)
				t.Fail()
			}
		})
	}
}

func TestValueToTypePath(t *testing.T) {
	samples := map[string]struct {
		In  *tftypes.AttributePath
		Out *tftypes.AttributePath
	}{
		"nil": {
			In:  nil,
			Out: nil,
		},
		"list": {
			In:  tftypes.NewAttributePath().WithElementKeyInt(6),
			Out: tftypes.NewAttributePath().WithElementKeyInt(-1),
		},
		"map": {
			In:  tftypes.NewAttributePath().WithElementKeyString("foo"),
			Out: tftypes.NewAttributePath().WithElementKeyString("#"),
		},
		"object": {
			In:  tftypes.NewAttributePath().WithAttributeName("bar"),
			Out: tftypes.NewAttributePath().WithAttributeName("bar"),
		},
		"object-map": {
			In:  tftypes.NewAttributePath().WithAttributeName("foo").WithElementKeyString("bar"),
			Out: tftypes.NewAttributePath().WithAttributeName("foo").WithElementKeyString("#"),
		},
		"object-list": {
			In:  tftypes.NewAttributePath().WithAttributeName("foo").WithElementKeyInt(42),
			Out: tftypes.NewAttributePath().WithAttributeName("foo").WithElementKeyInt(-1),
		},
		"object-list-map": {
			In:  tftypes.NewAttributePath().WithAttributeName("foo").WithElementKeyInt(42).WithElementKeyString("bar"),
			Out: tftypes.NewAttributePath().WithAttributeName("foo").WithElementKeyInt(-1).WithElementKeyString("#"),
		},
		"object-map-list": {
			In:  tftypes.NewAttributePath().WithAttributeName("foo").WithElementKeyString("bar").WithElementKeyInt(42),
			Out: tftypes.NewAttributePath().WithAttributeName("foo").WithElementKeyString("#").WithElementKeyInt(-1),
		},
		"list-object": {
			In:  tftypes.NewAttributePath().WithElementKeyInt(42).WithAttributeName("foo"),
			Out: tftypes.NewAttributePath().WithElementKeyInt(-1).WithAttributeName("foo"),
		},
		"list-map": {
			In:  tftypes.NewAttributePath().WithElementKeyInt(42).WithElementKeyString("bar"),
			Out: tftypes.NewAttributePath().WithElementKeyInt(-1).WithElementKeyString("#"),
		},
	}
	for n, s := range samples {
		t.Run(n, func(t *testing.T) {
			p := valueToTypePath(s.In)
			if !p.Equal(s.Out) {
				t.Logf("Expected %#v, received: %#v", s.Out, p)
				t.Fail()
			}
		})
	}
}
