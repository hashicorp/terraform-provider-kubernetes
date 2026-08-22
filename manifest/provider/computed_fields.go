// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/morph"
)

// backfillComputedFields prepares computed fields for apply payload creation.
// Configured computed fields are populated from manifest, while computed fields
// absent from manifest stay unknown so prior object state is not sent as desired.
func backfillComputedFields(obj tftypes.Value, manifest tftypes.Value, computedFields map[string]*tftypes.AttributePath) (tftypes.Value, []*tfprotov5.Diagnostic, error) {
	var diagnostics []*tfprotov5.Diagnostic

	backfilled, err := tftypes.Transform(obj, func(ap *tftypes.AttributePath, v tftypes.Value) (tftypes.Value, error) {
		_, isComputed := computedFields[ap.String()]
		if !isComputed {
			return v, nil
		}

		ppMan, restPath, err := tftypes.WalkAttributePath(manifest, ap)
		if err != nil {
			if len(restPath.Steps()) > 0 {
				return tftypes.NewValue(v.Type(), tftypes.UnknownValue), nil
			}
			return v, ap.NewError(err)
		}

		nv, d := morph.ValueToType(ppMan.(tftypes.Value), v.Type(), tftypes.NewAttributePath())
		if len(d) > 0 {
			diagnostics = append(diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Manifest configuration is incompatible with resource schema",
				Detail:   "Detailed descriptions of errors will follow below.",
			})
			diagnostics = append(diagnostics, d...)
			return v, nil
		}

		return nv, nil
	})
	if err != nil {
		return obj, diagnostics, err
	}

	return backfilled, diagnostics, nil
}
