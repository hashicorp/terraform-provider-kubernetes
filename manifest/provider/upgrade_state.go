// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/morph"
)

// UpgradeResourceState isn't really useful in this provider, but we have to loop the state back through to keep Terraform happy.
func (s *RawProviderServer) UpgradeResourceState(ctx context.Context, req *tfprotov6.UpgradeResourceStateRequest) (*tfprotov6.UpgradeResourceStateResponse, error) {
	resp := &tfprotov6.UpgradeResourceStateResponse{}
	resp.Diagnostics = []*tfprotov6.Diagnostic{}

	sch := GetProviderResourceSchema()
	rt := GetObjectTypeFromSchema(sch[req.TypeName])

	rv, err := req.RawState.Unmarshal(rt)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov6.Diagnostic{
			Severity: tfprotov6.DiagnosticSeverityError,
			Summary:  "Failed to unmarshal old state during upgrade",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	// test if credentials are valid - we're going to need them further down
	// if no credentials found, just loop the current state back in
	// we do this to work around https://github.com/hashicorp/terraform/issues/30460
	cd := s.checkValidCredentials(ctx)
	if len(cd) > 0 {
		us, err := tfprotov6.NewDynamicValue(rt, rv)
		if err != nil {
			resp.Diagnostics = append(resp.Diagnostics, &tfprotov6.Diagnostic{
				Severity: tfprotov6.DiagnosticSeverityError,
				Summary:  "Failed to encode new state during upgrade",
				Detail:   err.Error(),
			})
		}
		resp.UpgradedState = &us

		return resp, nil
	}

	var cs map[string]tftypes.Value
	err = rv.As(&cs)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov6.Diagnostic{
			Severity: tfprotov6.DiagnosticSeverityError,
			Summary:  "Failed to extract values from old state during upgrade",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	obj, ok := cs["object"]
	if !ok {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov6.Diagnostic{
			Severity: tfprotov6.DiagnosticSeverityError,
			Summary:  "Failed to find object value in existing resource state",
		})
		return resp, nil
	}

	m, err := s.getRestMapper()
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics,
			&tfprotov6.Diagnostic{
				Severity: tfprotov6.DiagnosticSeverityError,
				Summary:  "Failed to retrieve Kubernetes RESTMapper client during state upgrade",
				Detail:   err.Error(),
			})
		return resp, nil
	}

	gvk, err := GVKFromTftypesObject(&obj, m)
	if err != nil {
		return resp, fmt.Errorf("failed to determine resource GVK: %s", err)
	}

	tsch, _, err := s.TFTypeFromOpenAPI(ctx, gvk, false)
	if err != nil {
		return resp, fmt.Errorf("failed to determine resource type ID: %s", err)
	}

	morphedObject, d := morph.ValueToType(obj, tsch, tftypes.NewAttributePath())
	if len(d) > 0 {
		resp.Diagnostics = append(resp.Diagnostics, d...)
		for i := range d {
			if d[i].Severity == tfprotov6.DiagnosticSeverityError {
				return resp, nil
			}
		}
	}
	s.logger.Debug("[UpgradeResourceState]", "morphed object", dump(morphedObject))

	cs["object"] = obj

	newStateVal := tftypes.NewValue(rv.Type(), cs)

	us, err := tfprotov6.NewDynamicValue(rt, newStateVal)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov6.Diagnostic{
			Severity: tfprotov6.DiagnosticSeverityError,
			Summary:  "Failed to encode new state during upgrade",
			Detail:   err.Error(),
		})
	}
	resp.UpgradedState = &us

	return resp, nil
}
