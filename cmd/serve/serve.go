package serve

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/softika/slogging"

	"github.com/softika/gopherizer/api"
	"github.com/softika/gopherizer/config"
	"github.com/softika/gopherizer/database"
)

// shutdownSignals are the signals that trigger a graceful shutdown.
//
// SIGTERM is the one that matters in practice: it is what `docker stop` and
// Kubernetes send. SIGKILL is deliberately absent because it can never be
// caught or handled by a process.
var shutdownSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}

// notifyShutdown returns a context that is cancelled on the first shutdown signal.
func notifyShutdown() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), shutdownSignals...)
}

// httpServer drains in-flight requests and stops serving.
type httpServer interface {
	Shutdown(context.Context) error
}

// resource is a dependency released once the server has drained.
type resource interface {
	Close() error
}

// shutdown drains the http server, then releases the database pool.
//
// The pool is closed even when the server fails to drain, so a shutdown error
// cannot leak connections. Failures are reported together rather than dropped.
func shutdown(ctx context.Context, srv httpServer, db resource) error {
	var errs []error

	if err := srv.Shutdown(ctx); err != nil {
		errs = append(errs, fmt.Errorf("failed to shut down http server: %w", err))
	}

	if err := db.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close database pool: %w", err))
	}

	return errors.Join(errs...)
}

// Run starts the http server with graceful shutdown option.
func Run() {
	slog.SetDefault(slogging.Slogger()) // inject default logger

	cfg, err := config.New()
	if err != nil {
		slog.Error("failed to read config", "error", err)
		os.Exit(1)
	}

	// The pool is a process-wide singleton, so this is the same instance the
	// router builds its repositories on. Taking a handle here is what lets the
	// shutdown path release it.
	db := database.New(cfg.Database)

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

	// Wait for a shutdown signal to gracefully shut down the server with a timeout.
	ctx, stop := notifyShutdown()
	defer stop()
	<-ctx.Done()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if err = shutdown(ctx, srv, db); err != nil {
		slog.Error("graceful shutdown error", "error", err)
	}
	slog.Info("Graceful shutdown completed.")

}
