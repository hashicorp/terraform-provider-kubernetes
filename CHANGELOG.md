## 1.8.1 (July 19, 2019)

FEATURES:

* Add support for tolerations to Pod and Pod template.

IMPROVEMENTS:

* Update getting started guide to Terraform 0.12 syntax.

BUG FIXES:

* Align validation rules for names of Role and ClusterRole to Kubernetes.
* Allow non-negative replicas in kubernetes_stateful_set.
* Fix 'working_dir' attribute on Pod containers.

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

* **New Resource:** `kubernetes_network_policy` ([#118](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/118))
* **New Resource:** `kubernetes_role`
* **New Resource:** `kubernetes_role_binding`
* **New Datasource:** `kubernetes_secret datasource` ([#241](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/241))


IMPROVEMENTS:

* `resource/kubernetes_deployment`, `resource/kubernetes_pod`, `resource/kubernetes_replication_controller`, `resource/kubernetes_stateful_set`: Add `allow_privilege_escalation` to container security contexts attributes ([#249](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/249))
* Add pod metadata to replication controller spec template ([#193](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/193))
* Add support for `volume_binding_mode` attribute in `kubernetes_storage_class`
* Add `node_affinity` attribute to persistent volumes.
* Add support for `local` type persistent volumes.
* Upgrade to Go 1.11 + Go modules

BUG FIXES:

* `resource/kubernetes_stateful_set`: Fix updates of stateful set images ([#252](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/252))

## 1.4.0 (November 29, 2018)

FEATURES:

* **New Resource:** `kubernetes_stateful_set` ([#100](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/100))

IMPROVEMENTS:

* `resource/kubernetes_storage_class`: Add ReclaimPolicy attribute
* `resource/kubernetes_service_account`: Allow automount service account token

BUG FIXES:

* Fix waiting for Deployment rollout status ([#210](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/210))

## 1.3.0 (October 23, 2018)

FEATURES:

* **New Resource:** `kubernetes_cluster_role_binding` ([#73](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/73))
* **New Resource:** `kubernetes_deployment` ([#101](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/101))

IMPROVEMENTS:

* Update Kubernetes client library to 1.10 ([#162](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/162))
* Add support for `env_from` on container definitions ([#82](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/82))

## 1.2.0 (August 15, 2018)

IMPROVEMENTS:

* resource/kubernetes_pod: Add timeout to pod resource create and delete ([#151](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/151))
* resource/kubernetes_pod: Add support for init containers ([#156](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/156))

BUG FIXES:

* name label: All name labels will now allow DNS1123 subdomain format ex: `my.label123` ([#152](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/152))
* resource/kubernetes_service: Switch targetPort to string ([#154](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/154))
* data/kubernetes_service: Switch targetPort to string ([#159](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/159))
* resource/kubernetes_pod: env var value change forces new pod ([#155](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/155))
* Fix example in docs for an image pull secret ([#165](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/165))

## 1.1.0 (March 23, 2018)

NOTES:

* provider: Client library updated to support Kubernetes `1.7`

IMPROVEMENTS:

* resource/kubernetes_persistent_volume_claim: Improve event log polling for warnings ([#125](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/125))
* resource/kubernetes_persistent_volume: Add support for `storage_class_name` ([#111](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/111))

BUG FIXES:

* resource/kubernetes_secret: Prevent binary data corruption ([#103](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/103))
* resource/kubernetes_persistent_volume: Update `persistent_volume_reclaim_policy` correctly ([#111](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/111))
* resource/kubernetes_service: Update external_ips correctly on K8S 1.8+ ([#127](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/127))
* resource/kubernetes_*: Fix adding labels/annotations to resources when those were empty ([#116](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/116))
* resource/kubernetes_*: Treat non-string label values as invalid ([#135](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/135))
* resource/kubernetes_config_map: Fix adding `data` when it was empty ([#116](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/116))
* resource/kubernetes_secret: Fix adding `data` when it was empty ([#116](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/116))
* resource/kubernetes_limit_range: Avoid spurious diff when spec is empty ([#132](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/132))
* resource/kubernetes_persistent_volume: Use correct operation when updating `persistent_volume_source` (`1.8`) ([#133](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/133))
* resource/kubernetes_persistent_volume: Mark persistent_volume_source as ForceNew on `1.9+` ([#139](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/139))
* resource/kubernetes_pod: Bump deletion timeout to 5 mins ([#136](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/136))

## 1.0.1 (November 13, 2017)

BUG FIXES:

* resource/pod: Avoid crash in reading `spec.container.security_context` `capability` ([#53](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/53))
* resource/replication_controller: Avoid crash in reading `template.container.security_context` `capability` ([#53](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/53))
* resource/service: Make spec.port.target_port optional ([#69](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/69))
* resource/pod: Fix `mode` conversion in `config_map` volume items ([#83](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/83))
* resource/replication_controller: Fix `mode` conversion in `config_map` volume items ([#83](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/83))

## 1.0.0 (August 18, 2017)

IMPROVEMENTS:

* resource/kubernetes_pod: Add support for `default_mode`, `items` and `optional` in Secret Volume ([#44](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/44))
* resource/kubernetes_replication_controller: Add support for `default_mode`, `items` and `optional` in Secret Volume ([#44](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/44))

BUG FIXES:

* resource/kubernetes_pod: Respect previously ignored `node_selectors` field ([#42](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/42))
* resource/kubernetes_pod: Represent update-ability of spec correctly ([#49](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/49))
* resource/kubernetes_replication_controller: Respect previously ignored `node_selectors` field ([#42](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/42))
* all namespaced resources: Avoid crash when importing invalid ID ([#46](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/46))
* meta: Treat internal k8s annotations as invalid #50

## 0.1.2 (August 04, 2017)

FEATURES:

* **New Resource:** `kubernetes_storage_class` ([#22](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/22))
* **New Data Source:** `kubernetes_service` ([#23](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/23))
* **New Data Source:** `kubernetes_storage_class` ([#33](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/33))

IMPROVEMENTS: 

* provider: Add support of token in auth ([#35](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/35))
* provider: Add switch to disable loading file config (`load_config_file`) ([#36](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/36))

BUG FIXES:

* resource/kubernetes_service: Make port field optional ([#27](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/27))
* all resources: Escape '/' in JSON Patch path correctly ([#40](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/40))

## 0.1.1 (July 05, 2017)

FEATURES:

* **New Resource:** `kubernetes_replication_controller` ([#9](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/9))
* **New Resource:** `kubernetes_service_account` ([#17](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/17))

IMPROVEMENTS:

* resource/kubernetes_service: Wait for LoadBalancer ingress ([#12](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/12))
* resource/persistent_volume_claim: Expose last warnings from the eventlog ([#16](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/16))
* resource/pod: Expose last warnings from the eventlog ([#16](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/16))
* resource/service: Expose last warnings from the eventlog ([#16](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/16))

BUG FIXES:

* Register auth plugins (gcp, oidc) automatically ([#6](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/6))
* resource/pod: Fix a crash caused by wrong field name (config map volume source) ([#19](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/19))
* resource/pod: Add validation for `default_mode` (mode bits) ([#19](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/19))

## 0.1.0 (June 20, 2017)

FEATURES:

* **New Resource:** `kubernetes_pod` [[#13571](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/13571)](https://github.com/hashicorp/terraform/pull/13571)
