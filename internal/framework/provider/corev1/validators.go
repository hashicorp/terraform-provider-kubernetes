// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	apiValidation "k8s.io/apimachinery/pkg/api/validation"
	utilValidation "k8s.io/apimachinery/pkg/util/validation"
)

// ── annotationKeysValidator ───────────────────────────────────────────────────

// annotationKeysValidator checks that every key in an annotations map is a
// qualified name, matching the SDKv2 validateAnnotations behaviour.
type annotationKeysValidator struct{}

func (v annotationKeysValidator) Description(_ context.Context) string {
	return "annotation keys must be qualified names: an optional DNS subdomain prefix followed by a name"
}

func (v annotationKeysValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v annotationKeysValidator) ValidateMap(_ context.Context, req validator.MapRequest, resp *validator.MapResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	elements := req.ConfigValue.Elements()
	for k := range elements {
		if errs := utilValidation.IsQualifiedName(strings.ToLower(k)); len(errs) > 0 {
			for _, e := range errs {
				resp.Diagnostics.AddAttributeError(
					req.Path,
					"Invalid annotation key",
					fmt.Sprintf("%q %s", k, e),
				)
			}
		}
	}
}

// ── labelKeysAndValuesValidator ───────────────────────────────────────────────

// labelKeysAndValuesValidator checks that every key in a labels map is a
// qualified name and every value is a valid label value, matching the SDKv2
// validateLabels behaviour.
type labelKeysAndValuesValidator struct{}

func (v labelKeysAndValuesValidator) Description(_ context.Context) string {
	return "label keys must be qualified names and values must be valid Kubernetes label values"
}

func (v labelKeysAndValuesValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v labelKeysAndValuesValidator) ValidateMap(_ context.Context, req validator.MapRequest, resp *validator.MapResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	elements := req.ConfigValue.Elements()
	for k, val := range elements {
		if errs := utilValidation.IsQualifiedName(k); len(errs) > 0 {
			for _, e := range errs {
				resp.Diagnostics.AddAttributeError(
					req.Path,
					"Invalid label key",
					fmt.Sprintf("%q %s", k, e),
				)
			}
		}
		if errs := utilValidation.IsValidLabelValue(strings.Trim(val.String(), "\"")); len(errs) > 0 {
			for _, e := range errs {
				resp.Diagnostics.AddAttributeError(
					req.Path,
					"Invalid label value",
					fmt.Sprintf("%q %s", val, e),
				)
			}
		}
	}
}

// ── dnsSubdomainValidator ─────────────────────────────────────────────────────

// dnsSubdomainValidator checks that a name is a valid DNS subdomain, matching
// the SDKv2 validateName behaviour (apiValidation.NameIsDNSSubdomain).
type dnsSubdomainValidator struct{}

func (v dnsSubdomainValidator) Description(_ context.Context) string {
	return "must be a valid DNS subdomain: lowercase alphanumeric characters, '-' or '.', start and end with alphanumeric"
}

func (v dnsSubdomainValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v dnsSubdomainValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueString()
	if errs := apiValidation.NameIsDNSSubdomain(val, false); len(errs) > 0 {
		for _, e := range errs {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid resource name",
				fmt.Sprintf("%s %s", req.Path, e),
			)
		}
	}
}

// ── dnsLabelValidator ─────────────────────────────────────────────────────────

// dnsLabelValidator checks that a generate_name prefix is a valid DNS label,
// matching the SDKv2 validateGenerateName behaviour (apiValidation.NameIsDNSLabel).
type dnsLabelValidator struct{}

func (v dnsLabelValidator) Description(_ context.Context) string {
	return "must be a valid DNS label prefix: lowercase alphanumeric characters or '-', start with alphanumeric"
}

func (v dnsLabelValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v dnsLabelValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueString()
	if errs := apiValidation.NameIsDNSLabel(val, true); len(errs) > 0 {
		for _, e := range errs {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid generate_name prefix",
				fmt.Sprintf("%s %s", req.Path, e),
			)
		}
	}
}
