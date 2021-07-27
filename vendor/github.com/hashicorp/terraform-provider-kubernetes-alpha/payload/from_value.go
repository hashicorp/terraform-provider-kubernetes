package payload

import (
	"math/big"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// FromTFValue converts a Terraform specific tftypes.Value type object
// into a Kubernetes dynamic client specific unstructured object
func FromTFValue(in tftypes.Value, ap *tftypes.AttributePath) (interface{}, error) {
	var err error
	if !in.IsKnown() {
		return nil, ap.NewErrorf("[%s] cannot convert unknown value to Unstructured", ap.String())
	}
	if in.IsNull() {
		return nil, nil
	}
	if in.Type().Is(tftypes.DynamicPseudoType) {
		return nil, ap.NewErrorf("[%s] cannot convert dynamic value to Unstructured", ap.String())
	}
	switch {
	case in.Type().Is(tftypes.Bool):
		var bv bool
		err = in.As(&bv)
		if err != nil {
			return nil, ap.NewErrorf("[%s] cannot extract contents of attribute: %s", ap.String(), err)
		}
		return bv, nil
	case in.Type().Is(tftypes.Number):
		var nv big.Float
		err = in.As(&nv)
		if nv.IsInt() {
			inv, acc := nv.Int64()
			if acc != big.Exact {
				return nil, ap.NewErrorf("[%s] inexact integer approximation when converting number value at:", ap.String())
			}
			return inv, nil
		}
		fnv, acc := nv.Float64()
		if acc != big.Exact {
			return nil, ap.NewErrorf("[%s] inexact float approximation when converting number value", ap.String())
		}
		return fnv, err
	case in.Type().Is(tftypes.String):
		var sv string
		err = in.As(&sv)
		if err != nil {
			return nil, ap.NewErrorf("[%s] cannot extract contents of attribute: %s", ap.String(), err)
		}
		return sv, nil
	case in.Type().Is(tftypes.List{}) || in.Type().Is(tftypes.Tuple{}):
		var l []tftypes.Value
		var lv []interface{}
		err = in.As(&l)
		if err != nil {
			return nil, ap.NewErrorf("[%s] cannot extract contents of attribute: %s", ap.String(), err)
		}
		if len(l) == 0 {
			return lv, nil
		}
		for k, le := range l {
			nextAp := ap.WithElementKeyInt(int64(k))
			ne, err := FromTFValue(le, nextAp)
			if err != nil {
				return nil, nextAp.NewErrorf("[%s] cannot convert list element to Unstructured: %s", nextAp.String(), err)
			}
			if ne != nil {
				lv = append(lv, ne)
			}
		}
		if len(lv) == 0 {
			return nil, nil
		}
		return lv, nil
	case in.Type().Is(tftypes.Map{}) || in.Type().Is(tftypes.Object{}):
		m := make(map[string]tftypes.Value)
		mv := make(map[string]interface{})
		err = in.As(&m)
		if err != nil {
			return nil, ap.NewErrorf("[%s] cannot extract contents of attribute: %s", ap.String(), err)
		}
		if len(m) == 0 {
			return mv, nil
		}
		for k, me := range m {
			var nextAp *tftypes.AttributePath
			switch {
			case in.Type().Is(tftypes.Map{}):
				nextAp = ap.WithElementKeyString(k)
			case in.Type().Is(tftypes.Object{}):
				nextAp = ap.WithAttributeName(k)
			}
			ne, err := FromTFValue(me, nextAp)
			if err != nil {
				return nil, nextAp.NewErrorf("[%s]: cannot convert map element to Unstructured: %s", nextAp.String(), err.Error())
			}
			if ne != nil {
				mv[k] = ne
			}
		}
		if len(mv) == 0 {
			return nil, nil
		}
		return mv, nil
	default:
		return nil, ap.NewErrorf("[%s] cannot convert value of unknown type (%s)", ap.String(), in.Type().String())
	}
}
