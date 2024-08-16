package logging

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogger(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		level slog.Level
	}{
		{
			name:  "log info",
			level: slog.LevelInfo,
		},
		{
			name:  "log debug",
			level: slog.LevelDebug,
		},
		{
			name:  "log warn",
			level: slog.LevelWarn,
		},
		{
			name:  "log error",
			level: slog.LevelError,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := Get()
			assert.NotNil(t, got)

			got.WithGroup("logger test")
			switch tc.level {
			case slog.LevelInfo:
				got.Info("test info")
			case slog.LevelDebug:
				got.Debug("test debug")
			case slog.LevelWarn:
				got.Warn("test warn")
			case slog.LevelError:
				err := errors.New("test error message")
				got.Error("general message", "error", err)
			}
		})
	}
}
