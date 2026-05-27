---
subcategory: "batch/v1"
page_title: "Kubernetes: kubernetes_job"
description: |-
    A Job creates one or more Pods and ensures that a specified number of them successfully terminate. You can also use a Job to run multiple Pods in parallel.
---

# <no value>

A Job creates one or more Pods and ensures that a specified number of them successfully terminate. As pods successfully complete, the Job tracks the successful completions. When a specified number of successful completions is reached, the task (ie, Job) is complete. Deleting a Job will clean up the Pods it created.

A simple case is to create one Job object in order to reliably run one Pod to completion. The Job object will start a new Pod if the first Pod fails or is deleted (for example due to a node hardware failure or a node reboot.

You can also use a Job to run multiple Pods in parallel.

<no value>

## Example Usage - No waiting

```terraform
resource "kubernetes_job" "demo" {
  metadata {
    name = "demo"
  }
  spec {
    template {
      metadata {}
      spec {
        container {
          name    = "pi"
          image   = "alpine"
          command = ["sh", "-c", "sleep 10"]
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
resource "kubernetes_job" "demo" {
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
