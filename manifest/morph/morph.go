// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package morph

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// ValueToType transforms a value along a new type and returns a new value conforming to the given type
func ValueToType(v tftypes.Value, t tftypes.Type, p *tftypes.AttributePath) (tftypes.Value, []*tfprotov5.Diagnostic) {
	var diags []*tfprotov5.Diagnostic
	if t == nil {
		diags = append(diags, &tfprotov5.Diagnostic{
			Attribute: p,
			Severity:  tfprotov5.DiagnosticSeverityError,
			Summary:   "Invalid reference type for attribute",
			Detail:    fmt.Sprintf("Cannot convert value into 'nil' type at attribute: %s", attributePathSummary(p)),
		})
		return tftypes.Value{}, diags
	}
	if v.IsNull() {
		return newValue(t, nil, p)
	}
	switch {
	case v.Type().Is(tftypes.String):
		return morphStringToType(v, t, p)
	case v.Type().Is(tftypes.Number):
		return morphNumberToType(v, t, p)
	case v.Type().Is(tftypes.Bool):
		return morphBoolToType(v, t, p)
	case v.Type().Is(tftypes.DynamicPseudoType):
		return v, diags
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
		diags = append(diags, &tfprotov5.Diagnostic{
			Attribute: p,
			Severity:  tfprotov5.DiagnosticSeverityWarning,
			Summary:   "Value that isn't fully known",
			Detail:    fmt.Sprintf("Type conversion cannot be performed on (partially) unknown value at attribute: %s", attributePathSummary(p)),
		})
		return v, diags
	}
	diags = append(diags, &tfprotov5.Diagnostic{
		Attribute: p,
		Severity:  tfprotov5.DiagnosticSeverityError,
		Summary:   "Value incompatible with expected type",
		Detail:    fmt.Sprintf("Attribute: %s\n... of type:\n%+v\n...cannot be converted to type:\n%s", attributePathSummary(p), typeNameNoPrefix(v.Type()), typeNameNoPrefix(t)),
	})
	return tftypes.Value{}, diags
}

func morphBoolToType(v tftypes.Value, t tftypes.Type, p *tftypes.AttributePath) (tftypes.Value, []*tfprotov5.Diagnostic) {
	var diags []*tfprotov5.Diagnostic
	if t.Is(tftypes.Bool) {
		return v, diags
	}
	var bnat bool
	err := v.As(&bnat)
	if err != nil {
		diags = append(diags, &tfprotov5.Diagnostic{
			Attribute: p,
			Severity:  tfprotov5.DiagnosticSeverityError,
			Summary:   "Invalid Boolean value",
			Detail:    fmt.Sprintf("Error: %s\nat attribute:\n%s", err, attributePathSummary(p)),
		})
		return tftypes.Value{}, diags
	}
	switch {
	case t.Is(tftypes.String):
		return newValue(t, strconv.FormatBool(bnat), p)
	case t.Is(tftypes.DynamicPseudoType):
		return v, diags
	}
	diags = append(diags, &tfprotov5.Diagnostic{
		Attribute: p,
		Severity:  tfprotov5.DiagnosticSeverityError,
		Summary:   "Value incompatible with expected type",
		Detail:    fmt.Sprintf("Cannot convert Bool values into type %s\n ...at attribute\n%s", typeNameNoPrefix(t), attributePathSummary(p)),
	})
	return tftypes.Value{}, diags
}

func morphNumberToType(v tftypes.Value, t tftypes.Type, p *tftypes.AttributePath) (tftypes.Value, []*tfprotov5.Diagnostic) {
	var diags []*tfprotov5.Diagnostic
	if t.Is(tftypes.Number) {
		return v, diags
	}
	var vnat big.Float
	err := v.As(&vnat)
	if err != nil {
		diags = append(diags, &tfprotov5.Diagnostic{
			Attribute: p,
			Severity:  tfprotov5.DiagnosticSeverityError,
			Summary:   "Invalid Number value",
			Detail:    fmt.Sprintf("Error: %s\n...at attribute:\n%s", err, p),
		})
		return tftypes.Value{}, diags
	}
	switch {
	case t.Is(tftypes.String):
		return newValue(t, vnat.String(), p)
	case t.Is(tftypes.DynamicPseudoType):
		return v, diags
	}
	diags = append(diags, &tfprotov5.Diagnostic{
		Attribute: p,
		Severity:  tfprotov5.DiagnosticSeverityError,
		Summary:   "Value incompatible with expected type",
		Detail:    fmt.Sprintf("Cannot convert Number values into type %s\n ...at attribute\n%s", typeNameNoPrefix(t), attributePathSummary(p)),
	})
	return tftypes.Value{}, diags
}

func morphStringToType(v tftypes.Value, t tftypes.Type, p *tftypes.AttributePath) (tftypes.Value, []*tfprotov5.Diagnostic) {
	var diags []*tfprotov5.Diagnostic
	if t.Is(tftypes.String) {
		return v, diags
	}
	var vnat string
	err := v.As(&vnat)
	if err != nil {
		diags = append(diags, &tfprotov5.Diagnostic{
			Attribute: p,
			Severity:  tfprotov5.DiagnosticSeverityError,
			Summary:   "Failed to extract value of String attribute",
			Detail:    fmt.Sprintf("Error: %s\n...at attribute:\n%s", err, attributePathSummary(p)),
		})
		return tftypes.Value{}, diags
	}
	switch {
	case t.Is(tftypes.Number):
		fv, err := strconv.ParseFloat(vnat, 64)
		if err != nil {
			diags = append(diags, &tfprotov5.Diagnostic{
				Attribute: p,
				Severity:  tfprotov5.DiagnosticSeverityError,
				Summary:   "String value doesn't parse as Number",
				Detail:    fmt.Sprintf("Error: %s\n...at attribute:\n%s", err, attributePathSummary(p)),
			})
			return tftypes.Value{}, diags
		}
		nv := new(big.Float).SetFloat64(fv)
		return newValue(t, nv, p)
	case t.Is(tftypes.Bool):
		bv, err := strconv.ParseBool(vnat)
		if err != nil {
			diags = append(diags, &tfprotov5.Diagnostic{
				Attribute: p,
				Severity:  tfprotov5.DiagnosticSeverityError,
				Summary:   "String value doesn't parse as Boolean",
				Detail:    fmt.Sprintf("Error: %s\n...at attribute:\n%s", err, attributePathSummary(p)),
			})
			return tftypes.Value{}, diags
		}
		return newValue(t, bv, p)
	case t.Is(tftypes.DynamicPseudoType):
		return v, diags
	}
	diags = append(diags, &tfprotov5.Diagnostic{
		Attribute: p,
		Severity:  tfprotov5.DiagnosticSeverityError,
		Summary:   "Value incompatible with expected type",
		Detail:    fmt.Sprintf("Cannot transform String value into type %s\n ...at attribute\n%s", typeNameNoPrefix(t), attributePathSummary(p)),
	})
	return tftypes.Value{}, diags
}

func morphListToType(v tftypes.Value, t tftypes.Type, p *tftypes.AttributePath) (tftypes.Value, []*tfprotov5.Diagnostic) {
	var diags []*tfprotov5.Diagnostic
	var lvals []tftypes.Value
	err := v.As(&lvals)
	if err != nil {
		diags = append(diags, &tfprotov5.Diagnostic{
			Attribute: p,
			Severity:  tfprotov5.DiagnosticSeverityError,
			Summary:   "Failed to extract value of List attribute",
			Detail:    fmt.Sprintf("Error: %s\n...at attribute:\n%s", err, attributePathSummary(p)),
		})
		return tftypes.Value{}, diags
	}
	switch {
	case t.Is(tftypes.List{}):
		var nlvals []tftypes.Value = make([]tftypes.Value, len(lvals))
		for i, v := range lvals {
			elp := p.WithElementKeyInt(i)
			nv, d := ValueToType(v, t.(tftypes.List).ElementType, elp)
			if len(d) > 0 {
				diags = append(diags, d...)
				for i := range d {
					if d[i].Severity == tfprotov5.DiagnosticSeverityError {
						diags = append(diags, &tfprotov5.Diagnostic{
							Attribute: elp,
							Severity:  tfprotov5.DiagnosticSeverityError,
							Summary:   "Invalid List value element",
							Detail:    fmt.Sprintf("Error at attribute:\n%s", attributePathSummary(elp)),
						})
						return tftypes.Value{}, diags
					}
				}
			}
			nlvals[i] = nv
		}
		return newValue(t, nlvals, p)
	case t.Is(tftypes.Tuple{}):
		if len(t.(tftypes.Tuple).ElementTypes) != len(lvals) {
			diags = append(diags, &tfprotov5.Diagnostic{
				Attribute: p,
				Severity:  tfprotov5.DiagnosticSeverityError,
				Summary:   "Failed to transform List value into Tuple of different length",
				Detail:    fmt.Sprintf("Error: %s\n...at attribute:\n%s", err, attributePathSummary(p)),
			})
			return tftypes.Value{}, diags
		}
		var tvals []tftypes.Value = make([]tftypes.Value, len(lvals))
		for i, v := range lvals {
			elp := p.WithElementKeyInt(i)
			nv, d := ValueToType(v, t.(tftypes.Tuple).ElementTypes[i], elp)
			if len(d) > 0 {
				diags = append(diags, d...)
				for i := range d {
					if d[i].Severity == tfprotov5.DiagnosticSeverityError {
						diags = append(diags, &tfprotov5.Diagnostic{
							Attribute: elp,
							Severity:  tfprotov5.DiagnosticSeverityError,
							Summary:   "Failed to transform List element into Tuple element type",
							Detail:    fmt.Sprintf("Error: %s\n...at attribute:\n%s", err, attributePathSummary(elp)),
						})
						return tftypes.Value{}, diags
					}
				}
			}
			tvals[i] = nv
		}
		return newValue(t, tvals, p)
	case t.Is(tftypes.Set{}):
		var svals []tftypes.Value = make([]tftypes.Value, len(lvals))
		for i, v := range lvals {
			elp := p.WithElementKeyInt(i)
			nv, d := ValueToType(v, t.(tftypes.Set).ElementType, elp)
			if len(d) > 0 {
				diags = append(diags, d...)
				for i := range d {
					if d[i].Severity == tfprotov5.DiagnosticSeverityError {
						diags = append(diags, &tfprotov5.Diagnostic{
							Attribute: elp,
							Severity:  tfprotov5.DiagnosticSeverityError,
							Summary:   "Failed to transform List element into Set element type",
							Detail:    fmt.Sprintf("Error: %s\n...at attribute:\n%s", err, attributePathSummary(elp)),
						})
						return tftypes.Value{}, diags
					}
				}
			}
			svals[i] = nv
		}
		return newValue(t, svals, p)
	case t.Is(tftypes.DynamicPseudoType):
		return v, diags
	}
	diags = append(diags, &tfprotov5.Diagnostic{
		Attribute: p,
		Severity:  tfprotov5.DiagnosticSeverityError,
		Summary:   "Cannot transform List value into unsupported type",
		Detail:    fmt.Sprintf("Required type %s, but got %s\n ...at attribute\n%s", typeNameNoPrefix(t), typeNameNoPrefix(v.Type()), attributePathSummary(p)),
	})
	return tftypes.Value{}, diags
}

func morphTupleIntoType(v tftypes.Value, t tftypes.Type, p *tftypes.AttributePath) (tftypes.Value, []*tfprotov5.Diagnostic) {
	var diags []*tfprotov5.Diagnostic
	var tvals []tftypes.Value
	err := v.As(&tvals)
	if err != nil {
		diags = append(diags, &tfprotov5.Diagnostic{
			Attribute: p,
			Severity:  tfprotov5.DiagnosticSeverityError,
			Summary:   "Failed to extract value of Tuple attribute",
			Detail:    fmt.Sprintf("Error: %s\n...at attribute:\n%s", err, attributePathSummary(p)),
		})
		return tftypes.Value{}, diags
	}
	switch {
	case t.Is(tftypes.Tuple{}):
		var eltypes []tftypes.Type = make([]tftypes.Type, len(tvals))
		var lvals []tftypes.Value = make([]tftypes.Value, len(tvals))
		if len(tvals) != len(t.(tftypes.Tuple).ElementTypes) {
			if len(t.(tftypes.Tuple).ElementTypes) > 1 {
				diags = append(diags, &tfprotov5.Diagnostic{
					Attribute: p,
					Severity:  tfprotov5.DiagnosticSeverityError,
					Summary:   "Failed to transform Tuple value into Tuple of different length",
					Detail:    fmt.Sprintf("Error at attribute:\n%s", attributePathSummary(p)),
				})
				return tftypes.Value{}, diags
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
			nv, d := ValueToType(v, eltypes[i], elp)
			if len(d) > 0 {
				diags = append(diags, d...)
				for i := range d {
					if d[i].Severity == tfprotov5.DiagnosticSeverityError {
						diags = append(diags, &tfprotov5.Diagnostic{
							Attribute: elp,
							Severity:  tfprotov5.DiagnosticSeverityError,
							Summary:   "Failed to transform Tuple element into Tuple element type",
							Detail:    fmt.Sprintf("Error (see above) at attribute:\n%s", attributePathSummary(elp)),
						})
						return tftypes.Value{}, diags
					}
				}
			}
			lvals[i] = nv
			eltypes[i] = nv.Type()
		}
		return newValue(tftypes.Tuple{ElementTypes: eltypes}, lvals, p)
	case t.Is(tftypes.List{}):
		var lvals []tftypes.Value = make([]tftypes.Value, len(tvals))
		for i, v := range tvals {
			elp := p.WithElementKeyInt(i)
			nv, d := ValueToType(v, t.(tftypes.List).ElementType, elp)
			if len(d) > 0 {
				diags = append(diags, d...)
				for i := range d {
					if d[i].Severity == tfprotov5.DiagnosticSeverityError {
						diags = append(diags, &tfprotov5.Diagnostic{
							Attribute: elp,
							Severity:  tfprotov5.DiagnosticSeverityError,
							Summary:   "Failed to transform Tuple element into List element type",
							Detail:    fmt.Sprintf("Error (see above) at attribute:\n%s", attributePathSummary(elp)),
						})
						return tftypes.Value{}, diags
					}
				}
			}
			lvals[i] = nv
		}
		return newValue(t, lvals, p)
	case t.Is(tftypes.Set{}):
		var svals []tftypes.Value = make([]tftypes.Value, len(tvals))
		for i, v := range tvals {
			elp := p.WithElementKeyInt(i)
			nv, d := ValueToType(v, t.(tftypes.Set).ElementType, elp)
			if len(d) > 0 {
				diags = append(diags, d...)
				for i := range d {
					if d[i].Severity == tfprotov5.DiagnosticSeverityError {
						diags = append(diags, &tfprotov5.Diagnostic{
							Attribute: elp,
							Severity:  tfprotov5.DiagnosticSeverityError,
							Summary:   "Failed to transform Tuple element into Set element type",
							Detail:    fmt.Sprintf("Error (see above) at attribute:\n%s", attributePathSummary(elp)),
						})
						return tftypes.Value{}, diags
					}
				}
			}
			svals[i] = nv
		}
		return newValue(t, svals, p)
	case t.Is(tftypes.DynamicPseudoType):
		return v, diags
	}
	diags = append(diags, &tfprotov5.Diagnostic{
		Attribute: p,
		Severity:  tfprotov5.DiagnosticSeverityError,
		Summary:   "Cannot transform Tuple value into unsupported type",
		Detail:    fmt.Sprintf("Required type %s, but got %s\n ...at attribute\n%s", typeNameNoPrefix(t), typeNameNoPrefix(v.Type()), attributePathSummary(p)),
	})
	return tftypes.Value{}, diags
}

func morphSetToType(v tftypes.Value, t tftypes.Type, p *tftypes.AttributePath) (tftypes.Value, []*tfprotov5.Diagnostic) {
	var diags []*tfprotov5.Diagnostic
	var svals []tftypes.Value
	err := v.As(&svals)
	if err != nil {
		diags = append(diags, &tfprotov5.Diagnostic{
			Attribute: p,
			Severity:  tfprotov5.DiagnosticSeverityError,
			Summary:   "Failed to extract value of Set attribute",
			Detail:    fmt.Sprintf("Error: %s\n...at attribute:\n%s", err, attributePathSummary(p)),
		})
		return tftypes.Value{}, diags
	}
	switch {
	case t.Is(tftypes.Set{}):
		var svals []tftypes.Value = make([]tftypes.Value, len(svals))
		for i, v := range svals {
			elp := p.WithElementKeyInt(i)
			nv, d := ValueToType(v, t.(tftypes.Set).ElementType, elp)
			if len(d) > 0 {
				diags = append(diags, d...)
				for i := range d {
					if d[i].Severity == tfprotov5.DiagnosticSeverityError {
						diags = append(diags, &tfprotov5.Diagnostic{
							Attribute: elp,
							Severity:  tfprotov5.DiagnosticSeverityError,
							Summary:   "Failed to transform Set element into Set element type",
							Detail:    fmt.Sprintf("Error (see above) at attribute:\n%s", attributePathSummary(elp)),
						})
						return tftypes.Value{}, diags
					}
				}
			}
			svals[i] = nv
		}
		return newValue(t, svals, p)
	case t.Is(tftypes.List{}):
		var lvals []tftypes.Value = make([]tftypes.Value, len(svals))
		for i, v := range svals {
			elp := p.WithElementKeyInt(i)
			nv, d := ValueToType(v, t.(tftypes.List).ElementType, elp)
			if len(d) > 0 {
				diags = append(diags, d...)
				for i := range d {
					if d[i].Severity == tfprotov5.DiagnosticSeverityError {
						diags = append(diags, &tfprotov5.Diagnostic{
							Attribute: elp,
							Severity:  tfprotov5.DiagnosticSeverityError,
							Summary:   "Failed to transform Set element into List element type",
							Detail:    fmt.Sprintf("Error (see above) at attribute:\n%s", attributePathSummary(elp)),
						})
						return tftypes.Value{}, diags
					}
				}
			}
			lvals[i] = nv
		}
		return newValue(t, lvals, p)
	case t.Is(tftypes.Tuple{}):
		if len(t.(tftypes.Tuple).ElementTypes) != len(svals) {
			diags = append(diags, &tfprotov5.Diagnostic{
				Attribute: p,
				Severity:  tfprotov5.DiagnosticSeverityError,
				Summary:   "Failed to transform Set value into Tuple of different length",
				Detail:    fmt.Sprintf("Error at attribute:\n%s", attributePathSummary(p)),
			})
			return tftypes.Value{}, diags
		}
		var tvals []tftypes.Value = make([]tftypes.Value, len(svals))
		for i, v := range svals {
			elp := p.WithElementKeyInt(i)
			nv, d := ValueToType(v, t.(tftypes.Tuple).ElementTypes[i], elp)
			if len(d) > 0 {
				diags = append(diags, d...)
				for i := range d {
					if d[i].Severity == tfprotov5.DiagnosticSeverityError {
						diags = append(diags, &tfprotov5.Diagnostic{
							Attribute: elp,
							Severity:  tfprotov5.DiagnosticSeverityError,
							Summary:   "Failed to transform Set element into Tuple element type",
							Detail:    fmt.Sprintf("Error (see above) at attribute:\n%s", attributePathSummary(elp)),
						})
						return tftypes.Value{}, diags
					}
				}
			}
			tvals[i] = nv
		}
		return newValue(t, tvals, p)
	case t.Is(tftypes.DynamicPseudoType):
		return v, diags
	}
	diags = append(diags, &tfprotov5.Diagnostic{
		Attribute: p,
		Severity:  tfprotov5.DiagnosticSeverityError,
		Summary:   "Cannot transform Set value into unsupported type",
		Detail:    fmt.Sprintf("Required type %s, but got %s\n...at attribute:\n%s", typeNameNoPrefix(t), typeNameNoPrefix(v.Type()), attributePathSummary(p)),
	})
	return tftypes.Value{}, diags
}

func morphMapToType(v tftypes.Value, t tftypes.Type, p *tftypes.AttributePath) (tftypes.Value, []*tfprotov5.Diagnostic) {
	var diags []*tfprotov5.Diagnostic
	var mvals map[string]tftypes.Value
	err := v.As(&mvals)
	if err != nil {
		diags = append(diags, &tfprotov5.Diagnostic{
			Attribute: p,
			Severity:  tfprotov5.DiagnosticSeverityError,
			Summary:   "Failed to extract value of Map attribute",
			Detail:    fmt.Sprintf("Error: %s\n...at attribute:\n%s", err, attributePathSummary(p)),
		})
		return tftypes.Value{}, diags
	}
	switch {
	case t.Is(tftypes.Object{}):
		var ovals map[string]tftypes.Value = make(map[string]tftypes.Value, len(mvals))
		for k, v := range mvals {
			elp := p.WithElementKeyString(k)
			et, ok := t.(tftypes.Object).AttributeTypes[k]
			if !ok {
				diags = append(diags, &tfprotov5.Diagnostic{
					Attribute: p,
					Severity:  tfprotov5.DiagnosticSeverityWarning,
					Summary:   "Attribute not found in schema",
					Detail:    fmt.Sprintf("Unable to find schema type for attribute:\n%s", attributePathSummary(elp)),
				})
				continue
			}
			nv, d := ValueToType(v, et, elp)
			if len(d) > 0 {
				diags = append(diags, d...)
				for i := range d {
					if d[i].Severity == tfprotov5.DiagnosticSeverityError {
						diags = append(diags, &tfprotov5.Diagnostic{
							Attribute: elp,
							Severity:  tfprotov5.DiagnosticSeverityError,
							Summary:   "Failed to transform Map element into Object element type",
							Detail:    fmt.Sprintf("Error (see above) at attribute:\n%s", attributePathSummary(elp)),
						})
						return tftypes.Value{}, diags
					}
				}
			}
			ovals[k] = nv
		}
		return newValue(t, ovals, p)
	case t.Is(tftypes.Map{}):
		var nmvals map[string]tftypes.Value = make(map[string]tftypes.Value, len(mvals))
		for k, v := range mvals {
			elp := p.WithElementKeyString(k)
			nv, d := ValueToType(v, t.(tftypes.Map).ElementType, elp)
			if len(d) > 0 {
				diags = append(diags, d...)
				for i := range d {
					if d[i].Severity == tfprotov5.DiagnosticSeverityError {
						diags = append(diags, &tfprotov5.Diagnostic{
							Attribute: elp,
							Severity:  tfprotov5.DiagnosticSeverityError,
							Summary:   "Failed to transform Map element into Map element type",
							Detail:    fmt.Sprintf("Error (see above) at attribute:\n%s", attributePathSummary(elp)),
						})
						return tftypes.Value{}, diags
					}
				}
			}
			nmvals[k] = nv
		}
		return newValue(t, nmvals, p)
	case t.Is(tftypes.DynamicPseudoType):
		return v, diags
	}
	diags = append(diags, &tfprotov5.Diagnostic{
		Attribute: p,
		Severity:  tfprotov5.DiagnosticSeverityError,
		Summary:   "Cannot transform Map value into unsupported type",
		Detail:    fmt.Sprintf("Required type %s, but got %s\n...at attribute:\n%s", typeNameNoPrefix(t), typeNameNoPrefix(v.Type()), attributePathSummary(p)),
	})
	return tftypes.Value{}, diags
}

func morphObjectToType(v tftypes.Value, t tftypes.Type, p *tftypes.AttributePath) (tftypes.Value, []*tfprotov5.Diagnostic) {
	var diags []*tfprotov5.Diagnostic
	var vals map[string]tftypes.Value
	err := v.As(&vals)
	if err != nil {
		diags = append(diags, &tfprotov5.Diagnostic{
			Attribute: p,
			Severity:  tfprotov5.DiagnosticSeverityError,
			Summary:   "Failed to extract value of Object attribute",
			Detail:    fmt.Sprintf("Error: %s\n...at attribute:\n%s", err, p),
		})
		return tftypes.Value{}, diags
	}
	switch {
	case t.Is(tftypes.Object{}):
		var ovals map[string]tftypes.Value = make(map[string]tftypes.Value, len(vals))
		for k, v := range vals {
			elp := p.WithAttributeName(k)
			nt, ok := t.(tftypes.Object).AttributeTypes[k]
			if !ok {
				diags = append(diags, &tfprotov5.Diagnostic{
					Attribute: p,
					Severity:  tfprotov5.DiagnosticSeverityWarning,
					Summary:   "Attribute not found in schema",
					Detail:    fmt.Sprintf("Unable to find schema type for attribute:\n%s", attributePathSummary(elp)),
				})
				continue
			}
			nv, d := ValueToType(v, nt, elp)
			if len(d) > 0 {
				diags = append(diags, d...)
				for i := range d {
					if d[i].Severity == tfprotov5.DiagnosticSeverityError {
						diags = append(diags, &tfprotov5.Diagnostic{
							Attribute: p,
							Severity:  tfprotov5.DiagnosticSeverityError,
							Summary:   "Failed to transform Object element into Object element type",
							Detail:    fmt.Sprintf("Error (see above) at attribute:\n%s", attributePathSummary(elp)),
						})
						return tftypes.Value{}, diags
					}
				}
			}
			ovals[k] = nv
		}
		// for attributes not specified by user add a nil value of their respective type
		// tftypes.NewValue() fails if any of the attributes in the object don't have a corresponding value
		for k := range t.(tftypes.Object).AttributeTypes {
			if _, ok := ovals[k]; !ok {
				nv, d := newValue(t.(tftypes.Object).AttributeTypes[k], nil, p)
				if d != nil {
					diags = append(diags, d...)
					return tftypes.Value{}, diags
				}
				ovals[k] = nv
			}
		}
		otypes := make(map[string]tftypes.Type, len(ovals))
		for k, v := range ovals {
			otypes[k] = v.Type()
		}
		return tftypes.NewValue(tftypes.Object{AttributeTypes: otypes}, ovals), diags
	case t.Is(tftypes.Map{}):
		var mvals map[string]tftypes.Value = make(map[string]tftypes.Value, len(vals))
		for k, v := range vals {
			elp := p.WithElementKeyString(k)
			nv, d := ValueToType(v, t.(tftypes.Map).ElementType, elp)
			if len(d) > 0 {
				diags = append(diags, d...)
				for i := range d {
					if d[i].Severity == tfprotov5.DiagnosticSeverityError {
						diags = append(diags, &tfprotov5.Diagnostic{
							Attribute: p,
							Severity:  tfprotov5.DiagnosticSeverityError,
							Summary:   "Failed to transform Object element into Map element type",
							Detail:    fmt.Sprintf("Error (see above) at attribute:\n%s", attributePathSummary(elp)),
						})
						return tftypes.Value{}, diags
					}
				}
			}
			mvals[k] = nv
		}
		return newValue(t, mvals, p)
	case t.Is(tftypes.DynamicPseudoType):
		return v, diags
	}
	diags = append(diags, &tfprotov5.Diagnostic{
		Attribute: p,
		Severity:  tfprotov5.DiagnosticSeverityError,
		Summary:   "Failed to transform Object into unsupported type",
		Detail:    fmt.Sprintf("Required type %s, but got %s\n...at attribute:\n%s", typeNameNoPrefix(t), typeNameNoPrefix(v.Type()), attributePathSummary(p)),
	})
	return tftypes.Value{}, diags
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

func typeNameNoPrefix(t tftypes.Type) string {
	return strings.ReplaceAll(t.String(), "tftypes.", "")
}

func attributePathSummary(p *tftypes.AttributePath) string {
	var b strings.Builder
	for pos, step := range p.Steps() {
		switch v := step.(type) {
		case tftypes.AttributeName:
			if pos != 0 {
				b.WriteString(".")
			}
			b.WriteString(string(v))
		case tftypes.ElementKeyString:
			b.WriteString("[" + string(v) + "]")
		case tftypes.ElementKeyInt:
			b.WriteString("[" + strconv.FormatInt(int64(v), 10) + "]")
		case tftypes.ElementKeyValue:
			b.WriteString("[" + tftypes.Value(v).String() + "]")
		}
	}
	return b.String()
}

func validateValue(t tftypes.Type, val interface{}, p *tftypes.AttributePath) []*tfprotov5.Diagnostic {
	var diags []*tfprotov5.Diagnostic
	if err := tftypes.ValidateValue(t, val); err != nil {
		diags = append(diags, &tfprotov5.Diagnostic{
			Attribute: p,
			Severity:  tfprotov5.DiagnosticSeverityError,
			Summary:   "Provider encountered an error when trying to determine the Terraform type information for the configured manifest",
			Detail:    err.(error).Error(),
		})
		return diags
	}
	return nil
}

func newValue(t tftypes.Type, val interface{}, p *tftypes.AttributePath) (tftypes.Value, []*tfprotov5.Diagnostic) {
	if diags := validateValue(t, val, p); diags != nil {
		return tftypes.Value{}, diags
	}
	return tftypes.NewValue(t, val), nil
}
