package gcpconnect

import (
	"log/slog"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"github.com/mycujoo/go-stdlib/pkg/connectlog"
	"google.golang.org/protobuf/encoding/protojson"
)

// GetHandlerOptions returns the default options for a connect.Handler.
func GetHandlerOptions(logger *slog.Logger) []connect.HandlerOption {
	return []connect.HandlerOption{
		connect.WithCodec(&protoJSONCodec{marshalOptions: protojson.MarshalOptions{
			// Fill unpopulated fields with their default values
			EmitUnpopulated: true,
		}}),
		connect.WithInterceptors(
			// Disable metrics since they are producing a lot of data
			otelconnect.NewInterceptor(
				otelconnect.WithoutMetrics(),
				otelconnect.WithoutServerPeerAttributes(),
			),
		),
		connect.WithRecover(connectlog.NewLoggingRecoverHandler(logger)),
		// We log after recover so panic logs are not duplicated.
		// Internally, `connect.WithRecover` is adding interceptor.
		connect.WithInterceptors(connectlog.NewLoggingInterceptor(logger)),
	}
}
