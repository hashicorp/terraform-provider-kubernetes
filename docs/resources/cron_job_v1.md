---
subcategory: "batch/v1"
page_title: "Kubernetes: kubernetes_cron_job_v1"
description: |-
    A Cron Job creates Jobs on a time-based schedule. One CronJob object is like one line of a crontab (cron table) file.
---

# <no value>

<no value>

<no value>

## Example Usage

```terraform
resource "kubernetes_cron_job_v1" "demo" {
  metadata {
    name = "demo"
  }
  spec {
    concurrency_policy            = "Replace"
    failed_jobs_history_limit     = 5
    schedule                      = "1 0 * * *"
    timezone                      = "Etc/UTC"
    starting_deadline_seconds     = 10
    successful_jobs_history_limit = 10
    job_template {
      metadata {}
      spec {
        backoff_limit              = 2
        ttl_seconds_after_finished = 10
        template {
          metadata {}
          spec {
            container {
              name    = "hello"
              image   = "busybox"
              command = ["/bin/sh", "-c", "date; echo Hello from the Kubernetes cluster"]
            }
          }
        }
      }
    }
  }
}
```
