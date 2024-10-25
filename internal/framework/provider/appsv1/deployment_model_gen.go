package appsv1

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type DeploymentModel struct {
	Timeouts timeouts.Value `tfsdk:"timeouts"`

	ID       types.String `tfsdk:"id" manifest:""`
	Metadata struct {
		Annotations     map[string]types.String `tfsdk:"annotations" manifest:"annotations"`
		GenerateName    types.String            `tfsdk:"generate_name" manifest:"generateName"`
		Generation      types.Int64             `tfsdk:"generation" manifest:"generation"`
		Labels          map[string]types.String `tfsdk:"labels" manifest:"labels"`
		Name            types.String            `tfsdk:"name" manifest:"name"`
		Namespace       types.String            `tfsdk:"namespace" manifest:"namespace"`
		ResourceVersion types.String            `tfsdk:"resource_version" manifest:"resourceVersion"`
		UID             types.String            `tfsdk:"uid" manifest:"uid"`
	} `tfsdk:"metadata" manifest:"metadata"`
	Spec struct {
		MinReadySeconds         types.Int64 `tfsdk:"min_ready_seconds" manifest:"minReadySeconds"`
		Paused                  types.Bool  `tfsdk:"paused" manifest:"paused"`
		ProgressDeadlineSeconds types.Int64 `tfsdk:"progress_deadline_seconds" manifest:"progressDeadlineSeconds"`
		Replicas                types.Int64 `tfsdk:"replicas" manifest:"replicas"`
		RevisionHistoryLimit    types.Int64 `tfsdk:"revision_history_limit" manifest:"revisionHistoryLimit"`
		Selector                struct {
			MatchExpressions []struct {
				Key      types.String   `tfsdk:"key" manifest:"key"`
				Operator types.String   `tfsdk:"operator" manifest:"operator"`
				Values   []types.String `tfsdk:"values" manifest:"values"`
			} `tfsdk:"match_expressions" manifest:"matchExpressions"`
			MatchLabels map[string]types.String `tfsdk:"match_labels" manifest:"matchLabels"`
		} `tfsdk:"selector" manifest:"selector"`
		Strategy struct {
			RollingUpdate struct {
				MaxSurge       types.String `tfsdk:"max_surge" manifest:"maxSurge"`
				MaxUnavailable types.String `tfsdk:"max_unavailable" manifest:"maxUnavailable"`
			} `tfsdk:"rolling_update" manifest:"rollingUpdate"`
			Type types.String `tfsdk:"type" manifest:"type"`
		} `tfsdk:"strategy" manifest:"strategy"`
		Template struct {
			Metadata struct {
				Annotations                map[string]types.String `tfsdk:"annotations" manifest:"annotations"`
				CreationTimestamp          types.String            `tfsdk:"creation_timestamp" manifest:"creationTimestamp"`
				DeletionGracePeriodSeconds types.Int64             `tfsdk:"deletion_grace_period_seconds" manifest:"deletionGracePeriodSeconds"`
				DeletionTimestamp          types.String            `tfsdk:"deletion_timestamp" manifest:"deletionTimestamp"`
				Finalizers                 []types.String          `tfsdk:"finalizers" manifest:"finalizers"`
				GenerateName               types.String            `tfsdk:"generate_name" manifest:"generateName"`
				Generation                 types.Int64             `tfsdk:"generation" manifest:"generation"`
				Labels                     map[string]types.String `tfsdk:"labels" manifest:"labels"`
				ManagedFields              []struct {
					APIVersion  types.String `tfsdk:"api_version" manifest:"apiVersion"`
					FieldsType  types.String `tfsdk:"fields_type" manifest:"fieldsType"`
					Manager     types.String `tfsdk:"manager" manifest:"manager"`
					Operation   types.String `tfsdk:"operation" manifest:"operation"`
					Subresource types.String `tfsdk:"subresource" manifest:"subresource"`
					Time        types.String `tfsdk:"time" manifest:"time"`
				} `tfsdk:"managed_fields" manifest:"managedFields"`
				Name            types.String `tfsdk:"name" manifest:"name"`
				Namespace       types.String `tfsdk:"namespace" manifest:"namespace"`
				OwnerReferences []struct {
					APIVersion         types.String `tfsdk:"api_version" manifest:"apiVersion"`
					BlockOwnerDeletion types.Bool   `tfsdk:"block_owner_deletion" manifest:"blockOwnerDeletion"`
					Controller         types.Bool   `tfsdk:"controller" manifest:"controller"`
					Kind               types.String `tfsdk:"kind" manifest:"kind"`
					Name               types.String `tfsdk:"name" manifest:"name"`
					UID                types.String `tfsdk:"uid" manifest:"uid"`
				} `tfsdk:"owner_references" manifest:"ownerReferences"`
				ResourceVersion types.String `tfsdk:"resource_version" manifest:"resourceVersion"`
				SelfLink        types.String `tfsdk:"self_link" manifest:"selfLink"`
				UID             types.String `tfsdk:"uid" manifest:"uid"`
			} `tfsdk:"metadata" manifest:"metadata"`
			Spec struct {
				ActiveDeadlineSeconds types.Int64 `tfsdk:"active_deadline_seconds" manifest:"activeDeadlineSeconds"`
				Affinity              struct {
					NodeAffinity struct {
						PreferredDuringSchedulingIgnoredDuringExecution []struct {
							Preference struct {
								MatchExpressions []struct {
									Key      types.String   `tfsdk:"key" manifest:"key"`
									Operator types.String   `tfsdk:"operator" manifest:"operator"`
									Values   []types.String `tfsdk:"values" manifest:"values"`
								} `tfsdk:"match_expressions" manifest:"matchExpressions"`
								MatchFields []struct {
									Key      types.String   `tfsdk:"key" manifest:"key"`
									Operator types.String   `tfsdk:"operator" manifest:"operator"`
									Values   []types.String `tfsdk:"values" manifest:"values"`
								} `tfsdk:"match_fields" manifest:"matchFields"`
							} `tfsdk:"preference" manifest:"preference"`
							Weight types.Int64 `tfsdk:"weight" manifest:"weight"`
						} `tfsdk:"preferred_during_scheduling_ignored_during_execution" manifest:"preferredDuringSchedulingIgnoredDuringExecution"`
						RequiredDuringSchedulingIgnoredDuringExecution struct {
							NodeSelectorTerms []struct {
								MatchExpressions []struct {
									Key      types.String   `tfsdk:"key" manifest:"key"`
									Operator types.String   `tfsdk:"operator" manifest:"operator"`
									Values   []types.String `tfsdk:"values" manifest:"values"`
								} `tfsdk:"match_expressions" manifest:"matchExpressions"`
								MatchFields []struct {
									Key      types.String   `tfsdk:"key" manifest:"key"`
									Operator types.String   `tfsdk:"operator" manifest:"operator"`
									Values   []types.String `tfsdk:"values" manifest:"values"`
								} `tfsdk:"match_fields" manifest:"matchFields"`
							} `tfsdk:"node_selector_terms" manifest:"nodeSelectorTerms"`
						} `tfsdk:"required_during_scheduling_ignored_during_execution" manifest:"requiredDuringSchedulingIgnoredDuringExecution"`
					} `tfsdk:"node_affinity" manifest:"nodeAffinity"`
					PodAffinity struct {
						PreferredDuringSchedulingIgnoredDuringExecution []struct {
							PodAffinityTerm struct {
								LabelSelector struct {
									MatchExpressions []struct {
										Key      types.String   `tfsdk:"key" manifest:"key"`
										Operator types.String   `tfsdk:"operator" manifest:"operator"`
										Values   []types.String `tfsdk:"values" manifest:"values"`
									} `tfsdk:"match_expressions" manifest:"matchExpressions"`
									MatchLabels map[string]types.String `tfsdk:"match_labels" manifest:"matchLabels"`
								} `tfsdk:"label_selector" manifest:"labelSelector"`
								NamespaceSelector struct {
									MatchExpressions []struct {
										Key      types.String   `tfsdk:"key" manifest:"key"`
										Operator types.String   `tfsdk:"operator" manifest:"operator"`
										Values   []types.String `tfsdk:"values" manifest:"values"`
									} `tfsdk:"match_expressions" manifest:"matchExpressions"`
									MatchLabels map[string]types.String `tfsdk:"match_labels" manifest:"matchLabels"`
								} `tfsdk:"namespace_selector" manifest:"namespaceSelector"`
								Namespaces  []types.String `tfsdk:"namespaces" manifest:"namespaces"`
								TopologyKey types.String   `tfsdk:"topology_key" manifest:"topologyKey"`
							} `tfsdk:"pod_affinity_term" manifest:"podAffinityTerm"`
							Weight types.Int64 `tfsdk:"weight" manifest:"weight"`
						} `tfsdk:"preferred_during_scheduling_ignored_during_execution" manifest:"preferredDuringSchedulingIgnoredDuringExecution"`
						RequiredDuringSchedulingIgnoredDuringExecution []struct {
							LabelSelector struct {
								MatchExpressions []struct {
									Key      types.String   `tfsdk:"key" manifest:"key"`
									Operator types.String   `tfsdk:"operator" manifest:"operator"`
									Values   []types.String `tfsdk:"values" manifest:"values"`
								} `tfsdk:"match_expressions" manifest:"matchExpressions"`
								MatchLabels map[string]types.String `tfsdk:"match_labels" manifest:"matchLabels"`
							} `tfsdk:"label_selector" manifest:"labelSelector"`
							NamespaceSelector struct {
								MatchExpressions []struct {
									Key      types.String   `tfsdk:"key" manifest:"key"`
									Operator types.String   `tfsdk:"operator" manifest:"operator"`
									Values   []types.String `tfsdk:"values" manifest:"values"`
								} `tfsdk:"match_expressions" manifest:"matchExpressions"`
								MatchLabels map[string]types.String `tfsdk:"match_labels" manifest:"matchLabels"`
							} `tfsdk:"namespace_selector" manifest:"namespaceSelector"`
							Namespaces  []types.String `tfsdk:"namespaces" manifest:"namespaces"`
							TopologyKey types.String   `tfsdk:"topology_key" manifest:"topologyKey"`
						} `tfsdk:"required_during_scheduling_ignored_during_execution" manifest:"requiredDuringSchedulingIgnoredDuringExecution"`
					} `tfsdk:"pod_affinity" manifest:"podAffinity"`
					PodAntiAffinity struct {
						PreferredDuringSchedulingIgnoredDuringExecution []struct {
							PodAffinityTerm struct {
								LabelSelector struct {
									MatchExpressions []struct {
										Key      types.String   `tfsdk:"key" manifest:"key"`
										Operator types.String   `tfsdk:"operator" manifest:"operator"`
										Values   []types.String `tfsdk:"values" manifest:"values"`
									} `tfsdk:"match_expressions" manifest:"matchExpressions"`
									MatchLabels map[string]types.String `tfsdk:"match_labels" manifest:"matchLabels"`
								} `tfsdk:"label_selector" manifest:"labelSelector"`
								NamespaceSelector struct {
									MatchExpressions []struct {
										Key      types.String   `tfsdk:"key" manifest:"key"`
										Operator types.String   `tfsdk:"operator" manifest:"operator"`
										Values   []types.String `tfsdk:"values" manifest:"values"`
									} `tfsdk:"match_expressions" manifest:"matchExpressions"`
									MatchLabels map[string]types.String `tfsdk:"match_labels" manifest:"matchLabels"`
								} `tfsdk:"namespace_selector" manifest:"namespaceSelector"`
								Namespaces  []types.String `tfsdk:"namespaces" manifest:"namespaces"`
								TopologyKey types.String   `tfsdk:"topology_key" manifest:"topologyKey"`
							} `tfsdk:"pod_affinity_term" manifest:"podAffinityTerm"`
							Weight types.Int64 `tfsdk:"weight" manifest:"weight"`
						} `tfsdk:"preferred_during_scheduling_ignored_during_execution" manifest:"preferredDuringSchedulingIgnoredDuringExecution"`
						RequiredDuringSchedulingIgnoredDuringExecution []struct {
							LabelSelector struct {
								MatchExpressions []struct {
									Key      types.String   `tfsdk:"key" manifest:"key"`
									Operator types.String   `tfsdk:"operator" manifest:"operator"`
									Values   []types.String `tfsdk:"values" manifest:"values"`
								} `tfsdk:"match_expressions" manifest:"matchExpressions"`
								MatchLabels map[string]types.String `tfsdk:"match_labels" manifest:"matchLabels"`
							} `tfsdk:"label_selector" manifest:"labelSelector"`
							NamespaceSelector struct {
								MatchExpressions []struct {
									Key      types.String   `tfsdk:"key" manifest:"key"`
									Operator types.String   `tfsdk:"operator" manifest:"operator"`
									Values   []types.String `tfsdk:"values" manifest:"values"`
								} `tfsdk:"match_expressions" manifest:"matchExpressions"`
								MatchLabels map[string]types.String `tfsdk:"match_labels" manifest:"matchLabels"`
							} `tfsdk:"namespace_selector" manifest:"namespaceSelector"`
							Namespaces  []types.String `tfsdk:"namespaces" manifest:"namespaces"`
							TopologyKey types.String   `tfsdk:"topology_key" manifest:"topologyKey"`
						} `tfsdk:"required_during_scheduling_ignored_during_execution" manifest:"requiredDuringSchedulingIgnoredDuringExecution"`
					} `tfsdk:"pod_anti_affinity" manifest:"podAntiAffinity"`
				} `tfsdk:"affinity" manifest:"affinity"`
				AutomountServiceAccountToken types.Bool `tfsdk:"automount_service_account_token" manifest:"automountServiceAccountToken"`
				Containers                   []struct {
					Args    []types.String `tfsdk:"args" manifest:"args"`
					Command []types.String `tfsdk:"command" manifest:"command"`
					Env     []struct {
						Name      types.String `tfsdk:"name" manifest:"name"`
						Value     types.String `tfsdk:"value" manifest:"value"`
						ValueFrom struct {
							ConfigMapKeyRef struct {
								Key      types.String `tfsdk:"key" manifest:"key"`
								Name     types.String `tfsdk:"name" manifest:"name"`
								Optional types.Bool   `tfsdk:"optional" manifest:"optional"`
							} `tfsdk:"config_map_key_ref" manifest:"configMapKeyRef"`
							FieldRef struct {
								APIVersion types.String `tfsdk:"api_version" manifest:"apiVersion"`
								FieldPath  types.String `tfsdk:"field_path" manifest:"fieldPath"`
							} `tfsdk:"field_ref" manifest:"fieldRef"`
							ResourceFieldRef struct {
								ContainerName types.String `tfsdk:"container_name" manifest:"containerName"`
								Divisor       types.String `tfsdk:"divisor" manifest:"divisor"`
								Resource      types.String `tfsdk:"resource" manifest:"resource"`
							} `tfsdk:"resource_field_ref" manifest:"resourceFieldRef"`
							SecretKeyRef struct {
								Key      types.String `tfsdk:"key" manifest:"key"`
								Name     types.String `tfsdk:"name" manifest:"name"`
								Optional types.Bool   `tfsdk:"optional" manifest:"optional"`
							} `tfsdk:"secret_key_ref" manifest:"secretKeyRef"`
						} `tfsdk:"value_from" manifest:"valueFrom"`
					} `tfsdk:"env" manifest:"env"`
					EnvFrom []struct {
						ConfigMapRef struct {
							Name     types.String `tfsdk:"name" manifest:"name"`
							Optional types.Bool   `tfsdk:"optional" manifest:"optional"`
						} `tfsdk:"config_map_ref" manifest:"configMapRef"`
						Prefix    types.String `tfsdk:"prefix" manifest:"prefix"`
						SecretRef struct {
							Name     types.String `tfsdk:"name" manifest:"name"`
							Optional types.Bool   `tfsdk:"optional" manifest:"optional"`
						} `tfsdk:"secret_ref" manifest:"secretRef"`
					} `tfsdk:"env_from" manifest:"envFrom"`
					Image           types.String `tfsdk:"image" manifest:"image"`
					ImagePullPolicy types.String `tfsdk:"image_pull_policy" manifest:"imagePullPolicy"`
					Lifecycle       struct {
						PostStart struct {
							Exec struct {
								Command []types.String `tfsdk:"command" manifest:"command"`
							} `tfsdk:"exec" manifest:"exec"`
							HttpGet struct {
								Host        types.String `tfsdk:"host" manifest:"host"`
								HttpHeaders []struct {
									Name  types.String `tfsdk:"name" manifest:"name"`
									Value types.String `tfsdk:"value" manifest:"value"`
								} `tfsdk:"http_headers" manifest:"httpHeaders"`
								Path   types.String `tfsdk:"path" manifest:"path"`
								Port   types.String `tfsdk:"port" manifest:"port"`
								Scheme types.String `tfsdk:"scheme" manifest:"scheme"`
							} `tfsdk:"http_get" manifest:"httpGet"`
							TcpSocket struct {
								Host types.String `tfsdk:"host" manifest:"host"`
								Port types.String `tfsdk:"port" manifest:"port"`
							} `tfsdk:"tcp_socket" manifest:"tcpSocket"`
						} `tfsdk:"post_start" manifest:"postStart"`
						PreStop struct {
							Exec struct {
								Command []types.String `tfsdk:"command" manifest:"command"`
							} `tfsdk:"exec" manifest:"exec"`
							HttpGet struct {
								Host        types.String `tfsdk:"host" manifest:"host"`
								HttpHeaders []struct {
									Name  types.String `tfsdk:"name" manifest:"name"`
									Value types.String `tfsdk:"value" manifest:"value"`
								} `tfsdk:"http_headers" manifest:"httpHeaders"`
								Path   types.String `tfsdk:"path" manifest:"path"`
								Port   types.String `tfsdk:"port" manifest:"port"`
								Scheme types.String `tfsdk:"scheme" manifest:"scheme"`
							} `tfsdk:"http_get" manifest:"httpGet"`
							TcpSocket struct {
								Host types.String `tfsdk:"host" manifest:"host"`
								Port types.String `tfsdk:"port" manifest:"port"`
							} `tfsdk:"tcp_socket" manifest:"tcpSocket"`
						} `tfsdk:"pre_stop" manifest:"preStop"`
					} `tfsdk:"lifecycle" manifest:"lifecycle"`
					LivenessProbe struct {
						Exec struct {
							Command []types.String `tfsdk:"command" manifest:"command"`
						} `tfsdk:"exec" manifest:"exec"`
						FailureThreshold types.Int64 `tfsdk:"failure_threshold" manifest:"failureThreshold"`
						Grpc             struct {
							Port    types.Int64  `tfsdk:"port" manifest:"port"`
							Service types.String `tfsdk:"service" manifest:"service"`
						} `tfsdk:"grpc" manifest:"grpc"`
						HttpGet struct {
							Host        types.String `tfsdk:"host" manifest:"host"`
							HttpHeaders []struct {
								Name  types.String `tfsdk:"name" manifest:"name"`
								Value types.String `tfsdk:"value" manifest:"value"`
							} `tfsdk:"http_headers" manifest:"httpHeaders"`
							Path   types.String `tfsdk:"path" manifest:"path"`
							Port   types.String `tfsdk:"port" manifest:"port"`
							Scheme types.String `tfsdk:"scheme" manifest:"scheme"`
						} `tfsdk:"http_get" manifest:"httpGet"`
						InitialDelaySeconds types.Int64 `tfsdk:"initial_delay_seconds" manifest:"initialDelaySeconds"`
						PeriodSeconds       types.Int64 `tfsdk:"period_seconds" manifest:"periodSeconds"`
						SuccessThreshold    types.Int64 `tfsdk:"success_threshold" manifest:"successThreshold"`
						TcpSocket           struct {
							Host types.String `tfsdk:"host" manifest:"host"`
							Port types.String `tfsdk:"port" manifest:"port"`
						} `tfsdk:"tcp_socket" manifest:"tcpSocket"`
						TerminationGracePeriodSeconds types.Int64 `tfsdk:"termination_grace_period_seconds" manifest:"terminationGracePeriodSeconds"`
						TimeoutSeconds                types.Int64 `tfsdk:"timeout_seconds" manifest:"timeoutSeconds"`
					} `tfsdk:"liveness_probe" manifest:"livenessProbe"`
					Name  types.String `tfsdk:"name" manifest:"name"`
					Ports []struct {
						ContainerPort types.Int64  `tfsdk:"container_port" manifest:"containerPort"`
						HostIp        types.String `tfsdk:"host_ip" manifest:"hostIp"`
						HostPort      types.Int64  `tfsdk:"host_port" manifest:"hostPort"`
						Name          types.String `tfsdk:"name" manifest:"name"`
						Protocol      types.String `tfsdk:"protocol" manifest:"protocol"`
					} `tfsdk:"ports" manifest:"ports"`
					ReadinessProbe struct {
						Exec struct {
							Command []types.String `tfsdk:"command" manifest:"command"`
						} `tfsdk:"exec" manifest:"exec"`
						FailureThreshold types.Int64 `tfsdk:"failure_threshold" manifest:"failureThreshold"`
						Grpc             struct {
							Port    types.Int64  `tfsdk:"port" manifest:"port"`
							Service types.String `tfsdk:"service" manifest:"service"`
						} `tfsdk:"grpc" manifest:"grpc"`
						HttpGet struct {
							Host        types.String `tfsdk:"host" manifest:"host"`
							HttpHeaders []struct {
								Name  types.String `tfsdk:"name" manifest:"name"`
								Value types.String `tfsdk:"value" manifest:"value"`
							} `tfsdk:"http_headers" manifest:"httpHeaders"`
							Path   types.String `tfsdk:"path" manifest:"path"`
							Port   types.String `tfsdk:"port" manifest:"port"`
							Scheme types.String `tfsdk:"scheme" manifest:"scheme"`
						} `tfsdk:"http_get" manifest:"httpGet"`
						InitialDelaySeconds types.Int64 `tfsdk:"initial_delay_seconds" manifest:"initialDelaySeconds"`
						PeriodSeconds       types.Int64 `tfsdk:"period_seconds" manifest:"periodSeconds"`
						SuccessThreshold    types.Int64 `tfsdk:"success_threshold" manifest:"successThreshold"`
						TcpSocket           struct {
							Host types.String `tfsdk:"host" manifest:"host"`
							Port types.String `tfsdk:"port" manifest:"port"`
						} `tfsdk:"tcp_socket" manifest:"tcpSocket"`
						TerminationGracePeriodSeconds types.Int64 `tfsdk:"termination_grace_period_seconds" manifest:"terminationGracePeriodSeconds"`
						TimeoutSeconds                types.Int64 `tfsdk:"timeout_seconds" manifest:"timeoutSeconds"`
					} `tfsdk:"readiness_probe" manifest:"readinessProbe"`
					ResizePolicy []struct {
						ResourceName  types.String `tfsdk:"resource_name" manifest:"resourceName"`
						RestartPolicy types.String `tfsdk:"restart_policy" manifest:"restartPolicy"`
					} `tfsdk:"resize_policy" manifest:"resizePolicy"`
					Resources struct {
						Claims []struct {
							Name types.String `tfsdk:"name" manifest:"name"`
						} `tfsdk:"claims" manifest:"claims"`
						Limits   map[string]types.String `tfsdk:"limits" manifest:"limits"`
						Requests map[string]types.String `tfsdk:"requests" manifest:"requests"`
					} `tfsdk:"resources" manifest:"resources"`
					RestartPolicy   types.String `tfsdk:"restart_policy" manifest:"restartPolicy"`
					SecurityContext struct {
						AllowPrivilegeEscalation types.Bool `tfsdk:"allow_privilege_escalation" manifest:"allowPrivilegeEscalation"`
						Capabilities             struct {
							Add  []types.String `tfsdk:"add" manifest:"add"`
							Drop []types.String `tfsdk:"drop" manifest:"drop"`
						} `tfsdk:"capabilities" manifest:"capabilities"`
						Privileged             types.Bool   `tfsdk:"privileged" manifest:"privileged"`
						ProcMount              types.String `tfsdk:"proc_mount" manifest:"procMount"`
						ReadOnlyRootFilesystem types.Bool   `tfsdk:"read_only_root_filesystem" manifest:"readOnlyRootFilesystem"`
						RunAsGroup             types.Int64  `tfsdk:"run_as_group" manifest:"runAsGroup"`
						RunAsNonRoot           types.Bool   `tfsdk:"run_as_non_root" manifest:"runAsNonRoot"`
						RunAsUser              types.Int64  `tfsdk:"run_as_user" manifest:"runAsUser"`
						SeLinuxOptions         struct {
							Level types.String `tfsdk:"level" manifest:"level"`
							Role  types.String `tfsdk:"role" manifest:"role"`
							Type  types.String `tfsdk:"type" manifest:"type"`
							User  types.String `tfsdk:"user" manifest:"user"`
						} `tfsdk:"se_linux_options" manifest:"seLinuxOptions"`
						SeccompProfile struct {
							LocalhostProfile types.String `tfsdk:"localhost_profile" manifest:"localhostProfile"`
							Type             types.String `tfsdk:"type" manifest:"type"`
						} `tfsdk:"seccomp_profile" manifest:"seccompProfile"`
						WindowsOptions struct {
							GmsaCredentialSpec     types.String `tfsdk:"gmsa_credential_spec" manifest:"gmsaCredentialSpec"`
							GmsaCredentialSpecName types.String `tfsdk:"gmsa_credential_spec_name" manifest:"gmsaCredentialSpecName"`
							HostProcess            types.Bool   `tfsdk:"host_process" manifest:"hostProcess"`
							RunAsUserName          types.String `tfsdk:"run_as_user_name" manifest:"runAsUserName"`
						} `tfsdk:"windows_options" manifest:"windowsOptions"`
					} `tfsdk:"security_context" manifest:"securityContext"`
					StartupProbe struct {
						Exec struct {
							Command []types.String `tfsdk:"command" manifest:"command"`
						} `tfsdk:"exec" manifest:"exec"`
						FailureThreshold types.Int64 `tfsdk:"failure_threshold" manifest:"failureThreshold"`
						Grpc             struct {
							Port    types.Int64  `tfsdk:"port" manifest:"port"`
							Service types.String `tfsdk:"service" manifest:"service"`
						} `tfsdk:"grpc" manifest:"grpc"`
						HttpGet struct {
							Host        types.String `tfsdk:"host" manifest:"host"`
							HttpHeaders []struct {
								Name  types.String `tfsdk:"name" manifest:"name"`
								Value types.String `tfsdk:"value" manifest:"value"`
							} `tfsdk:"http_headers" manifest:"httpHeaders"`
							Path   types.String `tfsdk:"path" manifest:"path"`
							Port   types.String `tfsdk:"port" manifest:"port"`
							Scheme types.String `tfsdk:"scheme" manifest:"scheme"`
						} `tfsdk:"http_get" manifest:"httpGet"`
						InitialDelaySeconds types.Int64 `tfsdk:"initial_delay_seconds" manifest:"initialDelaySeconds"`
						PeriodSeconds       types.Int64 `tfsdk:"period_seconds" manifest:"periodSeconds"`
						SuccessThreshold    types.Int64 `tfsdk:"success_threshold" manifest:"successThreshold"`
						TcpSocket           struct {
							Host types.String `tfsdk:"host" manifest:"host"`
							Port types.String `tfsdk:"port" manifest:"port"`
						} `tfsdk:"tcp_socket" manifest:"tcpSocket"`
						TerminationGracePeriodSeconds types.Int64 `tfsdk:"termination_grace_period_seconds" manifest:"terminationGracePeriodSeconds"`
						TimeoutSeconds                types.Int64 `tfsdk:"timeout_seconds" manifest:"timeoutSeconds"`
					} `tfsdk:"startup_probe" manifest:"startupProbe"`
					Stdin                    types.Bool   `tfsdk:"stdin" manifest:"stdin"`
					StdinOnce                types.Bool   `tfsdk:"stdin_once" manifest:"stdinOnce"`
					TerminationMessagePath   types.String `tfsdk:"termination_message_path" manifest:"terminationMessagePath"`
					TerminationMessagePolicy types.String `tfsdk:"termination_message_policy" manifest:"terminationMessagePolicy"`
					Tty                      types.Bool   `tfsdk:"tty" manifest:"tty"`
					VolumeDevices            []struct {
						DevicePath types.String `tfsdk:"device_path" manifest:"devicePath"`
						Name       types.String `tfsdk:"name" manifest:"name"`
					} `tfsdk:"volume_devices" manifest:"volumeDevices"`
					VolumeMounts []struct {
						MountPath        types.String `tfsdk:"mount_path" manifest:"mountPath"`
						MountPropagation types.String `tfsdk:"mount_propagation" manifest:"mountPropagation"`
						Name             types.String `tfsdk:"name" manifest:"name"`
						ReadOnly         types.Bool   `tfsdk:"read_only" manifest:"readOnly"`
						SubPath          types.String `tfsdk:"sub_path" manifest:"subPath"`
						SubPathExpr      types.String `tfsdk:"sub_path_expr" manifest:"subPathExpr"`
					} `tfsdk:"volume_mounts" manifest:"volumeMounts"`
					WorkingDir types.String `tfsdk:"working_dir" manifest:"workingDir"`
				} `tfsdk:"containers" manifest:"containers"`
				DnsConfig struct {
					Nameservers []types.String `tfsdk:"nameservers" manifest:"nameservers"`
					Options     []struct {
						Name  types.String `tfsdk:"name" manifest:"name"`
						Value types.String `tfsdk:"value" manifest:"value"`
					} `tfsdk:"options" manifest:"options"`
					Searches []types.String `tfsdk:"searches" manifest:"searches"`
				} `tfsdk:"dns_config" manifest:"dnsConfig"`
				DnsPolicy           types.String `tfsdk:"dns_policy" manifest:"dnsPolicy"`
				EnableServiceLinks  types.Bool   `tfsdk:"enable_service_links" manifest:"enableServiceLinks"`
				EphemeralContainers []struct {
					Args    []types.String `tfsdk:"args" manifest:"args"`
					Command []types.String `tfsdk:"command" manifest:"command"`
					Env     []struct {
						Name      types.String `tfsdk:"name" manifest:"name"`
						Value     types.String `tfsdk:"value" manifest:"value"`
						ValueFrom struct {
							ConfigMapKeyRef struct {
								Key      types.String `tfsdk:"key" manifest:"key"`
								Name     types.String `tfsdk:"name" manifest:"name"`
								Optional types.Bool   `tfsdk:"optional" manifest:"optional"`
							} `tfsdk:"config_map_key_ref" manifest:"configMapKeyRef"`
							FieldRef struct {
								APIVersion types.String `tfsdk:"api_version" manifest:"apiVersion"`
								FieldPath  types.String `tfsdk:"field_path" manifest:"fieldPath"`
							} `tfsdk:"field_ref" manifest:"fieldRef"`
							ResourceFieldRef struct {
								ContainerName types.String `tfsdk:"container_name" manifest:"containerName"`
								Divisor       types.String `tfsdk:"divisor" manifest:"divisor"`
								Resource      types.String `tfsdk:"resource" manifest:"resource"`
							} `tfsdk:"resource_field_ref" manifest:"resourceFieldRef"`
							SecretKeyRef struct {
								Key      types.String `tfsdk:"key" manifest:"key"`
								Name     types.String `tfsdk:"name" manifest:"name"`
								Optional types.Bool   `tfsdk:"optional" manifest:"optional"`
							} `tfsdk:"secret_key_ref" manifest:"secretKeyRef"`
						} `tfsdk:"value_from" manifest:"valueFrom"`
					} `tfsdk:"env" manifest:"env"`
					EnvFrom []struct {
						ConfigMapRef struct {
							Name     types.String `tfsdk:"name" manifest:"name"`
							Optional types.Bool   `tfsdk:"optional" manifest:"optional"`
						} `tfsdk:"config_map_ref" manifest:"configMapRef"`
						Prefix    types.String `tfsdk:"prefix" manifest:"prefix"`
						SecretRef struct {
							Name     types.String `tfsdk:"name" manifest:"name"`
							Optional types.Bool   `tfsdk:"optional" manifest:"optional"`
						} `tfsdk:"secret_ref" manifest:"secretRef"`
					} `tfsdk:"env_from" manifest:"envFrom"`
					Image           types.String `tfsdk:"image" manifest:"image"`
					ImagePullPolicy types.String `tfsdk:"image_pull_policy" manifest:"imagePullPolicy"`
					Lifecycle       struct {
						PostStart struct {
							Exec struct {
								Command []types.String `tfsdk:"command" manifest:"command"`
							} `tfsdk:"exec" manifest:"exec"`
							HttpGet struct {
								Host        types.String `tfsdk:"host" manifest:"host"`
								HttpHeaders []struct {
									Name  types.String `tfsdk:"name" manifest:"name"`
									Value types.String `tfsdk:"value" manifest:"value"`
								} `tfsdk:"http_headers" manifest:"httpHeaders"`
								Path   types.String `tfsdk:"path" manifest:"path"`
								Port   types.String `tfsdk:"port" manifest:"port"`
								Scheme types.String `tfsdk:"scheme" manifest:"scheme"`
							} `tfsdk:"http_get" manifest:"httpGet"`
							TcpSocket struct {
								Host types.String `tfsdk:"host" manifest:"host"`
								Port types.String `tfsdk:"port" manifest:"port"`
							} `tfsdk:"tcp_socket" manifest:"tcpSocket"`
						} `tfsdk:"post_start" manifest:"postStart"`
						PreStop struct {
							Exec struct {
								Command []types.String `tfsdk:"command" manifest:"command"`
							} `tfsdk:"exec" manifest:"exec"`
							HttpGet struct {
								Host        types.String `tfsdk:"host" manifest:"host"`
								HttpHeaders []struct {
									Name  types.String `tfsdk:"name" manifest:"name"`
									Value types.String `tfsdk:"value" manifest:"value"`
								} `tfsdk:"http_headers" manifest:"httpHeaders"`
								Path   types.String `tfsdk:"path" manifest:"path"`
								Port   types.String `tfsdk:"port" manifest:"port"`
								Scheme types.String `tfsdk:"scheme" manifest:"scheme"`
							} `tfsdk:"http_get" manifest:"httpGet"`
							TcpSocket struct {
								Host types.String `tfsdk:"host" manifest:"host"`
								Port types.String `tfsdk:"port" manifest:"port"`
							} `tfsdk:"tcp_socket" manifest:"tcpSocket"`
						} `tfsdk:"pre_stop" manifest:"preStop"`
					} `tfsdk:"lifecycle" manifest:"lifecycle"`
					LivenessProbe struct {
						Exec struct {
							Command []types.String `tfsdk:"command" manifest:"command"`
						} `tfsdk:"exec" manifest:"exec"`
						FailureThreshold types.Int64 `tfsdk:"failure_threshold" manifest:"failureThreshold"`
						Grpc             struct {
							Port    types.Int64  `tfsdk:"port" manifest:"port"`
							Service types.String `tfsdk:"service" manifest:"service"`
						} `tfsdk:"grpc" manifest:"grpc"`
						HttpGet struct {
							Host        types.String `tfsdk:"host" manifest:"host"`
							HttpHeaders []struct {
								Name  types.String `tfsdk:"name" manifest:"name"`
								Value types.String `tfsdk:"value" manifest:"value"`
							} `tfsdk:"http_headers" manifest:"httpHeaders"`
							Path   types.String `tfsdk:"path" manifest:"path"`
							Port   types.String `tfsdk:"port" manifest:"port"`
							Scheme types.String `tfsdk:"scheme" manifest:"scheme"`
						} `tfsdk:"http_get" manifest:"httpGet"`
						InitialDelaySeconds types.Int64 `tfsdk:"initial_delay_seconds" manifest:"initialDelaySeconds"`
						PeriodSeconds       types.Int64 `tfsdk:"period_seconds" manifest:"periodSeconds"`
						SuccessThreshold    types.Int64 `tfsdk:"success_threshold" manifest:"successThreshold"`
						TcpSocket           struct {
							Host types.String `tfsdk:"host" manifest:"host"`
							Port types.String `tfsdk:"port" manifest:"port"`
						} `tfsdk:"tcp_socket" manifest:"tcpSocket"`
						TerminationGracePeriodSeconds types.Int64 `tfsdk:"termination_grace_period_seconds" manifest:"terminationGracePeriodSeconds"`
						TimeoutSeconds                types.Int64 `tfsdk:"timeout_seconds" manifest:"timeoutSeconds"`
					} `tfsdk:"liveness_probe" manifest:"livenessProbe"`
					Name  types.String `tfsdk:"name" manifest:"name"`
					Ports []struct {
						ContainerPort types.Int64  `tfsdk:"container_port" manifest:"containerPort"`
						HostIp        types.String `tfsdk:"host_ip" manifest:"hostIp"`
						HostPort      types.Int64  `tfsdk:"host_port" manifest:"hostPort"`
						Name          types.String `tfsdk:"name" manifest:"name"`
						Protocol      types.String `tfsdk:"protocol" manifest:"protocol"`
					} `tfsdk:"ports" manifest:"ports"`
					ReadinessProbe struct {
						Exec struct {
							Command []types.String `tfsdk:"command" manifest:"command"`
						} `tfsdk:"exec" manifest:"exec"`
						FailureThreshold types.Int64 `tfsdk:"failure_threshold" manifest:"failureThreshold"`
						Grpc             struct {
							Port    types.Int64  `tfsdk:"port" manifest:"port"`
							Service types.String `tfsdk:"service" manifest:"service"`
						} `tfsdk:"grpc" manifest:"grpc"`
						HttpGet struct {
							Host        types.String `tfsdk:"host" manifest:"host"`
							HttpHeaders []struct {
								Name  types.String `tfsdk:"name" manifest:"name"`
								Value types.String `tfsdk:"value" manifest:"value"`
							} `tfsdk:"http_headers" manifest:"httpHeaders"`
							Path   types.String `tfsdk:"path" manifest:"path"`
							Port   types.String `tfsdk:"port" manifest:"port"`
							Scheme types.String `tfsdk:"scheme" manifest:"scheme"`
						} `tfsdk:"http_get" manifest:"httpGet"`
						InitialDelaySeconds types.Int64 `tfsdk:"initial_delay_seconds" manifest:"initialDelaySeconds"`
						PeriodSeconds       types.Int64 `tfsdk:"period_seconds" manifest:"periodSeconds"`
						SuccessThreshold    types.Int64 `tfsdk:"success_threshold" manifest:"successThreshold"`
						TcpSocket           struct {
							Host types.String `tfsdk:"host" manifest:"host"`
							Port types.String `tfsdk:"port" manifest:"port"`
						} `tfsdk:"tcp_socket" manifest:"tcpSocket"`
						TerminationGracePeriodSeconds types.Int64 `tfsdk:"termination_grace_period_seconds" manifest:"terminationGracePeriodSeconds"`
						TimeoutSeconds                types.Int64 `tfsdk:"timeout_seconds" manifest:"timeoutSeconds"`
					} `tfsdk:"readiness_probe" manifest:"readinessProbe"`
					ResizePolicy []struct {
						ResourceName  types.String `tfsdk:"resource_name" manifest:"resourceName"`
						RestartPolicy types.String `tfsdk:"restart_policy" manifest:"restartPolicy"`
					} `tfsdk:"resize_policy" manifest:"resizePolicy"`
					Resources struct {
						Claims []struct {
							Name types.String `tfsdk:"name" manifest:"name"`
						} `tfsdk:"claims" manifest:"claims"`
						Limits   map[string]types.String `tfsdk:"limits" manifest:"limits"`
						Requests map[string]types.String `tfsdk:"requests" manifest:"requests"`
					} `tfsdk:"resources" manifest:"resources"`
					RestartPolicy   types.String `tfsdk:"restart_policy" manifest:"restartPolicy"`
					SecurityContext struct {
						AllowPrivilegeEscalation types.Bool `tfsdk:"allow_privilege_escalation" manifest:"allowPrivilegeEscalation"`
						Capabilities             struct {
							Add  []types.String `tfsdk:"add" manifest:"add"`
							Drop []types.String `tfsdk:"drop" manifest:"drop"`
						} `tfsdk:"capabilities" manifest:"capabilities"`
						Privileged             types.Bool   `tfsdk:"privileged" manifest:"privileged"`
						ProcMount              types.String `tfsdk:"proc_mount" manifest:"procMount"`
						ReadOnlyRootFilesystem types.Bool   `tfsdk:"read_only_root_filesystem" manifest:"readOnlyRootFilesystem"`
						RunAsGroup             types.Int64  `tfsdk:"run_as_group" manifest:"runAsGroup"`
						RunAsNonRoot           types.Bool   `tfsdk:"run_as_non_root" manifest:"runAsNonRoot"`
						RunAsUser              types.Int64  `tfsdk:"run_as_user" manifest:"runAsUser"`
						SeLinuxOptions         struct {
							Level types.String `tfsdk:"level" manifest:"level"`
							Role  types.String `tfsdk:"role" manifest:"role"`
							Type  types.String `tfsdk:"type" manifest:"type"`
							User  types.String `tfsdk:"user" manifest:"user"`
						} `tfsdk:"se_linux_options" manifest:"seLinuxOptions"`
						SeccompProfile struct {
							LocalhostProfile types.String `tfsdk:"localhost_profile" manifest:"localhostProfile"`
							Type             types.String `tfsdk:"type" manifest:"type"`
						} `tfsdk:"seccomp_profile" manifest:"seccompProfile"`
						WindowsOptions struct {
							GmsaCredentialSpec     types.String `tfsdk:"gmsa_credential_spec" manifest:"gmsaCredentialSpec"`
							GmsaCredentialSpecName types.String `tfsdk:"gmsa_credential_spec_name" manifest:"gmsaCredentialSpecName"`
							HostProcess            types.Bool   `tfsdk:"host_process" manifest:"hostProcess"`
							RunAsUserName          types.String `tfsdk:"run_as_user_name" manifest:"runAsUserName"`
						} `tfsdk:"windows_options" manifest:"windowsOptions"`
					} `tfsdk:"security_context" manifest:"securityContext"`
					StartupProbe struct {
						Exec struct {
							Command []types.String `tfsdk:"command" manifest:"command"`
						} `tfsdk:"exec" manifest:"exec"`
						FailureThreshold types.Int64 `tfsdk:"failure_threshold" manifest:"failureThreshold"`
						Grpc             struct {
							Port    types.Int64  `tfsdk:"port" manifest:"port"`
							Service types.String `tfsdk:"service" manifest:"service"`
						} `tfsdk:"grpc" manifest:"grpc"`
						HttpGet struct {
							Host        types.String `tfsdk:"host" manifest:"host"`
							HttpHeaders []struct {
								Name  types.String `tfsdk:"name" manifest:"name"`
								Value types.String `tfsdk:"value" manifest:"value"`
							} `tfsdk:"http_headers" manifest:"httpHeaders"`
							Path   types.String `tfsdk:"path" manifest:"path"`
							Port   types.String `tfsdk:"port" manifest:"port"`
							Scheme types.String `tfsdk:"scheme" manifest:"scheme"`
						} `tfsdk:"http_get" manifest:"httpGet"`
						InitialDelaySeconds types.Int64 `tfsdk:"initial_delay_seconds" manifest:"initialDelaySeconds"`
						PeriodSeconds       types.Int64 `tfsdk:"period_seconds" manifest:"periodSeconds"`
						SuccessThreshold    types.Int64 `tfsdk:"success_threshold" manifest:"successThreshold"`
						TcpSocket           struct {
							Host types.String `tfsdk:"host" manifest:"host"`
							Port types.String `tfsdk:"port" manifest:"port"`
						} `tfsdk:"tcp_socket" manifest:"tcpSocket"`
						TerminationGracePeriodSeconds types.Int64 `tfsdk:"termination_grace_period_seconds" manifest:"terminationGracePeriodSeconds"`
						TimeoutSeconds                types.Int64 `tfsdk:"timeout_seconds" manifest:"timeoutSeconds"`
					} `tfsdk:"startup_probe" manifest:"startupProbe"`
					Stdin                    types.Bool   `tfsdk:"stdin" manifest:"stdin"`
					StdinOnce                types.Bool   `tfsdk:"stdin_once" manifest:"stdinOnce"`
					TargetContainerName      types.String `tfsdk:"target_container_name" manifest:"targetContainerName"`
					TerminationMessagePath   types.String `tfsdk:"termination_message_path" manifest:"terminationMessagePath"`
					TerminationMessagePolicy types.String `tfsdk:"termination_message_policy" manifest:"terminationMessagePolicy"`
					Tty                      types.Bool   `tfsdk:"tty" manifest:"tty"`
					VolumeDevices            []struct {
						DevicePath types.String `tfsdk:"device_path" manifest:"devicePath"`
						Name       types.String `tfsdk:"name" manifest:"name"`
					} `tfsdk:"volume_devices" manifest:"volumeDevices"`
					VolumeMounts []struct {
						MountPath        types.String `tfsdk:"mount_path" manifest:"mountPath"`
						MountPropagation types.String `tfsdk:"mount_propagation" manifest:"mountPropagation"`
						Name             types.String `tfsdk:"name" manifest:"name"`
						ReadOnly         types.Bool   `tfsdk:"read_only" manifest:"readOnly"`
						SubPath          types.String `tfsdk:"sub_path" manifest:"subPath"`
						SubPathExpr      types.String `tfsdk:"sub_path_expr" manifest:"subPathExpr"`
					} `tfsdk:"volume_mounts" manifest:"volumeMounts"`
					WorkingDir types.String `tfsdk:"working_dir" manifest:"workingDir"`
				} `tfsdk:"ephemeral_containers" manifest:"ephemeralContainers"`
				HostAliases []struct {
					Hostnames []types.String `tfsdk:"hostnames" manifest:"hostnames"`
					Ip        types.String   `tfsdk:"ip" manifest:"ip"`
				} `tfsdk:"host_aliases" manifest:"hostAliases"`
				HostIpc          types.Bool   `tfsdk:"host_ipc" manifest:"hostIpc"`
				HostNetwork      types.Bool   `tfsdk:"host_network" manifest:"hostNetwork"`
				HostPid          types.Bool   `tfsdk:"host_pid" manifest:"hostPid"`
				HostUsers        types.Bool   `tfsdk:"host_users" manifest:"hostUsers"`
				Hostname         types.String `tfsdk:"hostname" manifest:"hostname"`
				ImagePullSecrets []struct {
					Name types.String `tfsdk:"name" manifest:"name"`
				} `tfsdk:"image_pull_secrets" manifest:"imagePullSecrets"`
				InitContainers []struct {
					Args    []types.String `tfsdk:"args" manifest:"args"`
					Command []types.String `tfsdk:"command" manifest:"command"`
					Env     []struct {
						Name      types.String `tfsdk:"name" manifest:"name"`
						Value     types.String `tfsdk:"value" manifest:"value"`
						ValueFrom struct {
							ConfigMapKeyRef struct {
								Key      types.String `tfsdk:"key" manifest:"key"`
								Name     types.String `tfsdk:"name" manifest:"name"`
								Optional types.Bool   `tfsdk:"optional" manifest:"optional"`
							} `tfsdk:"config_map_key_ref" manifest:"configMapKeyRef"`
							FieldRef struct {
								APIVersion types.String `tfsdk:"api_version" manifest:"apiVersion"`
								FieldPath  types.String `tfsdk:"field_path" manifest:"fieldPath"`
							} `tfsdk:"field_ref" manifest:"fieldRef"`
							ResourceFieldRef struct {
								ContainerName types.String `tfsdk:"container_name" manifest:"containerName"`
								Divisor       types.String `tfsdk:"divisor" manifest:"divisor"`
								Resource      types.String `tfsdk:"resource" manifest:"resource"`
							} `tfsdk:"resource_field_ref" manifest:"resourceFieldRef"`
							SecretKeyRef struct {
								Key      types.String `tfsdk:"key" manifest:"key"`
								Name     types.String `tfsdk:"name" manifest:"name"`
								Optional types.Bool   `tfsdk:"optional" manifest:"optional"`
							} `tfsdk:"secret_key_ref" manifest:"secretKeyRef"`
						} `tfsdk:"value_from" manifest:"valueFrom"`
					} `tfsdk:"env" manifest:"env"`
					EnvFrom []struct {
						ConfigMapRef struct {
							Name     types.String `tfsdk:"name" manifest:"name"`
							Optional types.Bool   `tfsdk:"optional" manifest:"optional"`
						} `tfsdk:"config_map_ref" manifest:"configMapRef"`
						Prefix    types.String `tfsdk:"prefix" manifest:"prefix"`
						SecretRef struct {
							Name     types.String `tfsdk:"name" manifest:"name"`
							Optional types.Bool   `tfsdk:"optional" manifest:"optional"`
						} `tfsdk:"secret_ref" manifest:"secretRef"`
					} `tfsdk:"env_from" manifest:"envFrom"`
					Image           types.String `tfsdk:"image" manifest:"image"`
					ImagePullPolicy types.String `tfsdk:"image_pull_policy" manifest:"imagePullPolicy"`
					Lifecycle       struct {
						PostStart struct {
							Exec struct {
								Command []types.String `tfsdk:"command" manifest:"command"`
							} `tfsdk:"exec" manifest:"exec"`
							HttpGet struct {
								Host        types.String `tfsdk:"host" manifest:"host"`
								HttpHeaders []struct {
									Name  types.String `tfsdk:"name" manifest:"name"`
									Value types.String `tfsdk:"value" manifest:"value"`
								} `tfsdk:"http_headers" manifest:"httpHeaders"`
								Path   types.String `tfsdk:"path" manifest:"path"`
								Port   types.String `tfsdk:"port" manifest:"port"`
								Scheme types.String `tfsdk:"scheme" manifest:"scheme"`
							} `tfsdk:"http_get" manifest:"httpGet"`
							TcpSocket struct {
								Host types.String `tfsdk:"host" manifest:"host"`
								Port types.String `tfsdk:"port" manifest:"port"`
							} `tfsdk:"tcp_socket" manifest:"tcpSocket"`
						} `tfsdk:"post_start" manifest:"postStart"`
						PreStop struct {
							Exec struct {
								Command []types.String `tfsdk:"command" manifest:"command"`
							} `tfsdk:"exec" manifest:"exec"`
							HttpGet struct {
								Host        types.String `tfsdk:"host" manifest:"host"`
								HttpHeaders []struct {
									Name  types.String `tfsdk:"name" manifest:"name"`
									Value types.String `tfsdk:"value" manifest:"value"`
								} `tfsdk:"http_headers" manifest:"httpHeaders"`
								Path   types.String `tfsdk:"path" manifest:"path"`
								Port   types.String `tfsdk:"port" manifest:"port"`
								Scheme types.String `tfsdk:"scheme" manifest:"scheme"`
							} `tfsdk:"http_get" manifest:"httpGet"`
							TcpSocket struct {
								Host types.String `tfsdk:"host" manifest:"host"`
								Port types.String `tfsdk:"port" manifest:"port"`
							} `tfsdk:"tcp_socket" manifest:"tcpSocket"`
						} `tfsdk:"pre_stop" manifest:"preStop"`
					} `tfsdk:"lifecycle" manifest:"lifecycle"`
					LivenessProbe struct {
						Exec struct {
							Command []types.String `tfsdk:"command" manifest:"command"`
						} `tfsdk:"exec" manifest:"exec"`
						FailureThreshold types.Int64 `tfsdk:"failure_threshold" manifest:"failureThreshold"`
						Grpc             struct {
							Port    types.Int64  `tfsdk:"port" manifest:"port"`
							Service types.String `tfsdk:"service" manifest:"service"`
						} `tfsdk:"grpc" manifest:"grpc"`
						HttpGet struct {
							Host        types.String `tfsdk:"host" manifest:"host"`
							HttpHeaders []struct {
								Name  types.String `tfsdk:"name" manifest:"name"`
								Value types.String `tfsdk:"value" manifest:"value"`
							} `tfsdk:"http_headers" manifest:"httpHeaders"`
							Path   types.String `tfsdk:"path" manifest:"path"`
							Port   types.String `tfsdk:"port" manifest:"port"`
							Scheme types.String `tfsdk:"scheme" manifest:"scheme"`
						} `tfsdk:"http_get" manifest:"httpGet"`
						InitialDelaySeconds types.Int64 `tfsdk:"initial_delay_seconds" manifest:"initialDelaySeconds"`
						PeriodSeconds       types.Int64 `tfsdk:"period_seconds" manifest:"periodSeconds"`
						SuccessThreshold    types.Int64 `tfsdk:"success_threshold" manifest:"successThreshold"`
						TcpSocket           struct {
							Host types.String `tfsdk:"host" manifest:"host"`
							Port types.String `tfsdk:"port" manifest:"port"`
						} `tfsdk:"tcp_socket" manifest:"tcpSocket"`
						TerminationGracePeriodSeconds types.Int64 `tfsdk:"termination_grace_period_seconds" manifest:"terminationGracePeriodSeconds"`
						TimeoutSeconds                types.Int64 `tfsdk:"timeout_seconds" manifest:"timeoutSeconds"`
					} `tfsdk:"liveness_probe" manifest:"livenessProbe"`
					Name  types.String `tfsdk:"name" manifest:"name"`
					Ports []struct {
						ContainerPort types.Int64  `tfsdk:"container_port" manifest:"containerPort"`
						HostIp        types.String `tfsdk:"host_ip" manifest:"hostIp"`
						HostPort      types.Int64  `tfsdk:"host_port" manifest:"hostPort"`
						Name          types.String `tfsdk:"name" manifest:"name"`
						Protocol      types.String `tfsdk:"protocol" manifest:"protocol"`
					} `tfsdk:"ports" manifest:"ports"`
					ReadinessProbe struct {
						Exec struct {
							Command []types.String `tfsdk:"command" manifest:"command"`
						} `tfsdk:"exec" manifest:"exec"`
						FailureThreshold types.Int64 `tfsdk:"failure_threshold" manifest:"failureThreshold"`
						Grpc             struct {
							Port    types.Int64  `tfsdk:"port" manifest:"port"`
							Service types.String `tfsdk:"service" manifest:"service"`
						} `tfsdk:"grpc" manifest:"grpc"`
						HttpGet struct {
							Host        types.String `tfsdk:"host" manifest:"host"`
							HttpHeaders []struct {
								Name  types.String `tfsdk:"name" manifest:"name"`
								Value types.String `tfsdk:"value" manifest:"value"`
							} `tfsdk:"http_headers" manifest:"httpHeaders"`
							Path   types.String `tfsdk:"path" manifest:"path"`
							Port   types.String `tfsdk:"port" manifest:"port"`
							Scheme types.String `tfsdk:"scheme" manifest:"scheme"`
						} `tfsdk:"http_get" manifest:"httpGet"`
						InitialDelaySeconds types.Int64 `tfsdk:"initial_delay_seconds" manifest:"initialDelaySeconds"`
						PeriodSeconds       types.Int64 `tfsdk:"period_seconds" manifest:"periodSeconds"`
						SuccessThreshold    types.Int64 `tfsdk:"success_threshold" manifest:"successThreshold"`
						TcpSocket           struct {
							Host types.String `tfsdk:"host" manifest:"host"`
							Port types.String `tfsdk:"port" manifest:"port"`
						} `tfsdk:"tcp_socket" manifest:"tcpSocket"`
						TerminationGracePeriodSeconds types.Int64 `tfsdk:"termination_grace_period_seconds" manifest:"terminationGracePeriodSeconds"`
						TimeoutSeconds                types.Int64 `tfsdk:"timeout_seconds" manifest:"timeoutSeconds"`
					} `tfsdk:"readiness_probe" manifest:"readinessProbe"`
					ResizePolicy []struct {
						ResourceName  types.String `tfsdk:"resource_name" manifest:"resourceName"`
						RestartPolicy types.String `tfsdk:"restart_policy" manifest:"restartPolicy"`
					} `tfsdk:"resize_policy" manifest:"resizePolicy"`
					Resources struct {
						Claims []struct {
							Name types.String `tfsdk:"name" manifest:"name"`
						} `tfsdk:"claims" manifest:"claims"`
						Limits   map[string]types.String `tfsdk:"limits" manifest:"limits"`
						Requests map[string]types.String `tfsdk:"requests" manifest:"requests"`
					} `tfsdk:"resources" manifest:"resources"`
					RestartPolicy   types.String `tfsdk:"restart_policy" manifest:"restartPolicy"`
					SecurityContext struct {
						AllowPrivilegeEscalation types.Bool `tfsdk:"allow_privilege_escalation" manifest:"allowPrivilegeEscalation"`
						Capabilities             struct {
							Add  []types.String `tfsdk:"add" manifest:"add"`
							Drop []types.String `tfsdk:"drop" manifest:"drop"`
						} `tfsdk:"capabilities" manifest:"capabilities"`
						Privileged             types.Bool   `tfsdk:"privileged" manifest:"privileged"`
						ProcMount              types.String `tfsdk:"proc_mount" manifest:"procMount"`
						ReadOnlyRootFilesystem types.Bool   `tfsdk:"read_only_root_filesystem" manifest:"readOnlyRootFilesystem"`
						RunAsGroup             types.Int64  `tfsdk:"run_as_group" manifest:"runAsGroup"`
						RunAsNonRoot           types.Bool   `tfsdk:"run_as_non_root" manifest:"runAsNonRoot"`
						RunAsUser              types.Int64  `tfsdk:"run_as_user" manifest:"runAsUser"`
						SeLinuxOptions         struct {
							Level types.String `tfsdk:"level" manifest:"level"`
							Role  types.String `tfsdk:"role" manifest:"role"`
							Type  types.String `tfsdk:"type" manifest:"type"`
							User  types.String `tfsdk:"user" manifest:"user"`
						} `tfsdk:"se_linux_options" manifest:"seLinuxOptions"`
						SeccompProfile struct {
							LocalhostProfile types.String `tfsdk:"localhost_profile" manifest:"localhostProfile"`
							Type             types.String `tfsdk:"type" manifest:"type"`
						} `tfsdk:"seccomp_profile" manifest:"seccompProfile"`
						WindowsOptions struct {
							GmsaCredentialSpec     types.String `tfsdk:"gmsa_credential_spec" manifest:"gmsaCredentialSpec"`
							GmsaCredentialSpecName types.String `tfsdk:"gmsa_credential_spec_name" manifest:"gmsaCredentialSpecName"`
							HostProcess            types.Bool   `tfsdk:"host_process" manifest:"hostProcess"`
							RunAsUserName          types.String `tfsdk:"run_as_user_name" manifest:"runAsUserName"`
						} `tfsdk:"windows_options" manifest:"windowsOptions"`
					} `tfsdk:"security_context" manifest:"securityContext"`
					StartupProbe struct {
						Exec struct {
							Command []types.String `tfsdk:"command" manifest:"command"`
						} `tfsdk:"exec" manifest:"exec"`
						FailureThreshold types.Int64 `tfsdk:"failure_threshold" manifest:"failureThreshold"`
						Grpc             struct {
							Port    types.Int64  `tfsdk:"port" manifest:"port"`
							Service types.String `tfsdk:"service" manifest:"service"`
						} `tfsdk:"grpc" manifest:"grpc"`
						HttpGet struct {
							Host        types.String `tfsdk:"host" manifest:"host"`
							HttpHeaders []struct {
								Name  types.String `tfsdk:"name" manifest:"name"`
								Value types.String `tfsdk:"value" manifest:"value"`
							} `tfsdk:"http_headers" manifest:"httpHeaders"`
							Path   types.String `tfsdk:"path" manifest:"path"`
							Port   types.String `tfsdk:"port" manifest:"port"`
							Scheme types.String `tfsdk:"scheme" manifest:"scheme"`
						} `tfsdk:"http_get" manifest:"httpGet"`
						InitialDelaySeconds types.Int64 `tfsdk:"initial_delay_seconds" manifest:"initialDelaySeconds"`
						PeriodSeconds       types.Int64 `tfsdk:"period_seconds" manifest:"periodSeconds"`
						SuccessThreshold    types.Int64 `tfsdk:"success_threshold" manifest:"successThreshold"`
						TcpSocket           struct {
							Host types.String `tfsdk:"host" manifest:"host"`
							Port types.String `tfsdk:"port" manifest:"port"`
						} `tfsdk:"tcp_socket" manifest:"tcpSocket"`
						TerminationGracePeriodSeconds types.Int64 `tfsdk:"termination_grace_period_seconds" manifest:"terminationGracePeriodSeconds"`
						TimeoutSeconds                types.Int64 `tfsdk:"timeout_seconds" manifest:"timeoutSeconds"`
					} `tfsdk:"startup_probe" manifest:"startupProbe"`
					Stdin                    types.Bool   `tfsdk:"stdin" manifest:"stdin"`
					StdinOnce                types.Bool   `tfsdk:"stdin_once" manifest:"stdinOnce"`
					TerminationMessagePath   types.String `tfsdk:"termination_message_path" manifest:"terminationMessagePath"`
					TerminationMessagePolicy types.String `tfsdk:"termination_message_policy" manifest:"terminationMessagePolicy"`
					Tty                      types.Bool   `tfsdk:"tty" manifest:"tty"`
					VolumeDevices            []struct {
						DevicePath types.String `tfsdk:"device_path" manifest:"devicePath"`
						Name       types.String `tfsdk:"name" manifest:"name"`
					} `tfsdk:"volume_devices" manifest:"volumeDevices"`
					VolumeMounts []struct {
						MountPath        types.String `tfsdk:"mount_path" manifest:"mountPath"`
						MountPropagation types.String `tfsdk:"mount_propagation" manifest:"mountPropagation"`
						Name             types.String `tfsdk:"name" manifest:"name"`
						ReadOnly         types.Bool   `tfsdk:"read_only" manifest:"readOnly"`
						SubPath          types.String `tfsdk:"sub_path" manifest:"subPath"`
						SubPathExpr      types.String `tfsdk:"sub_path_expr" manifest:"subPathExpr"`
					} `tfsdk:"volume_mounts" manifest:"volumeMounts"`
					WorkingDir types.String `tfsdk:"working_dir" manifest:"workingDir"`
				} `tfsdk:"init_containers" manifest:"initContainers"`
				NodeName     types.String            `tfsdk:"node_name" manifest:"nodeName"`
				NodeSelector map[string]types.String `tfsdk:"node_selector" manifest:"nodeSelector"`
				Os           struct {
					Name types.String `tfsdk:"name" manifest:"name"`
				} `tfsdk:"os" manifest:"os"`
				Overhead          map[string]types.String `tfsdk:"overhead" manifest:"overhead"`
				PreemptionPolicy  types.String            `tfsdk:"preemption_policy" manifest:"preemptionPolicy"`
				Priority          types.Int64             `tfsdk:"priority" manifest:"priority"`
				PriorityClassName types.String            `tfsdk:"priority_class_name" manifest:"priorityClassName"`
				ReadinessGates    []struct {
					ConditionType types.String `tfsdk:"condition_type" manifest:"conditionType"`
				} `tfsdk:"readiness_gates" manifest:"readinessGates"`
				ResourceClaims []struct {
					Name   types.String `tfsdk:"name" manifest:"name"`
					Source struct {
						ResourceClaimName         types.String `tfsdk:"resource_claim_name" manifest:"resourceClaimName"`
						ResourceClaimTemplateName types.String `tfsdk:"resource_claim_template_name" manifest:"resourceClaimTemplateName"`
					} `tfsdk:"source" manifest:"source"`
				} `tfsdk:"resource_claims" manifest:"resourceClaims"`
				RestartPolicy    types.String `tfsdk:"restart_policy" manifest:"restartPolicy"`
				RuntimeClassName types.String `tfsdk:"runtime_class_name" manifest:"runtimeClassName"`
				SchedulerName    types.String `tfsdk:"scheduler_name" manifest:"schedulerName"`
				SchedulingGates  []struct {
					Name types.String `tfsdk:"name" manifest:"name"`
				} `tfsdk:"scheduling_gates" manifest:"schedulingGates"`
				SecurityContext struct {
					FsGroup             types.Int64  `tfsdk:"fs_group" manifest:"fsGroup"`
					FsGroupChangePolicy types.String `tfsdk:"fs_group_change_policy" manifest:"fsGroupChangePolicy"`
					RunAsGroup          types.Int64  `tfsdk:"run_as_group" manifest:"runAsGroup"`
					RunAsNonRoot        types.Bool   `tfsdk:"run_as_non_root" manifest:"runAsNonRoot"`
					RunAsUser           types.Int64  `tfsdk:"run_as_user" manifest:"runAsUser"`
					SeLinuxOptions      struct {
						Level types.String `tfsdk:"level" manifest:"level"`
						Role  types.String `tfsdk:"role" manifest:"role"`
						Type  types.String `tfsdk:"type" manifest:"type"`
						User  types.String `tfsdk:"user" manifest:"user"`
					} `tfsdk:"se_linux_options" manifest:"seLinuxOptions"`
					SeccompProfile struct {
						LocalhostProfile types.String `tfsdk:"localhost_profile" manifest:"localhostProfile"`
						Type             types.String `tfsdk:"type" manifest:"type"`
					} `tfsdk:"seccomp_profile" manifest:"seccompProfile"`
					SupplementalGroups []types.Int64 `tfsdk:"supplemental_groups" manifest:"supplementalGroups"`
					Sysctls            []struct {
						Name  types.String `tfsdk:"name" manifest:"name"`
						Value types.String `tfsdk:"value" manifest:"value"`
					} `tfsdk:"sysctls" manifest:"sysctls"`
					WindowsOptions struct {
						GmsaCredentialSpec     types.String `tfsdk:"gmsa_credential_spec" manifest:"gmsaCredentialSpec"`
						GmsaCredentialSpecName types.String `tfsdk:"gmsa_credential_spec_name" manifest:"gmsaCredentialSpecName"`
						HostProcess            types.Bool   `tfsdk:"host_process" manifest:"hostProcess"`
						RunAsUserName          types.String `tfsdk:"run_as_user_name" manifest:"runAsUserName"`
					} `tfsdk:"windows_options" manifest:"windowsOptions"`
				} `tfsdk:"security_context" manifest:"securityContext"`
				ServiceAccount                types.String `tfsdk:"service_account" manifest:"serviceAccount"`
				ServiceAccountName            types.String `tfsdk:"service_account_name" manifest:"serviceAccountName"`
				SetHostnameAsFqdn             types.Bool   `tfsdk:"set_hostname_as_fqdn" manifest:"setHostnameAsFqdn"`
				ShareProcessNamespace         types.Bool   `tfsdk:"share_process_namespace" manifest:"shareProcessNamespace"`
				Subdomain                     types.String `tfsdk:"subdomain" manifest:"subdomain"`
				TerminationGracePeriodSeconds types.Int64  `tfsdk:"termination_grace_period_seconds" manifest:"terminationGracePeriodSeconds"`
				Tolerations                   []struct {
					Effect            types.String `tfsdk:"effect" manifest:"effect"`
					Key               types.String `tfsdk:"key" manifest:"key"`
					Operator          types.String `tfsdk:"operator" manifest:"operator"`
					TolerationSeconds types.Int64  `tfsdk:"toleration_seconds" manifest:"tolerationSeconds"`
					Value             types.String `tfsdk:"value" manifest:"value"`
				} `tfsdk:"tolerations" manifest:"tolerations"`
				TopologySpreadConstraints []struct {
					LabelSelector struct {
						MatchExpressions []struct {
							Key      types.String   `tfsdk:"key" manifest:"key"`
							Operator types.String   `tfsdk:"operator" manifest:"operator"`
							Values   []types.String `tfsdk:"values" manifest:"values"`
						} `tfsdk:"match_expressions" manifest:"matchExpressions"`
						MatchLabels map[string]types.String `tfsdk:"match_labels" manifest:"matchLabels"`
					} `tfsdk:"label_selector" manifest:"labelSelector"`
					MatchLabelKeys     []types.String `tfsdk:"match_label_keys" manifest:"matchLabelKeys"`
					MaxSkew            types.Int64    `tfsdk:"max_skew" manifest:"maxSkew"`
					MinDomains         types.Int64    `tfsdk:"min_domains" manifest:"minDomains"`
					NodeAffinityPolicy types.String   `tfsdk:"node_affinity_policy" manifest:"nodeAffinityPolicy"`
					NodeTaintsPolicy   types.String   `tfsdk:"node_taints_policy" manifest:"nodeTaintsPolicy"`
					TopologyKey        types.String   `tfsdk:"topology_key" manifest:"topologyKey"`
					WhenUnsatisfiable  types.String   `tfsdk:"when_unsatisfiable" manifest:"whenUnsatisfiable"`
				} `tfsdk:"topology_spread_constraints" manifest:"topologySpreadConstraints"`
				Volumes []struct {
					AwsElasticBlockStore struct {
						FsType    types.String `tfsdk:"fs_type" manifest:"fsType"`
						Partition types.Int64  `tfsdk:"partition" manifest:"partition"`
						ReadOnly  types.Bool   `tfsdk:"read_only" manifest:"readOnly"`
						VolumeId  types.String `tfsdk:"volume_id" manifest:"volumeId"`
					} `tfsdk:"aws_elastic_block_store" manifest:"awsElasticBlockStore"`
					AzureDisk struct {
						CachingMode types.String `tfsdk:"caching_mode" manifest:"cachingMode"`
						DiskName    types.String `tfsdk:"disk_name" manifest:"diskName"`
						DiskUri     types.String `tfsdk:"disk_uri" manifest:"diskUri"`
						FsType      types.String `tfsdk:"fs_type" manifest:"fsType"`
						Kind        types.String `tfsdk:"kind" manifest:"kind"`
						ReadOnly    types.Bool   `tfsdk:"read_only" manifest:"readOnly"`
					} `tfsdk:"azure_disk" manifest:"azureDisk"`
					AzureFile struct {
						ReadOnly   types.Bool   `tfsdk:"read_only" manifest:"readOnly"`
						SecretName types.String `tfsdk:"secret_name" manifest:"secretName"`
						ShareName  types.String `tfsdk:"share_name" manifest:"shareName"`
					} `tfsdk:"azure_file" manifest:"azureFile"`
					Cephfs struct {
						Monitors   []types.String `tfsdk:"monitors" manifest:"monitors"`
						Path       types.String   `tfsdk:"path" manifest:"path"`
						ReadOnly   types.Bool     `tfsdk:"read_only" manifest:"readOnly"`
						SecretFile types.String   `tfsdk:"secret_file" manifest:"secretFile"`
						SecretRef  struct {
							Name types.String `tfsdk:"name" manifest:"name"`
						} `tfsdk:"secret_ref" manifest:"secretRef"`
						User types.String `tfsdk:"user" manifest:"user"`
					} `tfsdk:"cephfs" manifest:"cephfs"`
					Cinder struct {
						FsType    types.String `tfsdk:"fs_type" manifest:"fsType"`
						ReadOnly  types.Bool   `tfsdk:"read_only" manifest:"readOnly"`
						SecretRef struct {
							Name types.String `tfsdk:"name" manifest:"name"`
						} `tfsdk:"secret_ref" manifest:"secretRef"`
						VolumeId types.String `tfsdk:"volume_id" manifest:"volumeId"`
					} `tfsdk:"cinder" manifest:"cinder"`
					ConfigMap struct {
						DefaultMode types.Int64 `tfsdk:"default_mode" manifest:"defaultMode"`
						Items       []struct {
							Key  types.String `tfsdk:"key" manifest:"key"`
							Mode types.Int64  `tfsdk:"mode" manifest:"mode"`
							Path types.String `tfsdk:"path" manifest:"path"`
						} `tfsdk:"items" manifest:"items"`
						Name     types.String `tfsdk:"name" manifest:"name"`
						Optional types.Bool   `tfsdk:"optional" manifest:"optional"`
					} `tfsdk:"config_map" manifest:"configMap"`
					Csi struct {
						Driver               types.String `tfsdk:"driver" manifest:"driver"`
						FsType               types.String `tfsdk:"fs_type" manifest:"fsType"`
						NodePublishSecretRef struct {
							Name types.String `tfsdk:"name" manifest:"name"`
						} `tfsdk:"node_publish_secret_ref" manifest:"nodePublishSecretRef"`
						ReadOnly         types.Bool              `tfsdk:"read_only" manifest:"readOnly"`
						VolumeAttributes map[string]types.String `tfsdk:"volume_attributes" manifest:"volumeAttributes"`
					} `tfsdk:"csi" manifest:"csi"`
					DownwardApi struct {
						DefaultMode types.Int64 `tfsdk:"default_mode" manifest:"defaultMode"`
						Items       []struct {
							FieldRef struct {
								APIVersion types.String `tfsdk:"api_version" manifest:"apiVersion"`
								FieldPath  types.String `tfsdk:"field_path" manifest:"fieldPath"`
							} `tfsdk:"field_ref" manifest:"fieldRef"`
							Mode             types.Int64  `tfsdk:"mode" manifest:"mode"`
							Path             types.String `tfsdk:"path" manifest:"path"`
							ResourceFieldRef struct {
								ContainerName types.String `tfsdk:"container_name" manifest:"containerName"`
								Divisor       types.String `tfsdk:"divisor" manifest:"divisor"`
								Resource      types.String `tfsdk:"resource" manifest:"resource"`
							} `tfsdk:"resource_field_ref" manifest:"resourceFieldRef"`
						} `tfsdk:"items" manifest:"items"`
					} `tfsdk:"downward_api" manifest:"downwardApi"`
					EmptyDir struct {
						Medium    types.String `tfsdk:"medium" manifest:"medium"`
						SizeLimit types.String `tfsdk:"size_limit" manifest:"sizeLimit"`
					} `tfsdk:"empty_dir" manifest:"emptyDir"`
					Ephemeral struct {
						VolumeClaimTemplate struct {
							Metadata struct {
								Annotations                map[string]types.String `tfsdk:"annotations" manifest:"annotations"`
								CreationTimestamp          types.String            `tfsdk:"creation_timestamp" manifest:"creationTimestamp"`
								DeletionGracePeriodSeconds types.Int64             `tfsdk:"deletion_grace_period_seconds" manifest:"deletionGracePeriodSeconds"`
								DeletionTimestamp          types.String            `tfsdk:"deletion_timestamp" manifest:"deletionTimestamp"`
								Finalizers                 []types.String          `tfsdk:"finalizers" manifest:"finalizers"`
								GenerateName               types.String            `tfsdk:"generate_name" manifest:"generateName"`
								Generation                 types.Int64             `tfsdk:"generation" manifest:"generation"`
								Labels                     map[string]types.String `tfsdk:"labels" manifest:"labels"`
								ManagedFields              []struct {
									APIVersion  types.String `tfsdk:"api_version" manifest:"apiVersion"`
									FieldsType  types.String `tfsdk:"fields_type" manifest:"fieldsType"`
									Manager     types.String `tfsdk:"manager" manifest:"manager"`
									Operation   types.String `tfsdk:"operation" manifest:"operation"`
									Subresource types.String `tfsdk:"subresource" manifest:"subresource"`
									Time        types.String `tfsdk:"time" manifest:"time"`
								} `tfsdk:"managed_fields" manifest:"managedFields"`
								Name            types.String `tfsdk:"name" manifest:"name"`
								Namespace       types.String `tfsdk:"namespace" manifest:"namespace"`
								OwnerReferences []struct {
									APIVersion         types.String `tfsdk:"api_version" manifest:"apiVersion"`
									BlockOwnerDeletion types.Bool   `tfsdk:"block_owner_deletion" manifest:"blockOwnerDeletion"`
									Controller         types.Bool   `tfsdk:"controller" manifest:"controller"`
									Kind               types.String `tfsdk:"kind" manifest:"kind"`
									Name               types.String `tfsdk:"name" manifest:"name"`
									UID                types.String `tfsdk:"uid" manifest:"uid"`
								} `tfsdk:"owner_references" manifest:"ownerReferences"`
								ResourceVersion types.String `tfsdk:"resource_version" manifest:"resourceVersion"`
								SelfLink        types.String `tfsdk:"self_link" manifest:"selfLink"`
								UID             types.String `tfsdk:"uid" manifest:"uid"`
							} `tfsdk:"metadata" manifest:"metadata"`
							Spec struct {
								AccessModes []types.String `tfsdk:"access_modes" manifest:"accessModes"`
								DataSource  struct {
									ApiGroup types.String `tfsdk:"api_group" manifest:"apiGroup"`
									Kind     types.String `tfsdk:"kind" manifest:"kind"`
									Name     types.String `tfsdk:"name" manifest:"name"`
								} `tfsdk:"data_source" manifest:"dataSource"`
								DataSourceRef struct {
									ApiGroup  types.String `tfsdk:"api_group" manifest:"apiGroup"`
									Kind      types.String `tfsdk:"kind" manifest:"kind"`
									Name      types.String `tfsdk:"name" manifest:"name"`
									Namespace types.String `tfsdk:"namespace" manifest:"namespace"`
								} `tfsdk:"data_source_ref" manifest:"dataSourceRef"`
								Resources struct {
									Claims []struct {
										Name types.String `tfsdk:"name" manifest:"name"`
									} `tfsdk:"claims" manifest:"claims"`
									Limits   map[string]types.String `tfsdk:"limits" manifest:"limits"`
									Requests map[string]types.String `tfsdk:"requests" manifest:"requests"`
								} `tfsdk:"resources" manifest:"resources"`
								Selector struct {
									MatchExpressions []struct {
										Key      types.String   `tfsdk:"key" manifest:"key"`
										Operator types.String   `tfsdk:"operator" manifest:"operator"`
										Values   []types.String `tfsdk:"values" manifest:"values"`
									} `tfsdk:"match_expressions" manifest:"matchExpressions"`
									MatchLabels map[string]types.String `tfsdk:"match_labels" manifest:"matchLabels"`
								} `tfsdk:"selector" manifest:"selector"`
								StorageClassName types.String `tfsdk:"storage_class_name" manifest:"storageClassName"`
								VolumeMode       types.String `tfsdk:"volume_mode" manifest:"volumeMode"`
								VolumeName       types.String `tfsdk:"volume_name" manifest:"volumeName"`
							} `tfsdk:"spec" manifest:"spec"`
						} `tfsdk:"volume_claim_template" manifest:"volumeClaimTemplate"`
					} `tfsdk:"ephemeral" manifest:"ephemeral"`
					Fc struct {
						FsType     types.String   `tfsdk:"fs_type" manifest:"fsType"`
						Lun        types.Int64    `tfsdk:"lun" manifest:"lun"`
						ReadOnly   types.Bool     `tfsdk:"read_only" manifest:"readOnly"`
						TargetWwns []types.String `tfsdk:"target_wwns" manifest:"targetWwns"`
						Wwids      []types.String `tfsdk:"wwids" manifest:"wwids"`
					} `tfsdk:"fc" manifest:"fc"`
					FlexVolume struct {
						Driver    types.String            `tfsdk:"driver" manifest:"driver"`
						FsType    types.String            `tfsdk:"fs_type" manifest:"fsType"`
						Options   map[string]types.String `tfsdk:"options" manifest:"options"`
						ReadOnly  types.Bool              `tfsdk:"read_only" manifest:"readOnly"`
						SecretRef struct {
							Name types.String `tfsdk:"name" manifest:"name"`
						} `tfsdk:"secret_ref" manifest:"secretRef"`
					} `tfsdk:"flex_volume" manifest:"flexVolume"`
					Flocker struct {
						DatasetName types.String `tfsdk:"dataset_name" manifest:"datasetName"`
						DatasetUuid types.String `tfsdk:"dataset_uuid" manifest:"datasetUuid"`
					} `tfsdk:"flocker" manifest:"flocker"`
					GcePersistentDisk struct {
						FsType    types.String `tfsdk:"fs_type" manifest:"fsType"`
						Partition types.Int64  `tfsdk:"partition" manifest:"partition"`
						PdName    types.String `tfsdk:"pd_name" manifest:"pdName"`
						ReadOnly  types.Bool   `tfsdk:"read_only" manifest:"readOnly"`
					} `tfsdk:"gce_persistent_disk" manifest:"gcePersistentDisk"`
					GitRepo struct {
						Directory  types.String `tfsdk:"directory" manifest:"directory"`
						Repository types.String `tfsdk:"repository" manifest:"repository"`
						Revision   types.String `tfsdk:"revision" manifest:"revision"`
					} `tfsdk:"git_repo" manifest:"gitRepo"`
					Glusterfs struct {
						Endpoints types.String `tfsdk:"endpoints" manifest:"endpoints"`
						Path      types.String `tfsdk:"path" manifest:"path"`
						ReadOnly  types.Bool   `tfsdk:"read_only" manifest:"readOnly"`
					} `tfsdk:"glusterfs" manifest:"glusterfs"`
					HostPath struct {
						Path types.String `tfsdk:"path" manifest:"path"`
						Type types.String `tfsdk:"type" manifest:"type"`
					} `tfsdk:"host_path" manifest:"hostPath"`
					Iscsi struct {
						ChapAuthDiscovery types.Bool     `tfsdk:"chap_auth_discovery" manifest:"chapAuthDiscovery"`
						ChapAuthSession   types.Bool     `tfsdk:"chap_auth_session" manifest:"chapAuthSession"`
						FsType            types.String   `tfsdk:"fs_type" manifest:"fsType"`
						InitiatorName     types.String   `tfsdk:"initiator_name" manifest:"initiatorName"`
						Iqn               types.String   `tfsdk:"iqn" manifest:"iqn"`
						IscsiInterface    types.String   `tfsdk:"iscsi_interface" manifest:"iscsiInterface"`
						Lun               types.Int64    `tfsdk:"lun" manifest:"lun"`
						Portals           []types.String `tfsdk:"portals" manifest:"portals"`
						ReadOnly          types.Bool     `tfsdk:"read_only" manifest:"readOnly"`
						SecretRef         struct {
							Name types.String `tfsdk:"name" manifest:"name"`
						} `tfsdk:"secret_ref" manifest:"secretRef"`
						TargetPortal types.String `tfsdk:"target_portal" manifest:"targetPortal"`
					} `tfsdk:"iscsi" manifest:"iscsi"`
					Name types.String `tfsdk:"name" manifest:"name"`
					Nfs  struct {
						Path     types.String `tfsdk:"path" manifest:"path"`
						ReadOnly types.Bool   `tfsdk:"read_only" manifest:"readOnly"`
						Server   types.String `tfsdk:"server" manifest:"server"`
					} `tfsdk:"nfs" manifest:"nfs"`
					PersistentVolumeClaim struct {
						ClaimName types.String `tfsdk:"claim_name" manifest:"claimName"`
						ReadOnly  types.Bool   `tfsdk:"read_only" manifest:"readOnly"`
					} `tfsdk:"persistent_volume_claim" manifest:"persistentVolumeClaim"`
					PhotonPersistentDisk struct {
						FsType types.String `tfsdk:"fs_type" manifest:"fsType"`
						PdId   types.String `tfsdk:"pd_id" manifest:"pdId"`
					} `tfsdk:"photon_persistent_disk" manifest:"photonPersistentDisk"`
					PortworxVolume struct {
						FsType   types.String `tfsdk:"fs_type" manifest:"fsType"`
						ReadOnly types.Bool   `tfsdk:"read_only" manifest:"readOnly"`
						VolumeId types.String `tfsdk:"volume_id" manifest:"volumeId"`
					} `tfsdk:"portworx_volume" manifest:"portworxVolume"`
					Projected struct {
						DefaultMode types.Int64 `tfsdk:"default_mode" manifest:"defaultMode"`
						Sources     []struct {
							ConfigMap struct {
								Items []struct {
									Key  types.String `tfsdk:"key" manifest:"key"`
									Mode types.Int64  `tfsdk:"mode" manifest:"mode"`
									Path types.String `tfsdk:"path" manifest:"path"`
								} `tfsdk:"items" manifest:"items"`
								Name     types.String `tfsdk:"name" manifest:"name"`
								Optional types.Bool   `tfsdk:"optional" manifest:"optional"`
							} `tfsdk:"config_map" manifest:"configMap"`
							DownwardApi struct {
								Items []struct {
									FieldRef struct {
										APIVersion types.String `tfsdk:"api_version" manifest:"apiVersion"`
										FieldPath  types.String `tfsdk:"field_path" manifest:"fieldPath"`
									} `tfsdk:"field_ref" manifest:"fieldRef"`
									Mode             types.Int64  `tfsdk:"mode" manifest:"mode"`
									Path             types.String `tfsdk:"path" manifest:"path"`
									ResourceFieldRef struct {
										ContainerName types.String `tfsdk:"container_name" manifest:"containerName"`
										Divisor       types.String `tfsdk:"divisor" manifest:"divisor"`
										Resource      types.String `tfsdk:"resource" manifest:"resource"`
									} `tfsdk:"resource_field_ref" manifest:"resourceFieldRef"`
								} `tfsdk:"items" manifest:"items"`
							} `tfsdk:"downward_api" manifest:"downwardApi"`
							Secret struct {
								Items []struct {
									Key  types.String `tfsdk:"key" manifest:"key"`
									Mode types.Int64  `tfsdk:"mode" manifest:"mode"`
									Path types.String `tfsdk:"path" manifest:"path"`
								} `tfsdk:"items" manifest:"items"`
								Name     types.String `tfsdk:"name" manifest:"name"`
								Optional types.Bool   `tfsdk:"optional" manifest:"optional"`
							} `tfsdk:"secret" manifest:"secret"`
							ServiceAccountToken struct {
								Audience          types.String `tfsdk:"audience" manifest:"audience"`
								ExpirationSeconds types.Int64  `tfsdk:"expiration_seconds" manifest:"expirationSeconds"`
								Path              types.String `tfsdk:"path" manifest:"path"`
							} `tfsdk:"service_account_token" manifest:"serviceAccountToken"`
						} `tfsdk:"sources" manifest:"sources"`
					} `tfsdk:"projected" manifest:"projected"`
					Quobyte struct {
						Group    types.String `tfsdk:"group" manifest:"group"`
						ReadOnly types.Bool   `tfsdk:"read_only" manifest:"readOnly"`
						Registry types.String `tfsdk:"registry" manifest:"registry"`
						Tenant   types.String `tfsdk:"tenant" manifest:"tenant"`
						User     types.String `tfsdk:"user" manifest:"user"`
						Volume   types.String `tfsdk:"volume" manifest:"volume"`
					} `tfsdk:"quobyte" manifest:"quobyte"`
					Rbd struct {
						FsType    types.String   `tfsdk:"fs_type" manifest:"fsType"`
						Image     types.String   `tfsdk:"image" manifest:"image"`
						Keyring   types.String   `tfsdk:"keyring" manifest:"keyring"`
						Monitors  []types.String `tfsdk:"monitors" manifest:"monitors"`
						Pool      types.String   `tfsdk:"pool" manifest:"pool"`
						ReadOnly  types.Bool     `tfsdk:"read_only" manifest:"readOnly"`
						SecretRef struct {
							Name types.String `tfsdk:"name" manifest:"name"`
						} `tfsdk:"secret_ref" manifest:"secretRef"`
						User types.String `tfsdk:"user" manifest:"user"`
					} `tfsdk:"rbd" manifest:"rbd"`
					ScaleIo struct {
						FsType           types.String `tfsdk:"fs_type" manifest:"fsType"`
						Gateway          types.String `tfsdk:"gateway" manifest:"gateway"`
						ProtectionDomain types.String `tfsdk:"protection_domain" manifest:"protectionDomain"`
						ReadOnly         types.Bool   `tfsdk:"read_only" manifest:"readOnly"`
						SecretRef        struct {
							Name types.String `tfsdk:"name" manifest:"name"`
						} `tfsdk:"secret_ref" manifest:"secretRef"`
						SslEnabled  types.Bool   `tfsdk:"ssl_enabled" manifest:"sslEnabled"`
						StorageMode types.String `tfsdk:"storage_mode" manifest:"storageMode"`
						StoragePool types.String `tfsdk:"storage_pool" manifest:"storagePool"`
						System      types.String `tfsdk:"system" manifest:"system"`
						VolumeName  types.String `tfsdk:"volume_name" manifest:"volumeName"`
					} `tfsdk:"scale_io" manifest:"scaleIo"`
					Secret struct {
						DefaultMode types.Int64 `tfsdk:"default_mode" manifest:"defaultMode"`
						Items       []struct {
							Key  types.String `tfsdk:"key" manifest:"key"`
							Mode types.Int64  `tfsdk:"mode" manifest:"mode"`
							Path types.String `tfsdk:"path" manifest:"path"`
						} `tfsdk:"items" manifest:"items"`
						Optional   types.Bool   `tfsdk:"optional" manifest:"optional"`
						SecretName types.String `tfsdk:"secret_name" manifest:"secretName"`
					} `tfsdk:"secret" manifest:"secret"`
					Storageos struct {
						FsType    types.String `tfsdk:"fs_type" manifest:"fsType"`
						ReadOnly  types.Bool   `tfsdk:"read_only" manifest:"readOnly"`
						SecretRef struct {
							Name types.String `tfsdk:"name" manifest:"name"`
						} `tfsdk:"secret_ref" manifest:"secretRef"`
						VolumeName      types.String `tfsdk:"volume_name" manifest:"volumeName"`
						VolumeNamespace types.String `tfsdk:"volume_namespace" manifest:"volumeNamespace"`
					} `tfsdk:"storageos" manifest:"storageos"`
					VsphereVolume struct {
						FsType            types.String `tfsdk:"fs_type" manifest:"fsType"`
						StoragePolicyId   types.String `tfsdk:"storage_policy_id" manifest:"storagePolicyId"`
						StoragePolicyName types.String `tfsdk:"storage_policy_name" manifest:"storagePolicyName"`
						VolumePath        types.String `tfsdk:"volume_path" manifest:"volumePath"`
					} `tfsdk:"vsphere_volume" manifest:"vsphereVolume"`
				} `tfsdk:"volumes" manifest:"volumes"`
			} `tfsdk:"spec" manifest:"spec"`
		} `tfsdk:"template" manifest:"template"`
	} `tfsdk:"spec" manifest:"spec"`
}
