package provider

import (
	"context"
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes-alpha/morph"
	"github.com/hashicorp/terraform-provider-kubernetes-alpha/payload"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ReadResource function
func (s *RawProviderServer) ReadResource(ctx context.Context, req *tfprotov5.ReadResourceRequest) (*tfprotov5.ReadResourceResponse, error) {
	resp := &tfprotov5.ReadResourceResponse{}
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
	cu, err := payload.FromTFValue(co, tftypes.NewAttributePath())
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed encode 'object' attribute to Unstructured",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	s.logger.Trace("[ReadResource]", "[unstructured.FromTFValue]", spew.Sdump(cu))

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
			Summary:  fmt.Sprintf("Cannot GET resource %s", spew.Sdump(co)),
			Detail:   err.Error(),
		}
		resp.Diagnostics = append(resp.Diagnostics, &d)
		return resp, nil
	}

	gvk, err := GVKFromTftypesObject(&co, rm)
	if err != nil {
		return resp, fmt.Errorf("failed to determine resource GVR: %s", err)
	}

	objectType, err := s.TFTypeFromOpenAPI(ctx, gvk, false)
	if err != nil {
		return resp, fmt.Errorf("failed to determine resource type ID: %s", err)
	}

	fo := RemoveServerSideFields(ro.Object)
	nobj, err := payload.ToTFValue(fo, objectType, tftypes.NewAttributePath())
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
