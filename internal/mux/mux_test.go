// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package mux

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// Every server has to answer Terraform's stop request. An error from one of
// them keeps the plugin server from cancelling the requests in progress.
func TestMuxServerStopProvider(t *testing.T) {
	ctx := context.Background()
	server, err := MuxServer(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := server.StopProvider(ctx, &tfprotov6.StopProviderRequest{})
	if err != nil {
		t.Fatalf("stop request: %s", err)
	}
	if resp.Error != "" {
		t.Fatalf("stop request: %s", resp.Error)
	}
}
