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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ReadResource function
func (s *RawProviderServer) ReadResource(ctx context.Context, req *tfprotov5.ReadResourceRequest) (*tfprotov5.ReadResourceResponse, error) {
	resp := &tfprotov5.ReadResourceResponse{}

	// loop private state back in - ATM it's not needed here
	resp.Private = req.Private

	execDiag := s.canExecute()
	if len(execDiag) > 0 {
		resp.Diagnostics = append(resp.Diagnostics, execDiag...)
		return resp, nil
	}

	var resState map[string]tftypes.Value
	var err error
	rt, err := GetResourceType(req.TypeName)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to determine resource type",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	currentState, err := req.CurrentState.Unmarshal(rt)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to decode current state",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	if currentState.IsNull() {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to read resource",
			Detail:   "Incomplete of missing state",
		})
		return resp, nil
	}
	err = currentState.As(&resState)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to extract resource from current state",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	co, hasOb := resState["object"]
	if !hasOb || co.IsNull() {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Current state of resource has no 'object' attribute",
			Detail:   "This should not happen. The state may be incomplete or corrupted.\nIf this error is reproducible, plese report issue to provider maintainers.",
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
	gvk, err := GVKFromTftypesObject(&co, rm)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to determine GroupVersionResource for manifest",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	objectType, th, err := s.TFTypeFromOpenAPI(ctx, gvk, false)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  fmt.Sprintf("Failed to determine resource type from GVK: %s", gvk),
			Detail:   err.Error(),
		})
		return resp, nil
	}

	cu, err := payload.FromTFValue(co, th, tftypes.NewAttributePath())
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed encode 'object' attribute to Unstructured",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	s.logger.Trace("[ReadResource]", "[unstructured.FromTFValue]", dump(cu))

	client, err := s.getDynamicClient()
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "failed to get Dynamic client",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	uo := unstructured.Unstructured{Object: cu.(map[string]interface{})}
	cGVR, err := GVRFromUnstructured(&uo, rm)
	if err != nil {
		return resp, err
	}
	ns, err := IsResourceNamespaced(uo.GroupVersionKind(), rm)
	if err != nil {
		return resp, err
	}
	rcl := client.Resource(cGVR)

	rnamespace := uo.GetNamespace()
	rname := uo.GetName()

	var ro *unstructured.Unstructured
	if ns {
		ro, err = rcl.Namespace(rnamespace).Get(ctx, rname, metav1.GetOptions{})
	} else {
		ro, err = rcl.Get(ctx, rname, metav1.GetOptions{})
	}
	if err != nil {
		if apierrors.IsNotFound(err) {
			return resp, nil
		}
		d := tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  fmt.Sprintf("Cannot GET resource %s", dump(co)),
			Detail:   err.Error(),
		}
		resp.Diagnostics = append(resp.Diagnostics, &d)
		return resp, nil
	}

	fo := RemoveServerSideFields(ro.Object)
	nobj, err := payload.ToTFValue(fo, objectType, th, tftypes.NewAttributePath())
	if err != nil {
		return resp, err
	}

	nobj, err = morph.DeepUnknown(objectType, nobj, tftypes.NewAttributePath())
	if err != nil {
		return resp, err
	}

	rawState := make(map[string]tftypes.Value)
	err = currentState.As(&rawState)
	if err != nil {
		return resp, err
	}
	rawState["object"] = morph.UnknownToNull(nobj)

	nsVal := tftypes.NewValue(currentState.Type(), rawState)
	newState, err := tfprotov5.NewDynamicValue(nsVal.Type(), nsVal)
	if err != nil {
		return resp, err
	}
	resp.NewState = &newState
	return resp, nil
}
