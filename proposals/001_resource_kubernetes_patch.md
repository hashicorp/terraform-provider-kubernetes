# Adding a `kubernetes_patch` resource 

This proposal discusses adding a `kubernetes_patch` resource to the provider. 


## Some Background

[kubectl patch](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/update-api-object-kubectl-patch/) allows the user to mutate an existing API resource by specifying its name and kind, and a JSON or YAML partial with the desired change, e.g:

```
kubectl patch node k8s-node-1 -p '{"spec":{"unschedulable":true}}
```

There are 3 types of patch that can be applied:

1. **strategic** – This is the default and does a Strategic Merge Patch on the resource. This means that lists in the resource will be merged with lists in the patch, combining the items. This lets you add things to lists without knowing the current contents. This type is not supported on custom resources.


2. **merge** – This patch type causes your patch to replace lists entirely, which means if you want to add items to a list, the list in your patch needs to contain the current contents. This is the more blunt “clobber everything” strategy.


3. **json** – The patch type allows you to do a [JSON patch](http://jsonpatch.com/) using a list of positional operations. This lets you specify a list of operations, i.e add/replace/remove, the JSON path a specific field, and the new value, e.g:

```json
[
   {
       "op": "replace",
       "path": "/spec/containers/0/image",
       "value":"newimage"
   }
]
```

Technically, there is a 4th kind of patch supported by the API – `apply-patch+yaml` which is used to do server side apply although this is not exposed via the `--type` option of the `kubectl patch` command. 


# Use Cases

The [feature request](https://github.com/hashicorp/terraform-provider-kubernetes/issues/723) for this has 150+ 👍 now. A lot of people want this for a variety of reasons:

- Updating resources that come baked into AWS EKS so tolerations can be used
- Removing annotations from resources so they can be used with AWS Fargate
- Patching resources to be used with service meshes
- Patching ConfigMaps to be used in projects like ArgoCD
- Modifying the default namespace/service account on a cluster without importing / recreating

Previously we viewed this functionality as more of a "carry out this operation with `kubectl` before running terraform" thing, but given the mounting list of use cases and user appetite I think this fits our workflows based approach and we should try and accommodate.


# Proposed Configuration

A generic `kubernetes_patch` resource should allow you specify:

- The kind of resource being patched
- The resource's metadata (i.e name and namespace) 
- The type of patch operation 
- The patch itself 

In the kubernetes provider, it would look like this. 

```hcl
resource "kubernetes_patch" {
  kind = "namespace"

  metadata {
    name = "default"
  }
 
  patch = jsonencode({
    metadata = {
      labels = {
        test = "test"
      }
    }
  })
 
  type = "merge"
}
```

In the kubernetes-alpha provider:

```hcl
resource "kubernetes_patch" {
  provider = kubernetes-alpha

  kind = "namespace"

  metadata {
    name = "default"
  }
 
  patch = {
    metadata = {
      labels = {
        test = "test"
      }
    }
  }
 
  type = "merge"
}
```

## Possible Alternatives

A possible alternative to the generic patch resource above is to implement use-case specific resources for managing things like annotations, tolerations, and so on. 

For example:

```hcl
resource "kubernetes_annotations" {
  kind = "pod"

  metadata {
    name = "pod-name"
  }  

  annotations = {
    "test-annotation" = "value"
  }
}
```

The big downside of this is increased maintenance burden – we will have to implement a new resource for every use case that merits a resource being patched.  


## Discussion


1. **Should we implement this in the kubernetes provider, or in the kubernetes-alpha provider?**

Arguments _for_ kubernetes provider:
- Stability – users of the main provider won’t need to add kubernetes-alpha which still in flux. 
- Simple patch doesn’t have as many exotic challenges as custom resources.
- Easier implementation than in the alpha provider because it uses the main Terraform SDK. 

Arguments _against_ kubernetes provider:
- The patch has to be stored as a text string and used with `jsondecode`/`jsonencode`.
- We will have to do JSON munging to do any validation on the patch.



Arguments _for_ kubernetes-alpha provider:
- Cleaner syntax, no need for `jsonencode()`.
- Adding a highly wanted feature that incentivizes people to adopt the kubernetes-alpha provider.
- Code already exists in kubernetes-alpha to do things like validate against the OpenAPI schema at plan time.

Arguments _against_ kubernetes-alpha provider:
- Scope of the alpha provider becomes less tightly circumscribed around custom resources
- More conservative users will have to wait for the alpha provider to hit GA 


2. **Should we support the JSON patch type of just strategic and merge?**

The k8s documentation actually doesn’t illustrate the JSON patch type – it’s only there in the `--help` information.

You can do everything with “merge” and “strategic” that json patch can do (I think). None of the commenters in the issue seem to have asked for this and the examples were strategic patches.


3. **Should the resource be a “logical resource” that is applied once and then considered created forever, or should we read the resource to see if the patch needs to be applied again?**

"Patching" is something that doesn’t quite fit with the CRUD model of Terraform as it is a stateless one time operation. We could make this a resource that is simply created and has no read function, like we did with CertificateSigningRequest.

We could also use a custom diff to do a dry-run to check if the patch is actually going to do anything and then do a force-replace on it if there is.
