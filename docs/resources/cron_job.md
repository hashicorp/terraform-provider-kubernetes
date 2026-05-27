---
subcategory: "batch/v1beta1"
page_title: "Kubernetes: kubernetes_cron_job"
description: |-
    A Cron Job creates Jobs on a time-based schedule. One CronJob object is like one line of a crontab (cron table) file.
---

# <no value> 

A Cron Job creates Jobs on a time-based schedule.

One CronJob object is like one line of a crontab (cron table) file. It runs a job periodically on a given schedule, written in Cron format.

Note: All CronJob `schedule` times are based on the timezone of the master where the job is initiated. For instructions on creating and working with cron jobs, and for an example of a spec file for a cron job, see [Kubernetes reference](https://kubernetes.io/docs/tasks/job/automated-tasks-with-cron-jobs/).

~> NOTE: With the release of Kubernetes v1.25, CronJob batch/v1beta1 has been removed. You can read more information about migrating to batch/v1 in the [Kubernetes 1.25 migration guide](https://kubernetes.io/docs/reference/using-api/deprecation-guide/#cronjob-v125).

<no value>

## Example Usage

```terraform
resource "kubernetes_cron_job" "demo" {
  metadata {
    name = "demo"
  }
  spec {
    concurrency_policy            = "Replace"
    failed_jobs_history_limit     = 5
    schedule                      = "1 0 * * *"
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


## Import 

```
$ terraform import kubernetes_corn_job_v1/example default/example
```