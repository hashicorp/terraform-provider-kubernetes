## 0.1.1 (Unreleased)

FEATURES:

* **New Resource:** `kubernetes_replication_controller` [GH-9]

IMPROVEMENTS:

* resource/kubernetes_service: Wait for LoadBalancer ingress [GH-12]
* resource/persistent_volume_claim: Expose last warnings from the eventlog [GH-16]
* resource/pod: Expose last warnings from the eventlog [GH-16]
* resource/service: Expose last warnings from the eventlog [GH-16]

BUG FIXES:

* Register auth plugins (gcp, oidc) automatically [GH-6]
* resource/pod: Fix a crash caused by wrong field name (config map volume source) [GH-19]
* resource/pod: Add validation for `default_mode` (mode bits) [GH-19]

## 0.1.0 (June 20, 2017)

FEATURES:

* **New Resource:** `kubernetes_pod` [[#13571](https://github.com/terraform-providers/terraform-provider-kubernetes/issues/13571)](https://github.com/hashicorp/terraform/pull/13571)
