#!/usr/bin/bash
set -e

yum install -y curl yum-utils libvirt qemu-kvm virt-install libguestfs-tools libvirt-client qemu-kvm-tools

systemctl enable libvirtd.service
systemctl restart libvirtd.service

# Install docker-machine
curl -Lo docker-machine https://github.com/docker/machine/releases/download/v0.15.0/docker-machine-Linux-x86_64
chmod +x docker-machine
cp -f docker-machine /usr/local/bin/

# Install KVM2 driver for docker-machine
curl -Lo docker-machine-driver-kvm2 https://storage.googleapis.com/minikube/releases/latest/docker-machine-driver-kvm2
chmod +x docker-machine-driver-kvm2
cp -f docker-machine-driver-kvm2 /usr/local/bin/
