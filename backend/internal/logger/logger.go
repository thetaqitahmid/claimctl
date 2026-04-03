package logger

import (
	"log/slog"
	"os"
)

// Init initializes the global logger based on the environment.
// If env is "production", it uses a JSON handler.
// Otherwise (e.g. "development"), it uses a Text handler.
func Init(env string) {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
}
