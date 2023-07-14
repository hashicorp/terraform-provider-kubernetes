// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// APIStatusErrorToDiagnostics converts an Kubernetes API machinery StatusError into Terraform Diagnostics
func APIStatusErrorToDiagnostics(s metav1.Status) []*tfprotov5.Diagnostic {
	var diags []*tfprotov5.Diagnostic
	diags = append(diags, &tfprotov5.Diagnostic{
		Severity: tfprotov5.DiagnosticSeverityError,
		Summary:  "API response status: " + s.Status,
		Detail:   s.Message,
	})
	if s.Details == nil {
		return diags
	}
	gk := metav1.GroupKind{Group: s.Details.Group, Kind: s.Details.Kind}
	diags = append(diags, &tfprotov5.Diagnostic{
		Severity: tfprotov5.DiagnosticSeverityError,
		Summary:  fmt.Sprintf("Kubernetes API Error: %s %s [%s]", string(s.Reason), gk.String(), s.Details.Name),
	})
	for _, c := range s.Details.Causes {
		diags = append(diags, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Detail:   c.Message,
			Summary:  c.Field,
		})
	}
	return diags
}
