# Testing Infrastructure

Here is where we keep the code of testing infrastructure. We have a few environments here:

 - GKE (Google Container Engine)
 - kops @ AWS
 - kops @ GCE
 - minikube @ Packet

The goal is to make it **as simple as possible** with **as little maintenance burden**
as possible to spin up a particular version of K8S cluster and run
the whole acceptance test suite against it.

All tests are intended to run on a TeamCity agent in AWS.

## FAQ

### Why not just one environment?

Some environments may seem redundant. Kubernetes is designed
in a way that users shouldn't notice much difference between environments
and the provider (as well as K8S) should _just work_ in most environments.
However there _are_ differences we have to deal with as maintainers
of this provider. e.g. annotations, labels or volumes.

### Minikube & SSH tunnel

Running a VM on bare metal and setting up SSH tunnel to get to it may not be ideal,
but it still presents less maintainenace burden compared to building our own environment
or trying to expose tools which were designed to run locally.

#### Why not just native minikube?

Because AWS doesn't support nested virtualization [yet](https://aws.amazon.com/ec2/instance-types/i3/).

#### Why not just run test agent on Packet?

Because there is [no Packet plugin for TeamCity](https://plugins.jetbrains.com/search?headline=102-cloud-support&pr_productId=&canRedirectToPlugin=false&showPluginCount=false&tags=Cloud+Support).

### Why kops? Can't we just build our own Terraform configs?

Possibly, but that introduces potential maintenance burden
any time a new K8S version is introduced.

## How

Spinning up most environments should be as simple as

```
terraform apply -var=kubernetes_version=1.6.4
```

See each folder for more specific instructions.
