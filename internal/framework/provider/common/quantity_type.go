// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"k8s.io/apimachinery/pkg/api/resource"
)

// QuantityType is a string type for Kubernetes resource quantities such as "500m", "2Gi" or "4".
//
// The API server returns a canonical form ("2048Mi" comes back as "2Gi", "1000m" as "1"),
// so the value read back rarely matches what the user wrote. SDKv2 hid that with a
// DiffSuppressFunc (suppressEquivalentResourceQuantity). The Framework equivalent is
// semantic equality: when the new value is equivalent to the prior one, the prior value is
// kept, so state continues to hold the user's spelling and no diff appears.
//
// Semantic equality is skipped for null and unknown values, and it only equates known
// quantities. It does not make null and an empty map interchangeable.
type QuantityType struct {
	basetypes.StringType
}

var (
	_ basetypes.StringTypable = QuantityType{}
	_ attr.Type               = QuantityType{}
)

// Equal implements [attr.Type].
func (t QuantityType) Equal(o attr.Type) bool {
	other, ok := o.(QuantityType)
	if !ok {
		return false
	}
	return t.StringType.Equal(other.StringType)
}

// String implements [attr.Type].
func (t QuantityType) String() string {
	return "QuantityType"
}

// ValueFromString implements [basetypes.StringTypable].
func (t QuantityType) ValueFromString(_ context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return QuantityValue{StringValue: in}, nil
}

// ValueFromTerraform implements [attr.Type].
func (t QuantityType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}

	stringValue, ok := attrValue.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type of %T", attrValue)
	}

	stringValuable, diags := t.ValueFromString(ctx, stringValue)
	if diags.HasError() {
		return nil, fmt.Errorf("unexpected error converting StringValue to StringValuable: %v", diags)
	}

	return stringValuable, nil
}

// ValueType implements [attr.Type].
func (t QuantityType) ValueType(_ context.Context) attr.Value {
	return QuantityValue{}
}

// QuantityValue is a string holding a Kubernetes resource quantity.
type QuantityValue struct {
	basetypes.StringValue
}

var (
	_ basetypes.StringValuable                   = QuantityValue{}
	_ basetypes.StringValuableWithSemanticEquals = QuantityValue{}
)

// NewQuantityValue returns a known QuantityValue.
func NewQuantityValue(s string) QuantityValue {
	return QuantityValue{StringValue: basetypes.NewStringValue(s)}
}

// Equal implements [attr.Value].
func (v QuantityValue) Equal(o attr.Value) bool {
	other, ok := o.(QuantityValue)
	if !ok {
		return false
	}
	return v.StringValue.Equal(other.StringValue)
}

// Type implements [attr.Value].
func (v QuantityValue) Type(_ context.Context) attr.Type {
	return QuantityType{}
}

// StringSemanticEquals implements [basetypes.StringValuableWithSemanticEquals]. Two quantities
// are equivalent when they parse and compare equal, as in suppressEquivalentResourceQuantity.
// A value that does not parse is never equivalent: the validator rejects it in configuration,
// and treating it as different here keeps a malformed API value visible.
func (v QuantityValue) StringSemanticEquals(_ context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	newValue, ok := newValuable.(QuantityValue)
	if !ok {
		diags.AddError(
			"Semantic Equality Check Error",
			fmt.Sprintf("Expected value type %T but got value type %T. Please report this to the provider developers.", v, newValuable),
		)
		return false, diags
	}

	if v.IsNull() || v.IsUnknown() || newValue.IsNull() || newValue.IsUnknown() {
		return false, diags
	}

	oldQ, err := resource.ParseQuantity(v.ValueString())
	if err != nil {
		return false, diags
	}
	newQ, err := resource.ParseQuantity(newValue.ValueString())
	if err != nil {
		return false, diags
	}

	return oldQ.Cmp(newQ) == 0, diags
}
