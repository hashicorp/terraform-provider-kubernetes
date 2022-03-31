package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/morph"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/payload"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

var defaultCreateTimeout = "10m"
var defaultUpdateTimeout = "10m"
var defaultDeleteTimeout = "10m"

// ApplyResourceChange function
func (s *RawProviderServer) ApplyResourceChange(ctx context.Context, req *tfprotov5.ApplyResourceChangeRequest) (*tfprotov5.ApplyResourceChangeResponse, error) {
	resp := &tfprotov5.ApplyResourceChangeResponse{}

	execDiag := s.canExecute()
	if len(execDiag) > 0 {
		resp.Diagnostics = append(resp.Diagnostics, execDiag...)
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

	applyPlannedState, err := req.PlannedState.Unmarshal(rt)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to unmarshal planned resource state",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	s.logger.Trace("[ApplyResourceChange][PlannedState] %#v", applyPlannedState)

	applyPriorState, err := req.PriorState.Unmarshal(rt)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to unmarshal prior resource state",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	s.logger.Trace("[ApplyResourceChange]", "[PriorState]", dump(applyPriorState))

	config, err := req.Config.Unmarshal(rt)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to unmarshal manifest configuration",
			Detail:   err.Error(),
		})
		return resp, nil
	}
	confVals := make(map[string]tftypes.Value)
	err = config.As(&confVals)
	if err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
			Severity: tfprotov5.DiagnosticSeverityError,
			Summary:  "Failed to extract attributes from resource configuration",
			Detail:   err.Error(),
		})
		return resp, nil
	}

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

	// Extract computed fields configuration
	computedFields := make(map[string]*tftypes.AttributePath)
	var atp *tftypes.AttributePath
	cfVal, ok := plannedStateVal["computed_fields"]
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
				s.logger.Error("[Configure]", "[computed_fields] cannot parse field path element", err)
				resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
					Severity: tfprotov5.DiagnosticSeverityError,
					Summary:  "[computed_fields] cannot parse filed path element: " + vs,
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

		tsch, th, err := s.TFTypeFromOpenAPI(ctx, gvk, false)
		if err != nil {
			return resp, fmt.Errorf("failed to determine resource type ID: %s", err)
		}

		// "Computed" attributes would have been replaced with Unknown values during
		// planning in order to allow the response from apply to return potentially
		// different values to the ones the user configured.
		//
		// Here we replace "computed" attributes (showing as Unknown) with their actual
		// user-supplied values from "manifest" (if present).
		obj, err = tftypes.Transform(obj, func(ap *tftypes.AttributePath, v tftypes.Value) (tftypes.Value, error) {
			_, isComputed := computedFields[ap.String()]
			if !isComputed {
				return v, nil
			}
			if v.IsKnown() {
				return v, nil
			}
			ppMan, restPath, err := tftypes.WalkAttributePath(plannedStateVal["manifest"], ap)
			if err != nil {
				if len(restPath.Steps()) > 0 {
					// attribute not in manifest
					return v, nil
				}
				return v, ap.NewError(err)
			}
			nv, err := morph.ValueToType(ppMan.(tftypes.Value), v.Type(), tftypes.NewAttributePath())
			if err != nil {
				return v, ap.NewError(err)
			}
			return nv, nil
		})
		if err != nil {
			resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Failed to backfill computed values in proposed value",
				Detail:   err.Error(),
			})
			return resp, nil
		}

		nullObj := morph.UnknownToNull(obj)
		s.logger.Trace("[ApplyResourceChange][Apply]", "[UnknownToNull]", dump(nullObj))

		// Remove empty objects unless explicitly set by the user in manifest.
		// They only serve a structural purpose in the planning phase and should not be included in the API payload.
		minObj, err := tftypes.Transform(nullObj, func(ap *tftypes.AttributePath, v tftypes.Value) (tftypes.Value, error) {
			if v.IsNull() {
				return tftypes.NewValue(v.Type(), nil), nil
			}
			switch {
			case v.Type().Is(tftypes.Object{}) || v.Type().Is(tftypes.Map{}):
				atts := make(map[string]tftypes.Value)
				err := v.As(&atts)
				if err != nil {
					return v, err
				}
				var isEmpty bool = true
				for _, atv := range atts {
					if !atv.IsNull() {
						isEmpty = false
						break
					}
				}
				// check if attribute path is present in user-supplied manifest
				// (this means the value is intentional, not structural)
				_, restPath, err := tftypes.WalkAttributePath(confVals["manifest"], ap)
				if (err == nil && len(restPath.Steps()) == 0) || !isEmpty {
					// attribute is not empty and/or was set by the user -> retain
					return tftypes.NewValue(v.Type(), atts), nil
				}
				return tftypes.NewValue(v.Type(), nil), nil
			case v.Type().Is(tftypes.List{}) || v.Type().Is(tftypes.Set{}) || v.Type().Is(tftypes.Tuple{}):
				atts := make([]tftypes.Value, 0)
				err := v.As(&atts)
				if err != nil {
					return v, err
				}
				return tftypes.NewValue(v.Type(), atts), nil
			default:
				return v, nil
			}
		})
		if err != nil {
			resp.Diagnostics = append(resp.Diagnostics,
				&tfprotov5.Diagnostic{
					Severity: tfprotov5.DiagnosticSeverityError,
					Detail:   err.Error(),
					Summary:  "Failed to sanitize empty block ahead of payload preparation",
				})
			return resp, nil
		}

		pu, err := payload.FromTFValue(minObj, th, tftypes.NewAttributePath())
		if err != nil {
			return resp, err
		}
		s.logger.Trace("[ApplyResourceChange][Apply]", "[payload.FromTFValue]", dump(pu))

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

		// Check the resource does not exist if this is a create operation
		if applyPriorState.IsNull() {
			_, err := rs.Get(ctx, rname, metav1.GetOptions{})
			if err == nil {
				resp.Diagnostics = append(resp.Diagnostics,
					&tfprotov5.Diagnostic{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "Cannot create resource that already exists",
						Detail:   fmt.Sprintf("resource %q already exists", rnn),
					})
				return resp, nil
			} else if !apierrors.IsNotFound(err) {
				resp.Diagnostics = append(resp.Diagnostics,
					&tfprotov5.Diagnostic{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  fmt.Sprintf("Failed to determine if resource %q exists", rnn),
						Detail:   err.Error(),
					})
				return resp, nil
			}
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

		// get fieldManager config
		fieldManagerName, forceConflicts, err := s.getFieldManagerConfig(plannedStateVal)
		if err != nil {
			resp.Diagnostics = append(resp.Diagnostics, &tfprotov5.Diagnostic{
				Severity: tfprotov5.DiagnosticSeverityError,
				Summary:  "Could not extract field_manager config",
				Detail:   err.Error(),
			})
			return resp, nil
		}

		// figure out the timeout deadline
		timeouts := s.getTimeouts(plannedStateVal)
		var timeout time.Duration
		if applyPriorState.IsNull() {
			timeout, _ = time.ParseDuration(timeouts["create"])
		} else {
			timeout, _ = time.ParseDuration(timeouts["update"])
		}
		deadline := time.Now().Add(timeout)
		ctxDeadline, cancel := context.WithDeadline(ctx, deadline)
		defer cancel()

		// Call the Kubernetes API to create the new resource
		s.logger.Trace("[ApplyResourceChange][API Payload]: %s", jsonManifest)
		result, err := rs.Patch(ctxDeadline, rname, types.ApplyPatchType, jsonManifest,
			metav1.PatchOptions{
				FieldManager: fieldManagerName,
				Force:        &forceConflicts,
			},
		)
		if err != nil {
			s.logger.Error("[ApplyResourceChange][Apply]", "API error", dump(err), "API response", dump(result))
			if apierrors.IsConflict(err) {
				resp.Diagnostics = append(resp.Diagnostics,
					&tfprotov5.Diagnostic{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  fmt.Sprintf(`There was a field manager conflict when trying to apply the manifest for %q`, rnn),
						Detail: fmt.Sprintf(
							"The API returned the following conflict: %q\n\n"+
								"You can override this conflict by setting \"force_conflicts\" to true in the \"field_manager\" block.",
							err.Error(),
						),
					},
				)
			} else if status := apierrors.APIStatus(nil); errors.As(err, &status) {
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

		newResObject, err := payload.ToTFValue(RemoveServerSideFields(result.Object), tsch, th, tftypes.NewAttributePath())
		if err != nil {
			resp.Diagnostics = append(resp.Diagnostics,
				&tfprotov5.Diagnostic{
					Severity: tfprotov5.DiagnosticSeverityError,
					Summary:  "Conversion from Unstructured to tftypes.Value failed",
					Detail:   err.Error(),
				})
			return resp, nil
		}
		s.logger.Trace("[ApplyResourceChange][Apply]", "[payload.ToTFValue]", dump(newResObject))

		wt, _, err := s.TFTypeFromOpenAPI(ctx, gvk, true)
		if err != nil {
			return resp, fmt.Errorf("failed to determine resource type ID: %s", err)
		}

		var waitConfig tftypes.Value
		if w, ok := plannedStateVal["wait"]; ok && !w.IsNull() {
			s.logger.Trace("[ApplyResourceChange][Wait] Using waiter config from `wait` block")
			var waitBlocks []tftypes.Value
			w.As(&waitBlocks)
			if len(waitBlocks) > 0 {
				waitConfig = waitBlocks[0]
			}
		}
		if wf, ok := plannedStateVal["wait_for"]; ok && !wf.IsNull() {
			s.logger.Trace("[ApplyResourceChange][Wait] Using waiter config from deprecated `wait_for` attribute")
			waitConfig = wf
		}
		if !waitConfig.IsNull() {
			err = s.waitForCompletion(ctxDeadline, waitConfig, rs, rname, wt, th)
			if err != nil {
				if err == context.DeadlineExceeded {
					resp.Diagnostics = append(resp.Diagnostics,
						&tfprotov5.Diagnostic{
							Severity: tfprotov5.DiagnosticSeverityError,
							Summary:  "Operation timed out",
							Detail:   "Terraform timed out waiting on the operation to complete",
						})
				} else {
					resp.Diagnostics = append(resp.Diagnostics,
						&tfprotov5.Diagnostic{
							Severity: tfprotov5.DiagnosticSeverityError,
							Summary:  "Error waiting for operation to complete",
							Detail:   err.Error(),
						})
				}
				return resp, nil
			}
		}

		compObj, err := morph.DeepUnknown(tsch, newResObject, tftypes.NewAttributePath())
		if err != nil {
			return resp, err
		}
		plannedStateVal["object"] = morph.UnknownToNull(compObj)

		newStateVal := tftypes.NewValue(applyPlannedState.Type(), plannedStateVal)
		s.logger.Trace("[ApplyResourceChange][Apply]", "new state value", dump(newStateVal))

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

		pu, err := payload.FromTFValue(pco, nil, tftypes.NewAttributePath())
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

		// figure out the timeout deadline
		timeouts := s.getTimeouts(priorStateVal)
		timeout, _ := time.ParseDuration(timeouts["delete"])
		deadline := time.Now().Add(timeout)
		ctxDeadline, cancel := context.WithDeadline(ctx, deadline)
		defer cancel()

		err = rs.Delete(ctxDeadline, rname, metav1.DeleteOptions{})
		if err != nil {
			rn := types.NamespacedName{Namespace: rnamespace, Name: rname}.String()
			resp.Diagnostics = append(resp.Diagnostics,
				&tfprotov5.Diagnostic{
					Severity: tfprotov5.DiagnosticSeverityError,
					Summary:  fmt.Sprintf("Error deleting resource %s: %s", rn, err),
					Detail:   err.Error(),
				})
			return resp, nil
		}

		// wait for delete
		for {
			if time.Now().After(deadline) {
				resp.Diagnostics = append(resp.Diagnostics,
					&tfprotov5.Diagnostic{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  fmt.Sprintf("Timed out when waiting for resource %q to be deleted", rname),
						Detail:   "Deletion timed out. This can happen when there is a finalizer on a resource. You may need to delete this resource manually with kubectl.",
					})
				return resp, nil
			}
			_, err := rs.Get(ctxDeadline, rname, metav1.GetOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) {
					s.logger.Trace("[ApplyResourceChange][Delete]", "Resource is deleted")
					break
				}
				resp.Diagnostics = append(resp.Diagnostics,
					&tfprotov5.Diagnostic{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "Error waiting for deletion.",
						Detail:   fmt.Sprintf("Error when waiting for resource %q to be deleted: %v", rname, err),
					})
				return resp, nil
			}
			time.Sleep(1 * time.Second) // lintignore:R018
		}

		resp.NewState = req.PlannedState
	}

	return resp, nil
}

func (s *RawProviderServer) getTimeouts(v map[string]tftypes.Value) map[string]string {
	timeouts := map[string]string{
		"create": defaultCreateTimeout,
		"update": defaultUpdateTimeout,
		"delete": defaultDeleteTimeout,
	}
	if !v["timeouts"].IsNull() && v["timeouts"].IsKnown() {
		var timeoutsBlock []tftypes.Value
		v["timeouts"].As(&timeoutsBlock)
		if len(timeoutsBlock) > 0 {
			var t map[string]tftypes.Value
			timeoutsBlock[0].As(&t)
			var s string
			for _, k := range []string{"create", "update", "delete"} {
				if vv, ok := t[k]; ok && !vv.IsNull() {
					vv.As(&s)
					if s != "" {
						timeouts[k] = s
					}
				}
			}
		}
	}
	return timeouts
}
