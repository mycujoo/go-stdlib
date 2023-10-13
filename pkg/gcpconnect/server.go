package gcpconnect

import (
	"context"
	"net/http"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// NewServer creates a new HTTP server.
// It contains a healthz endpoint and a handler for the given path.
// Healthz will return 200 OK if the given context is not done.
func NewServer(ctx context.Context, addr string, path string, handler http.Handler) (*http.Server, error) {
	mux := http.NewServeMux()

	mux.Handle(path, handler)
	mux.HandleFunc("/healthz", healthZHandleFunc(ctx))

	srv := &http.Server{
		Addr: addr,
		// Use h2c, so we can serve HTTP/2 without TLS.
		Handler: h2c.NewHandler(
			mux,
			&http2.Server{},
		),
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       1 * time.Minute,
		WriteTimeout:      1 * time.Minute,
		MaxHeaderBytes:    16 * 1024, // 16KiB
	}

	return srv, nil
}

var (
	statusError = []byte(`{"status":"NOT_SERVING"}`)
	statusOK    = []byte(`{"status":"SERVING"}`)
)

func healthZHandleFunc(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		if ctx.Err() != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write(statusError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(statusOK)
	}
}
