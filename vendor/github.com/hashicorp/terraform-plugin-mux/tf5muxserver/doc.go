// Combine multiple protocol version 5 provider servers into a single server.
//
// Supported protocol version 5 provider servers include any which implement
// the github.com/hashicorp/terraform-plugin-go/tfprotov5.ProviderServer
// interface, such as:
//
// - github.com/hashicorp/terraform-plugin-go/tfprotov5/tf5server
// - github.com/hashicorp/terraform-plugin-mux/tf6to5server
// - github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema
//
// Refer to the NewMuxServer() function for creating a combined server.
package tf5muxserver
