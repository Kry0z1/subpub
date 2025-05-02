package slogdiscard

import (
	"context"
	"log/slog"
)

type Handler struct{}

func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return false
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	return nil
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return h
}

func NewDiscardHandler() slog.Handler {
	return &Handler{}
}
