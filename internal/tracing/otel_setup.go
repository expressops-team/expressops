package tracing

import (
	"log"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

// InitTracerProvider initializes the OpenTelemetry tracer provider.
// Exports to the default standard output.
func InitTracerProvider(serviceName string) (*sdktrace.TracerProvider, error) {
	writer := os.Stdout

	exporter, err := stdouttrace.New(
		stdouttrace.WithWriter(writer),
		stdouttrace.WithPrettyPrint(),
		stdouttrace.WithoutTimestamps(),
	)
	if err != nil {
		return nil, err
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("0.1.0"),
		),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	tracer = otel.Tracer(serviceName + "-tracer")

	log.Println("TracerProvider inicializado, exportando a stdout.")
	return tp, nil
}

// GetTracer returns the global tracer instance.
func GetTracer() trace.Tracer {
	if tracer == nil {
		log.Println("ADVERTENCIA: Tracer no inicializado, usando NoopTracer.")
		return trace.NewNoopTracerProvider().Tracer("noop")
	}
	return tracer
}
