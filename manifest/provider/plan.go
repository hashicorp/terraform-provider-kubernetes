package provider

import (
	"context"
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes-alpha/morph"
	"github.com/hashicorp/terraform-provider-kubernetes-alpha/payload"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

func (s *RawProviderServer) dryRun(ctx context.Context, obj tftypes.Value) error {
	c, err := s.getDynamicClient()
	if err != nil {
		return fmt.Errorf("failed to retrieve Kubernetes dynamic client during apply: %v", err)
	}
	m, err := s.getRestMapper()
	if err != nil {
		return fmt.Errorf("failed to retrieve Kubernetes RESTMapper client during apply: %v", err)
	}

	gvk, err := GVKFromTftypesObject(&obj, m)
	if err != nil {
		return fmt.Errorf("failed to determine resource GVK: %s", err)
	}

	minObj := morph.UnknownToNull(obj)
	pu, err := payload.FromTFValue(minObj, tftypes.NewAttributePath())
	if err != nil {
		return err
	}

	rqObj := mapRemoveNulls(pu.(map[string]interface{}))
	uo := unstructured.Unstructured{}
	uo.SetUnstructuredContent(rqObj)
	rnamespace := uo.GetNamespace()
	rname := uo.GetName()
	rnn := types.NamespacedName{Namespace: rnamespace, Name: rname}.String()

	gvr, err := GVRFromUnstructured(&uo, m)
	if err != nil {
		return fmt.Errorf("failed to determine resource GVR: %s", err)
	}

	ns, err := IsResourceNamespaced(gvk, m)
	if err != nil {
		return fmt.Errorf("failed to discover scope of resource %q: %v", rnn, err)
	}

	var rs dynamic.ResourceInterface
	if ns {
		rs = c.Resource(gvr).Namespace(rnamespace)
	} else {
		rs = c.Resource(gvr)
	}

	jsonManifest, err := uo.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshall resource %q to JSON: %v", rnn, err)
	}
	_, err = rs.Patch(ctx, rname, types.ApplyPatchType, jsonManifest,
		metav1.PatchOptions{
			FieldManager: "Terraform",
			DryRun:       []string{"All"},
		},
	)

	return err
}

// PlanResourceChange function
func (s *RawProviderServer) PlanResourceChange(ctx context.Context, req *tfprotov5.PlanResourceChangeRequest) (*tfprotov5.PlanResourceChangeResponse, error) {
	resp := &tfprotov5.PlanResourceChangeResponse{}

	// test if credentials are valid - we're going to need them further down
	resp.Diagnostics = append(resp.Diagnostics, s.checkValidCredentials(ctx)...)
	if len(resp.Diagnostics) > 0 {
		return resp, nil
	}

	rt, err := GetResourceType(req.TypeName)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to determine planned resource type",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	// Decode proposed resource state
	proposedState, err := req.ProposedNewState.Unmarshal(rt)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to unmarshal planned resource state",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	s.logger.Trace("[PlanResourceChange]", "[ProposedState]", spew.Sdump(proposedState))

	proposedVal := make(map[string]tftypes.Value)
	err = proposedState.As(&proposedVal)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to extract planned resource state from tftypes.Value",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	// Decode prior resource state
	priorState, err := req.PriorState.Unmarshal(rt)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to unmarshal prior resource state",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	s.logger.Trace("[PlanResourceChange]", "[PriorState]", spew.Sdump(priorState))

	priorVal := make(map[string]tftypes.Value)
	err = priorState.As(&priorVal)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to extract prior resource state from tftypes.Value",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	if proposedState.IsNull() {
		// we plan to delete the resource
		if _, ok := priorVal["object"]; ok {
			resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Invalid prior state while planning for destroy",
				Detail:   fmt.Sprintf("'object' attribute missing from state: %s", err),
			})
			return resp, nil
		}
		resp.PlannedState = req.ProposedNewState
		return resp, nil
	}

	ppMan, ok := proposedVal["manifest"]
	if !ok {
		matp := tftypes.NewAttributePath()
		matp = matp.WithAttributeName("manifest")
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity:  tfprotov5.DiagnosticSeverityError,
			Summary:   "Invalid proposed state during planning",
			Detail:    "Missing 'manifest' attribute",
			Attribute: matp,
		})
		return resp, nil
	}

	rm, err := s.getRestMapper()
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to create K8s RESTMapper client",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	gvk, err := GVKFromTftypesObject(&ppMan, rm)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to determine GroupVersionResource for manifest",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	vdiags := s.validateResourceOnline(&ppMan)
	if len(vdiags) > 0 {
		resp.Diagnostics = append(resp.Diagnostics, vdiags...)
		return resp, nil
	}

	// Request a complete type for the resource from the OpenAPI spec
	objectType, err := s.TFTypeFromOpenAPI(ctx, gvk, false)
	if err != nil {
		return resp, fmt.Errorf("failed to determine resource type ID: %s", err)
	}

	if !objectType.Is(tftypes.Object{}) {
		// non-structural resources have no schema so we just use the
		// type information we can get from the config
		objectType = ppMan.Type()

		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityWarning,
			Summary:  "This custom resource does not have an associated OpenAPI schema.",
			Detail:   "We could not find an OpenAPI schema for this custom resource. Updates to this resource will cause a forced replacement.",
		})

		err := s.dryRun(ctx, ppMan)
		if err != nil {
			resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Dry-run failed for non-structured resource",
				Detail:   fmt.Sprintf("A dry-run apply was performed for this resource but was unsuccessful: %v", err),
			})
			return resp, nil
		}

		resp.RequiresReplace = []*tftypes.AttributePath{
			tftypes.NewAttributePath().WithAttributeName("manifest"),
			tftypes.NewAttributePath().WithAttributeName("object"),
		}
	}

	so := objectType.(tftypes.Object)
	s.logger.Debug("[PlanUpdateResource]", "OAPI type", spew.Sdump(so))

	// Transform the input manifest to adhere to the type model from the OpenAPI spec
	mobj, err := morph.ValueToType(ppMan, objectType, tftypes.NewAttributePath())
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to morph manifest to OAPI type",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	s.logger.Debug("[PlanResourceChange]", "morphed manifest", spew.Sdump(mobj))

	completeObj, err := morph.DeepUnknown(objectType, mobj, tftypes.NewAttributePath())
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to backfill manifest from OAPI type",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	s.logger.Debug("[PlanResourceChange]", "backfilled manifest", spew.Sdump(completeObj))

	if proposedVal["object"].IsNull() { // plan for Create
		s.logger.Debug("[PlanResourceChange]", "creating object", spew.Sdump(completeObj))
		proposedVal["object"] = completeObj
	} else { // plan for Update
		priorObj, ok := priorVal["object"]
		if !ok {
			oatp := tftypes.NewAttributePath()
			oatp = oatp.WithAttributeName("object")
			resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
				Severity:  tfprotov5.DiagnosticSeverityError,
				Summary:   "Invalid prior state during planning",
				Detail:    "Missing 'object' attribute",
				Attribute: oatp,
			})
			return resp, nil
		}
		updatedObj, err := tftypes.Transform(completeObj, func(ap *tftypes.AttributePath, v tftypes.Value) (tftypes.Value, error) {
			if v.IsKnown() { // this is a value from current configuration - include it in the plan
				return v, nil
			}
			// check if value was present in the previous configuration
			wasVal, restPath, err := tftypes.WalkAttributePath(priorVal["manifest"], ap)
			if err == nil && len(restPath.Steps()) == 0 && wasVal.(tftypes.Value).IsKnown() {
				// attribute was previously set in config and has now been removed
				// return the new unknown value to give the API a chance to set a default
				return v, nil
			}
			// at this point, check if there is a default value in the previous state
			priorAtrVal, restPath, err := tftypes.WalkAttributePath(priorObj, ap)
			if err != nil {
				return v, ap.NewError(err)
			}
			if len(restPath.Steps()) > 0 {
				s.logger.Warn("[PlanResourceChange]", "Unexpected missing attribute from state at", ap.String(), " + ", restPath.String())
			}
			return priorAtrVal.(tftypes.Value), nil
		})
		if err != nil {
			oatp := tftypes.NewAttributePath()
			oatp = oatp.WithAttributeName("object")
			resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
				Severity:  tfprotov5.DiagnosticSeverityError,
				Summary:   "Failed to update proposed state from prior state",
				Detail:    err.Error(),
				Attribute: oatp,
			})
			return resp, nil
		}

		proposedVal["object"] = updatedObj
	}

	id, err := createIDFromManifest(proposedVal["manifest"])
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Could not generate ID for resource",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	proposedVal["id"] = id

	propStateVal := tftypes.NewValue(proposedState.Type(), proposedVal)
	s.logger.Trace("[PlanResourceChange]", "new planned state", spew.Sdump(propStateVal))

	plannedState, err := tfprotov5.NewDynamicValue(propStateVal.Type(), propStateVal)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to assemble proposed state during plan",
			Detail:   err.Error(),
		})
		return resp, nil
	}

	resp.PlannedState = &plannedState
	return resp, nil
}

func getAttributeValue(v tftypes.Value, path string) (tftypes.Value, error) {
	p, err := FieldPathToTftypesPath(path)
	if err != nil {
		return tftypes.Value{}, err
	}
	vv, _, err := tftypes.WalkAttributePath(v, p)
	if err != nil {
		return tftypes.Value{}, err
	}
	return vv.(tftypes.Value), nil
}

func createIDFromManifest(manifest tftypes.Value) (tftypes.Value, error) {
	id := ""
	name, err := getAttributeValue(manifest, "metadata.name")
	if err != nil {
		return tftypes.Value{}, err
	}
	var s string
	name.As(&s)
	id += s
	namespace, err := getAttributeValue(manifest, "metadata.namespace")
	if err != nil {
		return tftypes.NewValue(tftypes.String, id), nil
	}
	namespace.As(&s)
	id += "/" + s
	return tftypes.NewValue(tftypes.String, id), nil
}
