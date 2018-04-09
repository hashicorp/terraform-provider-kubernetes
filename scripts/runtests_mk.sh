#!/bin/bash

export TF_ACC=1
export KUBE_CTX=minikube
export KUBE_CTX_CLUSTER=minikube
export KUBE_CTX_AUTH_INFO=minikube

/usr/local/bin/go test -v -parallel 1 -timeout 3600s github.com/sl1pm4t/terraform-provider-kubernetes/kubernetes -run "^TestAccKubernetes.*"
