#!/usr/bin/env bash +e

ORPHAN_ELBS=$(aws elb describe-tags \
    --load-balancer-names $(aws elb describe-load-balancers | jq -r '.LoadBalancerDescriptions[] | .LoadBalancerName') | \
    jq -r ".TagDescriptions[] | select(.Tags[] | .Key == \"kubernetes.io/cluster/$(terraform output cluster_name)\") | .LoadBalancerName")

for ELB in $ORPHAN_ELBS; do
    aws elb delete-load-balancer --load-balancer-name $ELB
done