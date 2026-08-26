package logx_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/softika/slogging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/softika/gopherizer/config"
	"github.com/softika/gopherizer/pkg/logx"
)

func TestLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  config.AppConfig
		want slog.Level
	}{
		{"explicit debug", config.AppConfig{LogLevel: "debug"}, slog.LevelDebug},
		{"explicit info", config.AppConfig{LogLevel: "info"}, slog.LevelInfo},
		{"explicit warn", config.AppConfig{LogLevel: "warn"}, slog.LevelWarn},
		{"explicit error", config.AppConfig{LogLevel: "error"}, slog.LevelError},
		{"case insensitive", config.AppConfig{LogLevel: "WARN"}, slog.LevelWarn},

		{"local derives debug", config.AppConfig{Environment: "local"}, slog.LevelDebug},
		{"development derives debug", config.AppConfig{Environment: "development"}, slog.LevelDebug},
		{"staging derives info", config.AppConfig{Environment: "staging"}, slog.LevelInfo},

		// slogging maps production to Error, which discards startup, shutdown
		// and every readiness failure detail. This package deliberately does
		// not inherit that.
		{"production derives info, not error", config.AppConfig{Environment: "production"}, slog.LevelInfo},

		{"explicit level wins over environment", config.AppConfig{Environment: "local", LogLevel: "error"}, slog.LevelError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, logx.Level(tt.cfg))
		})
	}
}

func TestNewHonoursLevel(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	logger := logx.New(config.AppConfig{LogLevel: "error"}, slogging.WithWriter(buf))

	logger.Info("dropped")
	assert.Zero(t, buf.Len(), "info emitted at error level: %s", buf.String())

	logger.Error("kept")
	assert.NotZero(t, buf.Len(), "error record was dropped")
}

func TestNewStampsContextIds(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	logger := logx.New(config.AppConfig{LogLevel: "info"}, slogging.WithWriter(buf))

	ctx := context.WithValue(context.Background(), slogging.CorrelationIdKey, "corr-1")
	logger.InfoContext(ctx, "hello")

	var got map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, "corr-1", got[string(slogging.CorrelationIdKey)])
}

// TestNewSurvivesDerivedLoggers is the gopherizer-side guard for the slogging
// bug fixed in v1.1.0: a logger built with With must keep its context ids.
func TestNewStampsContextIdsOnDerivedLogger(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	logger := logx.New(config.AppConfig{LogLevel: "info"}, slogging.WithWriter(buf)).With("component", "db")

	ctx := context.WithValue(context.Background(), slogging.CorrelationIdKey, "corr-1")
	logger.InfoContext(ctx, "hello")

	var got map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, "db", got["component"])
	assert.Equal(t, "corr-1", got[string(slogging.CorrelationIdKey)])
}
