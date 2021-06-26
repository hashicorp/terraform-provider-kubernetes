## Building and installing

As we do not yet publish releases for this provider to registry.terraform.io, you have to either [download a release from Github](https://github.com/hashicorp/terraform-provider-kubernetes/manifest/releases) or build and install it manually as indicated below.

Make sure you have a supported version of Go installed and working.

Checkout or download this repository, then open a terminal and change to its directory.

### Installing the provider to `terraform.d/plugins`
```
make install
```
This will build the provider and place the provider binary in your [plugins directory](https://www.terraform.io/docs/extend/how-terraform-works.html#plugin-locations).

You are now ready to use the provider. You can find example TF configurations in this repository under the `./examples`.

### Using `-plugin-dir` 

Alternatively, you can run:

```
make build
```

This will place the provider binary in the top level of the provider directory. You can then use it with terraform by specifying the `-plugin-dir` option when running `terraform init`

```
terraform init -plugin-dir /path/to/terraform-provider-alpha
```
