// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	_ resource.ResourceWithModifyPlan = (*JobV1)(nil)
	_ resource.ResourceWithModifyPlan = (*CronJobV1)(nil)
)

var jobTemplatePath = path.Root("spec").AtListIndex(0).AtName("template")

func (r *JobV1) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}
	if noOpPlan(req, resp) {
		return
	}
	apiDefaulted := apiDefaultedStrings(ctx, req.Plan.Schema)
	replace, diags := jobTemplateChanged(ctx, req.Config.Raw, resp.Plan.Raw, req.State.Raw, apiDefaulted)
	if replace {
		// State written without a refresh can disagree with Kubernetes; decide
		// against the Job as it is now.
		if live, ok := r.liveState(ctx, req.State); ok {
			replace, diags = jobTemplateChanged(ctx, req.Config.Raw, resp.Plan.Raw, live, apiDefaulted)
		}
	}
	resp.Diagnostics.Append(diags...)
	if replace {
		resp.RequiresReplace = append(resp.RequiresReplace, jobTemplatePath)
	}
}

func (r *JobV1) liveState(ctx context.Context, state tfsdk.State) (tftypes.Value, bool) {
	var model JobV1Model
	if state.Get(ctx, &model).HasError() || r.SDKv2Meta == nil {
		return tftypes.Value{}, false
	}
	namespace, name, err := kubernetes.IdParts(model.ID.ValueString())
	if err != nil {
		return tftypes.Value{}, false
	}
	clients, filters, diags := r.sdkv2Meta()
	if diags.HasError() {
		return tftypes.Value{}, false
	}
	conn, err := clients.MainClientset()
	if err != nil {
		return tftypes.Value{}, false
	}
	job, err := conn.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil || flattenJob(ctx, job, &model, filters).HasError() {
		return tftypes.Value{}, false
	}
	live := tfsdk.State{Schema: state.Schema}
	if live.Set(ctx, &model).HasError() {
		return tftypes.Value{}, false
	}
	return live.Raw, true
}

func (r *CronJobV1) ModifyPlan(_ context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}
	noOpPlan(req, resp)
}

func noOpPlan(req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) bool {
	plan, unchanged, err := common.NoOpPlan(req.Config.Raw, req.Plan.Raw, req.State.Raw)
	if err != nil {
		resp.Diagnostics.AddError("Unable to normalize plan", err.Error())
		return false
	}
	if unchanged {
		resp.Plan.Raw = plan
	}
	return unchanged
}

// Kubernetes does not allow a Job's pod template to change. The fields SDKv2
// declared ForceNew replace the Job on their own; any other configured change
// to the template the API would receive replaces it here, rather than planning
// an update that cannot take effect.
func jobTemplateChanged(ctx context.Context, configRaw, planRaw, stateRaw tftypes.Value, apiDefaulted func(*tftypes.AttributePath) bool) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	at := tftypes.NewAttributePath().WithAttributeName("spec").WithElementKeyInt(0).WithAttributeName("template")
	config, configOK := valueAt(configRaw, at)
	plan, planOK := valueAt(planRaw, at)
	state, stateOK := valueAt(stateRaw, at)
	if !configOK || !planOK || !stateOK || plan.Equal(state) {
		return false, diags
	}
	resolved, ok := resolveUnconfigured(at, config, plan, state, apiDefaulted)
	if !ok {
		return true, diags
	}
	typ := jobSpecType().AttrTypes["template"]
	before, err := typ.ValueFromTerraform(ctx, state)
	if err != nil {
		diags.AddError("Unable to compare Job pod template", err.Error())
		return false, diags
	}
	after, err := typ.ValueFromTerraform(ctx, resolved)
	if err != nil {
		diags.AddError("Unable to compare Job pod template", err.Error())
		return false, diags
	}
	previous, d := expandPodTemplate(ctx, before, true, jobTemplatePath)
	if d.HasError() {
		return false, diags
	}
	desired, d := expandPodTemplate(ctx, after, true, jobTemplatePath)
	if d.HasError() {
		return true, diags
	}
	return !podTemplatesEqual(previous, desired), diags
}

func podTemplatesEqual(a, b corev1.PodTemplateSpec) bool {
	for _, template := range []*corev1.PodTemplateSpec{&a, &b} {
		*template = *template.DeepCopy()
		for _, key := range jobGeneratedLabels {
			delete(template.Labels, key)
		}
		clearUnsetFalse(&template.Spec)
	}
	return payloadsEqual(a, b)
}

func podSpecsEqual(a, b corev1.PodSpec) bool {
	a, b = *a.DeepCopy(), *b.DeepCopy()
	clearUnsetFalse(&a)
	clearUnsetFalse(&b)
	return payloadsEqual(a, b)
}

// clearUnsetFalse clears the pointer booleans whose false is what Kubernetes
// does when they are unset. Every other false is a request of its own:
// allowPrivilegeEscalation, automountServiceAccountToken and
// enableServiceLinks default to true, and a container's runAsNonRoot overrides
// the pod's.
func clearUnsetFalse(spec *corev1.PodSpec) {
	unset := func(b **bool) {
		if *b != nil && !**b {
			*b = nil
		}
	}
	unset(&spec.ShareProcessNamespace)
	if spec.SecurityContext != nil {
		unset(&spec.SecurityContext.RunAsNonRoot)
	}
	for _, containers := range [][]corev1.Container{spec.InitContainers, spec.Containers} {
		for i := range containers {
			if sc := containers[i].SecurityContext; sc != nil {
				unset(&sc.Privileged)
				unset(&sc.ReadOnlyRootFilesystem)
			}
		}
	}
}

// payloadsEqual compares objects as the API receives them, where absent, null
// and empty values are the same and quantities have one spelling. False is a
// value: callers clear the booleans for which it is not (see clearUnsetFalse).
func payloadsEqual(a, b any) bool {
	x, errA := prunedJSON(a)
	y, errB := prunedJSON(b)
	return errA == nil && errB == nil && reflect.DeepEqual(x, y)
}

func prunedJSON(v any) (any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return prune(out), nil
}

func prune(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for key, child := range t {
			if pruned := prune(child); pruned == nil {
				delete(t, key)
			} else {
				t[key] = pruned
			}
		}
		if len(t) == 0 {
			return nil
		}
	case []any:
		if len(t) == 0 {
			return nil
		}
		for i := range t {
			t[i] = prune(t[i])
		}
	}
	return v
}

// apiDefaultedStrings reports the strings the API fills when they are empty:
// those Optional and Computed without a default, as SDKv2's Optional+Computed
// strings were. Configuring "" for one of them keeps the value Kubernetes has.
func apiDefaultedStrings(ctx context.Context, s any) func(*tftypes.AttributePath) bool {
	resourceSchema, _ := s.(schema.Schema)
	return func(at *tftypes.AttributePath) bool {
		attribute, err := resourceSchema.AttributeAtTerraformPath(ctx, at)
		str, ok := attribute.(schema.StringAttribute)
		return err == nil && ok && str.Computed && str.Default == nil
	}
}

// resolveUnconfigured replaces unknown values, and API-defaulted strings
// configured as "", with their prior values, so that comparing API payloads
// ignores server-populated values. It reports false when a configured value is
// unknown.
func resolveUnconfigured(at *tftypes.AttributePath, config, plan, state tftypes.Value, apiDefaulted func(*tftypes.AttributePath) bool) (tftypes.Value, bool) {
	if !config.IsKnown() {
		return plan, false
	}
	prior := state
	if !state.IsKnown() || state.IsNull() {
		prior = plan
	}
	switch {
	case config.IsNull() && !plan.IsKnown():
		return prior, prior.IsKnown()
	case config.IsNull() && isScalar(plan.Type()) && state.IsNull() && isZero(plan):
		// SDKv2 state may lack a field that later releases default to zero.
		return state, true
	case config.IsNull():
		// A value removed from the configuration is removed, not inherited.
		return plan, true
	case config.Equal(tftypes.NewValue(tftypes.String, "")) && apiDefaulted(at):
		return prior, prior.IsKnown()
	case !plan.IsKnown():
		return plan, false
	case plan.IsNull():
		return plan, true
	}
	switch typ := plan.Type().(type) {
	case tftypes.Object:
		var configured, planned, previous map[string]tftypes.Value
		if config.As(&configured) != nil || plan.As(&planned) != nil {
			return plan, false
		}
		if state.IsKnown() && !state.IsNull() && state.As(&previous) != nil {
			return plan, false
		}
		out := make(map[string]tftypes.Value, len(planned))
		for name, value := range planned {
			before, ok := previous[name]
			if !ok {
				before = tftypes.NewValue(value.Type(), nil)
			}
			resolved, ok := resolveUnconfigured(at.WithAttributeName(name), configured[name], value, before, apiDefaulted)
			if !ok {
				return plan, false
			}
			out[name] = resolved
		}
		return tftypes.NewValue(typ, out), true
	case tftypes.List:
		var configured, planned, previous []tftypes.Value
		if config.As(&configured) != nil || plan.As(&planned) != nil {
			return plan, false
		}
		if state.IsKnown() && !state.IsNull() && state.As(&previous) != nil {
			return plan, false
		}
		out := make([]tftypes.Value, len(planned))
		for i, value := range planned {
			before, conf := tftypes.NewValue(value.Type(), nil), tftypes.NewValue(value.Type(), nil)
			if i < len(previous) {
				before = previous[i]
			}
			if i < len(configured) {
				conf = configured[i]
			}
			resolved, ok := resolveUnconfigured(at.WithElementKeyInt(i), conf, value, before, apiDefaulted)
			if !ok {
				return plan, false
			}
			out[i] = resolved
		}
		return tftypes.NewValue(typ, out), true
	}
	return plan, plan.IsFullyKnown()
}

func isZero(value tftypes.Value) bool {
	return value.Equal(tftypes.NewValue(tftypes.String, "")) || value.Equal(tftypes.NewValue(tftypes.Bool, false)) ||
		value.Equal(tftypes.NewValue(tftypes.Number, 0))
}

func isScalar(typ tftypes.Type) bool {
	switch typ.(type) {
	case tftypes.Object, tftypes.List, tftypes.Set, tftypes.Map, tftypes.Tuple:
		return false
	}
	return true
}

// knownOrActual is the state after a write: the plan, with each unknown value
// taken from the object Kubernetes returned.
func knownOrActual(plan, actual tftypes.Value) (tftypes.Value, error) {
	return tftypes.Transform(plan, func(at *tftypes.AttributePath, value tftypes.Value) (tftypes.Value, error) {
		if value.IsKnown() {
			return value, nil
		}
		if found, ok := valueAt(actual, at); ok && found.IsFullyKnown() {
			return found, nil
		}
		return tftypes.NewValue(value.Type(), nil), nil
	})
}

func valueAt(root tftypes.Value, at *tftypes.AttributePath) (tftypes.Value, bool) {
	found, _, err := tftypes.WalkAttributePath(root, at)
	if err != nil {
		return tftypes.Value{}, false
	}
	value, ok := found.(tftypes.Value)
	return value, ok
}

// SDKv2 stored omitted integers as zero; moving between zero and null is not a
// change of the object.
type zeroEquivalentInt64RequiresReplace struct{}

func (zeroEquivalentInt64RequiresReplace) Description(context.Context) string {
	return "Changes require replacement; null and zero are equivalent."
}

func (m zeroEquivalentInt64RequiresReplace) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (zeroEquivalentInt64RequiresReplace) PlanModifyInt64(_ context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	zero := types.Int64Value(0)
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() || req.PlanValue.Equal(req.StateValue) ||
		((req.PlanValue.IsNull() || req.PlanValue.Equal(zero)) && (req.StateValue.IsNull() || req.StateValue.Equal(zero))) {
		return
	}
	resp.RequiresReplace = true
}

type zeroEquivalentStringRequiresReplace struct{}

func (zeroEquivalentStringRequiresReplace) Description(context.Context) string {
	return "Changes require replacement; null and an empty string are equivalent."
}

func (m zeroEquivalentStringRequiresReplace) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (zeroEquivalentStringRequiresReplace) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() || req.PlanValue.Equal(req.StateValue) ||
		(req.PlanValue.ValueString() == "" && !req.PlanValue.IsUnknown() && req.StateValue.ValueString() == "") {
		return
	}
	resp.RequiresReplace = true
}

// An empty value defers to the API default, so it never replaces.
type apiDefaultedStringRequiresReplace struct{}

func (apiDefaultedStringRequiresReplace) Description(context.Context) string {
	return "Changes require replacement; an empty value keeps the Kubernetes default."
}

func (m apiDefaultedStringRequiresReplace) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (apiDefaultedStringRequiresReplace) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() || req.PlanValue.Equal(req.StateValue) ||
		req.PlanValue.Equal(types.StringValue("")) {
		return
	}
	resp.RequiresReplace = true
}

// SDKv2's ForceNew on a block governs only how many elements it has.
type listSizeRequiresReplace struct{}

func (listSizeRequiresReplace) Description(context.Context) string {
	return "Adding or removing the block requires replacement."
}

func (m listSizeRequiresReplace) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (listSizeRequiresReplace) PlanModifyList(_ context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() || req.PlanValue.IsUnknown() || req.StateValue.IsUnknown() {
		return
	}
	resp.RequiresReplace = len(req.PlanValue.Elements()) != len(req.StateValue.Elements())
}

// The selector replaces the object when the selector Kubernetes would receive
// changes, not when only its representation does.
type jobSelectorRequiresReplace struct{}

func (jobSelectorRequiresReplace) Description(context.Context) string {
	return "Changes to the selector require replacement."
}

func (m jobSelectorRequiresReplace) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (jobSelectorRequiresReplace) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() || req.PlanValue.Equal(req.StateValue) {
		return
	}
	resp.RequiresReplace = !fullyKnown(ctx, req.PlanValue) ||
		!payloadsEqual(expandLabelSelector(req.StateValue), expandLabelSelector(req.PlanValue))
}

// Kubernetes rejects any change to a Job's pod failure policy.
type jobPolicyRequiresReplace struct{}

func (jobPolicyRequiresReplace) Description(context.Context) string {
	return "Changes to the pod failure policy require replacement."
}

func (m jobPolicyRequiresReplace) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (jobPolicyRequiresReplace) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() || req.PlanValue.Equal(req.StateValue) {
		return
	}
	resp.RequiresReplace = !fullyKnown(ctx, req.PlanValue) ||
		!payloadsEqual(expandPodFailurePolicy(req.StateValue), expandPodFailurePolicy(req.PlanValue))
}

func fullyKnown(ctx context.Context, value types.List) bool {
	raw, err := value.ToTerraformValue(ctx)
	return err == nil && raw.IsFullyKnown()
}
