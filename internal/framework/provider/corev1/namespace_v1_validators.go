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

// The SDKv2 metadata ValidateFuncs run during terraform validate and plan —
// offline, before any API call. Dropping one moves the failure to an apply-time
// Kubernetes 422 (rule K8S-MIGRATE-003), so each has a counterpart here.
//
// These wrap the same apimachinery functions kubernetes/validators.go calls, at
// the same revision, rather than restating their rules:
//
//	validateAnnotations   -> IsQualifiedName(strings.ToLower(key))
//	validateLabels        -> IsQualifiedName(key) and IsValidLabelValue(value)
//	validateName          -> NameIsDNSSubdomain(v, false)
//	validateGenerateName  -> NameIsDNSLabel(v, true)

// stringFuncValidator adapts a function returning apimachinery's []string error
// messages into a Framework string validator.
type stringFuncValidator struct {
	description string
	validate    func(string) []string
}

func (v stringFuncValidator) Description(_ context.Context) string         { return v.description }
func (v stringFuncValidator) MarkdownDescription(_ context.Context) string { return v.description }

func (v stringFuncValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// Null defers to Optional/Required; unknown cannot be checked until it
	// resolves. Validating either would reject legal configuration.
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	for _, msg := range v.validate(req.ConfigValue.ValueString()) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid value",
			fmt.Sprintf("%s %s", req.Path, msg),
		)
	}
}

// annotationKeyValidator reproduces validateAnnotations, including its
// lowercasing of the key before the qualified-name check.
func annotationKeyValidator() validator.String {
	return stringFuncValidator{
		description: "must be a qualified name",
		validate: func(s string) []string {
			return utilValidation.IsQualifiedName(strings.ToLower(s))
		},
	}
}

// labelKeyValidator reproduces the key half of validateLabels. Unlike
// annotations it does NOT lowercase — that asymmetry is in the SDKv2 source and
// is preserved deliberately.
func labelKeyValidator() validator.String {
	return stringFuncValidator{
		description: "must be a qualified name",
		validate:    utilValidation.IsQualifiedName,
	}
}

// labelValueValidator reproduces the value half of validateLabels.
func labelValueValidator() validator.String {
	return stringFuncValidator{
		description: "must be a valid label value",
		validate:    utilValidation.IsValidLabelValue,
	}
}

// nameValidator reproduces validateName.
func nameValidator() validator.String {
	return stringFuncValidator{
		description: "must be a DNS subdomain name",
		validate: func(s string) []string {
			return apiValidation.NameIsDNSSubdomain(s, false)
		},
	}
}

// generateNameValidator reproduces validateGenerateName. The trailing `true`
// tells apimachinery the value is a generated-name prefix.
func generateNameValidator() validator.String {
	return stringFuncValidator{
		description: "must be a DNS label prefix",
		validate: func(s string) []string {
			return apiValidation.NameIsDNSLabel(s, true)
		},
	}
}
