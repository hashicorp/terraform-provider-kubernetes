data "kubernetes_service_v1" "example" {
  metadata {
    name = "terraform-example"
  }
}

resource "aws_route53_record" "example" {
  zone_id = "data.aws_route53_zone.k8.zone_id"
  name    = "example"
  type    = "CNAME"
  ttl     = "300"
  records = [data.kubernetes_service_v1.example.status.0.load_balancer.0.ingress.0.hostname]
}
