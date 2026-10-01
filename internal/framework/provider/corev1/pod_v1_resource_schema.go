// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"fmt"
	"net"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
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

func (p *PodV1) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	spec := podBlock(podSpecWithDescriptions(podSpecObject()), 1, 1, false)
	spec.Description = "Specification of the desired behavior of the pod. Exactly one spec block is required."
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "A pod is a group of one or more containers, the shared storage for those containers, and options about how to run the containers. Pods are always co-located and co-scheduled, and run in a shared context. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod/.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"target_state": schema.ListAttribute{
				Description: "A list of the pod phases that indicate whether it was successfully created. Options: Pending, Running, Succeeded, Failed, Unknown. Defaults to Running.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.ValueStringsAre(stringvalidator.OneOf("Pending", "Running", "Succeeded", "Failed", "Unknown")),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": common.NamespacedMetadataSchema("pod", true),
			"spec":     spec,
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Delete: true}),
		},
	}
}

// These constructors describe the Pod's static native schema. The zero defaults
// preserve SDKv2's omitted-scalar values; null and unknown remain distinct until
// Framework planning applies the defaults or the API boundary is reached.
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

func podBool(computed, replace, fallback bool) schema.BoolAttribute {
	a := schema.BoolAttribute{Optional: true, Computed: true}
	if !computed {
		a.Default = booldefault.StaticBool(fallback)
	}
	if computed {
		a.PlanModifiers = append(a.PlanModifiers, boolplanmodifier.UseStateForUnknown())
	}
	if replace {
		a.PlanModifiers = append(a.PlanModifiers, podBoolRequiresReplace{boolplanmodifier.RequiresReplace()})
	}
	return a
}

func podInt(required, computed, replace bool, fallback int64, validators ...validator.Int64) schema.Int64Attribute {
	a := schema.Int64Attribute{Required: required, Optional: !required, Computed: computed, Validators: validators}
	if !required && !computed {
		a.Computed = true
		a.Default = int64default.StaticInt64(fallback)
	}
	if computed {
		a.PlanModifiers = append(a.PlanModifiers, int64planmodifier.UseStateForUnknown())
	}
	if replace {
		a.PlanModifiers = append(a.PlanModifiers, podInt64RequiresReplace{int64planmodifier.RequiresReplace()})
	}
	return a
}

func podList(required, replace bool, element attr.Type, validators ...validator.List) schema.ListAttribute {
	a := schema.ListAttribute{Required: required, Optional: !required, ElementType: element, Validators: validators}
	if replace {
		a.PlanModifiers = []planmodifier.List{podListRequiresReplace{listplanmodifier.RequiresReplace()}}
	}
	return a
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
			"active_deadline_seconds":          podInt(false, false, false, 0, int64validator.AtLeast(1)),
			"automount_service_account_token":  podBool(false, true, true),
			"dns_policy":                       podString(false, false, true, "ClusterFirst", stringvalidator.OneOf("ClusterFirst", "ClusterFirstWithHostNet", "Default", "None")),
			"enable_service_links":             podBool(false, true, true),
			"host_ipc":                         podBool(false, true, false),
			"host_network":                     podBool(false, true, false),
			"host_pid":                         podBool(false, true, false),
			"hostname":                         podString(false, true, true, ""),
			"image_pull_secrets":               podComputedReferences("name"),
			"node_name":                        podString(false, true, true, ""),
			"node_selector":                    podMap(false, true),
			"priority_class_name":              podString(false, false, true, ""),
			"readiness_gate":                   podComputedReferences("condition_type"),
			"restart_policy":                   podString(false, false, true, "Always", stringvalidator.OneOf("Always", "OnFailure", "Never")),
			"runtime_class_name":               podString(false, false, true, ""),
			"scheduler_name":                   podString(false, true, true, ""),
			"service_account_name":             podString(false, true, true, ""),
			"share_process_namespace":          podBool(false, true, false),
			"subdomain":                        podString(false, false, true, ""),
			"termination_grace_period_seconds": podInt(false, false, true, 30, int64validator.AtLeast(0)),
		},
		Blocks: map[string]schema.Block{
			"affinity":       podBlock(podAffinityObject(), 0, 1, true),
			"container":      podBlock(podContainerObject(), 0, 0, true),
			"init_container": podBlock(podContainerObject(), 0, 0, true),
			"dns_config":     podBlock(podDNSConfigObject(), 0, 1, false),
			"host_aliases": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"hostnames": podList(true, true, types.StringType),
				"ip":        podString(true, false, true, "", podStringRule("ip")),
			}}, 0, 0, true),
			"os": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"name": podString(true, false, false, "", stringvalidator.OneOf("linux", "windows")),
			}}, 0, 1, false),
			"security_context": podBlock(podSecurityContextObject(), 0, 1, false),
			"toleration": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"effect":             podString(false, false, true, "", stringvalidator.OneOf("NoSchedule", "PreferNoSchedule", "NoExecute")),
				"key":                podString(false, false, true, ""),
				"operator":           podString(false, false, true, "Equal", stringvalidator.OneOf("Exists", "Equal")),
				"toleration_seconds": podString(false, false, true, "", podStringRule("nullable-int")),
				"value":              podString(false, false, true, ""),
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
		PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown(), podListRequiresReplace{listplanmodifier.RequiresReplace()}},
	}
}

func podDNSConfigObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"nameservers": podList(false, true, types.StringType, listvalidator.ValueStringsAre(podStringRule("ip"))),
			"searches":    podList(false, true, types.StringType, listvalidator.ValueStringsAre(podStringRule("name"))),
		},
		Blocks: map[string]schema.Block{
			"option": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"name":  podString(true, false, true, ""),
				"value": podString(false, false, true, ""),
			}}, 0, 0, false),
		},
	}
}

func podTopologySpreadObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"match_label_keys":     podSet(true, types.StringType),
			"max_skew":             podInt(false, false, false, 1, int64validator.AtLeast(1)),
			"min_domains":          podInt(false, false, true, 0, int64validator.AtLeast(1)),
			"node_affinity_policy": podString(false, false, true, "", stringvalidator.OneOf("Honor", "Ignore")),
			"node_taints_policy":   podString(false, false, true, "", stringvalidator.OneOf("Honor", "Ignore")),
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
			Attributes: map[string]schema.Attribute{"weight": podInt(true, false, false, 0)},
			Blocks:     map[string]schema.Block{"preference": podBlock(nodeTerm, 1, 1, false)},
		}, 0, 0, false),
	}}
	pod := schema.NestedBlockObject{Blocks: map[string]schema.Block{
		"required_during_scheduling_ignored_during_execution": podBlock(podAffinityTermObject(), 0, 0, false),
		"preferred_during_scheduling_ignored_during_execution": podBlock(schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{"weight": podInt(true, false, false, 0)},
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
			messages = append(messages, "must be a Kubernetes resource quantity: "+err.Error())
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
