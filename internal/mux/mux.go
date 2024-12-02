// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package mux

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-mux/tf5muxserver"
	framework "github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	manifest "github.com/hashicorp/terraform-provider-kubernetes/manifest/provider"
)

func MuxServer(ctx context.Context, v string) (tfprotov5.ProviderServer, error) {
	kubernetesProvider := kubernetes.Provider()

	providers := []func() tfprotov5.ProviderServer{
		kubernetesProvider.GRPCProvider,
		manifest.Provider(),
		providerserver.NewProtocol5(framework.New(v, kubernetesProvider.Meta)),
	}

	return tf5muxserver.NewMuxServer(ctx, providers...)
}
