# Trace package

This package contains shared initialization code that exports collected spans to Google
Cloud Trace.

## Initialization example:
```go
func main() {
	ctx := context.Background()
	tp, err := trace.InitTracer(ctx, true, 1.0, "my-service")
	if err != nil {
		log.Fatalf("unable to set up tracing: %v", err)
	}
	defer func () {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatalf("unable to shutdown tracing: %v", err)
		}
	}()
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
		grpc.ChainUnaryInterceptor(
			otelgrpc.UnaryServerInterceptor(),
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
tracer:  otel.Tracer("mycujoo.tv/bff-events/postgres")

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
	if c.tracer != nil {
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
	}


	err := c.handler.Handle(tctx, msg.Value)
	if err != nil {
		if c.tracer != nil {
			span.RecordError(err)
		}
		// log error, handle, etc
		return
	}

	if _, err = c.consumer.CommitMessage(msg.Message); err != nil {
		if c.tracer != nil {
			span.RecordError(err)
		}
		// log, handle error
	}
}
```
