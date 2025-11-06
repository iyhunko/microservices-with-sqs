package logger

import (
	"log/slog"
	"os"
)

// InitJSONLogger configures and sets the default slog logger to use JSON format.
// This ensures all log output is structured in JSON format for better parsing and analysis.
func InitJSONLogger() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))
}
