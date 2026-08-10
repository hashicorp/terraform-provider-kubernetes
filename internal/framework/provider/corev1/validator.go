// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	apiValidation "k8s.io/apimachinery/pkg/api/validation"
	utilValidation "k8s.io/apimachinery/pkg/util/validation"
)

var (
	_ validator.String = dnsSubdomainNameValidator{}
	_ validator.String = dnsLabelPrefixValidator{}
	_ validator.Map    = annotationsValidator{}
	_ validator.Map    = labelsValidator{}
)

// dnsSubdomainName validates metadata.name. Ports validateName from
// kubernetes/validators.go.
func dnsSubdomainName() validator.String {
	return dnsSubdomainNameValidator{}
}

type dnsSubdomainNameValidator struct{}

func (v dnsSubdomainNameValidator) Description(ctx context.Context) string {
	return v.MarkdownDescription(ctx)
}

func (v dnsSubdomainNameValidator) MarkdownDescription(_ context.Context) string {
	return "must be a valid DNS subdomain name"
}

func (v dnsSubdomainNameValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	for _, msg := range apiValidation.NameIsDNSSubdomain(req.ConfigValue.ValueString(), false) {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid name", msg)
	}
}

// dnsLabelPrefix validates metadata.generate_name. Ports validateGenerateName.
// The trailing `true` tells apimachinery the value is a name *prefix*, so it is
// validated as such rather than as a complete name.
func dnsLabelPrefix() validator.String {
	return dnsLabelPrefixValidator{}
}

type dnsLabelPrefixValidator struct{}

func (v dnsLabelPrefixValidator) Description(ctx context.Context) string {
	return v.MarkdownDescription(ctx)
}

func (v dnsLabelPrefixValidator) MarkdownDescription(_ context.Context) string {
	return "must be a valid DNS label prefix"
}

func (v dnsLabelPrefixValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	for _, msg := range apiValidation.NameIsDNSLabel(req.ConfigValue.ValueString(), true) {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid generate_name", msg)
	}
}

// annotationKeys validates metadata.annotations. Ports validateAnnotations.
func annotationKeys() validator.Map {
	return annotationsValidator{}
}

type annotationsValidator struct{}

func (v annotationsValidator) Description(ctx context.Context) string {
	return v.MarkdownDescription(ctx)
}

func (v annotationsValidator) MarkdownDescription(_ context.Context) string {
	return "annotation keys must be qualified names"
}

func (v annotationsValidator) ValidateMap(_ context.Context, req validator.MapRequest, resp *validator.MapResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	for k := range req.ConfigValue.Elements() {
		// SDKv2's validateAnnotations lowercases the key before checking;
		// validateLabels does not. Inconsistent, but it is shipped behaviour and
		// changing it here would reject configs that currently apply.
		for _, msg := range utilValidation.IsQualifiedName(strings.ToLower(k)) {
			resp.Diagnostics.AddAttributeError(req.Path.AtMapKey(k), "Invalid annotation key", msg)
		}
	}
}

// labelKeyValues validates metadata.labels. Ports validateLabels, which checks
// both the key and the value.
func labelKeyValues() validator.Map {
	return labelsValidator{}
}

type labelsValidator struct{}

func (v labelsValidator) Description(ctx context.Context) string {
	return v.MarkdownDescription(ctx)
}

func (v labelsValidator) MarkdownDescription(_ context.Context) string {
	return "label keys must be qualified names and values must be valid label values"
}

func (v labelsValidator) ValidateMap(_ context.Context, req validator.MapRequest, resp *validator.MapResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	for k, raw := range req.ConfigValue.Elements() {
		for _, msg := range utilValidation.IsQualifiedName(k) {
			resp.Diagnostics.AddAttributeError(req.Path.AtMapKey(k), "Invalid label key", msg)
		}

		val, ok := raw.(types.String)
		if !ok || val.IsNull() || val.IsUnknown() {
			continue
		}
		for _, msg := range utilValidation.IsValidLabelValue(val.ValueString()) {
			resp.Diagnostics.AddAttributeError(req.Path.AtMapKey(k), "Invalid label value", msg)
		}
	}
}
