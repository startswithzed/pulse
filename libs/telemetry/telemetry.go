package telemetry

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
)

func InitSDK(ctx context.Context, serviceName, serviceVersion, otelEndpoint, environment string, logJSON bool) (func(context.Context) error, error) {
	if environment == "" {
		environment = "development"
	}

	res, err := newResource(ctx, serviceName, serviceVersion, environment)
	if err != nil {
		return nil, err
	}

	tracerProvider, err := initTracing(ctx, res, otelEndpoint)
	if err != nil {
		return nil, err
	}

	meterProvider, err := initMetrics(ctx, res, otelEndpoint)
	if err != nil {
		return nil, err
	}

	loggerProvider, err := initLogging(ctx, res, otelEndpoint, serviceName, logJSON)
	if err != nil {
		return nil, err
	}

	shutdown := func(ctx context.Context) error {
		var errs []error

		if err := tracerProvider.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}

		if err := meterProvider.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}

		if err := loggerProvider.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}

		if len(errs) > 0 {
			return errs[0]
		}
		return nil
	}

	return shutdown, nil
}

func newResource(ctx context.Context, serviceName, serviceVersion, environment string) (*resource.Resource, error) {
	return resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
			semconv.ServiceInstanceID(uuid.New().String()),
			attribute.String("deployment.environment.name", environment),
		),
	)
}

func initTracing(ctx context.Context, res *resource.Resource, otelEndpoint string) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(otelEndpoint),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithTimeout(30*time.Second),
	)
	if err != nil {
		return nil, err
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	otel.SetTracerProvider(provider)

	return provider, nil
}

func initMetrics(ctx context.Context, res *resource.Resource, otelEndpoint string) (*metric.MeterProvider, error) {
	exporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(otelEndpoint),
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithTimeout(30*time.Second),
	)
	if err != nil {
		return nil, err
	}

	provider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(
			exporter,
			metric.WithInterval(3*time.Second),
		)),
		metric.WithResource(res),
	)

	otel.SetMeterProvider(provider)

	return provider, nil
}

func initLogging(ctx context.Context, res *resource.Resource, otelEndpoint, serviceName string, logJSON bool) (*sdklog.LoggerProvider, error) {
	exporter, err := otlploggrpc.New(ctx,
		otlploggrpc.WithEndpoint(otelEndpoint),
		otlploggrpc.WithInsecure(),
		otlploggrpc.WithTimeout(30*time.Second),
	)
	if err != nil {
		return nil, err
	}

	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		sdklog.WithResource(res),
	)

	global.SetLoggerProvider(provider)

	setupSlogHandlers(serviceName, provider, logJSON)

	return provider, nil
}

func setupSlogHandlers(serviceName string, provider *sdklog.LoggerProvider, logJSON bool) {
	var consoleHandler slog.Handler
	if logJSON {
		consoleHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	} else {
		consoleHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}

	baseOtelHandler := otelslog.NewHandler(
		serviceName,
		otelslog.WithLoggerProvider(provider),
		otelslog.WithSource(true),
	)

	otelHandler := jsonBodyHandler{
		handler: baseOtelHandler,
		service: serviceName,
	}

	logger := slog.New(multiHandler{
		handlers: []slog.Handler{consoleHandler, otelHandler},
		service:  serviceName,
	})

	slog.SetDefault(logger)
}
