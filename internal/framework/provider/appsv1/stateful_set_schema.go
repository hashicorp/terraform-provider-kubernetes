// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common/podspec"
	corev1 "k8s.io/api/core/v1"
)

var statefulSetFrozenSchema = common.FrozenSchema(buildStatefulSetSchema)

func (r *StatefulSetV1) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	statefulSetFrozenSchema(ctx, req, resp)
}

func buildStatefulSetSchema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: 2,
		Description: "Manages the deployment and scaling of a set of Pods, and provides guarantees about the ordering and uniqueness of these Pods. " +
			"Like a Deployment, a StatefulSet manages Pods that are based on an identical container spec. " +
			"Unlike a Deployment, a StatefulSet maintains a sticky identity for each of their Pods. " +
			"These pods are created from the same spec, but are not interchangeable: each has a persistent identifier that it maintains across any rescheduling. " +
			"A StatefulSet operates under the same pattern as any other Controller. " +
			"You define your desired state in a StatefulSet object, and the StatefulSet controller makes any necessary updates to get there from the current state.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"wait_for_rollout": schema.BoolAttribute{
				Description: "Wait for the rollout of the stateful set to complete. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": common.WithEmptyMetadataCompatibility(common.NamespacedMetadataSchema("stateful set", true)),
			"spec": schema.ListNestedBlock{
				Description: "Spec defines the desired identities of pods in this set.",
				Validators: []validator.List{
					listvalidator.IsRequired(),
					listvalidator.SizeAtLeast(1),
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"pod_management_policy": schema.StringAttribute{
							Description: "Controls how pods are created during initial scale up, when replacing pods on nodes, or when scaling down.",
							Optional:    true,
							Computed:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("OrderedReady", "Parallel"),
							},
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
								stringplanmodifier.RequiresReplace(),
							},
						},
						"replicas": schema.StringAttribute{
							Description: "The desired number of replicas of the given Template, in the sense that they are instantiations of the same Template. Value must be a positive integer.",
							Optional:    true,
							Computed:    true,
							Validators:  []validator.String{nullableIntStringValidator{}},
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
								preserveReplicasOnEmpty{},
							},
						},
						"revision_history_limit": schema.Int64Attribute{
							Description: "The maximum number of revisions that will be maintained in the StatefulSet's revision history. The default value is 10.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
							Validators: []validator.Int64{int64validator.Between(0, 2147483647)},
						},
						"service_name": schema.StringAttribute{
							Description:   "The name of the service that governs this StatefulSet. This service must exist before the StatefulSet, and is responsible for the network identity of the set.",
							Required:      true,
							PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
						},
						"min_ready_seconds": schema.Int64Attribute{
							Description: "Minimum number of seconds for which a newly created pod should be ready without any of its container crashing for it to be considered available. Defaults to 0 (pod will be considered available as soon as it is ready).",
							Optional:    true,
							Computed:    true,
							Default:     int64default.StaticInt64(0),
							Validators:  []validator.Int64{nonNegativeInt64Validator{}},
						},
						"persistent_volume_claim_retention_policy": persistentVolumeClaimRetentionPolicyAttribute(),
						"ordinals": schema.SingleNestedAttribute{
							Description: "Ordinal numbering for StatefulSet pods. Kubernetes defaults the start ordinal to zero. Requires Kubernetes 1.31 or later.",
							Optional:    true,
							Attributes: map[string]schema.Attribute{
								"start": schema.Int64Attribute{Required: true, Description: "The first pod ordinal. Changing it changes the pods managed by this StatefulSet.", Validators: []validator.Int64{int64validator.Between(0, 2147483647)}},
							},
						},
					},
					Blocks: map[string]schema.Block{
						"selector":              labelSelectorBlock(true, "A label query over pods that should match the replica count. It must match the pod template's labels."),
						"template":              statefulSetTemplateBlock(),
						"update_strategy":       updateStrategyBlock(),
						"volume_claim_template": persistentVolumeClaimBlock(),
					},
				},
			},
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true}),
		},
	}
}

func statefulSetTemplateBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "The object that describes the pod that will be created if insufficient replicas are detected. Each pod stamped out by the StatefulSet will fulfill this Template.",
		Validators:  []validator.List{listvalidator.IsRequired(), listvalidator.SizeAtLeast(1), listvalidator.SizeAtMost(1)},
		NestedObject: schema.NestedBlockObject{
			Blocks: map[string]schema.Block{
				"metadata": workloadTemplateMetadataBlock(),
				"spec":     podspec.For(podspec.StatefulSet()).Spec,
			},
		},
	}
}

func labelSelectorBlock(required bool, description string) schema.ListNestedBlock {
	validators := []validator.List{listvalidator.SizeAtMost(1)}
	if required {
		validators = append([]validator.List{listvalidator.IsRequired(), listvalidator.SizeAtLeast(1)}, validators...)
	}
	return schema.ListNestedBlock{
		Description:   description,
		Validators:    validators,
		PlanModifiers: []planmodifier.List{workloadSelectorRequiresReplace()},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"match_labels": schema.MapAttribute{Description: common.LabelSelectorMatchLabelsDescription, Optional: true, ElementType: types.StringType},
			},
			Blocks: map[string]schema.Block{
				"match_expressions": schema.ListNestedBlock{
					Description: common.LabelSelectorMatchExpressionsDescription,
					NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
						"key":      schema.StringAttribute{Description: common.LabelSelectorKeyDescription, Optional: true},
						"operator": schema.StringAttribute{Description: common.LabelSelectorOperatorDescription, Optional: true},
						"values": schema.SetAttribute{
							Description: common.LabelSelectorValuesDescription,
							Optional:    true,
							ElementType: types.StringType,
						},
					}},
				},
			},
		},
	}
}

func updateStrategyBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "The strategy that the StatefulSet controller will use to perform updates.",
		NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{
				Description: "Indicates the type of the StatefulSet update strategy. Default is RollingUpdate",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("RollingUpdate"),
				Validators: []validator.String{
					stringvalidator.OneOf("RollingUpdate", "OnDelete"),
				},
			},
		}, Blocks: map[string]schema.Block{
			"rolling_update": schema.ListNestedBlock{
				Description: "RollingUpdate strategy type for StatefulSet",
				NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
					"partition": schema.Int64Attribute{
						Description: "Indicates the ordinal at which the StatefulSet should be partitioned. Default value is 0.",
						Optional:    true,
						Computed:    true,
						Default:     int64default.StaticInt64(0),
					},
				}},
			},
		}},
	}
}

func persistentVolumeClaimRetentionPolicyBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
			"when_deleted": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("Retain"), Validators: []validator.String{stringvalidator.OneOf("Retain", "Delete")}},
			"when_scaled":  schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("Retain"), Validators: []validator.String{stringvalidator.OneOf("Retain", "Delete")}},
		}},
	}
}

func persistentVolumeClaimRetentionPolicyAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Description: "The field controls if and how PVCs are deleted during the lifecycle of a StatefulSet.",
		Optional:    true,
		Computed:    true,
		PlanModifiers: []planmodifier.Object{
			objectplanmodifier.UseStateForUnknown(),
		},
		Attributes: map[string]schema.Attribute{
			"when_deleted": schema.StringAttribute{
				Description: "This field controls what happens when a Statefulset is deleted. Default is Retain.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("Retain"),
				Validators:  []validator.String{stringvalidator.OneOf("Retain", "Delete")},
			},
			"when_scaled": schema.StringAttribute{
				Description: "This field controls what happens when a Statefulset is scaled. Default is Retain.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("Retain"),
				Validators:  []validator.String{stringvalidator.OneOf("Retain", "Delete")},
			},
		},
	}
}

func persistentVolumeClaimBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description:   "A list of claims that pods are allowed to reference. Every claim in this list must have at least one matching (by name) volumeMount in one container in the template.",
		PlanModifiers: []planmodifier.List{statefulSetVolumeClaimRequiresReplace{}},
		NestedObject: schema.NestedBlockObject{
			Blocks: map[string]schema.Block{
				"metadata": common.WithEmptyMetadataCompatibility(common.NamespacedMetadataSchema("persistent volume claim", true)),
				"spec": schema.ListNestedBlock{
					Description: "Spec defines the desired characteristics of a volume requested by a pod author. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes/#persistentvolumeclaims",
					Validators:  []validator.List{listvalidator.IsRequired(), listvalidator.SizeAtLeast(1), listvalidator.SizeAtMost(1)},
					NestedObject: schema.NestedBlockObject{
						Attributes: map[string]schema.Attribute{
							"access_modes": schema.SetAttribute{Description: "A set of the desired access modes the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes", Required: true, ElementType: types.StringType, Validators: []validator.Set{setvalidator.ValueStringsAre(stringvalidator.OneOf("ReadWriteOnce", "ReadOnlyMany", "ReadWriteMany", "ReadWriteOncePod"))}},
							"volume_name": schema.StringAttribute{
								Description: "The binding reference to the PersistentVolume backing this claim.",
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
									stringplanmodifier.RequiresReplace(),
								},
							},
							"storage_class_name": schema.StringAttribute{
								Description: "Name of the storage class requested by the claim",
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
									stringplanmodifier.RequiresReplace(),
								},
							},
							"volume_mode": schema.StringAttribute{
								Description: "Defines what type of volume is required by the claim.",
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
									stringplanmodifier.RequiresReplace(),
								},
								Validators: []validator.String{stringvalidator.OneOf(string(corev1.PersistentVolumeBlock), string(corev1.PersistentVolumeFilesystem))},
							},
						},
						Blocks: map[string]schema.Block{
							"resources": schema.ListNestedBlock{
								Description: "A list of the minimum resources the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources",
								Validators:  []validator.List{listvalidator.IsRequired(), listvalidator.SizeAtLeast(1), listvalidator.SizeAtMost(1)},
								NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
									"limits":   claimQuantitiesAttribute("Map describing the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/"),
									"requests": claimQuantitiesAttribute("Map describing the minimum amount of compute resources required. If this is omitted for a container, it defaults to `limits` if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/"),
								}},
							},
							"selector": labelSelectorBlock(false, "A label query over volumes to consider for binding."),
						},
					},
				},
			},
		},
	}
}

// claimQuantitiesAttribute keeps an empty map SDKv2 wrote to state when the
// configuration omits it; replacement is decided by the claim-level modifier.
func claimQuantitiesAttribute(description string) schema.MapAttribute {
	return schema.MapAttribute{
		Description:   description,
		Optional:      true,
		Computed:      true,
		ElementType:   types.StringType,
		Default:       mapdefault.StaticValue(types.MapNull(types.StringType)),
		PlanModifiers: []planmodifier.Map{statefulSetQuantityMapModifier{}, common.EmptyMapCompatibility{}},
	}
}

type nullableIntStringValidator struct{}

type preserveReplicasOnEmpty struct{}

func (m preserveReplicasOnEmpty) Description(context.Context) string {
	return "preserves the current replica count when configured as an empty string"
}

func (m preserveReplicasOnEmpty) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m preserveReplicasOnEmpty) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || req.ConfigValue.ValueString() != "" {
		return
	}
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() || req.StateValue.ValueString() == "" {
		return
	}
	// Empty means leave scaling to another controller. Core permits retaining
	// the exact non-null prior value for an equivalent configured value.
	resp.PlanValue = req.StateValue
}

func (v nullableIntStringValidator) Description(context.Context) string {
	return "must be an integer string or empty"
}
func (v nullableIntStringValidator) MarkdownDescription(context.Context) string {
	return v.Description(context.Background())
}
func (v nullableIntStringValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if value == "" {
		return
	}
	if _, err := strconv.ParseInt(value, 10, 64); err != nil {
		resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(req.Path, err.Error(), req.ConfigValue.String()))
	}
}

type nonNegativeInt64Validator struct{}

func (v nonNegativeInt64Validator) Description(context.Context) string {
	return "must be greater than or equal to 0"
}
func (v nonNegativeInt64Validator) MarkdownDescription(context.Context) string {
	return v.Description(context.Background())
}
func (v nonNegativeInt64Validator) ValidateInt64(_ context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if req.ConfigValue.ValueInt64() < 0 {
		resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(req.Path, "must be greater than or equal to 0", req.ConfigValue.String()))
	}
}
