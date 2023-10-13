module github.com/mycujoo/go-stdlib/pkg/gcpconnect

go 1.21

require (
	buf.build/gen/go/bufbuild/eliza/connectrpc/go v1.11.1-20230726230109-bf1eaaff2a44.1
	buf.build/gen/go/bufbuild/eliza/protocolbuffers/go v1.31.0-20230726230109-bf1eaaff2a44.1
	connectrpc.com/connect v1.11.1
	connectrpc.com/otelconnect v0.6.0
	github.com/mycujoo/go-stdlib/pkg/connectlog v1.0.0
	github.com/mycujoo/go-stdlib/pkg/ctxslog v1.0.0
	golang.org/x/net v0.17.0
	google.golang.org/protobuf v1.31.0
)

replace (
	github.com/mycujoo/go-stdlib/pkg/connectlog => ../connectlog
	github.com/mycujoo/go-stdlib/pkg/ctxslog => ../ctxslog
)

require (
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	go.opentelemetry.io/otel v1.19.0 // indirect
	go.opentelemetry.io/otel/metric v1.19.0 // indirect
	go.opentelemetry.io/otel/trace v1.19.0 // indirect
	golang.org/x/text v0.13.0 // indirect
)
