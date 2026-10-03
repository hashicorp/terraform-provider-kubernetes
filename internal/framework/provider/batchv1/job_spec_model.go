// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"slices"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	batchapi "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// The Job controller labels the pods of every Job it creates.
var jobGeneratedLabels = []string{"batch.kubernetes.io/controller-uid", "batch.kubernetes.io/job-name", "controller-uid", "job-name"}

// expandJobSpec converts the single "spec" element into a JobSpec, with the
// same field semantics as SDKv2. Unknown values are left for the API to set.
func expandJobSpec(ctx context.Context, value types.List, job bool, at path.Path) (batchapi.JobSpec, diag.Diagnostics) {
	var diags diag.Diagnostics
	spec, ok := singleObject(value)
	if !ok {
		diags.AddAttributeError(at, "Invalid Job specification", "Exactly one known spec block is required.")
		return batchapi.JobSpec{}, diags
	}
	a := spec.Attributes()
	var out batchapi.JobSpec
	if n, ok := knownInt64(a["active_deadline_seconds"]); ok && n > 0 {
		out.ActiveDeadlineSeconds = ptr.To(n)
	}
	if n, ok := knownInt64(a["backoff_limit"]); ok && n >= 0 {
		out.BackoffLimit = ptr.To(int32(n))
	}
	if n, ok := knownInt64(a["completions"]); ok && n > 0 {
		out.Completions = ptr.To(int32(n))
	}
	if s, ok := knownString(a["completion_mode"]); ok && s != "" {
		out.CompletionMode = ptr.To(batchapi.CompletionMode(s))
	}
	if out.CompletionMode != nil && *out.CompletionMode == batchapi.IndexedCompletion {
		if n, ok := knownInt64(a["backoff_limit_per_index"]); ok && n >= 0 {
			out.BackoffLimitPerIndex = ptr.To(int32(n))
		}
		if n, ok := knownInt64(a["max_failed_indexes"]); ok && n >= 0 {
			out.MaxFailedIndexes = ptr.To(int32(n))
		}
	}
	if b, ok := a["manual_selector"].(types.Bool); ok && !b.IsNull() && !b.IsUnknown() {
		out.ManualSelector = ptr.To(b.ValueBool())
	}
	if n, ok := knownInt64(a["parallelism"]); ok && n >= 0 {
		out.Parallelism = ptr.To(int32(n))
	}
	out.PodFailurePolicy = expandPodFailurePolicy(a["pod_failure_policy"])
	out.Selector = expandLabelSelector(a["selector"])
	if s, ok := knownString(a["ttl_seconds_after_finished"]); ok && s != "" {
		ttl, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			diags.AddAttributeError(at.AtListIndex(0).AtName("ttl_seconds_after_finished"), "Invalid job TTL", err.Error())
			return out, diags
		}
		out.TTLSecondsAfterFinished = ptr.To(int32(ttl))
	}
	template, d := expandPodTemplate(ctx, a["template"], job, at.AtListIndex(0).AtName("template"))
	diags.Append(d...)
	out.Template = template
	return out, diags
}

func expandPodTemplate(ctx context.Context, value attr.Value, job bool, at path.Path) (corev1.PodTemplateSpec, diag.Diagnostics) {
	var diags diag.Diagnostics
	template, ok := singleObject(value)
	if !ok {
		diags.AddAttributeError(at, "Invalid pod template", "Exactly one known template block is required.")
		return corev1.PodTemplateSpec{}, diags
	}
	spec, _ := template.Attributes()["spec"].(types.List)
	podSpec, d := jobPodSpec(job).ExpandSpec(ctx, spec, at.AtListIndex(0).AtName("spec"))
	diags.Append(d...)
	return corev1.PodTemplateSpec{ObjectMeta: expandTemplateMetadata(template.Attributes()["metadata"]), Spec: podSpec}, diags
}

func expandTemplateMetadata(value attr.Value) metav1.ObjectMeta {
	metadata, ok := singleObject(value)
	if !ok {
		return metav1.ObjectMeta{}
	}
	a := metadata.Attributes()
	var out metav1.ObjectMeta
	out.Annotations = knownStringMap(a["annotations"])
	out.Labels = knownStringMap(a["labels"])
	out.Name, _ = knownString(a["name"])
	out.GenerateName, _ = knownString(a["generate_name"])
	out.Namespace, _ = knownString(a["namespace"])
	return out
}

func expandLabelSelector(value attr.Value) *metav1.LabelSelector {
	selector, ok := singleObject(value)
	if !ok {
		return nil
	}
	a := selector.Attributes()
	out := &metav1.LabelSelector{MatchLabels: knownStringMap(a["match_labels"])}
	for _, element := range knownElements(a["match_expressions"]) {
		expression, ok := element.(types.Object)
		if !ok || expression.IsNull() || expression.IsUnknown() {
			continue
		}
		e := expression.Attributes()
		requirement := metav1.LabelSelectorRequirement{}
		key, _ := knownString(e["key"])
		operator, _ := knownString(e["operator"])
		requirement.Key, requirement.Operator = key, metav1.LabelSelectorOperator(operator)
		for _, v := range knownElements(e["values"]) {
			if s, ok := knownString(v); ok {
				requirement.Values = append(requirement.Values, s)
			}
		}
		out.MatchExpressions = append(out.MatchExpressions, requirement)
	}
	return out
}

func expandPodFailurePolicy(value attr.Value) *batchapi.PodFailurePolicy {
	policy, ok := singleObject(value)
	if !ok {
		return nil
	}
	out := &batchapi.PodFailurePolicy{}
	for _, element := range knownElements(policy.Attributes()["rule"]) {
		rule, ok := element.(types.Object)
		if !ok || rule.IsNull() || rule.IsUnknown() {
			continue
		}
		r := rule.Attributes()
		var result batchapi.PodFailurePolicyRule
		action, _ := knownString(r["action"])
		result.Action = batchapi.PodFailurePolicyAction(action)
		if codes, ok := singleObject(r["on_exit_codes"]); ok {
			c := codes.Attributes()
			requirement := &batchapi.PodFailurePolicyOnExitCodesRequirement{}
			if name, _ := knownString(c["container_name"]); name != "" {
				requirement.ContainerName = ptr.To(name)
			}
			operator, _ := knownString(c["operator"])
			requirement.Operator = batchapi.PodFailurePolicyOnExitCodesOperator(operator)
			for _, v := range knownElements(c["values"]) {
				if n, ok := knownInt64(v); ok {
					requirement.Values = append(requirement.Values, int32(n))
				}
			}
			result.OnExitCodes = requirement
		}
		for _, element := range knownElements(r["on_pod_condition"]) {
			condition, ok := element.(types.Object)
			if !ok || condition.IsNull() || condition.IsUnknown() {
				continue
			}
			status, _ := knownString(condition.Attributes()["status"])
			typ, _ := knownString(condition.Attributes()["type"])
			result.OnPodConditions = append(result.OnPodConditions, batchapi.PodFailurePolicyOnPodConditionsPattern{
				Status: corev1.ConditionStatus(status), Type: corev1.PodConditionType(typ),
			})
		}
		out.Rules = append(out.Rules, result)
	}
	return out
}

// flattenJobSpec converts a JobSpec into the "spec" list. The prior value, the
// plan on writes and the state on reads (refresh), decides null versus empty values.
func flattenJobSpec(ctx context.Context, in batchapi.JobSpec, prior types.List, job, refresh bool, at path.Path) (types.List, diag.Diagnostics) {
	typ := jobSpecTypeFor(job)
	previous := priorAttributes(prior)
	selectorLabels := jobGeneratedLabels
	if !job || (in.ManualSelector != nil && *in.ManualSelector) {
		selectorLabels = nil
	}
	ttl := ""
	if in.TTLSecondsAfterFinished != nil {
		ttl = strconv.Itoa(int(*in.TTLSecondsAfterFinished))
	}
	completionMode := ""
	if in.CompletionMode != nil {
		completionMode = string(*in.CompletionMode)
	}
	templateLabels := []string(nil)
	if job {
		templateLabels = jobGeneratedLabels
	}
	template, diags := flattenPodTemplate(ctx, in.Template, previous["template"], typ.AttrTypes["template"].(types.ListType), job, refresh, templateLabels, at.AtListIndex(0).AtName("template"))
	attributes := map[string]attr.Value{
		"active_deadline_seconds":    types.Int64Value(ptr.Deref(in.ActiveDeadlineSeconds, 0)),
		"backoff_limit":              types.Int64Value(int64(ptr.Deref(in.BackoffLimit, 0))),
		"backoff_limit_per_index":    types.Int64Value(int64(ptr.Deref(in.BackoffLimitPerIndex, 0))),
		"completion_mode":            types.StringValue(completionMode),
		"completions":                types.Int64Value(int64(ptr.Deref(in.Completions, 0))),
		"manual_selector":            types.BoolValue(ptr.Deref(in.ManualSelector, false)),
		"max_failed_indexes":         types.Int64Value(int64(ptr.Deref(in.MaxFailedIndexes, 0))),
		"parallelism":                types.Int64Value(int64(ptr.Deref(in.Parallelism, 0))),
		"pod_failure_policy":         flattenPodFailurePolicy(in.PodFailurePolicy, typ.AttrTypes["pod_failure_policy"].(types.ListType)),
		"selector":                   flattenLabelSelector(in.Selector, previous["selector"], typ.AttrTypes["selector"].(types.ListType), selectorLabels),
		"template":                   template,
		"ttl_seconds_after_finished": types.StringValue(ttl),
	}
	if diags.HasError() {
		return types.ListNull(typ), diags
	}
	return singletonList(typ, attributes, &diags), diags
}

func flattenPodTemplate(ctx context.Context, in corev1.PodTemplateSpec, prior attr.Value, typ types.ListType, job, refresh bool, generatedLabels []string, at path.Path) (types.List, diag.Diagnostics) {
	objectType := typ.ElemType.(types.ObjectType)
	previous := priorAttributes(prior)
	priorSpec, ok := previous["spec"].(types.List)
	if !ok {
		priorSpec = types.ListNull(jobPodSpec(job).ObjectType())
	}
	spec, diags := flattenPodSpec(ctx, in.Spec, priorSpec, job, refresh, at.AtListIndex(0).AtName("spec"))
	metadata := flattenTemplateMetadata(in.ObjectMeta, previous["metadata"], objectType.AttrTypes["metadata"].(types.ListType), refresh && !job, generatedLabels, &diags)
	if diags.HasError() {
		return types.ListNull(objectType), diags
	}
	return singletonList(objectType, map[string]attr.Value{"metadata": metadata, "spec": spec}, &diags), diags
}

// flattenPodSpec records the pod spec a write returned against the plan, or
// on a read the live values against the prior state.
func flattenPodSpec(ctx context.Context, in corev1.PodSpec, prior types.List, job, refresh bool, at path.Path) (types.List, diag.Diagnostics) {
	if refresh {
		return jobPodSpec(job).RefreshSpec(ctx, in, prior, at)
	}
	return jobPodSpec(job).FlattenSpec(ctx, in, prior, at)
}

// Template labels and annotations record the planned maps after a write. A
// Job's template cannot change, so a refresh keeps them too, and keys or
// values admission set at create never drift. Otherwise, as on an import, a
// refresh records every live key, as SDKv2 did.
func flattenTemplateMetadata(in metav1.ObjectMeta, prior attr.Value, typ types.ListType, live bool, generatedLabels []string, diags *diag.Diagnostics) types.List {
	objectType := typ.ElemType.(types.ObjectType)
	previous := priorAttributes(prior)
	live = live || previous == nil
	attributes := map[string]attr.Value{
		"annotations":      templateMap(in.Annotations, previous["annotations"], live, nil),
		"labels":           templateMap(in.Labels, previous["labels"], live, generatedLabels),
		"generate_name":    types.StringNull(),
		"generation":       types.Int64Value(in.Generation),
		"name":             types.StringValue(in.Name),
		"resource_version": types.StringValue(in.ResourceVersion),
		"uid":              types.StringValue(string(in.UID)),
	}
	if in.GenerateName != "" {
		attributes["generate_name"] = types.StringValue(in.GenerateName)
	}
	if _, ok := objectType.AttrTypes["namespace"]; ok {
		attributes["namespace"] = types.StringValue(in.Namespace)
	}
	return singletonList(objectType, attributes, diags)
}

func templateMap(in map[string]string, prior attr.Value, live bool, generated []string) types.Map {
	previous, ok := prior.(types.Map)
	if ok && !live && !previous.IsUnknown() {
		return previous
	}
	values := make(map[string]attr.Value, len(in))
	for key, value := range in {
		if !slices.Contains(generated, key) {
			values[key] = types.StringValue(value)
		}
	}
	if len(values) == 0 && (!ok || previous.IsNull() || previous.IsUnknown()) {
		return types.MapNull(types.StringType)
	}
	return types.MapValueMust(types.StringType, values)
}

func flattenLabelSelector(in *metav1.LabelSelector, prior attr.Value, typ types.ListType, generatedLabels []string) types.List {
	objectType := typ.ElemType.(types.ObjectType)
	if in == nil {
		return types.ListValueMust(objectType, []attr.Value{})
	}
	var previous map[string]attr.Value
	if element, ok := singleObject(prior); ok {
		previous = element.Attributes()
	}
	labels := map[string]attr.Value{}
	for key, value := range in.MatchLabels {
		if !slices.Contains(generatedLabels, key) {
			labels[key] = types.StringValue(value)
		}
	}
	matchLabels := types.MapNull(types.StringType)
	if len(labels) > 0 || isEmptyCollection(previous["match_labels"]) {
		matchLabels = types.MapValueMust(types.StringType, labels)
	}
	expressionsType := objectType.AttrTypes["match_expressions"].(types.ListType)
	expressionType := expressionsType.ElemType.(types.ObjectType)
	expressions := types.ListNull(expressionType)
	if len(in.MatchExpressions) > 0 || isEmptyCollection(previous["match_expressions"]) {
		var previousExpressions []attr.Value
		if list, ok := previous["match_expressions"].(types.List); ok {
			previousExpressions = list.Elements()
		}
		elements := make([]attr.Value, 0, len(in.MatchExpressions))
		for i, expression := range in.MatchExpressions {
			values := types.SetNull(types.StringType)
			// Kubernetes drops empty values; keep a configured empty set.
			if i < len(previousExpressions) && isEmptyCollection(objectAttribute(previousExpressions[i], "values")) {
				values = types.SetValueMust(types.StringType, []attr.Value{})
			}
			if len(expression.Values) > 0 {
				items := make([]attr.Value, len(expression.Values))
				for i, v := range expression.Values {
					items[i] = types.StringValue(v)
				}
				values = types.SetValueMust(types.StringType, items)
			}
			elements = append(elements, types.ObjectValueMust(expressionType.AttrTypes, map[string]attr.Value{
				"key": types.StringValue(expression.Key), "operator": types.StringValue(string(expression.Operator)), "values": values,
			}))
		}
		expressions = types.ListValueMust(expressionType, elements)
	}
	return types.ListValueMust(objectType, []attr.Value{types.ObjectValueMust(objectType.AttrTypes, map[string]attr.Value{
		"match_labels": matchLabels, "match_expressions": expressions,
	})})
}

func flattenPodFailurePolicy(in *batchapi.PodFailurePolicy, typ types.ListType) types.List {
	policyType := typ.ElemType.(types.ObjectType)
	if in == nil {
		return types.ListValueMust(policyType, []attr.Value{})
	}
	rulesType := policyType.AttrTypes["rule"].(types.ListType)
	ruleType := rulesType.ElemType.(types.ObjectType)
	codesType := ruleType.AttrTypes["on_exit_codes"].(types.ListType)
	conditionsType := ruleType.AttrTypes["on_pod_condition"].(types.ListType)
	rules := make([]attr.Value, 0, len(in.Rules))
	for _, rule := range in.Rules {
		codes := []attr.Value{}
		if rule.OnExitCodes != nil {
			values := make([]attr.Value, len(rule.OnExitCodes.Values))
			for i, v := range rule.OnExitCodes.Values {
				values[i] = types.Int64Value(int64(v))
			}
			codes = append(codes, types.ObjectValueMust(codesType.ElemType.(types.ObjectType).AttrTypes, map[string]attr.Value{
				"container_name": types.StringValue(ptr.Deref(rule.OnExitCodes.ContainerName, "")),
				"operator":       types.StringValue(string(rule.OnExitCodes.Operator)),
				"values":         types.ListValueMust(types.Int64Type, values),
			}))
		}
		conditions := make([]attr.Value, 0, len(rule.OnPodConditions))
		for _, condition := range rule.OnPodConditions {
			conditions = append(conditions, types.ObjectValueMust(conditionsType.ElemType.(types.ObjectType).AttrTypes, map[string]attr.Value{
				"status": types.StringValue(string(condition.Status)), "type": types.StringValue(string(condition.Type)),
			}))
		}
		rules = append(rules, types.ObjectValueMust(ruleType.AttrTypes, map[string]attr.Value{
			"action":           types.StringValue(string(rule.Action)),
			"on_exit_codes":    types.ListValueMust(codesType.ElemType, codes),
			"on_pod_condition": types.ListValueMust(conditionsType.ElemType, conditions),
		}))
	}
	return types.ListValueMust(policyType, []attr.Value{types.ObjectValueMust(policyType.AttrTypes, map[string]attr.Value{
		"rule": types.ListValueMust(ruleType, rules),
	})})
}

func singletonList(typ types.ObjectType, attributes map[string]attr.Value, diags *diag.Diagnostics) types.List {
	object, d := types.ObjectValue(typ.AttrTypes, attributes)
	diags.Append(d...)
	if d.HasError() {
		return types.ListNull(typ)
	}
	list, d := types.ListValue(typ, []attr.Value{object})
	diags.Append(d...)
	return list
}

func singleObject(value attr.Value) (types.Object, bool) {
	list, ok := value.(types.List)
	if !ok || list.IsNull() || list.IsUnknown() || len(list.Elements()) != 1 {
		return types.Object{}, false
	}
	object, ok := list.Elements()[0].(types.Object)
	return object, ok && !object.IsNull() && !object.IsUnknown()
}

func priorAttributes(value attr.Value) map[string]attr.Value {
	if object, ok := singleObject(value); ok {
		return object.Attributes()
	}
	return nil
}

func knownElements(value attr.Value) []attr.Value {
	switch v := value.(type) {
	case types.List:
		if !v.IsNull() && !v.IsUnknown() {
			return v.Elements()
		}
	case types.Set:
		if !v.IsNull() && !v.IsUnknown() {
			return v.Elements()
		}
	}
	return nil
}

func knownInt64(value attr.Value) (int64, bool) {
	v, ok := value.(types.Int64)
	return v.ValueInt64(), ok && !v.IsNull() && !v.IsUnknown()
}

func knownString(value attr.Value) (string, bool) {
	v, ok := value.(types.String)
	return v.ValueString(), ok && !v.IsNull() && !v.IsUnknown()
}

func knownStringMap(value attr.Value) map[string]string {
	m, ok := value.(types.Map)
	if !ok || m.IsNull() || m.IsUnknown() {
		return nil
	}
	out := make(map[string]string, len(m.Elements()))
	for key, element := range m.Elements() {
		if s, ok := knownString(element); ok {
			out[key] = s
		}
	}
	return out
}

func isEmptyCollection(value attr.Value) bool {
	switch v := value.(type) {
	case types.List:
		return !v.IsNull() && !v.IsUnknown() && len(v.Elements()) == 0
	case types.Map:
		return !v.IsNull() && !v.IsUnknown() && len(v.Elements()) == 0
	case types.Set:
		return !v.IsNull() && !v.IsUnknown() && len(v.Elements()) == 0
	}
	return false
}

func objectAttribute(value attr.Value, name string) attr.Value {
	if object, ok := value.(types.Object); ok && !object.IsNull() && !object.IsUnknown() {
		return object.Attributes()[name]
	}
	return nil
}
