package v1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (r *Secret) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `provides mechanisms to inject containers with sensitive information`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: `The unique ID for this terraform resource`,
				Optional:            true,
				Computed:            true,
			},
			"api_version": schema.StringAttribute{
				MarkdownDescription: `APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources`,
				Optional:            true,
				Computed:            true,
			},
			"data": schema.MapAttribute{
				MarkdownDescription: `Data contains the secret data. Each key must consist of alphanumeric characters, '-', '_' or '.'. The serialized form of the secret data is a base64 encoded string, representing the arbitrary (possibly non-string) data value here. Described in https://tools.ietf.org/html/rfc4648#section-4`,
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
			},
			"immutable": schema.BoolAttribute{
				MarkdownDescription: `Immutable, if set to true, ensures that data stored in the Secret cannot be updated (only object metadata can be modified). If not set to true, the field can be modified at any time. Defaulted to nil.`,
				Optional:            true,
				Computed:            true,
			},
			"kind": schema.StringAttribute{
				MarkdownDescription: `Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds`,
				Optional:            true,
				Computed:            true,
			},
			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: `Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata`,
				Optional:            true,
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"annotations": schema.MapAttribute{
						MarkdownDescription: `Annotations is an unstructured key value map stored with a resource that may be set by external tools to store and retrieve arbitrary metadata. They are not queryable and should be preserved when modifying objects. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations`,
						ElementType:         types.StringType,
						Optional:            true,
						Computed:            true,
					},
					"creation_timestamp": schema.StringAttribute{
						MarkdownDescription: `CreationTimestamp is a timestamp representing the server time when this object was created. It is not guaranteed to be set in happens-before order across separate operations. Clients may not set this value. It is represented in RFC3339 form and is in UTC.

Populated by the system. Read-only. Null for lists. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata`,
						Optional: true,
						Computed: true,
					},
					"deletion_grace_period_seconds": schema.Int64Attribute{
						MarkdownDescription: `Number of seconds allowed for this object to gracefully terminate before it will be removed from the system. Only set when deletionTimestamp is also set. May only be shortened. Read-only.`,
						Optional:            true,
						Computed:            true,
					},
					"deletion_timestamp": schema.StringAttribute{
						MarkdownDescription: `DeletionTimestamp is RFC 3339 date and time at which this resource will be deleted. This field is set by the server when a graceful deletion is requested by the user, and is not directly settable by a client. The resource is expected to be deleted (no longer visible from resource lists, and not reachable by name) after the time in this field, once the finalizers list is empty. As long as the finalizers list contains items, deletion is blocked. Once the deletionTimestamp is set, this value may not be unset or be set further into the future, although it may be shortened or the resource may be deleted prior to this time. For example, a user may request that a pod is deleted in 30 seconds. The Kubelet will react by sending a graceful termination signal to the containers in the pod. After that 30 seconds, the Kubelet will send a hard termination signal (SIGKILL) to the container and after cleanup, remove the pod from the API. In the presence of network partitions, this object may still exist after this timestamp, until an administrator or automated process can determine the resource is fully terminated. If not set, graceful deletion of the object has not been requested.

Populated by the system when a graceful deletion is requested. Read-only. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata`,
						Optional: true,
						Computed: true,
					},
					"finalizers": schema.ListAttribute{
						MarkdownDescription: `Must be empty before the object is deleted from the registry. Each entry is an identifier for the responsible component that will remove the entry from the list. If the deletionTimestamp of the object is non-nil, entries in this list can only be removed. Finalizers may be processed and removed in any order.  Order is NOT enforced because it introduces significant risk of stuck finalizers. finalizers is a shared field, any actor with permission can reorder it. If the finalizer list is processed in order, then this can lead to a situation in which the component responsible for the first finalizer in the list is waiting for a signal (field value, external system, or other) produced by a component responsible for a finalizer later in the list, resulting in a deadlock. Without enforced ordering finalizers are free to order amongst themselves and are not vulnerable to ordering changes in the list.`,
						ElementType:         types.StringType,
						Optional:            true,
						Computed:            true,
					},
					"generate_name": schema.StringAttribute{
						MarkdownDescription: `GenerateName is an optional prefix, used by the server, to generate a unique name ONLY IF the Name field has not been provided. If this field is used, the name returned to the client will be different than the name passed. This value will also be combined with a unique suffix. The provided value has the same validation rules as the Name field, and may be truncated by the length of the suffix required to make the value unique on the server.

If this field is specified and the generated name exists, the server will return a 409.

Applied only if Name is not specified. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#idempotency`,
						Optional: true,
						Computed: true,
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
						Computed:            true,
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
						Computed: true,
					},
					"owner_references": schema.ListNestedAttribute{
						MarkdownDescription: `List of objects depended by this object. If ALL objects in the list have been deleted, this object will be garbage collected. If this object is managed by a controller, then an entry in this list will point to this controller, with the controller field set to true. There cannot be more than one managing controller.`,
						Optional:            true,
						Computed:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"api_version": schema.StringAttribute{
									MarkdownDescription: `API version of the referent.`,
									Optional:            true,
									Computed:            true,
								},
								"block_owner_deletion": schema.BoolAttribute{
									MarkdownDescription: `If true, AND if the owner has the "foregroundDeletion" finalizer, then the owner cannot be deleted from the key-value store until this reference is removed. See https://kubernetes.io/docs/concepts/architecture/garbage-collection/#foreground-deletion for how the garbage collector interacts with this field and enforces the foreground deletion. Defaults to false. To set this field, a user needs "delete" permission of the owner, otherwise 422 (Unprocessable Entity) will be returned.`,
									Optional:            true,
									Computed:            true,
								},
								"controller": schema.BoolAttribute{
									MarkdownDescription: `If true, this reference points to the managing controller.`,
									Optional:            true,
									Computed:            true,
								},
								"kind": schema.StringAttribute{
									MarkdownDescription: `Kind of the referent. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds`,
									Optional:            true,
									Computed:            true,
								},
								"name": schema.StringAttribute{
									MarkdownDescription: `Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#names`,
									Optional:            true,
									Computed:            true,
								},
								"uid": schema.StringAttribute{
									MarkdownDescription: `UID of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#uids`,
									Optional:            true,
									Computed:            true,
								},
							},
						},
					},
					"resource_version": schema.StringAttribute{
						MarkdownDescription: `An opaque value that represents the internal version of this object that can be used by clients to determine when objects have changed. May be used for optimistic concurrency, change detection, and the watch operation on a resource or set of resources. Clients must treat these values as opaque and passed unmodified back to the server. They may only be valid for a particular resource or set of resources.

Populated by the system. Read-only. Value must be treated as opaque by clients and . More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency`,
						Optional: true,
						Computed: true,
					},
					"self_link": schema.StringAttribute{
						MarkdownDescription: `Deprecated: selfLink is a legacy read-only field that is no longer populated by the system.`,
						Optional:            true,
						Computed:            true,
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
				Computed:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: `Used to facilitate programmatic handling of secret data. More info: https://kubernetes.io/docs/concepts/configuration/secret/#secret-types`,
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: `name of the Secret`,
				Optional:            true,
				Computed:            true,
			},
			"namespace": schema.StringAttribute{
				MarkdownDescription: `object name and auth scope, such as for teams and projects`,
				Optional:            true,
				Computed:            true,
			},
			"pretty": schema.StringAttribute{
				MarkdownDescription: `If 'true', then the output is pretty printed.`,
				Optional:            true,
				Computed:            true,
			},
		},
	}
}
