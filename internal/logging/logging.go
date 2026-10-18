package logging

import (
	"log/slog"
	"os"
)

func New(level string) *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))
}

func Enter(l *slog.Logger, name string, kv ...any) {
}

func Exit(l *slog.Logger, name string, kv ...any) {
}

func Fail(l *slog.Logger, name string, err error, kv ...any) {
}
