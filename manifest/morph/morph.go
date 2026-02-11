// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package morph

import (
	"fmt"
	"math/big"
	"sort"
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
		schemaTypes := t.(tftypes.Tuple).ElementTypes
		var lvals []tftypes.Value = make([]tftypes.Value, len(tvals))
		outputTypes := make([]tftypes.Type, len(tvals))
		// Handle case where schema has 1 element type but data has multiple elements
		if len(tvals) != len(schemaTypes) {
			if len(schemaTypes) > 1 {
				diags = append(diags, &tfprotov5.Diagnostic{
					Attribute: p,
					Severity:  tfprotov5.DiagnosticSeverityError,
					Summary:   "Failed to transform Tuple value into Tuple of different length",
					Detail:    fmt.Sprintf("Error at attribute:\n%s", attributePathSummary(p)),
				})
				return tftypes.Value{}, diags
			}
			// this is the special case workaround for non-uniform lists in OpenAPI (e.g. for CustomResourceDefinitionSpec.versions)
			schemaTypes = make([]tftypes.Type, len(tvals))
			for i := range tvals {
				schemaTypes[i] = t.(tftypes.Tuple).ElementTypes[0]
			}
		}
		for i, v := range tvals {
			elp := p.WithElementKeyInt(i)
			nv, d := ValueToType(v, schemaTypes[i], elp)
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
			// Preserve DynamicPseudoType from schema, otherwise use actual morphed type
			// (which may be expanded from nested tuple morphing)
			if schemaTypes[i].Is(tftypes.DynamicPseudoType) {
				outputTypes[i] = tftypes.DynamicPseudoType
			} else {
				outputTypes[i] = nv.Type()
			}
		}
		return newValue(tftypes.Tuple{ElementTypes: outputTypes}, lvals, p)
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
		if t.(tftypes.Map).ElementType.Is(tftypes.DynamicPseudoType) {
			nmvals, err = NormalizeDynamicMapElements(nmvals, p)
			if err != nil {
				diags = append(diags, &tfprotov5.Diagnostic{
					Attribute: p,
					Severity:  tfprotov5.DiagnosticSeverityError,
					Summary:   "Failed to normalize map(dynamic) element types",
					Detail:    err.Error(),
				})
				return tftypes.Value{}, diags
			}
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
		// Build output type preserving DynamicPseudoType from schema while allowing
		// type expansion for tuples (e.g., when schema has 1 element but data has many)
		otypes := make(map[string]tftypes.Type, len(ovals))
		for k, v := range ovals {
			schemaType := t.(tftypes.Object).AttributeTypes[k]
			if schemaType.Is(tftypes.DynamicPseudoType) {
				// Preserve DynamicPseudoType to ensure type consistency between plan and apply
				otypes[k] = tftypes.DynamicPseudoType
			} else {
				// Use actual type to allow expansion (e.g., tuples with variable length)
				otypes[k] = v.Type()
			}
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
		if t.(tftypes.Map).ElementType.Is(tftypes.DynamicPseudoType) {
			mvals, err = NormalizeDynamicMapElements(mvals, p)
			if err != nil {
				diags = append(diags, &tfprotov5.Diagnostic{
					Attribute: p,
					Severity:  tfprotov5.DiagnosticSeverityError,
					Summary:   "Failed to normalize map(dynamic) element types",
					Detail:    err.Error(),
				})
				return tftypes.Value{}, diags
			}
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

// NormalizeDynamicMapElements normalizes the value set for a map(dynamic) so all
// elements have a coherent type shape before serialization.
//
// If all element types already match, values are returned unchanged.
// If elements are all objects with differing attribute sets/types, element
// object types are merged by attribute name and missing/incompatible attributes
// are represented as DynamicPseudoType.
// If elements are not all objects and have differing types, an error is
// returned rather than coercing values.
func NormalizeDynamicMapElements(vals map[string]tftypes.Value, p *tftypes.AttributePath) (map[string]tftypes.Value, error) {
	if p == nil {
		p = tftypes.NewAttributePath()
	}
	if len(vals) == 0 {
		return vals, nil
	}

	var firstType tftypes.Type
	allSameType := true
	allObjectTypes := true
	for _, v := range vals {
		if firstType == nil {
			firstType = v.Type()
		} else if !firstType.Equal(v.Type()) {
			allSameType = false
		}
		if !v.Type().Is(tftypes.Object{}) {
			allObjectTypes = false
		}
	}
	if allSameType {
		return vals, nil
	}
	if !allObjectTypes {
		return nil, p.NewErrorf(
			"[%s] cannot normalize map(dynamic) with incompatible element types: %s",
			attributePathSummary(p),
			describeMapElementTypes(vals),
		)
	}

	mergedObjectType := mergeDynamicMapObjectType(vals)
	mergedAttrTypes := mergedObjectType.AttributeTypes
	normalized := make(map[string]tftypes.Value, len(vals))

	for key, v := range vals {
		ep := p.WithElementKeyString(key)
		var objVals map[string]tftypes.Value
		if err := v.As(&objVals); err != nil {
			return nil, ep.NewError(err)
		}

		outVals := make(map[string]tftypes.Value, len(mergedAttrTypes))
		for attrName, targetType := range mergedAttrTypes {
			ap := ep.WithAttributeName(attrName)
			if av, ok := objVals[attrName]; ok {
				nv, err := normalizeMapObjectAttributeValue(av, targetType, ap)
				if err != nil {
					return nil, err
				}
				outVals[attrName] = nv
				continue
			}
			outVals[attrName] = tftypes.NewValue(targetType, nil)
		}
		normalized[key] = tftypes.NewValue(mergedObjectType, outVals)
	}

	return normalized, nil
}

// NormalizeDynamicMapShapes recursively normalizes map(dynamic) values nested in
// the supplied value to ensure serialization-safe element type structures.
func NormalizeDynamicMapShapes(v tftypes.Value, p *tftypes.AttributePath) (tftypes.Value, error) {
	if p == nil {
		p = tftypes.NewAttributePath()
	}
	if !v.IsKnown() || v.IsNull() {
		return v, nil
	}

	switch {
	case v.Type().Is(tftypes.Object{}):
		var vals map[string]tftypes.Value
		if err := v.As(&vals); err != nil {
			return tftypes.Value{}, p.NewError(err)
		}
		atts := v.Type().(tftypes.Object).AttributeTypes
		ovals := make(map[string]tftypes.Value, len(atts))
		for name, attType := range atts {
			np := p.WithAttributeName(name)
			cv, ok := vals[name]
			if !ok {
				ovals[name] = tftypes.NewValue(attType, nil)
				continue
			}
			nv, err := NormalizeDynamicMapShapes(cv, np)
			if err != nil {
				return tftypes.Value{}, err
			}
			ovals[name] = nv
		}
		return tftypes.NewValue(v.Type(), ovals), nil

	case v.Type().Is(tftypes.Map{}):
		var vals map[string]tftypes.Value
		if err := v.As(&vals); err != nil {
			return tftypes.Value{}, p.NewError(err)
		}
		mvals := make(map[string]tftypes.Value, len(vals))
		for key, el := range vals {
			np := p.WithElementKeyString(key)
			nv, err := NormalizeDynamicMapShapes(el, np)
			if err != nil {
				return tftypes.Value{}, err
			}
			mvals[key] = nv
		}
		if v.Type().(tftypes.Map).ElementType.Is(tftypes.DynamicPseudoType) {
			var err error
			mvals, err = NormalizeDynamicMapElements(mvals, p)
			if err != nil {
				return tftypes.Value{}, err
			}
		}
		return tftypes.NewValue(v.Type(), mvals), nil

	case v.Type().Is(tftypes.List{}) || v.Type().Is(tftypes.Set{}) || v.Type().Is(tftypes.Tuple{}):
		var vals []tftypes.Value
		if err := v.As(&vals); err != nil {
			return tftypes.Value{}, p.NewError(err)
		}
		out := make([]tftypes.Value, len(vals))
		for i, el := range vals {
			np := p.WithElementKeyInt(i)
			nv, err := NormalizeDynamicMapShapes(el, np)
			if err != nil {
				return tftypes.Value{}, err
			}
			out[i] = nv
		}
		return tftypes.NewValue(v.Type(), out), nil
	}

	return v, nil
}

func mergeDynamicMapObjectType(vals map[string]tftypes.Value) tftypes.Object {
	attrType := make(map[string]tftypes.Type)
	attrPresentCount := make(map[string]int)
	attrTypeConsistent := make(map[string]bool)
	totalElements := len(vals)

	for _, v := range vals {
		objType := v.Type().(tftypes.Object)
		for name, t := range objType.AttributeTypes {
			attrPresentCount[name]++
			if priorType, ok := attrType[name]; !ok {
				attrType[name] = t
				attrTypeConsistent[name] = true
			} else if !priorType.Equal(t) {
				attrTypeConsistent[name] = false
			}
		}
	}

	merged := make(map[string]tftypes.Type, len(attrType))
	for name, t := range attrType {
		if attrPresentCount[name] != totalElements || !attrTypeConsistent[name] {
			merged[name] = tftypes.DynamicPseudoType
			continue
		}
		merged[name] = t
	}
	return tftypes.Object{AttributeTypes: merged}
}

func normalizeMapObjectAttributeValue(v tftypes.Value, targetType tftypes.Type, p *tftypes.AttributePath) (tftypes.Value, error) {
	if targetType.Is(tftypes.DynamicPseudoType) {
		return v, nil
	}
	if v.Type().Equal(targetType) {
		return v, nil
	}
	if v.IsNull() {
		return tftypes.NewValue(targetType, nil), nil
	}
	if !v.IsKnown() {
		return tftypes.NewValue(targetType, tftypes.UnknownValue), nil
	}
	return tftypes.Value{}, p.NewErrorf(
		"[%s] incompatible map(dynamic) attribute type: expected %s, got %s",
		attributePathSummary(p),
		typeNameNoPrefix(targetType),
		typeNameNoPrefix(v.Type()),
	)
}

func describeMapElementTypes(vals map[string]tftypes.Value) string {
	types := make(map[string]struct{}, len(vals))
	for _, v := range vals {
		types[typeNameNoPrefix(v.Type())] = struct{}{}
	}
	list := make([]string, 0, len(types))
	for t := range types {
		list = append(list, t)
	}
	sort.Strings(list)
	return strings.Join(list, ", ")
}

// MorphTypeStructure converts a value to have a different type structure while preserving
// the underlying data. This is needed when plan and apply produce values with different
// type structures due to DynamicPseudoType being converted to concrete types during
// serialization/deserialization.
//
// For example, if the source has type Object{masquerade: DynamicPseudoType} and the target
// has type Object{masquerade: Object{}}, this function will re-wrap the nested values
// to match the target type structure.
func MorphTypeStructure(source tftypes.Value, targetType tftypes.Type, p *tftypes.AttributePath) (tftypes.Value, error) {
	if source.IsNull() {
		return tftypes.NewValue(targetType, nil), nil
	}
	if !source.IsKnown() {
		return tftypes.NewValue(targetType, tftypes.UnknownValue), nil
	}

	// If source type equals target type, return as-is
	if source.Type().Equal(targetType) {
		return source, nil
	}

	// Handle Objects
	if targetType.Is(tftypes.Object{}) {
		var vals map[string]tftypes.Value
		// Try to extract as object - works for both Object and DynamicPseudoType containing object
		if err := source.As(&vals); err != nil {
			return tftypes.Value{}, p.NewError(err)
		}

		targetAttrs := targetType.(tftypes.Object).AttributeTypes
		newVals := make(map[string]tftypes.Value, len(targetAttrs))

		// Process each target attribute
		for k, targetAttrType := range targetAttrs {
			np := p.WithAttributeName(k)
			if sourceVal, ok := vals[k]; ok {
				newVal, err := MorphTypeStructure(sourceVal, targetAttrType, np)
				if err != nil {
					return tftypes.Value{}, err
				}
				newVals[k] = newVal
			} else {
				// Missing in source - add as null
				newVals[k] = tftypes.NewValue(targetAttrType, nil)
			}
		}

		return tftypes.NewValue(targetType, newVals), nil
	}

	// Handle Lists
	if targetType.Is(tftypes.List{}) {
		var vals []tftypes.Value
		if err := source.As(&vals); err != nil {
			return tftypes.Value{}, p.NewError(err)
		}

		elemType := targetType.(tftypes.List).ElementType
		newVals := make([]tftypes.Value, len(vals))
		for i, v := range vals {
			np := p.WithElementKeyInt(i)
			newVal, err := MorphTypeStructure(v, elemType, np)
			if err != nil {
				return tftypes.Value{}, err
			}
			newVals[i] = newVal
		}

		return tftypes.NewValue(targetType, newVals), nil
	}

	// Handle Tuples
	if targetType.Is(tftypes.Tuple{}) {
		var vals []tftypes.Value
		if err := source.As(&vals); err != nil {
			return tftypes.Value{}, p.NewError(err)
		}

		elemTypes := targetType.(tftypes.Tuple).ElementTypes
		if len(vals) != len(elemTypes) {
			return tftypes.Value{}, p.NewErrorf("tuple length mismatch: source has %d elements, target expects %d", len(vals), len(elemTypes))
		}

		newVals := make([]tftypes.Value, len(vals))
		for i, v := range vals {
			np := p.WithElementKeyInt(i)
			newVal, err := MorphTypeStructure(v, elemTypes[i], np)
			if err != nil {
				return tftypes.Value{}, err
			}
			newVals[i] = newVal
		}

		return tftypes.NewValue(targetType, newVals), nil
	}

	// Handle Maps
	if targetType.Is(tftypes.Map{}) {
		var vals map[string]tftypes.Value
		if err := source.As(&vals); err != nil {
			return tftypes.Value{}, p.NewError(err)
		}

		elemType := targetType.(tftypes.Map).ElementType
		newVals := make(map[string]tftypes.Value, len(vals))
		for k, v := range vals {
			np := p.WithElementKeyString(k)
			newVal, err := MorphTypeStructure(v, elemType, np)
			if err != nil {
				return tftypes.Value{}, err
			}
			newVals[k] = newVal
		}

		return tftypes.NewValue(targetType, newVals), nil
	}

	// Handle Sets
	if targetType.Is(tftypes.Set{}) {
		var vals []tftypes.Value
		if err := source.As(&vals); err != nil {
			return tftypes.Value{}, p.NewError(err)
		}

		elemType := targetType.(tftypes.Set).ElementType
		newVals := make([]tftypes.Value, len(vals))
		for i, v := range vals {
			newVal, err := MorphTypeStructure(v, elemType, p)
			if err != nil {
				return tftypes.Value{}, err
			}
			newVals[i] = newVal
		}

		return tftypes.NewValue(targetType, newVals), nil
	}

	// Handle DynamicPseudoType target - return source as-is
	if targetType.Is(tftypes.DynamicPseudoType) {
		return source, nil
	}

	// For primitives (String, Number, Bool), extract and re-wrap
	switch {
	case targetType.Is(tftypes.String):
		var s string
		if err := source.As(&s); err != nil {
			return tftypes.Value{}, p.NewError(err)
		}
		return tftypes.NewValue(targetType, s), nil
	case targetType.Is(tftypes.Number):
		var n *big.Float
		if err := source.As(&n); err != nil {
			return tftypes.Value{}, p.NewError(err)
		}
		return tftypes.NewValue(targetType, n), nil
	case targetType.Is(tftypes.Bool):
		var b bool
		if err := source.As(&b); err != nil {
			return tftypes.Value{}, p.NewError(err)
		}
		return tftypes.NewValue(targetType, b), nil
	}

	// Fallback - return error for unsupported conversions
	return tftypes.Value{}, p.NewErrorf("cannot morph type %s to %s", source.Type().String(), targetType.String())
}
