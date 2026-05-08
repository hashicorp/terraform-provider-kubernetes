# Copyright IBM Corp. 2017, 2026
# SPDX-License-Identifier: MPL-2.0

output "config_manifest" {
  value = kubernetes_manifest.oidc_conf.object
}
