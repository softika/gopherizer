package serve

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/softika/slogging"

	"github.com/softika/gopherizer/api"
	"github.com/softika/gopherizer/config"
)

func Run() {
	log := slogging.Slogger()
	cfg, err := config.New()
	if err != nil {
		log.Error("failed to read config", "error", err)
		os.Exit(1)
	}

	router := api.NewRouter(cfg)

	srv := api.NewServer(cfg.Http)

	// Start the server in a goroutine.
	go func() {
		log.Info("starting the server...", "address", cfg.Http.Host+":"+cfg.Http.Port)
		if err = srv.Run(router); !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed to run", "error", err)
			os.Exit(1)
		}
		log.Info("stopped serving new connections.")
	}()

	// Wait for interrupt signal to gracefully shut down the server with a timeout.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()
	<-ctx.Done()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if err = srv.Shutdown(ctx); err != nil {
		log.Error("server shutdown error", "error", err)
	}
	log.Info("Graceful shutdown completed.")

}
