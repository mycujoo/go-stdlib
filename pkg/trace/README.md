# Trace package
[![GoDoc][godoc:image]][godoc:url]

This package contains shared initialization code that exports collected spans via otlp exporter.

```bash
go get "github.com/mycujoo/go-stdlib/pkg/trace"
```

## Initialization example:
Kubernetes deployment example:
```yaml
env:
 - name: POD_NAME
   valueFrom:
     fieldRef:
       fieldPath: metadata.name
 - name: NAMESPACE_NAME
   valueFrom:
     fieldRef:
       fieldPath: metadata.namespace
 - name: CONTAINER_NAME
   value: my-container-name
 - name: OTEL_RESOURCE_ATTRIBUTES
   value: k8s.pod.name=$(POD_NAME),k8s.namespace.name=$(NAMESPACE_NAME),k8s.container.name=$(CONTAINER_NAME),SampleRate=10
 - name: OTEL_SERVICE_NAME
   value: my-service-name
 - name: OTEL_TRACES_SAMPLER
   value: parentbased_traceidratio
 - name: OTEL_TRACES_SAMPLER_ARG
   value: 0.1 # value must match with SampleRate attribute
 - name: OTEL_EXPORTER_OTLP_ENDPOINT
   value: http://opentelemetry-collector.monitoring.svc.cluster.local.:4317
```
main.go:
```go
func main() {
	ctx := context.Background()
	shutdown, err := trace.Tracing(ctx)
	if err != nil {
		log.Fatalf("unable to set up tracing: %v", err)
	}
	defer shutdown()
}
```

## Setting up GRPC service interceptor:
```go
package main

import (
	"google.golang.org/grpc"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

func main() {
	grpcServer := grpc.NewServer(
		otelgrpc.UnaryServerInterceptor(
			otelgrpc.WithInterceptorFilter(func(info *otelgrpc.InterceptorInfo) bool {
				return !strings.HasPrefix(info.UnaryServerInfo.FullMethod, "/grpc.health.v1.Health")
			}),
			otelgrpc.WithMeterProvider(noop.NewMeterProvider()),
		))
}
```

## Tracing your own code (storage layer in this example):
```go
import (
	"go.opentelemetry.io/otel"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
)

// Inside constructor initialize separate instance of tracer 
tracer:  otel.Tracer("mycujoo.tv/postgres")

func (s *storage) startSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	spanCtx, span := s.tracer.Start(ctx,
		name,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			semconv.DBSystemPostgreSQL,
		),
	)
	return spanCtx, span
}

func (s *storage) GetDRMTechnologies(ctx context.Context, streamID string) ([]*bff_v1.Event, error) {
	spanCtx, span := s.startSpan(ctx, "GetDRMTechnologies")
	defer span.End()
	span.SetAttributes(attribute.String("stream_id", streamID))
	...
	if err != nil {
		// Record error in the span
		span.RecordError(err)
		return nil, errors.Wrap(err, "failed to read playlist DRM info")
	}
}
```

## Tracing kafka consumer:
```go
package consumer

import (
	...
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	kafkaPartitionKey = attribute.Key("messaging.kafka.partition")
	kafkaOffsetKey    = attribute.Key("mycujoo.tv/kafka.offset")
)

...

func (c *consumer) processMessage(ctx context.Context, msg *kafkaavro.Message) {
	topic := *msg.TopicPartition.Topic

	var span trace.Span
    ctx, span = c.tracer.Start(rootCtx,
        "kafka.process",
        trace.WithSpanKind(trace.SpanKindConsumer),
        trace.WithAttributes(
            semconv.MessagingSystemKey.String("kafka"),
            semconv.MessagingOperationProcess,
            semconv.MessagingDestinationKindTopic,
            semconv.MessagingDestinationKey.String(topic),
            semconv.MessagingMessageIDKey.String(string(msg.Key)),
            kafkaPartitionKey.Int(int(msg.TopicPartition.Partition)),
            kafkaOffsetKey.Int64(int64(msg.TopicPartition.Offset)),
        ),
    )
    defer span.End()

	err := c.handler.Handle(tctx, msg.Value)
	span.RecordError(err)
	if err != nil {
		// log error, handle, etc
		return
	}

	if _, err = c.consumer.CommitMessage(msg.Message); err != nil {
        span.RecordError(err)
		// log, handle error
	}
}
```

[godoc:image]:  https://godoc.org/github.com/mycujoo/go-stdlib/pkg/trace?status.svg
[godoc:url]:    https://godoc.org/github.com/mycujoo/go-stdlib/pkg/trace
