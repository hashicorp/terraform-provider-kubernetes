package appsv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (r *DaemonSet) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `damonset`,
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: `The unique ID for this terraform resource`,
				Optional:            true,
				Computed:            true,
			},
			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: `Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata`,
				Required:            true,

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

						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"namespace": schema.StringAttribute{
						MarkdownDescription: `Namespace defines the space within which each name must be unique. An empty namespace is equivalent to the "default" namespace, but "default" is the canonical representation. Not all objects are required to be scoped to a namespace - the value of this field for those objects will be empty.

Must be a DNS_LABEL. Cannot be updated. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces`,
						Optional: true,
						Computed: true,

						Default: stringdefault.StaticString("default"),
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
			"spec": schema.SingleNestedAttribute{
				MarkdownDescription: `The desired behavior of this daemon set. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status`,
				Required:            true,

				Attributes: map[string]schema.Attribute{
					"min_ready_seconds": schema.Int64Attribute{
						MarkdownDescription: `The minimum number of seconds for which a newly created DaemonSet pod should be ready without any of its container crashing, for it to be considered available. Defaults to 0 (pod will be considered available as soon as it is ready).`,
						Optional:            true,
					},
					"revision_history_limit": schema.Int64Attribute{
						MarkdownDescription: `The number of old history to retain to allow rollback. This is a pointer to distinguish between explicit zero and not specified. Defaults to 10.`,
						Optional:            true,
					},
					"selector": schema.SingleNestedAttribute{
						MarkdownDescription: `A label query over pods that are managed by the daemon set. Must match in order to be controlled. It must match the pod template's labels. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors`,
						Optional:            true,

						Attributes: map[string]schema.Attribute{
							"match_expressions": schema.ListNestedAttribute{
								MarkdownDescription: `matchExpressions is a list of label selector requirements. The requirements are ANDed.`,
								Optional:            true,

								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"key": schema.StringAttribute{
											MarkdownDescription: `key is the label key that the selector applies to.`,
											Optional:            true,
										},
										"operator": schema.StringAttribute{
											MarkdownDescription: `operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.`,
											Optional:            true,
										},
										"values": schema.ListAttribute{
											MarkdownDescription: `values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.`,
											ElementType:         types.StringType,
											Optional:            true,
										},
									},
								},
							},
							"match_labels": schema.MapAttribute{
								MarkdownDescription: `matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.`,
								ElementType:         types.StringType,
								Optional:            true,
							},
						},
					},
					"template": schema.SingleNestedAttribute{
						MarkdownDescription: `An object that describes the pod that will be created. The DaemonSet will create exactly one copy of this pod on every node that matches the template's node selector (or on every node if no node selector is specified). The only allowed template.spec.restartPolicy value is "Always". More info: https://kubernetes.io/docs/concepts/workloads/controllers/replicationcontroller#pod-template`,
						Optional:            true,

						Attributes: map[string]schema.Attribute{
							"metadata": schema.SingleNestedAttribute{
								MarkdownDescription: `Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata`,
								Optional:            true,

								Attributes: map[string]schema.Attribute{
									"annotations": schema.MapAttribute{
										MarkdownDescription: `Annotations is an unstructured key value map stored with a resource that may be set by external tools to store and retrieve arbitrary metadata. They are not queryable and should be preserved when modifying objects. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations`,
										ElementType:         types.StringType,
										Optional:            true,
									},
									"creation_timestamp": schema.StringAttribute{
										MarkdownDescription: `CreationTimestamp is a timestamp representing the server time when this object was created. It is not guaranteed to be set in happens-before order across separate operations. Clients may not set this value. It is represented in RFC3339 form and is in UTC.

Populated by the system. Read-only. Null for lists. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata`,
										Optional: true,
									},
									"deletion_grace_period_seconds": schema.Int64Attribute{
										MarkdownDescription: `Number of seconds allowed for this object to gracefully terminate before it will be removed from the system. Only set when deletionTimestamp is also set. May only be shortened. Read-only.`,
										Optional:            true,
									},
									"deletion_timestamp": schema.StringAttribute{
										MarkdownDescription: `DeletionTimestamp is RFC 3339 date and time at which this resource will be deleted. This field is set by the server when a graceful deletion is requested by the user, and is not directly settable by a client. The resource is expected to be deleted (no longer visible from resource lists, and not reachable by name) after the time in this field, once the finalizers list is empty. As long as the finalizers list contains items, deletion is blocked. Once the deletionTimestamp is set, this value may not be unset or be set further into the future, although it may be shortened or the resource may be deleted prior to this time. For example, a user may request that a pod is deleted in 30 seconds. The Kubelet will react by sending a graceful termination signal to the containers in the pod. After that 30 seconds, the Kubelet will send a hard termination signal (SIGKILL) to the container and after cleanup, remove the pod from the API. In the presence of network partitions, this object may still exist after this timestamp, until an administrator or automated process can determine the resource is fully terminated. If not set, graceful deletion of the object has not been requested.

Populated by the system when a graceful deletion is requested. Read-only. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata`,
										Optional: true,
									},
									"finalizers": schema.ListAttribute{
										MarkdownDescription: `Must be empty before the object is deleted from the registry. Each entry is an identifier for the responsible component that will remove the entry from the list. If the deletionTimestamp of the object is non-nil, entries in this list can only be removed. Finalizers may be processed and removed in any order.  Order is NOT enforced because it introduces significant risk of stuck finalizers. finalizers is a shared field, any actor with permission can reorder it. If the finalizer list is processed in order, then this can lead to a situation in which the component responsible for the first finalizer in the list is waiting for a signal (field value, external system, or other) produced by a component responsible for a finalizer later in the list, resulting in a deadlock. Without enforced ordering finalizers are free to order amongst themselves and are not vulnerable to ordering changes in the list.`,
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
									},
									"labels": schema.MapAttribute{
										MarkdownDescription: `Map of string keys and values that can be used to organize and categorize (scope and select) objects. May match selectors of replication controllers and services. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels`,
										ElementType:         types.StringType,
										Optional:            true,
									},
									"managed_fields": schema.ListNestedAttribute{
										MarkdownDescription: `ManagedFields maps workflow-id and version to the set of fields that are managed by that workflow. This is mostly for internal housekeeping, and users typically shouldn't need to set or understand this field. A workflow can be the user's name, a controller's name, or the name of a specific apply path like "ci-cd". The set of fields is always in the version that the workflow used when modifying the object.`,
										Optional:            true,

										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"api_version": schema.StringAttribute{
													MarkdownDescription: `APIVersion defines the version of this resource that this field set applies to. The format is "group/version" just like the top-level APIVersion field. It is necessary to track the version of a field set because it cannot be automatically converted.`,
													Optional:            true,
												},
												"fields_type": schema.StringAttribute{
													MarkdownDescription: `FieldsType is the discriminator for the different fields format and version. There is currently only one possible value: "FieldsV1"`,
													Optional:            true,
												},
												"fields_v1": schema.SingleNestedAttribute{
													MarkdownDescription: `FieldsV1 holds the first JSON version format as described in the "FieldsV1" type.`,
													Optional:            true,
												},
												"manager": schema.StringAttribute{
													MarkdownDescription: `Manager is an identifier of the workflow managing these fields.`,
													Optional:            true,
												},
												"operation": schema.StringAttribute{
													MarkdownDescription: `Operation is the type of operation which lead to this ManagedFieldsEntry being created. The only valid values for this field are 'Apply' and 'Update'.`,
													Optional:            true,
												},
												"subresource": schema.StringAttribute{
													MarkdownDescription: `Subresource is the name of the subresource used to update that object, or empty string if the object was updated through the main resource. The value of this field is used to distinguish between managers, even if they share the same name. For example, a status update will be distinct from a regular update using the same manager name. Note that the APIVersion field is not related to the Subresource field and it always corresponds to the version of the main resource.`,
													Optional:            true,
												},
												"time": schema.StringAttribute{
													MarkdownDescription: `Time is the timestamp of when the ManagedFields entry was added. The timestamp will also be updated if a field is added, the manager changes any of the owned fields value or removes a field. The timestamp does not update when a field is removed from the entry because another manager took it over.`,
													Optional:            true,
												},
											},
										},
									},
									"name": schema.StringAttribute{
										MarkdownDescription: `Name must be unique within a namespace. Is required when creating resources, although some resources may allow a client to request the generation of an appropriate name automatically. Name is primarily intended for creation idempotence and configuration definition. Cannot be updated. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#names`,
										Optional:            true,
									},
									"namespace": schema.StringAttribute{
										MarkdownDescription: `Namespace defines the space within which each name must be unique. An empty namespace is equivalent to the "default" namespace, but "default" is the canonical representation. Not all objects are required to be scoped to a namespace - the value of this field for those objects will be empty.

Must be a DNS_LABEL. Cannot be updated. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces`,
										Optional: true,
									},
									"owner_references": schema.ListNestedAttribute{
										MarkdownDescription: `List of objects depended by this object. If ALL objects in the list have been deleted, this object will be garbage collected. If this object is managed by a controller, then an entry in this list will point to this controller, with the controller field set to true. There cannot be more than one managing controller.`,
										Optional:            true,

										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"api_version": schema.StringAttribute{
													MarkdownDescription: `API version of the referent.`,
													Optional:            true,
												},
												"block_owner_deletion": schema.BoolAttribute{
													MarkdownDescription: `If true, AND if the owner has the "foregroundDeletion" finalizer, then the owner cannot be deleted from the key-value store until this reference is removed. See https://kubernetes.io/docs/concepts/architecture/garbage-collection/#foreground-deletion for how the garbage collector interacts with this field and enforces the foreground deletion. Defaults to false. To set this field, a user needs "delete" permission of the owner, otherwise 422 (Unprocessable Entity) will be returned.`,
													Optional:            true,
												},
												"controller": schema.BoolAttribute{
													MarkdownDescription: `If true, this reference points to the managing controller.`,
													Optional:            true,
												},
												"kind": schema.StringAttribute{
													MarkdownDescription: `Kind of the referent. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds`,
													Optional:            true,
												},
												"name": schema.StringAttribute{
													MarkdownDescription: `Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#names`,
													Optional:            true,
												},
												"uid": schema.StringAttribute{
													MarkdownDescription: `UID of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#uids`,
													Optional:            true,
												},
											},
										},
									},
									"resource_version": schema.StringAttribute{
										MarkdownDescription: `An opaque value that represents the internal version of this object that can be used by clients to determine when objects have changed. May be used for optimistic concurrency, change detection, and the watch operation on a resource or set of resources. Clients must treat these values as opaque and passed unmodified back to the server. They may only be valid for a particular resource or set of resources.

Populated by the system. Read-only. Value must be treated as opaque by clients and . More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency`,
										Optional: true,
									},
									"self_link": schema.StringAttribute{
										MarkdownDescription: `Deprecated: selfLink is a legacy read-only field that is no longer populated by the system.`,
										Optional:            true,
									},
									"uid": schema.StringAttribute{
										MarkdownDescription: `UID is the unique in time and space value for this object. It is typically generated by the server on successful creation of a resource and is not allowed to change on PUT operations.

Populated by the system. Read-only. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#uids`,
										Optional: true,
									},
								},
							},
							"spec": schema.SingleNestedAttribute{
								MarkdownDescription: `Specification of the desired behavior of the pod. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status`,
								Optional:            true,

								Attributes: map[string]schema.Attribute{
									"active_deadline_seconds": schema.Int64Attribute{
										MarkdownDescription: `Optional duration in seconds the pod may be active on the node relative to StartTime before the system will actively try to mark it failed and kill associated containers. Value must be a positive integer.`,
										Optional:            true,
									},
									"affinity": schema.SingleNestedAttribute{
										MarkdownDescription: `If specified, the pod's scheduling constraints`,
										Optional:            true,

										Attributes: map[string]schema.Attribute{
											"node_affinity": schema.SingleNestedAttribute{
												MarkdownDescription: `Describes node affinity scheduling rules for the pod.`,
												Optional:            true,

												Attributes: map[string]schema.Attribute{
													"preferred_during_scheduling_ignored_during_execution": schema.ListNestedAttribute{
														MarkdownDescription: `The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node matches the corresponding matchExpressions; the node(s) with the highest sum are the most preferred.`,
														Optional:            true,

														NestedObject: schema.NestedAttributeObject{
															Attributes: map[string]schema.Attribute{
																"preference": schema.SingleNestedAttribute{
																	MarkdownDescription: `A node selector term, associated with the corresponding weight.`,
																	Optional:            true,

																	Attributes: map[string]schema.Attribute{
																		"match_expressions": schema.ListNestedAttribute{
																			MarkdownDescription: `A list of node selector requirements by node's labels.`,
																			Optional:            true,

																			NestedObject: schema.NestedAttributeObject{
																				Attributes: map[string]schema.Attribute{
																					"key": schema.StringAttribute{
																						MarkdownDescription: `The label key that the selector applies to.`,
																						Optional:            true,
																					},
																					"operator": schema.StringAttribute{
																						MarkdownDescription: `Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.`,
																						Optional:            true,
																					},
																					"values": schema.ListAttribute{
																						MarkdownDescription: `An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.`,
																						ElementType:         types.StringType,
																						Optional:            true,
																					},
																				},
																			},
																		},
																		"match_fields": schema.ListNestedAttribute{
																			MarkdownDescription: `A list of node selector requirements by node's fields.`,
																			Optional:            true,

																			NestedObject: schema.NestedAttributeObject{
																				Attributes: map[string]schema.Attribute{
																					"key": schema.StringAttribute{
																						MarkdownDescription: `The label key that the selector applies to.`,
																						Optional:            true,
																					},
																					"operator": schema.StringAttribute{
																						MarkdownDescription: `Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.`,
																						Optional:            true,
																					},
																					"values": schema.ListAttribute{
																						MarkdownDescription: `An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.`,
																						ElementType:         types.StringType,
																						Optional:            true,
																					},
																				},
																			},
																		},
																	},
																},
																"weight": schema.Int64Attribute{
																	MarkdownDescription: `Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100.`,
																	Optional:            true,
																},
															},
														},
													},
													"required_during_scheduling_ignored_during_execution": schema.SingleNestedAttribute{
														MarkdownDescription: `If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to an update), the system may or may not try to eventually evict the pod from its node.`,
														Optional:            true,

														Attributes: map[string]schema.Attribute{
															"node_selector_terms": schema.ListNestedAttribute{
																MarkdownDescription: `Required. A list of node selector terms. The terms are ORed.`,
																Optional:            true,

																NestedObject: schema.NestedAttributeObject{
																	Attributes: map[string]schema.Attribute{
																		"match_expressions": schema.ListNestedAttribute{
																			MarkdownDescription: `A list of node selector requirements by node's labels.`,
																			Optional:            true,

																			NestedObject: schema.NestedAttributeObject{
																				Attributes: map[string]schema.Attribute{
																					"key": schema.StringAttribute{
																						MarkdownDescription: `The label key that the selector applies to.`,
																						Optional:            true,
																					},
																					"operator": schema.StringAttribute{
																						MarkdownDescription: `Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.`,
																						Optional:            true,
																					},
																					"values": schema.ListAttribute{
																						MarkdownDescription: `An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.`,
																						ElementType:         types.StringType,
																						Optional:            true,
																					},
																				},
																			},
																		},
																		"match_fields": schema.ListNestedAttribute{
																			MarkdownDescription: `A list of node selector requirements by node's fields.`,
																			Optional:            true,

																			NestedObject: schema.NestedAttributeObject{
																				Attributes: map[string]schema.Attribute{
																					"key": schema.StringAttribute{
																						MarkdownDescription: `The label key that the selector applies to.`,
																						Optional:            true,
																					},
																					"operator": schema.StringAttribute{
																						MarkdownDescription: `Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.`,
																						Optional:            true,
																					},
																					"values": schema.ListAttribute{
																						MarkdownDescription: `An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.`,
																						ElementType:         types.StringType,
																						Optional:            true,
																					},
																				},
																			},
																		},
																	},
																},
															},
														},
													},
												},
											},
											"pod_affinity": schema.SingleNestedAttribute{
												MarkdownDescription: `Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).`,
												Optional:            true,

												Attributes: map[string]schema.Attribute{
													"preferred_during_scheduling_ignored_during_execution": schema.ListNestedAttribute{
														MarkdownDescription: `The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred.`,
														Optional:            true,

														NestedObject: schema.NestedAttributeObject{
															Attributes: map[string]schema.Attribute{
																"pod_affinity_term": schema.SingleNestedAttribute{
																	MarkdownDescription: `Required. A pod affinity term, associated with the corresponding weight.`,
																	Optional:            true,

																	Attributes: map[string]schema.Attribute{
																		"label_selector": schema.SingleNestedAttribute{
																			MarkdownDescription: `A label query over a set of resources, in this case pods.`,
																			Optional:            true,

																			Attributes: map[string]schema.Attribute{
																				"match_expressions": schema.ListNestedAttribute{
																					MarkdownDescription: `matchExpressions is a list of label selector requirements. The requirements are ANDed.`,
																					Optional:            true,

																					NestedObject: schema.NestedAttributeObject{
																						Attributes: map[string]schema.Attribute{
																							"key": schema.StringAttribute{
																								MarkdownDescription: `key is the label key that the selector applies to.`,
																								Optional:            true,
																							},
																							"operator": schema.StringAttribute{
																								MarkdownDescription: `operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.`,
																								Optional:            true,
																							},
																							"values": schema.ListAttribute{
																								MarkdownDescription: `values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.`,
																								ElementType:         types.StringType,
																								Optional:            true,
																							},
																						},
																					},
																				},
																				"match_labels": schema.MapAttribute{
																					MarkdownDescription: `matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.`,
																					ElementType:         types.StringType,
																					Optional:            true,
																				},
																			},
																		},
																		"namespace_selector": schema.SingleNestedAttribute{
																			MarkdownDescription: `A label query over the set of namespaces that the term applies to. The term is applied to the union of the namespaces selected by this field and the ones listed in the namespaces field. null selector and null or empty namespaces list means "this pod's namespace". An empty selector ({}) matches all namespaces.`,
																			Optional:            true,

																			Attributes: map[string]schema.Attribute{
																				"match_expressions": schema.ListNestedAttribute{
																					MarkdownDescription: `matchExpressions is a list of label selector requirements. The requirements are ANDed.`,
																					Optional:            true,

																					NestedObject: schema.NestedAttributeObject{
																						Attributes: map[string]schema.Attribute{
																							"key": schema.StringAttribute{
																								MarkdownDescription: `key is the label key that the selector applies to.`,
																								Optional:            true,
																							},
																							"operator": schema.StringAttribute{
																								MarkdownDescription: `operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.`,
																								Optional:            true,
																							},
																							"values": schema.ListAttribute{
																								MarkdownDescription: `values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.`,
																								ElementType:         types.StringType,
																								Optional:            true,
																							},
																						},
																					},
																				},
																				"match_labels": schema.MapAttribute{
																					MarkdownDescription: `matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.`,
																					ElementType:         types.StringType,
																					Optional:            true,
																				},
																			},
																		},
																		"namespaces": schema.ListAttribute{
																			MarkdownDescription: `namespaces specifies a static list of namespace names that the term applies to. The term is applied to the union of the namespaces listed in this field and the ones selected by namespaceSelector. null or empty namespaces list and null namespaceSelector means "this pod's namespace".`,
																			ElementType:         types.StringType,
																			Optional:            true,
																		},
																		"topology_key": schema.StringAttribute{
																			MarkdownDescription: `This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.`,
																			Optional:            true,
																		},
																	},
																},
																"weight": schema.Int64Attribute{
																	MarkdownDescription: `weight associated with matching the corresponding podAffinityTerm, in the range 1-100.`,
																	Optional:            true,
																},
															},
														},
													},
													"required_during_scheduling_ignored_during_execution": schema.ListNestedAttribute{
														MarkdownDescription: `If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied.`,
														Optional:            true,

														NestedObject: schema.NestedAttributeObject{
															Attributes: map[string]schema.Attribute{
																"label_selector": schema.SingleNestedAttribute{
																	MarkdownDescription: `A label query over a set of resources, in this case pods.`,
																	Optional:            true,

																	Attributes: map[string]schema.Attribute{
																		"match_expressions": schema.ListNestedAttribute{
																			MarkdownDescription: `matchExpressions is a list of label selector requirements. The requirements are ANDed.`,
																			Optional:            true,

																			NestedObject: schema.NestedAttributeObject{
																				Attributes: map[string]schema.Attribute{
																					"key": schema.StringAttribute{
																						MarkdownDescription: `key is the label key that the selector applies to.`,
																						Optional:            true,
																					},
																					"operator": schema.StringAttribute{
																						MarkdownDescription: `operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.`,
																						Optional:            true,
																					},
																					"values": schema.ListAttribute{
																						MarkdownDescription: `values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.`,
																						ElementType:         types.StringType,
																						Optional:            true,
																					},
																				},
																			},
																		},
																		"match_labels": schema.MapAttribute{
																			MarkdownDescription: `matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.`,
																			ElementType:         types.StringType,
																			Optional:            true,
																		},
																	},
																},
																"namespace_selector": schema.SingleNestedAttribute{
																	MarkdownDescription: `A label query over the set of namespaces that the term applies to. The term is applied to the union of the namespaces selected by this field and the ones listed in the namespaces field. null selector and null or empty namespaces list means "this pod's namespace". An empty selector ({}) matches all namespaces.`,
																	Optional:            true,

																	Attributes: map[string]schema.Attribute{
																		"match_expressions": schema.ListNestedAttribute{
																			MarkdownDescription: `matchExpressions is a list of label selector requirements. The requirements are ANDed.`,
																			Optional:            true,

																			NestedObject: schema.NestedAttributeObject{
																				Attributes: map[string]schema.Attribute{
																					"key": schema.StringAttribute{
																						MarkdownDescription: `key is the label key that the selector applies to.`,
																						Optional:            true,
																					},
																					"operator": schema.StringAttribute{
																						MarkdownDescription: `operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.`,
																						Optional:            true,
																					},
																					"values": schema.ListAttribute{
																						MarkdownDescription: `values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.`,
																						ElementType:         types.StringType,
																						Optional:            true,
																					},
																				},
																			},
																		},
																		"match_labels": schema.MapAttribute{
																			MarkdownDescription: `matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.`,
																			ElementType:         types.StringType,
																			Optional:            true,
																		},
																	},
																},
																"namespaces": schema.ListAttribute{
																	MarkdownDescription: `namespaces specifies a static list of namespace names that the term applies to. The term is applied to the union of the namespaces listed in this field and the ones selected by namespaceSelector. null or empty namespaces list and null namespaceSelector means "this pod's namespace".`,
																	ElementType:         types.StringType,
																	Optional:            true,
																},
																"topology_key": schema.StringAttribute{
																	MarkdownDescription: `This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.`,
																	Optional:            true,
																},
															},
														},
													},
												},
											},
											"pod_anti_affinity": schema.SingleNestedAttribute{
												MarkdownDescription: `Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).`,
												Optional:            true,

												Attributes: map[string]schema.Attribute{
													"preferred_during_scheduling_ignored_during_execution": schema.ListNestedAttribute{
														MarkdownDescription: `The scheduler will prefer to schedule pods to nodes that satisfy the anti-affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling anti-affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred.`,
														Optional:            true,

														NestedObject: schema.NestedAttributeObject{
															Attributes: map[string]schema.Attribute{
																"pod_affinity_term": schema.SingleNestedAttribute{
																	MarkdownDescription: `Required. A pod affinity term, associated with the corresponding weight.`,
																	Optional:            true,

																	Attributes: map[string]schema.Attribute{
																		"label_selector": schema.SingleNestedAttribute{
																			MarkdownDescription: `A label query over a set of resources, in this case pods.`,
																			Optional:            true,

																			Attributes: map[string]schema.Attribute{
																				"match_expressions": schema.ListNestedAttribute{
																					MarkdownDescription: `matchExpressions is a list of label selector requirements. The requirements are ANDed.`,
																					Optional:            true,

																					NestedObject: schema.NestedAttributeObject{
																						Attributes: map[string]schema.Attribute{
																							"key": schema.StringAttribute{
																								MarkdownDescription: `key is the label key that the selector applies to.`,
																								Optional:            true,
																							},
																							"operator": schema.StringAttribute{
																								MarkdownDescription: `operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.`,
																								Optional:            true,
																							},
																							"values": schema.ListAttribute{
																								MarkdownDescription: `values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.`,
																								ElementType:         types.StringType,
																								Optional:            true,
																							},
																						},
																					},
																				},
																				"match_labels": schema.MapAttribute{
																					MarkdownDescription: `matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.`,
																					ElementType:         types.StringType,
																					Optional:            true,
																				},
																			},
																		},
																		"namespace_selector": schema.SingleNestedAttribute{
																			MarkdownDescription: `A label query over the set of namespaces that the term applies to. The term is applied to the union of the namespaces selected by this field and the ones listed in the namespaces field. null selector and null or empty namespaces list means "this pod's namespace". An empty selector ({}) matches all namespaces.`,
																			Optional:            true,

																			Attributes: map[string]schema.Attribute{
																				"match_expressions": schema.ListNestedAttribute{
																					MarkdownDescription: `matchExpressions is a list of label selector requirements. The requirements are ANDed.`,
																					Optional:            true,

																					NestedObject: schema.NestedAttributeObject{
																						Attributes: map[string]schema.Attribute{
																							"key": schema.StringAttribute{
																								MarkdownDescription: `key is the label key that the selector applies to.`,
																								Optional:            true,
																							},
																							"operator": schema.StringAttribute{
																								MarkdownDescription: `operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.`,
																								Optional:            true,
																							},
																							"values": schema.ListAttribute{
																								MarkdownDescription: `values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.`,
																								ElementType:         types.StringType,
																								Optional:            true,
																							},
																						},
																					},
																				},
																				"match_labels": schema.MapAttribute{
																					MarkdownDescription: `matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.`,
																					ElementType:         types.StringType,
																					Optional:            true,
																				},
																			},
																		},
																		"namespaces": schema.ListAttribute{
																			MarkdownDescription: `namespaces specifies a static list of namespace names that the term applies to. The term is applied to the union of the namespaces listed in this field and the ones selected by namespaceSelector. null or empty namespaces list and null namespaceSelector means "this pod's namespace".`,
																			ElementType:         types.StringType,
																			Optional:            true,
																		},
																		"topology_key": schema.StringAttribute{
																			MarkdownDescription: `This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.`,
																			Optional:            true,
																		},
																	},
																},
																"weight": schema.Int64Attribute{
																	MarkdownDescription: `weight associated with matching the corresponding podAffinityTerm, in the range 1-100.`,
																	Optional:            true,
																},
															},
														},
													},
													"required_during_scheduling_ignored_during_execution": schema.ListNestedAttribute{
														MarkdownDescription: `If the anti-affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the anti-affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied.`,
														Optional:            true,

														NestedObject: schema.NestedAttributeObject{
															Attributes: map[string]schema.Attribute{
																"label_selector": schema.SingleNestedAttribute{
																	MarkdownDescription: `A label query over a set of resources, in this case pods.`,
																	Optional:            true,

																	Attributes: map[string]schema.Attribute{
																		"match_expressions": schema.ListNestedAttribute{
																			MarkdownDescription: `matchExpressions is a list of label selector requirements. The requirements are ANDed.`,
																			Optional:            true,

																			NestedObject: schema.NestedAttributeObject{
																				Attributes: map[string]schema.Attribute{
																					"key": schema.StringAttribute{
																						MarkdownDescription: `key is the label key that the selector applies to.`,
																						Optional:            true,
																					},
																					"operator": schema.StringAttribute{
																						MarkdownDescription: `operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.`,
																						Optional:            true,
																					},
																					"values": schema.ListAttribute{
																						MarkdownDescription: `values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.`,
																						ElementType:         types.StringType,
																						Optional:            true,
																					},
																				},
																			},
																		},
																		"match_labels": schema.MapAttribute{
																			MarkdownDescription: `matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.`,
																			ElementType:         types.StringType,
																			Optional:            true,
																		},
																	},
																},
																"namespace_selector": schema.SingleNestedAttribute{
																	MarkdownDescription: `A label query over the set of namespaces that the term applies to. The term is applied to the union of the namespaces selected by this field and the ones listed in the namespaces field. null selector and null or empty namespaces list means "this pod's namespace". An empty selector ({}) matches all namespaces.`,
																	Optional:            true,

																	Attributes: map[string]schema.Attribute{
																		"match_expressions": schema.ListNestedAttribute{
																			MarkdownDescription: `matchExpressions is a list of label selector requirements. The requirements are ANDed.`,
																			Optional:            true,

																			NestedObject: schema.NestedAttributeObject{
																				Attributes: map[string]schema.Attribute{
																					"key": schema.StringAttribute{
																						MarkdownDescription: `key is the label key that the selector applies to.`,
																						Optional:            true,
																					},
																					"operator": schema.StringAttribute{
																						MarkdownDescription: `operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.`,
																						Optional:            true,
																					},
																					"values": schema.ListAttribute{
																						MarkdownDescription: `values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.`,
																						ElementType:         types.StringType,
																						Optional:            true,
																					},
																				},
																			},
																		},
																		"match_labels": schema.MapAttribute{
																			MarkdownDescription: `matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.`,
																			ElementType:         types.StringType,
																			Optional:            true,
																		},
																	},
																},
																"namespaces": schema.ListAttribute{
																	MarkdownDescription: `namespaces specifies a static list of namespace names that the term applies to. The term is applied to the union of the namespaces listed in this field and the ones selected by namespaceSelector. null or empty namespaces list and null namespaceSelector means "this pod's namespace".`,
																	ElementType:         types.StringType,
																	Optional:            true,
																},
																"topology_key": schema.StringAttribute{
																	MarkdownDescription: `This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.`,
																	Optional:            true,
																},
															},
														},
													},
												},
											},
										},
									},
									"automount_service_account_token": schema.BoolAttribute{
										MarkdownDescription: `AutomountServiceAccountToken indicates whether a service account token should be automatically mounted.`,
										Optional:            true,
									},
									"containers": schema.ListNestedAttribute{
										MarkdownDescription: `List of containers belonging to the pod. Containers cannot currently be added or removed. There must be at least one container in a Pod. Cannot be updated.`,
										Optional:            true,

										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"args": schema.ListAttribute{
													MarkdownDescription: `Arguments to the entrypoint. The container image's CMD is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell`,
													ElementType:         types.StringType,
													Optional:            true,
												},
												"command": schema.ListAttribute{
													MarkdownDescription: `Entrypoint array. Not executed within a shell. The container image's ENTRYPOINT is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell`,
													ElementType:         types.StringType,
													Optional:            true,
												},
												"env": schema.ListNestedAttribute{
													MarkdownDescription: `List of environment variables to set in the container. Cannot be updated.`,
													Optional:            true,

													NestedObject: schema.NestedAttributeObject{
														Attributes: map[string]schema.Attribute{
															"name": schema.StringAttribute{
																MarkdownDescription: `Name of the environment variable. Must be a C_IDENTIFIER.`,
																Optional:            true,
															},
															"value": schema.StringAttribute{
																MarkdownDescription: `Variable references $(VAR_NAME) are expanded using the previously defined environment variables in the container and any service environment variables. If a variable cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless of whether the variable exists or not. Defaults to "".`,
																Optional:            true,
															},
															"value_from": schema.SingleNestedAttribute{
																MarkdownDescription: `Source for the environment variable's value. Cannot be used if value is not empty.`,
																Optional:            true,

																Attributes: map[string]schema.Attribute{
																	"config_map_key_ref": schema.SingleNestedAttribute{
																		MarkdownDescription: `Selects a key of a ConfigMap.`,
																		Optional:            true,

																		Attributes: map[string]schema.Attribute{
																			"key": schema.StringAttribute{
																				MarkdownDescription: `The key to select.`,
																				Optional:            true,
																			},
																			"name": schema.StringAttribute{
																				MarkdownDescription: `Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names`,
																				Optional:            true,
																			},
																			"optional": schema.BoolAttribute{
																				MarkdownDescription: `Specify whether the ConfigMap or its key must be defined`,
																				Optional:            true,
																			},
																		},
																	},
																	"field_ref": schema.SingleNestedAttribute{
																		MarkdownDescription: `Selects a field of the pod: supports metadata.name, metadata.namespace, metadata.labels['<KEY>'], metadata.annotations['<KEY>'], spec.nodeName, spec.serviceAccountName, status.hostIP, status.podIP, status.podIPs.`,
																		Optional:            true,

																		Attributes: map[string]schema.Attribute{
																			"api_version": schema.StringAttribute{
																				MarkdownDescription: `Version of the schema the FieldPath is written in terms of, defaults to "v1".`,
																				Optional:            true,
																			},
																			"field_path": schema.StringAttribute{
																				MarkdownDescription: `Path of the field to select in the specified API version.`,
																				Optional:            true,
																			},
																		},
																	},
																	"resource_field_ref": schema.SingleNestedAttribute{
																		MarkdownDescription: `Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.ephemeral-storage) are currently supported.`,
																		Optional:            true,

																		Attributes: map[string]schema.Attribute{
																			"container_name": schema.StringAttribute{
																				MarkdownDescription: `Container name: required for volumes, optional for env vars`,
																				Optional:            true,
																			},
																			"divisor": schema.StringAttribute{
																				MarkdownDescription: `Specifies the output format of the exposed resources, defaults to "1"`,
																				Optional:            true,
																			},
																			"resource": schema.StringAttribute{
																				MarkdownDescription: `Required: resource to select`,
																				Optional:            true,
																			},
																		},
																	},
																	"secret_key_ref": schema.SingleNestedAttribute{
																		MarkdownDescription: `Selects a key of a secret in the pod's namespace`,
																		Optional:            true,

																		Attributes: map[string]schema.Attribute{
																			"key": schema.StringAttribute{
																				MarkdownDescription: `The key of the secret to select from.  Must be a valid secret key.`,
																				Optional:            true,
																			},
																			"name": schema.StringAttribute{
																				MarkdownDescription: `Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names`,
																				Optional:            true,
																			},
																			"optional": schema.BoolAttribute{
																				MarkdownDescription: `Specify whether the Secret or its key must be defined`,
																				Optional:            true,
																			},
																		},
																	},
																},
															},
														},
													},
												},
												"env_from": schema.ListNestedAttribute{
													MarkdownDescription: `List of sources to populate environment variables in the container. The keys defined within a source must be a C_IDENTIFIER. All invalid keys will be reported as an event when the container is starting. When a key exists in multiple sources, the value associated with the last source will take precedence. Values defined by an Env with a duplicate key will take precedence. Cannot be updated.`,
													Optional:            true,

													NestedObject: schema.NestedAttributeObject{
														Attributes: map[string]schema.Attribute{
															"config_map_ref": schema.SingleNestedAttribute{
																MarkdownDescription: `The ConfigMap to select from`,
																Optional:            true,

																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		MarkdownDescription: `Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names`,
																		Optional:            true,
																	},
																	"optional": schema.BoolAttribute{
																		MarkdownDescription: `Specify whether the ConfigMap must be defined`,
																		Optional:            true,
																	},
																},
															},
															"prefix": schema.StringAttribute{
																MarkdownDescription: `An optional identifier to prepend to each key in the ConfigMap. Must be a C_IDENTIFIER.`,
																Optional:            true,
															},
															"secret_ref": schema.SingleNestedAttribute{
																MarkdownDescription: `The Secret to select from`,
																Optional:            true,

																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		MarkdownDescription: `Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names`,
																		Optional:            true,
																	},
																	"optional": schema.BoolAttribute{
																		MarkdownDescription: `Specify whether the Secret must be defined`,
																		Optional:            true,
																	},
																},
															},
														},
													},
												},
												"image": schema.StringAttribute{
													MarkdownDescription: `Container image name. More info: https://kubernetes.io/docs/concepts/containers/images This field is optional to allow higher level config management to default or override container images in workload controllers like Deployments and StatefulSets.`,
													Optional:            true,
												},
												"image_pull_policy": schema.StringAttribute{
													MarkdownDescription: `Image pull policy. One of Always, Never, IfNotPresent. Defaults to Always if :latest tag is specified, or IfNotPresent otherwise. Cannot be updated. More info: https://kubernetes.io/docs/concepts/containers/images#updating-images`,
													Optional:            true,
												},
												"lifecycle": schema.SingleNestedAttribute{
													MarkdownDescription: `Actions that the management system should take in response to container lifecycle events. Cannot be updated.`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"post_start": schema.SingleNestedAttribute{
															MarkdownDescription: `PostStart is called immediately after a container is created. If the handler fails, the container is terminated and restarted according to its restart policy. Other management of the container blocks until the hook completes. More info: https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#container-hooks`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"exec": schema.SingleNestedAttribute{
																	MarkdownDescription: `Exec specifies the action to take.`,
																	Optional:            true,

																	Attributes: map[string]schema.Attribute{
																		"command": schema.ListAttribute{
																			MarkdownDescription: `Command is the command line to execute inside the container, the working directory for the command  is root ('/') in the container's filesystem. The command is simply exec'd, it is not run inside a shell, so traditional shell instructions ('|', etc) won't work. To use a shell, you need to explicitly call out to that shell. Exit status of 0 is treated as live/healthy and non-zero is unhealthy.`,
																			ElementType:         types.StringType,
																			Optional:            true,
																		},
																	},
																},
																"http_get": schema.SingleNestedAttribute{
																	MarkdownDescription: `HTTPGet specifies the http request to perform.`,
																	Optional:            true,

																	Attributes: map[string]schema.Attribute{
																		"host": schema.StringAttribute{
																			MarkdownDescription: `Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.`,
																			Optional:            true,
																		},
																		"http_headers": schema.ListNestedAttribute{
																			MarkdownDescription: `Custom headers to set in the request. HTTP allows repeated headers.`,
																			Optional:            true,

																			NestedObject: schema.NestedAttributeObject{
																				Attributes: map[string]schema.Attribute{
																					"name": schema.StringAttribute{
																						MarkdownDescription: `The header field name. This will be canonicalized upon output, so case-variant names will be understood as the same header.`,
																						Optional:            true,
																					},
																					"value": schema.StringAttribute{
																						MarkdownDescription: `The header field value`,
																						Optional:            true,
																					},
																				},
																			},
																		},
																		"path": schema.StringAttribute{
																			MarkdownDescription: `Path to access on the HTTP server.`,
																			Optional:            true,
																		},
																		"port": schema.StringAttribute{
																			MarkdownDescription: `Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
																			Optional:            true,
																		},
																		"scheme": schema.StringAttribute{
																			MarkdownDescription: `Scheme to use for connecting to the host. Defaults to HTTP.`,
																			Optional:            true,
																		},
																	},
																},
																"tcp_socket": schema.SingleNestedAttribute{
																	MarkdownDescription: `Deprecated. TCPSocket is NOT supported as a LifecycleHandler and kept for the backward compatibility. There are no validation of this field and lifecycle hooks will fail in runtime when tcp handler is specified.`,
																	Optional:            true,

																	Attributes: map[string]schema.Attribute{
																		"host": schema.StringAttribute{
																			MarkdownDescription: `Optional: Host name to connect to, defaults to the pod IP.`,
																			Optional:            true,
																		},
																		"port": schema.StringAttribute{
																			MarkdownDescription: `Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
																			Optional:            true,
																		},
																	},
																},
															},
														},
														"pre_stop": schema.SingleNestedAttribute{
															MarkdownDescription: `PreStop is called immediately before a container is terminated due to an API request or management event such as liveness/startup probe failure, preemption, resource contention, etc. The handler is not called if the container crashes or exits. The Pod's termination grace period countdown begins before the PreStop hook is executed. Regardless of the outcome of the handler, the container will eventually terminate within the Pod's termination grace period (unless delayed by finalizers). Other management of the container blocks until the hook completes or until the termination grace period is reached. More info: https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#container-hooks`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"exec": schema.SingleNestedAttribute{
																	MarkdownDescription: `Exec specifies the action to take.`,
																	Optional:            true,

																	Attributes: map[string]schema.Attribute{
																		"command": schema.ListAttribute{
																			MarkdownDescription: `Command is the command line to execute inside the container, the working directory for the command  is root ('/') in the container's filesystem. The command is simply exec'd, it is not run inside a shell, so traditional shell instructions ('|', etc) won't work. To use a shell, you need to explicitly call out to that shell. Exit status of 0 is treated as live/healthy and non-zero is unhealthy.`,
																			ElementType:         types.StringType,
																			Optional:            true,
																		},
																	},
																},
																"http_get": schema.SingleNestedAttribute{
																	MarkdownDescription: `HTTPGet specifies the http request to perform.`,
																	Optional:            true,

																	Attributes: map[string]schema.Attribute{
																		"host": schema.StringAttribute{
																			MarkdownDescription: `Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.`,
																			Optional:            true,
																		},
																		"http_headers": schema.ListNestedAttribute{
																			MarkdownDescription: `Custom headers to set in the request. HTTP allows repeated headers.`,
																			Optional:            true,

																			NestedObject: schema.NestedAttributeObject{
																				Attributes: map[string]schema.Attribute{
																					"name": schema.StringAttribute{
																						MarkdownDescription: `The header field name. This will be canonicalized upon output, so case-variant names will be understood as the same header.`,
																						Optional:            true,
																					},
																					"value": schema.StringAttribute{
																						MarkdownDescription: `The header field value`,
																						Optional:            true,
																					},
																				},
																			},
																		},
																		"path": schema.StringAttribute{
																			MarkdownDescription: `Path to access on the HTTP server.`,
																			Optional:            true,
																		},
																		"port": schema.StringAttribute{
																			MarkdownDescription: `Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
																			Optional:            true,
																		},
																		"scheme": schema.StringAttribute{
																			MarkdownDescription: `Scheme to use for connecting to the host. Defaults to HTTP.`,
																			Optional:            true,
																		},
																	},
																},
																"tcp_socket": schema.SingleNestedAttribute{
																	MarkdownDescription: `Deprecated. TCPSocket is NOT supported as a LifecycleHandler and kept for the backward compatibility. There are no validation of this field and lifecycle hooks will fail in runtime when tcp handler is specified.`,
																	Optional:            true,

																	Attributes: map[string]schema.Attribute{
																		"host": schema.StringAttribute{
																			MarkdownDescription: `Optional: Host name to connect to, defaults to the pod IP.`,
																			Optional:            true,
																		},
																		"port": schema.StringAttribute{
																			MarkdownDescription: `Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
																			Optional:            true,
																		},
																	},
																},
															},
														},
													},
												},
												"liveness_probe": schema.SingleNestedAttribute{
													MarkdownDescription: `Periodic probe of container liveness. Container will be restarted if the probe fails. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"exec": schema.SingleNestedAttribute{
															MarkdownDescription: `Exec specifies the action to take.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"command": schema.ListAttribute{
																	MarkdownDescription: `Command is the command line to execute inside the container, the working directory for the command  is root ('/') in the container's filesystem. The command is simply exec'd, it is not run inside a shell, so traditional shell instructions ('|', etc) won't work. To use a shell, you need to explicitly call out to that shell. Exit status of 0 is treated as live/healthy and non-zero is unhealthy.`,
																	ElementType:         types.StringType,
																	Optional:            true,
																},
															},
														},
														"failure_threshold": schema.Int64Attribute{
															MarkdownDescription: `Minimum consecutive failures for the probe to be considered failed after having succeeded. Defaults to 3. Minimum value is 1.`,
															Optional:            true,
														},
														"grpc": schema.SingleNestedAttribute{
															MarkdownDescription: `GRPC specifies an action involving a GRPC port.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"port": schema.Int64Attribute{
																	MarkdownDescription: `Port number of the gRPC service. Number must be in the range 1 to 65535.`,
																	Optional:            true,
																},
																"service": schema.StringAttribute{
																	MarkdownDescription: `Service is the name of the service to place in the gRPC HealthCheckRequest (see https://github.com/grpc/grpc/blob/master/doc/health-checking.md).

If this is not specified, the default behavior is defined by gRPC.`,
																	Optional: true,
																},
															},
														},
														"http_get": schema.SingleNestedAttribute{
															MarkdownDescription: `HTTPGet specifies the http request to perform.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"host": schema.StringAttribute{
																	MarkdownDescription: `Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.`,
																	Optional:            true,
																},
																"http_headers": schema.ListNestedAttribute{
																	MarkdownDescription: `Custom headers to set in the request. HTTP allows repeated headers.`,
																	Optional:            true,

																	NestedObject: schema.NestedAttributeObject{
																		Attributes: map[string]schema.Attribute{
																			"name": schema.StringAttribute{
																				MarkdownDescription: `The header field name. This will be canonicalized upon output, so case-variant names will be understood as the same header.`,
																				Optional:            true,
																			},
																			"value": schema.StringAttribute{
																				MarkdownDescription: `The header field value`,
																				Optional:            true,
																			},
																		},
																	},
																},
																"path": schema.StringAttribute{
																	MarkdownDescription: `Path to access on the HTTP server.`,
																	Optional:            true,
																},
																"port": schema.StringAttribute{
																	MarkdownDescription: `Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
																	Optional:            true,
																},
																"scheme": schema.StringAttribute{
																	MarkdownDescription: `Scheme to use for connecting to the host. Defaults to HTTP.`,
																	Optional:            true,
																},
															},
														},
														"initial_delay_seconds": schema.Int64Attribute{
															MarkdownDescription: `Number of seconds after the container has started before liveness probes are initiated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes`,
															Optional:            true,
														},
														"period_seconds": schema.Int64Attribute{
															MarkdownDescription: `How often (in seconds) to perform the probe. Default to 10 seconds. Minimum value is 1.`,
															Optional:            true,
														},
														"success_threshold": schema.Int64Attribute{
															MarkdownDescription: `Minimum consecutive successes for the probe to be considered successful after having failed. Defaults to 1. Must be 1 for liveness and startup. Minimum value is 1.`,
															Optional:            true,
														},
														"tcp_socket": schema.SingleNestedAttribute{
															MarkdownDescription: `TCPSocket specifies an action involving a TCP port.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"host": schema.StringAttribute{
																	MarkdownDescription: `Optional: Host name to connect to, defaults to the pod IP.`,
																	Optional:            true,
																},
																"port": schema.StringAttribute{
																	MarkdownDescription: `Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
																	Optional:            true,
																},
															},
														},
														"termination_grace_period_seconds": schema.Int64Attribute{
															MarkdownDescription: `Optional duration in seconds the pod needs to terminate gracefully upon probe failure. The grace period is the duration in seconds after the processes running in the pod are sent a termination signal and the time when the processes are forcibly halted with a kill signal. Set this value longer than the expected cleanup time for your process. If this value is nil, the pod's terminationGracePeriodSeconds will be used. Otherwise, this value overrides the value provided by the pod spec. Value must be non-negative integer. The value zero indicates stop immediately via the kill signal (no opportunity to shut down). This is a beta field and requires enabling ProbeTerminationGracePeriod feature gate. Minimum value is 1. spec.terminationGracePeriodSeconds is used if unset.`,
															Optional:            true,
														},
														"timeout_seconds": schema.Int64Attribute{
															MarkdownDescription: `Number of seconds after which the probe times out. Defaults to 1 second. Minimum value is 1. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes`,
															Optional:            true,
														},
													},
												},
												"name": schema.StringAttribute{
													MarkdownDescription: `Name of the container specified as a DNS_LABEL. Each container in a pod must have a unique name (DNS_LABEL). Cannot be updated.`,
													Optional:            true,
												},
												"ports": schema.ListNestedAttribute{
													MarkdownDescription: `List of ports to expose from the container. Not specifying a port here DOES NOT prevent that port from being exposed. Any port which is listening on the default "0.0.0.0" address inside a container will be accessible from the network. Modifying this array with strategic merge patch may corrupt the data. For more information See https://github.com/kubernetes/kubernetes/issues/108255. Cannot be updated.`,
													Optional:            true,

													NestedObject: schema.NestedAttributeObject{
														Attributes: map[string]schema.Attribute{
															"container_port": schema.Int64Attribute{
																MarkdownDescription: `Number of port to expose on the pod's IP address. This must be a valid port number, 0 < x < 65536.`,
																Optional:            true,
															},
															"host_ip": schema.StringAttribute{
																MarkdownDescription: `What host IP to bind the external port to.`,
																Optional:            true,
															},
															"host_port": schema.Int64Attribute{
																MarkdownDescription: `Number of port to expose on the host. If specified, this must be a valid port number, 0 < x < 65536. If HostNetwork is specified, this must match ContainerPort. Most containers do not need this.`,
																Optional:            true,
															},
															"name": schema.StringAttribute{
																MarkdownDescription: `If specified, this must be an IANA_SVC_NAME and unique within the pod. Each named port in a pod must have a unique name. Name for the port that can be referred to by services.`,
																Optional:            true,
															},
															"protocol": schema.StringAttribute{
																MarkdownDescription: `Protocol for port. Must be UDP, TCP, or SCTP. Defaults to "TCP".`,
																Optional:            true,
															},
														},
													},
												},
												"readiness_probe": schema.SingleNestedAttribute{
													MarkdownDescription: `Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"exec": schema.SingleNestedAttribute{
															MarkdownDescription: `Exec specifies the action to take.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"command": schema.ListAttribute{
																	MarkdownDescription: `Command is the command line to execute inside the container, the working directory for the command  is root ('/') in the container's filesystem. The command is simply exec'd, it is not run inside a shell, so traditional shell instructions ('|', etc) won't work. To use a shell, you need to explicitly call out to that shell. Exit status of 0 is treated as live/healthy and non-zero is unhealthy.`,
																	ElementType:         types.StringType,
																	Optional:            true,
																},
															},
														},
														"failure_threshold": schema.Int64Attribute{
															MarkdownDescription: `Minimum consecutive failures for the probe to be considered failed after having succeeded. Defaults to 3. Minimum value is 1.`,
															Optional:            true,
														},
														"grpc": schema.SingleNestedAttribute{
															MarkdownDescription: `GRPC specifies an action involving a GRPC port.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"port": schema.Int64Attribute{
																	MarkdownDescription: `Port number of the gRPC service. Number must be in the range 1 to 65535.`,
																	Optional:            true,
																},
																"service": schema.StringAttribute{
																	MarkdownDescription: `Service is the name of the service to place in the gRPC HealthCheckRequest (see https://github.com/grpc/grpc/blob/master/doc/health-checking.md).

If this is not specified, the default behavior is defined by gRPC.`,
																	Optional: true,
																},
															},
														},
														"http_get": schema.SingleNestedAttribute{
															MarkdownDescription: `HTTPGet specifies the http request to perform.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"host": schema.StringAttribute{
																	MarkdownDescription: `Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.`,
																	Optional:            true,
																},
																"http_headers": schema.ListNestedAttribute{
																	MarkdownDescription: `Custom headers to set in the request. HTTP allows repeated headers.`,
																	Optional:            true,

																	NestedObject: schema.NestedAttributeObject{
																		Attributes: map[string]schema.Attribute{
																			"name": schema.StringAttribute{
																				MarkdownDescription: `The header field name. This will be canonicalized upon output, so case-variant names will be understood as the same header.`,
																				Optional:            true,
																			},
																			"value": schema.StringAttribute{
																				MarkdownDescription: `The header field value`,
																				Optional:            true,
																			},
																		},
																	},
																},
																"path": schema.StringAttribute{
																	MarkdownDescription: `Path to access on the HTTP server.`,
																	Optional:            true,
																},
																"port": schema.StringAttribute{
																	MarkdownDescription: `Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
																	Optional:            true,
																},
																"scheme": schema.StringAttribute{
																	MarkdownDescription: `Scheme to use for connecting to the host. Defaults to HTTP.`,
																	Optional:            true,
																},
															},
														},
														"initial_delay_seconds": schema.Int64Attribute{
															MarkdownDescription: `Number of seconds after the container has started before liveness probes are initiated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes`,
															Optional:            true,
														},
														"period_seconds": schema.Int64Attribute{
															MarkdownDescription: `How often (in seconds) to perform the probe. Default to 10 seconds. Minimum value is 1.`,
															Optional:            true,
														},
														"success_threshold": schema.Int64Attribute{
															MarkdownDescription: `Minimum consecutive successes for the probe to be considered successful after having failed. Defaults to 1. Must be 1 for liveness and startup. Minimum value is 1.`,
															Optional:            true,
														},
														"tcp_socket": schema.SingleNestedAttribute{
															MarkdownDescription: `TCPSocket specifies an action involving a TCP port.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"host": schema.StringAttribute{
																	MarkdownDescription: `Optional: Host name to connect to, defaults to the pod IP.`,
																	Optional:            true,
																},
																"port": schema.StringAttribute{
																	MarkdownDescription: `Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
																	Optional:            true,
																},
															},
														},
														"termination_grace_period_seconds": schema.Int64Attribute{
															MarkdownDescription: `Optional duration in seconds the pod needs to terminate gracefully upon probe failure. The grace period is the duration in seconds after the processes running in the pod are sent a termination signal and the time when the processes are forcibly halted with a kill signal. Set this value longer than the expected cleanup time for your process. If this value is nil, the pod's terminationGracePeriodSeconds will be used. Otherwise, this value overrides the value provided by the pod spec. Value must be non-negative integer. The value zero indicates stop immediately via the kill signal (no opportunity to shut down). This is a beta field and requires enabling ProbeTerminationGracePeriod feature gate. Minimum value is 1. spec.terminationGracePeriodSeconds is used if unset.`,
															Optional:            true,
														},
														"timeout_seconds": schema.Int64Attribute{
															MarkdownDescription: `Number of seconds after which the probe times out. Defaults to 1 second. Minimum value is 1. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes`,
															Optional:            true,
														},
													},
												},
												"resize_policy": schema.ListNestedAttribute{
													MarkdownDescription: `Resources resize policy for the container.`,
													Optional:            true,

													NestedObject: schema.NestedAttributeObject{
														Attributes: map[string]schema.Attribute{
															"resource_name": schema.StringAttribute{
																MarkdownDescription: `Name of the resource to which this resource resize policy applies. Supported values: cpu, memory.`,
																Optional:            true,
															},
															"restart_policy": schema.StringAttribute{
																MarkdownDescription: `Restart policy to apply when specified resource is resized. If not specified, it defaults to NotRequired.`,
																Optional:            true,
															},
														},
													},
												},
												"resources": schema.SingleNestedAttribute{
													MarkdownDescription: `Compute Resources required by this container. Cannot be updated. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"claims": schema.ListNestedAttribute{
															MarkdownDescription: `Claims lists the names of resources, defined in spec.resourceClaims, that are used by this container.

This is an alpha field and requires enabling the DynamicResourceAllocation feature gate.

This field is immutable. It can only be set for containers.`,
															Optional: true,

															NestedObject: schema.NestedAttributeObject{
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		MarkdownDescription: `Name must match the name of one entry in pod.spec.resourceClaims of the Pod where this field is used. It makes that resource available inside a container.`,
																		Optional:            true,
																	},
																},
															},
														},
														"limits": schema.MapAttribute{
															MarkdownDescription: `Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/`,
															ElementType:         types.StringType,
															Optional:            true,
														},
														"requests": schema.MapAttribute{
															MarkdownDescription: `Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. Requests cannot exceed Limits. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/`,
															ElementType:         types.StringType,
															Optional:            true,
														},
													},
												},
												"restart_policy": schema.StringAttribute{
													MarkdownDescription: `RestartPolicy defines the restart behavior of individual containers in a pod. This field may only be set for init containers, and the only allowed value is "Always". For non-init containers or when this field is not specified, the restart behavior is defined by the Pod's restart policy and the container type. Setting the RestartPolicy as "Always" for the init container will have the following effect: this init container will be continually restarted on exit until all regular containers have terminated. Once all regular containers have completed, all init containers with restartPolicy "Always" will be shut down. This lifecycle differs from normal init containers and is often referred to as a "sidecar" container. Although this init container still starts in the init container sequence, it does not wait for the container to complete before proceeding to the next init container. Instead, the next init container starts immediately after this init container is started, or after any startupProbe has successfully completed.`,
													Optional:            true,
												},
												"security_context": schema.SingleNestedAttribute{
													MarkdownDescription: `SecurityContext defines the security options the container should be run with. If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext. More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"allow_privilege_escalation": schema.BoolAttribute{
															MarkdownDescription: `AllowPrivilegeEscalation controls whether a process can gain more privileges than its parent process. This bool directly controls if the no_new_privs flag will be set on the container process. AllowPrivilegeEscalation is true always when the container is: 1) run as Privileged 2) has CAP_SYS_ADMIN Note that this field cannot be set when spec.os.name is windows.`,
															Optional:            true,
														},
														"capabilities": schema.SingleNestedAttribute{
															MarkdownDescription: `The capabilities to add/drop when running containers. Defaults to the default set of capabilities granted by the container runtime. Note that this field cannot be set when spec.os.name is windows.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"add": schema.ListAttribute{
																	MarkdownDescription: `Added capabilities`,
																	ElementType:         types.StringType,
																	Optional:            true,
																},
																"drop": schema.ListAttribute{
																	MarkdownDescription: `Removed capabilities`,
																	ElementType:         types.StringType,
																	Optional:            true,
																},
															},
														},
														"privileged": schema.BoolAttribute{
															MarkdownDescription: `Run container in privileged mode. Processes in privileged containers are essentially equivalent to root on the host. Defaults to false. Note that this field cannot be set when spec.os.name is windows.`,
															Optional:            true,
														},
														"proc_mount": schema.StringAttribute{
															MarkdownDescription: `procMount denotes the type of proc mount to use for the containers. The default is DefaultProcMount which uses the container runtime defaults for readonly paths and masked paths. This requires the ProcMountType feature flag to be enabled. Note that this field cannot be set when spec.os.name is windows.`,
															Optional:            true,
														},
														"read_only_root_filesystem": schema.BoolAttribute{
															MarkdownDescription: `Whether this container has a read-only root filesystem. Default is false. Note that this field cannot be set when spec.os.name is windows.`,
															Optional:            true,
														},
														"run_as_group": schema.Int64Attribute{
															MarkdownDescription: `The GID to run the entrypoint of the container process. Uses runtime default if unset. May also be set in PodSecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. Note that this field cannot be set when spec.os.name is windows.`,
															Optional:            true,
														},
														"run_as_non_root": schema.BoolAttribute{
															MarkdownDescription: `Indicates that the container must run as a non-root user. If true, the Kubelet will validate the image at runtime to ensure that it does not run as UID 0 (root) and fail to start the container if it does. If unset or false, no such validation will be performed. May also be set in PodSecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.`,
															Optional:            true,
														},
														"run_as_user": schema.Int64Attribute{
															MarkdownDescription: `The UID to run the entrypoint of the container process. Defaults to user specified in image metadata if unspecified. May also be set in PodSecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. Note that this field cannot be set when spec.os.name is windows.`,
															Optional:            true,
														},
														"se_linux_options": schema.SingleNestedAttribute{
															MarkdownDescription: `The SELinux context to be applied to the container. If unspecified, the container runtime will allocate a random SELinux context for each container.  May also be set in PodSecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. Note that this field cannot be set when spec.os.name is windows.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"level": schema.StringAttribute{
																	MarkdownDescription: `Level is SELinux level label that applies to the container.`,
																	Optional:            true,
																},
																"role": schema.StringAttribute{
																	MarkdownDescription: `Role is a SELinux role label that applies to the container.`,
																	Optional:            true,
																},
																"type": schema.StringAttribute{
																	MarkdownDescription: `Type is a SELinux type label that applies to the container.`,
																	Optional:            true,
																},
																"user": schema.StringAttribute{
																	MarkdownDescription: `User is a SELinux user label that applies to the container.`,
																	Optional:            true,
																},
															},
														},
														"seccomp_profile": schema.SingleNestedAttribute{
															MarkdownDescription: `The seccomp options to use by this container. If seccomp options are provided at both the pod & container level, the container options override the pod options. Note that this field cannot be set when spec.os.name is windows.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"localhost_profile": schema.StringAttribute{
																	MarkdownDescription: `localhostProfile indicates a profile defined in a file on the node should be used. The profile must be preconfigured on the node to work. Must be a descending path, relative to the kubelet's configured seccomp profile location. Must be set if type is "Localhost". Must NOT be set for any other type.`,
																	Optional:            true,
																},
																"type": schema.StringAttribute{
																	MarkdownDescription: `type indicates which kind of seccomp profile will be applied. Valid options are:

Localhost - a profile defined in a file on the node should be used. RuntimeDefault - the container runtime default profile should be used. Unconfined - no profile should be applied.`,
																	Optional: true,
																},
															},
														},
														"windows_options": schema.SingleNestedAttribute{
															MarkdownDescription: `The Windows specific settings applied to all containers. If unspecified, the options from the PodSecurityContext will be used. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. Note that this field cannot be set when spec.os.name is linux.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"gmsa_credential_spec": schema.StringAttribute{
																	MarkdownDescription: `GMSACredentialSpec is where the GMSA admission webhook (https://github.com/kubernetes-sigs/windows-gmsa) inlines the contents of the GMSA credential spec named by the GMSACredentialSpecName field.`,
																	Optional:            true,
																},
																"gmsa_credential_spec_name": schema.StringAttribute{
																	MarkdownDescription: `GMSACredentialSpecName is the name of the GMSA credential spec to use.`,
																	Optional:            true,
																},
																"host_process": schema.BoolAttribute{
																	MarkdownDescription: `HostProcess determines if a container should be run as a 'Host Process' container. All of a Pod's containers must have the same effective HostProcess value (it is not allowed to have a mix of HostProcess containers and non-HostProcess containers). In addition, if HostProcess is true then HostNetwork must also be set to true.`,
																	Optional:            true,
																},
																"run_as_user_name": schema.StringAttribute{
																	MarkdownDescription: `The UserName in Windows to run the entrypoint of the container process. Defaults to the user specified in image metadata if unspecified. May also be set in PodSecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.`,
																	Optional:            true,
																},
															},
														},
													},
												},
												"startup_probe": schema.SingleNestedAttribute{
													MarkdownDescription: `StartupProbe indicates that the Pod has successfully initialized. If specified, no other probes are executed until this completes successfully. If this probe fails, the Pod will be restarted, just as if the livenessProbe failed. This can be used to provide different probe parameters at the beginning of a Pod's lifecycle, when it might take a long time to load data or warm a cache, than during steady-state operation. This cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"exec": schema.SingleNestedAttribute{
															MarkdownDescription: `Exec specifies the action to take.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"command": schema.ListAttribute{
																	MarkdownDescription: `Command is the command line to execute inside the container, the working directory for the command  is root ('/') in the container's filesystem. The command is simply exec'd, it is not run inside a shell, so traditional shell instructions ('|', etc) won't work. To use a shell, you need to explicitly call out to that shell. Exit status of 0 is treated as live/healthy and non-zero is unhealthy.`,
																	ElementType:         types.StringType,
																	Optional:            true,
																},
															},
														},
														"failure_threshold": schema.Int64Attribute{
															MarkdownDescription: `Minimum consecutive failures for the probe to be considered failed after having succeeded. Defaults to 3. Minimum value is 1.`,
															Optional:            true,
														},
														"grpc": schema.SingleNestedAttribute{
															MarkdownDescription: `GRPC specifies an action involving a GRPC port.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"port": schema.Int64Attribute{
																	MarkdownDescription: `Port number of the gRPC service. Number must be in the range 1 to 65535.`,
																	Optional:            true,
																},
																"service": schema.StringAttribute{
																	MarkdownDescription: `Service is the name of the service to place in the gRPC HealthCheckRequest (see https://github.com/grpc/grpc/blob/master/doc/health-checking.md).

If this is not specified, the default behavior is defined by gRPC.`,
																	Optional: true,
																},
															},
														},
														"http_get": schema.SingleNestedAttribute{
															MarkdownDescription: `HTTPGet specifies the http request to perform.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"host": schema.StringAttribute{
																	MarkdownDescription: `Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.`,
																	Optional:            true,
																},
																"http_headers": schema.ListNestedAttribute{
																	MarkdownDescription: `Custom headers to set in the request. HTTP allows repeated headers.`,
																	Optional:            true,

																	NestedObject: schema.NestedAttributeObject{
																		Attributes: map[string]schema.Attribute{
																			"name": schema.StringAttribute{
																				MarkdownDescription: `The header field name. This will be canonicalized upon output, so case-variant names will be understood as the same header.`,
																				Optional:            true,
																			},
																			"value": schema.StringAttribute{
																				MarkdownDescription: `The header field value`,
																				Optional:            true,
																			},
																		},
																	},
																},
																"path": schema.StringAttribute{
																	MarkdownDescription: `Path to access on the HTTP server.`,
																	Optional:            true,
																},
																"port": schema.StringAttribute{
																	MarkdownDescription: `Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
																	Optional:            true,
																},
																"scheme": schema.StringAttribute{
																	MarkdownDescription: `Scheme to use for connecting to the host. Defaults to HTTP.`,
																	Optional:            true,
																},
															},
														},
														"initial_delay_seconds": schema.Int64Attribute{
															MarkdownDescription: `Number of seconds after the container has started before liveness probes are initiated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes`,
															Optional:            true,
														},
														"period_seconds": schema.Int64Attribute{
															MarkdownDescription: `How often (in seconds) to perform the probe. Default to 10 seconds. Minimum value is 1.`,
															Optional:            true,
														},
														"success_threshold": schema.Int64Attribute{
															MarkdownDescription: `Minimum consecutive successes for the probe to be considered successful after having failed. Defaults to 1. Must be 1 for liveness and startup. Minimum value is 1.`,
															Optional:            true,
														},
														"tcp_socket": schema.SingleNestedAttribute{
															MarkdownDescription: `TCPSocket specifies an action involving a TCP port.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"host": schema.StringAttribute{
																	MarkdownDescription: `Optional: Host name to connect to, defaults to the pod IP.`,
																	Optional:            true,
																},
																"port": schema.StringAttribute{
																	MarkdownDescription: `Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
																	Optional:            true,
																},
															},
														},
														"termination_grace_period_seconds": schema.Int64Attribute{
															MarkdownDescription: `Optional duration in seconds the pod needs to terminate gracefully upon probe failure. The grace period is the duration in seconds after the processes running in the pod are sent a termination signal and the time when the processes are forcibly halted with a kill signal. Set this value longer than the expected cleanup time for your process. If this value is nil, the pod's terminationGracePeriodSeconds will be used. Otherwise, this value overrides the value provided by the pod spec. Value must be non-negative integer. The value zero indicates stop immediately via the kill signal (no opportunity to shut down). This is a beta field and requires enabling ProbeTerminationGracePeriod feature gate. Minimum value is 1. spec.terminationGracePeriodSeconds is used if unset.`,
															Optional:            true,
														},
														"timeout_seconds": schema.Int64Attribute{
															MarkdownDescription: `Number of seconds after which the probe times out. Defaults to 1 second. Minimum value is 1. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes`,
															Optional:            true,
														},
													},
												},
												"stdin": schema.BoolAttribute{
													MarkdownDescription: `Whether this container should allocate a buffer for stdin in the container runtime. If this is not set, reads from stdin in the container will always result in EOF. Default is false.`,
													Optional:            true,
												},
												"stdin_once": schema.BoolAttribute{
													MarkdownDescription: `Whether the container runtime should close the stdin channel after it has been opened by a single attach. When stdin is true the stdin stream will remain open across multiple attach sessions. If stdinOnce is set to true, stdin is opened on container start, is empty until the first client attaches to stdin, and then remains open and accepts data until the client disconnects, at which time stdin is closed and remains closed until the container is restarted. If this flag is false, a container processes that reads from stdin will never receive an EOF. Default is false`,
													Optional:            true,
												},
												"termination_message_path": schema.StringAttribute{
													MarkdownDescription: `Optional: Path at which the file to which the container's termination message will be written is mounted into the container's filesystem. Message written is intended to be brief final status, such as an assertion failure message. Will be truncated by the node if greater than 4096 bytes. The total message length across all containers will be limited to 12kb. Defaults to /dev/termination-log. Cannot be updated.`,
													Optional:            true,
												},
												"termination_message_policy": schema.StringAttribute{
													MarkdownDescription: `Indicate how the termination message should be populated. File will use the contents of terminationMessagePath to populate the container status message on both success and failure. FallbackToLogsOnError will use the last chunk of container log output if the termination message file is empty and the container exited with an error. The log output is limited to 2048 bytes or 80 lines, whichever is smaller. Defaults to File. Cannot be updated.`,
													Optional:            true,
												},
												"tty": schema.BoolAttribute{
													MarkdownDescription: `Whether this container should allocate a TTY for itself, also requires 'stdin' to be true. Default is false.`,
													Optional:            true,
												},
												"volume_devices": schema.ListNestedAttribute{
													MarkdownDescription: `volumeDevices is the list of block devices to be used by the container.`,
													Optional:            true,

													NestedObject: schema.NestedAttributeObject{
														Attributes: map[string]schema.Attribute{
															"device_path": schema.StringAttribute{
																MarkdownDescription: `devicePath is the path inside of the container that the device will be mapped to.`,
																Optional:            true,
															},
															"name": schema.StringAttribute{
																MarkdownDescription: `name must match the name of a persistentVolumeClaim in the pod`,
																Optional:            true,
															},
														},
													},
												},
												"volume_mounts": schema.ListNestedAttribute{
													MarkdownDescription: `Pod volumes to mount into the container's filesystem. Cannot be updated.`,
													Optional:            true,

													NestedObject: schema.NestedAttributeObject{
														Attributes: map[string]schema.Attribute{
															"mount_path": schema.StringAttribute{
																MarkdownDescription: `Path within the container at which the volume should be mounted.  Must not contain ':'.`,
																Optional:            true,
															},
															"mount_propagation": schema.StringAttribute{
																MarkdownDescription: `mountPropagation determines how mounts are propagated from the host to container and the other way around. When not set, MountPropagationNone is used. This field is beta in 1.10.`,
																Optional:            true,
															},
															"name": schema.StringAttribute{
																MarkdownDescription: `This must match the Name of a Volume.`,
																Optional:            true,
															},
															"read_only": schema.BoolAttribute{
																MarkdownDescription: `Mounted read-only if true, read-write otherwise (false or unspecified). Defaults to false.`,
																Optional:            true,
															},
															"sub_path": schema.StringAttribute{
																MarkdownDescription: `Path within the volume from which the container's volume should be mounted. Defaults to "" (volume's root).`,
																Optional:            true,
															},
															"sub_path_expr": schema.StringAttribute{
																MarkdownDescription: `Expanded path within the volume from which the container's volume should be mounted. Behaves similarly to SubPath but environment variable references $(VAR_NAME) are expanded using the container's environment. Defaults to "" (volume's root). SubPathExpr and SubPath are mutually exclusive.`,
																Optional:            true,
															},
														},
													},
												},
												"working_dir": schema.StringAttribute{
													MarkdownDescription: `Container's working directory. If not specified, the container runtime's default will be used, which might be configured in the container image. Cannot be updated.`,
													Optional:            true,
												},
											},
										},
									},
									"dns_config": schema.SingleNestedAttribute{
										MarkdownDescription: `Specifies the DNS parameters of a pod. Parameters specified here will be merged to the generated DNS configuration based on DNSPolicy.`,
										Optional:            true,

										Attributes: map[string]schema.Attribute{
											"nameservers": schema.ListAttribute{
												MarkdownDescription: `A list of DNS name server IP addresses. This will be appended to the base nameservers generated from DNSPolicy. Duplicated nameservers will be removed.`,
												ElementType:         types.StringType,
												Optional:            true,
											},
											"options": schema.ListNestedAttribute{
												MarkdownDescription: `A list of DNS resolver options. This will be merged with the base options generated from DNSPolicy. Duplicated entries will be removed. Resolution options given in Options will override those that appear in the base DNSPolicy.`,
												Optional:            true,

												NestedObject: schema.NestedAttributeObject{
													Attributes: map[string]schema.Attribute{
														"name": schema.StringAttribute{
															MarkdownDescription: `Required.`,
															Optional:            true,
														},
														"value": schema.StringAttribute{
															MarkdownDescription: ``,
															Optional:            true,
														},
													},
												},
											},
											"searches": schema.ListAttribute{
												MarkdownDescription: `A list of DNS search domains for host-name lookup. This will be appended to the base search paths generated from DNSPolicy. Duplicated search paths will be removed.`,
												ElementType:         types.StringType,
												Optional:            true,
											},
										},
									},
									"dns_policy": schema.StringAttribute{
										MarkdownDescription: `Set DNS policy for the pod. Defaults to "ClusterFirst". Valid values are 'ClusterFirstWithHostNet', 'ClusterFirst', 'Default' or 'None'. DNS parameters given in DNSConfig will be merged with the policy selected with DNSPolicy. To have DNS options set along with hostNetwork, you have to specify DNS policy explicitly to 'ClusterFirstWithHostNet'.`,
										Optional:            true,
									},
									"enable_service_links": schema.BoolAttribute{
										MarkdownDescription: `EnableServiceLinks indicates whether information about services should be injected into pod's environment variables, matching the syntax of Docker links. Optional: Defaults to true.`,
										Optional:            true,
									},
									"ephemeral_containers": schema.ListNestedAttribute{
										MarkdownDescription: `List of ephemeral containers run in this pod. Ephemeral containers may be run in an existing pod to perform user-initiated actions such as debugging. This list cannot be specified when creating a pod, and it cannot be modified by updating the pod spec. In order to add an ephemeral container to an existing pod, use the pod's ephemeralcontainers subresource.`,
										Optional:            true,

										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"args": schema.ListAttribute{
													MarkdownDescription: `Arguments to the entrypoint. The image's CMD is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell`,
													ElementType:         types.StringType,
													Optional:            true,
												},
												"command": schema.ListAttribute{
													MarkdownDescription: `Entrypoint array. Not executed within a shell. The image's ENTRYPOINT is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell`,
													ElementType:         types.StringType,
													Optional:            true,
												},
												"env": schema.ListNestedAttribute{
													MarkdownDescription: `List of environment variables to set in the container. Cannot be updated.`,
													Optional:            true,

													NestedObject: schema.NestedAttributeObject{
														Attributes: map[string]schema.Attribute{
															"name": schema.StringAttribute{
																MarkdownDescription: `Name of the environment variable. Must be a C_IDENTIFIER.`,
																Optional:            true,
															},
															"value": schema.StringAttribute{
																MarkdownDescription: `Variable references $(VAR_NAME) are expanded using the previously defined environment variables in the container and any service environment variables. If a variable cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless of whether the variable exists or not. Defaults to "".`,
																Optional:            true,
															},
															"value_from": schema.SingleNestedAttribute{
																MarkdownDescription: `Source for the environment variable's value. Cannot be used if value is not empty.`,
																Optional:            true,

																Attributes: map[string]schema.Attribute{
																	"config_map_key_ref": schema.SingleNestedAttribute{
																		MarkdownDescription: `Selects a key of a ConfigMap.`,
																		Optional:            true,

																		Attributes: map[string]schema.Attribute{
																			"key": schema.StringAttribute{
																				MarkdownDescription: `The key to select.`,
																				Optional:            true,
																			},
																			"name": schema.StringAttribute{
																				MarkdownDescription: `Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names`,
																				Optional:            true,
																			},
																			"optional": schema.BoolAttribute{
																				MarkdownDescription: `Specify whether the ConfigMap or its key must be defined`,
																				Optional:            true,
																			},
																		},
																	},
																	"field_ref": schema.SingleNestedAttribute{
																		MarkdownDescription: `Selects a field of the pod: supports metadata.name, metadata.namespace, metadata.labels['<KEY>'], metadata.annotations['<KEY>'], spec.nodeName, spec.serviceAccountName, status.hostIP, status.podIP, status.podIPs.`,
																		Optional:            true,

																		Attributes: map[string]schema.Attribute{
																			"api_version": schema.StringAttribute{
																				MarkdownDescription: `Version of the schema the FieldPath is written in terms of, defaults to "v1".`,
																				Optional:            true,
																			},
																			"field_path": schema.StringAttribute{
																				MarkdownDescription: `Path of the field to select in the specified API version.`,
																				Optional:            true,
																			},
																		},
																	},
																	"resource_field_ref": schema.SingleNestedAttribute{
																		MarkdownDescription: `Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.ephemeral-storage) are currently supported.`,
																		Optional:            true,

																		Attributes: map[string]schema.Attribute{
																			"container_name": schema.StringAttribute{
																				MarkdownDescription: `Container name: required for volumes, optional for env vars`,
																				Optional:            true,
																			},
																			"divisor": schema.StringAttribute{
																				MarkdownDescription: `Specifies the output format of the exposed resources, defaults to "1"`,
																				Optional:            true,
																			},
																			"resource": schema.StringAttribute{
																				MarkdownDescription: `Required: resource to select`,
																				Optional:            true,
																			},
																		},
																	},
																	"secret_key_ref": schema.SingleNestedAttribute{
																		MarkdownDescription: `Selects a key of a secret in the pod's namespace`,
																		Optional:            true,

																		Attributes: map[string]schema.Attribute{
																			"key": schema.StringAttribute{
																				MarkdownDescription: `The key of the secret to select from.  Must be a valid secret key.`,
																				Optional:            true,
																			},
																			"name": schema.StringAttribute{
																				MarkdownDescription: `Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names`,
																				Optional:            true,
																			},
																			"optional": schema.BoolAttribute{
																				MarkdownDescription: `Specify whether the Secret or its key must be defined`,
																				Optional:            true,
																			},
																		},
																	},
																},
															},
														},
													},
												},
												"env_from": schema.ListNestedAttribute{
													MarkdownDescription: `List of sources to populate environment variables in the container. The keys defined within a source must be a C_IDENTIFIER. All invalid keys will be reported as an event when the container is starting. When a key exists in multiple sources, the value associated with the last source will take precedence. Values defined by an Env with a duplicate key will take precedence. Cannot be updated.`,
													Optional:            true,

													NestedObject: schema.NestedAttributeObject{
														Attributes: map[string]schema.Attribute{
															"config_map_ref": schema.SingleNestedAttribute{
																MarkdownDescription: `The ConfigMap to select from`,
																Optional:            true,

																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		MarkdownDescription: `Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names`,
																		Optional:            true,
																	},
																	"optional": schema.BoolAttribute{
																		MarkdownDescription: `Specify whether the ConfigMap must be defined`,
																		Optional:            true,
																	},
																},
															},
															"prefix": schema.StringAttribute{
																MarkdownDescription: `An optional identifier to prepend to each key in the ConfigMap. Must be a C_IDENTIFIER.`,
																Optional:            true,
															},
															"secret_ref": schema.SingleNestedAttribute{
																MarkdownDescription: `The Secret to select from`,
																Optional:            true,

																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		MarkdownDescription: `Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names`,
																		Optional:            true,
																	},
																	"optional": schema.BoolAttribute{
																		MarkdownDescription: `Specify whether the Secret must be defined`,
																		Optional:            true,
																	},
																},
															},
														},
													},
												},
												"image": schema.StringAttribute{
													MarkdownDescription: `Container image name. More info: https://kubernetes.io/docs/concepts/containers/images`,
													Optional:            true,
												},
												"image_pull_policy": schema.StringAttribute{
													MarkdownDescription: `Image pull policy. One of Always, Never, IfNotPresent. Defaults to Always if :latest tag is specified, or IfNotPresent otherwise. Cannot be updated. More info: https://kubernetes.io/docs/concepts/containers/images#updating-images`,
													Optional:            true,
												},
												"lifecycle": schema.SingleNestedAttribute{
													MarkdownDescription: `Lifecycle is not allowed for ephemeral containers.`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"post_start": schema.SingleNestedAttribute{
															MarkdownDescription: `PostStart is called immediately after a container is created. If the handler fails, the container is terminated and restarted according to its restart policy. Other management of the container blocks until the hook completes. More info: https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#container-hooks`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"exec": schema.SingleNestedAttribute{
																	MarkdownDescription: `Exec specifies the action to take.`,
																	Optional:            true,

																	Attributes: map[string]schema.Attribute{
																		"command": schema.ListAttribute{
																			MarkdownDescription: `Command is the command line to execute inside the container, the working directory for the command  is root ('/') in the container's filesystem. The command is simply exec'd, it is not run inside a shell, so traditional shell instructions ('|', etc) won't work. To use a shell, you need to explicitly call out to that shell. Exit status of 0 is treated as live/healthy and non-zero is unhealthy.`,
																			ElementType:         types.StringType,
																			Optional:            true,
																		},
																	},
																},
																"http_get": schema.SingleNestedAttribute{
																	MarkdownDescription: `HTTPGet specifies the http request to perform.`,
																	Optional:            true,

																	Attributes: map[string]schema.Attribute{
																		"host": schema.StringAttribute{
																			MarkdownDescription: `Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.`,
																			Optional:            true,
																		},
																		"http_headers": schema.ListNestedAttribute{
																			MarkdownDescription: `Custom headers to set in the request. HTTP allows repeated headers.`,
																			Optional:            true,

																			NestedObject: schema.NestedAttributeObject{
																				Attributes: map[string]schema.Attribute{
																					"name": schema.StringAttribute{
																						MarkdownDescription: `The header field name. This will be canonicalized upon output, so case-variant names will be understood as the same header.`,
																						Optional:            true,
																					},
																					"value": schema.StringAttribute{
																						MarkdownDescription: `The header field value`,
																						Optional:            true,
																					},
																				},
																			},
																		},
																		"path": schema.StringAttribute{
																			MarkdownDescription: `Path to access on the HTTP server.`,
																			Optional:            true,
																		},
																		"port": schema.StringAttribute{
																			MarkdownDescription: `Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
																			Optional:            true,
																		},
																		"scheme": schema.StringAttribute{
																			MarkdownDescription: `Scheme to use for connecting to the host. Defaults to HTTP.`,
																			Optional:            true,
																		},
																	},
																},
																"tcp_socket": schema.SingleNestedAttribute{
																	MarkdownDescription: `Deprecated. TCPSocket is NOT supported as a LifecycleHandler and kept for the backward compatibility. There are no validation of this field and lifecycle hooks will fail in runtime when tcp handler is specified.`,
																	Optional:            true,

																	Attributes: map[string]schema.Attribute{
																		"host": schema.StringAttribute{
																			MarkdownDescription: `Optional: Host name to connect to, defaults to the pod IP.`,
																			Optional:            true,
																		},
																		"port": schema.StringAttribute{
																			MarkdownDescription: `Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
																			Optional:            true,
																		},
																	},
																},
															},
														},
														"pre_stop": schema.SingleNestedAttribute{
															MarkdownDescription: `PreStop is called immediately before a container is terminated due to an API request or management event such as liveness/startup probe failure, preemption, resource contention, etc. The handler is not called if the container crashes or exits. The Pod's termination grace period countdown begins before the PreStop hook is executed. Regardless of the outcome of the handler, the container will eventually terminate within the Pod's termination grace period (unless delayed by finalizers). Other management of the container blocks until the hook completes or until the termination grace period is reached. More info: https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#container-hooks`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"exec": schema.SingleNestedAttribute{
																	MarkdownDescription: `Exec specifies the action to take.`,
																	Optional:            true,

																	Attributes: map[string]schema.Attribute{
																		"command": schema.ListAttribute{
																			MarkdownDescription: `Command is the command line to execute inside the container, the working directory for the command  is root ('/') in the container's filesystem. The command is simply exec'd, it is not run inside a shell, so traditional shell instructions ('|', etc) won't work. To use a shell, you need to explicitly call out to that shell. Exit status of 0 is treated as live/healthy and non-zero is unhealthy.`,
																			ElementType:         types.StringType,
																			Optional:            true,
																		},
																	},
																},
																"http_get": schema.SingleNestedAttribute{
																	MarkdownDescription: `HTTPGet specifies the http request to perform.`,
																	Optional:            true,

																	Attributes: map[string]schema.Attribute{
																		"host": schema.StringAttribute{
																			MarkdownDescription: `Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.`,
																			Optional:            true,
																		},
																		"http_headers": schema.ListNestedAttribute{
																			MarkdownDescription: `Custom headers to set in the request. HTTP allows repeated headers.`,
																			Optional:            true,

																			NestedObject: schema.NestedAttributeObject{
																				Attributes: map[string]schema.Attribute{
																					"name": schema.StringAttribute{
																						MarkdownDescription: `The header field name. This will be canonicalized upon output, so case-variant names will be understood as the same header.`,
																						Optional:            true,
																					},
																					"value": schema.StringAttribute{
																						MarkdownDescription: `The header field value`,
																						Optional:            true,
																					},
																				},
																			},
																		},
																		"path": schema.StringAttribute{
																			MarkdownDescription: `Path to access on the HTTP server.`,
																			Optional:            true,
																		},
																		"port": schema.StringAttribute{
																			MarkdownDescription: `Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
																			Optional:            true,
																		},
																		"scheme": schema.StringAttribute{
																			MarkdownDescription: `Scheme to use for connecting to the host. Defaults to HTTP.`,
																			Optional:            true,
																		},
																	},
																},
																"tcp_socket": schema.SingleNestedAttribute{
																	MarkdownDescription: `Deprecated. TCPSocket is NOT supported as a LifecycleHandler and kept for the backward compatibility. There are no validation of this field and lifecycle hooks will fail in runtime when tcp handler is specified.`,
																	Optional:            true,

																	Attributes: map[string]schema.Attribute{
																		"host": schema.StringAttribute{
																			MarkdownDescription: `Optional: Host name to connect to, defaults to the pod IP.`,
																			Optional:            true,
																		},
																		"port": schema.StringAttribute{
																			MarkdownDescription: `Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
																			Optional:            true,
																		},
																	},
																},
															},
														},
													},
												},
												"liveness_probe": schema.SingleNestedAttribute{
													MarkdownDescription: `Probes are not allowed for ephemeral containers.`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"exec": schema.SingleNestedAttribute{
															MarkdownDescription: `Exec specifies the action to take.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"command": schema.ListAttribute{
																	MarkdownDescription: `Command is the command line to execute inside the container, the working directory for the command  is root ('/') in the container's filesystem. The command is simply exec'd, it is not run inside a shell, so traditional shell instructions ('|', etc) won't work. To use a shell, you need to explicitly call out to that shell. Exit status of 0 is treated as live/healthy and non-zero is unhealthy.`,
																	ElementType:         types.StringType,
																	Optional:            true,
																},
															},
														},
														"failure_threshold": schema.Int64Attribute{
															MarkdownDescription: `Minimum consecutive failures for the probe to be considered failed after having succeeded. Defaults to 3. Minimum value is 1.`,
															Optional:            true,
														},
														"grpc": schema.SingleNestedAttribute{
															MarkdownDescription: `GRPC specifies an action involving a GRPC port.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"port": schema.Int64Attribute{
																	MarkdownDescription: `Port number of the gRPC service. Number must be in the range 1 to 65535.`,
																	Optional:            true,
																},
																"service": schema.StringAttribute{
																	MarkdownDescription: `Service is the name of the service to place in the gRPC HealthCheckRequest (see https://github.com/grpc/grpc/blob/master/doc/health-checking.md).

If this is not specified, the default behavior is defined by gRPC.`,
																	Optional: true,
																},
															},
														},
														"http_get": schema.SingleNestedAttribute{
															MarkdownDescription: `HTTPGet specifies the http request to perform.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"host": schema.StringAttribute{
																	MarkdownDescription: `Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.`,
																	Optional:            true,
																},
																"http_headers": schema.ListNestedAttribute{
																	MarkdownDescription: `Custom headers to set in the request. HTTP allows repeated headers.`,
																	Optional:            true,

																	NestedObject: schema.NestedAttributeObject{
																		Attributes: map[string]schema.Attribute{
																			"name": schema.StringAttribute{
																				MarkdownDescription: `The header field name. This will be canonicalized upon output, so case-variant names will be understood as the same header.`,
																				Optional:            true,
																			},
																			"value": schema.StringAttribute{
																				MarkdownDescription: `The header field value`,
																				Optional:            true,
																			},
																		},
																	},
																},
																"path": schema.StringAttribute{
																	MarkdownDescription: `Path to access on the HTTP server.`,
																	Optional:            true,
																},
																"port": schema.StringAttribute{
																	MarkdownDescription: `Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
																	Optional:            true,
																},
																"scheme": schema.StringAttribute{
																	MarkdownDescription: `Scheme to use for connecting to the host. Defaults to HTTP.`,
																	Optional:            true,
																},
															},
														},
														"initial_delay_seconds": schema.Int64Attribute{
															MarkdownDescription: `Number of seconds after the container has started before liveness probes are initiated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes`,
															Optional:            true,
														},
														"period_seconds": schema.Int64Attribute{
															MarkdownDescription: `How often (in seconds) to perform the probe. Default to 10 seconds. Minimum value is 1.`,
															Optional:            true,
														},
														"success_threshold": schema.Int64Attribute{
															MarkdownDescription: `Minimum consecutive successes for the probe to be considered successful after having failed. Defaults to 1. Must be 1 for liveness and startup. Minimum value is 1.`,
															Optional:            true,
														},
														"tcp_socket": schema.SingleNestedAttribute{
															MarkdownDescription: `TCPSocket specifies an action involving a TCP port.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"host": schema.StringAttribute{
																	MarkdownDescription: `Optional: Host name to connect to, defaults to the pod IP.`,
																	Optional:            true,
																},
																"port": schema.StringAttribute{
																	MarkdownDescription: `Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
																	Optional:            true,
																},
															},
														},
														"termination_grace_period_seconds": schema.Int64Attribute{
															MarkdownDescription: `Optional duration in seconds the pod needs to terminate gracefully upon probe failure. The grace period is the duration in seconds after the processes running in the pod are sent a termination signal and the time when the processes are forcibly halted with a kill signal. Set this value longer than the expected cleanup time for your process. If this value is nil, the pod's terminationGracePeriodSeconds will be used. Otherwise, this value overrides the value provided by the pod spec. Value must be non-negative integer. The value zero indicates stop immediately via the kill signal (no opportunity to shut down). This is a beta field and requires enabling ProbeTerminationGracePeriod feature gate. Minimum value is 1. spec.terminationGracePeriodSeconds is used if unset.`,
															Optional:            true,
														},
														"timeout_seconds": schema.Int64Attribute{
															MarkdownDescription: `Number of seconds after which the probe times out. Defaults to 1 second. Minimum value is 1. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes`,
															Optional:            true,
														},
													},
												},
												"name": schema.StringAttribute{
													MarkdownDescription: `Name of the ephemeral container specified as a DNS_LABEL. This name must be unique among all containers, init containers and ephemeral containers.`,
													Optional:            true,
												},
												"ports": schema.ListNestedAttribute{
													MarkdownDescription: `Ports are not allowed for ephemeral containers.`,
													Optional:            true,

													NestedObject: schema.NestedAttributeObject{
														Attributes: map[string]schema.Attribute{
															"container_port": schema.Int64Attribute{
																MarkdownDescription: `Number of port to expose on the pod's IP address. This must be a valid port number, 0 < x < 65536.`,
																Optional:            true,
															},
															"host_ip": schema.StringAttribute{
																MarkdownDescription: `What host IP to bind the external port to.`,
																Optional:            true,
															},
															"host_port": schema.Int64Attribute{
																MarkdownDescription: `Number of port to expose on the host. If specified, this must be a valid port number, 0 < x < 65536. If HostNetwork is specified, this must match ContainerPort. Most containers do not need this.`,
																Optional:            true,
															},
															"name": schema.StringAttribute{
																MarkdownDescription: `If specified, this must be an IANA_SVC_NAME and unique within the pod. Each named port in a pod must have a unique name. Name for the port that can be referred to by services.`,
																Optional:            true,
															},
															"protocol": schema.StringAttribute{
																MarkdownDescription: `Protocol for port. Must be UDP, TCP, or SCTP. Defaults to "TCP".`,
																Optional:            true,
															},
														},
													},
												},
												"readiness_probe": schema.SingleNestedAttribute{
													MarkdownDescription: `Probes are not allowed for ephemeral containers.`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"exec": schema.SingleNestedAttribute{
															MarkdownDescription: `Exec specifies the action to take.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"command": schema.ListAttribute{
																	MarkdownDescription: `Command is the command line to execute inside the container, the working directory for the command  is root ('/') in the container's filesystem. The command is simply exec'd, it is not run inside a shell, so traditional shell instructions ('|', etc) won't work. To use a shell, you need to explicitly call out to that shell. Exit status of 0 is treated as live/healthy and non-zero is unhealthy.`,
																	ElementType:         types.StringType,
																	Optional:            true,
																},
															},
														},
														"failure_threshold": schema.Int64Attribute{
															MarkdownDescription: `Minimum consecutive failures for the probe to be considered failed after having succeeded. Defaults to 3. Minimum value is 1.`,
															Optional:            true,
														},
														"grpc": schema.SingleNestedAttribute{
															MarkdownDescription: `GRPC specifies an action involving a GRPC port.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"port": schema.Int64Attribute{
																	MarkdownDescription: `Port number of the gRPC service. Number must be in the range 1 to 65535.`,
																	Optional:            true,
																},
																"service": schema.StringAttribute{
																	MarkdownDescription: `Service is the name of the service to place in the gRPC HealthCheckRequest (see https://github.com/grpc/grpc/blob/master/doc/health-checking.md).

If this is not specified, the default behavior is defined by gRPC.`,
																	Optional: true,
																},
															},
														},
														"http_get": schema.SingleNestedAttribute{
															MarkdownDescription: `HTTPGet specifies the http request to perform.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"host": schema.StringAttribute{
																	MarkdownDescription: `Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.`,
																	Optional:            true,
																},
																"http_headers": schema.ListNestedAttribute{
																	MarkdownDescription: `Custom headers to set in the request. HTTP allows repeated headers.`,
																	Optional:            true,

																	NestedObject: schema.NestedAttributeObject{
																		Attributes: map[string]schema.Attribute{
																			"name": schema.StringAttribute{
																				MarkdownDescription: `The header field name. This will be canonicalized upon output, so case-variant names will be understood as the same header.`,
																				Optional:            true,
																			},
																			"value": schema.StringAttribute{
																				MarkdownDescription: `The header field value`,
																				Optional:            true,
																			},
																		},
																	},
																},
																"path": schema.StringAttribute{
																	MarkdownDescription: `Path to access on the HTTP server.`,
																	Optional:            true,
																},
																"port": schema.StringAttribute{
																	MarkdownDescription: `Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
																	Optional:            true,
																},
																"scheme": schema.StringAttribute{
																	MarkdownDescription: `Scheme to use for connecting to the host. Defaults to HTTP.`,
																	Optional:            true,
																},
															},
														},
														"initial_delay_seconds": schema.Int64Attribute{
															MarkdownDescription: `Number of seconds after the container has started before liveness probes are initiated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes`,
															Optional:            true,
														},
														"period_seconds": schema.Int64Attribute{
															MarkdownDescription: `How often (in seconds) to perform the probe. Default to 10 seconds. Minimum value is 1.`,
															Optional:            true,
														},
														"success_threshold": schema.Int64Attribute{
															MarkdownDescription: `Minimum consecutive successes for the probe to be considered successful after having failed. Defaults to 1. Must be 1 for liveness and startup. Minimum value is 1.`,
															Optional:            true,
														},
														"tcp_socket": schema.SingleNestedAttribute{
															MarkdownDescription: `TCPSocket specifies an action involving a TCP port.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"host": schema.StringAttribute{
																	MarkdownDescription: `Optional: Host name to connect to, defaults to the pod IP.`,
																	Optional:            true,
																},
																"port": schema.StringAttribute{
																	MarkdownDescription: `Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
																	Optional:            true,
																},
															},
														},
														"termination_grace_period_seconds": schema.Int64Attribute{
															MarkdownDescription: `Optional duration in seconds the pod needs to terminate gracefully upon probe failure. The grace period is the duration in seconds after the processes running in the pod are sent a termination signal and the time when the processes are forcibly halted with a kill signal. Set this value longer than the expected cleanup time for your process. If this value is nil, the pod's terminationGracePeriodSeconds will be used. Otherwise, this value overrides the value provided by the pod spec. Value must be non-negative integer. The value zero indicates stop immediately via the kill signal (no opportunity to shut down). This is a beta field and requires enabling ProbeTerminationGracePeriod feature gate. Minimum value is 1. spec.terminationGracePeriodSeconds is used if unset.`,
															Optional:            true,
														},
														"timeout_seconds": schema.Int64Attribute{
															MarkdownDescription: `Number of seconds after which the probe times out. Defaults to 1 second. Minimum value is 1. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes`,
															Optional:            true,
														},
													},
												},
												"resize_policy": schema.ListNestedAttribute{
													MarkdownDescription: `Resources resize policy for the container.`,
													Optional:            true,

													NestedObject: schema.NestedAttributeObject{
														Attributes: map[string]schema.Attribute{
															"resource_name": schema.StringAttribute{
																MarkdownDescription: `Name of the resource to which this resource resize policy applies. Supported values: cpu, memory.`,
																Optional:            true,
															},
															"restart_policy": schema.StringAttribute{
																MarkdownDescription: `Restart policy to apply when specified resource is resized. If not specified, it defaults to NotRequired.`,
																Optional:            true,
															},
														},
													},
												},
												"resources": schema.SingleNestedAttribute{
													MarkdownDescription: `Resources are not allowed for ephemeral containers. Ephemeral containers use spare resources already allocated to the pod.`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"claims": schema.ListNestedAttribute{
															MarkdownDescription: `Claims lists the names of resources, defined in spec.resourceClaims, that are used by this container.

This is an alpha field and requires enabling the DynamicResourceAllocation feature gate.

This field is immutable. It can only be set for containers.`,
															Optional: true,

															NestedObject: schema.NestedAttributeObject{
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		MarkdownDescription: `Name must match the name of one entry in pod.spec.resourceClaims of the Pod where this field is used. It makes that resource available inside a container.`,
																		Optional:            true,
																	},
																},
															},
														},
														"limits": schema.MapAttribute{
															MarkdownDescription: `Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/`,
															ElementType:         types.StringType,
															Optional:            true,
														},
														"requests": schema.MapAttribute{
															MarkdownDescription: `Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. Requests cannot exceed Limits. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/`,
															ElementType:         types.StringType,
															Optional:            true,
														},
													},
												},
												"restart_policy": schema.StringAttribute{
													MarkdownDescription: `Restart policy for the container to manage the restart behavior of each container within a pod. This may only be set for init containers. You cannot set this field on ephemeral containers.`,
													Optional:            true,
												},
												"security_context": schema.SingleNestedAttribute{
													MarkdownDescription: `Optional: SecurityContext defines the security options the ephemeral container should be run with. If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext.`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"allow_privilege_escalation": schema.BoolAttribute{
															MarkdownDescription: `AllowPrivilegeEscalation controls whether a process can gain more privileges than its parent process. This bool directly controls if the no_new_privs flag will be set on the container process. AllowPrivilegeEscalation is true always when the container is: 1) run as Privileged 2) has CAP_SYS_ADMIN Note that this field cannot be set when spec.os.name is windows.`,
															Optional:            true,
														},
														"capabilities": schema.SingleNestedAttribute{
															MarkdownDescription: `The capabilities to add/drop when running containers. Defaults to the default set of capabilities granted by the container runtime. Note that this field cannot be set when spec.os.name is windows.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"add": schema.ListAttribute{
																	MarkdownDescription: `Added capabilities`,
																	ElementType:         types.StringType,
																	Optional:            true,
																},
																"drop": schema.ListAttribute{
																	MarkdownDescription: `Removed capabilities`,
																	ElementType:         types.StringType,
																	Optional:            true,
																},
															},
														},
														"privileged": schema.BoolAttribute{
															MarkdownDescription: `Run container in privileged mode. Processes in privileged containers are essentially equivalent to root on the host. Defaults to false. Note that this field cannot be set when spec.os.name is windows.`,
															Optional:            true,
														},
														"proc_mount": schema.StringAttribute{
															MarkdownDescription: `procMount denotes the type of proc mount to use for the containers. The default is DefaultProcMount which uses the container runtime defaults for readonly paths and masked paths. This requires the ProcMountType feature flag to be enabled. Note that this field cannot be set when spec.os.name is windows.`,
															Optional:            true,
														},
														"read_only_root_filesystem": schema.BoolAttribute{
															MarkdownDescription: `Whether this container has a read-only root filesystem. Default is false. Note that this field cannot be set when spec.os.name is windows.`,
															Optional:            true,
														},
														"run_as_group": schema.Int64Attribute{
															MarkdownDescription: `The GID to run the entrypoint of the container process. Uses runtime default if unset. May also be set in PodSecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. Note that this field cannot be set when spec.os.name is windows.`,
															Optional:            true,
														},
														"run_as_non_root": schema.BoolAttribute{
															MarkdownDescription: `Indicates that the container must run as a non-root user. If true, the Kubelet will validate the image at runtime to ensure that it does not run as UID 0 (root) and fail to start the container if it does. If unset or false, no such validation will be performed. May also be set in PodSecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.`,
															Optional:            true,
														},
														"run_as_user": schema.Int64Attribute{
															MarkdownDescription: `The UID to run the entrypoint of the container process. Defaults to user specified in image metadata if unspecified. May also be set in PodSecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. Note that this field cannot be set when spec.os.name is windows.`,
															Optional:            true,
														},
														"se_linux_options": schema.SingleNestedAttribute{
															MarkdownDescription: `The SELinux context to be applied to the container. If unspecified, the container runtime will allocate a random SELinux context for each container.  May also be set in PodSecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. Note that this field cannot be set when spec.os.name is windows.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"level": schema.StringAttribute{
																	MarkdownDescription: `Level is SELinux level label that applies to the container.`,
																	Optional:            true,
																},
																"role": schema.StringAttribute{
																	MarkdownDescription: `Role is a SELinux role label that applies to the container.`,
																	Optional:            true,
																},
																"type": schema.StringAttribute{
																	MarkdownDescription: `Type is a SELinux type label that applies to the container.`,
																	Optional:            true,
																},
																"user": schema.StringAttribute{
																	MarkdownDescription: `User is a SELinux user label that applies to the container.`,
																	Optional:            true,
																},
															},
														},
														"seccomp_profile": schema.SingleNestedAttribute{
															MarkdownDescription: `The seccomp options to use by this container. If seccomp options are provided at both the pod & container level, the container options override the pod options. Note that this field cannot be set when spec.os.name is windows.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"localhost_profile": schema.StringAttribute{
																	MarkdownDescription: `localhostProfile indicates a profile defined in a file on the node should be used. The profile must be preconfigured on the node to work. Must be a descending path, relative to the kubelet's configured seccomp profile location. Must be set if type is "Localhost". Must NOT be set for any other type.`,
																	Optional:            true,
																},
																"type": schema.StringAttribute{
																	MarkdownDescription: `type indicates which kind of seccomp profile will be applied. Valid options are:

Localhost - a profile defined in a file on the node should be used. RuntimeDefault - the container runtime default profile should be used. Unconfined - no profile should be applied.`,
																	Optional: true,
																},
															},
														},
														"windows_options": schema.SingleNestedAttribute{
															MarkdownDescription: `The Windows specific settings applied to all containers. If unspecified, the options from the PodSecurityContext will be used. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. Note that this field cannot be set when spec.os.name is linux.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"gmsa_credential_spec": schema.StringAttribute{
																	MarkdownDescription: `GMSACredentialSpec is where the GMSA admission webhook (https://github.com/kubernetes-sigs/windows-gmsa) inlines the contents of the GMSA credential spec named by the GMSACredentialSpecName field.`,
																	Optional:            true,
																},
																"gmsa_credential_spec_name": schema.StringAttribute{
																	MarkdownDescription: `GMSACredentialSpecName is the name of the GMSA credential spec to use.`,
																	Optional:            true,
																},
																"host_process": schema.BoolAttribute{
																	MarkdownDescription: `HostProcess determines if a container should be run as a 'Host Process' container. All of a Pod's containers must have the same effective HostProcess value (it is not allowed to have a mix of HostProcess containers and non-HostProcess containers). In addition, if HostProcess is true then HostNetwork must also be set to true.`,
																	Optional:            true,
																},
																"run_as_user_name": schema.StringAttribute{
																	MarkdownDescription: `The UserName in Windows to run the entrypoint of the container process. Defaults to the user specified in image metadata if unspecified. May also be set in PodSecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.`,
																	Optional:            true,
																},
															},
														},
													},
												},
												"startup_probe": schema.SingleNestedAttribute{
													MarkdownDescription: `Probes are not allowed for ephemeral containers.`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"exec": schema.SingleNestedAttribute{
															MarkdownDescription: `Exec specifies the action to take.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"command": schema.ListAttribute{
																	MarkdownDescription: `Command is the command line to execute inside the container, the working directory for the command  is root ('/') in the container's filesystem. The command is simply exec'd, it is not run inside a shell, so traditional shell instructions ('|', etc) won't work. To use a shell, you need to explicitly call out to that shell. Exit status of 0 is treated as live/healthy and non-zero is unhealthy.`,
																	ElementType:         types.StringType,
																	Optional:            true,
																},
															},
														},
														"failure_threshold": schema.Int64Attribute{
															MarkdownDescription: `Minimum consecutive failures for the probe to be considered failed after having succeeded. Defaults to 3. Minimum value is 1.`,
															Optional:            true,
														},
														"grpc": schema.SingleNestedAttribute{
															MarkdownDescription: `GRPC specifies an action involving a GRPC port.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"port": schema.Int64Attribute{
																	MarkdownDescription: `Port number of the gRPC service. Number must be in the range 1 to 65535.`,
																	Optional:            true,
																},
																"service": schema.StringAttribute{
																	MarkdownDescription: `Service is the name of the service to place in the gRPC HealthCheckRequest (see https://github.com/grpc/grpc/blob/master/doc/health-checking.md).

If this is not specified, the default behavior is defined by gRPC.`,
																	Optional: true,
																},
															},
														},
														"http_get": schema.SingleNestedAttribute{
															MarkdownDescription: `HTTPGet specifies the http request to perform.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"host": schema.StringAttribute{
																	MarkdownDescription: `Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.`,
																	Optional:            true,
																},
																"http_headers": schema.ListNestedAttribute{
																	MarkdownDescription: `Custom headers to set in the request. HTTP allows repeated headers.`,
																	Optional:            true,

																	NestedObject: schema.NestedAttributeObject{
																		Attributes: map[string]schema.Attribute{
																			"name": schema.StringAttribute{
																				MarkdownDescription: `The header field name. This will be canonicalized upon output, so case-variant names will be understood as the same header.`,
																				Optional:            true,
																			},
																			"value": schema.StringAttribute{
																				MarkdownDescription: `The header field value`,
																				Optional:            true,
																			},
																		},
																	},
																},
																"path": schema.StringAttribute{
																	MarkdownDescription: `Path to access on the HTTP server.`,
																	Optional:            true,
																},
																"port": schema.StringAttribute{
																	MarkdownDescription: `Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
																	Optional:            true,
																},
																"scheme": schema.StringAttribute{
																	MarkdownDescription: `Scheme to use for connecting to the host. Defaults to HTTP.`,
																	Optional:            true,
																},
															},
														},
														"initial_delay_seconds": schema.Int64Attribute{
															MarkdownDescription: `Number of seconds after the container has started before liveness probes are initiated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes`,
															Optional:            true,
														},
														"period_seconds": schema.Int64Attribute{
															MarkdownDescription: `How often (in seconds) to perform the probe. Default to 10 seconds. Minimum value is 1.`,
															Optional:            true,
														},
														"success_threshold": schema.Int64Attribute{
															MarkdownDescription: `Minimum consecutive successes for the probe to be considered successful after having failed. Defaults to 1. Must be 1 for liveness and startup. Minimum value is 1.`,
															Optional:            true,
														},
														"tcp_socket": schema.SingleNestedAttribute{
															MarkdownDescription: `TCPSocket specifies an action involving a TCP port.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"host": schema.StringAttribute{
																	MarkdownDescription: `Optional: Host name to connect to, defaults to the pod IP.`,
																	Optional:            true,
																},
																"port": schema.StringAttribute{
																	MarkdownDescription: `Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
																	Optional:            true,
																},
															},
														},
														"termination_grace_period_seconds": schema.Int64Attribute{
															MarkdownDescription: `Optional duration in seconds the pod needs to terminate gracefully upon probe failure. The grace period is the duration in seconds after the processes running in the pod are sent a termination signal and the time when the processes are forcibly halted with a kill signal. Set this value longer than the expected cleanup time for your process. If this value is nil, the pod's terminationGracePeriodSeconds will be used. Otherwise, this value overrides the value provided by the pod spec. Value must be non-negative integer. The value zero indicates stop immediately via the kill signal (no opportunity to shut down). This is a beta field and requires enabling ProbeTerminationGracePeriod feature gate. Minimum value is 1. spec.terminationGracePeriodSeconds is used if unset.`,
															Optional:            true,
														},
														"timeout_seconds": schema.Int64Attribute{
															MarkdownDescription: `Number of seconds after which the probe times out. Defaults to 1 second. Minimum value is 1. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes`,
															Optional:            true,
														},
													},
												},
												"stdin": schema.BoolAttribute{
													MarkdownDescription: `Whether this container should allocate a buffer for stdin in the container runtime. If this is not set, reads from stdin in the container will always result in EOF. Default is false.`,
													Optional:            true,
												},
												"stdin_once": schema.BoolAttribute{
													MarkdownDescription: `Whether the container runtime should close the stdin channel after it has been opened by a single attach. When stdin is true the stdin stream will remain open across multiple attach sessions. If stdinOnce is set to true, stdin is opened on container start, is empty until the first client attaches to stdin, and then remains open and accepts data until the client disconnects, at which time stdin is closed and remains closed until the container is restarted. If this flag is false, a container processes that reads from stdin will never receive an EOF. Default is false`,
													Optional:            true,
												},
												"target_container_name": schema.StringAttribute{
													MarkdownDescription: `If set, the name of the container from PodSpec that this ephemeral container targets. The ephemeral container will be run in the namespaces (IPC, PID, etc) of this container. If not set then the ephemeral container uses the namespaces configured in the Pod spec.

The container runtime must implement support for this feature. If the runtime does not support namespace targeting then the result of setting this field is undefined.`,
													Optional: true,
												},
												"termination_message_path": schema.StringAttribute{
													MarkdownDescription: `Optional: Path at which the file to which the container's termination message will be written is mounted into the container's filesystem. Message written is intended to be brief final status, such as an assertion failure message. Will be truncated by the node if greater than 4096 bytes. The total message length across all containers will be limited to 12kb. Defaults to /dev/termination-log. Cannot be updated.`,
													Optional:            true,
												},
												"termination_message_policy": schema.StringAttribute{
													MarkdownDescription: `Indicate how the termination message should be populated. File will use the contents of terminationMessagePath to populate the container status message on both success and failure. FallbackToLogsOnError will use the last chunk of container log output if the termination message file is empty and the container exited with an error. The log output is limited to 2048 bytes or 80 lines, whichever is smaller. Defaults to File. Cannot be updated.`,
													Optional:            true,
												},
												"tty": schema.BoolAttribute{
													MarkdownDescription: `Whether this container should allocate a TTY for itself, also requires 'stdin' to be true. Default is false.`,
													Optional:            true,
												},
												"volume_devices": schema.ListNestedAttribute{
													MarkdownDescription: `volumeDevices is the list of block devices to be used by the container.`,
													Optional:            true,

													NestedObject: schema.NestedAttributeObject{
														Attributes: map[string]schema.Attribute{
															"device_path": schema.StringAttribute{
																MarkdownDescription: `devicePath is the path inside of the container that the device will be mapped to.`,
																Optional:            true,
															},
															"name": schema.StringAttribute{
																MarkdownDescription: `name must match the name of a persistentVolumeClaim in the pod`,
																Optional:            true,
															},
														},
													},
												},
												"volume_mounts": schema.ListNestedAttribute{
													MarkdownDescription: `Pod volumes to mount into the container's filesystem. Subpath mounts are not allowed for ephemeral containers. Cannot be updated.`,
													Optional:            true,

													NestedObject: schema.NestedAttributeObject{
														Attributes: map[string]schema.Attribute{
															"mount_path": schema.StringAttribute{
																MarkdownDescription: `Path within the container at which the volume should be mounted.  Must not contain ':'.`,
																Optional:            true,
															},
															"mount_propagation": schema.StringAttribute{
																MarkdownDescription: `mountPropagation determines how mounts are propagated from the host to container and the other way around. When not set, MountPropagationNone is used. This field is beta in 1.10.`,
																Optional:            true,
															},
															"name": schema.StringAttribute{
																MarkdownDescription: `This must match the Name of a Volume.`,
																Optional:            true,
															},
															"read_only": schema.BoolAttribute{
																MarkdownDescription: `Mounted read-only if true, read-write otherwise (false or unspecified). Defaults to false.`,
																Optional:            true,
															},
															"sub_path": schema.StringAttribute{
																MarkdownDescription: `Path within the volume from which the container's volume should be mounted. Defaults to "" (volume's root).`,
																Optional:            true,
															},
															"sub_path_expr": schema.StringAttribute{
																MarkdownDescription: `Expanded path within the volume from which the container's volume should be mounted. Behaves similarly to SubPath but environment variable references $(VAR_NAME) are expanded using the container's environment. Defaults to "" (volume's root). SubPathExpr and SubPath are mutually exclusive.`,
																Optional:            true,
															},
														},
													},
												},
												"working_dir": schema.StringAttribute{
													MarkdownDescription: `Container's working directory. If not specified, the container runtime's default will be used, which might be configured in the container image. Cannot be updated.`,
													Optional:            true,
												},
											},
										},
									},
									"host_aliases": schema.ListNestedAttribute{
										MarkdownDescription: `HostAliases is an optional list of hosts and IPs that will be injected into the pod's hosts file if specified. This is only valid for non-hostNetwork pods.`,
										Optional:            true,

										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"hostnames": schema.ListAttribute{
													MarkdownDescription: `Hostnames for the above IP address.`,
													ElementType:         types.StringType,
													Optional:            true,
												},
												"ip": schema.StringAttribute{
													MarkdownDescription: `IP address of the host file entry.`,
													Optional:            true,
												},
											},
										},
									},
									"host_ipc": schema.BoolAttribute{
										MarkdownDescription: `Use the host's ipc namespace. Optional: Default to false.`,
										Optional:            true,
									},
									"host_network": schema.BoolAttribute{
										MarkdownDescription: `Host networking requested for this pod. Use the host's network namespace. If this option is set, the ports that will be used must be specified. Default to false.`,
										Optional:            true,
									},
									"host_pid": schema.BoolAttribute{
										MarkdownDescription: `Use the host's pid namespace. Optional: Default to false.`,
										Optional:            true,
									},
									"host_users": schema.BoolAttribute{
										MarkdownDescription: `Use the host's user namespace. Optional: Default to true. If set to true or not present, the pod will be run in the host user namespace, useful for when the pod needs a feature only available to the host user namespace, such as loading a kernel module with CAP_SYS_MODULE. When set to false, a new userns is created for the pod. Setting false is useful for mitigating container breakout vulnerabilities even allowing users to run their containers as root without actually having root privileges on the host. This field is alpha-level and is only honored by servers that enable the UserNamespacesSupport feature.`,
										Optional:            true,
									},
									"hostname": schema.StringAttribute{
										MarkdownDescription: `Specifies the hostname of the Pod If not specified, the pod's hostname will be set to a system-defined value.`,
										Optional:            true,
									},
									"image_pull_secrets": schema.ListNestedAttribute{
										MarkdownDescription: `ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec. If specified, these secrets will be passed to individual puller implementations for them to use. More info: https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod`,
										Optional:            true,

										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"name": schema.StringAttribute{
													MarkdownDescription: `Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names`,
													Optional:            true,
												},
											},
										},
									},
									"init_containers": schema.ListNestedAttribute{
										MarkdownDescription: `List of initialization containers belonging to the pod. Init containers are executed in order prior to containers being started. If any init container fails, the pod is considered to have failed and is handled according to its restartPolicy. The name for an init container or normal container must be unique among all containers. Init containers may not have Lifecycle actions, Readiness probes, Liveness probes, or Startup probes. The resourceRequirements of an init container are taken into account during scheduling by finding the highest request/limit for each resource type, and then using the max of of that value or the sum of the normal containers. Limits are applied to init containers in a similar fashion. Init containers cannot currently be added or removed. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/init-containers/`,
										Optional:            true,

										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"args": schema.ListAttribute{
													MarkdownDescription: `Arguments to the entrypoint. The container image's CMD is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell`,
													ElementType:         types.StringType,
													Optional:            true,
												},
												"command": schema.ListAttribute{
													MarkdownDescription: `Entrypoint array. Not executed within a shell. The container image's ENTRYPOINT is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell`,
													ElementType:         types.StringType,
													Optional:            true,
												},
												"env": schema.ListNestedAttribute{
													MarkdownDescription: `List of environment variables to set in the container. Cannot be updated.`,
													Optional:            true,

													NestedObject: schema.NestedAttributeObject{
														Attributes: map[string]schema.Attribute{
															"name": schema.StringAttribute{
																MarkdownDescription: `Name of the environment variable. Must be a C_IDENTIFIER.`,
																Optional:            true,
															},
															"value": schema.StringAttribute{
																MarkdownDescription: `Variable references $(VAR_NAME) are expanded using the previously defined environment variables in the container and any service environment variables. If a variable cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless of whether the variable exists or not. Defaults to "".`,
																Optional:            true,
															},
															"value_from": schema.SingleNestedAttribute{
																MarkdownDescription: `Source for the environment variable's value. Cannot be used if value is not empty.`,
																Optional:            true,

																Attributes: map[string]schema.Attribute{
																	"config_map_key_ref": schema.SingleNestedAttribute{
																		MarkdownDescription: `Selects a key of a ConfigMap.`,
																		Optional:            true,

																		Attributes: map[string]schema.Attribute{
																			"key": schema.StringAttribute{
																				MarkdownDescription: `The key to select.`,
																				Optional:            true,
																			},
																			"name": schema.StringAttribute{
																				MarkdownDescription: `Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names`,
																				Optional:            true,
																			},
																			"optional": schema.BoolAttribute{
																				MarkdownDescription: `Specify whether the ConfigMap or its key must be defined`,
																				Optional:            true,
																			},
																		},
																	},
																	"field_ref": schema.SingleNestedAttribute{
																		MarkdownDescription: `Selects a field of the pod: supports metadata.name, metadata.namespace, metadata.labels['<KEY>'], metadata.annotations['<KEY>'], spec.nodeName, spec.serviceAccountName, status.hostIP, status.podIP, status.podIPs.`,
																		Optional:            true,

																		Attributes: map[string]schema.Attribute{
																			"api_version": schema.StringAttribute{
																				MarkdownDescription: `Version of the schema the FieldPath is written in terms of, defaults to "v1".`,
																				Optional:            true,
																			},
																			"field_path": schema.StringAttribute{
																				MarkdownDescription: `Path of the field to select in the specified API version.`,
																				Optional:            true,
																			},
																		},
																	},
																	"resource_field_ref": schema.SingleNestedAttribute{
																		MarkdownDescription: `Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.ephemeral-storage) are currently supported.`,
																		Optional:            true,

																		Attributes: map[string]schema.Attribute{
																			"container_name": schema.StringAttribute{
																				MarkdownDescription: `Container name: required for volumes, optional for env vars`,
																				Optional:            true,
																			},
																			"divisor": schema.StringAttribute{
																				MarkdownDescription: `Specifies the output format of the exposed resources, defaults to "1"`,
																				Optional:            true,
																			},
																			"resource": schema.StringAttribute{
																				MarkdownDescription: `Required: resource to select`,
																				Optional:            true,
																			},
																		},
																	},
																	"secret_key_ref": schema.SingleNestedAttribute{
																		MarkdownDescription: `Selects a key of a secret in the pod's namespace`,
																		Optional:            true,

																		Attributes: map[string]schema.Attribute{
																			"key": schema.StringAttribute{
																				MarkdownDescription: `The key of the secret to select from.  Must be a valid secret key.`,
																				Optional:            true,
																			},
																			"name": schema.StringAttribute{
																				MarkdownDescription: `Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names`,
																				Optional:            true,
																			},
																			"optional": schema.BoolAttribute{
																				MarkdownDescription: `Specify whether the Secret or its key must be defined`,
																				Optional:            true,
																			},
																		},
																	},
																},
															},
														},
													},
												},
												"env_from": schema.ListNestedAttribute{
													MarkdownDescription: `List of sources to populate environment variables in the container. The keys defined within a source must be a C_IDENTIFIER. All invalid keys will be reported as an event when the container is starting. When a key exists in multiple sources, the value associated with the last source will take precedence. Values defined by an Env with a duplicate key will take precedence. Cannot be updated.`,
													Optional:            true,

													NestedObject: schema.NestedAttributeObject{
														Attributes: map[string]schema.Attribute{
															"config_map_ref": schema.SingleNestedAttribute{
																MarkdownDescription: `The ConfigMap to select from`,
																Optional:            true,

																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		MarkdownDescription: `Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names`,
																		Optional:            true,
																	},
																	"optional": schema.BoolAttribute{
																		MarkdownDescription: `Specify whether the ConfigMap must be defined`,
																		Optional:            true,
																	},
																},
															},
															"prefix": schema.StringAttribute{
																MarkdownDescription: `An optional identifier to prepend to each key in the ConfigMap. Must be a C_IDENTIFIER.`,
																Optional:            true,
															},
															"secret_ref": schema.SingleNestedAttribute{
																MarkdownDescription: `The Secret to select from`,
																Optional:            true,

																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		MarkdownDescription: `Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names`,
																		Optional:            true,
																	},
																	"optional": schema.BoolAttribute{
																		MarkdownDescription: `Specify whether the Secret must be defined`,
																		Optional:            true,
																	},
																},
															},
														},
													},
												},
												"image": schema.StringAttribute{
													MarkdownDescription: `Container image name. More info: https://kubernetes.io/docs/concepts/containers/images This field is optional to allow higher level config management to default or override container images in workload controllers like Deployments and StatefulSets.`,
													Optional:            true,
												},
												"image_pull_policy": schema.StringAttribute{
													MarkdownDescription: `Image pull policy. One of Always, Never, IfNotPresent. Defaults to Always if :latest tag is specified, or IfNotPresent otherwise. Cannot be updated. More info: https://kubernetes.io/docs/concepts/containers/images#updating-images`,
													Optional:            true,
												},
												"lifecycle": schema.SingleNestedAttribute{
													MarkdownDescription: `Actions that the management system should take in response to container lifecycle events. Cannot be updated.`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"post_start": schema.SingleNestedAttribute{
															MarkdownDescription: `PostStart is called immediately after a container is created. If the handler fails, the container is terminated and restarted according to its restart policy. Other management of the container blocks until the hook completes. More info: https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#container-hooks`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"exec": schema.SingleNestedAttribute{
																	MarkdownDescription: `Exec specifies the action to take.`,
																	Optional:            true,

																	Attributes: map[string]schema.Attribute{
																		"command": schema.ListAttribute{
																			MarkdownDescription: `Command is the command line to execute inside the container, the working directory for the command  is root ('/') in the container's filesystem. The command is simply exec'd, it is not run inside a shell, so traditional shell instructions ('|', etc) won't work. To use a shell, you need to explicitly call out to that shell. Exit status of 0 is treated as live/healthy and non-zero is unhealthy.`,
																			ElementType:         types.StringType,
																			Optional:            true,
																		},
																	},
																},
																"http_get": schema.SingleNestedAttribute{
																	MarkdownDescription: `HTTPGet specifies the http request to perform.`,
																	Optional:            true,

																	Attributes: map[string]schema.Attribute{
																		"host": schema.StringAttribute{
																			MarkdownDescription: `Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.`,
																			Optional:            true,
																		},
																		"http_headers": schema.ListNestedAttribute{
																			MarkdownDescription: `Custom headers to set in the request. HTTP allows repeated headers.`,
																			Optional:            true,

																			NestedObject: schema.NestedAttributeObject{
																				Attributes: map[string]schema.Attribute{
																					"name": schema.StringAttribute{
																						MarkdownDescription: `The header field name. This will be canonicalized upon output, so case-variant names will be understood as the same header.`,
																						Optional:            true,
																					},
																					"value": schema.StringAttribute{
																						MarkdownDescription: `The header field value`,
																						Optional:            true,
																					},
																				},
																			},
																		},
																		"path": schema.StringAttribute{
																			MarkdownDescription: `Path to access on the HTTP server.`,
																			Optional:            true,
																		},
																		"port": schema.StringAttribute{
																			MarkdownDescription: `Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
																			Optional:            true,
																		},
																		"scheme": schema.StringAttribute{
																			MarkdownDescription: `Scheme to use for connecting to the host. Defaults to HTTP.`,
																			Optional:            true,
																		},
																	},
																},
																"tcp_socket": schema.SingleNestedAttribute{
																	MarkdownDescription: `Deprecated. TCPSocket is NOT supported as a LifecycleHandler and kept for the backward compatibility. There are no validation of this field and lifecycle hooks will fail in runtime when tcp handler is specified.`,
																	Optional:            true,

																	Attributes: map[string]schema.Attribute{
																		"host": schema.StringAttribute{
																			MarkdownDescription: `Optional: Host name to connect to, defaults to the pod IP.`,
																			Optional:            true,
																		},
																		"port": schema.StringAttribute{
																			MarkdownDescription: `Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
																			Optional:            true,
																		},
																	},
																},
															},
														},
														"pre_stop": schema.SingleNestedAttribute{
															MarkdownDescription: `PreStop is called immediately before a container is terminated due to an API request or management event such as liveness/startup probe failure, preemption, resource contention, etc. The handler is not called if the container crashes or exits. The Pod's termination grace period countdown begins before the PreStop hook is executed. Regardless of the outcome of the handler, the container will eventually terminate within the Pod's termination grace period (unless delayed by finalizers). Other management of the container blocks until the hook completes or until the termination grace period is reached. More info: https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#container-hooks`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"exec": schema.SingleNestedAttribute{
																	MarkdownDescription: `Exec specifies the action to take.`,
																	Optional:            true,

																	Attributes: map[string]schema.Attribute{
																		"command": schema.ListAttribute{
																			MarkdownDescription: `Command is the command line to execute inside the container, the working directory for the command  is root ('/') in the container's filesystem. The command is simply exec'd, it is not run inside a shell, so traditional shell instructions ('|', etc) won't work. To use a shell, you need to explicitly call out to that shell. Exit status of 0 is treated as live/healthy and non-zero is unhealthy.`,
																			ElementType:         types.StringType,
																			Optional:            true,
																		},
																	},
																},
																"http_get": schema.SingleNestedAttribute{
																	MarkdownDescription: `HTTPGet specifies the http request to perform.`,
																	Optional:            true,

																	Attributes: map[string]schema.Attribute{
																		"host": schema.StringAttribute{
																			MarkdownDescription: `Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.`,
																			Optional:            true,
																		},
																		"http_headers": schema.ListNestedAttribute{
																			MarkdownDescription: `Custom headers to set in the request. HTTP allows repeated headers.`,
																			Optional:            true,

																			NestedObject: schema.NestedAttributeObject{
																				Attributes: map[string]schema.Attribute{
																					"name": schema.StringAttribute{
																						MarkdownDescription: `The header field name. This will be canonicalized upon output, so case-variant names will be understood as the same header.`,
																						Optional:            true,
																					},
																					"value": schema.StringAttribute{
																						MarkdownDescription: `The header field value`,
																						Optional:            true,
																					},
																				},
																			},
																		},
																		"path": schema.StringAttribute{
																			MarkdownDescription: `Path to access on the HTTP server.`,
																			Optional:            true,
																		},
																		"port": schema.StringAttribute{
																			MarkdownDescription: `Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
																			Optional:            true,
																		},
																		"scheme": schema.StringAttribute{
																			MarkdownDescription: `Scheme to use for connecting to the host. Defaults to HTTP.`,
																			Optional:            true,
																		},
																	},
																},
																"tcp_socket": schema.SingleNestedAttribute{
																	MarkdownDescription: `Deprecated. TCPSocket is NOT supported as a LifecycleHandler and kept for the backward compatibility. There are no validation of this field and lifecycle hooks will fail in runtime when tcp handler is specified.`,
																	Optional:            true,

																	Attributes: map[string]schema.Attribute{
																		"host": schema.StringAttribute{
																			MarkdownDescription: `Optional: Host name to connect to, defaults to the pod IP.`,
																			Optional:            true,
																		},
																		"port": schema.StringAttribute{
																			MarkdownDescription: `Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
																			Optional:            true,
																		},
																	},
																},
															},
														},
													},
												},
												"liveness_probe": schema.SingleNestedAttribute{
													MarkdownDescription: `Periodic probe of container liveness. Container will be restarted if the probe fails. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"exec": schema.SingleNestedAttribute{
															MarkdownDescription: `Exec specifies the action to take.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"command": schema.ListAttribute{
																	MarkdownDescription: `Command is the command line to execute inside the container, the working directory for the command  is root ('/') in the container's filesystem. The command is simply exec'd, it is not run inside a shell, so traditional shell instructions ('|', etc) won't work. To use a shell, you need to explicitly call out to that shell. Exit status of 0 is treated as live/healthy and non-zero is unhealthy.`,
																	ElementType:         types.StringType,
																	Optional:            true,
																},
															},
														},
														"failure_threshold": schema.Int64Attribute{
															MarkdownDescription: `Minimum consecutive failures for the probe to be considered failed after having succeeded. Defaults to 3. Minimum value is 1.`,
															Optional:            true,
														},
														"grpc": schema.SingleNestedAttribute{
															MarkdownDescription: `GRPC specifies an action involving a GRPC port.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"port": schema.Int64Attribute{
																	MarkdownDescription: `Port number of the gRPC service. Number must be in the range 1 to 65535.`,
																	Optional:            true,
																},
																"service": schema.StringAttribute{
																	MarkdownDescription: `Service is the name of the service to place in the gRPC HealthCheckRequest (see https://github.com/grpc/grpc/blob/master/doc/health-checking.md).

If this is not specified, the default behavior is defined by gRPC.`,
																	Optional: true,
																},
															},
														},
														"http_get": schema.SingleNestedAttribute{
															MarkdownDescription: `HTTPGet specifies the http request to perform.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"host": schema.StringAttribute{
																	MarkdownDescription: `Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.`,
																	Optional:            true,
																},
																"http_headers": schema.ListNestedAttribute{
																	MarkdownDescription: `Custom headers to set in the request. HTTP allows repeated headers.`,
																	Optional:            true,

																	NestedObject: schema.NestedAttributeObject{
																		Attributes: map[string]schema.Attribute{
																			"name": schema.StringAttribute{
																				MarkdownDescription: `The header field name. This will be canonicalized upon output, so case-variant names will be understood as the same header.`,
																				Optional:            true,
																			},
																			"value": schema.StringAttribute{
																				MarkdownDescription: `The header field value`,
																				Optional:            true,
																			},
																		},
																	},
																},
																"path": schema.StringAttribute{
																	MarkdownDescription: `Path to access on the HTTP server.`,
																	Optional:            true,
																},
																"port": schema.StringAttribute{
																	MarkdownDescription: `Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
																	Optional:            true,
																},
																"scheme": schema.StringAttribute{
																	MarkdownDescription: `Scheme to use for connecting to the host. Defaults to HTTP.`,
																	Optional:            true,
																},
															},
														},
														"initial_delay_seconds": schema.Int64Attribute{
															MarkdownDescription: `Number of seconds after the container has started before liveness probes are initiated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes`,
															Optional:            true,
														},
														"period_seconds": schema.Int64Attribute{
															MarkdownDescription: `How often (in seconds) to perform the probe. Default to 10 seconds. Minimum value is 1.`,
															Optional:            true,
														},
														"success_threshold": schema.Int64Attribute{
															MarkdownDescription: `Minimum consecutive successes for the probe to be considered successful after having failed. Defaults to 1. Must be 1 for liveness and startup. Minimum value is 1.`,
															Optional:            true,
														},
														"tcp_socket": schema.SingleNestedAttribute{
															MarkdownDescription: `TCPSocket specifies an action involving a TCP port.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"host": schema.StringAttribute{
																	MarkdownDescription: `Optional: Host name to connect to, defaults to the pod IP.`,
																	Optional:            true,
																},
																"port": schema.StringAttribute{
																	MarkdownDescription: `Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
																	Optional:            true,
																},
															},
														},
														"termination_grace_period_seconds": schema.Int64Attribute{
															MarkdownDescription: `Optional duration in seconds the pod needs to terminate gracefully upon probe failure. The grace period is the duration in seconds after the processes running in the pod are sent a termination signal and the time when the processes are forcibly halted with a kill signal. Set this value longer than the expected cleanup time for your process. If this value is nil, the pod's terminationGracePeriodSeconds will be used. Otherwise, this value overrides the value provided by the pod spec. Value must be non-negative integer. The value zero indicates stop immediately via the kill signal (no opportunity to shut down). This is a beta field and requires enabling ProbeTerminationGracePeriod feature gate. Minimum value is 1. spec.terminationGracePeriodSeconds is used if unset.`,
															Optional:            true,
														},
														"timeout_seconds": schema.Int64Attribute{
															MarkdownDescription: `Number of seconds after which the probe times out. Defaults to 1 second. Minimum value is 1. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes`,
															Optional:            true,
														},
													},
												},
												"name": schema.StringAttribute{
													MarkdownDescription: `Name of the container specified as a DNS_LABEL. Each container in a pod must have a unique name (DNS_LABEL). Cannot be updated.`,
													Optional:            true,
												},
												"ports": schema.ListNestedAttribute{
													MarkdownDescription: `List of ports to expose from the container. Not specifying a port here DOES NOT prevent that port from being exposed. Any port which is listening on the default "0.0.0.0" address inside a container will be accessible from the network. Modifying this array with strategic merge patch may corrupt the data. For more information See https://github.com/kubernetes/kubernetes/issues/108255. Cannot be updated.`,
													Optional:            true,

													NestedObject: schema.NestedAttributeObject{
														Attributes: map[string]schema.Attribute{
															"container_port": schema.Int64Attribute{
																MarkdownDescription: `Number of port to expose on the pod's IP address. This must be a valid port number, 0 < x < 65536.`,
																Optional:            true,
															},
															"host_ip": schema.StringAttribute{
																MarkdownDescription: `What host IP to bind the external port to.`,
																Optional:            true,
															},
															"host_port": schema.Int64Attribute{
																MarkdownDescription: `Number of port to expose on the host. If specified, this must be a valid port number, 0 < x < 65536. If HostNetwork is specified, this must match ContainerPort. Most containers do not need this.`,
																Optional:            true,
															},
															"name": schema.StringAttribute{
																MarkdownDescription: `If specified, this must be an IANA_SVC_NAME and unique within the pod. Each named port in a pod must have a unique name. Name for the port that can be referred to by services.`,
																Optional:            true,
															},
															"protocol": schema.StringAttribute{
																MarkdownDescription: `Protocol for port. Must be UDP, TCP, or SCTP. Defaults to "TCP".`,
																Optional:            true,
															},
														},
													},
												},
												"readiness_probe": schema.SingleNestedAttribute{
													MarkdownDescription: `Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails. Cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"exec": schema.SingleNestedAttribute{
															MarkdownDescription: `Exec specifies the action to take.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"command": schema.ListAttribute{
																	MarkdownDescription: `Command is the command line to execute inside the container, the working directory for the command  is root ('/') in the container's filesystem. The command is simply exec'd, it is not run inside a shell, so traditional shell instructions ('|', etc) won't work. To use a shell, you need to explicitly call out to that shell. Exit status of 0 is treated as live/healthy and non-zero is unhealthy.`,
																	ElementType:         types.StringType,
																	Optional:            true,
																},
															},
														},
														"failure_threshold": schema.Int64Attribute{
															MarkdownDescription: `Minimum consecutive failures for the probe to be considered failed after having succeeded. Defaults to 3. Minimum value is 1.`,
															Optional:            true,
														},
														"grpc": schema.SingleNestedAttribute{
															MarkdownDescription: `GRPC specifies an action involving a GRPC port.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"port": schema.Int64Attribute{
																	MarkdownDescription: `Port number of the gRPC service. Number must be in the range 1 to 65535.`,
																	Optional:            true,
																},
																"service": schema.StringAttribute{
																	MarkdownDescription: `Service is the name of the service to place in the gRPC HealthCheckRequest (see https://github.com/grpc/grpc/blob/master/doc/health-checking.md).

If this is not specified, the default behavior is defined by gRPC.`,
																	Optional: true,
																},
															},
														},
														"http_get": schema.SingleNestedAttribute{
															MarkdownDescription: `HTTPGet specifies the http request to perform.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"host": schema.StringAttribute{
																	MarkdownDescription: `Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.`,
																	Optional:            true,
																},
																"http_headers": schema.ListNestedAttribute{
																	MarkdownDescription: `Custom headers to set in the request. HTTP allows repeated headers.`,
																	Optional:            true,

																	NestedObject: schema.NestedAttributeObject{
																		Attributes: map[string]schema.Attribute{
																			"name": schema.StringAttribute{
																				MarkdownDescription: `The header field name. This will be canonicalized upon output, so case-variant names will be understood as the same header.`,
																				Optional:            true,
																			},
																			"value": schema.StringAttribute{
																				MarkdownDescription: `The header field value`,
																				Optional:            true,
																			},
																		},
																	},
																},
																"path": schema.StringAttribute{
																	MarkdownDescription: `Path to access on the HTTP server.`,
																	Optional:            true,
																},
																"port": schema.StringAttribute{
																	MarkdownDescription: `Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
																	Optional:            true,
																},
																"scheme": schema.StringAttribute{
																	MarkdownDescription: `Scheme to use for connecting to the host. Defaults to HTTP.`,
																	Optional:            true,
																},
															},
														},
														"initial_delay_seconds": schema.Int64Attribute{
															MarkdownDescription: `Number of seconds after the container has started before liveness probes are initiated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes`,
															Optional:            true,
														},
														"period_seconds": schema.Int64Attribute{
															MarkdownDescription: `How often (in seconds) to perform the probe. Default to 10 seconds. Minimum value is 1.`,
															Optional:            true,
														},
														"success_threshold": schema.Int64Attribute{
															MarkdownDescription: `Minimum consecutive successes for the probe to be considered successful after having failed. Defaults to 1. Must be 1 for liveness and startup. Minimum value is 1.`,
															Optional:            true,
														},
														"tcp_socket": schema.SingleNestedAttribute{
															MarkdownDescription: `TCPSocket specifies an action involving a TCP port.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"host": schema.StringAttribute{
																	MarkdownDescription: `Optional: Host name to connect to, defaults to the pod IP.`,
																	Optional:            true,
																},
																"port": schema.StringAttribute{
																	MarkdownDescription: `Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
																	Optional:            true,
																},
															},
														},
														"termination_grace_period_seconds": schema.Int64Attribute{
															MarkdownDescription: `Optional duration in seconds the pod needs to terminate gracefully upon probe failure. The grace period is the duration in seconds after the processes running in the pod are sent a termination signal and the time when the processes are forcibly halted with a kill signal. Set this value longer than the expected cleanup time for your process. If this value is nil, the pod's terminationGracePeriodSeconds will be used. Otherwise, this value overrides the value provided by the pod spec. Value must be non-negative integer. The value zero indicates stop immediately via the kill signal (no opportunity to shut down). This is a beta field and requires enabling ProbeTerminationGracePeriod feature gate. Minimum value is 1. spec.terminationGracePeriodSeconds is used if unset.`,
															Optional:            true,
														},
														"timeout_seconds": schema.Int64Attribute{
															MarkdownDescription: `Number of seconds after which the probe times out. Defaults to 1 second. Minimum value is 1. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes`,
															Optional:            true,
														},
													},
												},
												"resize_policy": schema.ListNestedAttribute{
													MarkdownDescription: `Resources resize policy for the container.`,
													Optional:            true,

													NestedObject: schema.NestedAttributeObject{
														Attributes: map[string]schema.Attribute{
															"resource_name": schema.StringAttribute{
																MarkdownDescription: `Name of the resource to which this resource resize policy applies. Supported values: cpu, memory.`,
																Optional:            true,
															},
															"restart_policy": schema.StringAttribute{
																MarkdownDescription: `Restart policy to apply when specified resource is resized. If not specified, it defaults to NotRequired.`,
																Optional:            true,
															},
														},
													},
												},
												"resources": schema.SingleNestedAttribute{
													MarkdownDescription: `Compute Resources required by this container. Cannot be updated. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"claims": schema.ListNestedAttribute{
															MarkdownDescription: `Claims lists the names of resources, defined in spec.resourceClaims, that are used by this container.

This is an alpha field and requires enabling the DynamicResourceAllocation feature gate.

This field is immutable. It can only be set for containers.`,
															Optional: true,

															NestedObject: schema.NestedAttributeObject{
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		MarkdownDescription: `Name must match the name of one entry in pod.spec.resourceClaims of the Pod where this field is used. It makes that resource available inside a container.`,
																		Optional:            true,
																	},
																},
															},
														},
														"limits": schema.MapAttribute{
															MarkdownDescription: `Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/`,
															ElementType:         types.StringType,
															Optional:            true,
														},
														"requests": schema.MapAttribute{
															MarkdownDescription: `Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. Requests cannot exceed Limits. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/`,
															ElementType:         types.StringType,
															Optional:            true,
														},
													},
												},
												"restart_policy": schema.StringAttribute{
													MarkdownDescription: `RestartPolicy defines the restart behavior of individual containers in a pod. This field may only be set for init containers, and the only allowed value is "Always". For non-init containers or when this field is not specified, the restart behavior is defined by the Pod's restart policy and the container type. Setting the RestartPolicy as "Always" for the init container will have the following effect: this init container will be continually restarted on exit until all regular containers have terminated. Once all regular containers have completed, all init containers with restartPolicy "Always" will be shut down. This lifecycle differs from normal init containers and is often referred to as a "sidecar" container. Although this init container still starts in the init container sequence, it does not wait for the container to complete before proceeding to the next init container. Instead, the next init container starts immediately after this init container is started, or after any startupProbe has successfully completed.`,
													Optional:            true,
												},
												"security_context": schema.SingleNestedAttribute{
													MarkdownDescription: `SecurityContext defines the security options the container should be run with. If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext. More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"allow_privilege_escalation": schema.BoolAttribute{
															MarkdownDescription: `AllowPrivilegeEscalation controls whether a process can gain more privileges than its parent process. This bool directly controls if the no_new_privs flag will be set on the container process. AllowPrivilegeEscalation is true always when the container is: 1) run as Privileged 2) has CAP_SYS_ADMIN Note that this field cannot be set when spec.os.name is windows.`,
															Optional:            true,
														},
														"capabilities": schema.SingleNestedAttribute{
															MarkdownDescription: `The capabilities to add/drop when running containers. Defaults to the default set of capabilities granted by the container runtime. Note that this field cannot be set when spec.os.name is windows.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"add": schema.ListAttribute{
																	MarkdownDescription: `Added capabilities`,
																	ElementType:         types.StringType,
																	Optional:            true,
																},
																"drop": schema.ListAttribute{
																	MarkdownDescription: `Removed capabilities`,
																	ElementType:         types.StringType,
																	Optional:            true,
																},
															},
														},
														"privileged": schema.BoolAttribute{
															MarkdownDescription: `Run container in privileged mode. Processes in privileged containers are essentially equivalent to root on the host. Defaults to false. Note that this field cannot be set when spec.os.name is windows.`,
															Optional:            true,
														},
														"proc_mount": schema.StringAttribute{
															MarkdownDescription: `procMount denotes the type of proc mount to use for the containers. The default is DefaultProcMount which uses the container runtime defaults for readonly paths and masked paths. This requires the ProcMountType feature flag to be enabled. Note that this field cannot be set when spec.os.name is windows.`,
															Optional:            true,
														},
														"read_only_root_filesystem": schema.BoolAttribute{
															MarkdownDescription: `Whether this container has a read-only root filesystem. Default is false. Note that this field cannot be set when spec.os.name is windows.`,
															Optional:            true,
														},
														"run_as_group": schema.Int64Attribute{
															MarkdownDescription: `The GID to run the entrypoint of the container process. Uses runtime default if unset. May also be set in PodSecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. Note that this field cannot be set when spec.os.name is windows.`,
															Optional:            true,
														},
														"run_as_non_root": schema.BoolAttribute{
															MarkdownDescription: `Indicates that the container must run as a non-root user. If true, the Kubelet will validate the image at runtime to ensure that it does not run as UID 0 (root) and fail to start the container if it does. If unset or false, no such validation will be performed. May also be set in PodSecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.`,
															Optional:            true,
														},
														"run_as_user": schema.Int64Attribute{
															MarkdownDescription: `The UID to run the entrypoint of the container process. Defaults to user specified in image metadata if unspecified. May also be set in PodSecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. Note that this field cannot be set when spec.os.name is windows.`,
															Optional:            true,
														},
														"se_linux_options": schema.SingleNestedAttribute{
															MarkdownDescription: `The SELinux context to be applied to the container. If unspecified, the container runtime will allocate a random SELinux context for each container.  May also be set in PodSecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. Note that this field cannot be set when spec.os.name is windows.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"level": schema.StringAttribute{
																	MarkdownDescription: `Level is SELinux level label that applies to the container.`,
																	Optional:            true,
																},
																"role": schema.StringAttribute{
																	MarkdownDescription: `Role is a SELinux role label that applies to the container.`,
																	Optional:            true,
																},
																"type": schema.StringAttribute{
																	MarkdownDescription: `Type is a SELinux type label that applies to the container.`,
																	Optional:            true,
																},
																"user": schema.StringAttribute{
																	MarkdownDescription: `User is a SELinux user label that applies to the container.`,
																	Optional:            true,
																},
															},
														},
														"seccomp_profile": schema.SingleNestedAttribute{
															MarkdownDescription: `The seccomp options to use by this container. If seccomp options are provided at both the pod & container level, the container options override the pod options. Note that this field cannot be set when spec.os.name is windows.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"localhost_profile": schema.StringAttribute{
																	MarkdownDescription: `localhostProfile indicates a profile defined in a file on the node should be used. The profile must be preconfigured on the node to work. Must be a descending path, relative to the kubelet's configured seccomp profile location. Must be set if type is "Localhost". Must NOT be set for any other type.`,
																	Optional:            true,
																},
																"type": schema.StringAttribute{
																	MarkdownDescription: `type indicates which kind of seccomp profile will be applied. Valid options are:

Localhost - a profile defined in a file on the node should be used. RuntimeDefault - the container runtime default profile should be used. Unconfined - no profile should be applied.`,
																	Optional: true,
																},
															},
														},
														"windows_options": schema.SingleNestedAttribute{
															MarkdownDescription: `The Windows specific settings applied to all containers. If unspecified, the options from the PodSecurityContext will be used. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. Note that this field cannot be set when spec.os.name is linux.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"gmsa_credential_spec": schema.StringAttribute{
																	MarkdownDescription: `GMSACredentialSpec is where the GMSA admission webhook (https://github.com/kubernetes-sigs/windows-gmsa) inlines the contents of the GMSA credential spec named by the GMSACredentialSpecName field.`,
																	Optional:            true,
																},
																"gmsa_credential_spec_name": schema.StringAttribute{
																	MarkdownDescription: `GMSACredentialSpecName is the name of the GMSA credential spec to use.`,
																	Optional:            true,
																},
																"host_process": schema.BoolAttribute{
																	MarkdownDescription: `HostProcess determines if a container should be run as a 'Host Process' container. All of a Pod's containers must have the same effective HostProcess value (it is not allowed to have a mix of HostProcess containers and non-HostProcess containers). In addition, if HostProcess is true then HostNetwork must also be set to true.`,
																	Optional:            true,
																},
																"run_as_user_name": schema.StringAttribute{
																	MarkdownDescription: `The UserName in Windows to run the entrypoint of the container process. Defaults to the user specified in image metadata if unspecified. May also be set in PodSecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.`,
																	Optional:            true,
																},
															},
														},
													},
												},
												"startup_probe": schema.SingleNestedAttribute{
													MarkdownDescription: `StartupProbe indicates that the Pod has successfully initialized. If specified, no other probes are executed until this completes successfully. If this probe fails, the Pod will be restarted, just as if the livenessProbe failed. This can be used to provide different probe parameters at the beginning of a Pod's lifecycle, when it might take a long time to load data or warm a cache, than during steady-state operation. This cannot be updated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"exec": schema.SingleNestedAttribute{
															MarkdownDescription: `Exec specifies the action to take.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"command": schema.ListAttribute{
																	MarkdownDescription: `Command is the command line to execute inside the container, the working directory for the command  is root ('/') in the container's filesystem. The command is simply exec'd, it is not run inside a shell, so traditional shell instructions ('|', etc) won't work. To use a shell, you need to explicitly call out to that shell. Exit status of 0 is treated as live/healthy and non-zero is unhealthy.`,
																	ElementType:         types.StringType,
																	Optional:            true,
																},
															},
														},
														"failure_threshold": schema.Int64Attribute{
															MarkdownDescription: `Minimum consecutive failures for the probe to be considered failed after having succeeded. Defaults to 3. Minimum value is 1.`,
															Optional:            true,
														},
														"grpc": schema.SingleNestedAttribute{
															MarkdownDescription: `GRPC specifies an action involving a GRPC port.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"port": schema.Int64Attribute{
																	MarkdownDescription: `Port number of the gRPC service. Number must be in the range 1 to 65535.`,
																	Optional:            true,
																},
																"service": schema.StringAttribute{
																	MarkdownDescription: `Service is the name of the service to place in the gRPC HealthCheckRequest (see https://github.com/grpc/grpc/blob/master/doc/health-checking.md).

If this is not specified, the default behavior is defined by gRPC.`,
																	Optional: true,
																},
															},
														},
														"http_get": schema.SingleNestedAttribute{
															MarkdownDescription: `HTTPGet specifies the http request to perform.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"host": schema.StringAttribute{
																	MarkdownDescription: `Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.`,
																	Optional:            true,
																},
																"http_headers": schema.ListNestedAttribute{
																	MarkdownDescription: `Custom headers to set in the request. HTTP allows repeated headers.`,
																	Optional:            true,

																	NestedObject: schema.NestedAttributeObject{
																		Attributes: map[string]schema.Attribute{
																			"name": schema.StringAttribute{
																				MarkdownDescription: `The header field name. This will be canonicalized upon output, so case-variant names will be understood as the same header.`,
																				Optional:            true,
																			},
																			"value": schema.StringAttribute{
																				MarkdownDescription: `The header field value`,
																				Optional:            true,
																			},
																		},
																	},
																},
																"path": schema.StringAttribute{
																	MarkdownDescription: `Path to access on the HTTP server.`,
																	Optional:            true,
																},
																"port": schema.StringAttribute{
																	MarkdownDescription: `Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
																	Optional:            true,
																},
																"scheme": schema.StringAttribute{
																	MarkdownDescription: `Scheme to use for connecting to the host. Defaults to HTTP.`,
																	Optional:            true,
																},
															},
														},
														"initial_delay_seconds": schema.Int64Attribute{
															MarkdownDescription: `Number of seconds after the container has started before liveness probes are initiated. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes`,
															Optional:            true,
														},
														"period_seconds": schema.Int64Attribute{
															MarkdownDescription: `How often (in seconds) to perform the probe. Default to 10 seconds. Minimum value is 1.`,
															Optional:            true,
														},
														"success_threshold": schema.Int64Attribute{
															MarkdownDescription: `Minimum consecutive successes for the probe to be considered successful after having failed. Defaults to 1. Must be 1 for liveness and startup. Minimum value is 1.`,
															Optional:            true,
														},
														"tcp_socket": schema.SingleNestedAttribute{
															MarkdownDescription: `TCPSocket specifies an action involving a TCP port.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"host": schema.StringAttribute{
																	MarkdownDescription: `Optional: Host name to connect to, defaults to the pod IP.`,
																	Optional:            true,
																},
																"port": schema.StringAttribute{
																	MarkdownDescription: `Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
																	Optional:            true,
																},
															},
														},
														"termination_grace_period_seconds": schema.Int64Attribute{
															MarkdownDescription: `Optional duration in seconds the pod needs to terminate gracefully upon probe failure. The grace period is the duration in seconds after the processes running in the pod are sent a termination signal and the time when the processes are forcibly halted with a kill signal. Set this value longer than the expected cleanup time for your process. If this value is nil, the pod's terminationGracePeriodSeconds will be used. Otherwise, this value overrides the value provided by the pod spec. Value must be non-negative integer. The value zero indicates stop immediately via the kill signal (no opportunity to shut down). This is a beta field and requires enabling ProbeTerminationGracePeriod feature gate. Minimum value is 1. spec.terminationGracePeriodSeconds is used if unset.`,
															Optional:            true,
														},
														"timeout_seconds": schema.Int64Attribute{
															MarkdownDescription: `Number of seconds after which the probe times out. Defaults to 1 second. Minimum value is 1. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes`,
															Optional:            true,
														},
													},
												},
												"stdin": schema.BoolAttribute{
													MarkdownDescription: `Whether this container should allocate a buffer for stdin in the container runtime. If this is not set, reads from stdin in the container will always result in EOF. Default is false.`,
													Optional:            true,
												},
												"stdin_once": schema.BoolAttribute{
													MarkdownDescription: `Whether the container runtime should close the stdin channel after it has been opened by a single attach. When stdin is true the stdin stream will remain open across multiple attach sessions. If stdinOnce is set to true, stdin is opened on container start, is empty until the first client attaches to stdin, and then remains open and accepts data until the client disconnects, at which time stdin is closed and remains closed until the container is restarted. If this flag is false, a container processes that reads from stdin will never receive an EOF. Default is false`,
													Optional:            true,
												},
												"termination_message_path": schema.StringAttribute{
													MarkdownDescription: `Optional: Path at which the file to which the container's termination message will be written is mounted into the container's filesystem. Message written is intended to be brief final status, such as an assertion failure message. Will be truncated by the node if greater than 4096 bytes. The total message length across all containers will be limited to 12kb. Defaults to /dev/termination-log. Cannot be updated.`,
													Optional:            true,
												},
												"termination_message_policy": schema.StringAttribute{
													MarkdownDescription: `Indicate how the termination message should be populated. File will use the contents of terminationMessagePath to populate the container status message on both success and failure. FallbackToLogsOnError will use the last chunk of container log output if the termination message file is empty and the container exited with an error. The log output is limited to 2048 bytes or 80 lines, whichever is smaller. Defaults to File. Cannot be updated.`,
													Optional:            true,
												},
												"tty": schema.BoolAttribute{
													MarkdownDescription: `Whether this container should allocate a TTY for itself, also requires 'stdin' to be true. Default is false.`,
													Optional:            true,
												},
												"volume_devices": schema.ListNestedAttribute{
													MarkdownDescription: `volumeDevices is the list of block devices to be used by the container.`,
													Optional:            true,

													NestedObject: schema.NestedAttributeObject{
														Attributes: map[string]schema.Attribute{
															"device_path": schema.StringAttribute{
																MarkdownDescription: `devicePath is the path inside of the container that the device will be mapped to.`,
																Optional:            true,
															},
															"name": schema.StringAttribute{
																MarkdownDescription: `name must match the name of a persistentVolumeClaim in the pod`,
																Optional:            true,
															},
														},
													},
												},
												"volume_mounts": schema.ListNestedAttribute{
													MarkdownDescription: `Pod volumes to mount into the container's filesystem. Cannot be updated.`,
													Optional:            true,

													NestedObject: schema.NestedAttributeObject{
														Attributes: map[string]schema.Attribute{
															"mount_path": schema.StringAttribute{
																MarkdownDescription: `Path within the container at which the volume should be mounted.  Must not contain ':'.`,
																Optional:            true,
															},
															"mount_propagation": schema.StringAttribute{
																MarkdownDescription: `mountPropagation determines how mounts are propagated from the host to container and the other way around. When not set, MountPropagationNone is used. This field is beta in 1.10.`,
																Optional:            true,
															},
															"name": schema.StringAttribute{
																MarkdownDescription: `This must match the Name of a Volume.`,
																Optional:            true,
															},
															"read_only": schema.BoolAttribute{
																MarkdownDescription: `Mounted read-only if true, read-write otherwise (false or unspecified). Defaults to false.`,
																Optional:            true,
															},
															"sub_path": schema.StringAttribute{
																MarkdownDescription: `Path within the volume from which the container's volume should be mounted. Defaults to "" (volume's root).`,
																Optional:            true,
															},
															"sub_path_expr": schema.StringAttribute{
																MarkdownDescription: `Expanded path within the volume from which the container's volume should be mounted. Behaves similarly to SubPath but environment variable references $(VAR_NAME) are expanded using the container's environment. Defaults to "" (volume's root). SubPathExpr and SubPath are mutually exclusive.`,
																Optional:            true,
															},
														},
													},
												},
												"working_dir": schema.StringAttribute{
													MarkdownDescription: `Container's working directory. If not specified, the container runtime's default will be used, which might be configured in the container image. Cannot be updated.`,
													Optional:            true,
												},
											},
										},
									},
									"node_name": schema.StringAttribute{
										MarkdownDescription: `NodeName is a request to schedule this pod onto a specific node. If it is non-empty, the scheduler simply schedules this pod onto that node, assuming that it fits resource requirements.`,
										Optional:            true,
									},
									"node_selector": schema.MapAttribute{
										MarkdownDescription: `NodeSelector is a selector which must be true for the pod to fit on a node. Selector which must match a node's labels for the pod to be scheduled on that node. More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/`,
										ElementType:         types.StringType,
										Optional:            true,
									},
									"os": schema.SingleNestedAttribute{
										MarkdownDescription: `Specifies the OS of the containers in the pod. Some pod and container fields are restricted if this is set.

If the OS field is set to linux, the following fields must be unset: -securityContext.windowsOptions

If the OS field is set to windows, following fields must be unset: - spec.hostPID - spec.hostIPC - spec.hostUsers - spec.securityContext.seLinuxOptions - spec.securityContext.seccompProfile - spec.securityContext.fsGroup - spec.securityContext.fsGroupChangePolicy - spec.securityContext.sysctls - spec.shareProcessNamespace - spec.securityContext.runAsUser - spec.securityContext.runAsGroup - spec.securityContext.supplementalGroups - spec.containers[*].securityContext.seLinuxOptions - spec.containers[*].securityContext.seccompProfile - spec.containers[*].securityContext.capabilities - spec.containers[*].securityContext.readOnlyRootFilesystem - spec.containers[*].securityContext.privileged - spec.containers[*].securityContext.allowPrivilegeEscalation - spec.containers[*].securityContext.procMount - spec.containers[*].securityContext.runAsUser - spec.containers[*].securityContext.runAsGroup`,
										Optional: true,

										Attributes: map[string]schema.Attribute{
											"name": schema.StringAttribute{
												MarkdownDescription: `Name is the name of the operating system. The currently supported values are linux and windows. Additional value may be defined in future and can be one of: https://github.com/opencontainers/runtime-spec/blob/master/config.md#platform-specific-configuration Clients should expect to handle additional values and treat unrecognized values in this field as os: null`,
												Optional:            true,
											},
										},
									},
									"overhead": schema.MapAttribute{
										MarkdownDescription: `Overhead represents the resource overhead associated with running a pod for a given RuntimeClass. This field will be autopopulated at admission time by the RuntimeClass admission controller. If the RuntimeClass admission controller is enabled, overhead must not be set in Pod create requests. The RuntimeClass admission controller will reject Pod create requests which have the overhead already set. If RuntimeClass is configured and selected in the PodSpec, Overhead will be set to the value defined in the corresponding RuntimeClass, otherwise it will remain unset and treated as zero. More info: https://git.k8s.io/enhancements/keps/sig-node/688-pod-overhead/README.md`,
										ElementType:         types.StringType,
										Optional:            true,
									},
									"preemption_policy": schema.StringAttribute{
										MarkdownDescription: `PreemptionPolicy is the Policy for preempting pods with lower priority. One of Never, PreemptLowerPriority. Defaults to PreemptLowerPriority if unset.`,
										Optional:            true,
									},
									"priority": schema.Int64Attribute{
										MarkdownDescription: `The priority value. Various system components use this field to find the priority of the pod. When Priority Admission Controller is enabled, it prevents users from setting this field. The admission controller populates this field from PriorityClassName. The higher the value, the higher the priority.`,
										Optional:            true,
									},
									"priority_class_name": schema.StringAttribute{
										MarkdownDescription: `If specified, indicates the pod's priority. "system-node-critical" and "system-cluster-critical" are two special keywords which indicate the highest priorities with the former being the highest priority. Any other name must be defined by creating a PriorityClass object with that name. If not specified, the pod priority will be default or zero if there is no default.`,
										Optional:            true,
									},
									"readiness_gates": schema.ListNestedAttribute{
										MarkdownDescription: `If specified, all readiness gates will be evaluated for pod readiness. A pod is ready when all its containers are ready AND all conditions specified in the readiness gates have status equal to "True" More info: https://git.k8s.io/enhancements/keps/sig-network/580-pod-readiness-gates`,
										Optional:            true,

										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"condition_type": schema.StringAttribute{
													MarkdownDescription: `ConditionType refers to a condition in the pod's condition list with matching type.`,
													Optional:            true,
												},
											},
										},
									},
									"resource_claims": schema.ListNestedAttribute{
										MarkdownDescription: `ResourceClaims defines which ResourceClaims must be allocated and reserved before the Pod is allowed to start. The resources will be made available to those containers which consume them by name.

This is an alpha field and requires enabling the DynamicResourceAllocation feature gate.

This field is immutable.`,
										Optional: true,

										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"name": schema.StringAttribute{
													MarkdownDescription: `Name uniquely identifies this resource claim inside the pod. This must be a DNS_LABEL.`,
													Optional:            true,
												},
												"source": schema.SingleNestedAttribute{
													MarkdownDescription: `Source describes where to find the ResourceClaim.`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"resource_claim_name": schema.StringAttribute{
															MarkdownDescription: `ResourceClaimName is the name of a ResourceClaim object in the same namespace as this pod.`,
															Optional:            true,
														},
														"resource_claim_template_name": schema.StringAttribute{
															MarkdownDescription: `ResourceClaimTemplateName is the name of a ResourceClaimTemplate object in the same namespace as this pod.

The template will be used to create a new ResourceClaim, which will be bound to this pod. When this pod is deleted, the ResourceClaim will also be deleted. The pod name and resource name, along with a generated component, will be used to form a unique name for the ResourceClaim, which will be recorded in pod.status.resourceClaimStatuses.

This field is immutable and no changes will be made to the corresponding ResourceClaim by the control plane after creating the ResourceClaim.`,
															Optional: true,
														},
													},
												},
											},
										},
									},
									"restart_policy": schema.StringAttribute{
										MarkdownDescription: `Restart policy for all containers within the pod. One of Always, OnFailure, Never. In some contexts, only a subset of those values may be permitted. Default to Always. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#restart-policy`,
										Optional:            true,
									},
									"runtime_class_name": schema.StringAttribute{
										MarkdownDescription: `RuntimeClassName refers to a RuntimeClass object in the node.k8s.io group, which should be used to run this pod.  If no RuntimeClass resource matches the named class, the pod will not be run. If unset or empty, the "legacy" RuntimeClass will be used, which is an implicit class with an empty definition that uses the default runtime handler. More info: https://git.k8s.io/enhancements/keps/sig-node/585-runtime-class`,
										Optional:            true,
									},
									"scheduler_name": schema.StringAttribute{
										MarkdownDescription: `If specified, the pod will be dispatched by specified scheduler. If not specified, the pod will be dispatched by default scheduler.`,
										Optional:            true,
									},
									"scheduling_gates": schema.ListNestedAttribute{
										MarkdownDescription: `SchedulingGates is an opaque list of values that if specified will block scheduling the pod. If schedulingGates is not empty, the pod will stay in the SchedulingGated state and the scheduler will not attempt to schedule the pod.

SchedulingGates can only be set at pod creation time, and be removed only afterwards.

This is a beta feature enabled by the PodSchedulingReadiness feature gate.`,
										Optional: true,

										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"name": schema.StringAttribute{
													MarkdownDescription: `Name of the scheduling gate. Each scheduling gate must have a unique name field.`,
													Optional:            true,
												},
											},
										},
									},
									"security_context": schema.SingleNestedAttribute{
										MarkdownDescription: `SecurityContext holds pod-level security attributes and common container settings. Optional: Defaults to empty.  See type description for default values of each field.`,
										Optional:            true,

										Attributes: map[string]schema.Attribute{
											"fs_group": schema.Int64Attribute{
												MarkdownDescription: `A special supplemental group that applies to all containers in a pod. Some volume types allow the Kubelet to change the ownership of that volume to be owned by the pod:

1. The owning GID will be the FSGroup 2. The setgid bit is set (new files created in the volume will be owned by FSGroup) 3. The permission bits are OR'd with rw-rw----

If unset, the Kubelet will not modify the ownership and permissions of any volume. Note that this field cannot be set when spec.os.name is windows.`,
												Optional: true,
											},
											"fs_group_change_policy": schema.StringAttribute{
												MarkdownDescription: `fsGroupChangePolicy defines behavior of changing ownership and permission of the volume before being exposed inside Pod. This field will only apply to volume types which support fsGroup based ownership(and permissions). It will have no effect on ephemeral volume types such as: secret, configmaps and emptydir. Valid values are "OnRootMismatch" and "Always". If not specified, "Always" is used. Note that this field cannot be set when spec.os.name is windows.`,
												Optional:            true,
											},
											"run_as_group": schema.Int64Attribute{
												MarkdownDescription: `The GID to run the entrypoint of the container process. Uses runtime default if unset. May also be set in SecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence for that container. Note that this field cannot be set when spec.os.name is windows.`,
												Optional:            true,
											},
											"run_as_non_root": schema.BoolAttribute{
												MarkdownDescription: `Indicates that the container must run as a non-root user. If true, the Kubelet will validate the image at runtime to ensure that it does not run as UID 0 (root) and fail to start the container if it does. If unset or false, no such validation will be performed. May also be set in SecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.`,
												Optional:            true,
											},
											"run_as_user": schema.Int64Attribute{
												MarkdownDescription: `The UID to run the entrypoint of the container process. Defaults to user specified in image metadata if unspecified. May also be set in SecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence for that container. Note that this field cannot be set when spec.os.name is windows.`,
												Optional:            true,
											},
											"se_linux_options": schema.SingleNestedAttribute{
												MarkdownDescription: `The SELinux context to be applied to all containers. If unspecified, the container runtime will allocate a random SELinux context for each container.  May also be set in SecurityContext.  If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence for that container. Note that this field cannot be set when spec.os.name is windows.`,
												Optional:            true,

												Attributes: map[string]schema.Attribute{
													"level": schema.StringAttribute{
														MarkdownDescription: `Level is SELinux level label that applies to the container.`,
														Optional:            true,
													},
													"role": schema.StringAttribute{
														MarkdownDescription: `Role is a SELinux role label that applies to the container.`,
														Optional:            true,
													},
													"type": schema.StringAttribute{
														MarkdownDescription: `Type is a SELinux type label that applies to the container.`,
														Optional:            true,
													},
													"user": schema.StringAttribute{
														MarkdownDescription: `User is a SELinux user label that applies to the container.`,
														Optional:            true,
													},
												},
											},
											"seccomp_profile": schema.SingleNestedAttribute{
												MarkdownDescription: `The seccomp options to use by the containers in this pod. Note that this field cannot be set when spec.os.name is windows.`,
												Optional:            true,

												Attributes: map[string]schema.Attribute{
													"localhost_profile": schema.StringAttribute{
														MarkdownDescription: `localhostProfile indicates a profile defined in a file on the node should be used. The profile must be preconfigured on the node to work. Must be a descending path, relative to the kubelet's configured seccomp profile location. Must be set if type is "Localhost". Must NOT be set for any other type.`,
														Optional:            true,
													},
													"type": schema.StringAttribute{
														MarkdownDescription: `type indicates which kind of seccomp profile will be applied. Valid options are:

Localhost - a profile defined in a file on the node should be used. RuntimeDefault - the container runtime default profile should be used. Unconfined - no profile should be applied.`,
														Optional: true,
													},
												},
											},
											"supplemental_groups": schema.ListAttribute{
												MarkdownDescription: `A list of groups applied to the first process run in each container, in addition to the container's primary GID, the fsGroup (if specified), and group memberships defined in the container image for the uid of the container process. If unspecified, no additional groups are added to any container. Note that group memberships defined in the container image for the uid of the container process are still effective, even if they are not included in this list. Note that this field cannot be set when spec.os.name is windows.`,
												ElementType:         types.Int64Type,
												Optional:            true,
											},
											"sysctls": schema.ListNestedAttribute{
												MarkdownDescription: `Sysctls hold a list of namespaced sysctls used for the pod. Pods with unsupported sysctls (by the container runtime) might fail to launch. Note that this field cannot be set when spec.os.name is windows.`,
												Optional:            true,

												NestedObject: schema.NestedAttributeObject{
													Attributes: map[string]schema.Attribute{
														"name": schema.StringAttribute{
															MarkdownDescription: `Name of a property to set`,
															Optional:            true,
														},
														"value": schema.StringAttribute{
															MarkdownDescription: `Value of a property to set`,
															Optional:            true,
														},
													},
												},
											},
											"windows_options": schema.SingleNestedAttribute{
												MarkdownDescription: `The Windows specific settings applied to all containers. If unspecified, the options within a container's SecurityContext will be used. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence. Note that this field cannot be set when spec.os.name is linux.`,
												Optional:            true,

												Attributes: map[string]schema.Attribute{
													"gmsa_credential_spec": schema.StringAttribute{
														MarkdownDescription: `GMSACredentialSpec is where the GMSA admission webhook (https://github.com/kubernetes-sigs/windows-gmsa) inlines the contents of the GMSA credential spec named by the GMSACredentialSpecName field.`,
														Optional:            true,
													},
													"gmsa_credential_spec_name": schema.StringAttribute{
														MarkdownDescription: `GMSACredentialSpecName is the name of the GMSA credential spec to use.`,
														Optional:            true,
													},
													"host_process": schema.BoolAttribute{
														MarkdownDescription: `HostProcess determines if a container should be run as a 'Host Process' container. All of a Pod's containers must have the same effective HostProcess value (it is not allowed to have a mix of HostProcess containers and non-HostProcess containers). In addition, if HostProcess is true then HostNetwork must also be set to true.`,
														Optional:            true,
													},
													"run_as_user_name": schema.StringAttribute{
														MarkdownDescription: `The UserName in Windows to run the entrypoint of the container process. Defaults to the user specified in image metadata if unspecified. May also be set in PodSecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.`,
														Optional:            true,
													},
												},
											},
										},
									},
									"service_account": schema.StringAttribute{
										MarkdownDescription: `DeprecatedServiceAccount is a depreciated alias for ServiceAccountName. Deprecated: Use serviceAccountName instead.`,
										Optional:            true,
									},
									"service_account_name": schema.StringAttribute{
										MarkdownDescription: `ServiceAccountName is the name of the ServiceAccount to use to run this pod. More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/`,
										Optional:            true,
									},
									"set_hostname_as_fqdn": schema.BoolAttribute{
										MarkdownDescription: `If true the pod's hostname will be configured as the pod's FQDN, rather than the leaf name (the default). In Linux containers, this means setting the FQDN in the hostname field of the kernel (the nodename field of struct utsname). In Windows containers, this means setting the registry value of hostname for the registry key HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters to FQDN. If a pod does not have FQDN, this has no effect. Default to false.`,
										Optional:            true,
									},
									"share_process_namespace": schema.BoolAttribute{
										MarkdownDescription: `Share a single process namespace between all of the containers in a pod. When this is set containers will be able to view and signal processes from other containers in the same pod, and the first process in each container will not be assigned PID 1. HostPID and ShareProcessNamespace cannot both be set. Optional: Default to false.`,
										Optional:            true,
									},
									"subdomain": schema.StringAttribute{
										MarkdownDescription: `If specified, the fully qualified Pod hostname will be "<hostname>.<subdomain>.<pod namespace>.svc.<cluster domain>". If not specified, the pod will not have a domainname at all.`,
										Optional:            true,
									},
									"termination_grace_period_seconds": schema.Int64Attribute{
										MarkdownDescription: `Optional duration in seconds the pod needs to terminate gracefully. May be decreased in delete request. Value must be non-negative integer. The value zero indicates stop immediately via the kill signal (no opportunity to shut down). If this value is nil, the default grace period will be used instead. The grace period is the duration in seconds after the processes running in the pod are sent a termination signal and the time when the processes are forcibly halted with a kill signal. Set this value longer than the expected cleanup time for your process. Defaults to 30 seconds.`,
										Optional:            true,
									},
									"tolerations": schema.ListNestedAttribute{
										MarkdownDescription: `If specified, the pod's tolerations.`,
										Optional:            true,

										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"effect": schema.StringAttribute{
													MarkdownDescription: `Effect indicates the taint effect to match. Empty means match all taint effects. When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.`,
													Optional:            true,
												},
												"key": schema.StringAttribute{
													MarkdownDescription: `Key is the taint key that the toleration applies to. Empty means match all taint keys. If the key is empty, operator must be Exists; this combination means to match all values and all keys.`,
													Optional:            true,
												},
												"operator": schema.StringAttribute{
													MarkdownDescription: `Operator represents a key's relationship to the value. Valid operators are Exists and Equal. Defaults to Equal. Exists is equivalent to wildcard for value, so that a pod can tolerate all taints of a particular category.`,
													Optional:            true,
												},
												"toleration_seconds": schema.Int64Attribute{
													MarkdownDescription: `TolerationSeconds represents the period of time the toleration (which must be of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default, it is not set, which means tolerate the taint forever (do not evict). Zero and negative values will be treated as 0 (evict immediately) by the system.`,
													Optional:            true,
												},
												"value": schema.StringAttribute{
													MarkdownDescription: `Value is the taint value the toleration matches to. If the operator is Exists, the value should be empty, otherwise just a regular string.`,
													Optional:            true,
												},
											},
										},
									},
									"topology_spread_constraints": schema.ListNestedAttribute{
										MarkdownDescription: `TopologySpreadConstraints describes how a group of pods ought to spread across topology domains. Scheduler will schedule pods in a way which abides by the constraints. All topologySpreadConstraints are ANDed.`,
										Optional:            true,

										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"label_selector": schema.SingleNestedAttribute{
													MarkdownDescription: `LabelSelector is used to find matching pods. Pods that match this label selector are counted to determine the number of pods in their corresponding topology domain.`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"match_expressions": schema.ListNestedAttribute{
															MarkdownDescription: `matchExpressions is a list of label selector requirements. The requirements are ANDed.`,
															Optional:            true,

															NestedObject: schema.NestedAttributeObject{
																Attributes: map[string]schema.Attribute{
																	"key": schema.StringAttribute{
																		MarkdownDescription: `key is the label key that the selector applies to.`,
																		Optional:            true,
																	},
																	"operator": schema.StringAttribute{
																		MarkdownDescription: `operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.`,
																		Optional:            true,
																	},
																	"values": schema.ListAttribute{
																		MarkdownDescription: `values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.`,
																		ElementType:         types.StringType,
																		Optional:            true,
																	},
																},
															},
														},
														"match_labels": schema.MapAttribute{
															MarkdownDescription: `matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.`,
															ElementType:         types.StringType,
															Optional:            true,
														},
													},
												},
												"match_label_keys": schema.ListAttribute{
													MarkdownDescription: `MatchLabelKeys is a set of pod label keys to select the pods over which spreading will be calculated. The keys are used to lookup values from the incoming pod labels, those key-value labels are ANDed with labelSelector to select the group of existing pods over which spreading will be calculated for the incoming pod. The same key is forbidden to exist in both MatchLabelKeys and LabelSelector. MatchLabelKeys cannot be set when LabelSelector isn't set. Keys that don't exist in the incoming pod labels will be ignored. A null or empty list means only match against labelSelector.

This is a beta field and requires the MatchLabelKeysInPodTopologySpread feature gate to be enabled (enabled by default).`,
													ElementType: types.StringType,
													Optional:    true,
												},
												"max_skew": schema.Int64Attribute{
													MarkdownDescription: `MaxSkew describes the degree to which pods may be unevenly distributed. When whenUnsatisfiable=DoNotSchedule, it is the maximum permitted difference between the number of matching pods in the target topology and the global minimum. The global minimum is the minimum number of matching pods in an eligible domain or zero if the number of eligible domains is less than MinDomains. For example, in a 3-zone cluster, MaxSkew is set to 1, and pods with the same labelSelector spread as 2/2/1: In this case, the global minimum is 1. | zone1 | zone2 | zone3 | |  P P  |  P P  |   P   | - if MaxSkew is 1, incoming pod can only be scheduled to zone3 to become 2/2/2; scheduling it onto zone1(zone2) would make the ActualSkew(3-1) on zone1(zone2) violate MaxSkew(1). - if MaxSkew is 2, incoming pod can be scheduled onto any zone. When whenUnsatisfiable=ScheduleAnyway, it is used to give higher precedence to topologies that satisfy it. It's a required field. Default value is 1 and 0 is not allowed.`,
													Optional:            true,
												},
												"min_domains": schema.Int64Attribute{
													MarkdownDescription: `MinDomains indicates a minimum number of eligible domains. When the number of eligible domains with matching topology keys is less than minDomains, Pod Topology Spread treats "global minimum" as 0, and then the calculation of Skew is performed. And when the number of eligible domains with matching topology keys equals or greater than minDomains, this value has no effect on scheduling. As a result, when the number of eligible domains is less than minDomains, scheduler won't schedule more than maxSkew Pods to those domains. If value is nil, the constraint behaves as if MinDomains is equal to 1. Valid values are integers greater than 0. When value is not nil, WhenUnsatisfiable must be DoNotSchedule.

For example, in a 3-zone cluster, MaxSkew is set to 2, MinDomains is set to 5 and pods with the same labelSelector spread as 2/2/2: | zone1 | zone2 | zone3 | |  P P  |  P P  |  P P  | The number of domains is less than 5(MinDomains), so "global minimum" is treated as 0. In this situation, new pod with the same labelSelector cannot be scheduled, because computed skew will be 3(3 - 0) if new Pod is scheduled to any of the three zones, it will violate MaxSkew.

This is a beta field and requires the MinDomainsInPodTopologySpread feature gate to be enabled (enabled by default).`,
													Optional: true,
												},
												"node_affinity_policy": schema.StringAttribute{
													MarkdownDescription: `NodeAffinityPolicy indicates how we will treat Pod's nodeAffinity/nodeSelector when calculating pod topology spread skew. Options are: - Honor: only nodes matching nodeAffinity/nodeSelector are included in the calculations. - Ignore: nodeAffinity/nodeSelector are ignored. All nodes are included in the calculations.

If this value is nil, the behavior is equivalent to the Honor policy. This is a beta-level feature default enabled by the NodeInclusionPolicyInPodTopologySpread feature flag.`,
													Optional: true,
												},
												"node_taints_policy": schema.StringAttribute{
													MarkdownDescription: `NodeTaintsPolicy indicates how we will treat node taints when calculating pod topology spread skew. Options are: - Honor: nodes without taints, along with tainted nodes for which the incoming pod has a toleration, are included. - Ignore: node taints are ignored. All nodes are included.

If this value is nil, the behavior is equivalent to the Ignore policy. This is a beta-level feature default enabled by the NodeInclusionPolicyInPodTopologySpread feature flag.`,
													Optional: true,
												},
												"topology_key": schema.StringAttribute{
													MarkdownDescription: `TopologyKey is the key of node labels. Nodes that have a label with this key and identical values are considered to be in the same topology. We consider each <key, value> as a "bucket", and try to put balanced number of pods into each bucket. We define a domain as a particular instance of a topology. Also, we define an eligible domain as a domain whose nodes meet the requirements of nodeAffinityPolicy and nodeTaintsPolicy. e.g. If TopologyKey is "kubernetes.io/hostname", each Node is a domain of that topology. And, if TopologyKey is "topology.kubernetes.io/zone", each zone is a domain of that topology. It's a required field.`,
													Optional:            true,
												},
												"when_unsatisfiable": schema.StringAttribute{
													MarkdownDescription: `WhenUnsatisfiable indicates how to deal with a pod if it doesn't satisfy the spread constraint. - DoNotSchedule (default) tells the scheduler not to schedule it. - ScheduleAnyway tells the scheduler to schedule the pod in any location,
  but giving higher precedence to topologies that would help reduce the
  skew.
A constraint is considered "Unsatisfiable" for an incoming pod if and only if every possible node assignment for that pod would violate "MaxSkew" on some topology. For example, in a 3-zone cluster, MaxSkew is set to 1, and pods with the same labelSelector spread as 3/1/1: | zone1 | zone2 | zone3 | | P P P |   P   |   P   | If WhenUnsatisfiable is set to DoNotSchedule, incoming pod can only be scheduled to zone2(zone3) to become 3/2/1(3/1/2) as ActualSkew(2-1) on zone2(zone3) satisfies MaxSkew(1). In other words, the cluster can still be imbalanced, but scheduler won't make it *more* imbalanced. It's a required field.`,
													Optional: true,
												},
											},
										},
									},
									"volumes": schema.ListNestedAttribute{
										MarkdownDescription: `List of volumes that can be mounted by containers belonging to the pod. More info: https://kubernetes.io/docs/concepts/storage/volumes`,
										Optional:            true,

										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"aws_elastic_block_store": schema.SingleNestedAttribute{
													MarkdownDescription: `awsElasticBlockStore represents an AWS Disk resource that is attached to a kubelet's host machine and then exposed to the pod. More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"fs_type": schema.StringAttribute{
															MarkdownDescription: `fsType is the filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore`,
															Optional:            true,
														},
														"partition": schema.Int64Attribute{
															MarkdownDescription: `partition is the partition in the volume that you want to mount. If omitted, the default is to mount by volume name. Examples: For volume /dev/sda1, you specify the partition as "1". Similarly, the volume partition for /dev/sda is "0" (or you can leave the property empty).`,
															Optional:            true,
														},
														"read_only": schema.BoolAttribute{
															MarkdownDescription: `readOnly value true will force the readOnly setting in VolumeMounts. More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore`,
															Optional:            true,
														},
														"volume_id": schema.StringAttribute{
															MarkdownDescription: `volumeID is unique ID of the persistent disk resource in AWS (Amazon EBS volume). More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore`,
															Optional:            true,
														},
													},
												},
												"azure_disk": schema.SingleNestedAttribute{
													MarkdownDescription: `azureDisk represents an Azure Data Disk mount on the host and bind mount to the pod.`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"caching_mode": schema.StringAttribute{
															MarkdownDescription: `cachingMode is the Host Caching mode: None, Read Only, Read Write.`,
															Optional:            true,
														},
														"disk_name": schema.StringAttribute{
															MarkdownDescription: `diskName is the Name of the data disk in the blob storage`,
															Optional:            true,
														},
														"disk_uri": schema.StringAttribute{
															MarkdownDescription: `diskURI is the URI of data disk in the blob storage`,
															Optional:            true,
														},
														"fs_type": schema.StringAttribute{
															MarkdownDescription: `fsType is Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.`,
															Optional:            true,
														},
														"kind": schema.StringAttribute{
															MarkdownDescription: `kind expected values are Shared: multiple blob disks per storage account  Dedicated: single blob disk per storage account  Managed: azure managed data disk (only in managed availability set). defaults to shared`,
															Optional:            true,
														},
														"read_only": schema.BoolAttribute{
															MarkdownDescription: `readOnly Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts.`,
															Optional:            true,
														},
													},
												},
												"azure_file": schema.SingleNestedAttribute{
													MarkdownDescription: `azureFile represents an Azure File Service mount on the host and bind mount to the pod.`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"read_only": schema.BoolAttribute{
															MarkdownDescription: `readOnly defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts.`,
															Optional:            true,
														},
														"secret_name": schema.StringAttribute{
															MarkdownDescription: `secretName is the  name of secret that contains Azure Storage Account Name and Key`,
															Optional:            true,
														},
														"share_name": schema.StringAttribute{
															MarkdownDescription: `shareName is the azure share Name`,
															Optional:            true,
														},
													},
												},
												"cephfs": schema.SingleNestedAttribute{
													MarkdownDescription: `cephFS represents a Ceph FS mount on the host that shares a pod's lifetime`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"monitors": schema.ListAttribute{
															MarkdownDescription: `monitors is Required: Monitors is a collection of Ceph monitors More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it`,
															ElementType:         types.StringType,
															Optional:            true,
														},
														"path": schema.StringAttribute{
															MarkdownDescription: `path is Optional: Used as the mounted root, rather than the full Ceph tree, default is /`,
															Optional:            true,
														},
														"read_only": schema.BoolAttribute{
															MarkdownDescription: `readOnly is Optional: Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it`,
															Optional:            true,
														},
														"secret_file": schema.StringAttribute{
															MarkdownDescription: `secretFile is Optional: SecretFile is the path to key ring for User, default is /etc/ceph/user.secret More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it`,
															Optional:            true,
														},
														"secret_ref": schema.SingleNestedAttribute{
															MarkdownDescription: `secretRef is Optional: SecretRef is reference to the authentication secret for User, default is empty. More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"name": schema.StringAttribute{
																	MarkdownDescription: `Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names`,
																	Optional:            true,
																},
															},
														},
														"user": schema.StringAttribute{
															MarkdownDescription: `user is optional: User is the rados user name, default is admin More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it`,
															Optional:            true,
														},
													},
												},
												"cinder": schema.SingleNestedAttribute{
													MarkdownDescription: `cinder represents a cinder volume attached and mounted on kubelets host machine. More info: https://examples.k8s.io/mysql-cinder-pd/README.md`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"fs_type": schema.StringAttribute{
															MarkdownDescription: `fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. More info: https://examples.k8s.io/mysql-cinder-pd/README.md`,
															Optional:            true,
														},
														"read_only": schema.BoolAttribute{
															MarkdownDescription: `readOnly defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. More info: https://examples.k8s.io/mysql-cinder-pd/README.md`,
															Optional:            true,
														},
														"secret_ref": schema.SingleNestedAttribute{
															MarkdownDescription: `secretRef is optional: points to a secret object containing parameters used to connect to OpenStack.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"name": schema.StringAttribute{
																	MarkdownDescription: `Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names`,
																	Optional:            true,
																},
															},
														},
														"volume_id": schema.StringAttribute{
															MarkdownDescription: `volumeID used to identify the volume in cinder. More info: https://examples.k8s.io/mysql-cinder-pd/README.md`,
															Optional:            true,
														},
													},
												},
												"config_map": schema.SingleNestedAttribute{
													MarkdownDescription: `configMap represents a configMap that should populate this volume`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"default_mode": schema.Int64Attribute{
															MarkdownDescription: `defaultMode is optional: mode bits used to set permissions on created files by default. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. Defaults to 0644. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.`,
															Optional:            true,
														},
														"items": schema.ListNestedAttribute{
															MarkdownDescription: `items if unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the ConfigMap, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'.`,
															Optional:            true,

															NestedObject: schema.NestedAttributeObject{
																Attributes: map[string]schema.Attribute{
																	"key": schema.StringAttribute{
																		MarkdownDescription: `key is the key to project.`,
																		Optional:            true,
																	},
																	"mode": schema.Int64Attribute{
																		MarkdownDescription: `mode is Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.`,
																		Optional:            true,
																	},
																	"path": schema.StringAttribute{
																		MarkdownDescription: `path is the relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.`,
																		Optional:            true,
																	},
																},
															},
														},
														"name": schema.StringAttribute{
															MarkdownDescription: `Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names`,
															Optional:            true,
														},
														"optional": schema.BoolAttribute{
															MarkdownDescription: `optional specify whether the ConfigMap or its keys must be defined`,
															Optional:            true,
														},
													},
												},
												"csi": schema.SingleNestedAttribute{
													MarkdownDescription: `csi (Container Storage Interface) represents ephemeral storage that is handled by certain external CSI drivers (Beta feature).`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"driver": schema.StringAttribute{
															MarkdownDescription: `driver is the name of the CSI driver that handles this volume. Consult with your admin for the correct name as registered in the cluster.`,
															Optional:            true,
														},
														"fs_type": schema.StringAttribute{
															MarkdownDescription: `fsType to mount. Ex. "ext4", "xfs", "ntfs". If not provided, the empty value is passed to the associated CSI driver which will determine the default filesystem to apply.`,
															Optional:            true,
														},
														"node_publish_secret_ref": schema.SingleNestedAttribute{
															MarkdownDescription: `nodePublishSecretRef is a reference to the secret object containing sensitive information to pass to the CSI driver to complete the CSI NodePublishVolume and NodeUnpublishVolume calls. This field is optional, and  may be empty if no secret is required. If the secret object contains more than one secret, all secret references are passed.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"name": schema.StringAttribute{
																	MarkdownDescription: `Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names`,
																	Optional:            true,
																},
															},
														},
														"read_only": schema.BoolAttribute{
															MarkdownDescription: `readOnly specifies a read-only configuration for the volume. Defaults to false (read/write).`,
															Optional:            true,
														},
														"volume_attributes": schema.MapAttribute{
															MarkdownDescription: `volumeAttributes stores driver-specific properties that are passed to the CSI driver. Consult your driver's documentation for supported values.`,
															ElementType:         types.StringType,
															Optional:            true,
														},
													},
												},
												"downward_api": schema.SingleNestedAttribute{
													MarkdownDescription: `downwardAPI represents downward API about the pod that should populate this volume`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"default_mode": schema.Int64Attribute{
															MarkdownDescription: `Optional: mode bits to use on created files by default. Must be a Optional: mode bits used to set permissions on created files by default. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. Defaults to 0644. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.`,
															Optional:            true,
														},
														"items": schema.ListNestedAttribute{
															MarkdownDescription: `Items is a list of downward API volume file`,
															Optional:            true,

															NestedObject: schema.NestedAttributeObject{
																Attributes: map[string]schema.Attribute{
																	"field_ref": schema.SingleNestedAttribute{
																		MarkdownDescription: `Required: Selects a field of the pod: only annotations, labels, name and namespace are supported.`,
																		Optional:            true,

																		Attributes: map[string]schema.Attribute{
																			"api_version": schema.StringAttribute{
																				MarkdownDescription: `Version of the schema the FieldPath is written in terms of, defaults to "v1".`,
																				Optional:            true,
																			},
																			"field_path": schema.StringAttribute{
																				MarkdownDescription: `Path of the field to select in the specified API version.`,
																				Optional:            true,
																			},
																		},
																	},
																	"mode": schema.Int64Attribute{
																		MarkdownDescription: `Optional: mode bits used to set permissions on this file, must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.`,
																		Optional:            true,
																	},
																	"path": schema.StringAttribute{
																		MarkdownDescription: `Required: Path is  the relative path name of the file to be created. Must not be absolute or contain the '..' path. Must be utf-8 encoded. The first item of the relative path must not start with '..'`,
																		Optional:            true,
																	},
																	"resource_field_ref": schema.SingleNestedAttribute{
																		MarkdownDescription: `Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.`,
																		Optional:            true,

																		Attributes: map[string]schema.Attribute{
																			"container_name": schema.StringAttribute{
																				MarkdownDescription: `Container name: required for volumes, optional for env vars`,
																				Optional:            true,
																			},
																			"divisor": schema.StringAttribute{
																				MarkdownDescription: `Specifies the output format of the exposed resources, defaults to "1"`,
																				Optional:            true,
																			},
																			"resource": schema.StringAttribute{
																				MarkdownDescription: `Required: resource to select`,
																				Optional:            true,
																			},
																		},
																	},
																},
															},
														},
													},
												},
												"empty_dir": schema.SingleNestedAttribute{
													MarkdownDescription: `emptyDir represents a temporary directory that shares a pod's lifetime. More info: https://kubernetes.io/docs/concepts/storage/volumes#emptydir`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"medium": schema.StringAttribute{
															MarkdownDescription: `medium represents what type of storage medium should back this directory. The default is "" which means to use the node's default medium. Must be an empty string (default) or Memory. More info: https://kubernetes.io/docs/concepts/storage/volumes#emptydir`,
															Optional:            true,
														},
														"size_limit": schema.StringAttribute{
															MarkdownDescription: `sizeLimit is the total amount of local storage required for this EmptyDir volume. The size limit is also applicable for memory medium. The maximum usage on memory medium EmptyDir would be the minimum value between the SizeLimit specified here and the sum of memory limits of all containers in a pod. The default is nil which means that the limit is undefined. More info: https://kubernetes.io/docs/concepts/storage/volumes#emptydir`,
															Optional:            true,
														},
													},
												},
												"ephemeral": schema.SingleNestedAttribute{
													MarkdownDescription: `ephemeral represents a volume that is handled by a cluster storage driver. The volume's lifecycle is tied to the pod that defines it - it will be created before the pod starts, and deleted when the pod is removed.

Use this if: a) the volume is only needed while the pod runs, b) features of normal volumes like restoring from snapshot or capacity
   tracking are needed,
c) the storage driver is specified through a storage class, and d) the storage driver supports dynamic volume provisioning through
   a PersistentVolumeClaim (see EphemeralVolumeSource for more
   information on the connection between this volume type
   and PersistentVolumeClaim).

Use PersistentVolumeClaim or one of the vendor-specific APIs for volumes that persist for longer than the lifecycle of an individual pod.

Use CSI for light-weight local ephemeral volumes if the CSI driver is meant to be used that way - see the documentation of the driver for more information.

A pod can use both types of ephemeral volumes and persistent volumes at the same time.`,
													Optional: true,

													Attributes: map[string]schema.Attribute{
														"volume_claim_template": schema.SingleNestedAttribute{
															MarkdownDescription: `Will be used to create a stand-alone PVC to provision the volume. The pod in which this EphemeralVolumeSource is embedded will be the owner of the PVC, i.e. the PVC will be deleted together with the pod.  The name of the PVC will be <pod name>-<volume name> where <volume name> is the name from the PodSpec.Volumes array entry. Pod validation will reject the pod if the concatenated name is not valid for a PVC (for example, too long).

An existing PVC with that name that is not owned by the pod will *not* be used for the pod to avoid using an unrelated volume by mistake. Starting the pod is then blocked until the unrelated PVC is removed. If such a pre-created PVC is meant to be used by the pod, the PVC has to updated with an owner reference to the pod once the pod exists. Normally this should not be necessary, but it may be useful when manually reconstructing a broken cluster.

This field is read-only and no changes will be made by Kubernetes to the PVC after it has been created.

Required, must not be nil.`,
															Optional: true,

															Attributes: map[string]schema.Attribute{
																"metadata": schema.SingleNestedAttribute{
																	MarkdownDescription: `May contain labels and annotations that will be copied into the PVC when creating it. No other fields are allowed and will be rejected during validation.`,
																	Optional:            true,

																	Attributes: map[string]schema.Attribute{
																		"annotations": schema.MapAttribute{
																			MarkdownDescription: `Annotations is an unstructured key value map stored with a resource that may be set by external tools to store and retrieve arbitrary metadata. They are not queryable and should be preserved when modifying objects. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations`,
																			ElementType:         types.StringType,
																			Optional:            true,
																		},
																		"creation_timestamp": schema.StringAttribute{
																			MarkdownDescription: `CreationTimestamp is a timestamp representing the server time when this object was created. It is not guaranteed to be set in happens-before order across separate operations. Clients may not set this value. It is represented in RFC3339 form and is in UTC.

Populated by the system. Read-only. Null for lists. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata`,
																			Optional: true,
																		},
																		"deletion_grace_period_seconds": schema.Int64Attribute{
																			MarkdownDescription: `Number of seconds allowed for this object to gracefully terminate before it will be removed from the system. Only set when deletionTimestamp is also set. May only be shortened. Read-only.`,
																			Optional:            true,
																		},
																		"deletion_timestamp": schema.StringAttribute{
																			MarkdownDescription: `DeletionTimestamp is RFC 3339 date and time at which this resource will be deleted. This field is set by the server when a graceful deletion is requested by the user, and is not directly settable by a client. The resource is expected to be deleted (no longer visible from resource lists, and not reachable by name) after the time in this field, once the finalizers list is empty. As long as the finalizers list contains items, deletion is blocked. Once the deletionTimestamp is set, this value may not be unset or be set further into the future, although it may be shortened or the resource may be deleted prior to this time. For example, a user may request that a pod is deleted in 30 seconds. The Kubelet will react by sending a graceful termination signal to the containers in the pod. After that 30 seconds, the Kubelet will send a hard termination signal (SIGKILL) to the container and after cleanup, remove the pod from the API. In the presence of network partitions, this object may still exist after this timestamp, until an administrator or automated process can determine the resource is fully terminated. If not set, graceful deletion of the object has not been requested.

Populated by the system when a graceful deletion is requested. Read-only. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata`,
																			Optional: true,
																		},
																		"finalizers": schema.ListAttribute{
																			MarkdownDescription: `Must be empty before the object is deleted from the registry. Each entry is an identifier for the responsible component that will remove the entry from the list. If the deletionTimestamp of the object is non-nil, entries in this list can only be removed. Finalizers may be processed and removed in any order.  Order is NOT enforced because it introduces significant risk of stuck finalizers. finalizers is a shared field, any actor with permission can reorder it. If the finalizer list is processed in order, then this can lead to a situation in which the component responsible for the first finalizer in the list is waiting for a signal (field value, external system, or other) produced by a component responsible for a finalizer later in the list, resulting in a deadlock. Without enforced ordering finalizers are free to order amongst themselves and are not vulnerable to ordering changes in the list.`,
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
																		},
																		"labels": schema.MapAttribute{
																			MarkdownDescription: `Map of string keys and values that can be used to organize and categorize (scope and select) objects. May match selectors of replication controllers and services. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels`,
																			ElementType:         types.StringType,
																			Optional:            true,
																		},
																		"managed_fields": schema.ListNestedAttribute{
																			MarkdownDescription: `ManagedFields maps workflow-id and version to the set of fields that are managed by that workflow. This is mostly for internal housekeeping, and users typically shouldn't need to set or understand this field. A workflow can be the user's name, a controller's name, or the name of a specific apply path like "ci-cd". The set of fields is always in the version that the workflow used when modifying the object.`,
																			Optional:            true,

																			NestedObject: schema.NestedAttributeObject{
																				Attributes: map[string]schema.Attribute{
																					"api_version": schema.StringAttribute{
																						MarkdownDescription: `APIVersion defines the version of this resource that this field set applies to. The format is "group/version" just like the top-level APIVersion field. It is necessary to track the version of a field set because it cannot be automatically converted.`,
																						Optional:            true,
																					},
																					"fields_type": schema.StringAttribute{
																						MarkdownDescription: `FieldsType is the discriminator for the different fields format and version. There is currently only one possible value: "FieldsV1"`,
																						Optional:            true,
																					},
																					"fields_v1": schema.SingleNestedAttribute{
																						MarkdownDescription: `FieldsV1 holds the first JSON version format as described in the "FieldsV1" type.`,
																						Optional:            true,
																					},
																					"manager": schema.StringAttribute{
																						MarkdownDescription: `Manager is an identifier of the workflow managing these fields.`,
																						Optional:            true,
																					},
																					"operation": schema.StringAttribute{
																						MarkdownDescription: `Operation is the type of operation which lead to this ManagedFieldsEntry being created. The only valid values for this field are 'Apply' and 'Update'.`,
																						Optional:            true,
																					},
																					"subresource": schema.StringAttribute{
																						MarkdownDescription: `Subresource is the name of the subresource used to update that object, or empty string if the object was updated through the main resource. The value of this field is used to distinguish between managers, even if they share the same name. For example, a status update will be distinct from a regular update using the same manager name. Note that the APIVersion field is not related to the Subresource field and it always corresponds to the version of the main resource.`,
																						Optional:            true,
																					},
																					"time": schema.StringAttribute{
																						MarkdownDescription: `Time is the timestamp of when the ManagedFields entry was added. The timestamp will also be updated if a field is added, the manager changes any of the owned fields value or removes a field. The timestamp does not update when a field is removed from the entry because another manager took it over.`,
																						Optional:            true,
																					},
																				},
																			},
																		},
																		"name": schema.StringAttribute{
																			MarkdownDescription: `Name must be unique within a namespace. Is required when creating resources, although some resources may allow a client to request the generation of an appropriate name automatically. Name is primarily intended for creation idempotence and configuration definition. Cannot be updated. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#names`,
																			Optional:            true,
																		},
																		"namespace": schema.StringAttribute{
																			MarkdownDescription: `Namespace defines the space within which each name must be unique. An empty namespace is equivalent to the "default" namespace, but "default" is the canonical representation. Not all objects are required to be scoped to a namespace - the value of this field for those objects will be empty.

Must be a DNS_LABEL. Cannot be updated. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces`,
																			Optional: true,
																		},
																		"owner_references": schema.ListNestedAttribute{
																			MarkdownDescription: `List of objects depended by this object. If ALL objects in the list have been deleted, this object will be garbage collected. If this object is managed by a controller, then an entry in this list will point to this controller, with the controller field set to true. There cannot be more than one managing controller.`,
																			Optional:            true,

																			NestedObject: schema.NestedAttributeObject{
																				Attributes: map[string]schema.Attribute{
																					"api_version": schema.StringAttribute{
																						MarkdownDescription: `API version of the referent.`,
																						Optional:            true,
																					},
																					"block_owner_deletion": schema.BoolAttribute{
																						MarkdownDescription: `If true, AND if the owner has the "foregroundDeletion" finalizer, then the owner cannot be deleted from the key-value store until this reference is removed. See https://kubernetes.io/docs/concepts/architecture/garbage-collection/#foreground-deletion for how the garbage collector interacts with this field and enforces the foreground deletion. Defaults to false. To set this field, a user needs "delete" permission of the owner, otherwise 422 (Unprocessable Entity) will be returned.`,
																						Optional:            true,
																					},
																					"controller": schema.BoolAttribute{
																						MarkdownDescription: `If true, this reference points to the managing controller.`,
																						Optional:            true,
																					},
																					"kind": schema.StringAttribute{
																						MarkdownDescription: `Kind of the referent. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds`,
																						Optional:            true,
																					},
																					"name": schema.StringAttribute{
																						MarkdownDescription: `Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#names`,
																						Optional:            true,
																					},
																					"uid": schema.StringAttribute{
																						MarkdownDescription: `UID of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#uids`,
																						Optional:            true,
																					},
																				},
																			},
																		},
																		"resource_version": schema.StringAttribute{
																			MarkdownDescription: `An opaque value that represents the internal version of this object that can be used by clients to determine when objects have changed. May be used for optimistic concurrency, change detection, and the watch operation on a resource or set of resources. Clients must treat these values as opaque and passed unmodified back to the server. They may only be valid for a particular resource or set of resources.

Populated by the system. Read-only. Value must be treated as opaque by clients and . More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency`,
																			Optional: true,
																		},
																		"self_link": schema.StringAttribute{
																			MarkdownDescription: `Deprecated: selfLink is a legacy read-only field that is no longer populated by the system.`,
																			Optional:            true,
																		},
																		"uid": schema.StringAttribute{
																			MarkdownDescription: `UID is the unique in time and space value for this object. It is typically generated by the server on successful creation of a resource and is not allowed to change on PUT operations.

Populated by the system. Read-only. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#uids`,
																			Optional: true,
																		},
																	},
																},
																"spec": schema.SingleNestedAttribute{
																	MarkdownDescription: `The specification for the PersistentVolumeClaim. The entire content is copied unchanged into the PVC that gets created from this template. The same fields as in a PersistentVolumeClaim are also valid here.`,
																	Optional:            true,

																	Attributes: map[string]schema.Attribute{
																		"access_modes": schema.ListAttribute{
																			MarkdownDescription: `accessModes contains the desired access modes the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1`,
																			ElementType:         types.StringType,
																			Optional:            true,
																		},
																		"data_source": schema.SingleNestedAttribute{
																			MarkdownDescription: `dataSource field can be used to specify either: * An existing VolumeSnapshot object (snapshot.storage.k8s.io/VolumeSnapshot) * An existing PVC (PersistentVolumeClaim) If the provisioner or an external controller can support the specified data source, it will create a new volume based on the contents of the specified data source. When the AnyVolumeDataSource feature gate is enabled, dataSource contents will be copied to dataSourceRef, and dataSourceRef contents will be copied to dataSource when dataSourceRef.namespace is not specified. If the namespace is specified, then dataSourceRef will not be copied to dataSource.`,
																			Optional:            true,

																			Attributes: map[string]schema.Attribute{
																				"api_group": schema.StringAttribute{
																					MarkdownDescription: `APIGroup is the group for the resource being referenced. If APIGroup is not specified, the specified Kind must be in the core API group. For any other third-party types, APIGroup is required.`,
																					Optional:            true,
																				},
																				"kind": schema.StringAttribute{
																					MarkdownDescription: `Kind is the type of resource being referenced`,
																					Optional:            true,
																				},
																				"name": schema.StringAttribute{
																					MarkdownDescription: `Name is the name of resource being referenced`,
																					Optional:            true,
																				},
																			},
																		},
																		"data_source_ref": schema.SingleNestedAttribute{
																			MarkdownDescription: `dataSourceRef specifies the object from which to populate the volume with data, if a non-empty volume is desired. This may be any object from a non-empty API group (non core object) or a PersistentVolumeClaim object. When this field is specified, volume binding will only succeed if the type of the specified object matches some installed volume populator or dynamic provisioner. This field will replace the functionality of the dataSource field and as such if both fields are non-empty, they must have the same value. For backwards compatibility, when namespace isn't specified in dataSourceRef, both fields (dataSource and dataSourceRef) will be set to the same value automatically if one of them is empty and the other is non-empty. When namespace is specified in dataSourceRef, dataSource isn't set to the same value and must be empty. There are three important differences between dataSource and dataSourceRef: * While dataSource only allows two specific types of objects, dataSourceRef
  allows any non-core object, as well as PersistentVolumeClaim objects.
* While dataSource ignores disallowed values (dropping them), dataSourceRef
  preserves all values, and generates an error if a disallowed value is
  specified.
* While dataSource only allows local objects, dataSourceRef allows objects
  in any namespaces.
(Beta) Using this field requires the AnyVolumeDataSource feature gate to be enabled. (Alpha) Using the namespace field of dataSourceRef requires the CrossNamespaceVolumeDataSource feature gate to be enabled.`,
																			Optional: true,

																			Attributes: map[string]schema.Attribute{
																				"api_group": schema.StringAttribute{
																					MarkdownDescription: `APIGroup is the group for the resource being referenced. If APIGroup is not specified, the specified Kind must be in the core API group. For any other third-party types, APIGroup is required.`,
																					Optional:            true,
																				},
																				"kind": schema.StringAttribute{
																					MarkdownDescription: `Kind is the type of resource being referenced`,
																					Optional:            true,
																				},
																				"name": schema.StringAttribute{
																					MarkdownDescription: `Name is the name of resource being referenced`,
																					Optional:            true,
																				},
																				"namespace": schema.StringAttribute{
																					MarkdownDescription: `Namespace is the namespace of resource being referenced Note that when a namespace is specified, a gateway.networking.k8s.io/ReferenceGrant object is required in the referent namespace to allow that namespace's owner to accept the reference. See the ReferenceGrant documentation for details. (Alpha) This field requires the CrossNamespaceVolumeDataSource feature gate to be enabled.`,
																					Optional:            true,
																				},
																			},
																		},
																		"resources": schema.SingleNestedAttribute{
																			MarkdownDescription: `resources represents the minimum resources the volume should have. If RecoverVolumeExpansionFailure feature is enabled users are allowed to specify resource requirements that are lower than previous value but must still be higher than capacity recorded in the status field of the claim. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources`,
																			Optional:            true,

																			Attributes: map[string]schema.Attribute{
																				"claims": schema.ListNestedAttribute{
																					MarkdownDescription: `Claims lists the names of resources, defined in spec.resourceClaims, that are used by this container.

This is an alpha field and requires enabling the DynamicResourceAllocation feature gate.

This field is immutable. It can only be set for containers.`,
																					Optional: true,

																					NestedObject: schema.NestedAttributeObject{
																						Attributes: map[string]schema.Attribute{
																							"name": schema.StringAttribute{
																								MarkdownDescription: `Name must match the name of one entry in pod.spec.resourceClaims of the Pod where this field is used. It makes that resource available inside a container.`,
																								Optional:            true,
																							},
																						},
																					},
																				},
																				"limits": schema.MapAttribute{
																					MarkdownDescription: `Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/`,
																					ElementType:         types.StringType,
																					Optional:            true,
																				},
																				"requests": schema.MapAttribute{
																					MarkdownDescription: `Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. Requests cannot exceed Limits. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/`,
																					ElementType:         types.StringType,
																					Optional:            true,
																				},
																			},
																		},
																		"selector": schema.SingleNestedAttribute{
																			MarkdownDescription: `selector is a label query over volumes to consider for binding.`,
																			Optional:            true,

																			Attributes: map[string]schema.Attribute{
																				"match_expressions": schema.ListNestedAttribute{
																					MarkdownDescription: `matchExpressions is a list of label selector requirements. The requirements are ANDed.`,
																					Optional:            true,

																					NestedObject: schema.NestedAttributeObject{
																						Attributes: map[string]schema.Attribute{
																							"key": schema.StringAttribute{
																								MarkdownDescription: `key is the label key that the selector applies to.`,
																								Optional:            true,
																							},
																							"operator": schema.StringAttribute{
																								MarkdownDescription: `operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.`,
																								Optional:            true,
																							},
																							"values": schema.ListAttribute{
																								MarkdownDescription: `values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.`,
																								ElementType:         types.StringType,
																								Optional:            true,
																							},
																						},
																					},
																				},
																				"match_labels": schema.MapAttribute{
																					MarkdownDescription: `matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.`,
																					ElementType:         types.StringType,
																					Optional:            true,
																				},
																			},
																		},
																		"storage_class_name": schema.StringAttribute{
																			MarkdownDescription: `storageClassName is the name of the StorageClass required by the claim. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1`,
																			Optional:            true,
																		},
																		"volume_mode": schema.StringAttribute{
																			MarkdownDescription: `volumeMode defines what type of volume is required by the claim. Value of Filesystem is implied when not included in claim spec.`,
																			Optional:            true,
																		},
																		"volume_name": schema.StringAttribute{
																			MarkdownDescription: `volumeName is the binding reference to the PersistentVolume backing this claim.`,
																			Optional:            true,
																		},
																	},
																},
															},
														},
													},
												},
												"fc": schema.SingleNestedAttribute{
													MarkdownDescription: `fc represents a Fibre Channel resource that is attached to a kubelet's host machine and then exposed to the pod.`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"fs_type": schema.StringAttribute{
															MarkdownDescription: `fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.`,
															Optional:            true,
														},
														"lun": schema.Int64Attribute{
															MarkdownDescription: `lun is Optional: FC target lun number`,
															Optional:            true,
														},
														"read_only": schema.BoolAttribute{
															MarkdownDescription: `readOnly is Optional: Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts.`,
															Optional:            true,
														},
														"target_wwns": schema.ListAttribute{
															MarkdownDescription: `targetWWNs is Optional: FC target worldwide names (WWNs)`,
															ElementType:         types.StringType,
															Optional:            true,
														},
														"wwids": schema.ListAttribute{
															MarkdownDescription: `wwids Optional: FC volume world wide identifiers (wwids) Either wwids or combination of targetWWNs and lun must be set, but not both simultaneously.`,
															ElementType:         types.StringType,
															Optional:            true,
														},
													},
												},
												"flex_volume": schema.SingleNestedAttribute{
													MarkdownDescription: `flexVolume represents a generic volume resource that is provisioned/attached using an exec based plugin.`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"driver": schema.StringAttribute{
															MarkdownDescription: `driver is the name of the driver to use for this volume.`,
															Optional:            true,
														},
														"fs_type": schema.StringAttribute{
															MarkdownDescription: `fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". The default filesystem depends on FlexVolume script.`,
															Optional:            true,
														},
														"options": schema.MapAttribute{
															MarkdownDescription: `options is Optional: this field holds extra command options if any.`,
															ElementType:         types.StringType,
															Optional:            true,
														},
														"read_only": schema.BoolAttribute{
															MarkdownDescription: `readOnly is Optional: defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts.`,
															Optional:            true,
														},
														"secret_ref": schema.SingleNestedAttribute{
															MarkdownDescription: `secretRef is Optional: secretRef is reference to the secret object containing sensitive information to pass to the plugin scripts. This may be empty if no secret object is specified. If the secret object contains more than one secret, all secrets are passed to the plugin scripts.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"name": schema.StringAttribute{
																	MarkdownDescription: `Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names`,
																	Optional:            true,
																},
															},
														},
													},
												},
												"flocker": schema.SingleNestedAttribute{
													MarkdownDescription: `flocker represents a Flocker volume attached to a kubelet's host machine. This depends on the Flocker control service being running`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"dataset_name": schema.StringAttribute{
															MarkdownDescription: `datasetName is Name of the dataset stored as metadata -> name on the dataset for Flocker should be considered as deprecated`,
															Optional:            true,
														},
														"dataset_uuid": schema.StringAttribute{
															MarkdownDescription: `datasetUUID is the UUID of the dataset. This is unique identifier of a Flocker dataset`,
															Optional:            true,
														},
													},
												},
												"gce_persistent_disk": schema.SingleNestedAttribute{
													MarkdownDescription: `gcePersistentDisk represents a GCE Disk resource that is attached to a kubelet's host machine and then exposed to the pod. More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"fs_type": schema.StringAttribute{
															MarkdownDescription: `fsType is filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk`,
															Optional:            true,
														},
														"partition": schema.Int64Attribute{
															MarkdownDescription: `partition is the partition in the volume that you want to mount. If omitted, the default is to mount by volume name. Examples: For volume /dev/sda1, you specify the partition as "1". Similarly, the volume partition for /dev/sda is "0" (or you can leave the property empty). More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk`,
															Optional:            true,
														},
														"pd_name": schema.StringAttribute{
															MarkdownDescription: `pdName is unique name of the PD resource in GCE. Used to identify the disk in GCE. More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk`,
															Optional:            true,
														},
														"read_only": schema.BoolAttribute{
															MarkdownDescription: `readOnly here will force the ReadOnly setting in VolumeMounts. Defaults to false. More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk`,
															Optional:            true,
														},
													},
												},
												"git_repo": schema.SingleNestedAttribute{
													MarkdownDescription: `gitRepo represents a git repository at a particular revision. DEPRECATED: GitRepo is deprecated. To provision a container with a git repo, mount an EmptyDir into an InitContainer that clones the repo using git, then mount the EmptyDir into the Pod's container.`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"directory": schema.StringAttribute{
															MarkdownDescription: `directory is the target directory name. Must not contain or start with '..'.  If '.' is supplied, the volume directory will be the git repository.  Otherwise, if specified, the volume will contain the git repository in the subdirectory with the given name.`,
															Optional:            true,
														},
														"repository": schema.StringAttribute{
															MarkdownDescription: `repository is the URL`,
															Optional:            true,
														},
														"revision": schema.StringAttribute{
															MarkdownDescription: `revision is the commit hash for the specified revision.`,
															Optional:            true,
														},
													},
												},
												"glusterfs": schema.SingleNestedAttribute{
													MarkdownDescription: `glusterfs represents a Glusterfs mount on the host that shares a pod's lifetime. More info: https://examples.k8s.io/volumes/glusterfs/README.md`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"endpoints": schema.StringAttribute{
															MarkdownDescription: `endpoints is the endpoint name that details Glusterfs topology. More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod`,
															Optional:            true,
														},
														"path": schema.StringAttribute{
															MarkdownDescription: `path is the Glusterfs volume path. More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod`,
															Optional:            true,
														},
														"read_only": schema.BoolAttribute{
															MarkdownDescription: `readOnly here will force the Glusterfs volume to be mounted with read-only permissions. Defaults to false. More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod`,
															Optional:            true,
														},
													},
												},
												"host_path": schema.SingleNestedAttribute{
													MarkdownDescription: `hostPath represents a pre-existing file or directory on the host machine that is directly exposed to the container. This is generally used for system agents or other privileged things that are allowed to see the host machine. Most containers will NOT need this. More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"path": schema.StringAttribute{
															MarkdownDescription: `path of the directory on the host. If the path is a symlink, it will follow the link to the real path. More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath`,
															Optional:            true,
														},
														"type": schema.StringAttribute{
															MarkdownDescription: `type for HostPath Volume Defaults to "" More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath`,
															Optional:            true,
														},
													},
												},
												"iscsi": schema.SingleNestedAttribute{
													MarkdownDescription: `iscsi represents an ISCSI Disk resource that is attached to a kubelet's host machine and then exposed to the pod. More info: https://examples.k8s.io/volumes/iscsi/README.md`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"chap_auth_discovery": schema.BoolAttribute{
															MarkdownDescription: `chapAuthDiscovery defines whether support iSCSI Discovery CHAP authentication`,
															Optional:            true,
														},
														"chap_auth_session": schema.BoolAttribute{
															MarkdownDescription: `chapAuthSession defines whether support iSCSI Session CHAP authentication`,
															Optional:            true,
														},
														"fs_type": schema.StringAttribute{
															MarkdownDescription: `fsType is the filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. More info: https://kubernetes.io/docs/concepts/storage/volumes#iscsi`,
															Optional:            true,
														},
														"initiator_name": schema.StringAttribute{
															MarkdownDescription: `initiatorName is the custom iSCSI Initiator Name. If initiatorName is specified with iscsiInterface simultaneously, new iSCSI interface <target portal>:<volume name> will be created for the connection.`,
															Optional:            true,
														},
														"iqn": schema.StringAttribute{
															MarkdownDescription: `iqn is the target iSCSI Qualified Name.`,
															Optional:            true,
														},
														"iscsi_interface": schema.StringAttribute{
															MarkdownDescription: `iscsiInterface is the interface Name that uses an iSCSI transport. Defaults to 'default' (tcp).`,
															Optional:            true,
														},
														"lun": schema.Int64Attribute{
															MarkdownDescription: `lun represents iSCSI Target Lun number.`,
															Optional:            true,
														},
														"portals": schema.ListAttribute{
															MarkdownDescription: `portals is the iSCSI Target Portal List. The portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260).`,
															ElementType:         types.StringType,
															Optional:            true,
														},
														"read_only": schema.BoolAttribute{
															MarkdownDescription: `readOnly here will force the ReadOnly setting in VolumeMounts. Defaults to false.`,
															Optional:            true,
														},
														"secret_ref": schema.SingleNestedAttribute{
															MarkdownDescription: `secretRef is the CHAP Secret for iSCSI target and initiator authentication`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"name": schema.StringAttribute{
																	MarkdownDescription: `Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names`,
																	Optional:            true,
																},
															},
														},
														"target_portal": schema.StringAttribute{
															MarkdownDescription: `targetPortal is iSCSI Target Portal. The Portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260).`,
															Optional:            true,
														},
													},
												},
												"name": schema.StringAttribute{
													MarkdownDescription: `name of the volume. Must be a DNS_LABEL and unique within the pod. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names`,
													Optional:            true,
												},
												"nfs": schema.SingleNestedAttribute{
													MarkdownDescription: `nfs represents an NFS mount on the host that shares a pod's lifetime More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"path": schema.StringAttribute{
															MarkdownDescription: `path that is exported by the NFS server. More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs`,
															Optional:            true,
														},
														"read_only": schema.BoolAttribute{
															MarkdownDescription: `readOnly here will force the NFS export to be mounted with read-only permissions. Defaults to false. More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs`,
															Optional:            true,
														},
														"server": schema.StringAttribute{
															MarkdownDescription: `server is the hostname or IP address of the NFS server. More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs`,
															Optional:            true,
														},
													},
												},
												"persistent_volume_claim": schema.SingleNestedAttribute{
													MarkdownDescription: `persistentVolumeClaimVolumeSource represents a reference to a PersistentVolumeClaim in the same namespace. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"claim_name": schema.StringAttribute{
															MarkdownDescription: `claimName is the name of a PersistentVolumeClaim in the same namespace as the pod using this volume. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims`,
															Optional:            true,
														},
														"read_only": schema.BoolAttribute{
															MarkdownDescription: `readOnly Will force the ReadOnly setting in VolumeMounts. Default false.`,
															Optional:            true,
														},
													},
												},
												"photon_persistent_disk": schema.SingleNestedAttribute{
													MarkdownDescription: `photonPersistentDisk represents a PhotonController persistent disk attached and mounted on kubelets host machine`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"fs_type": schema.StringAttribute{
															MarkdownDescription: `fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.`,
															Optional:            true,
														},
														"pd_id": schema.StringAttribute{
															MarkdownDescription: `pdID is the ID that identifies Photon Controller persistent disk`,
															Optional:            true,
														},
													},
												},
												"portworx_volume": schema.SingleNestedAttribute{
													MarkdownDescription: `portworxVolume represents a portworx volume attached and mounted on kubelets host machine`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"fs_type": schema.StringAttribute{
															MarkdownDescription: `fSType represents the filesystem type to mount Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs". Implicitly inferred to be "ext4" if unspecified.`,
															Optional:            true,
														},
														"read_only": schema.BoolAttribute{
															MarkdownDescription: `readOnly defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts.`,
															Optional:            true,
														},
														"volume_id": schema.StringAttribute{
															MarkdownDescription: `volumeID uniquely identifies a Portworx volume`,
															Optional:            true,
														},
													},
												},
												"projected": schema.SingleNestedAttribute{
													MarkdownDescription: `projected items for all in one resources secrets, configmaps, and downward API`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"default_mode": schema.Int64Attribute{
															MarkdownDescription: `defaultMode are the mode bits used to set permissions on created files by default. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.`,
															Optional:            true,
														},
														"sources": schema.ListNestedAttribute{
															MarkdownDescription: `sources is the list of volume projections`,
															Optional:            true,

															NestedObject: schema.NestedAttributeObject{
																Attributes: map[string]schema.Attribute{
																	"config_map": schema.SingleNestedAttribute{
																		MarkdownDescription: `configMap information about the configMap data to project`,
																		Optional:            true,

																		Attributes: map[string]schema.Attribute{
																			"items": schema.ListNestedAttribute{
																				MarkdownDescription: `items if unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the ConfigMap, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'.`,
																				Optional:            true,

																				NestedObject: schema.NestedAttributeObject{
																					Attributes: map[string]schema.Attribute{
																						"key": schema.StringAttribute{
																							MarkdownDescription: `key is the key to project.`,
																							Optional:            true,
																						},
																						"mode": schema.Int64Attribute{
																							MarkdownDescription: `mode is Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.`,
																							Optional:            true,
																						},
																						"path": schema.StringAttribute{
																							MarkdownDescription: `path is the relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.`,
																							Optional:            true,
																						},
																					},
																				},
																			},
																			"name": schema.StringAttribute{
																				MarkdownDescription: `Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names`,
																				Optional:            true,
																			},
																			"optional": schema.BoolAttribute{
																				MarkdownDescription: `optional specify whether the ConfigMap or its keys must be defined`,
																				Optional:            true,
																			},
																		},
																	},
																	"downward_api": schema.SingleNestedAttribute{
																		MarkdownDescription: `downwardAPI information about the downwardAPI data to project`,
																		Optional:            true,

																		Attributes: map[string]schema.Attribute{
																			"items": schema.ListNestedAttribute{
																				MarkdownDescription: `Items is a list of DownwardAPIVolume file`,
																				Optional:            true,

																				NestedObject: schema.NestedAttributeObject{
																					Attributes: map[string]schema.Attribute{
																						"field_ref": schema.SingleNestedAttribute{
																							MarkdownDescription: `Required: Selects a field of the pod: only annotations, labels, name and namespace are supported.`,
																							Optional:            true,

																							Attributes: map[string]schema.Attribute{
																								"api_version": schema.StringAttribute{
																									MarkdownDescription: `Version of the schema the FieldPath is written in terms of, defaults to "v1".`,
																									Optional:            true,
																								},
																								"field_path": schema.StringAttribute{
																									MarkdownDescription: `Path of the field to select in the specified API version.`,
																									Optional:            true,
																								},
																							},
																						},
																						"mode": schema.Int64Attribute{
																							MarkdownDescription: `Optional: mode bits used to set permissions on this file, must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.`,
																							Optional:            true,
																						},
																						"path": schema.StringAttribute{
																							MarkdownDescription: `Required: Path is  the relative path name of the file to be created. Must not be absolute or contain the '..' path. Must be utf-8 encoded. The first item of the relative path must not start with '..'`,
																							Optional:            true,
																						},
																						"resource_field_ref": schema.SingleNestedAttribute{
																							MarkdownDescription: `Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.`,
																							Optional:            true,

																							Attributes: map[string]schema.Attribute{
																								"container_name": schema.StringAttribute{
																									MarkdownDescription: `Container name: required for volumes, optional for env vars`,
																									Optional:            true,
																								},
																								"divisor": schema.StringAttribute{
																									MarkdownDescription: `Specifies the output format of the exposed resources, defaults to "1"`,
																									Optional:            true,
																								},
																								"resource": schema.StringAttribute{
																									MarkdownDescription: `Required: resource to select`,
																									Optional:            true,
																								},
																							},
																						},
																					},
																				},
																			},
																		},
																	},
																	"secret": schema.SingleNestedAttribute{
																		MarkdownDescription: `secret information about the secret data to project`,
																		Optional:            true,

																		Attributes: map[string]schema.Attribute{
																			"items": schema.ListNestedAttribute{
																				MarkdownDescription: `items if unspecified, each key-value pair in the Data field of the referenced Secret will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the Secret, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'.`,
																				Optional:            true,

																				NestedObject: schema.NestedAttributeObject{
																					Attributes: map[string]schema.Attribute{
																						"key": schema.StringAttribute{
																							MarkdownDescription: `key is the key to project.`,
																							Optional:            true,
																						},
																						"mode": schema.Int64Attribute{
																							MarkdownDescription: `mode is Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.`,
																							Optional:            true,
																						},
																						"path": schema.StringAttribute{
																							MarkdownDescription: `path is the relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.`,
																							Optional:            true,
																						},
																					},
																				},
																			},
																			"name": schema.StringAttribute{
																				MarkdownDescription: `Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names`,
																				Optional:            true,
																			},
																			"optional": schema.BoolAttribute{
																				MarkdownDescription: `optional field specify whether the Secret or its key must be defined`,
																				Optional:            true,
																			},
																		},
																	},
																	"service_account_token": schema.SingleNestedAttribute{
																		MarkdownDescription: `serviceAccountToken is information about the serviceAccountToken data to project`,
																		Optional:            true,

																		Attributes: map[string]schema.Attribute{
																			"audience": schema.StringAttribute{
																				MarkdownDescription: `audience is the intended audience of the token. A recipient of a token must identify itself with an identifier specified in the audience of the token, and otherwise should reject the token. The audience defaults to the identifier of the apiserver.`,
																				Optional:            true,
																			},
																			"expiration_seconds": schema.Int64Attribute{
																				MarkdownDescription: `expirationSeconds is the requested duration of validity of the service account token. As the token approaches expiration, the kubelet volume plugin will proactively rotate the service account token. The kubelet will start trying to rotate the token if the token is older than 80 percent of its time to live or if the token is older than 24 hours.Defaults to 1 hour and must be at least 10 minutes.`,
																				Optional:            true,
																			},
																			"path": schema.StringAttribute{
																				MarkdownDescription: `path is the path relative to the mount point of the file to project the token into.`,
																				Optional:            true,
																			},
																		},
																	},
																},
															},
														},
													},
												},
												"quobyte": schema.SingleNestedAttribute{
													MarkdownDescription: `quobyte represents a Quobyte mount on the host that shares a pod's lifetime`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"group": schema.StringAttribute{
															MarkdownDescription: `group to map volume access to Default is no group`,
															Optional:            true,
														},
														"read_only": schema.BoolAttribute{
															MarkdownDescription: `readOnly here will force the Quobyte volume to be mounted with read-only permissions. Defaults to false.`,
															Optional:            true,
														},
														"registry": schema.StringAttribute{
															MarkdownDescription: `registry represents a single or multiple Quobyte Registry services specified as a string as host:port pair (multiple entries are separated with commas) which acts as the central registry for volumes`,
															Optional:            true,
														},
														"tenant": schema.StringAttribute{
															MarkdownDescription: `tenant owning the given Quobyte volume in the Backend Used with dynamically provisioned Quobyte volumes, value is set by the plugin`,
															Optional:            true,
														},
														"user": schema.StringAttribute{
															MarkdownDescription: `user to map volume access to Defaults to serivceaccount user`,
															Optional:            true,
														},
														"volume": schema.StringAttribute{
															MarkdownDescription: `volume is a string that references an already created Quobyte volume by name.`,
															Optional:            true,
														},
													},
												},
												"rbd": schema.SingleNestedAttribute{
													MarkdownDescription: `rbd represents a Rados Block Device mount on the host that shares a pod's lifetime. More info: https://examples.k8s.io/volumes/rbd/README.md`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"fs_type": schema.StringAttribute{
															MarkdownDescription: `fsType is the filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. More info: https://kubernetes.io/docs/concepts/storage/volumes#rbd`,
															Optional:            true,
														},
														"image": schema.StringAttribute{
															MarkdownDescription: `image is the rados image name. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it`,
															Optional:            true,
														},
														"keyring": schema.StringAttribute{
															MarkdownDescription: `keyring is the path to key ring for RBDUser. Default is /etc/ceph/keyring. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it`,
															Optional:            true,
														},
														"monitors": schema.ListAttribute{
															MarkdownDescription: `monitors is a collection of Ceph monitors. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it`,
															ElementType:         types.StringType,
															Optional:            true,
														},
														"pool": schema.StringAttribute{
															MarkdownDescription: `pool is the rados pool name. Default is rbd. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it`,
															Optional:            true,
														},
														"read_only": schema.BoolAttribute{
															MarkdownDescription: `readOnly here will force the ReadOnly setting in VolumeMounts. Defaults to false. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it`,
															Optional:            true,
														},
														"secret_ref": schema.SingleNestedAttribute{
															MarkdownDescription: `secretRef is name of the authentication secret for RBDUser. If provided overrides keyring. Default is nil. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"name": schema.StringAttribute{
																	MarkdownDescription: `Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names`,
																	Optional:            true,
																},
															},
														},
														"user": schema.StringAttribute{
															MarkdownDescription: `user is the rados user name. Default is admin. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it`,
															Optional:            true,
														},
													},
												},
												"scale_io": schema.SingleNestedAttribute{
													MarkdownDescription: `scaleIO represents a ScaleIO persistent volume attached and mounted on Kubernetes nodes.`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"fs_type": schema.StringAttribute{
															MarkdownDescription: `fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Default is "xfs".`,
															Optional:            true,
														},
														"gateway": schema.StringAttribute{
															MarkdownDescription: `gateway is the host address of the ScaleIO API Gateway.`,
															Optional:            true,
														},
														"protection_domain": schema.StringAttribute{
															MarkdownDescription: `protectionDomain is the name of the ScaleIO Protection Domain for the configured storage.`,
															Optional:            true,
														},
														"read_only": schema.BoolAttribute{
															MarkdownDescription: `readOnly Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts.`,
															Optional:            true,
														},
														"secret_ref": schema.SingleNestedAttribute{
															MarkdownDescription: `secretRef references to the secret for ScaleIO user and other sensitive information. If this is not provided, Login operation will fail.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"name": schema.StringAttribute{
																	MarkdownDescription: `Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names`,
																	Optional:            true,
																},
															},
														},
														"ssl_enabled": schema.BoolAttribute{
															MarkdownDescription: `sslEnabled Flag enable/disable SSL communication with Gateway, default false`,
															Optional:            true,
														},
														"storage_mode": schema.StringAttribute{
															MarkdownDescription: `storageMode indicates whether the storage for a volume should be ThickProvisioned or ThinProvisioned. Default is ThinProvisioned.`,
															Optional:            true,
														},
														"storage_pool": schema.StringAttribute{
															MarkdownDescription: `storagePool is the ScaleIO Storage Pool associated with the protection domain.`,
															Optional:            true,
														},
														"system": schema.StringAttribute{
															MarkdownDescription: `system is the name of the storage system as configured in ScaleIO.`,
															Optional:            true,
														},
														"volume_name": schema.StringAttribute{
															MarkdownDescription: `volumeName is the name of a volume already created in the ScaleIO system that is associated with this volume source.`,
															Optional:            true,
														},
													},
												},
												"secret": schema.SingleNestedAttribute{
													MarkdownDescription: `secret represents a secret that should populate this volume. More info: https://kubernetes.io/docs/concepts/storage/volumes#secret`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"default_mode": schema.Int64Attribute{
															MarkdownDescription: `defaultMode is Optional: mode bits used to set permissions on created files by default. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. Defaults to 0644. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.`,
															Optional:            true,
														},
														"items": schema.ListNestedAttribute{
															MarkdownDescription: `items If unspecified, each key-value pair in the Data field of the referenced Secret will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the Secret, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'.`,
															Optional:            true,

															NestedObject: schema.NestedAttributeObject{
																Attributes: map[string]schema.Attribute{
																	"key": schema.StringAttribute{
																		MarkdownDescription: `key is the key to project.`,
																		Optional:            true,
																	},
																	"mode": schema.Int64Attribute{
																		MarkdownDescription: `mode is Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.`,
																		Optional:            true,
																	},
																	"path": schema.StringAttribute{
																		MarkdownDescription: `path is the relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.`,
																		Optional:            true,
																	},
																},
															},
														},
														"optional": schema.BoolAttribute{
															MarkdownDescription: `optional field specify whether the Secret or its keys must be defined`,
															Optional:            true,
														},
														"secret_name": schema.StringAttribute{
															MarkdownDescription: `secretName is the name of the secret in the pod's namespace to use. More info: https://kubernetes.io/docs/concepts/storage/volumes#secret`,
															Optional:            true,
														},
													},
												},
												"storageos": schema.SingleNestedAttribute{
													MarkdownDescription: `storageOS represents a StorageOS volume attached and mounted on Kubernetes nodes.`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"fs_type": schema.StringAttribute{
															MarkdownDescription: `fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.`,
															Optional:            true,
														},
														"read_only": schema.BoolAttribute{
															MarkdownDescription: `readOnly defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts.`,
															Optional:            true,
														},
														"secret_ref": schema.SingleNestedAttribute{
															MarkdownDescription: `secretRef specifies the secret to use for obtaining the StorageOS API credentials.  If not specified, default values will be attempted.`,
															Optional:            true,

															Attributes: map[string]schema.Attribute{
																"name": schema.StringAttribute{
																	MarkdownDescription: `Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names`,
																	Optional:            true,
																},
															},
														},
														"volume_name": schema.StringAttribute{
															MarkdownDescription: `volumeName is the human-readable name of the StorageOS volume.  Volume names are only unique within a namespace.`,
															Optional:            true,
														},
														"volume_namespace": schema.StringAttribute{
															MarkdownDescription: `volumeNamespace specifies the scope of the volume within StorageOS.  If no namespace is specified then the Pod's namespace will be used.  This allows the Kubernetes name scoping to be mirrored within StorageOS for tighter integration. Set VolumeName to any name to override the default behaviour. Set to "default" if you are not using namespaces within StorageOS. Namespaces that do not pre-exist within StorageOS will be created.`,
															Optional:            true,
														},
													},
												},
												"vsphere_volume": schema.SingleNestedAttribute{
													MarkdownDescription: `vsphereVolume represents a vSphere volume attached and mounted on kubelets host machine`,
													Optional:            true,

													Attributes: map[string]schema.Attribute{
														"fs_type": schema.StringAttribute{
															MarkdownDescription: `fsType is filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.`,
															Optional:            true,
														},
														"storage_policy_id": schema.StringAttribute{
															MarkdownDescription: `storagePolicyID is the storage Policy Based Management (SPBM) profile ID associated with the StoragePolicyName.`,
															Optional:            true,
														},
														"storage_policy_name": schema.StringAttribute{
															MarkdownDescription: `storagePolicyName is the storage Policy Based Management (SPBM) profile name.`,
															Optional:            true,
														},
														"volume_path": schema.StringAttribute{
															MarkdownDescription: `volumePath is the path that identifies vSphere volume vmdk`,
															Optional:            true,
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
					"update_strategy": schema.SingleNestedAttribute{
						MarkdownDescription: `An update strategy to replace existing DaemonSet pods with new pods.`,
						Optional:            true,

						Attributes: map[string]schema.Attribute{
							"rolling_update": schema.SingleNestedAttribute{
								MarkdownDescription: `Rolling update config params. Present only if type = "RollingUpdate".`,
								Optional:            true,

								Attributes: map[string]schema.Attribute{
									"max_surge": schema.StringAttribute{
										MarkdownDescription: `The maximum number of nodes with an existing available DaemonSet pod that can have an updated DaemonSet pod during during an update. Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%). This can not be 0 if MaxUnavailable is 0. Absolute number is calculated from percentage by rounding up to a minimum of 1. Default value is 0. Example: when this is set to 30%, at most 30% of the total number of nodes that should be running the daemon pod (i.e. status.desiredNumberScheduled) can have their a new pod created before the old pod is marked as deleted. The update starts by launching new pods on 30% of nodes. Once an updated pod is available (Ready for at least minReadySeconds) the old DaemonSet pod on that node is marked deleted. If the old pod becomes unavailable for any reason (Ready transitions to false, is evicted, or is drained) an updated pod is immediatedly created on that node without considering surge limits. Allowing surge implies the possibility that the resources consumed by the daemonset on any given node can double if the readiness check fails, and so resource intensive daemonsets should take into account that they may cause evictions during disruption.`,
										Optional:            true,
									},
									"max_unavailable": schema.StringAttribute{
										MarkdownDescription: `The maximum number of DaemonSet pods that can be unavailable during the update. Value can be an absolute number (ex: 5) or a percentage of total number of DaemonSet pods at the start of the update (ex: 10%). Absolute number is calculated from percentage by rounding up. This cannot be 0 if MaxSurge is 0 Default value is 1. Example: when this is set to 30%, at most 30% of the total number of nodes that should be running the daemon pod (i.e. status.desiredNumberScheduled) can have their pods stopped for an update at any given time. The update starts by stopping at most 30% of those DaemonSet pods and then brings up new DaemonSet pods in their place. Once the new pods are available, it then proceeds onto other DaemonSet pods, thus ensuring that at least 70% of original number of DaemonSet pods are available at all times during the update.`,
										Optional:            true,
									},
								},
							},
							"type": schema.StringAttribute{
								MarkdownDescription: `Type of daemon set update. Can be "RollingUpdate" or "OnDelete". Default is RollingUpdate.`,
								Optional:            true,
							},
						},
					},
				},
			},
		},
	}
}
