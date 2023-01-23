package tf5muxserver

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-mux/internal/logging"
)

// PrepareProviderConfig calls the PrepareProviderConfig method on each server
// in order, passing `req`. Response diagnostics are appended from all servers.
// Response PreparedConfig must be equal across all servers with nil values
// skipped.
func (s muxServer) PrepareProviderConfig(ctx context.Context, req *tfprotov5.PrepareProviderConfigRequest) (*tfprotov5.PrepareProviderConfigResponse, error) {
	rpc := "PrepareProviderConfig"
	ctx = logging.InitContext(ctx)
	ctx = logging.RpcContext(ctx, rpc)
	var resp *tfprotov5.PrepareProviderConfigResponse

	for _, server := range s.servers {
		ctx = logging.Tfprotov5ProviderServerContext(ctx, server)
		logging.MuxTrace(ctx, "calling downstream server")

		res, err := server.PrepareProviderConfig(ctx, req)

		if err != nil {
			return resp, fmt.Errorf("error from %T validating provider config: %w", server, err)
		}

		if res == nil {
			continue
		}

		if resp == nil {
			resp = res
			continue
		}

		if len(res.Diagnostics) > 0 {
			// This could implement Diagnostic deduplication if/when
			// implemented upstream.
			resp.Diagnostics = append(resp.Diagnostics, res.Diagnostics...)
		}

		// Do not check equality on missing PreparedConfig or unset PreparedConfig
		if res.PreparedConfig == nil {
			continue
		}

		equal, err := dynamicValueEquals(s.providerSchema.ValueType(), res.PreparedConfig, resp.PreparedConfig)

		if err != nil {
			return nil, fmt.Errorf("unable to compare PrepareProviderConfig PreparedConfig responses: %w", err)
		}

		if !equal {
			return nil, fmt.Errorf("got different PrepareProviderConfig PreparedConfig response from multiple servers, not sure which to use")
		}

		resp.PreparedConfig = res.PreparedConfig
	}

	return resp, nil
}
