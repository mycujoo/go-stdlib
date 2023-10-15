module github.com/mycujoo/go-stdlib/pkg/connectlog

go 1.21

require (
	connectrpc.com/connect v1.11.1
	github.com/mycujoo/go-stdlib/pkg/ctxslog v1.0.0
)

replace github.com/mycujoo/go-stdlib/pkg/ctxslog => ../ctxslog

require google.golang.org/protobuf v1.31.0 // indirect
