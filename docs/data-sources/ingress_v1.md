---
subcategory: "networking/v1"
page_title: "Kubernetes: kubernetes_ingress_v1"
description: |-
  Ingress is a collection of rules that allow inbound connections to reach the endpoints defined by a backend. An Ingress can be configured to give services externally-reachable urls, load balance traffic, terminate SSL, offer name based virtual hosting etc.
---

# <no value>

<no value>

<no value> 

## Example Usage

```terraform
data "kubernetes_ingress_v1" "example" {
  metadata {
    name = "terraform-example"
  }
}

resource "aws_route53_record" "example" {
  zone_id = data.aws_route53_zone.k8.zone_id
  name    = "example"
  type    = "CNAME"
  ttl     = "300"
  records = [data.kubernetes_ingress_v1.example.status.0.load_balancer.0.ingress.0.hostname]
}
```
