# gcplog

[![GoDoc][godoc:image]][godoc:url]

This package contains `slog` handler that can be used to log to stdout in a format that is compatible with Google
Cloud Logging.

NewAutoHandler also falls back to console logging if the environment is not GCP.

## Example

```go
package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/mycujoo/go-stdlib/pkg/gcplog"
)

func main() {
    h := gcplog.NewAutoHandler(os.Stderr, &gcplog.HandlerOptions{
        AddSource:      true,
        ServiceName:    "some-service",
        ReportErrors:   true,
    })
    
    err := fmt.Errorf("storage.Get: %w", os.ErrNotExist)
    
    logger := slog.New(h)
    logger.Error("operation failed", gcplog.Error(err))
}
```

To automatically set your service version during build you can use following command:
```shell
go build -ldflags "-X github.com/mycujoo/go-stdlib/pkg/gcplog.serviceVersion=$(git rev-parse HEAD)" -o ./bin/server ./cmd/server
```
Or something similar.

It is based on [slogdriver][slogdriver:url] package, but has some changes:

1. Integrated with open telemetry directly.
2. Trace context is optional.
3. Labels removed.
4. Added service context.
5. Support for cloud error reporting.

[godoc:image]:    https://pkg.go.dev/badge/github.com/mycujoo/go-stdlib/pkg/gcplog
[godoc:url]:      https://pkg.go.dev/github.com/mycujoo/go-stdlib/pkg/gcplog
[slogdriver:url]: https://github.com/jussi-kalliokoski/slogdriver
