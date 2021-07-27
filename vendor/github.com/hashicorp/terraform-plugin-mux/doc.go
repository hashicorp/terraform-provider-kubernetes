// Package tfmux provides a multiplexer that allows joining multiple Terraform
// provider servers into a single gRPC server.
//
// This allows providers to use any framework or SDK built on
// github.com/hashicorp/terraform-plugin-go to build resources for their
// provider, and to join all the resources into a single logical provider even
// though they're implemented in different SDKs or frameworks.
package tfmux
