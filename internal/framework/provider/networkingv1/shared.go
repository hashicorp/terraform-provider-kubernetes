// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package networkingv1

import (
	"context"
	"fmt"
	"maps"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclient "k8s.io/client-go/kubernetes"
)

func networkingClient(meta func() any) (kubeclient.Interface, kubernetes.MetadataFilters, diag.Diagnostics) {
	var diags diag.Diagnostics
	if meta == nil {
		diags.AddError("Provider not configured", "The Kubernetes provider metadata is unavailable.")
		return nil, nil, diags
	}
	value := meta()
	clients, ok := value.(kubernetes.KubeClientsets)
	if !ok {
		diags.AddError("Unexpected provider data", fmt.Sprintf("Expected kubernetes.KubeClientsets, got %T.", value))
		return nil, nil, diags
	}
	filters, ok := value.(kubernetes.MetadataFilters)
	if !ok {
		diags.AddError("Unexpected provider data", fmt.Sprintf("Expected kubernetes.MetadataFilters, got %T.", value))
		return nil, nil, diags
	}
	client, err := clients.MainClientset()
	if err != nil {
		diags.AddError("Kubernetes client error", err.Error())
		return nil, nil, diags
	}
	if client == nil {
		diags.AddError("Kubernetes client error", "The Kubernetes client is unavailable.")
		return nil, nil, diags
	}
	return client, filters, diags
}

func networkingMetadataSchema(objectName string, namespaced bool) schema.ListNestedBlock {
	block := common.MetadataSchema(objectName, true)
	if namespaced {
		block = common.NamespacedMetadataSchema(objectName, true)
	}
	// Preserve SDKv2's stored zero value, including refresh-disabled upgrade plans.
	generateName := block.NestedObject.Attributes["generate_name"].(schema.StringAttribute)
	generateName.Computed = true
	generateName.Default = stringdefault.StaticString("")
	generateName.PlanModifiers = []planmodifier.String{stringplanmodifier.RequiresReplace()}
	block.NestedObject.Attributes["generate_name"] = generateName
	for _, name := range []string{"annotations", "labels"} {
		attribute := block.NestedObject.Attributes[name].(schema.MapAttribute)
		attribute.Computed = true
		attribute.PlanModifiers = []planmodifier.Map{networkingEmptyCollectionPlanModifier{}}
		block.NestedObject.Attributes[name] = attribute
	}
	return block
}

// SDKv2 can persist an omitted collection as null on create and empty after a
// later refresh. Computed ownership permits retaining that empty state without
// changing outputs, while removal of previously nonempty configuration still clears it.
type networkingEmptyCollectionPlanModifier struct{}

func (networkingEmptyCollectionPlanModifier) Description(context.Context) string {
	return "Preserve historical empty collection state when omitted; clear nonempty values on removal."
}

func (m networkingEmptyCollectionPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (networkingEmptyCollectionPlanModifier) PlanModifyMap(_ context.Context, req planmodifier.MapRequest, resp *planmodifier.MapResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}
	resp.PlanValue = types.MapNull(types.StringType)
	if !req.StateValue.IsNull() && !req.StateValue.IsUnknown() && len(req.StateValue.Elements()) == 0 {
		resp.PlanValue = req.StateValue
	}
}

func (networkingEmptyCollectionPlanModifier) PlanModifyList(_ context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}
	resp.PlanValue = types.ListNull(types.StringType)
	if !req.StateValue.IsNull() && !req.StateValue.IsUnknown() && len(req.StateValue.Elements()) == 0 {
		resp.PlanValue = req.StateValue
	}
}

func (networkingEmptyCollectionPlanModifier) PlanModifySet(_ context.Context, req planmodifier.SetRequest, resp *planmodifier.SetResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}
	resp.PlanValue = types.SetNull(types.StringType)
	if !req.StateValue.IsNull() && !req.StateValue.IsUnknown() && len(req.StateValue.Elements()) == 0 {
		resp.PlanValue = req.StateValue
	}
}

func flattenNetworkingMetadata(ctx context.Context, meta metav1.ObjectMeta, prior []common.MetadataModel, ignoreAnnotations, ignoreLabels []string) ([]common.MetadataModel, diag.Diagnostics) {
	result, diags := common.FlattenMetadata(ctx, meta, prior, ignoreAnnotations, ignoreLabels)
	if !diags.HasError() {
		result[0].GenerateName = types.StringValue(meta.GenerateName)
	}
	return result, diags
}

func flattenNetworkingNamespacedMetadata(ctx context.Context, meta metav1.ObjectMeta, prior []common.NamespacedMetadataModel, ignoreAnnotations, ignoreLabels []string) ([]common.NamespacedMetadataModel, diag.Diagnostics) {
	result, diags := common.FlattenNamespacedMetadata(ctx, meta, prior, ignoreAnnotations, ignoreLabels)
	if !diags.HasError() {
		result[0].GenerateName = types.StringValue(meta.GenerateName)
	}
	return result, diags
}

func networkingMetadataPatch(ctx context.Context, live metav1.ObjectMeta, prior, plan common.MetadataBase) (kubernetes.PatchOperations, diag.Diagnostics) {
	var diags diag.Diagnostics
	ops := make(kubernetes.PatchOperations, 0)
	for _, field := range []struct {
		name  string
		live  map[string]string
		prior types.Map
		plan  types.Map
	}{
		{"annotations", live.Annotations, prior.Annotations, plan.Annotations},
		{"labels", live.Labels, prior.Labels, plan.Labels},
	} {
		oldManaged, newManaged := map[string]string{}, map[string]string{}
		if !field.prior.IsNull() {
			diags.Append(field.prior.ElementsAs(ctx, &oldManaged, false)...)
		}
		if !field.plan.IsNull() {
			diags.Append(field.plan.ElementsAs(ctx, &newManaged, false)...)
		}
		if diags.HasError() {
			return nil, diags
		}
		// Diff the full live map against a copy with only Terraform-owned keys changed.
		// This also preserves external keys when adding the first managed key.
		desired := maps.Clone(field.live)
		if desired == nil {
			desired = map[string]string{}
		}
		for key := range oldManaged {
			delete(desired, key)
		}
		maps.Copy(desired, newManaged)
		if maps.Equal(field.live, desired) {
			continue
		}
		oldValue, newValue := map[string]interface{}{}, map[string]interface{}{}
		for key, value := range field.live {
			oldValue[key] = value
		}
		for key, value := range desired {
			newValue[key] = value
		}
		ops = append(ops, kubernetes.DiffStringMap("/metadata/"+field.name, oldValue, newValue)...)
	}
	return ops, diags
}
