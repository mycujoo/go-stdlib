package trace

import (
	"context"

	cloudtrace "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"google.golang.org/api/option"
)

func InitTracer(ctx context.Context, useCloudtrace bool, ratio float64, serviceName string) (*trace.TracerProvider, error) {
	var exporter trace.SpanExporter
	var err error
	if useCloudtrace {
		exporter, err = cloudtrace.New(
			cloudtrace.WithContext(ctx),
			// we disable telemetry to avoid infinite loop when using OpenCensus bridge
			// https://github.com/open-telemetry/opentelemetry-go/issues/1928
			cloudtrace.WithTraceClientOptions([]option.ClientOption{option.WithTelemetryDisabled()}),
		)
		if err != nil {
			return nil, err
		}
	} else {
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, err
		}
	}
	sampler := trace.TraceIDRatioBased(ratio)

	res, err := resource.New(ctx,
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String(serviceName),
		),
	)

	tp := trace.NewTracerProvider(
		trace.WithSampler(noHealthCheckSampler{fallback: trace.ParentBased(sampler)}),
		trace.WithResource(res),
		trace.WithBatcher(exporter),
	)

	otel.SetTracerProvider(tp)
	return tp, nil
}
