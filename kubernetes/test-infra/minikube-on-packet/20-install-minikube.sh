#!/usr/bin/bash

# Install kubectl

KUBECTL_VERSION=$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)
curl -LO https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl \
  && chmod a+x kubectl && mv kubectl /usr/local/bin/
kubectl version

# Install minikube
curl -Lo minikube https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64 \
  && chmod a+x minikube && mv minikube /usr/local/bin/
minikube version
minikube config set WantReportErrorPrompt false
minikube config set vm-driver kvm2
