## 0.1.2 (Unreleased)

FEATURES:

* **New Resource:** `kubernetes_storage_class` [GH-22]

BUG FIXES:

* resource/kubernetes_service: Make port field optional [GH-27]

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
