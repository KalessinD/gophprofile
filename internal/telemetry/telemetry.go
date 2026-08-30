package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

const serviceName = "gophprofile"

type (
	Shutdown func()
)

var tracerProvider *sdktrace.TracerProvider

// InitTracer initializes the global OpenTelemetry TracerProvider and sets it as default.
// It configures the OTLP gRPC exporter to send traces to Jaeger.
func InitTracer(ctx context.Context, shutdownTimeout time.Duration, jaegerEndpoint string) (Shutdown, error) {
	resources, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating OTel resources: %w", err)
	}

	traceExporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint(jaegerEndpoint),
		otlptracegrpc.WithInsecure(), // don't use TLS inside docker
	)
	if err != nil {
		return nil, fmt.Errorf("creating OTLP exporter: %w", err)
	}

	tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithResource(resources),
		sdktrace.WithBatcher(traceExporter),           // Используем BatchSpanProcessor для эффективности (отправляет пачками)
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // Sampler: AlwaysSample пишет 100% запросов (хорошо для дебага)
	)

	otel.SetTracerProvider(tracerProvider)

	// Настраиваем propagation контекста.
	// Это позволяет передавать TraceID в заголовках запросов к другим сервисам.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// return tracerProvider.Shutdown, nil

	// Возвращаем функцию для корректного завершения (flush данных перед выходом)
	return func() {
		ctx, cancel := context.WithTimeout(ctx, shutdownTimeout)
		defer cancel()

		if err := tracerProvider.Shutdown(ctx); err != nil {
			otel.Handle(err)
		}
	}, nil
}
