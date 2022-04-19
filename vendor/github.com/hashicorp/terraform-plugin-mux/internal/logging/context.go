package logging

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-log/tfsdklog"
)

// InitContext creates SDK logger contexts.
func InitContext(ctx context.Context) context.Context {
	ctx = tfsdklog.NewSubsystem(ctx, SubsystemMux, tfsdklog.WithLevelFromEnv(EnvTfLogSdkMux))

	return ctx
}

// RpcContext injects the RPC name into logger contexts.
func RpcContext(ctx context.Context, rpc string) context.Context {
	ctx = tflog.With(ctx, KeyTfRpc, rpc)
	ctx = tfsdklog.With(ctx, KeyTfRpc, rpc)
	ctx = tfsdklog.SubsystemWith(ctx, SubsystemMux, KeyTfRpc, rpc)

	return ctx
}

// Tfprotov5ProviderServerContext injects the chosen provider Go type
func Tfprotov5ProviderServerContext(ctx context.Context, p tfprotov5.ProviderServer) context.Context {
	providerType := fmt.Sprintf("%T", p)
	ctx = tflog.With(ctx, KeyTfMuxProvider, providerType)
	ctx = tfsdklog.With(ctx, KeyTfMuxProvider, providerType)
	ctx = tfsdklog.SubsystemWith(ctx, SubsystemMux, KeyTfMuxProvider, providerType)

	return ctx
}

// Tfprotov6ProviderServerContext injects the chosen provider Go type
func Tfprotov6ProviderServerContext(ctx context.Context, p tfprotov6.ProviderServer) context.Context {
	providerType := fmt.Sprintf("%T", p)
	ctx = tflog.With(ctx, KeyTfMuxProvider, providerType)
	ctx = tfsdklog.With(ctx, KeyTfMuxProvider, providerType)
	ctx = tfsdklog.SubsystemWith(ctx, SubsystemMux, KeyTfMuxProvider, providerType)

	return ctx
}
