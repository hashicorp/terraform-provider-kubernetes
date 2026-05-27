---
subcategory: "batch/v1"
page_title: "Kubernetes: kubernetes_job_v1"
description: |-
    A Job creates one or more Pods and ensures that a specified number of them successfully terminate. You can also use a Job to run multiple Pods in parallel.
---

# <no value>

<no value>

<no value>

## Example Usage - No waiting

```terraform
resource "kubernetes_job_v1" "demo" {
  metadata {
    name = "demo"
  }
  spec {
    template {
      metadata {}
      spec {
        container {
          name    = "pi"
          image   = "perl"
          command = ["perl", "-Mbignum=bpi", "-wle", "print bpi(2000)"]
        }
        restart_policy = "Never"
      }
    }
    backoff_limit = 4
  }
  wait_for_completion = false
}
```

## Example Usage - waiting for job successful completion

```terraform
resource "kubernetes_job_v1" "demo" {
  metadata {
    name = "demo"
  }
  spec {
    template {
      metadata {}
      spec {
        container {
          name    = "pi"
          image   = "perl"
          command = ["perl", "-Mbignum=bpi", "-wle", "print bpi(2000)"]
        }
        restart_policy = "Never"
      }
    }
    backoff_limit = 4
  }
  wait_for_completion = true
  timeouts {
    create = "2m"
    update = "2m"
  }
}
```

Note:

- Kubernetes provider will treat update operations that change the Job spec resulting in the job re-run as "# forces replacement". In such cases, the `create` timeout value is used for both Create and Update operations.
- `wait_for_completion` is not applicable during Delete operations; thus, there is no "delete" timeout value for Delete operation.
