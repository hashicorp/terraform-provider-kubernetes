---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_job"
sidebar_current: "docs-kubernetes-resource-job"
description: |-
    A Job creates one or more Pods and ensures that a specified number of them successfully terminate. You can also use a Job to run multiple Pods in parallel.
---

# kubernetes_job

  A Job creates one or more Pods and ensures that a specified number of them successfully terminate. As pods successfully complete, the Job tracks the successful completions. When a specified number of successful completions is reached, the task (ie, Job) is complete. Deleting a Job will clean up the Pods it created.

  A simple case is to create one Job object in order to reliably run one Pod to completion. The Job object will start a new Pod if the first Pod fails or is deleted (for example due to a node hardware failure or a node reboot.

  You can also use a Job to run multiple Pods in parallel.

## Example Usage

```hcl
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
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard resource's metadata. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
* `spec` - (Required) Specification of the desired behavior of a job. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#spec-and-status

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the resource that may be used to store arbitrary metadata.
**By default, the provider ignores any annotations whose key names end with *kubernetes.io*. This is necessary because such annotations can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such annotations in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem).**
More info: http://kubernetes.io/docs/user-guide/annotations
* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. Read more: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#idempotency
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the service. May match selectors of replication controllers and services. 
**By default, the provider ignores any labels whose key names end with *kubernetes.io*. This is necessary because such labels can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such labels in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem).**
More info: http://kubernetes.io/docs/user-guide/labels
* `name` - (Optional) Name of the service, must be unique. Cannot be updated. More info: http://kubernetes.io/docs/user-guide/identifiers#names
* `namespace` - (Optional) Namespace defines the space within which name of the service must be unique.

#### Attributes

* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this service that can be used by clients to determine when service has changed. Read more: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#concurrency-control-and-consistency
* `self_link` - A URL representing this service.
* `uid` - The unique in time and space value for this service. More info: http://kubernetes.io/docs/user-guide/identifiers#uids

### `spec`

#### Arguments

* `active_deadline_seconds` - (Optional) Specifies the duration in seconds relative to the startTime that the job may be active before the system tries to terminate it; value must be positive integer.
* `backoff_limit` - (Optional) Specifies the number of retries before marking this job failed. Defaults to 6
* `completions` - (Optional) Specifies the desired number of successfully finished pods the job should be run with. Setting to nil means that the success of any pod signals the success of all pods, and allows parallelism to have any positive value. Setting to 1 means that parallelism is limited to 1 and the success of that pod signals the success of the job. More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/
* `manual_selector` - (Optional) Controls generation of pod labels and pod selectors. Leave `manualSelector` unset unless you are certain what you are doing. When false or unset, the system pick labels unique to this job and appends those labels to the pod template. When true, the user is responsible for picking unique labels and specifying the selector. Failure to pick a unique label may cause this and other jobs to not function correctly. However, You may see `manualSelector=true` in jobs that were created with the old `extensions/v1beta1` API. More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/#specifying-your-own-pod-selector
* `parallelism` - (Optional) Specifies the maximum desired number of pods the job should run at any given time. The actual number of pods running in steady state will be less than this number when `((.spec.completions - .status.successful) < .spec.parallelism)`, i.e. when the work left to do is less than max parallelism. More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/
* `selector` - (Optional) A label query over pods that should match the pod count. Normally, the system sets this field for you. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
* `template` - (Optional) Describes the pod that will be created when executing a job. More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/

### `selector`

#### Arguments

* `match_expressions` - (Optional) A list of label selector requirements. The requirements are ANDed.
* `match_labels` - (Optional) A map of `{key,value}` pairs. A single `{key,value}` in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.

### `template`

#### Arguments

These arguments are the same as the for the `spec` block of a Pod.

Please see the [Pod resource](pod.html#spec-1) for reference.
