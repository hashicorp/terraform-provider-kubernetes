package main

import (
	"github.com/hashicorp/terraform/plugin"
	"github.com/srhaber/terraform-provider-kubernetes/kubernetes"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: kubernetes.Provider})
}
