// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func deploymentPodSpecDiagnostics(block schema.NestedBlockObject) {
	if attribute, ok := block.Attributes["divisor"].(schema.StringAttribute); ok {
		for i, validation := range attribute.Validators {
			attribute.Validators[i] = deploymentQuantityDiagnostics{String: validation}
		}
		block.Attributes["divisor"] = attribute
	}
	for _, child := range block.Blocks {
		if child, ok := child.(schema.ListNestedBlock); ok {
			deploymentPodSpecDiagnostics(child.NestedObject)
		}
	}
}

// Keep the original Deployment diagnostic contract without changing the shared
// PodSpec validator's accepted values or unknown-value handling.
type deploymentRestartPolicyDiagnostics struct {
	validator.String
}

func (validation deploymentRestartPolicyDiagnostics) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	var result validator.StringResponse
	validation.String.ValidateString(ctx, req, &result)
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Severity() != diag.SeverityError {
			resp.Diagnostics.Append(diagnostic)
			continue
		}
		legacyPath := strings.NewReplacer("[", ".", "]", "").Replace(req.Path.String())
		resp.Diagnostics.AddAttributeError(req.Path,
			fmt.Sprintf("expected %s to be one of [\"Always\"], got %s", legacyPath, req.ConfigValue.ValueString()), "")
	}
}

type deploymentQuantityDiagnostics struct {
	validator.String
}

func (validation deploymentQuantityDiagnostics) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	var result validator.StringResponse
	validation.String.ValidateString(ctx, req, &result)
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Severity() != diag.SeverityError {
			resp.Diagnostics.Append(diagnostic)
			continue
		}
		// Retain Kubernetes' diagnostic verbatim: a prefixed detail wraps the
		// original error across lines in Terraform's human-readable output.
		resp.Diagnostics.AddAttributeError(req.Path, diagnostic.Summary(),
			strings.TrimPrefix(diagnostic.Detail(), "must be a Kubernetes resource quantity: "))
	}
}
