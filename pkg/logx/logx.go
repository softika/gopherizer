// Package logx builds the application logger.
//
// It is a thin policy layer over slogging: it decides the level from
// configuration and leaves handler construction to the library.
package logx

import (
	"log/slog"
	"strings"

	"github.com/softika/slogging"

	"github.com/softika/gopherizer/config"
)

// levels maps a configured name onto a slog level.
var levels = map[string]slog.Level{
	"debug": slog.LevelDebug,
	"info":  slog.LevelInfo,
	"warn":  slog.LevelWarn,
	"error": slog.LevelError,
}

// Level reports the level to log at.
//
// An explicit app.log_level wins. When it is unset the level is derived from
// the environment, so a local run is verbose and a deployed one is not.
//
// Note that production maps to Info, not Error. slogging's own ENVIRONMENT
// mapping uses Error, which discards startup, shutdown and the detail behind
// every readiness failure -- making production the environment you can see
// least about. Configuration decides the level here, and the default keeps
// operational records.
func Level(cfg config.AppConfig) slog.Level {
	if level, ok := levels[strings.ToLower(cfg.LogLevel)]; ok {
		return level
	}

	switch strings.ToLower(cfg.Environment) {
	case "local", "development":
		return slog.LevelDebug
	default:
		return slog.LevelInfo
	}
}

// New builds the application logger.
//
// Options are applied after the configured level, so a caller can override it;
// tests use slogging.WithWriter to capture output.
func New(cfg config.AppConfig, opts ...slogging.Option) *slog.Logger {
	base := []slogging.Option{slogging.WithLevel(Level(cfg))}

	return slog.New(slogging.NewHandler(append(base, opts...)...))
}
