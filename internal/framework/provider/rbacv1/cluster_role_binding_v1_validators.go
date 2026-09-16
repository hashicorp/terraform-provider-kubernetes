// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	pathValidation "k8s.io/apimachinery/pkg/api/validation/path"
)

type clusterRoleBindingNameValidator struct{}

func (v clusterRoleBindingNameValidator) Description(_ context.Context) string {
	return "must be a valid Kubernetes API path segment name"
}

func (v clusterRoleBindingNameValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v clusterRoleBindingNameValidator) ValidateString(
	_ context.Context,
	req validator.StringRequest,
	resp *validator.StringResponse,
) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	for _, err := range pathValidation.IsValidPathSegmentName(
		req.ConfigValue.ValueString(),
	) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Value",
			fmt.Sprintf("%s %s", req.Path, err),
		)
	}
}
