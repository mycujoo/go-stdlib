package trace

import (
	"fmt"
	"strings"

	"go.opentelemetry.io/otel/sdk/trace"
)

type noHealthCheckSampler struct {
	fallback trace.Sampler
}

func (ps noHealthCheckSampler) ShouldSample(p trace.SamplingParameters) trace.SamplingResult {
	if strings.HasPrefix(p.Name, "grpc.health.v1.Health") {
		return trace.SamplingResult{Decision: trace.Drop}
	}
	return ps.fallback.ShouldSample(p)
}

func (ps noHealthCheckSampler) Description() string {
	return fmt.Sprintf("SkipHealthOr{%s}", ps.fallback.Description())
}
