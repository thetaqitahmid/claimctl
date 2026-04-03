package telemetry

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// InitTracer initializes the OpenTelemetry tracer
func InitTracer(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	var exporter trace.SpanExporter
	var err error

	// Check if stdout exporter is requested
	if isStdoutExporter("OTEL_TRACES_EXPORTER") {
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
	} else {
		endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
		if endpoint == "" {
			return func(context.Context) error { return nil }, nil
		}
		exporter, err = otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(endpoint))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Create resource
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			"",
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create TracerProvider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)

	// Set global TracerProvider
	otel.SetTracerProvider(tp)

	// Set global TextMapPropagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

// InitMeter initializes the OpenTelemetry meter and runtime metrics
func InitMeter(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	var metricExporter metric.Exporter
	var err error

	// Check if stdout exporter is requested
	if isStdoutExporter("OTEL_METRICS_EXPORTER") {
		metricExporter, err = stdoutmetric.New()
	} else {
		endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
		if endpoint == "" {
			return func(context.Context) error { return nil }, nil
		}
		metricExporter, err = otlpmetrichttp.New(ctx, otlpmetrichttp.WithEndpointURL(endpoint))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	// Create resource
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			"",
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create MeterProvider
	mp := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExporter, metric.WithInterval(3*time.Second))),
		metric.WithResource(res),
	)

	// Set global MeterProvider
	otel.SetMeterProvider(mp)

	// Start runtime metrics
	err = runtime.Start(runtime.WithMeterProvider(mp))
	if err != nil {
		return nil, fmt.Errorf("failed to start runtime metrics: %w", err)
	}

	return mp.Shutdown, nil
}

func isStdoutExporter(envVar string) bool {
	val := os.Getenv(envVar)
	return val == "stdout" || val == "console"
}
