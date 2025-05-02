// Partially yoinked from https://github.com/GolangLessons/sso/blob/guide-version/internal/lib/logger/handlers/slogpretty/slogpretty.go

package slogpretty

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"log/slog"
)

var (
	reset  = "\033[0m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	blue   = "\033[34m"
	purple = "\033[35m"
	cyan   = "\033[36m"
	gray   = "\033[37m"
	white  = "\033[97m"
)

func getColoredString(s string, c string) string {
	return c + s + reset
}

type Handler struct {
	slog.Handler
	attrs []slog.Attr
	l     *log.Logger
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	level := r.Level.String() + ":"

	switch r.Level {
	case slog.LevelDebug:
		level = getColoredString(level, purple)
	case slog.LevelError:
		level = getColoredString(level, red)
	case slog.LevelInfo:
		level = getColoredString(level, blue)
	case slog.LevelWarn:
		level = getColoredString(level, red)
	}

	fields := make(map[string]any, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		fields[a.Key] = a.Value.Any()

		return true
	})

	for _, a := range h.attrs {
		fields[a.Key] = a.Value.Any()
	}

	var b []byte
	var err error

	if len(fields) > 0 {
		b, err = json.MarshalIndent(fields, "", "  ")
		if err != nil {
			return err
		}
	}
	timeStr := r.Time.Format("[15:05:05.000]")
	msg := getColoredString(r.Message, cyan)

	h.l.Println(
		timeStr,
		level,
		msg,
		getColoredString(string(b), white),
	)

	return nil
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{
		Handler: h.Handler,
		attrs:   attrs,
		l:       h.l,
	}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		l:       h.l,
		attrs:   h.attrs,
		Handler: h.Handler.WithGroup(name),
	}
}

func NewPrettyHandler(output io.Writer, opts *slog.HandlerOptions) slog.Handler {
	return &Handler{
		l:       log.New(output, "", 0),
		Handler: slog.NewJSONHandler(output, opts),
	}
}
