// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	apiValidation "k8s.io/apimachinery/pkg/api/validation"
	pathValidation "k8s.io/apimachinery/pkg/api/validation/path"
	utilValidation "k8s.io/apimachinery/pkg/util/validation"
)

// Every diagnostic here is built with validatordiag.InvalidAttributeValueDiagnostic, the
// helper the framework's own validators use. The attribute is taken from req.Path rather
// than named in the message, so these validators report correctly wherever they are
// reused. It renders as:
//
//	Invalid Attribute Value
//	Attribute metadata[0].generate_name <reason>, got: "<value>"
var (
	_ validator.String = dnsSubdomainNameValidator{}
	_ validator.String = dnsLabelPrefixValidator{}
	_ validator.Map    = annotationsValidator{}
	_ validator.Map    = labelsValidator{}
)

// DNSSubdomainNameValidator returns a validator that checks a string is a valid DNS
// subdomain name, as Kubernetes requires for most object names. Ports validateName from
// kubernetes/validators.go.
func DNSSubdomainNameValidator() validator.String {
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
		resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
			req.Path, msg, req.ConfigValue.String()))
	}
}

// DNSLabelPrefixValidator ports validateGenerateName from kubernetes/validators.go.
func DNSLabelPrefixValidator() validator.String {
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
		resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
			req.Path, msg, req.ConfigValue.String()))
	}
}

// AnnotationsValidator Ports validateAnnotations from kubernetes/validators.go.
// Returns a validator that checks every annotation key is a qualified name and every value is non-null.
func AnnotationsValidator() validator.Map {
	return annotationsValidator{}
}

// dataSourceAnnotationsValidator is AnnotationsValidator without the null-value check, for
// data sources. SDKv2 accepted `annotations = { a = null }` there: validateAnnotations
// checks keys only, and validateConfigNulls rejects null elements in lists and sets but not
// in maps. Rejecting it would break configurations that validate today. The resource-side
// reasons for the check do not apply either — a data source has no apply, and its read
// replaces the map with the API response.
func dataSourceAnnotationsValidator() validator.Map {
	return annotationsValidator{allowNullValues: true}
}

type annotationsValidator struct {
	allowNullValues bool
}

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
	for k, raw := range req.ConfigValue.Elements() {
		// SDKv2's validateAnnotations lowercases the key before checking;
		for _, msg := range utilValidation.IsQualifiedName(strings.ToLower(k)) {
			resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
				req.Path.AtMapKey(k), "key "+msg, fmt.Sprintf("%q", k)))
		}

		val, ok := raw.(types.String)
		if !ok {
			continue // unreachable: the schema declares ElementType: types.StringType
		}
		if val.IsUnknown() {
			continue
		}
		// Terraform passes `annotations = { a = null }` through to the provider. Without
		// this check, Create fails at apply with a Value Conversion Error that blames the
		// provider, and Update silently drops the key while state keeps `a = null`,
		// producing a permanent diff. SDKv2's validateAnnotations checks keys only, so
		// this is a deliberate tightening;
		if val.IsNull() && !v.allowNullValues {
			resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
				req.Path.AtMapKey(k), "value must be a string", val.String()))
		}
	}
}

// LabelsValidator ports validateLabels from kubernetes/validators.go.
// Returns a validator that checks every label key is a qualified name and every value is a non-null, valid label value.
func LabelsValidator() validator.Map {
	return labelsValidator{}
}

// dataSourceLabelsValidator is LabelsValidator without the null-value check, for data
// sources. See dataSourceAnnotationsValidator for why.
func dataSourceLabelsValidator() validator.Map {
	return labelsValidator{allowNullValues: true}
}

type labelsValidator struct {
	allowNullValues bool
}

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
			resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
				req.Path.AtMapKey(k), "key "+msg, fmt.Sprintf("%q", k)))
		}

		val, ok := raw.(types.String)
		if !ok {
			continue // unreachable: the schema declares ElementType: types.StringType
		}
		if val.IsUnknown() {
			continue
		}
		if val.IsNull() {
			if !v.allowNullValues {
				resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
					req.Path.AtMapKey(k), "value must be a string", val.String()))
			}
			continue
		}
		for _, msg := range utilValidation.IsValidLabelValue(val.ValueString()) {
			resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
				req.Path.AtMapKey(k), "value "+msg, val.String()))
		}
	}
}

// RBACNameValidator validates a name the way SDKv2's validateRBACNameFunc does
// (kubernetes/schema_rbac.go), with apimachinery's IsValidPathSegmentName rather than
// NameIsDNSSubdomain.
//
// RBAC names are path segments, not DNS subdomains: "system:controller:foo" is a legitimate
// ClusterRole name and the API accepts it. Using the DNS rule here would reject names that
// both Kubernetes and every released provider version accept.
func RBACNameValidator() validator.String {
	return rbacNameValidator{}
}

type rbacNameValidator struct{}

func (v rbacNameValidator) Description(ctx context.Context) string {
	return v.MarkdownDescription(ctx)
}

func (v rbacNameValidator) MarkdownDescription(_ context.Context) string {
	return "must be a valid path segment name"
}

func (v rbacNameValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	for _, msg := range pathValidation.IsValidPathSegmentName(req.ConfigValue.ValueString()) {
		resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
			req.Path, msg, req.ConfigValue.ValueString()))
	}
}
