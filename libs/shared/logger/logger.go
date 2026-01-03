package logger

import (
	"context"
	"log/slog"
	"os"
)

type contextHandler struct {
	slog.Handler
	service string
}

func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	r.AddAttrs(slog.String("service", h.service))
	return h.Handler.Handle(ctx, r)
}

func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{h.Handler.WithAttrs(attrs), h.service}
}

func (h contextHandler) WithGroup(group string) slog.Handler {
	return contextHandler{h.Handler.WithGroup(group), h.service}
}

func Init(serviceName string, logJSON bool) {
	var handler slog.Handler

	if logJSON {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}

	logger := slog.New(contextHandler{handler, serviceName})
	slog.SetDefault(logger)
}
