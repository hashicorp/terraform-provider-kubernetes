package main

import (
	"context"
	"flag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

func main() {
	debugFlag := flag.Bool("debug", false, "Start provider in stand-alone debug mode.")
	flag.Parse()

	serveOpts := &plugin.ServeOpts{
		ProviderFunc: kubernetes.Provider,
	}
	if debugFlag != nil && *debugFlag {
		plugin.Debug(context.Background(), "registry.terraform.io/hashicorp/kubernetes", serveOpts)
	} else {
		plugin.Serve(serveOpts)
	}
}
