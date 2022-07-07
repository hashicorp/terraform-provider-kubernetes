## 2.13.0

* Add Acceptance Tests (GKE) to Github Actions

## 2.12.1 (July 6, 2022)

IMPROVEMENTS:

* Update documentation and correct some errors (#1759)

BUG FIXES:

* Fix type morphing of nested tuples that causes `Failed to morph` errors (#1756)
* Fix an issue when provider crashes intermittently in version `v2.12.0` (#1762)

## 2.12.0 (June 30, 2022)

NEW:

* Attribute `ignore_annotations` of `provider` (#746)
* Attribute `ignore_labels` of `provider` (#746)
* Attribute `condition` to `wait` block of `kubernetes_manifest` (#1595)
* Attribute `allocate_load_balancer_node_ports` of `kubernetes_service(_v1)` (#1683)
* Attribute `cluster_ips` of `kubernetes_service(_v1)` (#1683)
* Attribute `internal_traffic_policy` of `kubernetes_service(_v1)` (#1683)
* Attribute `load_balancer_class` of `kubernetes_service(_v1)` (#1683)
* Attribute `session_affinity_config` of `kubernetes_service(_v1)` (#1683)

IMPROVEMENTS:

* Update documentation and correct some errors (#1706, #1708)
* Fix security scan alerts (#1727, #1730, #1731)
* Attribute `topology_key` of `kubernetes_deployment(_v1)` marked as `Required` (#1736)

BUG FIXES:

* Fix `kubernetes_default_service_account` doesn't set the `automount_service_account_token` to `false` (#1247)
* Fix an issue when the imported `kubernetes_manifest` resource is replaced instead of getting updated (#1712)
* Fix provider crash when `image_pull_secret` of `kubernetes_service_account(_v1)` is `null`

## 2.11.0 (April 27, 2022)

NEW:

* Add a new resource `kubernetes_horizontal_pod_autoscaler_v2` (#1674)

IMPROVEMENTS:

* Add `ip_families` and `ip_family_policy` attributes to `kubernetes_service` (#1662)
* Handle `x-kubernetes-preserve-unknown-fields` type annotation from OpenAPI: changes to attributes of this type trigger whole resource recreation. (#1646)
* Upgrade terraform-plugin-mux to v0.6.0 (#1686)
* Add GitHub action for EKS acceptance tests (#1656)
* Add github action for acceptance tests using kind (#1691)

BUG FIXES:

* Fix conversion of big.Float to float64 in `kubernetes_manifest` (#1661)
* Fix identification of `int-or-string` type attributes to include 3rd party types defined by aggregated APIs (#1640)
* Fix not handling multiple `cluster_role_selectors` of `kubernetes_cluster_role(_v1)` (#1360)

## 2.10.0 (April 7, 2022)

NEW:

* Resource `kubernetes_labels` (#692)
* Resource `kubernetes_annotations` (#692)
* Resource `kubernetes_config_map_v1_data` (#723)
* Block `wait` with attribute `rollout` of `kubernetes_manifest` (#1549)
* Data source and resource attributes `app_protocol` of `kubernetes_service` (#1554)
* Attribute `container_resource` of resource `kubernetes_horizontal_pod_autoscaler_v2beta2` (#1637)

IMPROVEMENTS:

* Deprecate `wait_for` attribute in favor of `wait` block in `kubernetes_manifest` (#1549)
* Make attribute `rule` optional of `kubernetes_validating_webhook_configuration(_v1)` and `kubernetes_mutating_webhook_configuration(_v1)` (#1618, #1643)
* Update documentation and correct some errors (#1622, #1628, #1657, #1681)

BUG FIXES:

* Fix crash when multiple `match_expression` are used in `kubernetes_resource_quota` (#1561)
* Fix issue when in some circumstances changes of `seLinuxOptions.Type` doesn't reflect in the state file (#1650)
* Ignore service account volumes with `kube-api-access` prefix (#1663)

## 2.9.0 (March 17, 2022)

IMPROVEMENTS:

* Add attribute `csi` to pod spec (#1092)
* Add `kubernetes_resource` data source (#1548)
* `kubernetes_manifest` resource force the re-creation of the resource when either `apiVersion` or `kind` attributes change (#1593)
* Make attribute `http` of resource `kubernetes_ingress_v1` optional (#1613)
* Add a new attribute `seccomp_profile` to pod and container spec (#1617)
* Add additional check to resource `kubernetes_job_v1` when attributes `wait_for_completion` and `ttl_seconds_after_finished` are used together (#1619)
* Update documentation examples and correct some errors (#1597, #1611, #1612, #1626)

BUG FIXES:

* Fix logic of `wait_for_rollout` attribute of `kubernetes_deployment` (#1405)
* Fix fail when the provider cannot determine `default_secret_name` (#1634)

## 2.8.0 (February 09, 2022)

IMPROVEMENTS:

* Add mutating_webhook_configuration_v1 data source (#1423)
* Remove enabling experiment section (#1564)
* Update kubernetes dependencies (#1574)
* Update terraform-plugin-go and terraform-plugin-sdk (#1551)

BUG FIXES:

* Fix `panic: lists must only contain one type of element` errors on `kubernetes_manifest`
* Attribute `backend.service.port.name` in `kubernetes_ingress_v1` should be type String  (#1541)

## 2.7.1 (December 06, 2021)

BUG FIXES:
* Fix type-morphing of Map into Map (#1521)

## 2.7.0 (November 30, 2021)

IMPROVEMENTS:
* Add support for storage/v1
* Add support for certificates/v1
* Add support for networking/v1
* Add support for policy/v1
* Add `completion_mode` to job spec 
* Improve performance of `kubernetes_manifest` by reducing amount of API calls

BUG FIXES:
* Fix crash when container env block is empty 
* Fix invalid allowedHostPaths PodSecurityPolicy patch 
* Fix handling of "null" values on fields of `kubernetes_manifest` (#1478)

This release introduces version suffixes to the names of resources and datasources. See our [documentation page](https://registry.terraform.io/providers/hashicorp/kubernetes/latest/docs/guides/versioned-resources) for more details on this convention and the motivation behind it.   

## 2.6.1 (October 22, 2021)

BUG FIXES:
  * Fix import ID syntax in manifest import docs
  * Tolerate unknown values in "env" and "exec" provider attributes
  * Remove "beta" designation of the kubernetes_manifest from documentation

## 2.6.0 (October 19, 2021)

IMPROVEMENTS:
* kubernetes_manifest is now GA and enabled by default

BUG FIXES:
* kubernetes_manifest now correctly handles empty blocks as attribute values (#1352)
* kubernetes_manifest now correctly handles multiple CRDs with different number of schema versions (#1460)

## 2.5.1 (October 14, 2021)

IMPROVEMENTS:
* Allow setting kubernetes_job parallelism to zero (#1334)
* Add kubernetes_ingress_class resource (#1236)
* Add immutable field to kubernetes_secret (#1280)
* Add behavior field to horizontal_pod_autoscaler (#1030)
* Add proxy_url attribute to provider configuration block (#1441)

BUG FIXES:
* Always generate standard ObjectMeta for CRD types (#1418)
* Fix importing kubernetes_manifest resources (#1440)
* Fix documentation example for field_manager block (#1410)
* Fix kubernetes_job "No waiting" documentation example (#1383)
* Fix docs formatting for kubernetes_secret (#1434)

## 2.5.0 (September 14, 2021)

IMPROVEMENTS:
* Timeouts block on `kubernetes_manifest`
* `kubernetes_manifest` supports setting field_manager name and "force" mode
* `kubernetes_manifest` checks that resource exists before trying to create
* `kubernetes_manifest` supports "computed" attributtes
* `kubernetes_manifest` supports import operations

BUG FIXES:
* Fix typo in kubernetes_manifest documentation
* Document that kubernetes_manifest must be enabled in the provider block.
* Docs for ingress_class_name in kubernetes_ingress

## 2.4.1 (August 03, 2021)

HOTFIX:
* Fix kubernetes_manifest Terraform version constraint causing error on 0.12/0.13  ([#1345](https://github.com/hashicorp/terraform-provider-kubernetes/pull/1345))

## 2.4.0 (August 02, 2021)

IMPROVEMENTS:
* Add `kubernetes_manifest` resource as experimental feature 
* Upgrade Terraform SDK to v2.7.0

## 2.3.2 (June 10, 2021)

BUG FIXES:
* Revert "Filter well known labels and annotations" ([#1298](https://github.com/hashicorp/terraform-provider-kubernetes/pull/1298))

IMPROVEMENTS:
* docs/stateful_set: add import section ([#1287](https://github.com/hashicorp/terraform-provider-kubernetes/pull/1287))


## 2.3.1 (June 03, 2021)

BUG FIXES:
* `cluster_ip` for `kubernetes_service` should support value `None` ([#1291](https://github.com/hashicorp/terraform-provider-kubernetes/pull/1291))
* Remove `self_link` from metadata ([#1294](https://github.com/hashicorp/terraform-provider-kubernetes/pull/1294))
* Add missing labels to fix "`kubernetes.io/metadata.name` always in plan" ([#1293](https://github.com/hashicorp/terraform-provider-kubernetes/pull/1293))


## 2.3.0 (June 02, 2021)

BUG FIXES:
* Add missing annotations ([#1289](https://github.com/hashicorp/terraform-provider-kubernetes/pull/1289))


IMPROVEMENTS:
* Datasource: `kubernetes_secret`: add `binary_data` attribute ([#1285](https://github.com/hashicorp/terraform-provider-kubernetes/pull/1285))
* Add validations to `validating_webhook_configuration` ([#1279](https://github.com/hashicorp/terraform-provider-kubernetes/pull/1279))
* Add validations to `mutating_webhook_configuration` ([#1278](https://github.com/hashicorp/terraform-provider-kubernetes/pull/1278))
* Add validations to `storage_class` ([#1276](https://github.com/hashicorp/terraform-provider-kubernetes/pull/1276))
* Add validations to container PodSpec ([#1275](https://github.com/hashicorp/terraform-provider-kubernetes/pull/1275))
* Add validations to `service` ([#1273](https://github.com/hashicorp/terraform-provider-kubernetes/pull/1273))
* Update EKS example to use two applies ([#1260](https://github.com/hashicorp/terraform-provider-kubernetes/pull/1260))
* Resource `kubernetes_deployment`: allow changing strategy from `rolling` to `recreate` ([#1255](https://github.com/hashicorp/terraform-provider-kubernetes/pull/1255))
* Filter well known labels and annotations ([#1253](https://github.com/hashicorp/terraform-provider-kubernetes/pull/1253))
* Resource `kubernetes_resource_quota`: suppress diff for no-op changes ([#1251](https://github.com/hashicorp/terraform-provider-kubernetes/pull/1251))
* Resource `kubernetes_deployment`: allow removing volume mount ([#1246](https://github.com/hashicorp/terraform-provider-kubernetes/pull/1246))

## 2.2.0 (May 12, 2021)

IMPROVEMENTS:
* Match specific tolerations to prevent diffs (#978)
* Update all go modules (#1240)
* Docs: fix broken links (#1041)
* Docs: fix typo in getting started guide (#1262)


## 2.1.0 (April 15, 2021)

BUG FIXES:
* Fix `kubernetes_cron_job` ForceNew when modifying `job_template` (#1212)
* Fix error returned by Create CSR (#1206)
* Fix `kubernetes_pod_disruption_budget`: `100%` now is a valid value (#1107)
* Fix perpetual diff in persistent volume claimRef (#1227)

IMPROVEMENTS:
* Add `binary_data` field to `kubernetes_secret` (#1228)
* Add support for setting the persistent volume claimRef (#1020)
* Add `secret_namespace` to `volume_source` `azure_file` (#1204)
* Docs: fix grammar in Network Policy (#1210)
* Docs: `kubernetes_cron_job` add link to Kubernetes reference (#1200)

## 2.0.3 (March 17, 2021)

BUG FIXES:

* Fix resource_field_ref schema for projected_volume (#1189)
* Add diff suppression to persistent_volume and persistent_volume_claim (#1145)
* Remove error for missing kubeconfig, to allow generating it at apply time (#1142)

IMPROVEMENTS:

* Support topologySpreadConstraint in pod spec schema (#1022)
* Wait for kubernetes_ingress to be deleted (#1143)
* Improve docs for configuring the provider (#1132)
* Update docs to reflect Kubernetes service status attribute (#1148)

## 2.0.2 (February 02, 2021)

BUG FIXES:
* Read operation should set resource id to null if not found (#1136)

IMPROVEMENTS:
* Add service timeouts docs (#963)

## 2.0.1 (January 22, 2021)

BUG FIXES:
* Resources state migration should migrate empty array (#1124)

IMPROVEMENTS:
* Update docs to reflect new schema for `load_balancer_ingress` (#1123)

## 2.0.0 (January 21, 2021)

BREAKING CHANGES:
* Replace support for `KUBECONFIG` environment variable with `KUBE_CONFIG_PATH` (#1052)
* Remove `load_config_file` attribute from provider block (#1052)
* Remove default of `~/.kube/config` for `config_path` (#1052)
* Update Terraform SDK to v2 (#1027) 
* Restructure service and ingress to match K8s API (#1071)
* Normalize `automount_service_account_token` to be in line with the K8s API (#1054)
* Normalize `enable_service_links` to be in line with the K8s API (#1074)
* Normalize wait defaults across Deployment, DaemonSet, StatefulSet, Service, Ingress, and Job (#1053)
* Change resources requests and limits to TypeMap (#1065)

FEATURES:
* Add timeout argument to kubernetes_stateful_set (#1047)
* Add divisor to resource_field_ref (#1063)
* Add ingressClassName as field in Ingress manifest (#1057)

BUG FIXES:
* Fix typo in Job error message (#1048)
* Fix assertion in TestAccKubernetesPersistentVolume_hostPath_nodeAffinty (#1067)
* Fix service load balancer crash (#1070)
* Fix `cronJob.ttl_seconds_after_finished` causing requests to fail even without value specified (#929)
* Fix perpetual diff when using Pod resource with `automount_service_account_token=true` (#1085)
* Fix perpetual diff in StatefulSet when `update_strategy` is not specified (#1088)
* Fix delete/recreate when updating `init_containers` (#951)
* Fix delete/recreate of Jobs when updating mutable fields (#1074)

IMPROVEMENTS:
* Add upgrade test for daemonset (#1064)
* Add `kube_config_paths` to provider block (#1052)

## 1.13.3 (October 27, 2020)

FEATURES:

* Add support for readiness_gate on Pod spec (#811)
* Add Azure Managed disk to PV resource (#202)
* Add support for enable_service_links to the pod specification (#975)

BUG FIXES:

* Fix annotation diffs on affinity tests (#993)
* Fix api_group requirement in cluster_role_binding and role_binding (#1024)
* Fix service test leaking ELBs (#947)
* Fix annotation diffs on affinity tests (#993)
* Fix job documentation
* Fix build on macOS (#1045) and windows/386

IMPROVEMENTS:

* Update Go dependencies (#968)
* Update acceptance tests for tfproviderlint (#887)
* Refactor Typhoon test configuration to allow selection of Kubernetes version (#992)
* Update Pull Request Lifecycle docs (#1032)
* CI checks for docs website (registry migration) (#953)

## 1.13.2 (September 10, 2020)

BUG FIXES:

* Fix spurious forced replacement in empty_dir volume (#985)
* Fix reported replica count when waiting for Deployment rollout (#998)
* health_check_port_node should force replacement (#986)
* Don't force replacement StatefulSet / Deployment when affinity rule selectors change (#755)

IMPROVEMENTS:

* Wait for `kubernetes_service` to be deleted
* Updates to CONTRIBUTING.md and PULL_REQUESTS.md

## 1.13.1 (September 03, 2020)

BUG FIXES:
* Fix crash when size_limit is not present on empty_dir volume (#983)

## 1.13.0 (September 02, 2020)

FEATURES:

* Add resource `CertificateSigningRequest` (#922)
* Add resource `default_service_account` (#876)


IMPROVEMENTS:

* Allow in-place update of PVC's storage request (#957)
* Add sysctl support to pod spec (#938)
* Add ability to wait for deployment to delete (#937)
* Add support for `aggregation_rule` to `cluster_role` resource (#911)
* Add `health_check_node_port` to Service resource (#908)
* Add support for `size_limit` for `empty_dir` block (#912)
* Add support for volume mode (#939)
* Add projected volumes in pod_spec (#907)
* Add termination_message_policy to container schema (#847)

BUG FIXES:

* Recreate Storage Class on VolumeBindingMode update (#757)
* Fix url attribute in admissionregistration client_config.service block (#959)
* Fix crash when deferencing nil pointer in v1beta1.IngressRule (#967)

## 1.12.0 (July 30, 2020)

BUG FIXES:

* Fix crash in `resource_kubernetes_pod_security_policy` attribute `host_ports` (#931)

IMPROVEMENTS:

* Add `wait_for_rollout` to `kubernetes_deployment` resource (#863)
* Add `wait_for_rollout` to `kubernetes_stateful_set` resource (#605)

## 1.11.4 (July 21, 2020)

IMPROVEMENTS:

* Add resource for CSIDriver (#825)
* Add resource for Pod Security Policies (#861)
* Add data source for Pod and PVC (#786)
* Add support for CSI volume type in persistent_volume resource (#817)
* Add Kubernetes Job `wait_for_completion` functionality (#625)
* Support `optional` flag for ConfigMap mounted as volume (#823)
* Add specific error message when failing to load provider config (#780)
* Support `optional` on env valueFrom for secret key/configmap key (#824)
* Skip tests for CSIDriver if cluster version is less than 1.16
* Allow `ttl_seconds_after_finished = 0` in `kubernetes_job` resource (#849)
* Set service block to `optional` for webhook configurations (#902)

## 1.11.3 (May 20, 2020)

IMPROVEMENTS:

* Add data source for ingress (#514)
* Add data sources for namespaces (#613)

## 1.11.2 (May 06, 2020)

IMPROVEMENTS:

* Add data source for config map (#76)
* Add data source for service account (#523)
* Add resource for ValidatingWebHookConfiguration and MutatingWebhookConfiguration (#791)

BUG FIXES:
* Update Go module versions to work with Go 1.13

## 1.11.1 (February 28, 2020)

IMPROVEMENTS:

* Bump provider SDK to v1.7.0

BUG FIXES:

* Defer client initialization to improve resilience (#759)

## 1.11.0 (February 10, 2020)

IMPROVEMENTS:

 * Add `mount_options` attribute to `kubernetes_persistent_volume` and `kubernetes_storage_class`
 * Refactor client config initialization and fix in-cluster config (#679) (#497)

BUG FIXES:

 * Do not force base64 encoding for the `ca_bundle` on `kubernetes_api_service` (#679)
 * Allow 3s age gap between `service account` and `secret` [(issue)](https://github.com/hashicorp/terraform-provider-kubernetes/pull/377#issuecomment-540126765)
 * Add `load_config_file = false` to documented provider configurations
 * Add support for `startup_probe` on container spec
 * Fix (cluster-)role bindings and rules updates (#713)
 * Fix namespacing issues on kubernetes_priority_class (#680) **See [comment](https://github.com/hashicorp/terraform-provider-kubernetes/pull/682#issuecomment-576475875) on backward compatibility**
  * Documentation fixes

## 1.10.0 (November 08, 2019)

FEATURES:

* New resource: `kubernetes_pod_disruption_budget` (#644 / PR #338)
* New resource: `kubernetes_priority_class` (PR #495)

IMPROVEMENTS:

* Add `mount_propagation` attribute to container volume mount
* Add support for `.spec.service.port` to `kubernetes_api_service` (#665)
* Update `k8s.io/client-go` to v12
* Set option to cascade delete job resources (#534 / PR #635)
* Support in-cluster configuration with service accounts (PR #497)
* Parametrize all existing timeout values (PR #607)
* Enable HTTP requests/responses tracing in debug mode (PR #630)

BUG FIXES:

* Do not set default namespace for replication controller and deployment pod templates (#275)
* Updated host_alias property name to host_aliases (PR #670)
* Docs - updated all broken and commit-specific Kubernetes links to point to master branch (PR #626)
* Allow 0 for `backoff_limit` on `kubernetes_job` (PR #632)

## 1.9.0 (August 22, 2019)

FEATURES:

* New resource: `kubernetes_api_service` (PR #487)

IMPROVEMENTS:

* Add `type` attribute to volume hostPath (#358)
* Configurable delete timeout for `kubernetes_namespace` resource

BUG FIXES:

* Allow all values for deployment rolling update config (PR #587)
* Align validation of `role_binding` and `cluster_role_binding` names to Kubernetes rules (PR #583)

## 1.8.1 (July 19, 2019)

FEATURES:

* Add support for tolerations to Pod and Pod template (PR #448).

IMPROVEMENTS:

* Update getting started guide to Terraform 0.12 syntax (PR #544).

BUG FIXES:

* Align validation rules for names of Role and ClusterRole to Kubernetes (PR #551).
* Allow non-negative replicas in kubernetes_stateful_set (PR #527).
* Fix 'working_dir' attribute on Pod containers (PR #539).

## 1.8.0 (July 02, 2019)

FEATURES:

* New resources: `kubernetes_job` and `kubernetes_cron_job`

IMPROVEMENTS:

* Add `automount_service_account_token` attribute to the Pod spec (PR #261)
* Add `share_process_namespace` attribute to the Pod spec (PR #516)
* Update Terraform SDK to v0.12.3
* Enable Renovate to keep package dependencies up to date.

BUG FIXES:

* Fix waiting for Deployments to finish (PR #502)
* Adapt examples to Terraform 0.12 syntax
* Documentation updates and fixes

## 1.7.0 (May 22, 2019)

FEATURES:

* Add support of client-go credential plugins in auth (#396)
* Add kubernetes_ingress resource (closes #14) (#417)

IMPROVEMENTS:

* Add `affinity` (Pod affinity rules) attribute to Pod and PodTemplate spec
* Add support for `binary_data` to kubernetes_config_map (#400)
* Add `run_as_group` to container security context attribute (#414)
* Add `local` attribute `persistent_volume_source` docs
* Add `external_traffic_policy` to `kubernetes_service`
* Allow `max_unavailable` and `max_surge` to be 0 on `kubernetes_deployment`

BUG FIXES:

* Fix docs typo: `kubernetes_service` takes `target_port` not `targetPort` (#409)
* Fix links to timeouts documentation for terraform 0.12+ (#406)
* Link Endpoints resource into sidebar (#431)
* Add doc examples for container health probes.
* Don’t prevent use of kubernetes.io annotation keys

## 1.6.2 (April 18, 2019)

BUG FIXES:

* Fix to release metadata to register the provider as compatible with Terraform 0.12.

## 1.6.1 (April 18, 2019)

IMPROVEMENTS:

* Updated the Terraform SDK to support the upcoming Terraform version 0.12.

UPGRADE NOTES:

* On volume source blocks, the `mode` and `default_mode` attributes are now of type string
  and will produce a diff on the first run with state coming from Terraform 0.11.x and lower.
  Also, `default_mode` now defaults to 0644 when not set, in accordance with Kubernetes API docs.
  This will also produce a diff when applied against state from Terraform 0.11.x and lower
  (where it was implicitly 0). Subsequent applies should behave as expected.

## 1.6.0 (April 17, 2019)

FEATURES:

* New resource: `kubernetes_endpoints` (#167)

IMPROVEMENTS:

* Add support for importing `kubernetes_service_account` resources.
* Add validation for `strategy` attribute on `kubernetes_daemonset` and `kubernetes_deployment`
* Add `allow_volume_expansion` attribute to `kubernetes_storage_class` resource.
* Add `host_aliases` attribute to Pod spec and Pod templates.
* Add support for `dns_config` attribute on Pods and Pod templates.
* Mark `node_affinity` attribute on PV as Computed to support server populated values.
* Wait for PVs to finish deleting.
* Documentation now mentions acceptance of beta Kubernetes resources.

BUG FIXES:

* Fix detection of default token secret (#349)
* Fix unexpected diffs on `kubernetes_network_policy` when `namespace_selector` is empty (#310)
* Fix crashes on empty node_affinity / node_selector_term / match_expressions (#394)
* Make entire Pod template updatable (#384)

## 1.5.2 (February 28, 2019)

BUG FIXES:
* Fix `api_group` attribute attribute of RBAC subjects. (#331)

## 1.5.1 (February 18, 2019)

FEATURES:
* New resources: DaemonSet and ClusterRole (#229)

IMPROVEMENTS:
* Add test infrastructure for AKS and EKS (#291)
* Add `publish_not_ready_addresses` to `kubernetes_service` (#306)
* Populate `default_secret` for Service Account when multiple secrets are present (#281)

BUG FIXES:
* Declare `env` argument type correctly in Pod config (#304)
* Fix service datasource after #306 broke it (#313)
* Fix docs correcting `automount_service_account_token` location for Service Acount (#278)
* Fix docs typo (#279)

## 1.5.0 (January 14, 2019)

FEATURES:

* **New Resource:** `kubernetes_network_policy` ([#118](https://github.com/hashicorp/terraform-provider-kubernetes/issues/118))
* **New Resource:** `kubernetes_role`
* **New Resource:** `kubernetes_role_binding`
* **New Datasource:** `kubernetes_secret datasource` ([#241](https://github.com/hashicorp/terraform-provider-kubernetes/issues/241))


IMPROVEMENTS:

* `resource/kubernetes_deployment`, `resource/kubernetes_pod`, `resource/kubernetes_replication_controller`, `resource/kubernetes_stateful_set`: Add `allow_privilege_escalation` to container security contexts attributes ([#249](https://github.com/hashicorp/terraform-provider-kubernetes/issues/249))
* Add pod metadata to replication controller spec template ([#193](https://github.com/hashicorp/terraform-provider-kubernetes/issues/193))
* Add support for `volume_binding_mode` attribute in `kubernetes_storage_class`
* Add `node_affinity` attribute to persistent volumes.
* Add support for `local` type persistent volumes.
* Upgrade to Go 1.11 + Go modules

BUG FIXES:

* `resource/kubernetes_stateful_set`: Fix updates of stateful set images ([#252](https://github.com/hashicorp/terraform-provider-kubernetes/issues/252))

## 1.4.0 (November 29, 2018)

FEATURES:

* **New Resource:** `kubernetes_stateful_set` ([#100](https://github.com/hashicorp/terraform-provider-kubernetes/issues/100))

IMPROVEMENTS:

* `resource/kubernetes_storage_class`: Add ReclaimPolicy attribute
* `resource/kubernetes_service_account`: Allow automount service account token

BUG FIXES:

* Fix waiting for Deployment rollout status ([#210](https://github.com/hashicorp/terraform-provider-kubernetes/issues/210))

## 1.3.0 (October 23, 2018)

FEATURES:

* **New Resource:** `kubernetes_cluster_role_binding` ([#73](https://github.com/hashicorp/terraform-provider-kubernetes/issues/73))
* **New Resource:** `kubernetes_deployment` ([#101](https://github.com/hashicorp/terraform-provider-kubernetes/issues/101))

IMPROVEMENTS:

* Update Kubernetes client library to 1.10 ([#162](https://github.com/hashicorp/terraform-provider-kubernetes/issues/162))
* Add support for `env_from` on container definitions ([#82](https://github.com/hashicorp/terraform-provider-kubernetes/issues/82))

## 1.2.0 (August 15, 2018)

IMPROVEMENTS:

* resource/kubernetes_pod: Add timeout to pod resource create and delete ([#151](https://github.com/hashicorp/terraform-provider-kubernetes/issues/151))
* resource/kubernetes_pod: Add support for init containers ([#156](https://github.com/hashicorp/terraform-provider-kubernetes/issues/156))

BUG FIXES:

* name label: All name labels will now allow DNS1123 subdomain format ex: `my.label123` ([#152](https://github.com/hashicorp/terraform-provider-kubernetes/issues/152))
* resource/kubernetes_service: Switch targetPort to string ([#154](https://github.com/hashicorp/terraform-provider-kubernetes/issues/154))
* data/kubernetes_service: Switch targetPort to string ([#159](https://github.com/hashicorp/terraform-provider-kubernetes/issues/159))
* resource/kubernetes_pod: env var value change forces new pod ([#155](https://github.com/hashicorp/terraform-provider-kubernetes/issues/155))
* Fix example in docs for an image pull secret ([#165](https://github.com/hashicorp/terraform-provider-kubernetes/issues/165))

## 1.1.0 (March 23, 2018)

NOTES:

* provider: Client library updated to support Kubernetes `1.7`

IMPROVEMENTS:

* resource/kubernetes_persistent_volume_claim: Improve event log polling for warnings ([#125](https://github.com/hashicorp/terraform-provider-kubernetes/issues/125))
* resource/kubernetes_persistent_volume: Add support for `storage_class_name` ([#111](https://github.com/hashicorp/terraform-provider-kubernetes/issues/111))

BUG FIXES:

* resource/kubernetes_secret: Prevent binary data corruption ([#103](https://github.com/hashicorp/terraform-provider-kubernetes/issues/103))
* resource/kubernetes_persistent_volume: Update `persistent_volume_reclaim_policy` correctly ([#111](https://github.com/hashicorp/terraform-provider-kubernetes/issues/111))
* resource/kubernetes_service: Update external_ips correctly on K8S 1.8+ ([#127](https://github.com/hashicorp/terraform-provider-kubernetes/issues/127))
* resource/kubernetes_*: Fix adding labels/annotations to resources when those were empty ([#116](https://github.com/hashicorp/terraform-provider-kubernetes/issues/116))
* resource/kubernetes_*: Treat non-string label values as invalid ([#135](https://github.com/hashicorp/terraform-provider-kubernetes/issues/135))
* resource/kubernetes_config_map: Fix adding `data` when it was empty ([#116](https://github.com/hashicorp/terraform-provider-kubernetes/issues/116))
* resource/kubernetes_secret: Fix adding `data` when it was empty ([#116](https://github.com/hashicorp/terraform-provider-kubernetes/issues/116))
* resource/kubernetes_limit_range: Avoid spurious diff when spec is empty ([#132](https://github.com/hashicorp/terraform-provider-kubernetes/issues/132))
* resource/kubernetes_persistent_volume: Use correct operation when updating `persistent_volume_source` (`1.8`) ([#133](https://github.com/hashicorp/terraform-provider-kubernetes/issues/133))
* resource/kubernetes_persistent_volume: Mark persistent_volume_source as ForceNew on `1.9+` ([#139](https://github.com/hashicorp/terraform-provider-kubernetes/issues/139))
* resource/kubernetes_pod: Bump deletion timeout to 5 mins ([#136](https://github.com/hashicorp/terraform-provider-kubernetes/issues/136))

## 1.0.1 (November 13, 2017)

BUG FIXES:

* resource/pod: Avoid crash in reading `spec.container.security_context` `capability` ([#53](https://github.com/hashicorp/terraform-provider-kubernetes/issues/53))
* resource/replication_controller: Avoid crash in reading `template.container.security_context` `capability` ([#53](https://github.com/hashicorp/terraform-provider-kubernetes/issues/53))
* resource/service: Make spec.port.target_port optional ([#69](https://github.com/hashicorp/terraform-provider-kubernetes/issues/69))
* resource/pod: Fix `mode` conversion in `config_map` volume items ([#83](https://github.com/hashicorp/terraform-provider-kubernetes/issues/83))
* resource/replication_controller: Fix `mode` conversion in `config_map` volume items ([#83](https://github.com/hashicorp/terraform-provider-kubernetes/issues/83))

## 1.0.0 (August 18, 2017)

IMPROVEMENTS:

* resource/kubernetes_pod: Add support for `default_mode`, `items` and `optional` in Secret Volume ([#44](https://github.com/hashicorp/terraform-provider-kubernetes/issues/44))
* resource/kubernetes_replication_controller: Add support for `default_mode`, `items` and `optional` in Secret Volume ([#44](https://github.com/hashicorp/terraform-provider-kubernetes/issues/44))

BUG FIXES:

* resource/kubernetes_pod: Respect previously ignored `node_selectors` field ([#42](https://github.com/hashicorp/terraform-provider-kubernetes/issues/42))
* resource/kubernetes_pod: Represent update-ability of spec correctly ([#49](https://github.com/hashicorp/terraform-provider-kubernetes/issues/49))
* resource/kubernetes_replication_controller: Respect previously ignored `node_selectors` field ([#42](https://github.com/hashicorp/terraform-provider-kubernetes/issues/42))
* all namespaced resources: Avoid crash when importing invalid ID ([#46](https://github.com/hashicorp/terraform-provider-kubernetes/issues/46))
* meta: Treat internal k8s annotations as invalid #50

## 0.1.2 (August 04, 2017)

FEATURES:

* **New Resource:** `kubernetes_storage_class` ([#22](https://github.com/hashicorp/terraform-provider-kubernetes/issues/22))
* **New Data Source:** `kubernetes_service` ([#23](https://github.com/hashicorp/terraform-provider-kubernetes/issues/23))
* **New Data Source:** `kubernetes_storage_class` ([#33](https://github.com/hashicorp/terraform-provider-kubernetes/issues/33))

IMPROVEMENTS: 

* provider: Add support of token in auth ([#35](https://github.com/hashicorp/terraform-provider-kubernetes/issues/35))
* provider: Add switch to disable loading file config (`load_config_file`) ([#36](https://github.com/hashicorp/terraform-provider-kubernetes/issues/36))

BUG FIXES:

* resource/kubernetes_service: Make port field optional ([#27](https://github.com/hashicorp/terraform-provider-kubernetes/issues/27))
* all resources: Escape '/' in JSON Patch path correctly ([#40](https://github.com/hashicorp/terraform-provider-kubernetes/issues/40))

## 0.1.1 (July 05, 2017)

FEATURES:

* **New Resource:** `kubernetes_replication_controller` ([#9](https://github.com/hashicorp/terraform-provider-kubernetes/issues/9))
* **New Resource:** `kubernetes_service_account` ([#17](https://github.com/hashicorp/terraform-provider-kubernetes/issues/17))

IMPROVEMENTS:

* resource/kubernetes_service: Wait for LoadBalancer ingress ([#12](https://github.com/hashicorp/terraform-provider-kubernetes/issues/12))
* resource/persistent_volume_claim: Expose last warnings from the eventlog ([#16](https://github.com/hashicorp/terraform-provider-kubernetes/issues/16))
* resource/pod: Expose last warnings from the eventlog ([#16](https://github.com/hashicorp/terraform-provider-kubernetes/issues/16))
* resource/service: Expose last warnings from the eventlog ([#16](https://github.com/hashicorp/terraform-provider-kubernetes/issues/16))

BUG FIXES:

* Register auth plugins (gcp, oidc) automatically ([#6](https://github.com/hashicorp/terraform-provider-kubernetes/issues/6))
* resource/pod: Fix a crash caused by wrong field name (config map volume source) ([#19](https://github.com/hashicorp/terraform-provider-kubernetes/issues/19))
* resource/pod: Add validation for `default_mode` (mode bits) ([#19](https://github.com/hashicorp/terraform-provider-kubernetes/issues/19))

## 0.1.0 (June 20, 2017)

FEATURES:

* **New Resource:** `kubernetes_pod` [[#13571](https://github.com/hashicorp/terraform-provider-kubernetes/issues/13571)](https://github.com/hashicorp/terraform/pull/13571)
