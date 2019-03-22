# Testing Infrastructure

Here is where we keep the code of testing infrastructure. We have a few environments here:

 - AKS (Azure Kubernetes Service)
 - EKS (Amazon's Elastic Container Service)
 - GKE (Google Container Engine)

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

## How

Spinning up most environments should be as simple as

```
terraform apply -var=kubernetes_version=1.6.4
```

See each folder for more specific instructions.
