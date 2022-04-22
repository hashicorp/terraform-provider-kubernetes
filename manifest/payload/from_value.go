package payload

import (
	"math/big"
	"strconv"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// FromTFValue converts a Terraform specific tftypes.Value type object
// into a Kubernetes dynamic client specific unstructured object
func FromTFValue(in tftypes.Value, th map[string]string, ap *tftypes.AttributePath) (interface{}, error) {
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
		fnv, _ := nv.Float64()
		return fnv, err
	case in.Type().Is(tftypes.String):
		var sv string
		err = in.As(&sv)
		if err != nil {
			return nil, ap.NewErrorf("[%s] cannot extract contents of attribute: %s", ap.String(), err)
		}
		tp := valueToTypePath(ap)
		ot, ok := th[tp.String()]
		if ok && ot == "io.k8s.apimachinery.pkg.util.intstr.IntOrString" {
			n, err := strconv.Atoi(sv)
			if err == nil {
				return n, nil
			}
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
			nextAp := ap.WithElementKeyInt(k)
			ne, err := FromTFValue(le, th, nextAp)
			if err != nil {
				return nil, nextAp.NewErrorf("[%s] cannot convert list element to Unstructured: %s", nextAp.String(), err)
			}
			if ne != nil {
				lv = append(lv, ne)
			}
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
			ne, err := FromTFValue(me, th, nextAp)
			if err != nil {
				return nil, nextAp.NewErrorf("[%s]: cannot convert map element to Unstructured: %s", nextAp.String(), err.Error())
			}
			mv[k] = ne
		}
		return mv, nil
	default:
		return nil, ap.NewErrorf("[%s] cannot convert value of unknown type (%s)", ap.String(), in.Type().String())
	}
}

// valueToTypePath "normalizes" AttributePaths of values into a form that only describes the type hyerarchy.
// this is used when comparing value paths to type hints generated during the translation from OpenAPI into tftypes.
func valueToTypePath(a *tftypes.AttributePath) *tftypes.AttributePath {
	if a == nil {
		return nil
	}
	ns := make([]tftypes.AttributePathStep, len(a.Steps()))
	os := a.Steps()
	for i := range os {
		switch os[i].(type) {
		case tftypes.AttributeName:
			ns[i] = tftypes.AttributeName(os[i].(tftypes.AttributeName))
		case tftypes.ElementKeyString:
			ns[i] = tftypes.ElementKeyString("#")
		case tftypes.ElementKeyInt:
			ns[i] = tftypes.ElementKeyInt(-1)
		}
	}

	return tftypes.NewAttributePathWithSteps(ns)
}
