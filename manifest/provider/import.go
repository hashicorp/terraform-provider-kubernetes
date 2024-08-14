// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/morph"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/payload"
	"github.com/hashicorp/terraform-provider-kubernetes/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ImportResourceState function
func (s *RawProviderServer) ImportResourceState(ctx context.Context, req *tfprotov5.ImportResourceStateRequest) (*tfprotov5.ImportResourceStateResponse, error) {
	// Terraform only gives us the schema name of the resource and an ID string, as passed by the user on the command line.
	// The ID should be a combination of a Kubernetes GVK and a namespace/name type of resource identifier.
	// Without the user supplying the GRV there is no way to fully identify the resource when making the Get API call to K8s.
	// Presumably the Kubernetes API machinery already has a standard for expressing such a group. We should look there first.
	resp := &tfprotov5.ImportResourceStateResponse{}

	cp := req.ClientCapabilities
	if cp != nil && cp.DeferralAllowed && s.clientConfigUnknown {
		// if client support it, request deferral when client configuration not fully known
		resp.Deferred = &tfprotov5.Deferred{
			Reason: tfprotov5.DeferredReasonProviderConfigUnknown,
		}
		return resp, nil
	}

	execDiag := s.canExecute()
	if len(execDiag) > 0 {
		resp.Diagnostics = append(resp.Diagnostics, execDiag...)
		return resp, nil
	}

	gvk, name, namespace, err := util.ParseResourceID(req.ID)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to parse import ID",
			Detail:   err.Error(),
		})
	}
	s.logger.Trace("[ImportResourceState]", "[ID]", gvk, name, namespace)
	rt, err := GetResourceType(req.TypeName)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to determine resource type",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	rm, err := s.getRestMapper()
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to get RESTMapper client",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	client, err := s.getDynamicClient()
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "failed to get Dynamic client",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	ns, err := IsResourceNamespaced(gvk, rm)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to get namespacing requirement from RESTMapper",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	io := unstructured.Unstructured{}
	io.SetKind(gvk.Kind)
	io.SetAPIVersion(gvk.GroupVersion().String())
	io.SetName(name)
	io.SetNamespace(namespace)

	gvr, err := GVRFromUnstructured(&io, rm)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to get GVR from GVK via RESTMapper",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	rcl := client.Resource(gvr)

	var ro *unstructured.Unstructured
	if ns {
		ro, err = rcl.Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	} else {
		ro, err = rcl.Get(ctx, name, metav1.GetOptions{})
	}
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  fmt.Sprintf("Failed to get resource %+v from API", io),
			Detail:   err.Error(),
		})
		return resp, nil
	}
	s.logger.Trace("[ImportResourceState]", "[API Resource]", ro)

	objectType, th, err := s.TFTypeFromOpenAPI(ctx, gvk, false)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  fmt.Sprintf("Failed to determine resource type from GVK: %s", gvk),
			Detail:   err.Error(),
		})
		return resp, nil
	}

	fo := RemoveServerSideFields(ro.UnstructuredContent())
	nobj, err := payload.ToTFValue(fo, objectType, th, tftypes.NewAttributePath())
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to convert unstructured to tftypes.Value",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	nobj, err = morph.DeepUnknown(objectType, nobj, tftypes.NewAttributePath())
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to backfill unknown values during import",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	s.logger.Trace("[ImportResourceState]", "[tftypes.Value]", nobj)

	newState := make(map[string]tftypes.Value)
	wftype := rt.(tftypes.Object).AttributeTypes["wait_for"]
	wtype := rt.(tftypes.Object).AttributeTypes["wait"]
	timeoutsType := rt.(tftypes.Object).AttributeTypes["timeouts"]
	fmType := rt.(tftypes.Object).AttributeTypes["field_manager"]
	cmpType := rt.(tftypes.Object).AttributeTypes["computed_fields"]

	newState["manifest"] = tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{}}, nil)
	newState["object"] = morph.UnknownToNull(nobj)
	newState["wait_for"] = tftypes.NewValue(wftype, nil)
	newState["wait"] = tftypes.NewValue(wtype, nil)
	newState["timeouts"] = tftypes.NewValue(timeoutsType, nil)
	newState["field_manager"] = tftypes.NewValue(fmType, nil)
	newState["computed_fields"] = tftypes.NewValue(cmpType, nil)

	nsVal := tftypes.NewValue(rt, newState)

	impState, err := tfprotov5.NewDynamicValue(nsVal.Type(), nsVal)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to construct dynamic value for imported state",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	impf := tftypes.NewValue(privateStateSchema,
		map[string]tftypes.Value{"IsImported": tftypes.NewValue(tftypes.Bool, true)},
	)
	fb, err := impf.MarshalMsgPack(privateStateSchema)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityWarning,
			Summary:  "Failed to earmark imported resource",
			Detail:   err.Error(),
		})
	}
	nr := &tfprotov5.ImportedResource{
		TypeName: req.TypeName,
		State:    &impState,
		Private:  fb,
	}
	resp.ImportedResources = append(resp.ImportedResources, nr)
	resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
		Severity: tfprotov5.DiagnosticSeverityWarning,
		Summary:  "Apply needed after 'import'",
		Detail:   "Please run apply after a successful import to realign the resource state to the configuration in Terraform.",
	})
	return resp, nil
}
