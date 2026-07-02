// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestyaml

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

// Ensure ManifestYAML satisfies the required framework interfaces.
var (
	_ resource.Resource                = (*ManifestYAML)(nil)
	_ resource.ResourceWithConfigure   = (*ManifestYAML)(nil)
	_ resource.ResourceWithImportState = (*ManifestYAML)(nil)
	_ resource.ResourceWithModifyPlan  = (*ManifestYAML)(nil)
)

// ManifestYAML implements the raw-YAML, Server-Side-Apply resource
// (RFC-011 `kubernetes_manifest_yaml`). It applies an arbitrary Kubernetes
// object supplied as raw YAML using SSA, without requiring the typed OpenAPI
// schema at plan time (dynamic discovery is used to resolve the GVR).
type ManifestYAML struct {
	// SDKv2Meta returns the SDKv2-initialized provider meta (shared client),
	// injected at Configure time. See internal/framework/provider/provider_configure.go.
	SDKv2Meta func() any
}

// NewManifestYAML is the resource factory registered on the provider.
func NewManifestYAML() resource.Resource {
	return &ManifestYAML{}
}

func (r *ManifestYAML) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_manifest_yaml"
}

func (r *ManifestYAML) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.SDKv2Meta = req.ProviderData.(func() any)
}

// clients returns the shared Kubernetes client set from the provider meta.
func (r *ManifestYAML) clients() kubernetes.KubeClientsets {
	return r.SDKv2Meta().(kubernetes.KubeClientsets)
}

// manifestYAMLModel is the tfsdk state model. It intentionally does NOT store
// the whole server object — only the user's YAML, identity, and (future)
// owned-field projection — per ADR-006 / RFC-011 §6.
type manifestYAMLModel struct {
	YamlBody       types.String `tfsdk:"yaml_body"`
	FieldManager   types.String `tfsdk:"field_manager"`
	ForceConflicts types.Bool   `tfsdk:"force_conflicts"`
	IgnoreFields   types.List   `tfsdk:"ignore_fields"`
	ForceReplaceOn types.List   `tfsdk:"force_replace_on"`

	ID              types.String `tfsdk:"id"`
	APIVersion      types.String `tfsdk:"api_version"`
	Kind            types.String `tfsdk:"kind"`
	Name            types.String `tfsdk:"name"`
	Namespace       types.String `tfsdk:"namespace"`
	UID             types.String `tfsdk:"uid"`
	ResourceVersion types.String `tfsdk:"resource_version"`
	LiveManifest    types.String `tfsdk:"live_manifest"`
	Status          types.String `tfsdk:"status"`

	Wait     *waitModel     `tfsdk:"wait"`
	Delete   *deleteModel   `tfsdk:"delete"`
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

type waitModel struct {
	Rollout   types.Bool   `tfsdk:"rollout"`
	Condition types.String `tfsdk:"condition"`
	Fields    types.Map    `tfsdk:"fields"`
	Timeout   types.String `tfsdk:"timeout"`
}

// hasAny reports whether any wait condition is configured.
func (w *waitModel) hasAny() bool {
	if w == nil {
		return false
	}
	return w.Rollout.ValueBool() ||
		(!w.Condition.IsNull() && w.Condition.ValueString() != "") ||
		(!w.Fields.IsNull() && !w.Fields.IsUnknown() && len(w.Fields.Elements()) > 0)
}

type deleteModel struct {
	PropagationPolicy  types.String `tfsdk:"propagation_policy"`
	GracePeriodSeconds types.Int64  `tfsdk:"grace_period_seconds"`
}
