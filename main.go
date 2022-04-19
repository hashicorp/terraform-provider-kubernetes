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
	tf5server "github.com/hashicorp/terraform-plugin-go/tfprotov5/tf5server"
	tf5muxserver "github.com/hashicorp/terraform-plugin-mux/tf5muxserver"

	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	manifest "github.com/hashicorp/terraform-provider-kubernetes/manifest/provider"
)

const providerName = "registry.terraform.io/hashicorp/kubernetes"

func main() {
	debugFlag := flag.Bool("debug", false, "Start provider in stand-alone debug mode.")
	flag.Parse()

	mainProvider := kubernetes.Provider().GRPCProvider
	manifestProvider := manifest.Provider()

	ctx := context.Background()
	muxer, err := tf5muxserver.NewMuxServer(ctx, mainProvider, manifestProvider)
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

	tf5server.Serve(providerName, muxer.ProviderServer, opts...)
}

// convertReattachConfig converts plugin.ReattachConfig to tfexec.ReattachConfig
func convertReattachConfig(reattachConfig *plugin.ReattachConfig) tfexec.ReattachConfig {
	return tfexec.ReattachConfig{
		Protocol: string(reattachConfig.Protocol),
		Pid:      reattachConfig.Pid,
		Test:     true,
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
