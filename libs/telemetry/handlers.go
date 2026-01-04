package telemetry

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/trace"
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
	attrs := h.buildTraceAttributes(ctx, r.Time)
	r.AddAttrs(attrs...)

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

func (h multiHandler) buildTraceAttributes(ctx context.Context, timestamp time.Time) []slog.Attr {
	spanCtx := trace.SpanContextFromContext(ctx)
	attrs := []slog.Attr{
		slog.String("service.name", h.service),
		slog.String("timestamp", timestamp.Format(time.RFC3339Nano)),
	}

	if spanCtx.HasTraceID() {
		attrs = append(attrs, slog.String("trace_id", spanCtx.TraceID().String()))
	}
	if spanCtx.HasSpanID() {
		attrs = append(attrs, slog.String("span_id", spanCtx.SpanID().String()))
	}

	return attrs
}

type jsonBodyHandler struct {
	handler slog.Handler
	service string
}

func (h jsonBodyHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h jsonBodyHandler) Handle(ctx context.Context, r slog.Record) error {
	spanCtx := trace.SpanContextFromContext(ctx)

	logData := h.buildLogData(r, spanCtx)

	jsonBytes, err := json.Marshal(logData)
	if err != nil {
		return err
	}

	newRecord := slog.NewRecord(r.Time, r.Level, string(jsonBytes), r.PC)

	attrs := h.buildStructuredMetadata(spanCtx, r.Time)
	newRecord.AddAttrs(attrs...)

	return h.handler.Handle(ctx, newRecord)
}

func (h jsonBodyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return jsonBodyHandler{
		handler: h.handler.WithAttrs(attrs),
		service: h.service,
	}
}

func (h jsonBodyHandler) WithGroup(group string) slog.Handler {
	return jsonBodyHandler{
		handler: h.handler.WithGroup(group),
		service: h.service,
	}
}

func (h jsonBodyHandler) buildLogData(r slog.Record, spanCtx trace.SpanContext) map[string]any {
	logData := map[string]any{
		"level":        r.Level.String(),
		"msg":          r.Message,
		"service.name": h.service,
		"timestamp":    r.Time.Format(time.RFC3339Nano),
	}

	if spanCtx.HasTraceID() {
		logData["trace_id"] = spanCtx.TraceID().String()
	}
	if spanCtx.HasSpanID() {
		logData["span_id"] = spanCtx.SpanID().String()
	}

	r.Attrs(func(a slog.Attr) bool {
		logData[a.Key] = a.Value.Any()
		return true
	})

	return logData
}

func (h jsonBodyHandler) buildStructuredMetadata(spanCtx trace.SpanContext, timestamp time.Time) []slog.Attr {
	attrs := []slog.Attr{
		slog.String("service.name", h.service),
		slog.String("timestamp", timestamp.Format(time.RFC3339Nano)),
	}

	if spanCtx.HasTraceID() {
		attrs = append(attrs, slog.String("trace_id", spanCtx.TraceID().String()))
	}
	if spanCtx.HasSpanID() {
		attrs = append(attrs, slog.String("span_id", spanCtx.SpanID().String()))
	}

	return attrs
}
