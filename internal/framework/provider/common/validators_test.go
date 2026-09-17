// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestLabelsValidator(t *testing.T) {
	t.Parallel()

	labels := func(elems map[string]attr.Value) types.Map {
		return types.MapValueMust(types.StringType, elems)
	}

	cases := []struct {
		name       string
		value      types.Map
		wantErrors int
	}{
		{"null map is not validated", types.MapNull(types.StringType), 0},
		{"unknown map is not validated", types.MapUnknown(types.StringType), 0},
		{"valid key and value", labels(map[string]attr.Value{"app.kubernetes.io/name": types.StringValue("web")}), 0},
		{"unknown value is deferred to apply", labels(map[string]attr.Value{"env": types.StringUnknown()}), 0},
		// The reason for this change: SDKv2's validateLabels rejects non-string values.
		{"null value is rejected", labels(map[string]attr.Value{"env": types.StringNull()}), 1},
		{"invalid key is rejected", labels(map[string]attr.Value{"bad key!": types.StringValue("x")}), 1},
		{"invalid value is rejected", labels(map[string]attr.Value{"env": types.StringValue("not valid!")}), 1},
		// Unlike SDKv2, which returns at the first null, every error is reported.
		{"all errors reported, not just the first null", labels(map[string]attr.Value{
			"a":        types.StringNull(),
			"b":        types.StringNull(),
			"bad key!": types.StringValue("x"),
		}), 3},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := validator.MapRequest{Path: path.Root("labels"), ConfigValue: tc.value}
			resp := &validator.MapResponse{}
			LabelsValidator().ValidateMap(context.Background(), req, resp)

			if got := resp.Diagnostics.ErrorsCount(); got != tc.wantErrors {
				t.Errorf("got %d errors, want %d: %v", got, tc.wantErrors, resp.Diagnostics)
			}
		})
	}
}

// TestValidatorDiagnosticFormat pins the message format. Diagnostics take the attribute
// from req.Path instead of naming it, so the same validator reports correctly wherever it
// is attached.
func TestValidatorDiagnosticFormat(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mapOf := func(elems map[string]attr.Value) types.Map {
		return types.MapValueMust(types.StringType, elems)
	}
	stringDiags := func(v validator.String, p path.Path, value types.String) diag.Diagnostics {
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, validator.StringRequest{Path: p, ConfigValue: value}, resp)
		return resp.Diagnostics
	}
	mapDiags := func(v validator.Map, p path.Path, value types.Map) diag.Diagnostics {
		resp := &validator.MapResponse{}
		v.ValidateMap(ctx, validator.MapRequest{Path: p, ConfigValue: value}, resp)
		return resp.Diagnostics
	}

	meta := path.Root("metadata").AtListIndex(0)
	cases := []struct {
		name         string
		diags        diag.Diagnostics
		wantPath     path.Path
		wantInDetail []string
	}{
		{
			name:         "name",
			diags:        stringDiags(DNSSubdomainNameValidator(), meta.AtName("name"), types.StringValue("Bad_Name")),
			wantPath:     meta.AtName("name"),
			wantInDetail: []string{`Attribute metadata[0].name `, `got: "Bad_Name"`},
		},
		{
			// The same validator on a different attribute names that attribute — the point
			// of not hardcoding it.
			name:         "name validator reused elsewhere",
			diags:        stringDiags(DNSSubdomainNameValidator(), path.Root("service_account_name"), types.StringValue("Bad_Name")),
			wantPath:     path.Root("service_account_name"),
			wantInDetail: []string{`Attribute service_account_name `, `got: "Bad_Name"`},
		},
		{
			name:         "generate_name",
			diags:        stringDiags(DNSLabelPrefixValidator(), meta.AtName("generate_name"), types.StringValue("Bad_")),
			wantPath:     meta.AtName("generate_name"),
			wantInDetail: []string{`Attribute metadata[0].generate_name `, `got: "Bad_"`},
		},
		{
			name:         "annotation key",
			diags:        mapDiags(AnnotationsValidator(), meta.AtName("annotations"), mapOf(map[string]attr.Value{"bad key!": types.StringValue("x")})),
			wantPath:     meta.AtName("annotations").AtMapKey("bad key!"),
			wantInDetail: []string{`Attribute metadata[0].annotations["bad key!"] key `, `got: "bad key!"`},
		},
		{
			name:         "label key",
			diags:        mapDiags(LabelsValidator(), meta.AtName("labels"), mapOf(map[string]attr.Value{"bad key!": types.StringValue("x")})),
			wantPath:     meta.AtName("labels").AtMapKey("bad key!"),
			wantInDetail: []string{`Attribute metadata[0].labels["bad key!"] key `, `got: "bad key!"`},
		},
		{
			name:         "label value",
			diags:        mapDiags(LabelsValidator(), meta.AtName("labels"), mapOf(map[string]attr.Value{"env": types.StringValue("not valid!")})),
			wantPath:     meta.AtName("labels").AtMapKey("env"),
			wantInDetail: []string{`Attribute metadata[0].labels["env"] value `, `got: "not valid!"`},
		},
		{
			name:         "null label value",
			diags:        mapDiags(LabelsValidator(), meta.AtName("labels"), mapOf(map[string]attr.Value{"env": types.StringNull()})),
			wantPath:     meta.AtName("labels").AtMapKey("env"),
			wantInDetail: []string{`Attribute metadata[0].labels["env"] value must be a string, got: <null>`},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if len(tc.diags) == 0 {
				t.Fatal("expected a diagnostic, got none")
			}
			d := tc.diags[0]
			if d.Summary() != "Invalid Attribute Value" {
				t.Errorf("summary = %q, want %q", d.Summary(), "Invalid Attribute Value")
			}
			withPath, ok := d.(diag.DiagnosticWithPath)
			if !ok || !withPath.Path().Equal(tc.wantPath) {
				t.Errorf("diagnostic path = %v, want %s", withPath, tc.wantPath)
			}
			for _, want := range tc.wantInDetail {
				if !strings.Contains(d.Detail(), want) {
					t.Errorf("detail %q does not contain %q", d.Detail(), want)
				}
			}
		})
	}
}

// TestAnnotationsValidator covers the null-value rejection added for bug 3. SDKv2's
// validateAnnotations checks keys only; Terraform passes null elements through, and an
// unchecked null causes a misleading apply error on Create and a permanent diff on Update.
func TestAnnotationsValidator(t *testing.T) {
	t.Parallel()

	annotations := func(elems map[string]attr.Value) types.Map {
		return types.MapValueMust(types.StringType, elems)
	}

	cases := []struct {
		name       string
		value      types.Map
		wantErrors int
	}{
		{"null map is not validated", types.MapNull(types.StringType), 0},
		{"unknown map is not validated", types.MapUnknown(types.StringType), 0},
		{"valid key and value", annotations(map[string]attr.Value{"owner": types.StringValue("platform")}), 0},
		{"unknown value is deferred to apply", annotations(map[string]attr.Value{"owner": types.StringUnknown()}), 0},
		{"null value is rejected", annotations(map[string]attr.Value{"owner": types.StringNull()}), 1},
		{"invalid key is rejected", annotations(map[string]attr.Value{"bad key!": types.StringValue("x")}), 1},
		// Annotation values are not otherwise constrained: unlike labels, any string is legal.
		{"long value is allowed", annotations(map[string]attr.Value{"owner": types.StringValue("a very long value, spaces and all")}), 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := validator.MapRequest{Path: path.Root("annotations"), ConfigValue: tc.value}
			resp := &validator.MapResponse{}
			AnnotationsValidator().ValidateMap(context.Background(), req, resp)

			if got := resp.Diagnostics.ErrorsCount(); got != tc.wantErrors {
				t.Errorf("got %d errors, want %d: %v", got, tc.wantErrors, resp.Diagnostics)
			}
		})
	}
}
