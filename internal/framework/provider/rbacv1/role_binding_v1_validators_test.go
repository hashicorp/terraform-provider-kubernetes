// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValidators_RBACName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		val       types.String
		wantError bool
	}{
		{"null", types.StringNull(), false},
		{"unknown", types.StringUnknown(), false},
		{"valid simple", types.StringValue("admin"), false},
		{"valid with hyphen and dot", types.StringValue("my-role.binding"), false},
		{"invalid with slash", types.StringValue("invalid/name"), true},
		{"invalid with percent", types.StringValue("invalid%name"), true},
		{"invalid dot prefix/only", types.StringValue("."), true},
		{"invalid dot dot", types.StringValue(".."), true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var v rbacNameValidator
			req := validator.StringRequest{
				Path:        path.Root("metadata").AtName("name"),
				ConfigValue: tc.val,
			}
			var resp validator.StringResponse
			v.ValidateString(context.Background(), req, &resp)

			if tc.wantError && !resp.Diagnostics.HasError() {
				t.Errorf("expected error for %v, got none", tc.val)
			}
			if !tc.wantError && resp.Diagnostics.HasError() {
				t.Errorf("unexpected error for %v: %s", tc.val, resp.Diagnostics)
			}
		})
	}
}

func TestValidators_AnnotationKey(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		val       types.Map
		wantError bool
	}{
		{"null", types.MapNull(types.StringType), false},
		{"unknown", types.MapUnknown(types.StringType), false},
		{
			"valid simple",
			types.MapValueMust(types.StringType, map[string]attr.Value{
				"example.com/note": types.StringValue("val"),
			}),
			false,
		},
		{
			"valid key with uppercase (lowercased before IsQualifiedName per SDKv2)",
			types.MapValueMust(types.StringType, map[string]attr.Value{
				"Example.com/Note": types.StringValue("val"),
			}),
			false,
		},
		{
			"invalid key with spaces and special characters",
			types.MapValueMust(types.StringType, map[string]attr.Value{
				"Not A Valid Key!": types.StringValue("val"),
			}),
			true,
		},
		{
			"invalid key with invalid characters",
			types.MapValueMust(types.StringType, map[string]attr.Value{
				"invalid_key_with_@#$": types.StringValue("val"),
			}),
			true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var v annotationKeyValidator
			req := validator.MapRequest{
				Path:        path.Root("metadata").AtName("annotations"),
				ConfigValue: tc.val,
			}
			var resp validator.MapResponse
			v.ValidateMap(context.Background(), req, &resp)

			if tc.wantError && !resp.Diagnostics.HasError() {
				t.Errorf("expected error for %v, got none", tc.val)
			}
			if !tc.wantError && resp.Diagnostics.HasError() {
				t.Errorf("unexpected error for %v: %s", tc.val, resp.Diagnostics)
			}
		})
	}
}

func TestValidators_Label(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		val       types.Map
		wantError bool
	}{
		{"null", types.MapNull(types.StringType), false},
		{"unknown", types.MapUnknown(types.StringType), false},
		{
			"valid key and value",
			types.MapValueMust(types.StringType, map[string]attr.Value{
				"app.kubernetes.io/name": types.StringValue("my-app"),
			}),
			false,
		},
		{
			"invalid key",
			types.MapValueMust(types.StringType, map[string]attr.Value{
				"Not A Valid Label Key!": types.StringValue("val"),
			}),
			true,
		},
		{
			"invalid value",
			types.MapValueMust(types.StringType, map[string]attr.Value{
				"app": types.StringValue("Not A Valid Label Value!"),
			}),
			true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var v labelValidator
			req := validator.MapRequest{
				Path:        path.Root("metadata").AtName("labels"),
				ConfigValue: tc.val,
			}
			var resp validator.MapResponse
			v.ValidateMap(context.Background(), req, &resp)

			if tc.wantError && !resp.Diagnostics.HasError() {
				t.Errorf("expected error for %v, got none", tc.val)
			}
			if !tc.wantError && resp.Diagnostics.HasError() {
				t.Errorf("unexpected error for %v: %s", tc.val, resp.Diagnostics)
			}
		})
	}
}
