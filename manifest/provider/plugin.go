// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	tf5server "github.com/hashicorp/terraform-plugin-go/tfprotov5/tf5server"
)

var providerName = "registry.terraform.io/hashicorp/kubernetes"

// Serve is the default entrypoint for the provider.
func Serve(ctx context.Context, logger hclog.Logger) error {
	return tf5server.Serve(providerName, func() tfprotov5.ProviderServer { return &(RawProviderServer{logger: logger}) })
}

// Provider
func Provider() func() tfprotov5.ProviderServer {
	var logLevel string
	var ok bool = false
	for _, ev := range []string{"TF_LOG_PROVIDER_KUBERNETES", "TF_LOG_PROVIDER", "TF_LOG"} {
		logLevel, ok = os.LookupEnv(ev)
		if ok {
			break
		}
	}
	if !ok {
		logLevel = "off"
	}

	return func() tfprotov5.ProviderServer {
		return &(RawProviderServer{logger: hclog.New(&hclog.LoggerOptions{
			Level:  hclog.LevelFromString(logLevel),
			Output: os.Stderr,
		})})
	}
}

// ServeTest is for serving the provider in-process when testing.
// Returns a ReattachInfo or an error.
func ServeTest(ctx context.Context, logger hclog.Logger, t *testing.T) (tfexec.ReattachInfo, error) {
	reattachConfigCh := make(chan *plugin.ReattachConfig)

	go tf5server.Serve(providerName,
		func() tfprotov5.ProviderServer { return &(RawProviderServer{logger: logger}) },
		tf5server.WithDebug(ctx, reattachConfigCh, nil),
		tf5server.WithLoggingSink(t),
		tf5server.WithGoPluginLogger(logger),
	)

	reattachConfig, err := waitForReattachConfig(reattachConfigCh)
	if err != nil {
		return nil, fmt.Errorf("Error getting reattach config: %s", err)
	}

	return map[string]tfexec.ReattachConfig{
		providerName: convertReattachConfig(reattachConfig),
	}, nil
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
