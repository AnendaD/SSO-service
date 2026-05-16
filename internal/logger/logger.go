package logger

import (
	"log/slog"
	"os"
)

func New(level slog.Level) *slog.Logger {
	opt := &slog.HandlerOptions{Level: level}
	return slog.New(slog.NewJSONHandler(os.Stdout, opt))
}
