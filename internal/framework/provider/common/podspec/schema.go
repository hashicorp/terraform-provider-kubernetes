// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

// Package podspec provides the Framework schema and API conversions for the
// PodSpec of Deployment, DaemonSet, StatefulSet, Pod, Job and CronJob.
package podspec

import (
	"context"
	"fmt"
	"net"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	kquantity "k8s.io/apimachinery/pkg/api/resource"
	apiValidation "k8s.io/apimachinery/pkg/api/validation"
	utilValidation "k8s.io/apimachinery/pkg/util/validation"
)

// builder builds the PodSpec schema for one Options value.
type builder struct{ o Options }

// replace reports whether a field with the given SDKv2 ForceNew forces
// replacement for this owner.
func (b builder) replace(f forceNew) bool {
	return f == alwaysNew || (f == immutable && b.o.Immutable)
}

const restartPolicyAlwaysDescription = "Restart policy for all containers within the pod. Defaults to Always as the only option. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#restart-policy."

func (b builder) specBlock() schema.ListNestedBlock {
	object := podSpecWithDescriptions(b.podSpecObject())
	if b.o.RestartPolicyAlwaysOnly {
		restartPolicy := object.Attributes["restart_policy"].(schema.StringAttribute)
		restartPolicy.Validators = []validator.String{podStringRule("restart-policy-always")}
		restartPolicy.Description = restartPolicyAlwaysDescription
		restartPolicy.MarkdownDescription = restartPolicy.Description
		object.Attributes["restart_policy"] = restartPolicy
	}
	presence := updatable
	if b.o.SpecForceNew {
		presence = alwaysNew
	}
	block := b.block(object, 0, 1, presence)
	if b.o.SpecRequired {
		block.Validators = append(block.Validators, listvalidator.SizeAtLeast(1), listvalidator.IsRequired())
	}
	if b.o.Template {
		block.Description = podSpecApplyDescription("spec", "", true)
	} else {
		block.Description = "Specification of the desired behavior of the pod."
		if b.o.SpecRequired {
			block.Description += " Exactly one spec block is required."
		}
	}
	block.MarkdownDescription = block.Description
	return block
}

// Optional scalars default to zero, the value SDKv2 stored when omitted.
func (b builder) str(required, computed bool, f forceNew, fallback string, validators ...validator.String) schema.StringAttribute {
	a := schema.StringAttribute{Required: required, Optional: !required, Computed: computed, Validators: validators}
	if !required && !computed {
		a.Computed = true
		a.Default = stringdefault.StaticString(fallback)
	}
	if computed {
		// API-defaulted Pod fields are unchanged by metadata/provider-only updates.
		a.PlanModifiers = append(a.PlanModifiers, stringplanmodifier.UseStateForUnknown(), podEmptyStringKeepsState{})
	}
	if keep := podSpelling(validators); keep != nil && a.Computed {
		a.PlanModifiers = append(a.PlanModifiers, podSpellingPlanModifier{keep})
	}
	if b.replace(f) {
		a.PlanModifiers = append(a.PlanModifiers, podStringRequiresReplace{stringplanmodifier.RequiresReplace()})
	}
	return a
}

func (b builder) boolean(computed bool, f forceNew, fallback bool) schema.BoolAttribute {
	a := schema.BoolAttribute{Optional: true, Computed: true}
	if !computed {
		a.Default = booldefault.StaticBool(fallback)
	}
	if computed {
		a.PlanModifiers = append(a.PlanModifiers, boolplanmodifier.UseStateForUnknown())
	}
	if b.replace(f) {
		a.PlanModifiers = append(a.PlanModifiers, podBoolRequiresReplace{boolplanmodifier.RequiresReplace()})
	}
	return a
}

func (b builder) integer(required, computed bool, f forceNew, fallback int64, validators ...validator.Int64) schema.Int64Attribute {
	a := schema.Int64Attribute{Required: required, Optional: !required, Computed: computed, Validators: validators}
	if !required && !computed {
		a.Computed = true
		a.Default = int64default.StaticInt64(fallback)
	}
	if computed {
		a.PlanModifiers = append(a.PlanModifiers, int64planmodifier.UseStateForUnknown())
	}
	if b.replace(f) {
		a.PlanModifiers = append(a.PlanModifiers, podInt64RequiresReplace{int64planmodifier.RequiresReplace()})
	}
	return a
}

func (b builder) list(required bool, f forceNew, element attr.Type, validators ...validator.List) schema.ListAttribute {
	a := schema.ListAttribute{Required: required, Optional: !required, ElementType: element, Validators: validators}
	if b.replace(f) {
		a.PlanModifiers = []planmodifier.List{podListRequiresReplace{listplanmodifier.RequiresReplace()}}
	}
	return a
}

func (b builder) emptyCompatibleList(f forceNew, element attr.Type) schema.ListAttribute {
	a := schema.ListAttribute{
		Optional: true, Computed: true, ElementType: element,
		Default:       listdefault.StaticValue(types.ListNull(element)),
		PlanModifiers: []planmodifier.List{common.EmptyListCompatibility{}},
	}
	if b.replace(f) {
		a.PlanModifiers = append(a.PlanModifiers, podListRequiresReplace{listplanmodifier.RequiresReplace()})
	}
	return a
}

func (b builder) emptyCompatibleMap(f forceNew) schema.MapAttribute {
	a := schema.MapAttribute{
		Optional: true, Computed: true, ElementType: types.StringType,
		Default:       mapdefault.StaticValue(types.MapNull(types.StringType)),
		PlanModifiers: []planmodifier.Map{common.EmptyMapCompatibility{}},
	}
	if b.replace(f) {
		a.PlanModifiers = append(a.PlanModifiers, podMapRequiresReplace{mapplanmodifier.RequiresReplace()})
	}
	return a
}

func (b builder) set(f forceNew, element attr.Type) schema.SetAttribute {
	a := schema.SetAttribute{Optional: true, ElementType: element}
	if b.replace(f) {
		a.PlanModifiers = []planmodifier.Set{podSetRequiresReplace{setplanmodifier.RequiresReplace()}}
	}
	return a
}

func (b builder) mapping(computed bool, f forceNew) schema.MapAttribute {
	a := schema.MapAttribute{Optional: true, Computed: computed, ElementType: types.StringType}
	if computed {
		a.PlanModifiers = append(a.PlanModifiers, mapplanmodifier.UseStateForUnknown())
	}
	if b.replace(f) {
		a.PlanModifiers = append(a.PlanModifiers, podMapRequiresReplace{mapplanmodifier.RequiresReplace()})
	}
	return a
}

// block mirrors an SDKv2 TypeList block. Its ForceNew governs only the number
// of elements; adding or removing an element that holds a replace-on-change
// value still replaces.
func (b builder) block(object schema.NestedBlockObject, minimum, maximum int, f forceNew) schema.ListNestedBlock {
	l := schema.ListNestedBlock{NestedObject: object}
	if minimum > 0 {
		l.Validators = append(l.Validators, listvalidator.SizeAtLeast(minimum))
	}
	if maximum > 0 {
		l.Validators = append(l.Validators, listvalidator.SizeAtMost(maximum))
	}
	if b.replace(f) {
		l.PlanModifiers = []planmodifier.List{podListStructureRequiresReplace{}}
	} else if podObjectRequiresStructuralReplacement(object) {
		l.PlanModifiers = []planmodifier.List{podListInheritedRequiresReplace{object: object}}
	}
	return l
}

func (b builder) podSpecObject() schema.NestedBlockObject {
	restartPolicy := []string{"Always", "OnFailure", "Never"}
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"active_deadline_seconds":          b.integer(false, false, updatable, 0, int64validator.AtLeast(1)),
			"automount_service_account_token":  b.boolean(false, immutable, true),
			"dns_policy":                       b.str(false, false, immutable, "ClusterFirst", stringvalidator.OneOf("ClusterFirst", "ClusterFirstWithHostNet", "Default", "None")),
			"enable_service_links":             b.boolean(false, immutable, true),
			"host_ipc":                         b.boolean(false, immutable, false),
			"host_network":                     b.boolean(false, immutable, false),
			"host_pid":                         b.boolean(false, immutable, false),
			"hostname":                         b.str(false, true, immutable, ""),
			"image_pull_secrets":               b.references("name", immutable),
			"node_name":                        b.str(false, true, immutable, ""),
			"node_selector":                    b.emptyCompatibleMap(immutable),
			"priority_class_name":              b.str(false, false, immutable, ""),
			"readiness_gate":                   b.references("condition_type", immutable),
			"restart_policy":                   b.str(false, false, immutable, b.o.RestartPolicy, stringvalidator.OneOf(restartPolicy...)),
			"runtime_class_name":               b.str(false, false, immutable, ""),
			"scheduler_name":                   b.str(false, true, immutable, ""),
			"service_account_name":             b.str(false, true, immutable, ""),
			"share_process_namespace":          b.boolean(false, immutable, false),
			"subdomain":                        b.str(false, false, immutable, ""),
			"termination_grace_period_seconds": b.integer(false, false, immutable, 30, int64validator.AtLeast(0)),
		},
		Blocks: map[string]schema.Block{
			"affinity":       b.block(b.podAffinityObject(), 0, 1, immutable),
			"container":      b.block(b.podContainerObject(), 0, 0, updatable),
			"init_container": b.block(b.podContainerObject(), 0, 0, immutable),
			"dns_config":     b.block(b.podDNSConfigObject(), 0, 1, updatable),
			"host_aliases": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"hostnames": b.list(true, immutable, types.StringType),
				"ip":        b.str(true, false, immutable, "", podStringRule("ip")),
			}}, 0, 0, immutable),
			"os": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"name": b.str(true, false, updatable, "", stringvalidator.OneOf("linux", "windows")),
			}}, 0, 1, updatable),
			"security_context": b.block(b.podSecurityContextObject(), 0, 1, updatable),
			"toleration": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"effect":             b.str(false, false, immutable, "", stringvalidator.OneOf("NoSchedule", "PreferNoSchedule", "NoExecute")),
				"key":                b.str(false, false, immutable, ""),
				"operator":           b.str(false, false, immutable, "Equal", stringvalidator.OneOf("Exists", "Equal")),
				"toleration_seconds": b.str(false, false, immutable, "", podStringRule("nullable-int")),
				"value":              b.str(false, false, immutable, ""),
			}}, 0, 0, updatable),
			"topology_spread_constraint": b.block(b.podTopologySpreadObject(), 0, 0, updatable),
			"volume":                     b.block(b.podVolumeObject(), 0, 0, updatable),
		},
	}
}

// references is an attribute list of objects, set with assignment syntax.
func (b builder) references(child string, f forceNew) schema.ListAttribute {
	a := schema.ListAttribute{
		Optional: true, Computed: true,
		Description:   "List of reference objects. Omit or use null to retain API-populated references. When configured, supply at least one reference; an empty list is not omission.",
		ElementType:   types.ObjectType{AttrTypes: map[string]attr.Type{child: types.StringType}},
		Validators:    []validator.List{common.NotEmptyList(), podRequiredReference{child: child}},
		PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
	}
	if b.replace(f) {
		a.PlanModifiers = append(a.PlanModifiers, podListRequiresReplace{listplanmodifier.RequiresReplace()})
	}
	return a
}

func (b builder) podDNSConfigObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"nameservers": b.list(false, immutable, types.StringType, listvalidator.ValueStringsAre(podStringRule("ip"))),
			// SDKv2 omitted ForceNew here, but the API rejects the change on an immutable spec.
			"searches": b.list(false, immutable, types.StringType, listvalidator.ValueStringsAre(podStringRule("name"))),
		},
		Blocks: map[string]schema.Block{
			"option": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"name":  b.str(true, false, immutable, ""),
				"value": b.str(false, false, immutable, ""),
			}}, 0, 0, updatable),
		},
	}
}

func (b builder) podTopologySpreadObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"match_label_keys":     b.set(immutable, types.StringType),
			"max_skew":             b.integer(false, false, updatable, 1, int64validator.AtLeast(1)),
			"min_domains":          b.integer(false, false, immutable, 0, int64validator.AtLeast(1)),
			"node_affinity_policy": b.str(false, false, immutable, "", stringvalidator.OneOf("Honor", "Ignore")),
			"node_taints_policy":   b.str(false, false, immutable, "", stringvalidator.OneOf("Honor", "Ignore")),
			"topology_key":         b.str(false, false, updatable, ""),
			"when_unsatisfiable":   b.str(false, false, updatable, "DoNotSchedule", stringvalidator.OneOf("DoNotSchedule", "ScheduleAnyway")),
		},
		Blocks: map[string]schema.Block{
			"label_selector": b.block(b.podLabelSelectorObject(), 0, 0, updatable),
		},
	}
}

func (b builder) podAffinityObject() schema.NestedBlockObject {
	nodeTerm := schema.NestedBlockObject{Blocks: map[string]schema.Block{
		"match_expressions": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
			"key":      b.str(false, false, updatable, ""),
			"operator": b.str(false, false, updatable, "", stringvalidator.OneOf("In", "NotIn", "Exists", "DoesNotExist", "Gt", "Lt")),
			"values":   b.set(updatable, types.StringType),
		}}, 0, 0, updatable),
		"match_fields": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
			"key":      b.str(true, false, alwaysNew, ""),
			"operator": b.str(true, false, alwaysNew, ""),
			"values":   b.set(alwaysNew, types.StringType),
		}}, 0, 0, alwaysNew),
	}}
	node := schema.NestedBlockObject{Blocks: map[string]schema.Block{
		"required_during_scheduling_ignored_during_execution": b.block(schema.NestedBlockObject{Blocks: map[string]schema.Block{
			"node_selector_term": b.block(nodeTerm, 0, 0, updatable),
		}}, 0, 1, updatable),
		"preferred_during_scheduling_ignored_during_execution": b.block(schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{"weight": b.integer(true, false, updatable, 0)},
			Blocks:     map[string]schema.Block{"preference": b.block(nodeTerm, 1, 1, updatable)},
		}, 0, 0, updatable),
	}}
	pod := schema.NestedBlockObject{Blocks: map[string]schema.Block{
		"required_during_scheduling_ignored_during_execution": b.block(b.podAffinityTermObject(), 0, 0, updatable),
		"preferred_during_scheduling_ignored_during_execution": b.block(schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{"weight": b.integer(true, false, updatable, 0)},
			Blocks:     map[string]schema.Block{"pod_affinity_term": b.block(b.podAffinityTermObject(), 1, 1, updatable)},
		}, 0, 0, updatable),
	}}
	return schema.NestedBlockObject{Blocks: map[string]schema.Block{
		"node_affinity":     b.block(node, 0, 1, updatable),
		"pod_affinity":      b.block(pod, 0, 1, updatable),
		"pod_anti_affinity": b.block(pod, 0, 1, updatable),
	}}
}

func (b builder) podAffinityTermObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"namespaces":   b.set(updatable, types.StringType),
			"topology_key": b.str(true, false, updatable, ""),
		},
		Blocks: map[string]schema.Block{
			"label_selector":     b.block(b.podLabelSelectorObject(), 0, 0, updatable),
			"namespace_selector": b.block(b.podLabelSelectorObject(), 0, 0, updatable),
		},
	}
}

func (b builder) podLabelSelectorObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{"match_labels": b.mapping(false, updatable)},
		Blocks: map[string]schema.Block{
			"match_expressions": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"key":      b.str(false, false, updatable, ""),
				"operator": b.str(false, false, updatable, ""),
				"values":   b.set(updatable, types.StringType),
			}}, 0, 0, updatable),
		},
	}
}

type podStringRule string

func (v podStringRule) Description(context.Context) string {
	return "must satisfy the Kubernetes " + string(v) + " constraint"
}
func (v podStringRule) MarkdownDescription(ctx context.Context) string { return v.Description(ctx) }
func (v podStringRule) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	s := req.ConfigValue.ValueString()
	var messages []string
	switch v {
	case "ip":
		if net.ParseIP(s) == nil {
			messages = append(messages, "must be a valid IP address")
		}
	case "name":
		messages = apiValidation.NameIsDNSSubdomain(s, false)
	case "port":
		if n, err := strconv.Atoi(s); err == nil {
			messages = utilValidation.IsValidPortNum(n)
		} else {
			messages = utilValidation.IsValidPortName(s)
		}
	case "nullable-int":
		if s != "" {
			if _, err := strconv.ParseInt(s, 10, 64); err != nil {
				messages = append(messages, "must be an integer or an empty string: "+err.Error())
			}
		}
	case "mode":
		n, err := strconv.ParseInt(s, 8, 32)
		if !strings.HasPrefix(s, "0") || err != nil || n < 0 || n > 0777 {
			messages = append(messages, "must be an octal numeral starting with 0 between 0 and 0777")
		}
	case "path":
		if path.IsAbs(s) || strings.HasPrefix(s, "..") {
			messages = append(messages, "must be a relative path that does not start with '..'")
		}
		for _, part := range strings.Split(filepath.ToSlash(s), "/") {
			if part == ".." {
				messages = append(messages, "must not contain a '..' path element")
				break
			}
		}
	case "quantity":
		if _, err := kquantity.ParseQuantity(s); err != nil {
			messages = append(messages, err.Error())
		}
	case "restart-policy-always":
		if s != "Always" {
			legacyPath := strings.NewReplacer("[", ".", "]", "").Replace(req.Path.String())
			// Keep the legacy diagnostic in the summary, which Terraform does not wrap.
			resp.Diagnostics.AddAttributeError(req.Path, fmt.Sprintf("expected %s to be one of [\"Always\"], got %s", legacyPath, s), "")
		}
	}
	for _, message := range messages {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Attribute Value", message)
	}
}

type podRequiredReference struct{ child string }

func (v podRequiredReference) Description(context.Context) string {
	return "each object must provide " + v.child
}
func (v podRequiredReference) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (v podRequiredReference) ValidateList(_ context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	for i, element := range req.ConfigValue.Elements() {
		if element.IsUnknown() {
			continue
		}
		if element.IsNull() {
			resp.Diagnostics.AddAttributeError(req.Path.AtListIndex(i), "Missing Required Argument", "The reference must be an object.")
			continue
		}
		object := element.(types.Object)
		child := object.Attributes()[v.child]
		if child.IsNull() {
			resp.Diagnostics.AddAttributeError(req.Path.AtListIndex(i).AtName(v.child), "Missing Required Argument", fmt.Sprintf("The argument %q is required.", v.child))
		}
	}
}
