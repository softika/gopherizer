package serve

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/softika/slogging"

	"github.com/softika/gopherizer/api"
	"github.com/softika/gopherizer/config"
)

// Run starts the http server with graceful shutdown option.
func Run() {
	slog.SetDefault(slogging.Slogger()) // inject default logger

	cfg, err := config.New()
	if err != nil {
		slog.Error("failed to read config", "error", err)
		os.Exit(1)
	}

	router := api.NewRouter(cfg)

	srv := api.NewServer(cfg.Http)

	// Start the server in a goroutine.
	go func() {
		slog.Info("starting the server...", "address", cfg.Http.Host+":"+cfg.Http.Port)
		if err = srv.Run(router); !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed to run", "error", err)
			os.Exit(1)
		}
		slog.Info("stopped serving new connections.")
	}()

	// Wait for interrupt signal to gracefully shut down the server with a timeout.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()
	<-ctx.Done()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if err = srv.Shutdown(ctx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}
	slog.Info("Graceful shutdown completed.")

}
