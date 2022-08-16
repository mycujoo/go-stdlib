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
	// env:
	// - name: POD_NAME
	//  valueFrom:
	//    fieldRef:
	//      fieldPath: metadata.name
	// - name: NAMESPACE_NAME
	//  valueFrom:
	//    fieldRef:
	//      fieldPath: metadata.namespace
	// - name: CONTAINER_NAME
	//  value: my-container-name
	// - name: OTEL_RESOURCE_ATTRIBUTES
	//  value: k8s.pod.name=$(POD_NAME),k8s.namespace.name=$(NAMESPACE_NAME),k8s.container.name=$(CONTAINER_NAME)
	tp, err := trace.InitCloudTracer(ctx, 1.0, "my-service")
	if err != nil {
		log.Fatalf("unable to set up tracing: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatalf("unable to shutdown tracing: %v", err)
		}
	}()
}
