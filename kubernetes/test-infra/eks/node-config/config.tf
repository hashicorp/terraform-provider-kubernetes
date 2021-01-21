data "aws_eks_cluster_auth" "cluster" {
  name = var.cluster_name
}

data "aws_eks_cluster" "cluster" {
  name = var.cluster_name
}

provider "kubernetes" {
  host                   = var.cluster_endpoint
  token                  = data.aws_eks_cluster_auth.cluster.token
  load_config_file       = false
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.cluster.certificate_authority.0.data)
  exec {
    api_version = "client.authentication.k8s.io/v1alpha1"
    args        = ["eks", "get-token", "--cluster-name", var.cluster_name]
    command     = "aws"
  }
}

locals {
  mapped_role_format = <<MAPPEDROLE
- rolearn: %s
  username: system:node:{{EC2PrivateDNSName}}
  groups:
    - system:bootstrappers
    - system:nodes
MAPPEDROLE

}

resource "local_file" "cluster_ca" {
  content = base64decode(var.cluster_ca)
  filename = "${path.root}/cluster_ca"
}

# This allows the kubeconfig file to be refreshed during every Terraform apply.
# Used for local testing.
resource "null_resource" "generate-kubeconfig" {
  provisioner "local-exec" {
    command = "aws eks update-kubeconfig --name ${var.cluster_name} --kubeconfig ${path.root}/kubeconfig"
  }
  triggers = {
    always_run = timestamp()
  }
}

resource "kubernetes_config_map" "name" {
  depends_on = [var.cluster_name]
  metadata {
    name      = "aws-auth"
    namespace = "kube-system"
  }

  data = {
    mapRoles = join(
      "\n",
      formatlist(local.mapped_role_format, var.k8s_node_role_arn),
    )
  }
}

# This module installs the AWS LoadBalancer Controller
# https://docs.aws.amazon.com/eks/latest/userguide/alb-ingress.html

data "tls_certificate" "testacc" {
  url = var.cluster_oidc_issuer_url
}

resource "aws_iam_openid_connect_provider" "testacc" {
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = [data.tls_certificate.testacc.certificates[0].sha1_fingerprint]
  url             = var.cluster_oidc_issuer_url
}

data "aws_iam_policy_document" "testacc" {
  statement {
    actions = ["sts:AssumeRoleWithWebIdentity"]
    effect  = "Allow"

    condition {
      test     = "StringEquals"
      variable = "${replace(aws_iam_openid_connect_provider.testacc.url, "https://", "")}:sub"
      values   = ["system:serviceaccount:kube-system:aws-node", "system:serviceaccount:kube-system:aws-load-balancer-controller"]
    }

    principals {
      identifiers = [aws_iam_openid_connect_provider.testacc.arn]
      type        = "Federated"
    }
  }
}

resource "aws_iam_role" "alb" {
  assume_role_policy = data.aws_iam_policy_document.testacc.json
  name               = "AWSLoadBalancerControllerIAMPolicy-${var.cluster_name}"
}

resource "aws_iam_policy" "alb" {
  name   = "AWSLoadBalancerControllerIAMPolicy-${var.cluster_name}"
  policy = file("${path.module}/iam_policy.json")
}

resource "aws_iam_role_policy_attachment" "alb" {
  policy_arn = aws_iam_policy.alb.arn
  role       = aws_iam_role.alb.name
}

resource "kubernetes_service_account" "alb" {
  depends_on = [var.cluster_name]
  metadata {
    name      = "aws-load-balancer-controller"
    namespace = "kube-system"
    annotations = {
      "eks.amazonaws.com/role-arn": aws_iam_role.alb.arn
    }
    labels = {
      "app.kubernetes.io/component": "controller"
      "app.kubernetes.io/name": "aws-load-balancer-controller"
    }
  }
}

# TODO: use the kubernetes provider to install as much of this as possible
resource "null_resource" "install-cert-manager-crds" {
  depends_on = [null_resource.generate-kubeconfig]
  provisioner "local-exec" {
    command = "kubectl --kubeconfig=${path.root}/kubeconfig apply --validate=false -f ${path.module}/cert-manager-crds.yaml"
  }
  triggers = {
    on_cluster_create = var.cluster_name
  }
}

resource "null_resource" "install-cert-manager-crs" {
  depends_on = [null_resource.install-cert-manager-crds]
  provisioner "local-exec" {
    command = "kubectl --kubeconfig=${path.root}/kubeconfig apply --validate=false -f ${path.module}/cert-manager-crs.yaml"
  }
  triggers = {
    on_cluster_create = var.cluster_name
  }
}


resource "null_resource" "install-controller-deps" {
  depends_on = [null_resource.install-cert-manager-crs]
  provisioner "local-exec" {
    command = "kubectl --kubeconfig=${path.root}/kubeconfig apply -f ${path.module}/aws_controller_deps.yaml"
  }
  triggers = {
    on_cluster_create = var.cluster_name
  }
}

resource "kubernetes_deployment" "aws-lb-controller" {
  depends_on = [null_resource.install-controller-deps, var.cluster_name]
  metadata {
    name      = "aws-load-balancer-controller"
    namespace = "kube-system"
    labels = {
      "app.kubernetes.io/component": "controller"
      "app.kubernetes.io/name": "aws-load-balancer-controller"
    }
  }
  spec {
   selector {
     match_labels = {
       "app.kubernetes.io/component": "controller"
       "app.kubernetes.io/name": "aws-load-balancer-controller"
     }
   }
    template {
      metadata {
        labels = {
          "app.kubernetes.io/component": "controller"
          "app.kubernetes.io/name": "aws-load-balancer-controller"
        }
      }
      spec {
        automount_service_account_token = true
        container {
          image = "amazon/aws-alb-ingress-controller:v2.1.0"
          name = "controller"
          args = [
            "--cluster-name=${var.cluster_name}",
            "--ingress-class=alb"]
          liveness_probe {
            failure_threshold = 2
            http_get {
              path   = "/healthz"
              port   = 61779
              scheme = "HTTP"
            }
            initial_delay_seconds = 30
            timeout_seconds       = 10
          }
          resources {
            limits {
              cpu    = "200m"
              memory = "500Mi"
            }
            requests {
              cpu    = "100m"
              memory = "200Mi"
            }
          }
          volume_mount {
            name       = "cert"
            mount_path = "/tmp/k8s-webhook-server/serving-certs"
            read_only  = true
          }
        }
        volume {
          name = "cert"
          secret {
            secret_name = "aws-load-balancer-webhook-tls"
          }
        }
        security_context {
          fs_group = 1337
        }
        service_account_name = "aws-load-balancer-controller"
      }
    }
  }
}
