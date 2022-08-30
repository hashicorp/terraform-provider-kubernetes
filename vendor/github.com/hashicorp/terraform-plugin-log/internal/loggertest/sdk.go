package loggertest

import (
	"context"
	"io"

	"github.com/hashicorp/terraform-plugin-log/internal/logging"
	"github.com/hashicorp/terraform-plugin-log/tfsdklog"
)

func SDKRoot(ctx context.Context, output io.Writer) context.Context {
	return tfsdklog.NewRootSDKLogger(
		ctx,
		logging.WithoutLocation(),
		logging.WithoutTimestamp(),
		logging.WithOutput(output),
	)
}

// SDKRootWithLocation is for testing code that affects go-hclog's caller
// information (location offset). Most testing code should avoid this, since
// correctly checking differences including the location is extra effort
// with little benefit.
func SDKRootWithLocation(ctx context.Context, output io.Writer) context.Context {
	return tfsdklog.NewRootSDKLogger(
		ctx,
		logging.WithoutTimestamp(),
		logging.WithOutput(output),
	)
}
