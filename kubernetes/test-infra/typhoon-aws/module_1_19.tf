# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

locals {
  # This local gets a value of 1 when the 'kubernetes_version' input variable requests a 1.19.x version, otherwise it is 0.
  # It's used to enable the module and resources specific to 1.19.x as a workaround for not being able 
  # to interpolate variables in the 'source' attribute of a module block.
  #
  enabled_1_19 = length(regexall("v?1.19.?[0-9]{0,2}", var.kubernetes_version))
}

# This module builds a 1.19.x Typhoon cluster. It is mutually exlusive to other modules of different versions.
#
module "typhoon-acc-1_19" {
  count  = local.enabled_1_19
  source = "git::https://github.com/poseidon/typhoon//aws/flatcar-linux/kubernetes?ref=v1.19.4"

  cluster_name = var.cluster_name
  dns_zone     = var.base_domain
  dns_zone_id  = data.aws_route53_zone.typhoon-acc.zone_id

  # node configuration
  ssh_authorized_key = tls_private_key.typhoon-acc.public_key_openssh

  worker_count     = var.worker_count
  controller_count = var.controller_count
  worker_type      = var.controller_type
  controller_type  = var.worker_type
}

resource "local_file" "typhoon-acc-1_19" {
  count    = local.enabled_1_19
  content  = module.typhoon-acc-1_19[0].kubeconfig-admin
  filename = local.kubeconfig_path
}
