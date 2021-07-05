package payload

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestFromTFValue(t *testing.T) {
	samples := map[string]struct {
		In  tftypes.Value
		Out interface{}
	}{
		"string-primitive": {
			In:  tftypes.NewValue(tftypes.String, "hello"),
			Out: "hello",
		},
		"float-primitive": {
			In:  tftypes.NewValue(tftypes.Number, big.NewFloat(100.2)),
			Out: 100.2,
		},
		"boolean-primitive": {
			In:  tftypes.NewValue(tftypes.Bool, true),
			Out: true,
		},
		"list-of-strings": {
			In: tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "test1"),
				tftypes.NewValue(tftypes.String, "test2"),
			}),
			Out: []interface{}{"test1", "test2"},
		},
		"map-of-strings": {
			In: tftypes.NewValue(tftypes.Map{AttributeType: tftypes.String}, map[string]tftypes.Value{
				"foo": tftypes.NewValue(tftypes.String, "test1"),
				"bar": tftypes.NewValue(tftypes.String, "test2"),
			}),
			Out: map[string]interface{}{
				"foo": "test1",
				"bar": "test2",
			},
		},
		"object": {
			In: tftypes.NewValue(tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"foo":    tftypes.String,
					"buzz":   tftypes.Number,
					"fake":   tftypes.Bool,
					"others": tftypes.List{ElementType: tftypes.String},
				},
			}, map[string]tftypes.Value{
				"foo":  tftypes.NewValue(tftypes.String, "bar"),
				"buzz": tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(42)),
				"fake": tftypes.NewValue(tftypes.Bool, true),
				"others": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
					tftypes.NewValue(tftypes.String, "this"),
					tftypes.NewValue(tftypes.String, "that"),
				}),
			}),
			Out: map[string]interface{}{
				"foo":    "bar",
				"buzz":   int64(42),
				"fake":   true,
				"others": []interface{}{"this", "that"},
			},
		},
	}
	for n, s := range samples {
		t.Run(n, func(t *testing.T) {
			r, err := FromTFValue(s.In, tftypes.NewAttributePath())
			if err != nil {
				t.Logf("Conversion failed for sample '%s': %s", n, err)
				t.FailNow()
			}
			if !reflect.DeepEqual(s.Out, r) {
				t.Logf("Result doesn't match expectation for sample '%s'", n)
				t.Logf("\t Sample:\t%s", spew.Sdump(s.In))
				t.Logf("\t Expected:\t%s", spew.Sdump(s.Out))
				t.Logf("\t Received:\t%s", spew.Sdump(r))
				t.Fail()
			}
		})
	}
}
