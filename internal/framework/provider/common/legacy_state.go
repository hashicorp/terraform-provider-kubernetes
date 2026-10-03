// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// DecodeLegacyState decodes JSON state written by an SDKv2 version of a
// resource, for UpgradeState and MoveState. Attributes the schema no longer
// has, such as metadata.self_link, are dropped as SDKv2 dropped them. When
// rewrite is set it is applied to the decoded JSON object first, to convert
// an older schema version's shapes. The result is passed through
// NormalizeLegacyState.
func DecodeLegacyState(ctx context.Context, raw *tfprotov6.RawState, s schema.Schema, rewrite func(map[string]any) error) (tftypes.Value, error) {
	if raw == nil || len(raw.JSON) == 0 {
		return tftypes.Value{}, errors.New("the source state has no JSON data")
	}
	stateJSON := raw.JSON
	if rewrite != nil {
		var object map[string]any
		decoder := json.NewDecoder(bytes.NewReader(stateJSON))
		decoder.UseNumber()
		if err := decoder.Decode(&object); err != nil {
			return tftypes.Value{}, err
		}
		if object == nil {
			return tftypes.Value{}, errors.New("the source state is null")
		}
		if err := rewrite(object); err != nil {
			return tftypes.Value{}, err
		}
		encoded, err := json.Marshal(object)
		if err != nil {
			return tftypes.Value{}, err
		}
		stateJSON = encoded
	}
	value, err := (&tfprotov6.RawState{JSON: stateJSON}).UnmarshalWithOpts(s.Type().TerraformType(ctx), tfprotov6.UnmarshalOpts{
		ValueFromJSONOpts: tftypes.ValueFromJSONOpts{IgnoreUndefinedAttributes: true},
	})
	if err != nil {
		return tftypes.Value{}, err
	}
	if value.IsNull() {
		return tftypes.Value{}, errors.New("the source state is null")
	}
	return NormalizeLegacyState(value)
}

// NormalizeLegacyState replaces SDKv2's stored zero value for an unset
// metadata generate_name ("") with null, at every metadata level, so that an
// unchanged configuration plans no update before the first refresh.
func NormalizeLegacyState(value tftypes.Value) (tftypes.Value, error) {
	empty := tftypes.NewValue(tftypes.String, "")
	return tftypes.Transform(value, func(at *tftypes.AttributePath, v tftypes.Value) (tftypes.Value, error) {
		steps := at.Steps()
		if len(steps) == 0 || steps[len(steps)-1] != tftypes.AttributeName("generate_name") || !v.Equal(empty) {
			return v, nil
		}
		return tftypes.NewValue(tftypes.String, nil), nil
	})
}

// LegacyStateName returns the namespace and name of decoded SDKv2 state from
// its namespace/name ID and checks them against its single metadata block.
// SDKv2 saved a failed create, such as a tainted generate_name object, before
// reading the object back, so an empty metadata name or namespace is filled
// from the ID. Two different non-empty values are an error.
func LegacyStateName(values map[string]any) (string, string, error) {
	id, _ := values["id"].(string)
	namespace, name, ok := strings.Cut(id, "/")
	if !ok || namespace == "" || name == "" || strings.Contains(name, "/") {
		return "", "", fmt.Errorf("the source state ID %q is not namespace/name", id)
	}
	list, _ := values["metadata"].([]any)
	if len(list) != 1 {
		return "", "", fmt.Errorf("the source state has %d metadata blocks, expected 1", len(list))
	}
	metadata, ok := list[0].(map[string]any)
	if !ok {
		return "", "", errors.New("the source state metadata is not an object")
	}
	for _, field := range [][2]string{{"namespace", namespace}, {"name", name}} {
		key, want := field[0], field[1]
		switch metadata[key] {
		case nil, "":
			metadata[key] = want
		case want:
		default:
			return "", "", fmt.Errorf("the source state ID %q does not match its metadata %s %v", id, key, metadata[key])
		}
	}
	return namespace, name, nil
}
