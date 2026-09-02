package telemetry

import (
	"context"
	"fmt"
	"time"

	"github.com/KalessinD/gophprofile/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

type (
	Shutdown func()
)

var tracerProvider *sdktrace.TracerProvider

// newResources creates OTel Resource attributes common for both Traces and Metrics.
func newResources(ctx context.Context, serviceName string) (*resource.Resource, error) {
	res, err := resource.Merge(
		resource.DefaultWithContext(ctx),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating OTel resources: %w", err)
	}
	return res, nil
}

// InitTracer initializes the global OpenTelemetry TracerProvider and sets it as default.
// It configures the OTLP gRPC exporter to send traces to Jaeger.
func InitTracer(ctx context.Context, shutdownTimeout time.Duration, jaegerEndpoint, serviceName string) (Shutdown, error) {
	resources, err := newResources(ctx, serviceName)
	if err != nil {
		return nil, err
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

// InitMeterProvider initializes the global OpenTelemetry MeterProvider.
// It uses the OTLP HTTP exporter (as recommended in Yandex Practicum) to push metrics.
func InitMeterProvider(ctx context.Context, shutdownTimeout, metricReadperiod time.Duration, otlpEndpoint, serviceName string) (Shutdown, error) {
	resources, err := newResources(ctx, serviceName)
	if err != nil {
		return nil, err
	}

	/*
		metricsEndpoint := strings.Replace(otlpEndpoint, ":4317", ":4318", 1)
		metricExporter, err := otlpmetrichttp.New(ctx,
			otlpmetrichttp.WithEndpoint(metricsEndpoint),
			otlpmetrichttp.WithInsecure(),
		)
		if err != nil {
			return nil, fmt.Errorf("creating OTLP metric exporter: %w", err)
		}
	*/

	metricExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(otlpEndpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("creating OTLP metric exporter: %w", err)
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithResource(resources),
		metric.WithReader(metric.NewPeriodicReader(metricExporter, metric.WithInterval(metricReadperiod))),
	)
	otel.SetMeterProvider(meterProvider)

	return func() {
		ctx, cancel := context.WithTimeout(ctx, shutdownTimeout)
		defer cancel()
		if err := meterProvider.Shutdown(ctx); err != nil {
			otel.Handle(err)
		}
	}, nil
}

// InitAll initializes the global OpenTelemetry MeterProvider and the global OpenTelemetry TracerProvider.
func InitAll(ctx context.Context, cfg *config.Otel, serviceName string) (Shutdown, error) {
	otelShutdown, err := InitTracer(ctx, cfg.ShutdownTimeout, cfg.ExporterOTLPEndpoint, serviceName)
	if err != nil {
		return nil, err
	}

	metricsShutdown, err := InitMeterProvider(ctx, cfg.ShutdownTimeout, cfg.MetricReadPeriod, cfg.ExporterOTLPEndpoint, serviceName)
	if err != nil {
		return nil, err
	}

	return func() {
		metricsShutdown()
		otelShutdown()
	}, nil
}
