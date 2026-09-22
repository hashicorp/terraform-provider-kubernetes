// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	pathValidation "k8s.io/apimachinery/pkg/api/validation/path"
	utilValidation "k8s.io/apimachinery/pkg/util/validation"
)

// rbacNameValidator mirrors the SDKv2 validateRBACNameFunc, rejecting names
// that are not valid Kubernetes API path segments.
type rbacNameValidator struct{}

func (v rbacNameValidator) Description(_ context.Context) string {
	return "must be a valid Kubernetes API path segment name"
}

func (v rbacNameValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v rbacNameValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	for _, err := range pathValidation.IsValidPathSegmentName(req.ConfigValue.ValueString()) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Value",
			fmt.Sprintf("%s %s", req.Path, err),
		)
	}
}

// annotationKeyValidator mirrors the SDKv2 validateAnnotations function.
// It rejects annotation keys that are not valid qualified names,
// matching the plan-time validation that SDKv2 performed.
type annotationKeyValidator struct{}

func (v annotationKeyValidator) Description(_ context.Context) string {
	return "annotation keys must be valid Kubernetes qualified names"
}

func (v annotationKeyValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v annotationKeyValidator) ValidateMap(_ context.Context, req validator.MapRequest, resp *validator.MapResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	elements := req.ConfigValue.Elements()
	for k, val := range elements {
		if val.IsNull() {
			continue
		}
		errs := utilValidation.IsQualifiedName(strings.ToLower(k))
		for _, e := range errs {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid Annotation Key",
				fmt.Sprintf("%s (%q) %s", req.Path, k, e),
			)
		}
	}
}

// labelValidator mirrors the SDKv2 validateLabels function.
// It rejects label keys that are not valid qualified names
// and label values that are not valid Kubernetes label values.
type labelValidator struct{}

func (v labelValidator) Description(_ context.Context) string {
	return "label keys must be valid Kubernetes qualified names and values must be valid label values"
}

func (v labelValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v labelValidator) ValidateMap(_ context.Context, req validator.MapRequest, resp *validator.MapResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	elements := req.ConfigValue.Elements()
	for k, val := range elements {
		if val.IsNull() {
			continue
		}
		// Validate key as a qualified name.
		for _, e := range utilValidation.IsQualifiedName(k) {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid Label Key",
				fmt.Sprintf("%s (%q) %s", req.Path, k, e),
			)
		}

		if val.IsUnknown() {
			continue
		}
		strVal, ok := val.(types.String)
		if !ok {
			continue
		}
		rawVal := strVal.ValueString()
		for _, e := range utilValidation.IsValidLabelValue(rawVal) {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid Label Value",
				fmt.Sprintf("%s (%q) %s", req.Path, rawVal, e),
			)
		}
	}
}
