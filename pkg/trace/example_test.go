package trace_test

import (
	"context"
	"log"

	"github.com/mycujoo/go-stdlib/pkg/trace"
)

func Example_initTracer() {
	ctx := context.Background()
	// For Kubernetes container you need to set some environment variables inside pod spec
	// to get correct pod/container and namespace name:
	// https://github.com/open-telemetry/opentelemetry-go-contrib/tree/main/detectors/gcp#setting-kubernetes-attributes
	// We also set some other environment variables to configure exporter and sampler:
	// env:
	// - name: POD_NAME
	//   valueFrom:
	//     fieldRef:
	//       fieldPath: metadata.name
	// - name: NAMESPACE_NAME
	//   valueFrom:
	//     fieldRef:
	//       fieldPath: metadata.namespace
	// - name: CONTAINER_NAME
	//   value: my-container-name
	// - name: OTEL_RESOURCE_ATTRIBUTES
	//   value: k8s.pod.name=$(POD_NAME),k8s.namespace.name=$(NAMESPACE_NAME),k8s.container.name=$(CONTAINER_NAME),SampleRate=10
	// - name: OTEL_SERVICE_NAME
	//   value: my-service-name
	// - name: OTEL_TRACES_SAMPLER
	//   value: parentbased_traceidratio
	// - name: OTEL_TRACES_SAMPLER_ARG
	//   value: 0.1 # value must match with SampleRate attribute
	// - name: OTEL_EXPORTER_OTLP_ENDPOINT
	//   value: http://opentelemetry-collector.monitoring.svc.cluster.local.:4317
	shutdown, err := trace.InitTracing(ctx)
	if err != nil {
		log.Fatalf("unable to set up tracing: %v", err)
	}
	defer shutdown()
}
