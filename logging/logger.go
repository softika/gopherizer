package logging

import (
	"log/slog"
	"os"
	"sync"
)

var (
	logger *slog.Logger

	once sync.Once
)

func Logger() *slog.Logger {
	once.Do(func() {
		logLevel := slog.LevelInfo

		env := os.Getenv("ENVIRONMENT")
		switch env {
		case "local", "development":
			logLevel = slog.LevelDebug
		case "production":
			logLevel = slog.LevelError
		}

		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: logLevel,
		}))

	})
	return logger
}
