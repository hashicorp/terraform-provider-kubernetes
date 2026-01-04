# Copyright IBM Corp. 2017, 2025
# SPDX-License-Identifier: MPL-2.0

output "kubeconfig" {
  value = abspath("${path.root}/${local_file.kubeconfig.filename}")
}
