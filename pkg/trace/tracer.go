package trace

import (
	"context"

	cloudtrace "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"go.opentelemetry.io/contrib/propagators/autoprop"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"google.golang.org/api/option"
)

func InitCloudTracer(ctx context.Context, ratio float64, serviceName string) (*trace.TracerProvider, error) {
	exporter, err := cloudtrace.New(
		cloudtrace.WithContext(ctx),
		// we disable telemetry to avoid infinite loop when using OpenCensus bridge
		// https://github.com/open-telemetry/opentelemetry-go/issues/1928
		cloudtrace.WithTraceClientOptions([]option.ClientOption{option.WithTelemetryDisabled()}),
	)
	if err != nil {
		return nil, err
	}

	sampler := trace.TraceIDRatioBased(ratio)

	res, err := resource.New(ctx,
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	tp := trace.NewTracerProvider(
		trace.WithSampler(noHealthCheckSampler{fallback: trace.ParentBased(sampler)}),
		trace.WithResource(res),
		trace.WithBatcher(exporter),
	)

	otel.SetTracerProvider(tp)
	// Per otel specification default propagator is no-op.
	// We use the contrib autoprop package to install some default propagators.
	// List of propagators can be overridden by setting OTEL_PROPAGATORS environment variable.
	otel.SetTextMapPropagator(autoprop.NewTextMapPropagator())
	return tp, nil
}
