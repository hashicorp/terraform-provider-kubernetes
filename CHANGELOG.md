## 1.1.0 (Unreleased)

NOTES:

* provider: Client library updated to support Kubernetes `1.7`

IMPROVEMENTS:

* resource/kubernetes_persistent_volume_claim: Improve event log polling for warnings [GH-125]
* resource/kubernetes_persistent_volume: Add support for `storage_class_name` [GH-111]

BUG FIXES:

* resource/kubernetes_secret: Prevent binary data corruption [GH-103]
* resource/kubernetes_persistent_volume: Update `persistent_volume_reclaim_policy` correctly [GH-111]
* resource/kubernetes_service: Update external_ips correctly on K8S 1.8+ [GH-127]
* resource/kubernetes_*: Fix adding labels/annotations to resources when those were empty [GH-116]
* resource/kubernetes_*: Treat non-string label values as invalid [GH-135]
* resource/kubernetes_config_map: Fix adding `data` when it was empty [GH-116]
* resource/kubernetes_secret: Fix adding `data` when it was empty [GH-116]
* resource/kubernetes_limit_range: Avoid spurious diff when spec is empty [GH-132]
* resource/kubernetes_persistent_volume: Use correct operation when updating `persistent_volume_source` (`1.8`) [GH-133]
* resource/kubernetes_pod: Bump deletion timeout to 5 mins [GH-136]

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
