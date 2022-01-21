package logging

import (
	"context"

	"github.com/hashicorp/terraform-plugin-log/tfsdklog"
)

const (
	// SubsystemProto is the tfsdklog subsystem name for protocol logging.
	SubsystemProto = "proto"
)

// ProtocolError emits a protocol subsystem log at ERROR level.
func ProtocolError(ctx context.Context, msg string, args ...interface{}) {
	tfsdklog.SubsystemError(ctx, SubsystemProto, msg, args)
}

// ProtocolTrace emits a protocol subsystem log at TRACE level.
func ProtocolTrace(ctx context.Context, msg string, args ...interface{}) {
	tfsdklog.SubsystemTrace(ctx, SubsystemProto, msg, args)
}
