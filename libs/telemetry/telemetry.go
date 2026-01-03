package telemetry

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
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

	otelslog "go.opentelemetry.io/contrib/bridges/otelslog"
)

type multiHandler struct {
	handlers []slog.Handler
	service  string
}

func (h multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h multiHandler) Handle(ctx context.Context, r slog.Record) error {
	r.AddAttrs(slog.String("service", h.service))
	for _, handler := range h.handlers {
		if err := handler.Handle(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

func (h multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithAttrs(attrs)
	}
	return multiHandler{handlers: handlers, service: h.service}
}

func (h multiHandler) WithGroup(group string) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithGroup(group)
	}
	return multiHandler{handlers: handlers, service: h.service}
}

func InitSDK(ctx context.Context, serviceName, serviceVersion, otelEndpoint, environment string, logJSON bool) (func(context.Context) error, error) {
	if environment == "" {
		environment = "development"
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
			semconv.ServiceInstanceID(uuid.New().String()),
			attribute.String("deployment.environment", environment),
		),
	)
	if err != nil {
		return nil, err
	}

	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(otelEndpoint),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithTimeout(30*time.Second),
	)
	if err != nil {
		return nil, err
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	otel.SetTracerProvider(tracerProvider)

	metricExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(otelEndpoint),
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithTimeout(30*time.Second),
	)
	if err != nil {
		return nil, err
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(
			metricExporter,
			metric.WithInterval(3*time.Second),
		)),
		metric.WithResource(res),
	)

	otel.SetMeterProvider(meterProvider)

	logExporter, err := otlploggrpc.New(ctx,
		otlploggrpc.WithEndpoint(otelEndpoint),
		otlploggrpc.WithInsecure(),
		otlploggrpc.WithTimeout(30*time.Second),
	)
	if err != nil {
		return nil, err
	}

	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		sdklog.WithResource(res),
	)

	global.SetLoggerProvider(loggerProvider)

	otelslogHandler := otelslog.NewHandler(
		serviceName,
		otelslog.WithLoggerProvider(loggerProvider),
		otelslog.WithSource(true),
	)

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

	logger := slog.New(multiHandler{
		handlers: []slog.Handler{consoleHandler, otelslogHandler},
		service:  serviceName,
	})
	slog.SetDefault(logger)

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
