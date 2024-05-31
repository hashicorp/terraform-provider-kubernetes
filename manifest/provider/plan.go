// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/morph"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/payload"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

func (s *RawProviderServer) dryRun(ctx context.Context, obj tftypes.Value, fieldManager string, forceConflicts bool, isNamespaced bool) error {
	c, err := s.getDynamicClient()
	if err != nil {
		return fmt.Errorf("failed to retrieve Kubernetes dynamic client during apply: %v", err)
	}
	m, err := s.getRestMapper()
	if err != nil {
		return fmt.Errorf("failed to retrieve Kubernetes RESTMapper client during apply: %v", err)
	}

	minObj := morph.UnknownToNull(obj)
	pu, err := payload.FromTFValue(minObj, nil, tftypes.NewAttributePath())
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

	var rs dynamic.ResourceInterface
	if isNamespaced {
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
			FieldManager: fieldManager,
			Force:        &forceConflicts,
			DryRun:       []string{"All"},
		},
	)

	return err
}

const defaultFieldManagerName = "Terraform"

func (s *RawProviderServer) getFieldManagerConfig(v map[string]tftypes.Value) (string, bool, error) {
	fieldManagerName := defaultFieldManagerName
	forceConflicts := false
	if !v["field_manager"].IsNull() && v["field_manager"].IsKnown() {
		var fieldManagerBlock []tftypes.Value
		err := v["field_manager"].As(&fieldManagerBlock)
		if err != nil {
			return "", false, err
		}
		if len(fieldManagerBlock) > 0 {
			var fieldManagerObj map[string]tftypes.Value
			err := fieldManagerBlock[0].As(&fieldManagerObj)
			if err != nil {
				return "", false, err
			}
			if !fieldManagerObj["name"].IsNull() && fieldManagerObj["name"].IsKnown() {
				err = fieldManagerObj["name"].As(&fieldManagerName)
				if err != nil {
					return "", false, err
				}
			}
			if !fieldManagerObj["force_conflicts"].IsNull() && fieldManagerObj["force_conflicts"].IsKnown() {
				err = fieldManagerObj["force_conflicts"].As(&forceConflicts)
				if err != nil {
					return "", false, err
				}
			}
		}
	}
	return fieldManagerName, forceConflicts, nil
}

func isImportedFlagFromPrivate(p []byte) (f bool, d []*tfprotov5.Diagnostic) {
	if len(p) == 0 {
		return
	}
	ps, err := getPrivateStateValue(p)
	if err != nil {
		d = append(d, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Unexpected format for private state",
			Detail:   err.Error(),
		})
	}
	err = ps["IsImported"].As(&f)
	if err != nil {
		d = append(d, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Unexpected format for import flag in private state",
			Detail:   err.Error(),
		})
	}
	return
}

// PlanResourceChange function
func (s *RawProviderServer) PlanResourceChange(ctx context.Context, req *tfprotov5.PlanResourceChangeRequest) (*tfprotov5.PlanResourceChangeResponse, error) {
	resp := &tfprotov5.PlanResourceChangeResponse{}

	canDeferr := req.ClientCapabilities != nil && req.ClientCapabilities.DeferralAllowed

	if canDeferr && s.clientConfigUnknown {
		// if client supports it, request deferral when client configuration not fully known
		resp.Deferred = &tfprotov5.Deferred{
			Reason: tfprotov5.DeferredReasonProviderConfigUnknown,
		}
		return resp, nil
	}

	isImported, d := isImportedFlagFromPrivate(req.PriorPrivate)
	resp.Diagnostics = append(resp.Diagnostics, d...)
	if !isImported {
		resp.RequiresReplace = append(resp.RequiresReplace,
			tftypes.NewAttributePath().WithAttributeName("manifest").WithAttributeName("apiVersion"),
			tftypes.NewAttributePath().WithAttributeName("manifest").WithAttributeName("kind"),
			tftypes.NewAttributePath().WithAttributeName("manifest").WithAttributeName("metadata").WithAttributeName("name"),
		)
	} else {
		resp.PlannedPrivate = req.PriorPrivate
	}

	execDiag := s.canExecute()
	if len(execDiag) > 0 {
		resp.Diagnostics = append(resp.Diagnostics, execDiag...)
		return resp, nil
	}

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
	s.logger.Trace("[PlanResourceChange]", "[ProposedState]", dump(proposedState))

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

	computedFields := make(map[string]*tftypes.AttributePath)
	var atp *tftypes.AttributePath
	cfVal, ok := proposedVal["computed_fields"]
	if ok && !cfVal.IsNull() && cfVal.IsKnown() {
		var cf []tftypes.Value
		cfVal.As(&cf)
		for _, v := range cf {
			var vs string
			err := v.As(&vs)
			if err != nil {
				s.logger.Error("[computed_fields] cannot extract element from list")
				continue
			}
			atp, err := FieldPathToTftypesPath(vs)
			if err != nil {
				s.logger.Error("[Configure]", "[computed_fields] cannot parse filed path element", err)
				resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
					Severity: tfprotov5.DiagnosticSeverityError,
					Summary:  "[computed_fields] cannot parse field path element: " + vs,
					Detail:   err.Error(),
				})
				continue
			}
			computedFields[atp.String()] = atp
		}
	} else {
		// When not specified by the user, 'metadata.annotations' and 'metadata.labels' are configured as default
		atp = tftypes.NewAttributePath().WithAttributeName("metadata").WithAttributeName("annotations")
		computedFields[atp.String()] = atp

		atp = tftypes.NewAttributePath().WithAttributeName("metadata").WithAttributeName("labels")
		computedFields[atp.String()] = atp
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
	s.logger.Trace("[PlanResourceChange]", "[PriorState]", dump(priorState))

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
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity:  tfprotov5.DiagnosticSeverityError,
			Summary:   "Invalid proposed state during planning",
			Detail:    "Missing 'manifest' attribute",
			Attribute: tftypes.NewAttributePath().WithAttributeName("manifest"),
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
		rd := &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "API did not recognize GroupVersionKind from manifest (CRD may not be installed)",
			Detail:   err.Error(),
		}
		resp.Diagnostics = append(resp.Diagnostics, rd)
		if canDeferr && meta.IsNoMatchError(err) {
			// request deferral when client configuration not fully known
			resp.Deferred = &tfprotov5.Deferred{
				Reason: tfprotov5.DeferredReasonResourceConfigUnknown,
			}
			rd.Severity = tfprotov5.DiagnosticSeverityWarning
		}
		return resp, nil
	}

	vdiags := s.validateResourceOnline(&ppMan)
	if len(vdiags) > 0 {
		resp.Diagnostics = append(resp.Diagnostics, vdiags...)
		return resp, nil
	}

	ns, err := IsResourceNamespaced(gvk, rm)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to discover scope of resource",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	if ns && !isImported {
		resp.RequiresReplace = append(resp.RequiresReplace,
			tftypes.NewAttributePath().WithAttributeName("manifest").WithAttributeName("metadata").WithAttributeName("namespace"),
		)
	}

	// Request a complete type for the resource from the OpenAPI spec
	objectType, hints, err := s.TFTypeFromOpenAPI(ctx, gvk, false)
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

		fieldManagerName, forceConflicts, err := s.getFieldManagerConfig(proposedVal)
		if err != nil {
			resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Could not extract field_manager config",
				Detail:   err.Error(),
			})
			return resp, nil
		}

		err = s.dryRun(ctx, ppMan, fieldManagerName, forceConflicts, ns)
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
	s.logger.Debug("[PlanUpdateResource]", "OAPI type", dump(so))

	// Transform the input manifest to adhere to the type model from the OpenAPI spec
	morphedManifest, d := morph.ValueToType(ppMan, objectType, tftypes.NewAttributePath().WithAttributeName("object"))
	if len(d) > 0 {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Manifest configuration incompatible with resource schema",
			Detail:   "Detailed descriptions of errors will follow below.",
		})
		resp.Diagnostics = append(resp.Diagnostics, d...)
		return resp, nil
	}
	s.logger.Debug("[PlanResourceChange]", "morphed manifest", dump(morphedManifest))

	completePropMan, err := morph.DeepUnknown(objectType, morphedManifest, tftypes.NewAttributePath().WithAttributeName("object"))
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity:  tfprotov5.DiagnosticSeverityError,
			Summary:   "Failed to backfill manifest from OpenAPI type",
			Detail:    fmt.Sprintf("This usually happens when the provider cannot fully process the schema retrieved from cluster. Please report this to the provider maintainers.\nError: %s", err.Error()),
			Attribute: tftypes.NewAttributePath().WithAttributeName("object"),
		})
		return resp, nil
	}
	s.logger.Debug("[PlanResourceChange]", "backfilled manifest", dump(completePropMan))

	if proposedVal["object"].IsNull() {
		// plan for Create
		s.logger.Debug("[PlanResourceChange]", "creating object", dump(completePropMan))
		newObj, err := tftypes.Transform(completePropMan, func(ap *tftypes.AttributePath, v tftypes.Value) (tftypes.Value, error) {
			_, ok := computedFields[ap.String()]
			if ok {
				return tftypes.NewValue(v.Type(), tftypes.UnknownValue), nil
			}
			return v, nil
		})
		if err != nil {
			resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
				Severity:  tfprotov5.DiagnosticSeverityError,
				Summary:   "Failed to set computed attributes in new resource state",
				Detail:    err.Error(),
				Attribute: tftypes.NewAttributePath().WithAttributeName("object"),
			})
			return resp, nil
		}
		proposedVal["object"] = newObj
	} else {
		// plan for Update
		priorObj, ok := priorVal["object"]
		if !ok {
			resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
				Severity:  tfprotov5.DiagnosticSeverityError,
				Summary:   "Invalid prior state during planning",
				Detail:    "Missing 'object' attribute",
				Attribute: tftypes.NewAttributePath().WithAttributeName("object"),
			})
			return resp, nil
		}
		priorMan, ok := priorVal["manifest"]
		if !ok {
			resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
				Severity:  tfprotov5.DiagnosticSeverityError,
				Summary:   "Invalid prior state during planning",
				Detail:    "Missing 'manifest' attribute",
				Attribute: tftypes.NewAttributePath().WithAttributeName("manifest"),
			})
			return resp, nil
		}
		updatedObj, err := tftypes.Transform(completePropMan, func(ap *tftypes.AttributePath, v tftypes.Value) (tftypes.Value, error) {
			_, isComputed := computedFields[ap.String()]
			if v.IsKnown() { // this is a value from current configuration - include it in the plan
				hasChanged := false
				wasCfg, restPath, err := tftypes.WalkAttributePath(priorMan, ap)
				if err != nil && len(restPath.Steps()) != 0 {
					hasChanged = true
				}
				nowCfg, restPath, err := tftypes.WalkAttributePath(ppMan, ap)
				hasChanged = err == nil && len(restPath.Steps()) == 0 && wasCfg.(tftypes.Value).IsKnown() && !wasCfg.(tftypes.Value).Equal(nowCfg.(tftypes.Value))
				if hasChanged {
					h, ok := hints[morph.ValueToTypePath(ap).String()]
					if ok && h == manifest.PreserveUnknownFieldsLabel {
						apm := append(tftypes.NewAttributePath().WithAttributeName("manifest").Steps(), ap.Steps()...)
						resp.RequiresReplace = append(resp.RequiresReplace, tftypes.NewAttributePathWithSteps(apm))
					}
				}
				if isComputed {
					if hasChanged {
						return tftypes.NewValue(v.Type(), tftypes.UnknownValue), nil
					}
					nowVal, restPath, err := tftypes.WalkAttributePath(proposedVal["object"], ap)
					if err == nil && len(restPath.Steps()) == 0 {
						return nowVal.(tftypes.Value), nil
					}
				}
				return v, nil
			}
			// check if value was present in the previous configuration
			wasVal, restPath, err := tftypes.WalkAttributePath(priorMan, ap)
			if err == nil && len(restPath.Steps()) == 0 && wasVal.(tftypes.Value).IsKnown() {
				// attribute was previously set in config and has now been removed
				// return the new unknown value to give the API a chance to set a default
				return v, nil
			}
			// at this point, check if there is a default value in the previous state
			priorAtrVal, restPath, err := tftypes.WalkAttributePath(priorObj, ap)
			if err != nil {
				if len(restPath.Steps()) > 0 {
					// attribute wasn't present, but part of its parent path is.
					// just stay on course and use the proposed value.
					return v, nil
				}
				// the entire attribute path is was not found - this should not happen
				// unless the path is totally foreign to the resource type. Return error.
				return v, ap.NewError(err)
			}
			if len(restPath.Steps()) > 0 {
				s.logger.Warn("[PlanResourceChange]", "Unexpected missing attribute from state at", ap.String(), " + ", restPath.String())
			}
			return priorAtrVal.(tftypes.Value), nil
		})
		if err != nil {
			resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
				Severity:  tfprotov5.DiagnosticSeverityError,
				Summary:   "Failed to update proposed state from prior state",
				Detail:    err.Error(),
				Attribute: tftypes.NewAttributePath().WithAttributeName("object"),
			})
			return resp, nil
		}

		proposedVal["object"] = updatedObj
	}

	propStateVal := tftypes.NewValue(proposedState.Type(), proposedVal)
	s.logger.Trace("[PlanResourceChange]", "new planned state", dump(propStateVal))

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
