// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestNormalizePlannedDynamicMapShapes(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		in := tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"name": tftypes.String,
		}}, map[string]tftypes.Value{
			"name": tftypes.NewValue(tftypes.String, "example"),
		})

		out, diag := normalizePlannedDynamicMapShapes(
			"object",
			in,
			func(v tftypes.Value, _ *tftypes.AttributePath) (tftypes.Value, error) {
				return v, nil
			},
		)
		if diag != nil {
			t.Fatalf("unexpected diagnostic: %+v", diag)
		}
		if !out.Equal(in) {
			t.Fatalf("unexpected normalized value:\nexpected: %s\nreceived: %s", in, out)
		}
	})

	t.Run("error-returns-diagnostic", func(t *testing.T) {
		in := tftypes.NewValue(tftypes.String, "value")
		out, diag := normalizePlannedDynamicMapShapes(
			"manifest",
			in,
			func(_ tftypes.Value, _ *tftypes.AttributePath) (tftypes.Value, error) {
				return tftypes.Value{}, fmt.Errorf("normalization failure")
			},
		)
		if !out.Equal(in) {
			t.Fatalf("expected original value on error:\nexpected: %s\nreceived: %s", in, out)
		}
		if diag == nil {
			t.Fatal("expected diagnostic, got nil")
		}
		if diag.Severity != tfprotov5.DiagnosticSeverityError {
			t.Fatalf("unexpected severity: %s", diag.Severity)
		}
		if diag.Attribute == nil || len(diag.Attribute.Steps()) != 1 || diag.Attribute.Steps()[0] != tftypes.AttributeName("manifest") {
			t.Fatalf("unexpected attribute path: %#v", diag.Attribute)
		}
		if diag.Summary != "Failed to normalize dynamic map element types in planned state" {
			t.Fatalf("unexpected summary: %s", diag.Summary)
		}
		if diag.Detail != "normalization failure" {
			t.Fatalf("unexpected detail: %s", diag.Detail)
		}
	})
}
