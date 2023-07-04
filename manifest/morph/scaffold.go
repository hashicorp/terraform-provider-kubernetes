// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package morph

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// DeepUnknown creates a value given an arbitrary type
// with a default value of Unknown for all its primitives.
func DeepUnknown(t tftypes.Type, v tftypes.Value, p *tftypes.AttributePath) (tftypes.Value, error) {
	if t == nil {
		return tftypes.Value{}, fmt.Errorf("type cannot be nil")
	}
	if !v.IsKnown() {
		return tftypes.NewValue(t, tftypes.UnknownValue), nil
	}
	switch {
	case t.Is(tftypes.Object{}):
		atts := t.(tftypes.Object).AttributeTypes
		var vals map[string]tftypes.Value
		ovals := make(map[string]tftypes.Value, len(atts))
		err := v.As(&vals)
		if err != nil {
			return tftypes.Value{}, p.NewError(err)
		}
		for name, att := range atts {
			np := p.WithAttributeName(name)
			nv, err := DeepUnknown(att, vals[name], np)
			if err != nil {
				return tftypes.Value{}, np.NewError(err)
			}
			ovals[name] = nv
			if nv.Type().Is(tftypes.Tuple{}) {
				atts[name] = nv.Type()
			}
		}
		return tftypes.NewValue(tftypes.Object{AttributeTypes: atts}, ovals), nil
	case t.Is(tftypes.Map{}):
		if v.IsNull() {
			return tftypes.NewValue(t, tftypes.UnknownValue), nil
		}
		var vals map[string]tftypes.Value
		err := v.As(&vals)
		if err != nil {
			return tftypes.Value{}, p.NewError(err)
		}
		for name, el := range vals {
			np := p.WithElementKeyString(name)
			nv, err := DeepUnknown(t.(tftypes.Map).ElementType, el, np)
			if err != nil {
				return tftypes.Value{}, np.NewError(err)
			}
			vals[name] = nv
		}
		return tftypes.NewValue(t, vals), nil
	case t.Is(tftypes.Tuple{}):
		if v.IsNull() {
			return tftypes.NewValue(t, tftypes.UnknownValue), nil
		}
		atts := t.(tftypes.Tuple).ElementTypes
		if len(v.Type().(tftypes.Tuple).ElementTypes) != len(atts) {
			if len(atts) != 1 {
				return tftypes.Value{}, p.NewErrorf("[%s] incompatible tuple types", p.String())
			}
			atts = make([]tftypes.Type, len(v.Type().(tftypes.Tuple).ElementTypes))
			for i := range v.Type().(tftypes.Tuple).ElementTypes {
				atts[i] = v.Type().(tftypes.Tuple).ElementTypes[i]
			}
		}
		var vals []tftypes.Value
		err := v.As(&vals)
		if err != nil {
			return tftypes.Value{}, p.NewError(err)
		}
		for i, et := range atts {
			np := p.WithElementKeyInt(i)
			nv, err := DeepUnknown(et, vals[i], np)
			if err != nil {
				return tftypes.Value{}, np.NewError(err)
			}
			vals[i] = nv
		}
		return tftypes.NewValue(tftypes.Tuple{ElementTypes: atts}, vals), nil
	case t.Is(tftypes.List{}) || t.Is(tftypes.Set{}):
		if v.IsNull() {
			return tftypes.NewValue(t, tftypes.UnknownValue), nil
		}
		vals := make([]tftypes.Value, 0)
		err := v.As(&vals)
		if err != nil {
			return tftypes.Value{}, p.NewError(err)
		}
		var elt tftypes.Type
		switch {
		case t.Is(tftypes.List{}):
			elt = t.(tftypes.List).ElementType
		case t.Is(tftypes.Set{}):
			elt = t.(tftypes.Set).ElementType
		}
		for i, el := range vals {
			np := p.WithElementKeyInt(i)
			nv, err := DeepUnknown(elt, el, np)
			if err != nil {
				return tftypes.Value{}, np.NewError(err)
			}
			vals[i] = nv
		}
		return tftypes.NewValue(t, vals), nil
	default:
		if v.IsKnown() && !v.IsNull() {
			return v, nil
		}
		return tftypes.NewValue(t, tftypes.UnknownValue), nil
	}
}

// UnknownToNull replaces all unknown values in a deep structure with null
func UnknownToNull(v tftypes.Value) tftypes.Value {
	if !v.IsKnown() {
		return tftypes.NewValue(v.Type(), nil)
	}
	if v.IsNull() {
		return v
	}
	switch {
	case v.Type().Is(tftypes.List{}) || v.Type().Is(tftypes.Set{}) || v.Type().Is(tftypes.Tuple{}):
		tpel := make([]tftypes.Value, 0)
		v.As(&tpel)
		for i := range tpel {
			tpel[i] = UnknownToNull(tpel[i])
		}
		return tftypes.NewValue(v.Type(), tpel)
	case v.Type().Is(tftypes.Map{}) || v.Type().Is(tftypes.Object{}):
		mpel := make(map[string]tftypes.Value)
		v.As(&mpel)
		for k, ev := range mpel {
			mpel[k] = UnknownToNull(ev)
		}
		return tftypes.NewValue(v.Type(), mpel)
	}
	return v
}
