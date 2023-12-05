---
subcategory: "batch/v1beta1"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_cron_job"
description: |-
    A Cron Job creates Jobs on a time-based schedule. One CronJob object is like one line of a crontab (cron table) file.
---

# kubernetes_cron_job

  A Cron Job creates Jobs on a time-based schedule.

  One CronJob object is like one line of a crontab (cron table) file. It runs a job periodically on a given schedule, written in Cron format.

  Note: All CronJob `schedule` times are based on the timezone of the master where the job is initiated.
  For instructions on creating and working with cron jobs, and for an example of a spec file for a cron job, see [Kubernetes reference](https://kubernetes.io/docs/tasks/job/automated-tasks-with-cron-jobs/).

## Example Usage

```hcl
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

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard resource's metadata. For more info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
* `spec` - (Required) Spec defines the behavior of a CronJob. https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the resource that may be used to store arbitrary metadata.

~> By default, the provider ignores any annotations whose key names end with *kubernetes.io*. This is necessary because such annotations can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such annotations in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/

* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. Read more: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the service. May match selectors of replication controllers and services. 

~> By default, the provider ignores any labels whose key names end with *kubernetes.io*. This is necessary because such labels can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such labels in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/

* `name` - (Optional) Name of the service, must be unique. Cannot be updated. For more info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
* `namespace` - (Optional) Namespace defines the space within which name of the service must be unique.

#### Attributes

* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this service that can be used by clients to determine when service has changed. Read more: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency
* `uid` - The unique in time and space value for this service. For more info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids

### `spec`

#### Arguments

* `concurrency_policy` - (Optional) Specifies how to treat concurrent executions of a Job. Valid values are: - "Allow" (default): allows CronJobs to run concurrently; - "Forbid": forbids concurrent runs, skipping next run if previous run hasn't finished yet; - "Replace": cancels currently running job and replaces it with a new one
* `failed_jobs_history_limit` - (Optional) The number of failed finished jobs to retain. This is a pointer to distinguish between explicit zero and not specified. Defaults to 1.
* `job_template` - (Required) Specifies the job that will be created when executing a CronJob.
* `schedule` - (Required) The schedule in Cron format, see https://en.wikipedia.org/wiki/Cron.
* `starting_deadline_seconds` - (Optional) Deadline in seconds for starting the job if it misses scheduled time for any reason. Missed jobs executions will be counted as failed ones.
* `successful_jobs_history_limit` - (Optional) The number of successful finished jobs to retain. This is a pointer to distinguish between explicit zero and not specified. Defaults to 3.
* `suspend` - (Optional) This flag tells the controller to suspend subsequent executions, it does not apply to already started executions. Defaults to false.

### `job_template`

#### Arguments

* `metadata` - (Required) Standard object's metadata of the jobs created from this template. For more info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata
* `spec` - (Required) Specification of the desired behavior of the job. For more info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status

### `spec`

#### Arguments

* `active_deadline_seconds` - (Optional) Specifies the duration in seconds relative to the startTime that the job may be active before the system tries to terminate it; value must be positive integer.
* `backoff_limit` - (Optional) Specifies the number of retries before marking this job failed. Defaults to 6
* `completions` - (Optional) Specifies the desired number of successfully finished pods the job should be run with. Setting to nil means that the success of any pod signals the success of all pods, and allows parallelism to have any positive value. Setting to 1 means that parallelism is limited to 1 and the success of that pod signals the success of the job. For more info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/
* `manual_selector` - (Optional) Controls generation of pod labels and pod selectors. Leave `manualSelector` unset unless you are certain what you are doing. When false or unset, the system pick labels unique to this job and appends those labels to the pod template. When true, the user is responsible for picking unique labels and specifying the selector. Failure to pick a unique label may cause this and other jobs to not function correctly. However, You may see `manualSelector=true` in jobs that were created with the old `extensions/v1beta1` API. For more info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/#specifying-your-own-pod-selector
* `parallelism` - (Optional) Specifies the maximum desired number of pods the job should run at any given time. The actual number of pods running in steady state will be less than this number when `((.spec.completions - .status.successful) < .spec.parallelism)`, i.e. when the work left to do is less than max parallelism. For more info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/
* `selector` - (Optional) A label query over pods that should match the pod count. Normally, the system sets this field for you. For more info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
* `template` - (Optional) Describes the pod that will be created when executing a job. For more info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/
* `ttl_seconds_after_finished` - (Optional) ttlSecondsAfterFinished limits the lifetime of a Job that has finished execution (either Complete or Failed). If this field is set, ttlSecondsAfterFinished after the Job finishes, it is eligible to be automatically deleted. When the Job is being deleted, its lifecycle guarantees (e.g. finalizers) will be honored. If this field is unset, the Job won't be automatically deleted. If this field is set to zero, the Job becomes eligible to be deleted immediately after it finishes.

### `selector`

#### Arguments

* `match_expressions` - (Optional) A list of label selector requirements. The requirements are ANDed.
* `match_labels` - (Optional) A map of `{key,value}` pairs. A single `{key,value}` in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.

### `template`

#### Arguments

These arguments are the same as the for the `spec` block of a Pod.

Please see the [Pod resource](pod.html#spec) for reference.
