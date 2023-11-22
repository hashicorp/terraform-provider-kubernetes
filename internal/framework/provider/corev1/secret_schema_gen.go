package corev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (r *Secret) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `configmaps store information for pods`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: `The unique ID for this terraform resource`,
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("str"),
			},
			"data": schema.MapAttribute{
				MarkdownDescription: `Data contains the secret data. Each key must consist of alphanumeric characters, '-', '_' or '.'. The serialized form of the secret data is a base64 encoded string, representing the arbitrary (possibly non-string) data value here. Described in https://tools.ietf.org/html/rfc4648#section-4`,
				ElementType:         types.StringType,
				Optional:            true,
			},
			"immutable": schema.BoolAttribute{
				MarkdownDescription: `Immutable, if set to true, ensures that data stored in the Secret cannot be updated (only object metadata can be modified). If not set to true, the field can be modified at any time. Defaulted to nil.`,
				Optional:            true,
			},
			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: `Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata`,
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"annotations": schema.MapAttribute{
						MarkdownDescription: `Annotations is an unstructured key value map stored with a resource that may be set by external tools to store and retrieve arbitrary metadata. They are not queryable and should be preserved when modifying objects. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations`,
						ElementType:         types.StringType,
						Optional:            true,
					},
					"generate_name": schema.StringAttribute{
						MarkdownDescription: `GenerateName is an optional prefix, used by the server, to generate a unique name ONLY IF the Name field has not been provided. If this field is used, the name returned to the client will be different than the name passed. This value will also be combined with a unique suffix. The provided value has the same validation rules as the Name field, and may be truncated by the length of the suffix required to make the value unique on the server.

If this field is specified and the generated name exists, the server will return a 409.

Applied only if Name is not specified. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#idempotency`,
						Optional: true,
					},
					"generation": schema.Int64Attribute{
						MarkdownDescription: `A sequence number representing a specific generation of the desired state. Populated by the system. Read-only.`,
						Optional:            true,
						Computed:            true,
					},
					"labels": schema.MapAttribute{
						MarkdownDescription: `Map of string keys and values that can be used to organize and categorize (scope and select) objects. May match selectors of replication controllers and services. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels`,
						ElementType:         types.StringType,
						Optional:            true,
					},
					"name": schema.StringAttribute{
						MarkdownDescription: `Name must be unique within a namespace. Is required when creating resources, although some resources may allow a client to request the generation of an appropriate name automatically. Name is primarily intended for creation idempotence and configuration definition. Cannot be updated. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#names`,
						Optional:            true,
						Computed:            true,
					},
					"namespace": schema.StringAttribute{
						MarkdownDescription: `Namespace defines the space within which each name must be unique. An empty namespace is equivalent to the "default" namespace, but "default" is the canonical representation. Not all objects are required to be scoped to a namespace - the value of this field for those objects will be empty.

Must be a DNS_LABEL. Cannot be updated. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces`,
						Optional: true,
					},
					"resource_version": schema.StringAttribute{
						MarkdownDescription: `An opaque value that represents the internal version of this object that can be used by clients to determine when objects have changed. May be used for optimistic concurrency, change detection, and the watch operation on a resource or set of resources. Clients must treat these values as opaque and passed unmodified back to the server. They may only be valid for a particular resource or set of resources.

Populated by the system. Read-only. Value must be treated as opaque by clients and . More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency`,
						Optional: true,
						Computed: true,
					},
					"uid": schema.StringAttribute{
						MarkdownDescription: `UID is the unique in time and space value for this object. It is typically generated by the server on successful creation of a resource and is not allowed to change on PUT operations.

Populated by the system. Read-only. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#uids`,
						Optional: true,
						Computed: true,
					},
				},
			},
			"string_data": schema.MapAttribute{
				MarkdownDescription: `stringData allows specifying non-binary secret data in string form. It is provided as a write-only input field for convenience. All keys and values are merged into the data field on write, overwriting any existing values. The stringData field is never output when reading from the API.`,
				ElementType:         types.StringType,
				Optional:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: `Used to facilitate programmatic handling of secret data. More info: https://kubernetes.io/docs/concepts/configuration/secret/#secret-types`,
				Optional:            true,
				Computed:            true,
			},
		},
	}
}
