// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package nodev1

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

// runtimeClassHandlerRegexp is SDKv2's handler pattern, copied verbatim. It is anchored at
// the start only, so it is a prefix match: "runc_v2" and "a-" are accepted. Anchoring it
// would reject handlers that every released provider version accepts.
var runtimeClassHandlerRegexp = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?`)

// Schema implements [resource.Resource].
func (r *RuntimeClassV1) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A runtime class is used to determine which container runtime is used to run all containers in a pod.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"handler": schema.StringAttribute{
				Description: "Specifies the underlying runtime and configuration that the CRI implementation will use to handle pods of this class",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(runtimeClassHandlerRegexp, ""),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": common.MetadataSchema("runtimeclass", true),
		},
	}
}
