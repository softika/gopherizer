package serve

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"time"

	"tldw/config"
	"tldw/http/server"
	"tldw/logger"
)

func Run() {
	log := logger.Logger()
	cfg, err := config.New()
	if err != nil {
		log.Error("failed to read config", "error", err)
		os.Exit(1)
	}

	api := initApi(cfg)

	srv := server.New(cfg)

	// Start the server in a goroutine.
	go func() {
		log.Info("starting the server...", "address", cfg.Http.Host+":"+cfg.Http.Port)
		if err = srv.Run(api); !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed to run", "error", err)
			os.Exit(1)
		}
		log.Info("stopped serving new connections.")
	}()

	// Wait for interrupt signal to gracefully shutdown the server with a timeout.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	<-ctx.Done()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if err = srv.Shutdown(ctx); err != nil {
		log.Error("server shutdown error", "error", err)
	}
	log.Info("Graceful shutdown completed.")

}
