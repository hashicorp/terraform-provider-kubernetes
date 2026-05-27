---
subcategory: "policy/v1beta1"
page_title: "Kubernetes: kubernetes_pod_security_policy_v1beta1"
description: |-
  A Pod Security Policy is a cluster-level resource that controls security sensitive aspects of the pod specification.
---

# <no value>

<no value>

<no value>

~> NOTE: With the release of Kubernetes v1.25, PodSecurityPolicy has been removed. You can read more information about the removal of PodSecurityPolicy in the [Kubernetes 1.25 release notes](https://kubernetes.io/blog/2022/08/23/kubernetes-v1-25-release/#pod-security-changes).

## Example Usage

```terraform
resource "kubernetes_pod_security_policy_v1beta1" "example" {
  metadata {
    name = "terraform-example"
  }
  spec {
    privileged                 = false
    allow_privilege_escalation = false

    volumes = [
      "configMap",
      "emptyDir",
      "projected",
      "secret",
      "downwardAPI",
      "persistentVolumeClaim",
    ]

    run_as_user {
      rule = "MustRunAsNonRoot"
    }

    se_linux {
      rule = "RunAsAny"
    }

    supplemental_groups {
      rule = "MustRunAs"
      range {
        min = 1
        max = 65535
      }
    }

    fs_group {
      rule = "MustRunAs"
      range {
        min = 1
        max = 65535
      }
    }

    read_only_root_filesystem = true
  }
}
```

## Import

Pod Security Policy can be imported using its name, e.g.

```
$ terraform import kubernetes_pod_security_policy_v1beta1.example terraform-example
```
