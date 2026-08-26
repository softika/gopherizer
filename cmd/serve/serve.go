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
	"go.opentelemetry.io/otel"

	"github.com/softika/gopherizer/api"
	"github.com/softika/gopherizer/config"
	"github.com/softika/gopherizer/database"
	"github.com/softika/gopherizer/pkg/logx"
	"github.com/softika/gopherizer/pkg/otelx"
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
	// Config is read before the logger is built, because the logger's level
	// comes from it. A failure here therefore reports through slog's default
	// handler, which is the only case in the process that does.
	cfg, err := config.New()
	if err != nil {
		slog.Error("failed to read config", "error", err)
		os.Exit(1)
	}

	// Tracing is initialised before the logger so the extractor below has a
	// provider to read from, and before the pool so queries are instrumented.
	flushTraces, err := otelx.Init(context.Background(), cfg.App, cfg.Tracing)
	if err != nil {
		slog.Error("failed to initialise tracing", "error", err)
		os.Exit(1)
	}

	// TraceAttrs is registered alongside ContextIds rather than instead of it:
	// correlation ids are present on every request, trace ids only on sampled
	// ones. Naming both is required, since an explicit extractor replaces the
	// default rather than adding to it.
	slog.SetDefault(logx.New(cfg.App, slogging.WithExtractor(
		slogging.ContextIds,
		otelx.TraceAttrs,
	)))

	db, err := database.New(cfg.Database, database.WithQueryTracer(otel.GetTracerProvider()))
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	router := api.NewRouter(cfg, db)

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

	// Spans are batched, so the exporter is flushed once the server has
	// drained. Without this the last requests before a deploy are the ones
	// that never reach the backend -- exactly the ones worth having.
	if err = flushTraces(ctx); err != nil {
		slog.Error("failed to flush traces", "error", err)
	}

	slog.Info("Graceful shutdown completed.")

}
