#!/usr/bin/bash
set -e

# Install kubectl
KUBECTL_VERSION=$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)
curl -LO https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl \
  && chmod a+x kubectl && mv kubectl /usr/local/bin/
kubectl version

# Install minikube
curl -Lo minikube https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64 \
  && chmod a+x minikube && mv minikube /usr/local/bin/
minikube version

# Setting WantReportErrorPrompt to false prevents minikube from interactively prompting the user 
# for consent on sending diagnostic information back home in case of failure.
minikube config set WantReportErrorPrompt false
minikube config set vm-driver kvm2

# These settings allocate half of the host's CPU cores and memory to the minikube virtual machine.
# The proportion can be adjusted by chaning the '/ 2' factor in the expressions below.
minikube config set cpus $(($(lscpu -p | grep -cv '#') / 2))
minikube config set memory $(($(free --mega -tw | grep Mem: | cut -d' ' -f12) / 2))
