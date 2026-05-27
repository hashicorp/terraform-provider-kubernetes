---
subcategory: "core/v1"
page_title: "Kubernetes: kubernetes_service"
description: |-
  A Service is an abstraction which defines a logical set of pods and a policy by which to access them - sometimes called a micro-service.
---

# <no value>

<no value>

<no value>

## Example Usage

```terraform
data "kubernetes_service" "example" {
  metadata {
    name = "terraform-example"
  }
}

resource "aws_route53_record" "example" {
  zone_id = "data.aws_route53_zone.k8.zone_id"
  name    = "example"
  type    = "CNAME"
  ttl     = "300"
  records = [data.kubernetes_service.example.status.0.load_balancer.0.ingress.0.hostname]
}
```

