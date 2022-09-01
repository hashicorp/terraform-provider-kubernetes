package tfsdklogtest

import (
	"context"
	"io"

	"github.com/hashicorp/terraform-plugin-log/internal/loggertest"
)

// RootLogger returns a context containing a SDK root logger suitable for unit
// testing that is:
//
//    - Written to the given io.Writer, such as a bytes.Buffer.
//    - Written with JSON output, that can be decoded with MultilineJSONDecode.
//    - Log level set to TRACE.
//    - Without location/caller information in log entries.
//    - Without timestamps in log entries.
//
func RootLogger(ctx context.Context, output io.Writer) context.Context {
	return loggertest.SDKRoot(ctx, output)
}
