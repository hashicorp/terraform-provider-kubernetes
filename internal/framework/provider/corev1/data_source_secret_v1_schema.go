// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Schema defines the Framework schema for data.kubernetes_secret_v1.
//
// The metadata block uses ListNestedBlock (not SingleNestedBlock) to preserve the
// metadata.0.* indexed path convention that matches the SDKv2 data source surface.
func (d *SecretV1DataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The resource provides mechanisms to inject containers with sensitive information, " +
			"such as passwords, while keeping containers agnostic of Kubernetes. Secrets can be used to store " +
			"sensitive information either as individual properties or coarse-grained entries like entire files " +
			"or JSON blobs.",
		Blocks: map[string]schema.Block{
			// ListNestedBlock preserves metadata.0.* indexed paths — identical to SDKv2 TypeList/MaxItems:1.
			"metadata": schema.ListNestedBlock{
				Description: "Standard object metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Name of the secret, must be unique. Cannot be updated. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
							Optional:    true,
							Computed:    true,
						},
						"namespace": schema.StringAttribute{
							Description: "Namespace defines the space within which name of the secret must be unique.",
							Optional:    true,
							Computed:    true,
						},
						"generate_name": schema.StringAttribute{
							Description: "Prefix, used by the server, to generate a unique name ONLY IF the Name field has not been provided. This value will also be combined with a unique suffix. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency",
							Optional:    true,
							Computed:    true,
						},
						"annotations": schema.MapAttribute{
							Description: "An unstructured key value map stored with the secret that may be used to store arbitrary metadata. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/",
							ElementType: types.StringType,
							Optional:    true,
							Computed:    true,
						},
						"labels": schema.MapAttribute{
							Description: "Map of string keys and values that can be used to organize and categorize (scope and select) the secret. May match selectors of replication controllers and services. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/",
							ElementType: types.StringType,
							Optional:    true,
							Computed:    true,
						},
						"generation": schema.Int64Attribute{
							Description: "A sequence number representing a specific generation of the desired state.",
							Computed:    true,
						},
						"resource_version": schema.StringAttribute{
							Description: "An opaque value that represents the internal version of this secret that can be used by clients to determine when secrets have changed. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency",
							Computed:    true,
						},
						"uid": schema.StringAttribute{
							Description: "The unique in time and space value for this secret. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids",
							Computed:    true,
						},
					},
				},
			},
		},
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The namespace/name path that identifies this secret.",
				Computed:    true,
			},
			"data": schema.MapAttribute{
				Description: "A map of the secret data.",
				ElementType: types.StringType,
				Computed:    true,
				Sensitive:   true,
			},
			"binary_data": schema.MapAttribute{
				Description: "A map of the secret data with values encoded in base64 format",
				ElementType: types.StringType,
				Optional:    true,
				Sensitive:   true,
			},
			"type": schema.StringAttribute{
				Description: "Type of secret",
				Computed:    true,
			},
			"immutable": schema.BoolAttribute{
				Description: "Ensures that data stored in the Secret cannot be updated (only object metadata can be modified).",
				Computed:    true,
			},
		},
	}
}
