// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	utilValidation "k8s.io/apimachinery/pkg/util/validation"
)

// annotationKeysValidator mirrors SDKv2 validateAnnotations, which validates keys
// only and accepts null map values. common.AnnotationsValidator also rejects null
// values, so it is not behavior-compatible for this data source.
type annotationKeysValidator struct{}

func (v annotationKeysValidator) Description(_ context.Context) string {
	return "annotation keys must be qualified names"
}

func (v annotationKeysValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v annotationKeysValidator) ValidateMap(_ context.Context, req validator.MapRequest, resp *validator.MapResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	for key := range req.ConfigValue.Elements() {
		for _, message := range utilValidation.IsQualifiedName(strings.ToLower(key)) {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid annotation key",
				fmt.Sprintf("%q %s", key, message),
			)
		}
	}
}
