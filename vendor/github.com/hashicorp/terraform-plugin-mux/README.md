[![PkgGoDev](https://pkg.go.dev/badge/github.com/hashicorp/terraform-plugin-mux)](https://pkg.go.dev/github.com/hashicorp/terraform-plugin-mux)

# terraform-plugin-mux

terraform-plugin-mux provides a method for combining Terraform providers built
in multiple different SDKs and frameworks to be combined into a single logical
provider for Terraform to work with. It is designed to allow provider
developers to implement resources and data sources at the level of abstraction
that is most suitable for that specific resource or data source, and to allow
provider developers to upgrade between SDKs or frameworks on a
resource-by-resource basis instead of all at once.

## Status

terraform-plugin-mux is a [Go
module](https://github.com/golang/go/wiki/Modules) versioned using [semantic
versioning](https://semver.org/).

The module is currently on a v0 major version, indicating our lack of
confidence in the stability of its exported API. Developers depending on it
should do so with an explicit understanding that the API may change and shift
until we hit v1.0.0, as we learn more about the needs and expectations of
developers working with the module.

We are confident in the correctness of the code and it is safe to build on so
long as the developer understands that the API may change in backwards
incompatible ways and they are expected to be tracking these changes.

## Compatibility

Providers built on terraform-plugin-mux will only be usable with Terraform
v0.12.0 and later. Developing providers for versions of Terraform below 0.12.0
is unsupported by the Terraform Plugin SDK team.

Providers built on the Terraform Plugin SDK must be using version 2.2.0 of the
Plugin SDK or higher to be able to be used with terraform-plugin-mux.

## Getting Started

terraform-plugin-mux exposes a minimal interface:

```go
func main() {
	ctx := context.Background()

	// the ProviderServer from SDKv2
	sdkv2 := sdkv2provider.Provider().GRPCProvider

	// the terraform-plugin-go provider
	tpg := protoprovider.Provider

	factory, err := tfmux.NewSchemaServerFactory(ctx, sdkv2, tpg)
	if err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}

	tf5server.Serve("registry.terraform.io/myorg/myprovider", factory.Server)
}
```

Each server needs a function that returns a
[`tfprotov5.ProviderServer`](https://pkg.go.dev/github.com/hashicorp/terraform-plugin-go/tfprotov5#ProviderServer).
Those get passed into a
[`NewSchemaServerFactory`](https://pkg.go.dev/github.com/hashicorp/terraform-plugin-mux#NewSchemaServerFactory)
function, which returns a factory capable of standing up Terraform provider
servers. Passing that factory into the
[`tf5server.Serve`](https://pkg.go.dev/github.com/hashicorp/terraform-plugin-go/tfprotov5/server#Serve)
function starts the server and lets Terraform connect to it.

## Testing

The Terraform Plugin SDK's [`helper/resource`](https://pkg.go.dev/github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource) package can be used to test any provider that implements the [`tfprotov5.ProviderServer`](https://pkg.go.dev/github.com/hashicorp/terraform-plugin-go/tfprotov5#ProviderServer) interface, which includes muxed providers created using `tfmux.NewSchemaServerFactory`.

You may wish to test a terraform-plugin-go provider's resources by supplying only that provider, and not the muxed provider, to the test framework: please see the example in https://github.com/hashicorp/terraform-plugin-go#testing in this case.

Otherwise, you should initialise a muxed provider in your testing code (conventionally in `provider_test.go`), and set this as the value of `ProtoV5ProviderFactories` in each [`TestCase`](https://pkg.go.dev/github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource#TestCase). For example:

```go
var testAccProtoV5ProviderFactories = map[string]func() (tfprotov5.ProviderServer, error){}

func init() {
  testAccProtoV5ProviderFactories["myprovider"] = func() (tfprotov5.ProviderServer, error) {
    ctx := context.Background()
    
    // the ProviderServer from SDKv2
    sdkv2 := sdkv2provider.Provider().GRPCProvider

    // the terraform-plugin-go provider
    tpg := protoprovider.Provider

    factory, err := tfmux.NewSchemaServerFactory(ctx, sdkv2, tpg)
    if err != nil {
      return nil, err
    }
    return factory.Server(), nil
  }
}
```

Here each `TestCase` in which you want to use the muxed provider should include `ProtoV5ProviderFactories: testAccProtoV5ProviderFactories`. Note that the test framework will return an error if you attempt to register the same provider using both `ProviderFactories` and `ProtoV5ProviderFactories`.


## Documentation

Documentation is a work in progress. The GoDoc for packages, types, functions,
and methods should have complete information, but we're working to add a
section to [terraform.io](https://terraform.io/) with more information about
the module, its common uses, and patterns developers may wish to take advantage
of.

Please bear with us as we work to get this information published, and please
[open
issues](https://github.com/hashicorp/terraform-plugin-mux/issues/new/choose)
with requests for the kind of documentation you would find useful.

## Contributing

Please see [`.github/CONTRIBUTING.md`](https://github.com/hashicorp/terraform-plugin-mux/blob/master/.github/CONTRIBUTING.md).

## License

This module is licensed under the [Mozilla Public License v2.0](https://github.com/hashicorp/terraform-plugin-mux/blob/master/LICENSE).
