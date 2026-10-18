package logging

import (
	"context"
	"log/slog"
	"os"
)

type discardHandler struct{}

func (discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (discardHandler) WithAttrs([]slog.Attr) slog.Handler        { return discardHandler{} }
func (discardHandler) WithGroup(string) slog.Handler             { return discardHandler{} }

func New(level string) *slog.Logger {
	switch level {
	case "", "discard":
		return slog.New(discardHandler{})
	}
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl}))
}

func Enter(l *slog.Logger, name string, kv ...any) {
	l.Debug(name, prependPhase("enter", kv)...)
}

func Exit(l *slog.Logger, name string, kv ...any) {
	l.Debug(name, prependPhase("exit", kv)...)
}

func Fail(l *slog.Logger, name string, err error, kv ...any) {
	args := append(prependPhase("fail", kv), "err", err.Error())
	l.Error(name, args...)
}

func prependPhase(phase string, kv []any) []any {
	return append([]any{"phase", phase}, kv...)
}
