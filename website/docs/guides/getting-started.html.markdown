---
layout: "kubernetes"
page_title: "Kubernetes: Getting Started with Kubernetes provider"
description: |-
  This guide focuses on configuring authentication to your existing Kubernetes
  cluster so that resources can be managed using the Kubernetes provider for Terraform.
---

# Getting Started with Kubernetes provider

## Kubernetes

-> Visit the [Manage Kubernetes Resources via Terraform](https://learn.hashicorp.com/tutorials/terraform/kubernetes-provider?in=terraform/kubernetes&utm_source=WEBSITE&utm_medium=WEB_IO&utm_offer=ARTICLE_PAGE&utm_content=DOCS) Learn tutorial for an interactive getting
started experience.

[Kubernetes](https://kubernetes.io/) (K8S) is an open-source workload scheduler with focus on containerized applications.

There are at least 2 steps involved in scheduling your first container
on a Kubernetes cluster. You need the Kubernetes cluster with all its components
running _somewhere_ and then define the Kubernetes resources, such as Deployments, Services, etc.

This guide focuses mainly on the latter part and expects you to have
a properly configured & running Kubernetes cluster.

## Why Terraform

While you could use `kubectl` or similar CLI-based tools mapped to API calls
to manage all Kubernetes resources described in YAML files,
orchestration with Terraform presents a few benefits.

- Use the same [configuration language](/docs/configuration/syntax.html)
    to provision the Kubernetes infrastructure and to deploy applications into it.
- drift detection - `terraform plan` will always present you the difference
    between reality at a given time and config you intend to apply.
- full lifecycle management - Terraform doesn't just initially create resources,
    but offers a single command for creation, update, and deletion of tracked
    resources without needing to inspect the API to identify those resources.
- synchronous feedback - While asynchronous behaviour is often useful,
    sometimes it's counter-productive as the job of identifying operation result
    (failures or details of created resource) is left to the user. e.g. you don't
    have IP/hostname of load balancer until it has finished provisioning,
    hence you can't create any DNS record pointing to it.
- [graph of relationships](https://www.terraform.io/docs/internals/graph.html) -
    Terraform understands relationships between resources which may help
    in scheduling - e.g. if a Persistent Volume Claim claims space from
    a particular Persistent Volume Terraform won't even attempt to create
    the PVC if creation of the PV has failed.

## Provider Setup

The provider needs to be configured with the proper credentials before it can be used. The simplest configuration is to specify the kubeconfig path:

```hcl
provider "kubernetes" {
  config_path = "~/.kube/config"
}
```

Another configuration option is to **statically** define TLS certificate credentials:

```hcl
provider "kubernetes" {
  host = "https://104.196.242.174"

  client_certificate     = "${file("~/.kube/client-cert.pem")}"
  client_key             = "${file("~/.kube/client-key.pem")}"
  cluster_ca_certificate = "${file("~/.kube/cluster-ca-cert.pem")}"
}
```

Static TLS certficate credentials are present in Azure AKS clusters by default, and can be used with the [azurerm_kubernetes_cluster](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/data-sources/kubernetes_cluster) data source as shown below. This will automatically read the certficate information from the AKS cluster and pass it to the Kubernetes provider.

```hcl
data "azurerm_kubernetes_cluster" "example" {
  name                = "myakscluster"
  resource_group_name = "my-example-resource-group"
}

provider "kubernetes" {
  host                   = "${data.azurerm_kubernetes_cluster.example.kube_config.0.host}"
  client_certificate     = "${base64decode(data.azurerm_kubernetes_cluster.example.kube_config.0.client_certificate)}"
  client_key             = "${base64decode(data.azurerm_kubernetes_cluster.example.kube_config.0.client_key)}"
  cluster_ca_certificate = "${base64decode(data.azurerm_kubernetes_cluster.example.kube_config.0.cluster_ca_certificate)}"
}
```

Another option is to use an oauth token, such as this example from a GKE cluster. The [google_client_config](https://registry.terraform.io/providers/hashicorp/google/latest/docs/data-sources/client_config) data source fetches a token from the Google Authorization server, which expires in 1 hour by default.

```hcl
data "google_client_config" "default" {}
data "google_container_cluster" "my_cluster" {
  name     = "my-cluster"
  location = "us-east1-a"
}

provider "kubernetes" {
  host                   = "https://${data.google_container_cluster.my_cluster.endpoint}"
  token                  = data.google_client_config.default.access_token
  cluster_ca_certificate = base64decode(data.google_container_cluster.my_cluster.master_auth[0].cluster_ca_certificate)
}
```

For short-lived authentication tokens, like those found in EKS, which [expire in 15 minutes](https://aws.github.io/aws-eks-best-practices/security/docs/iam#controlling-access-to-eks-clusters), an exec-based credential plugin can be used to ensure the token is always up to date:

```hcl
data "aws_eks_cluster" "example" {
  name = "example"
}
data "aws_eks_cluster_auth" "example" {
  name = "example"
}
provider "kubernetes" {
  host                   = data.aws_eks_cluster.example.endpoint
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.example.certificate_authority[0].data)
  exec {
    api_version = "client.authentication.k8s.io/v1beta1"
    args        = ["eks", "get-token", "--cluster-name", var.cluster_name]
    command     = "aws"
  }
}
```

## Creating your first Kubernetes resources

Once the provider is configured, you can apply the Kubernetes resources defined in your Terraform config file. The following is an example Terraform config file containing a few Kubernetes resources. We'll use [minikube](https://minikube.sigs.k8s.io/docs/start/) for the Kubernetes cluster in this example, but any Kubernetes cluster can be used. Ensure that a Kubernetes cluster of some kind is running before applying the example config below.

This configuration will create a scalable Nginx Deployment with 2 replicas. It will expose the Nginx frontend using a Service of type NodePort, which will make Nginx accessible via the public IP of the node running the containers.

```hcl
terraform {
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = ">= 2.0.0"
    }
  }
}
provider "kubernetes" {
  config_path = "~/.kube/config"
}
resource "kubernetes_namespace" "test" {
  metadata {
    name = "nginx"
  }
}
resource "kubernetes_deployment" "test" {
  metadata {
    name      = "nginx"
    namespace = kubernetes_namespace.test.metadata.0.name
  }
  spec {
    replicas = 2
    selector {
      match_labels = {
        app = "MyTestApp"
      }
    }
    template {
      metadata {
        labels = {
          app = "MyTestApp"
        }
      }
      spec {
        container {
          image = "nginx"
          name  = "nginx-container"
          port {
            container_port = 80
          }
        }
      }
    }
  }
}
resource "kubernetes_service" "test" {
  metadata {
    name      = "nginx"
    namespace = kubernetes_namespace.test.metadata.0.name
  }
  spec {
    selector = {
      app = kubernetes_deployment.test.spec.0.template.0.metadata.0.labels.app
    }
    type = "NodePort"
    port {
      node_port   = 30201
      port        = 80
      target_port = 80
    }
  }
}
```

Use `terraform init` to download the specified version of the Kubernetes provider:

```
$ terraform init

Initializing the backend...

Initializing provider plugins...
- Finding hashicorp/kubernetes versions matching "2.0"...
- Installing hashicorp/kubernetes v2.0...
- Installed hashicorp/kubernetes v2.0 (unauthenticated)

Terraform has created a lock file .terraform.lock.hcl to record the provider
selections it made above. Include this file in your version control repository
so that Terraform can guarantee to make the same selections by default when
you run "terraform init" in the future.

Terraform has been successfully initialized!

You may now begin working with Terraform. Try running "terraform plan" to see
any changes that are required for your infrastructure. All Terraform commands
should now work.

If you ever set or change modules or backend configuration for Terraform,
rerun this command to reinitialize your working directory. If you forget, other
commands will detect it and remind you to do so if necessary.
```

Next, use `terraform plan` to display a list of resources to be created, and highlight any possible unknown attributes at apply time. For Deployments, all disk options are shown at plan time, but none will be created unless explicitly configured in the Deployment resource.

```
$ terraform plan

An execution plan has been generated and is shown below.
Resource actions are indicated with the following symbols:
  + create

Terraform will perform the following actions:

  # kubernetes_deployment.test will be created
  + resource "kubernetes_deployment" "test" {
      + id               = (known after apply)
      + wait_for_rollout = true

      + metadata {
          + generation       = (known after apply)
          + name             = "nginx"
          + namespace        = "nginx"
          + resource_version = (known after apply)
          + uid              = (known after apply)
        }

      + spec {
          + min_ready_seconds         = 0
          + paused                    = false
          + progress_deadline_seconds = 600
          + replicas                  = "2"
          + revision_history_limit    = 10

          + selector {
              + match_labels = {
                  + "app" = "MyTestApp"
                }
            }

          + strategy {
              + type = (known after apply)

              + rolling_update {
                  + max_surge       = (known after apply)
                  + max_unavailable = (known after apply)
                }
            }

          + template {
              + metadata {
                  + generation       = (known after apply)
                  + labels           = {
                      + "app" = "MyTestApp"
                    }
                  + name             = (known after apply)
                  + resource_version = (known after apply)
                  + uid              = (known after apply)
                }

              + spec {
                  + automount_service_account_token  = true
                  + dns_policy                       = "ClusterFirst"
                  + enable_service_links             = true
                  + host_ipc                         = false
                  + host_network                     = false
                  + host_pid                         = false
                  + hostname                         = (known after apply)
                  + node_name                        = (known after apply)
                  + restart_policy                   = "Always"
                  + service_account_name             = (known after apply)
                  + share_process_namespace          = false
                  + termination_grace_period_seconds = 30

                  + container {
                      + image                      = "nginx"
                      + image_pull_policy          = (known after apply)
                      + name                       = "nginx-container"
                      + stdin                      = false
                      + stdin_once                 = false
                      + termination_message_path   = "/dev/termination-log"
                      + termination_message_policy = (known after apply)
                      + tty                        = false

                      + port {
                          + container_port = 80
                          + protocol       = "TCP"
                        }

                      + resources {
                          + limits   = (known after apply)
                          + requests = (known after apply)
                        }

                      + volume_mount {
                          + mount_path        = (known after apply)
                          + mount_propagation = (known after apply)
                          + name              = (known after apply)
                          + read_only         = (known after apply)
                          + sub_path          = (known after apply)
                        }
                    }

                  + image_pull_secrets {
                      + name = (known after apply)
                    }

                  + readiness_gate {
                      + condition_type = (known after apply)
                    }

                  + volume {
                      + name = (known after apply)

                      + aws_elastic_block_store {
                          + fs_type   = (known after apply)
                          + partition = (known after apply)
                          + read_only = (known after apply)
                          + volume_id = (known after apply)
                        }

                      + azure_disk {
                          + caching_mode  = (known after apply)
                          + data_disk_uri = (known after apply)
                          + disk_name     = (known after apply)
                          + fs_type       = (known after apply)
                          + kind          = (known after apply)
                          + read_only     = (known after apply)
                        }

                      + azure_file {
                          + read_only        = (known after apply)
                          + secret_name      = (known after apply)
                          + share_name       = (known after apply)
                          + secret_namespace = (known after apply)
                        }

                      + ceph_fs {
                          + monitors    = (known after apply)
                          + path        = (known after apply)
                          + read_only   = (known after apply)
                          + secret_file = (known after apply)
                          + user        = (known after apply)

                          + secret_ref {
                              + name      = (known after apply)
                              + namespace = (known after apply)
                            }
                        }

                      + cinder {
                          + fs_type   = (known after apply)
                          + read_only = (known after apply)
                          + volume_id = (known after apply)
                        }

                      + config_map {
                          + default_mode = (known after apply)
                          + name         = (known after apply)
                          + optional     = (known after apply)

                          + items {
                              + key  = (known after apply)
                              + mode = (known after apply)
                              + path = (known after apply)
                            }
                        }

                      + csi {
                          + driver            = (known after apply)
                          + fs_type           = (known after apply)
                          + read_only         = (known after apply)
                          + volume_attributes = (known after apply)
                          + volume_handle     = (known after apply)

                          + controller_expand_secret_ref {
                              + name      = (known after apply)
                              + namespace = (known after apply)
                            }

                          + controller_publish_secret_ref {
                              + name      = (known after apply)
                              + namespace = (known after apply)
                            }

                          + node_publish_secret_ref {
                              + name      = (known after apply)
                              + namespace = (known after apply)
                            }

                          + node_stage_secret_ref {
                              + name      = (known after apply)
                              + namespace = (known after apply)
                            }
                        }

                      + downward_api {
                          + default_mode = (known after apply)

                          + items {
                              + mode = (known after apply)
                              + path = (known after apply)

                              + field_ref {
                                  + api_version = (known after apply)
                                  + field_path  = (known after apply)
                                }

                              + resource_field_ref {
                                  + container_name = (known after apply)
                                  + divisor        = (known after apply)
                                  + resource       = (known after apply)
                                }
                            }
                        }

                      + empty_dir {
                          + medium     = (known after apply)
                          + size_limit = (known after apply)
                        }

                      + fc {
                          + fs_type      = (known after apply)
                          + lun          = (known after apply)
                          + read_only    = (known after apply)
                          + target_ww_ns = (known after apply)
                        }

                      + flex_volume {
                          + driver    = (known after apply)
                          + fs_type   = (known after apply)
                          + options   = (known after apply)
                          + read_only = (known after apply)

                          + secret_ref {
                              + name      = (known after apply)
                              + namespace = (known after apply)
                            }
                        }

                      + flocker {
                          + dataset_name = (known after apply)
                          + dataset_uuid = (known after apply)
                        }

                      + gce_persistent_disk {
                          + fs_type   = (known after apply)
                          + partition = (known after apply)
                          + pd_name   = (known after apply)
                          + read_only = (known after apply)
                        }

                      + git_repo {
                          + directory  = (known after apply)
                          + repository = (known after apply)
                          + revision   = (known after apply)
                        }

                      + glusterfs {
                          + endpoints_name = (known after apply)
                          + path           = (known after apply)
                          + read_only      = (known after apply)
                        }

                      + host_path {
                          + path = (known after apply)
                          + type = (known after apply)
                        }

                      + iscsi {
                          + fs_type         = (known after apply)
                          + iqn             = (known after apply)
                          + iscsi_interface = (known after apply)
                          + lun             = (known after apply)
                          + read_only       = (known after apply)
                          + target_portal   = (known after apply)
                        }

                      + local {
                          + path = (known after apply)
                        }

                      + nfs {
                          + path      = (known after apply)
                          + read_only = (known after apply)
                          + server    = (known after apply)
                        }

                      + persistent_volume_claim {
                          + claim_name = (known after apply)
                          + read_only  = (known after apply)
                        }

                      + photon_persistent_disk {
                          + fs_type = (known after apply)
                          + pd_id   = (known after apply)
                        }

                      + projected {
                          + default_mode = (known after apply)

                          + sources {
                              + config_map {
                                  + name     = (known after apply)
                                  + optional = (known after apply)

                                  + items {
                                      + key  = (known after apply)
                                      + mode = (known after apply)
                                      + path = (known after apply)
                                    }
                                }

                              + downward_api {
                                  + items {
                                      + mode = (known after apply)
                                      + path = (known after apply)

                                      + field_ref {
                                          + api_version = (known after apply)
                                          + field_path  = (known after apply)
                                        }

                                      + resource_field_ref {
                                          + container_name = (known after apply)
                                          + quantity       = (known after apply)
                                          + resource       = (known after apply)
                                        }
                                    }
                                }

                              + secret {
                                  + name     = (known after apply)
                                  + optional = (known after apply)

                                  + items {
                                      + key  = (known after apply)
                                      + mode = (known after apply)
                                      + path = (known after apply)
                                    }
                                }

                              + service_account_token {
                                  + audience           = (known after apply)
                                  + expiration_seconds = (known after apply)
                                  + path               = (known after apply)
                                }
                            }
                        }

                      + quobyte {
                          + group     = (known after apply)
                          + read_only = (known after apply)
                          + registry  = (known after apply)
                          + user      = (known after apply)
                          + volume    = (known after apply)
                        }

                      + rbd {
                          + ceph_monitors = (known after apply)
                          + fs_type       = (known after apply)
                          + keyring       = (known after apply)
                          + rados_user    = (known after apply)
                          + rbd_image     = (known after apply)
                          + rbd_pool      = (known after apply)
                          + read_only     = (known after apply)

                          + secret_ref {
                              + name      = (known after apply)
                              + namespace = (known after apply)
                            }
                        }

                      + secret {
                          + default_mode = (known after apply)
                          + optional     = (known after apply)
                          + secret_name  = (known after apply)

                          + items {
                              + key  = (known after apply)
                              + mode = (known after apply)
                              + path = (known after apply)
                            }
                        }

                      + vsphere_volume {
                          + fs_type     = (known after apply)
                          + volume_path = (known after apply)
                        }
                    }
                }
            }
        }
    }

  # kubernetes_namespace.test will be created
  + resource "kubernetes_namespace" "test" {
      + id = (known after apply)

      + metadata {
          + generation       = (known after apply)
          + name             = "nginx"
          + resource_version = (known after apply)
          + uid              = (known after apply)
        }
    }

  # kubernetes_service.test will be created
  + resource "kubernetes_service" "test" {
      + id                     = (known after apply)
      + status                 = (known after apply)
      + wait_for_load_balancer = true

      + metadata {
          + generation       = (known after apply)
          + name             = "nginx"
          + namespace        = "nginx"
          + resource_version = (known after apply)
          + uid              = (known after apply)
        }

      + spec {
          + cluster_ip                  = (known after apply)
          + external_traffic_policy     = (known after apply)
          + health_check_node_port      = (known after apply)
          + publish_not_ready_addresses = false
          + selector                    = {
              + "app" = "MyTestApp"
            }
          + session_affinity            = "None"
          + type                        = "NodePort"

          + port {
              + node_port   = 30201
              + port        = 80
              + protocol    = "TCP"
              + target_port = "80"
            }
        }
    }

Plan: 3 to add, 0 to change, 0 to destroy.

------------------------------------------------------------------------

Note: You didn't specify an "-out" parameter to save this plan, so Terraform
can't guarantee that exactly these actions will be performed if
"terraform apply" is subsequently run.
```

Use `terraform apply` to create the resources shown above.

```
$ terraform apply --auto-approve

kubernetes_namespace.test: Creating...
kubernetes_namespace.test: Creation complete after 0s [id=nginx]
kubernetes_deployment.test: Creating...
kubernetes_deployment.test: Creation complete after 7s [id=nginx/nginx]
kubernetes_service.test: Creating...
kubernetes_service.test: Creation complete after 0s [id=nginx/nginx]

Apply complete! Resources: 3 added, 0 changed, 0 destroyed.
```

The resources are now visible in the Kubernetes cluster.

```
$ kubectl get all -n nginx

NAME                         READY   STATUS    RESTARTS   AGE
pod/nginx-86c669bff4-8g7g2   1/1     Running   0          38s
pod/nginx-86c669bff4-zgjkv   1/1     Running   0          38s

NAME            TYPE       CLUSTER-IP      EXTERNAL-IP   PORT(S)        AGE
service/nginx   NodePort   10.109.205.23   <none>        80:30201/TCP   30s

NAME                    READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/nginx   2/2     2            2           38s

NAME                               DESIRED   CURRENT   READY   AGE
replicaset.apps/nginx-86c669bff4   2         2         2       38s
```

The web server can be accessed using the public IP of the node running the Deployment. In this example, we're using minikube as the Kubernetes cluster, so the IP can be fetched using `minikube ip`.

```
$ curl $(minikube ip):30201

<!DOCTYPE html>
<html>
<head>
<title>Welcome to nginx!</title>
<style>
    body {
        width: 35em;
        margin: 0 auto;
        font-family: Tahoma, Verdana, Arial, sans-serif;
    }
</style>
</head>
<body>
<h1>Welcome to nginx!</h1>
<p>If you see this page, the nginx web server is successfully installed and
working. Further configuration is required.</p>

<p>For online documentation and support please refer to
<a href="http://nginx.org/">nginx.org</a>.<br/>
Commercial support is available at
<a href="http://nginx.com/">nginx.com</a>.</p>

<p><em>Thank you for using nginx.</em></p>
</body>
</html>
```

Alternatively, look for the hostIP associated with a running Nginx pod and combine it with the NodePort to assemble the URL:

```
$ kubectl get pod nginx-86c669bff4-zgjkv -n nginx -o json | jq .status.hostIP
"192.168.39.189"

$ kubectl get services -n nginx
NAME    TYPE       CLUSTER-IP      EXTERNAL-IP   PORT(S)        AGE
nginx   NodePort   10.109.205.23   <none>        80:30201/TCP   19m

$ curl 192.168.39.189:30201
```
