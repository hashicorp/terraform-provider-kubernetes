package tf5muxserver

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-mux/internal/logging"
)

// ReadResource calls the ReadResource method, passing `req`, on the provider
// that returned the resource specified by req.TypeName in its schema.
func (s muxServer) ReadResource(ctx context.Context, req *tfprotov5.ReadResourceRequest) (*tfprotov5.ReadResourceResponse, error) {
	rpc := "ReadResource"
	ctx = logging.InitContext(ctx)
	ctx = logging.RpcContext(ctx, rpc)
	server, ok := s.resources[req.TypeName]

	if !ok {
		return nil, fmt.Errorf("%q isn't supported by any servers", req.TypeName)
	}

	ctx = logging.Tfprotov5ProviderServerContext(ctx, server)
	logging.MuxTrace(ctx, "calling downstream server")

	return server.ReadResource(ctx, req)
}
