output "vpc_id" {
  value = aws_vpc.k8s-acc.id
}

output "subnets" {
  value = aws_subnet.k8s-acc.*.id
}

output "cluster_name" {
  value = random_id.cluster_name.hex
}

