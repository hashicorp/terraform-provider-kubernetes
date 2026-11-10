// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"encoding/base64"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

var _ validator.Map = base64MapValidator{}

// Schema implements [resource.Resource].
func (c *ConfigMapV1) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The resource provides mechanisms to inject containers with configuration data while keeping containers agnostic of Kubernetes. Config Map can be used to store fine-grained information like individual properties or coarse-grained information like entire config files or JSON blobs.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"binary_data": schema.MapAttribute{
				Description: "BinaryData contains the binary data. Each key must consist of alphanumeric characters, '-', '_' or '.'. BinaryData can contain byte sequences that are not in the UTF-8 range. The keys stored in BinaryData must not overlap with the ones in the Data field, this is enforced during validation process. Using this field will require 1.10+ apiserver and kubelet. This field only accepts base64-encoded payloads that will be decoded/encoded before being sent/received to/from the apiserver.",
				ElementType: types.StringType,
				Optional:    true,
				Validators: []validator.Map{
					mapvalidator.NoNullValues(),
					base64MapValidator{},
				},
			},
			"data": schema.MapAttribute{
				Description: "Data contains the configuration data. Each key must consist of alphanumeric characters, '-', '_' or '.'. Values with non-UTF-8 byte sequences must use the BinaryData field. The keys stored in Data must not overlap with the keys in the BinaryData field, this is enforced during validation process.",
				ElementType: types.StringType,
				Optional:    true,
				Validators:  []validator.Map{mapvalidator.NoNullValues()},
			},
			"immutable": schema.BoolAttribute{
				Description: "Immutable, if set to true, ensures that data stored in the ConfigMap cannot be updated (only object metadata can be modified). If not set to true, the field can be modified at any time. Defaulted to nil.",
				Optional:    true,
				// Optional+Computed+Default is how the framework spells SDKv2's behaviour,
				// where the state of an unset bool is false and Create always sent
				// ptr(false). All three are required: Default needs Computed.
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": common.NamespacedMetadataSchema("config map", true),
		},
	}
}

// IdentitySchema implements [resource.ResourceWithIdentity].
//
// common.NamespacedIdentitySchema reproduces resourceIdentitySchemaNamespaced() from
// kubernetes/resourceidentity.go, including Version 1 and namespace being OptionalForImport.
func (c *ConfigMapV1) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = common.NamespacedIdentitySchema()
}

// base64MapValidator ports validateBase64EncodedMap from kubernetes/validators.go: every
// value must be standard base64. The offending value is deliberately not echoed, since
// binary data can be sensitive.
type base64MapValidator struct{}

func (base64MapValidator) Description(ctx context.Context) string {
	return base64MapValidator{}.MarkdownDescription(ctx)
}

func (base64MapValidator) MarkdownDescription(_ context.Context) string {
	return "values must be base64-encoded strings"
}

func (base64MapValidator) ValidateMap(_ context.Context, req validator.MapRequest, resp *validator.MapResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	for k, raw := range req.ConfigValue.Elements() {
		val, ok := raw.(types.String)
		if !ok || val.IsNull() || val.IsUnknown() {
			continue
		}
		if _, err := base64.StdEncoding.DecodeString(val.ValueString()); err != nil {
			resp.Diagnostics.Append(diag.NewAttributeErrorDiagnostic(
				req.Path.AtMapKey(k),
				"Invalid Attribute Value",
				"Attribute "+req.Path.AtMapKey(k).String()+" must be a base64-encoded string.",
			))
		}
	}
}
