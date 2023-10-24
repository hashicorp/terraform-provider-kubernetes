# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

provider "kubernetes" {
  host  = "https://${data.google_container_cluster.upstream.endpoint}"
  token = data.google_client_config.provider.access_token
  cluster_ca_certificate = base64decode(
    data.google_container_cluster.upstream.master_auth[0].cluster_ca_certificate,
  )
}

import {
  // The name of this resource is hardcoded by GKE as described in:
  // https://cloud.google.com/kubernetes-engine/docs/how-to/oidc#configuring_on_a_cluster
  //
  id = "apiVersion=authentication.gke.io/v2alpha1,kind=ClientConfig,namespace=kube-public,name=default"
  to = kubernetes_manifest.oidc_conf
}

resource "kubernetes_manifest" "oidc_conf" {
  manifest = {
    apiVersion = "authentication.gke.io/v2alpha1"
    kind       = "ClientConfig"
    metadata = {
      name      = "default"
      namespace = "kube-public"
    }
    spec = {
      authentication = [
        {
          name = data.google_container_cluster.upstream.name
          oidc = {
            clientID                 = var.oidc_audience
            issuerURI                = var.odic_issuer_uri
            userClaim                = var.oidc_user_claim
            groupsClaim              = var.oidc_group_claim
            certificateAuthorityData = var.TFE_CA_cert
          }
        }
      ]
    }
  }
}

resource "kubernetes_cluster_role_binding_v1" "oidc_role" {
  metadata {
    name = "odic-identity"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = var.rbac_group_cluster_role
  }

  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Group"
    name      = var.rbac_oidc_group_name
  }
}
