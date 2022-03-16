package morph

import (
	"math/big"
	"strconv"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// ValueToType transforms a value along a new type and returns a new value conforming to the given type
func ValueToType(v tftypes.Value, t tftypes.Type, p *tftypes.AttributePath) (tftypes.Value, error) {
	if t == nil {
		return tftypes.Value{}, p.NewErrorf("type is nil")
	}
	if v.IsNull() {
		return tftypes.NewValue(t, nil), nil
	}
	switch {
	case v.Type().Is(tftypes.String):
		return morphStringToType(v, t, p)
	case v.Type().Is(tftypes.Number):
		return morphNumberToType(v, t, p)
	case v.Type().Is(tftypes.Bool):
		return morphBoolToType(v, t, p)
	case v.Type().Is(tftypes.DynamicPseudoType):
		return v, nil
	case v.Type().Is(tftypes.List{}):
		return morphListToType(v, t, p)
	case v.Type().Is(tftypes.Tuple{}):
		return morphTupleIntoType(v, t, p)
	case v.Type().Is(tftypes.Set{}):
		return morphSetToType(v, t, p)
	case v.Type().Is(tftypes.Map{}):
		return morphMapToType(v, t, p)
	case v.Type().Is(tftypes.Object{}):
		return morphObjectToType(v, t, p)
	}
	if !v.IsKnown() {
		return v, p.NewErrorf("cannot morph value that isn't fully known")
	}
	return tftypes.Value{}, p.NewErrorf("[%s] unsupported morph from value: %v", p.String(), v)
}

func morphBoolToType(v tftypes.Value, t tftypes.Type, p *tftypes.AttributePath) (tftypes.Value, error) {
	if t.Is(tftypes.Bool) {
		return v, nil
	}
	var bnat bool
	err := v.As(&bnat)
	if err != nil {
		return tftypes.Value{}, p.NewErrorf("[%s] failed to morph boolean value: %v", p.String(), err)
	}
	switch {
	case t.Is(tftypes.String):
		return tftypes.NewValue(t, strconv.FormatBool(bnat)), nil
	case t.Is(tftypes.DynamicPseudoType):
		return v, nil
	}
	return tftypes.Value{}, p.NewErrorf("[%s] unsupported morph of bool value into type: %s", p.String(), t.String())
}

func morphNumberToType(v tftypes.Value, t tftypes.Type, p *tftypes.AttributePath) (tftypes.Value, error) {
	if t.Is(tftypes.Number) {
		return v, nil
	}
	var vnat big.Float
	err := v.As(&vnat)
	if err != nil {
		return tftypes.Value{}, p.NewErrorf("[%s] failed to morph number value: %v", p.String(), err)
	}
	switch {
	case t.Is(tftypes.String):
		return tftypes.NewValue(t, vnat.String()), nil
	case t.Is(tftypes.DynamicPseudoType):
		return v, nil

	}
	return tftypes.Value{}, p.NewErrorf("[%s] unsupported morph of number value into type: %s", p.String(), t.String())
}

func morphStringToType(v tftypes.Value, t tftypes.Type, p *tftypes.AttributePath) (tftypes.Value, error) {
	if t.Is(tftypes.String) {
		return v, nil
	}
	var vnat string
	err := v.As(&vnat)
	if err != nil {
		return tftypes.Value{}, p.NewErrorf("[%s] failed to morph string value: %v", p.String(), err)
	}
	switch {
	case t.Is(tftypes.Number):
		fv, err := strconv.ParseFloat(vnat, 64)
		if err != nil {
			return tftypes.Value{}, p.NewErrorf("[%s] failed to morph string value to tftypes.Number: %v", p.String(), err)
		}
		nv := new(big.Float).SetFloat64(fv)
		return tftypes.NewValue(t, nv), nil
	case t.Is(tftypes.Bool):
		bv, err := strconv.ParseBool(vnat)
		if err != nil {
			return tftypes.Value{}, p.NewErrorf("[%s] failed to morph string value: %v", p.String(), err)
		}
		return tftypes.NewValue(t, bv), nil
	case t.Is(tftypes.DynamicPseudoType):
		return v, nil
	}
	return tftypes.Value{}, p.NewErrorf("[%s] unsupported morph of string value into type: %s", p.String(), t.String())
}

func morphListToType(v tftypes.Value, t tftypes.Type, p *tftypes.AttributePath) (tftypes.Value, error) {
	var lvals []tftypes.Value
	err := v.As(&lvals)
	if err != nil {
		return tftypes.Value{}, p.NewErrorf("[%s] failed to morph list value: %s", p.String(), err)
	}
	switch {
	case t.Is(tftypes.List{}):
		var nlvals []tftypes.Value = make([]tftypes.Value, len(lvals))
		for i, v := range lvals {
			elp := p.WithElementKeyInt(i)
			nv, err := ValueToType(v, t.(tftypes.List).ElementType, elp)
			if err != nil {
				return tftypes.Value{}, elp.NewErrorf("[%s] failed to morph list element into list element: %v", elp.String(), err)
			}
			nlvals[i] = nv
		}
		return tftypes.NewValue(t, nlvals), nil
	case t.Is(tftypes.Tuple{}):
		if len(t.(tftypes.Tuple).ElementTypes) != len(lvals) {
			return tftypes.Value{}, p.NewErrorf("[%s] failed to morph list into tuple (length mismatch)", p.String())
		}
		var tvals []tftypes.Value = make([]tftypes.Value, len(lvals))
		for i, v := range lvals {
			elp := p.WithElementKeyInt(i)
			nv, err := ValueToType(v, t.(tftypes.Tuple).ElementTypes[i], elp)
			if err != nil {
				return tftypes.Value{}, elp.NewErrorf("[%s] failed to morph list element into tuple element: %v", elp.String(), err)
			}
			tvals[i] = nv
		}
		return tftypes.NewValue(t, tvals), nil
	case t.Is(tftypes.Set{}):
		var svals []tftypes.Value = make([]tftypes.Value, len(lvals))
		for i, v := range lvals {
			elp := p.WithElementKeyInt(i)
			nv, err := ValueToType(v, t.(tftypes.Set).ElementType, elp)
			if err != nil {
				return tftypes.Value{}, elp.NewErrorf("[%s] failed to morph list element into set element: %v", elp.String(), err)
			}
			svals[i] = nv
		}
		return tftypes.NewValue(t, svals), nil
	case t.Is(tftypes.DynamicPseudoType):
		return v, nil
	}
	return tftypes.Value{}, p.NewErrorf("[%s] unsupported morph of list value into type: %s", p.String(), t.String())
}

func morphTupleIntoType(v tftypes.Value, t tftypes.Type, p *tftypes.AttributePath) (tftypes.Value, error) {
	var tvals []tftypes.Value
	err := v.As(&tvals)
	if err != nil {
		return tftypes.Value{}, p.NewErrorf("[%s] failed to morph tuple value: %v", p.String(), err)
	}
	switch {
	case t.Is(tftypes.Tuple{}):
		var eltypes []tftypes.Type = make([]tftypes.Type, len(tvals))
		var lvals []tftypes.Value = make([]tftypes.Value, len(tvals))
		if len(tvals) != len(t.(tftypes.Tuple).ElementTypes) {
			if len(t.(tftypes.Tuple).ElementTypes) > 1 {
				return tftypes.Value{}, p.NewErrorf("[%s] failed to morph tuple value: incompatible tuples", p.String())
			}
			// this is the special case workaround for non-uniform lists in OpenAPI (e.g. for CustomResourceDefinitionSpec.versions)
			for i := range tvals {
				eltypes[i] = t.(tftypes.Tuple).ElementTypes[0]
			}
		} else {
			for i := range tvals {
				eltypes[i] = t.(tftypes.Tuple).ElementTypes[i]
			}
		}
		for i, v := range tvals {
			elp := p.WithElementKeyInt(i)
			nv, err := ValueToType(v, eltypes[i], elp)
			if err != nil {
				return tftypes.Value{}, elp.NewErrorf("[%s] failed to morph tuple element into tuple element: %v", elp.String(), err)
			}
			lvals[i] = nv
		}
		return tftypes.NewValue(tftypes.Tuple{ElementTypes: eltypes}, lvals), nil
	case t.Is(tftypes.List{}):
		var lvals []tftypes.Value = make([]tftypes.Value, len(tvals))
		for i, v := range tvals {
			elp := p.WithElementKeyInt(i)
			nv, err := ValueToType(v, t.(tftypes.List).ElementType, elp)
			if err != nil {
				return tftypes.Value{}, elp.NewErrorf("[%s] failed to morph tuple element into list element: %v", elp.String(), err)
			}
			lvals[i] = nv
		}
		return tftypes.NewValue(t, lvals), nil
	case t.Is(tftypes.Set{}):
		var svals []tftypes.Value = make([]tftypes.Value, len(tvals))
		for i, v := range tvals {
			elp := p.WithElementKeyInt(i)
			nv, err := ValueToType(v, t.(tftypes.Set).ElementType, elp)
			if err != nil {
				return tftypes.Value{}, elp.NewErrorf("[%s] failed to morph tuple element into set element: %v", elp.String(), err)
			}
			svals[i] = nv
		}
		return tftypes.NewValue(t, svals), nil
	case t.Is(tftypes.DynamicPseudoType):
		return v, nil
	}
	return tftypes.Value{}, p.NewErrorf("[%s] unsupported morph of tuple value into type: %s", p.String(), t.String())
}

func morphSetToType(v tftypes.Value, t tftypes.Type, p *tftypes.AttributePath) (tftypes.Value, error) {
	var svals []tftypes.Value
	err := v.As(&svals)
	if err != nil {
		return tftypes.Value{}, p.NewErrorf("[%s] failed to morph set value: %v", p.String(), err)
	}
	switch {
	case t.Is(tftypes.Set{}):
		var svals []tftypes.Value = make([]tftypes.Value, len(svals))
		for i, v := range svals {
			elp := p.WithElementKeyInt(i)
			nv, err := ValueToType(v, t.(tftypes.Set).ElementType, elp)
			if err != nil {
				return tftypes.Value{}, elp.NewErrorf("[%s] failed to morph set element into set element : %v", elp.String(), err)
			}
			svals[i] = nv
		}
		return tftypes.NewValue(t, svals), nil
	case t.Is(tftypes.List{}):
		var lvals []tftypes.Value = make([]tftypes.Value, len(svals))
		for i, v := range svals {
			elp := p.WithElementKeyInt(i)
			nv, err := ValueToType(v, t.(tftypes.List).ElementType, elp)
			if err != nil {
				return tftypes.Value{}, elp.NewErrorf("[%s] failed to morph set element into list element : %v", elp.String(), err)
			}
			lvals[i] = nv
		}
		return tftypes.NewValue(t, lvals), nil
	case t.Is(tftypes.Tuple{}):
		if len(t.(tftypes.Tuple).ElementTypes) != len(svals) {
			return tftypes.Value{}, p.NewErrorf("[%s] failed to morph list into tuple (length mismatch)", p.String())
		}
		var tvals []tftypes.Value = make([]tftypes.Value, len(svals))
		for i, v := range svals {
			elp := p.WithElementKeyInt(i)
			nv, err := ValueToType(v, t.(tftypes.Tuple).ElementTypes[i], elp)
			if err != nil {
				return tftypes.Value{}, elp.NewErrorf("[%s] failed to morph list element into tuple element: %v", elp.String(), err)
			}
			tvals[i] = nv
		}
		return tftypes.NewValue(t, tvals), nil
	case t.Is(tftypes.DynamicPseudoType):
		return v, nil
	}
	return tftypes.Value{}, p.NewErrorf("[%s] unsupported morph of set value into type: %s", p.String(), t.String())
}

func morphMapToType(v tftypes.Value, t tftypes.Type, p *tftypes.AttributePath) (tftypes.Value, error) {
	var mvals map[string]tftypes.Value
	err := v.As(&mvals)
	if err != nil {
		return tftypes.Value{}, p.NewErrorf("[%s] failed to morph map value: %v", p.String(), err)
	}
	switch {
	case t.Is(tftypes.Object{}):
		var ovals map[string]tftypes.Value = make(map[string]tftypes.Value, len(mvals))
		for k, v := range mvals {
			elp := p.WithElementKeyString(k)
			nv, err := ValueToType(v, t.(tftypes.Object).AttributeTypes[k], elp)
			if err != nil {
				return tftypes.Value{}, elp.NewErrorf("[%s] failed to morph map element into object element: %v", elp.String(), err)
			}
			ovals[k] = nv
		}
		return tftypes.NewValue(t, ovals), nil
	case t.Is(tftypes.Map{}):
		var nmvals map[string]tftypes.Value = make(map[string]tftypes.Value, len(mvals))
		for k, v := range mvals {
			elp := p.WithElementKeyString(k)
			nv, err := ValueToType(v, t.(tftypes.Map).ElementType, elp)
			if err != nil {
				return tftypes.Value{}, elp.NewErrorf("[%s] failed to morph object element into map element: %v", elp.String(), err)
			}
			nmvals[k] = nv
		}
		return tftypes.NewValue(t, nmvals), nil
	case t.Is(tftypes.DynamicPseudoType):
		return v, nil
	}
	return tftypes.Value{}, p.NewErrorf("[%s] unsupported morph of map value into type: %s", p.String(), t.String())
}

func morphObjectToType(v tftypes.Value, t tftypes.Type, p *tftypes.AttributePath) (tftypes.Value, error) {
	var vals map[string]tftypes.Value
	err := v.As(&vals)
	if err != nil {
		return tftypes.Value{}, p.NewErrorf("[%s] failed to morph object value %v", p.String(), err)
	}
	switch {
	case t.Is(tftypes.Object{}):
		var ovals map[string]tftypes.Value = make(map[string]tftypes.Value, len(vals))
		for k, v := range vals {
			elp := p.WithAttributeName(k)
			nv, err := ValueToType(v, t.(tftypes.Object).AttributeTypes[k], elp)
			if err != nil {
				return tftypes.Value{}, elp.NewErrorf("[%s] failed to morph object element into object element: %v", elp.String(), err)
			}
			ovals[k] = nv
		}
		// for attributes not specified by user add a nil value of their respective type
		// tftypes.NewValue() fails if any of the attributes in the object don't have a corresponding value
		for k := range t.(tftypes.Object).AttributeTypes {
			if _, ok := ovals[k]; !ok {
				ovals[k] = tftypes.NewValue(t.(tftypes.Object).AttributeTypes[k], nil)
			}
		}
		otypes := make(map[string]tftypes.Type, len(ovals))
		for k, v := range ovals {
			otypes[k] = v.Type()
		}
		return tftypes.NewValue(tftypes.Object{AttributeTypes: otypes}, ovals), nil
	case t.Is(tftypes.Map{}):
		var mvals map[string]tftypes.Value = make(map[string]tftypes.Value, len(vals))
		for k, v := range vals {
			elp := p.WithElementKeyString(k)
			nv, err := ValueToType(v, t.(tftypes.Map).ElementType, elp)
			if err != nil {
				return tftypes.Value{}, elp.NewErrorf("[%s] failed to morph object element into map element: %v", elp.String(), err)
			}
			mvals[k] = nv
		}
		return tftypes.NewValue(t, mvals), nil
	case t.Is(tftypes.DynamicPseudoType):
		return v, nil
	}
	return tftypes.Value{}, p.NewErrorf("[%s] unsupported morph of object value into type: %s", p.String(), t.String())
}

// ValueToTypePath "normalizes" AttributePaths of values into a form that only describes the type hyerarchy.
// this is used when comparing value paths to type hints generated during the translation from OpenAPI into tftypes.
func ValueToTypePath(a *tftypes.AttributePath) *tftypes.AttributePath {
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
