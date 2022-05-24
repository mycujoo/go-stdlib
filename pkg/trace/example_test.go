package trace_test

import (
	"context"
	"log"

	"github.com/mycujoo/go-stdlib/pkg/trace"
)

func Example_initTracer() {
	ctx := context.Background()
	tp, err := trace.InitTracer(ctx, true, 1.0, "my-service")
	if err != nil {
		log.Fatalf("unable to set up tracing: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatalf("unable to shutdown tracing: %v", err)
		}
	}()
}
