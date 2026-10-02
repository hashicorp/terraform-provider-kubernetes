// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common/podspec"
	corev1 "k8s.io/api/core/v1"
)

// The schema is built and frozen once per process; see common.FrozenSchema.
var statefulSetFrozenSchema = common.FrozenSchema(buildStatefulSetSchema)

func (r *StatefulSetV1) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	statefulSetFrozenSchema(ctx, req, resp)
}

func buildStatefulSetSchema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "Manages the deployment and scaling of a set of Pods, and provides guarantees about the ordering and uniqueness of these Pods.",
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
							Optional: true,
							Computed: true,
							Validators: []validator.String{
								stringvalidator.OneOf("OrderedReady", "Parallel"),
							},
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
								stringplanmodifier.RequiresReplace(),
							},
						},
						"replicas": schema.StringAttribute{
							Optional:   true,
							Computed:   true,
							Validators: []validator.String{nullableIntStringValidator{}},
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
								preserveReplicasOnEmpty{},
							},
						},
						"revision_history_limit": schema.Int64Attribute{
							Optional: true,
							Computed: true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
								int64planmodifier.RequiresReplace(),
							},
							Validators: []validator.Int64{positiveInt64Validator{}},
						},
						"service_name": schema.StringAttribute{
							Required:      true,
							PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
						},
						"min_ready_seconds": schema.Int64Attribute{
							Optional:   true,
							Computed:   true,
							Default:    int64default.StaticInt64(0),
							Validators: []validator.Int64{nonNegativeInt64Validator{}},
						},
						"persistent_volume_claim_retention_policy": persistentVolumeClaimRetentionPolicyAttribute(),
					},
					Blocks: map[string]schema.Block{
						"selector":              labelSelectorBlock(true),
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
	spec := podspec.SpecBlock(statefulSetPodTemplateOptions())
	spec.Validators = append([]validator.List{listvalidator.IsRequired(), listvalidator.SizeAtLeast(1)}, spec.Validators...)
	return schema.ListNestedBlock{
		Description: "The object that describes the pod that will be created if insufficient replicas are detected.",
		Validators:  []validator.List{listvalidator.IsRequired(), listvalidator.SizeAtLeast(1), listvalidator.SizeAtMost(1)},
		NestedObject: schema.NestedBlockObject{
			Blocks: map[string]schema.Block{
				"metadata": statefulSetTemplateMetadataBlock("stateful set"),
				"spec":     spec,
			},
		},
	}
}

func statefulSetPodTemplateOptions() podspec.Options {
	return podspec.Options{RestartPolicyAlways: false}
}

func statefulSetTemplateMetadataBlock(objectName string) schema.ListNestedBlock {
	block := schema.ListNestedBlock{
		Description: fmt.Sprintf("Standard %s template metadata.", objectName),
		Validators:  []validator.List{listvalidator.IsRequired(), listvalidator.SizeAtLeast(1), listvalidator.SizeAtMost(1)},
		NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
			"annotations": schema.MapAttribute{Optional: true, ElementType: types.StringType, Validators: []validator.Map{common.AnnotationsValidator()}},
			"labels":      schema.MapAttribute{Optional: true, ElementType: types.StringType, Validators: []validator.Map{common.LabelsValidator()}},
			"generation": schema.Int64Attribute{
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.ConflictsWith(pathMatchParent("generate_name")),
					common.DNSSubdomainNameValidator(),
				},
			},
			"generate_name": schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{common.GenerateNameRequiresReplace()},
				Validators: []validator.String{
					stringvalidator.ConflictsWith(pathMatchParent("name")),
					common.DNSLabelPrefixValidator(),
				},
			},
			"namespace": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resource_version": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"uid": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		}},
	}
	return common.WithEmptyMetadataCompatibility(block)
}

func pathMatchParent(name string) path.Expression {
	return path.MatchRelative().AtParent().AtName(name)
}

func labelSelectorBlock(required bool) schema.ListNestedBlock {
	validators := []validator.List{listvalidator.SizeAtMost(1)}
	if required {
		validators = append([]validator.List{listvalidator.IsRequired(), listvalidator.SizeAtLeast(1)}, validators...)
	}
	return schema.ListNestedBlock{
		Description:   "A label query over pods that should match the replica count.",
		Validators:    validators,
		PlanModifiers: []planmodifier.List{workloadSelectorRequiresReplace()},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"match_labels": schema.MapAttribute{Optional: true, ElementType: types.StringType},
			},
			Blocks: map[string]schema.Block{
				"match_expressions": schema.ListNestedBlock{
					NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
						"key":      schema.StringAttribute{Optional: true},
						"operator": schema.StringAttribute{Optional: true},
						"values": schema.SetAttribute{
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
		NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("RollingUpdate"),
				Validators: []validator.String{
					stringvalidator.OneOf("RollingUpdate", "OnDelete"),
				},
			},
		}, Blocks: map[string]schema.Block{
			"rolling_update": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
					"partition": schema.Int64Attribute{
						Optional: true,
						Computed: true,
						Default:  int64default.StaticInt64(0),
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

func persistentVolumeClaimRetentionPolicyAttribute() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Optional: true,
		Computed: true,
		PlanModifiers: []planmodifier.List{
			listplanmodifier.UseStateForUnknown(),
		},
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"when_deleted": schema.StringAttribute{
					Optional:   true,
					Computed:   true,
					Default:    stringdefault.StaticString("Retain"),
					Validators: []validator.String{stringvalidator.OneOf("Retain", "Delete")},
				},
				"when_scaled": schema.StringAttribute{
					Optional:   true,
					Computed:   true,
					Default:    stringdefault.StaticString("Retain"),
					Validators: []validator.String{stringvalidator.OneOf("Retain", "Delete")},
				},
			},
		},
	}
}

func persistentVolumeClaimBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		PlanModifiers: []planmodifier.List{statefulSetVolumeClaimRequiresReplace{}},
		NestedObject: schema.NestedBlockObject{
			Blocks: map[string]schema.Block{
				"metadata": common.WithEmptyMetadataCompatibility(common.NamespacedMetadataSchema("persistent volume claim", true)),
				"spec": schema.ListNestedBlock{
					Validators: []validator.List{listvalidator.IsRequired(), listvalidator.SizeAtLeast(1), listvalidator.SizeAtMost(1)},
					NestedObject: schema.NestedBlockObject{
						Attributes: map[string]schema.Attribute{
							"access_modes": schema.SetAttribute{Required: true, ElementType: types.StringType, Validators: []validator.Set{setvalidator.ValueStringsAre(stringvalidator.OneOf("ReadWriteOnce", "ReadOnlyMany", "ReadWriteMany", "ReadWriteOncePod"))}},
							"volume_name": schema.StringAttribute{
								Optional: true,
								Computed: true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
									stringplanmodifier.RequiresReplace(),
								},
							},
							"storage_class_name": schema.StringAttribute{
								Optional: true,
								Computed: true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
									stringplanmodifier.RequiresReplace(),
								},
							},
							"volume_mode": schema.StringAttribute{
								Optional: true,
								Computed: true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
									stringplanmodifier.RequiresReplace(),
								},
								Validators: []validator.String{stringvalidator.OneOf(string(corev1.PersistentVolumeBlock), string(corev1.PersistentVolumeFilesystem))},
							},
						},
						Blocks: map[string]schema.Block{
							"resources": schema.ListNestedBlock{
								Validators: []validator.List{listvalidator.IsRequired(), listvalidator.SizeAtLeast(1), listvalidator.SizeAtMost(1)},
								NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
									// The claim-level modifier handles immutable changes and legacy empty maps.
									"limits":   schema.MapAttribute{Optional: true, ElementType: types.StringType, PlanModifiers: []planmodifier.Map{statefulSetQuantityMapModifier{}}},
									"requests": schema.MapAttribute{Optional: true, ElementType: types.StringType, PlanModifiers: []planmodifier.Map{statefulSetQuantityMapModifier{}}},
								}},
							},
							"selector": labelSelectorBlock(false),
						},
					},
				},
			},
		},
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

type positiveInt64Validator struct{}

func (v positiveInt64Validator) Description(context.Context) string { return "must be greater than 0" }
func (v positiveInt64Validator) MarkdownDescription(context.Context) string {
	return v.Description(context.Background())
}
func (v positiveInt64Validator) ValidateInt64(_ context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if req.ConfigValue.ValueInt64() <= 0 {
		resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(req.Path, "must be greater than 0", req.ConfigValue.String()))
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
