package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes-alpha/morph"
	"github.com/hashicorp/terraform-provider-kubernetes-alpha/payload"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

// ApplyResourceChange function
func (s *RawProviderServer) ApplyResourceChange(ctx context.Context, req *tfprotov5.ApplyResourceChangeRequest) (*tfprotov5.ApplyResourceChangeResponse, error) {
	resp := &tfprotov5.ApplyResourceChangeResponse{}
	rt, err := GetResourceType(req.TypeName)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to determine planned resource type",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	applyPlannedState, err := req.PlannedState.Unmarshal(rt)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to unmarshal planned resource state",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	s.logger.Trace("[ApplyResourceChange]", "[PlannedState]", spew.Sdump(applyPlannedState))

	applyPriorState, err := req.PriorState.Unmarshal(rt)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to unmarshal prior resource state",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	s.logger.Trace("[ApplyResourceChange]", "[PriorState]", spew.Sdump(applyPriorState))

	c, err := s.getDynamicClient()
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics,
			&tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Failed to retrieve Kubernetes dynamic client during apply",
				Detail:   err.Error(),
			})
		return resp, nil
	}
	m, err := s.getRestMapper()
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics,
			&tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Failed to retrieve Kubernetes RESTMapper client during apply",
				Detail:   err.Error(),
			})
		return resp, nil
	}
	var rs dynamic.ResourceInterface

	switch {
	case applyPriorState.IsNull() || (!applyPlannedState.IsNull() && !applyPriorState.IsNull()):
		// Apply resource
		var plannedStateVal map[string]tftypes.Value = make(map[string]tftypes.Value)
		err = applyPlannedState.As(&plannedStateVal)
		if err != nil {
			resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Failed to extract planned resource state values",
				Detail:   err.Error(),
			})
			return resp, nil
		}
		obj, ok := plannedStateVal["object"]
		if !ok {
			resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Failed to find object value in planned resource state",
			})
			return resp, nil
		}

		gvk, err := GVKFromTftypesObject(&obj, m)
		if err != nil {
			return resp, fmt.Errorf("failed to determine resource GVK: %s", err)
		}

		tsch, err := s.TFTypeFromOpenAPI(ctx, gvk, false)
		if err != nil {
			return resp, fmt.Errorf("failed to determine resource type ID: %s", err)
		}

		minObj := morph.UnknownToNull(obj)
		s.logger.Trace("[ApplyResourceChange][Apply]", "[UnknownToNull]", spew.Sdump(minObj))

		pu, err := payload.FromTFValue(minObj, tftypes.NewAttributePath())
		if err != nil {
			return resp, err
		}
		s.logger.Trace("[ApplyResourceChange][Apply]", "[payload.FromTFValue]", spew.Sdump(pu))

		// remove null attributes - the API doesn't appreciate requests that include them
		rqObj := mapRemoveNulls(pu.(map[string]interface{}))

		uo := unstructured.Unstructured{}
		uo.SetUnstructuredContent(rqObj)
		rnamespace := uo.GetNamespace()
		rname := uo.GetName()
		rnn := types.NamespacedName{Namespace: rnamespace, Name: rname}.String()

		gvr, err := GVRFromUnstructured(&uo, m)
		if err != nil {
			return resp, fmt.Errorf("failed to determine resource GVR: %s", err)
		}

		ns, err := IsResourceNamespaced(gvk, m)
		if err != nil {
			resp.Diagnostics = append(resp.Diagnostics,
				&tfprotov5.Diagnostic{
					Severity: tfprotov5.DiagnosticSeverityError,
					Detail:   err.Error(),
					Summary:  fmt.Sprintf("Failed to discover scope of resource '%s'", rnn),
				})
			return resp, nil
		}

		if ns {
			rs = c.Resource(gvr).Namespace(rnamespace)
		} else {
			rs = c.Resource(gvr)
		}
		jsonManifest, err := uo.MarshalJSON()
		if err != nil {
			resp.Diagnostics = append(resp.Diagnostics,
				&tfprotov5.Diagnostic{
					Severity: tfprotov5.DiagnosticSeverityError,
					Detail:   err.Error(),
					Summary:  fmt.Sprintf("Failed to marshall resource '%s' to JSON", rnn),
				})
			return resp, nil
		}

		// Call the Kubernetes API to create the new resource
		result, err := rs.Patch(ctx, rname, types.ApplyPatchType, jsonManifest, metav1.PatchOptions{FieldManager: "Terraform"})
		if err != nil {
			s.logger.Error("[ApplyResourceChange][Apply]", "API error", spew.Sdump(err), "API response", spew.Sdump(result))
			if status := apierrors.APIStatus(nil); errors.As(err, &status) {
				resp.Diagnostics = append(resp.Diagnostics, APIStatusErrorToDiagnostics(status.Status())...)
			} else {
				resp.Diagnostics = append(resp.Diagnostics,
					&tfprotov5.Diagnostic{
						Severity: tfprotov5.DiagnosticSeverityError,
						Detail:   err.Error(),
						Summary:  fmt.Sprintf(`PATCH for resource "%s" failed to apply`, rnn),
					})
			}
			return resp, nil
		}

		newResObject, err := payload.ToTFValue(RemoveServerSideFields(result.Object), tsch, tftypes.NewAttributePath())
		if err != nil {
			return resp, err
		}
		s.logger.Trace("[ApplyResourceChange][Apply]", "[payload.ToTFValue]", spew.Sdump(newResObject))

		wt, err := s.TFTypeFromOpenAPI(ctx, gvk, true)
		if err != nil {
			return resp, fmt.Errorf("failed to determine resource type ID: %s", err)
		}

		wf, ok := plannedStateVal["wait_for"]
		if ok {
			err = s.waitForCompletion(ctx, wf, rs, rname, wt)
			if err != nil {
				return resp, err
			}
		}

		compObj, err := morph.DeepUnknown(tsch, newResObject, tftypes.NewAttributePath())
		if err != nil {
			return resp, err
		}
		plannedStateVal["object"] = morph.UnknownToNull(compObj)

		newStateVal := tftypes.NewValue(applyPlannedState.Type(), plannedStateVal)
		s.logger.Trace("[ApplyResourceChange][Apply]", "new state value", spew.Sdump(newStateVal))

		newResState, err := tfprotov5.NewDynamicValue(newStateVal.Type(), newStateVal)
		if err != nil {
			return resp, err
		}

		resp.NewState = &newResState

	case applyPlannedState.IsNull():
		// Delete the resource
		priorStateVal := make(map[string]tftypes.Value)
		err = applyPriorState.As(&priorStateVal)
		if err != nil {
			resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Failed to extract prior resource state values",
				Detail:   err.Error(),
			})
			return resp, nil
		}
		pco, ok := priorStateVal["object"]
		if !ok {
			resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Failed to find object value in prior resource state",
			})
			return resp, nil
		}

		pu, err := payload.FromTFValue(pco, tftypes.NewAttributePath())
		if err != nil {
			return resp, err
		}

		uo := unstructured.Unstructured{Object: pu.(map[string]interface{})}
		gvr, err := GVRFromUnstructured(&uo, m)
		if err != nil {
			return resp, err
		}

		gvk, err := GVKFromTftypesObject(&pco, m)
		if err != nil {
			return resp, fmt.Errorf("failed to determine resource GVK: %s", err)
		}

		ns, err := IsResourceNamespaced(gvk, m)
		if err != nil {
			return resp, err
		}

		rnamespace := uo.GetNamespace()
		rname := uo.GetName()

		if ns {
			rs = c.Resource(gvr).Namespace(rnamespace)
		} else {
			rs = c.Resource(gvr)
		}
		err = rs.Delete(ctx, rname, metav1.DeleteOptions{})
		if err != nil {
			rn := types.NamespacedName{Namespace: rnamespace, Name: rname}.String()
			resp.Diagnostics = append(resp.Diagnostics,
				&tfprotov5.Diagnostic{
					Severity: tfprotov5.DiagnosticSeverityError,
					Detail:   err.Error(),
					Summary:  fmt.Sprintf("DELETE resource %s failed: %s", rn, err),
				})
			return resp, nil
		}

		resp.NewState = req.PlannedState
	}
	// force a refresh of the OpenAPI foundry on next use
	// we do this to capture any potentially new resource type that might have been added
	s.OAPIFoundry = nil // this needs to be optimized to refresh only when CRDs are applied (or maybe other schema altering resources too?)

	return resp, nil
}
