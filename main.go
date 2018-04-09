package main

import (
	"github.com/hashicorp/terraform/plugin"
	"github.com/sl1pm4t/terraform-provider-kubernetes/kubernetes"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: kubernetes.Provider})
}
