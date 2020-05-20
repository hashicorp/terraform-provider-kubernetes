## Developing the provider

Thank you for your interest in contributing to the Kubernetes provider. We welcome your contributions. Here you'll find information to help you get started with provider development.

## Documentation

Our [provider development documentation](https://www.terraform.io/docs/extend/) provides a good start into developing an understanding of provider development. It's the best entry point if you are new to contributing to this provider.

To learn more about how to create issues and pull requests in this repository, and what happens after they are created, you may refer to the resources below:
- [Issue creation and lifecycle](ISSUES.md)
- [Pull Request creation and lifecycle](PULL_REQUESTS.md)
- [Frequently Asked Questions](FAQ.md)


## Building the provider

Clone repository to: `$GOPATH/src/github.com/terraform-providers/terraform-provider-kubernetes`

```sh
$ mkdir -p $GOPATH/src/github.com/terraform-providers; cd $GOPATH/src/github.com/terraform-providers
$ git clone git@github.com:terraform-providers/terraform-provider-kubernetes
```

Enter the provider directory and build the provider

```sh
$ cd $GOPATH/src/github.com/terraform-providers/terraform-provider-kubernetes
$ make build
```

Statically linking binaries can be required for testing development builds in containers not providing all dependencies, e.g.:

```
# CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"'
```

## Contributing to the provider

### Contributing Resources

In order to prevent breaking changes and migration of user-created resources, resources included in this provider will be limited to stable (aka `v1`) and beta APIs (with beta resources, readiness for inclusion will be assessed individually). You can find `v1` resources in the Kubernetes [API documentation](https://kubernetes.io/docs/reference/#api-reference) for the appropriate version of Kubernetes.

### Development Environment

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (version 1.9+ is *required*). You'll also need to correctly setup a [GOPATH](http://golang.org/doc/code.html#GOPATH), as well as adding `$GOPATH/bin` to your `$PATH`.

To compile the provider, run `make build`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

```sh
$ make build
...
$ $GOPATH/bin/terraform-provider-kubernetes
...
```

In order to test the provider, you can simply run `make test`.

```sh
$ make test
```

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```sh
$ make testacc
```
