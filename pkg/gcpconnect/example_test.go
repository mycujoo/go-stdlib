package gcpconnect_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"buf.build/gen/go/bufbuild/eliza/connectrpc/go/buf/connect/demo/eliza/v1/elizav1connect"
	elizav1 "buf.build/gen/go/bufbuild/eliza/protocolbuffers/go/buf/connect/demo/eliza/v1"
	connect2 "connectrpc.com/connect"
	"github.com/mycujoo/go-stdlib/pkg/ctxslog"
	"github.com/mycujoo/go-stdlib/pkg/gcpconnect"
)

func Example_server() {
	ctx := context.Background()
	logger := slog.Default()

	service := &mockService{}

	path, handler := elizav1connect.NewElizaServiceHandler(
		service,
		gcpconnect.GetHandlerOptions(logger)...,
	)

	srv, err := gcpconnect.NewServer(ctx, "localhost:8080", path, handler)
	if err != nil {
		logger.Error("failed to create server", "error", err)
	}

	logger.InfoContext(ctx, "server started",
		"addr", "localhost:8080",
	)

	err = srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("failed to start server", "error", err)
	}
}

type mockService struct{}

func (mockService) Say(ctx context.Context, req *connect2.Request[elizav1.SayRequest]) (*connect2.Response[elizav1.SayResponse], error) {
	ctxslog.AddArgs(ctx, slog.Int("state", 5), "sentence", req.Msg.GetSentence())
	return connect2.NewResponse(&elizav1.SayResponse{}), nil
}

func (mockService) Converse(ctx context.Context, c *connect2.BidiStream[elizav1.ConverseRequest, elizav1.ConverseResponse]) error {
	// TODO implement me
	panic("implement me")
}

func (mockService) Introduce(ctx context.Context, req *connect2.Request[elizav1.IntroduceRequest], c2 *connect2.ServerStream[elizav1.IntroduceResponse]) error {
	ctxslog.AddArgs(ctx, slog.String("name", req.Msg.GetName()))
	panic("implement me")
}
