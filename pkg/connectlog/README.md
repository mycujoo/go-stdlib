# Connect log package
[![GoDoc][godoc:image]][godoc:url]

This package contains Connect interceptor that uses `slog.Logger` to log 
results of handler calls.

Note: we have `gcpconnect` package that wraps this package and adds telemetry and server code.

By default it doesn't log when handler returns `nil` error. 
This can be changed by using `connectlog.WithSuccess()`.

Example:
```go
	path, handler := xxxconnect.NewXXXServiceHandler(
		service,
		connect.WithInterceptors(connectlog.NewLoggingInterceptor(logger)),
		connect.WithRecover(connectlog.NewLoggingRecoverHandler(logger)),
	)
```

[godoc:image]:  https://pkg.go.dev/badge/github.com/mycujoo/go-stdlib/pkg/connectlog
[godoc:url]:    https://pkg.go.dev/github.com/mycujoo/go-stdlib/pkg/connectlog
