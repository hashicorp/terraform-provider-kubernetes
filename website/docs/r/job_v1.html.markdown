---
subcategory: "batch/v1"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_job_v1"
description: |-
    A Job creates one or more Pods and ensures that a specified number of them successfully terminate. You can also use a Job to run multiple Pods in parallel.
---

# kubernetes_job_v1

  A Job creates one or more Pods and ensures that a specified number of them successfully terminate. As pods successfully complete, the Job tracks the successful completions. When a specified number of successful completions is reached, the task (ie, Job) is complete. Deleting a Job will clean up the Pods it created.

  A simple case is to create one Job object in order to reliably run one Pod to completion. The Job object will start a new Pod if the first Pod fails or is deleted (for example due to a node hardware failure or a node reboot.

  You can also use a Job to run multiple Pods in parallel.

## Example Usage - No waiting

```hcl
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

```hcl
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

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard resource's metadata. For more info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata
* `spec` - (Required) Specification of the desired behavior of a job. For more info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
* `wait_for_completion` - 
(Optional) If `true` blocks job `create` or `update` until the status of the job has a `Complete` or `Failed` condition. Defaults to `true`.

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

* `active_deadline_seconds` - (Optional) Specifies the duration in seconds relative to the startTime that the job may be active before the system tries to terminate it; value must be positive integer.
* `backoff_limit` - (Optional) Specifies the number of retries before marking this job failed. Defaults to 6
* `completions` - (Optional) Specifies the desired number of successfully finished pods the job should be run with. Setting to nil means that the success of any pod signals the success of all pods, and allows parallelism to have any positive value. Setting to 1 means that parallelism is limited to 1 and the success of that pod signals the success of the job. For more info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/
* `completion_mode` - (Optional) Specifies how Pod completions are tracked. It can be `NonIndexed` (default) or `Indexed`. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/workloads/controllers/job/#completion-mode).
* `manual_selector` - (Optional) Controls generation of pod labels and pod selectors. Leave `manualSelector` unset unless you are certain what you are doing. When false or unset, the system pick labels unique to this job and appends those labels to the pod template. When true, the user is responsible for picking unique labels and specifying the selector. Failure to pick a unique label may cause this and other jobs to not function correctly. However, You may see `manualSelector=true` in jobs that were created with the old `extensions/v1beta1` API. For more info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/#specifying-your-own-pod-selector
* `pod_failure_policy` - (Optional) Specifies the policy of handling failed pods. In particular, it allows to specify the set of actions and conditions which need to be satisfied to take the associated action. If empty, the default behaviour applies - the counter of failed pods, represented by the jobs's .status.failed field, is incremented and it is checked against the backoffLimit. This field cannot be used in combination with restartPolicy=OnFailure.
* `parallelism` - (Optional) Specifies the maximum desired number of pods the job should run at any given time. The actual number of pods running in steady state will be less than this number when `((.spec.completions - .status.successful) < .spec.parallelism)`, i.e. when the work left to do is less than max parallelism. For more info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/
* `selector` - (Optional) A label query over pods that should match the pod count. Normally, the system sets this field for you. For more info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
* `template` - (Optional) Describes the pod that will be created when executing a job. For more info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/
* `ttl_seconds_after_finished` - (Optional) ttlSecondsAfterFinished limits the lifetime of a Job that has finished execution (either Complete or Failed). If this field is set, ttlSecondsAfterFinished after the Job finishes, it is eligible to be automatically deleted. When the Job is being deleted, its lifecycle guarantees (e.g. finalizers) will be honored. If this field is unset, the Job won't be automatically deleted. If this field is set to zero, the Job becomes eligible to be deleted immediately after it finishes.

### `pod_failure_policy`

#### Arguments

* `rule` - (Required) A list of pod failure policy rules. The rules are evaluated in order. Once a rule matches a Pod failure, the remaining of the rules are ignored.

### `rule`

#### Arguments

* `action` - (Optional) Specifies the action taken on a pod failure when the requirements are satisfied. Possible values are: - FailJob: indicates that the pod's job is marked as Failed and all running pods are terminated. - FailIndex: indicates that the pod's index is marked as Failed and will not be restarted.
* `on_exit_codes` - (Optional) Represents the requirement on the container exit codes.
* `on_pod_condition` - (Optional) Represents the requirement on the pod conditions. The requirement is represented as a list of pod condition patterns. The requirement is satisfied if at least one pattern matches an actual pod condition.

### `on_exit_codes`

#### Arguments

* `container_name` - (Optional) Restricts the check for exit codes to the container with the specified name. When null, the rule applies to all containers. When specified, it should match one the container or initContainer names in the pod template.
* `operator` - (Optional) Represents the relationship between the container exit code(s) and the specified values.
* `values` - (Required) Specifies the set of values. Each returned container exit code (might be multiple in case of multiple containers) is checked against this set of values with respect to the operator.

### `on_pod_condition`

#### Arguments

* `status` - (Optional) Specifies the required Pod condition status. To match a pod condition it is required that the specified status equals the pod condition status. Defaults to True.
* `type` - (Optional) Specifies the required Pod condition type. To match a pod condition it is required that specified type equals the pod condition type.

### `selector`

#### Arguments

* `match_expressions` - (Optional) A list of label selector requirements. The requirements are ANDed.
* `match_labels` - (Optional) A map of `{key,value}` pairs. A single `{key,value}` in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.

### `template`

#### Arguments

These arguments are the same as the for the `spec` block of a Pod.

Please see the [Pod resource](pod.html#spec) for reference.

## Timeouts

The following [Timeout](/docs/language/resources/syntax.html#operation-timeouts) configuration options are available for the `kubernetes_job_v1` resource when used with `wait_for_completion = true`:

* `create` - (Default `1m`) Used for creating a new job and waiting for a successful job completion.
* `update` - (Default `1m`) Used for updating an existing job and waiting for a successful job completion.

Note: 

- Kubernetes provider will treat update operations that change the Job spec resulting in the job re-run as "# forces replacement". 
In such cases, the `create` timeout value is used for both Create and Update operations.
- `wait_for_completion` is not applicable during Delete operations; thus, there is no "delete" timeout value for Delete operation. 
