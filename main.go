// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	tf5server "github.com/hashicorp/terraform-plugin-go/tfprotov5/tf5server"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
)

const (
	providerName = "registry.terraform.io/hashicorp/kubernetes"

	Version = "dev"
)

// Generate docs for website
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

func main() {
	debugFlag := flag.Bool("debug", false, "Start provider in stand-alone debug mode.")
	flag.Parse()

	ctx := context.Background()

	muxer, err := mux.MuxServer(ctx, Version)
	if err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}

	opts := []tf5server.ServeOpt{}
	if *debugFlag {
		reattachConfigCh := make(chan *plugin.ReattachConfig)
		go func() {
			reattachConfig, err := waitForReattachConfig(reattachConfigCh)
			if err != nil {
				fmt.Printf("Error getting reattach config: %s\n", err)
				return
			}
			printReattachConfig(reattachConfig)
		}()
		opts = append(opts, tf5server.WithDebug(ctx, reattachConfigCh, nil))
	}

	tf5server.Serve(providerName, func() tfprotov5.ProviderServer { return muxer }, opts...)
}

// convertReattachConfig converts plugin.ReattachConfig to tfexec.ReattachConfig
func convertReattachConfig(reattachConfig *plugin.ReattachConfig) tfexec.ReattachConfig {
	return tfexec.ReattachConfig{
		Protocol:        string(reattachConfig.Protocol),
		ProtocolVersion: reattachConfig.ProtocolVersion,
		Pid:             reattachConfig.Pid,
		Test:            true,
		Addr: tfexec.ReattachConfigAddr{
			Network: reattachConfig.Addr.Network(),
			String:  reattachConfig.Addr.String(),
		},
	}
}

// printReattachConfig prints the line the user needs to copy and paste
// to set the TF_REATTACH_PROVIDERS variable
func printReattachConfig(config *plugin.ReattachConfig) {
	reattachStr, err := json.Marshal(map[string]tfexec.ReattachConfig{
		"kubernetes": convertReattachConfig(config),
	})
	if err != nil {
		fmt.Printf("Error building reattach string: %s", err)
		return
	}
	fmt.Printf("# Provider server started\nexport TF_REATTACH_PROVIDERS='%s'\n", string(reattachStr))
}

// waitForReattachConfig blocks until a ReattachConfig is recieved on the
// supplied channel or times out after 2 seconds.
func waitForReattachConfig(ch chan *plugin.ReattachConfig) (*plugin.ReattachConfig, error) {
	select {
	case config := <-ch:
		return config, nil
	case <-time.After(2 * time.Second):
		return nil, fmt.Errorf("timeout while waiting for reattach configuration")
	}
}
