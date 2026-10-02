// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

// Package podtemplate provides the native Framework schema and pure API
// conversions for the PodSpec of workload pod templates (Deployment,
// DaemonSet, StatefulSet). It mirrors SDKv2's podSpecFields(true, false).
package podtemplate

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

// Options selects the owner-specific variations of the template PodSpec.
type Options struct {
	// RestartPolicyAlways restricts spec.restart_policy to "Always", as the
	// SDKv2 Deployment did. DaemonSet and StatefulSet keep the generic
	// Always/OnFailure/Never validation.
	RestartPolicyAlways bool
}

const restartPolicyAlwaysDescription = "Restart policy for all containers within the pod. Defaults to Always as the only option. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#restart-policy."

// SpecBlock returns the template's "spec" block: an optional list of at most
// one PodSpec object. Owners that require the block or need an owner-specific
// description may append validators or set Description on the result.
func SpecBlock(options Options) schema.ListNestedBlock {
	object := podSpecWithDescriptions(podSpecObject())
	if options.RestartPolicyAlways {
		restartPolicy := object.Attributes["restart_policy"].(schema.StringAttribute)
		restartPolicy.Validators = []validator.String{podStringRule("restart-policy-always")}
		restartPolicy.Description = restartPolicyAlwaysDescription
		restartPolicy.MarkdownDescription = restartPolicy.Description
		object.Attributes["restart_policy"] = restartPolicy
	}
	block := podBlock(object, 0, 1, false)
	block.Description = podSpecApplyDescription("spec", "", true)
	block.MarkdownDescription = block.Description
	return block
}

// These constructors describe the template PodSpec's static native schema. The
// zero defaults preserve SDKv2's omitted-scalar values; null and unknown remain
// distinct until Framework planning applies the defaults or the API boundary is reached.
func podString(required, computed, replace bool, fallback string, validators ...validator.String) schema.StringAttribute {
	a := schema.StringAttribute{Required: required, Optional: !required, Computed: computed, Validators: validators}
	if !required && !computed {
		a.Computed = true
		a.Default = stringdefault.StaticString(fallback)
	}
	if computed {
		// API-defaulted Pod fields are unchanged by metadata/provider-only updates.
		a.PlanModifiers = append(a.PlanModifiers, stringplanmodifier.UseStateForUnknown())
	}
	if replace {
		a.PlanModifiers = append(a.PlanModifiers, podStringRequiresReplace{stringplanmodifier.RequiresReplace()})
	}
	return a
}

func podBool(computed, fallback bool) schema.BoolAttribute {
	a := schema.BoolAttribute{Optional: true, Computed: true}
	if !computed {
		a.Default = booldefault.StaticBool(fallback)
	}
	if computed {
		a.PlanModifiers = append(a.PlanModifiers, boolplanmodifier.UseStateForUnknown())
	}
	return a
}

func podInt(required, computed bool, fallback int64, validators ...validator.Int64) schema.Int64Attribute {
	a := schema.Int64Attribute{Required: required, Optional: !required, Computed: computed, Validators: validators}
	if !required && !computed {
		a.Computed = true
		a.Default = int64default.StaticInt64(fallback)
	}
	if computed {
		a.PlanModifiers = append(a.PlanModifiers, int64planmodifier.UseStateForUnknown())
	}
	return a
}

func podList(required bool, element attr.Type, validators ...validator.List) schema.ListAttribute {
	return schema.ListAttribute{Required: required, Optional: !required, ElementType: element, Validators: validators}
}

func podEmptyCompatibleList(element attr.Type) schema.ListAttribute {
	return schema.ListAttribute{
		Optional: true, Computed: true, ElementType: element,
		Default:       listdefault.StaticValue(types.ListNull(element)),
		PlanModifiers: []planmodifier.List{common.EmptyListCompatibility{}},
	}
}

func podEmptyCompatibleMap() schema.MapAttribute {
	return schema.MapAttribute{
		Optional: true, Computed: true, ElementType: types.StringType,
		Default:       mapdefault.StaticValue(types.MapNull(types.StringType)),
		PlanModifiers: []planmodifier.Map{common.EmptyMapCompatibility{}},
	}
}

func podSet(replace bool, element attr.Type) schema.SetAttribute {
	a := schema.SetAttribute{Optional: true, ElementType: element}
	if replace {
		a.PlanModifiers = []planmodifier.Set{podSetRequiresReplace{setplanmodifier.RequiresReplace()}}
	}
	return a
}

func podMap(computed, replace bool) schema.MapAttribute {
	a := schema.MapAttribute{Optional: true, Computed: computed, ElementType: types.StringType}
	if computed {
		a.PlanModifiers = append(a.PlanModifiers, mapplanmodifier.UseStateForUnknown())
	}
	if replace {
		a.PlanModifiers = append(a.PlanModifiers, podMapRequiresReplace{mapplanmodifier.RequiresReplace()})
	}
	return a
}

func podBlock(object schema.NestedBlockObject, minimum, maximum int, replaceStructure bool) schema.ListNestedBlock {
	b := schema.ListNestedBlock{NestedObject: object}
	if minimum > 0 {
		b.Validators = append(b.Validators, listvalidator.SizeAtLeast(minimum))
	}
	if maximum > 0 {
		b.Validators = append(b.Validators, listvalidator.SizeAtMost(maximum))
	}
	if replaceStructure {
		b.PlanModifiers = []planmodifier.List{podListStructureRequiresReplace{}}
	} else if podObjectRequiresStructuralReplacement(object) {
		b.PlanModifiers = []planmodifier.List{podListInheritedRequiresReplace{object: object}}
	}
	return b
}

func podSpecObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"active_deadline_seconds":          podInt(false, false, 0, int64validator.AtLeast(1)),
			"automount_service_account_token":  podBool(false, true),
			"dns_policy":                       podString(false, false, false, "ClusterFirst", stringvalidator.OneOf("ClusterFirst", "ClusterFirstWithHostNet", "Default", "None")),
			"enable_service_links":             podBool(false, true),
			"host_ipc":                         podBool(false, false),
			"host_network":                     podBool(false, false),
			"host_pid":                         podBool(false, false),
			"hostname":                         podString(false, true, false, ""),
			"image_pull_secrets":               podComputedReferences("name"),
			"node_name":                        podString(false, true, false, ""),
			"node_selector":                    podEmptyCompatibleMap(),
			"priority_class_name":              podString(false, false, false, ""),
			"readiness_gate":                   podComputedReferences("condition_type"),
			"restart_policy":                   podString(false, false, false, "Always", stringvalidator.OneOf("Always", "OnFailure", "Never")),
			"runtime_class_name":               podString(false, false, false, ""),
			"scheduler_name":                   podString(false, true, false, ""),
			"service_account_name":             podString(false, true, false, ""),
			"share_process_namespace":          podBool(false, false),
			"subdomain":                        podString(false, false, false, ""),
			"termination_grace_period_seconds": podInt(false, false, 30, int64validator.AtLeast(0)),
		},
		Blocks: map[string]schema.Block{
			"affinity":       podBlock(podAffinityObject(), 0, 1, false),
			"container":      podBlock(podContainerObject(), 0, 0, false),
			"init_container": podBlock(podContainerObject(), 0, 0, false),
			"dns_config":     podBlock(podDNSConfigObject(), 0, 1, false),
			"host_aliases": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"hostnames": podList(true, types.StringType),
				"ip":        podString(true, false, false, "", podStringRule("ip")),
			}}, 0, 0, false),
			"os": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"name": podString(true, false, false, "", stringvalidator.OneOf("linux", "windows")),
			}}, 0, 1, false),
			"security_context": podBlock(podSecurityContextObject(), 0, 1, false),
			"toleration": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"effect":             podString(false, false, false, "", stringvalidator.OneOf("NoSchedule", "PreferNoSchedule", "NoExecute")),
				"key":                podString(false, false, false, ""),
				"operator":           podString(false, false, false, "Equal", stringvalidator.OneOf("Exists", "Equal")),
				"toleration_seconds": podString(false, false, false, "", podStringRule("nullable-int")),
				"value":              podString(false, false, false, ""),
			}}, 0, 0, false),
			"topology_spread_constraint": podBlock(podTopologySpreadObject(), 0, 0, false),
			"volume":                     podBlock(podVolumeObject(), 0, 0, false),
		},
	}
}

// References are atomic objects with explicitly validated children. Assignment
// syntax is required because resources disables legacy block decoding.
func podComputedReferences(child string) schema.ListAttribute {
	return schema.ListAttribute{
		Optional: true, Computed: true,
		Description:   "List of reference objects. Omit or use null to retain API-populated references. When configured, supply at least one reference; an empty list is not omission.",
		ElementType:   types.ObjectType{AttrTypes: map[string]attr.Type{child: types.StringType}},
		Validators:    []validator.List{listvalidator.SizeAtLeast(1), podRequiredReference{child: child}},
		PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
	}
}

func podDNSConfigObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"nameservers": podList(false, types.StringType, listvalidator.ValueStringsAre(podStringRule("ip"))),
			"searches":    podList(false, types.StringType, listvalidator.ValueStringsAre(podStringRule("name"))),
		},
		Blocks: map[string]schema.Block{
			"option": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"name":  podString(true, false, false, ""),
				"value": podString(false, false, false, ""),
			}}, 0, 0, false),
		},
	}
}

func podTopologySpreadObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"match_label_keys":     podSet(false, types.StringType),
			"max_skew":             podInt(false, false, 1, int64validator.AtLeast(1)),
			"min_domains":          podInt(false, false, 0, int64validator.AtLeast(1)),
			"node_affinity_policy": podString(false, false, false, "", stringvalidator.OneOf("Honor", "Ignore")),
			"node_taints_policy":   podString(false, false, false, "", stringvalidator.OneOf("Honor", "Ignore")),
			"topology_key":         podString(false, false, false, ""),
			"when_unsatisfiable":   podString(false, false, false, "DoNotSchedule", stringvalidator.OneOf("DoNotSchedule", "ScheduleAnyway")),
		},
		Blocks: map[string]schema.Block{
			"label_selector": podBlock(podLabelSelectorObject(), 0, 0, false),
		},
	}
}

func podAffinityObject() schema.NestedBlockObject {
	nodeTerm := schema.NestedBlockObject{Blocks: map[string]schema.Block{
		"match_expressions": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
			"key":      podString(false, false, false, ""),
			"operator": podString(false, false, false, "", stringvalidator.OneOf("In", "NotIn", "Exists", "DoesNotExist", "Gt", "Lt")),
			"values":   podSet(false, types.StringType),
		}}, 0, 0, false),
		// SDKv2 kept match_fields ForceNew even for updatable pod templates.
		"match_fields": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
			"key":      podString(true, false, true, ""),
			"operator": podString(true, false, true, ""),
			"values":   podSet(true, types.StringType),
		}}, 0, 0, true),
	}}
	node := schema.NestedBlockObject{Blocks: map[string]schema.Block{
		"required_during_scheduling_ignored_during_execution": podBlock(schema.NestedBlockObject{Blocks: map[string]schema.Block{
			"node_selector_term": podBlock(nodeTerm, 0, 0, false),
		}}, 0, 1, false),
		"preferred_during_scheduling_ignored_during_execution": podBlock(schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{"weight": podInt(true, false, 0)},
			Blocks:     map[string]schema.Block{"preference": podBlock(nodeTerm, 1, 1, false)},
		}, 0, 0, false),
	}}
	pod := schema.NestedBlockObject{Blocks: map[string]schema.Block{
		"required_during_scheduling_ignored_during_execution": podBlock(podAffinityTermObject(), 0, 0, false),
		"preferred_during_scheduling_ignored_during_execution": podBlock(schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{"weight": podInt(true, false, 0)},
			Blocks:     map[string]schema.Block{"pod_affinity_term": podBlock(podAffinityTermObject(), 1, 1, false)},
		}, 0, 0, false),
	}}
	return schema.NestedBlockObject{Blocks: map[string]schema.Block{
		"node_affinity":     podBlock(node, 0, 1, false),
		"pod_affinity":      podBlock(pod, 0, 1, false),
		"pod_anti_affinity": podBlock(pod, 0, 1, false),
	}}
}

func podAffinityTermObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"namespaces":   podSet(false, types.StringType),
			"topology_key": podString(true, false, false, ""),
		},
		Blocks: map[string]schema.Block{
			"label_selector":     podBlock(podLabelSelectorObject(), 0, 0, false),
			"namespace_selector": podBlock(podLabelSelectorObject(), 0, 0, false),
		},
	}
}

func podLabelSelectorObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{"match_labels": podMap(false, false)},
		Blocks: map[string]schema.Block{
			"match_expressions": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"key":      podString(false, false, false, ""),
				"operator": podString(false, false, false, ""),
				"values":   podSet(false, types.StringType),
			}}, 0, 0, false),
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
