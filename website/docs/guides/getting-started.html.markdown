---
layout: "kubernetes"
page_title: "Kubernetes: Getting Started with Kubernetes provider"
sidebar_current: "docs-kubernetes-guide-getting-started"
description: |-
  This guide focuses on scheduling Kubernetes resources like Pods,
  Replication Controllers, Services etc. on top of a properly configured
  and running Kubernetes cluster.
---

# Getting Started with Kubernetes provider

## Kubernetes

[Kubernetes](https://kubernetes.io/) (K8S) is an open-source workload scheduler 
with focus on containerized applications.

There are at least 2 steps involved in scheduling your first container
on a Kubernetes cluster. You need the Kubernetes cluster with all its components
running _somewhere_ and then schedule the Kubernetes resources, like Pods,
Replication Controllers, Services etc.

This guide focuses mainly on the latter part and expects you to have
a properly configured & running Kubernetes cluster.

The guide also expects you to run the cluster on a cloud provider
where Kubernetes can automatically provision a load balancer.

## Why Terraform?

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

The easiest way to configure the provider is by creating/generating a config
in a default location (`~/.kube/config`). That allows you to leave the
provider block completely empty.

```hcl
provider "kubernetes" {}
```

If you wish to configure the provider statically you can do so by providing TLS certificates:

```hcl
provider "kubernetes" {
  host = "https://104.196.242.174"

  client_certificate     = file("~/.kube/client-cert.pem")
  client_key             = file("~/.kube/client-key.pem")
  cluster_ca_certificate = file("~/.kube/cluster-ca-cert.pem")
}
```

or by providing username and password (HTTP Basic Authorization):

```hcl
provider "kubernetes" {
  host = "https://104.196.242.174"

  username = "ClusterMaster"
  password = "MindTheGap"
}
```

After specifying the provider we may now run the following command
to download the latest version of the Kubernetes provider.

```
$ terraform init

Initializing the backend...

Initializing provider plugins...
- Checking for available provider plugins...
- Downloading plugin for provider "kubernetes" (terraform-providers/kubernetes) 1.8.0...

The following providers do not have any version constraints in configuration,
so the latest version was installed.

To prevent automatic upgrades to new major versions that may contain breaking
changes, it is recommended to add version = "..." constraints to the
corresponding provider blocks in configuration, with the constraint strings
suggested below.

* provider.kubernetes: version = "~> 1.8"

Terraform has been successfully initialized!

You may now begin working with Terraform. Try running "terraform plan" to see
any changes that are required for your infrastructure. All Terraform commands
should now work.

If you ever set or change modules or backend configuration for Terraform,
rerun this command to reinitialize your working directory. If you forget, other
commands will detect it and remind you to do so if necessary.
```

## Scheduling a Simple Application

The main object in any Kubernetes application is [a Pod](https://kubernetes.io/docs/concepts/workloads/pods/pod/#what-is-a-pod).
Pod consists of one or more containers that are placed
on cluster nodes based on CPU or memory availability.

Here we create a pod with a single container running the nginx web server,
exposing port 80 (HTTP) which can be then exposed
through the load balancer to the real user.

Unlike in this simple example you'd commonly run more than
a single instance of your application in production to reach
high availability and adding labels will allow Kubernetes to find all
pods (instances) for the purpose of forwarding the traffic
to the exposed port.

```hcl
resource "kubernetes_pod" "nginx" {
  metadata {
    name = "nginx-example"
    labels = {
      App = "nginx"
    }
  }

  spec {
    container {
      image = "nginx:1.7.8"
      name  = "example"

      port {
        container_port = 80
      }
    }
  }
}
```

The simplest way to expose your application to users is via [Service](https://kubernetes.io/docs/concepts/services-networking/service/).
Service is capable of provisioning a load-balancer in some cloud providers
and managing the relationship between pods and that load balancer
as new pods are launched and others die for any reason.

```hcl
resource "kubernetes_service" "nginx" {
  metadata {
    name = "nginx-example"
  }
  spec {
    selector = {
      App = kubernetes_pod.nginx.metadata[0].labels.App
    }
    port {
      port        = 80
      target_port = 80
    }

    type = "LoadBalancer"
  }
}
```

We may also add an output which will expose the IP address to the user

```hcl
output "lb_ip" {
  value = kubernetes_service.nginx.load_balancer_ingress[0].ip
}
```

Please note that this assumes a cloud provider provisioning IP-based
load balancer (like in Google Cloud Platform). If you run on a provider
with hostname-based load balancer (like in Amazon Web Services) you
should use the following snippet instead.

```hcl
output "lb_ip" {
  value = kubernetes_service.nginx.load_balancer_ingress[0].hostname
}
```

The plan will provide you an overview of planned changes, in this case
we should see 2 resources (Pod + Service) being added.
This commands gets more useful as your infrastructure grows and
becomes more complex with more components depending on each other
and it's especially helpful during updates.

```
$ terraform plan

Refreshing Terraform state in-memory prior to plan...
The refreshed state will be used to calculate this plan, but will not be
persisted to local or remote state storage.


------------------------------------------------------------------------

An execution plan has been generated and is shown below.
Resource actions are indicated with the following symbols:
  + create

Terraform will perform the following actions:

  # kubernetes_pod.nginx will be created
  + resource "kubernetes_pod" "nginx" {
      + id = (known after apply)

      + metadata {
          + generation       = (known after apply)
          + labels           = {
              + "App" = "nginx"
            }
          + name             = "nginx-example"
          + namespace        = "default"
          + resource_version = (known after apply)
          + self_link        = (known after apply)
          + uid              = (known after apply)
        }

      + spec {
          + automount_service_account_token  = false
          + dns_policy                       = "ClusterFirst"
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
              + image                    = "nginx:1.7.8"
              + image_pull_policy        = (known after apply)
              + name                     = "example"
              + stdin                    = false
              + stdin_once               = false
              + termination_message_path = "/dev/termination-log"
              + tty                      = false

              + port {
                  + container_port = 80
                  + protocol       = "TCP"
                }

              + resources {
                  + limits {
                      + cpu    = (known after apply)
                      + memory = (known after apply)
                    }

                  + requests {
                      + cpu    = (known after apply)
                      + memory = (known after apply)
                    }
                }

              + volume_mount {
                  + mount_path = (known after apply)
                  + name       = (known after apply)
                  + read_only  = (known after apply)
                  + sub_path   = (known after apply)
                }
            }

          + image_pull_secrets {
              + name = (known after apply)
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
                  + read_only     = (known after apply)
                }

              + azure_file {
                  + read_only   = (known after apply)
                  + secret_name = (known after apply)
                  + share_name  = (known after apply)
                }

              + ceph_fs {
                  + monitors    = (known after apply)
                  + path        = (known after apply)
                  + read_only   = (known after apply)
                  + secret_file = (known after apply)
                  + user        = (known after apply)

                  + secret_ref {
                      + name = (known after apply)
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

                  + items {
                      + key  = (known after apply)
                      + mode = (known after apply)
                      + path = (known after apply)
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
                          + quantity       = (known after apply)
                          + resource       = (known after apply)
                        }
                    }
                }

              + empty_dir {
                  + medium = (known after apply)
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
                      + name = (known after apply)
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
                      + name = (known after apply)
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

  # kubernetes_service.nginx will be created
  + resource "kubernetes_service" "nginx" {
      + id                    = (known after apply)
      + load_balancer_ingress = (known after apply)

      + metadata {
          + generation       = (known after apply)
          + name             = "nginx-example"
          + namespace        = "default"
          + resource_version = (known after apply)
          + self_link        = (known after apply)
          + uid              = (known after apply)
        }

      + spec {
          + cluster_ip                  = (known after apply)
          + external_traffic_policy     = (known after apply)
          + publish_not_ready_addresses = false
          + selector                    = {
              + "App" = "nginx"
            }
          + session_affinity            = "None"
          + type                        = "LoadBalancer"

          + port {
              + node_port   = (known after apply)
              + port        = 80
              + protocol    = "TCP"
              + target_port = "80"
            }
        }
    }

Plan: 2 to add, 0 to change, 0 to destroy.

------------------------------------------------------------------------

Note: You didn't specify an "-out" parameter to save this plan, so Terraform
can't guarantee that exactly these actions will be performed if
"terraform apply" is subsequently run.
```

As we're happy with the plan output we may carry on applying
proposed changes. `terraform apply` will take of all the hard work
which includes creating resources via API in the right order,
supplying any defaults as necessary and waiting for
resources to finish provisioning to the point when it can either
present useful attributes or a failure (with reason) to the user.

```
$ terraform apply -auto-approve

kubernetes_pod.nginx: Creating...
kubernetes_pod.nginx: Creation complete after 8s [id=default/nginx-example]
kubernetes_service.nginx: Creating...
kubernetes_service.nginx: Still creating... [10s elapsed]
kubernetes_service.nginx: Still creating... [20s elapsed]
kubernetes_service.nginx: Still creating... [30s elapsed]
kubernetes_service.nginx: Still creating... [40s elapsed]
kubernetes_service.nginx: Still creating... [50s elapsed]
kubernetes_service.nginx: Creation complete after 56s [id=default/nginx-example]

Apply complete! Resources: 2 added, 0 changed, 0 destroyed.

Outputs:

lb_ip = 34.77.88.233
```

You may now enter that IP address to your favourite browser
and you should see the nginx welcome page.

The [Kubernetes UI](https://kubernetes.io/docs/tasks/access-application-cluster/web-ui-dashboard/)
provides another way to check both the pod and the service there
once they're scheduled.

## Reaching Scalability and Availability

The Replication Controller allows you to replicate pods. This is useful
for maintaining overall availability and scalability of your application
exposed to the user.

We can just replace our Pod with RC from the previous config
and keep the Service there.

```hcl
resource "kubernetes_deployment" "nginx" {
  metadata {
    name = "scalable-nginx-example"
    labels = {
      App = "ScalableNginxExample"
    }
  }

  spec {
    replicas = 2
    selector {
      match_labels = {
        App = "ScalableNginxExample"
      }
    }
    template {
      metadata {
        labels = {
          App = "ScalableNginxExample"
        }
      }
      spec {
        container {
          image = "nginx:1.7.8"
          name  = "example"

          port {
            container_port = 80
          }

          resources {
            limits {
              cpu    = "0.5"
              memory = "512Mi"
            }
            requests {
              cpu    = "250m"
              memory = "50Mi"
            }
          }
        }
      }
    }
  }
}

resource "kubernetes_service" "nginx" {
  metadata {
    name = "nginx-example"
  }
  spec {
    selector = {
      App = kubernetes_deployment.nginx.spec.0.template.0.metadata[0].labels.App
    }
    port {
      port        = 80
      target_port = 80
    }

    type = "LoadBalancer"
  }
}

output "lb_ip" {
  value = kubernetes_service.nginx.load_balancer_ingress[0].ip
}
```

You may notice we also specified how much CPU and memory do we expect
single instance of that application to consume. This is incredibly
helpful for Kubernetes as it helps avoiding under-provisioning or over-provisioning
that would result in either unused resources (costing money) or lack
of resources (causing the app to crash or slow down).

```
$ terraform plan

# ...

Plan: 2 to add, 0 to change, 0 to destroy.

------------------------------------------------------------------------
# ...

```

```
$ terraform apply -auto-approve
kubernetes_deployment.nginx: Creating...
kubernetes_deployment.nginx: Creation complete after 10s [id=default/scalable-nginx-example]
kubernetes_service.nginx: Creating...
kubernetes_service.nginx: Still creating... [10s elapsed]
kubernetes_service.nginx: Still creating... [20s elapsed]
kubernetes_service.nginx: Still creating... [30s elapsed]
kubernetes_service.nginx: Still creating... [40s elapsed]
kubernetes_service.nginx: Still creating... [50s elapsed]
kubernetes_service.nginx: Creation complete after 59s [id=default/nginx-example]

Apply complete! Resources: 2 added, 0 changed, 0 destroyed.

Outputs:

lb_ip = 34.77.88.233
```

Unlike in previous example, the IP address here will direct traffic
to one of the 2 pods scheduled in the cluster.

### Updating Configuration

As our application user-base grows we might need more instances to be scheduled.
The easiest way to achieve this is to increase `replicas` field in the config
accordingly.

```hcl
resource "kubernetes_deployment" "example" {
# ...

  spec {
    replicas = 5

# ...

}
```

You can verify before hitting the API that you're only changing what
you intended to change and that someone else didn't modify
the resource you created earlier.

```
$ terraform plan

Refreshing Terraform state in-memory prior to plan...
The refreshed state will be used to calculate this plan, but will not be
persisted to local or remote state storage.

kubernetes_deployment.nginx: Refreshing state... (ID: default/scalable-nginx-example)
kubernetes_service.nginx: Refreshing state... (ID: default/nginx-example)

The Terraform execution plan has been generated and is shown below.
Resources are shown in alphabetical order for quick scanning. Green resources
will be created (or destroyed and then created if an existing resource
exists), yellow resources are being changed in-place, and red resources
will be destroyed. Cyan entries are data sources to be read.

Note: You didn't specify an "-out" parameter to save this plan, so when
"apply" is called, Terraform can't guarantee this is what will execute.

  ~ kubernetes_deployment.nginx
      spec.0.replicas: "2" => "5"


Plan: 0 to add, 1 to change, 0 to destroy.
```

As we're happy with the proposed plan, we can just apply that change.

```
$ terraform apply
```

and 3 more replicas will be scheduled & attached to the load balancer.

## Bonus: Managing Quotas and Limits

As an operator managing cluster you're likely also responsible for
using the cluster responsibly and fairly within teams.

Resource Quotas and Limit Ranges both offer ways to put constraints
in place around CPU, memory, disk space and other resources that
will be consumed by cluster users.

Resource Quota can constrain the whole namespace

```hcl
resource "kubernetes_resource_quota" "example" {
  metadata {
    name = "terraform-example"
  }
  spec {
    hard = {
      pods = 10
    }
    scopes = ["BestEffort"]
  }
}
```

whereas Limit Range can impose limits on a specific resource
type (e.g. Pod or Persistent Volume Claim).

```hcl
resource "kubernetes_limit_range" "example" {
  metadata {
    name = "terraform-example"
  }
  spec {
    limit {
      type = "Pod"
      max = {
        cpu    = "200m"
        memory = "1024M"
      }
    }
    limit {
      type = "PersistentVolumeClaim"
      min = {
        storage = "24M"
      }
    }
    limit {
      type = "Container"
      default = {
        cpu    = "50m"
        memory = "24M"
      }
    }
  }
}
```

```
$ terraform plan
```

```
$ terraform apply
```

## Conclusion

Terraform offers you an effective way to manage both compute for
your Kubernetes cluster and Kubernetes resources. Check out
the extensive documentation of the Kubernetes provider linked
from the menu.
