package api

import (
	"context"
	"net/http"
	"time"

	"github.com/softika/gopherizer/config"
)

type Server struct {
	cfg  config.HTTPConfig
	http *http.Server
}

// NewServer creates a new Server.
func NewServer(cfg config.HTTPConfig) *Server {
	return &Server{cfg: cfg}
}

// defaultReadHeaderTimeout bounds header reads when none is configured, so the
// server is never left open to slow-header attacks by omission.
const defaultReadHeaderTimeout = 10 * time.Second

// httpServer builds the configured *http.Server.
func (s *Server) httpServer(api http.Handler) *http.Server {
	readHeaderTimeout := s.cfg.ReadHeaderTimeout
	if readHeaderTimeout <= 0 {
		readHeaderTimeout = defaultReadHeaderTimeout
	}

	return &http.Server{
		Addr:              s.cfg.Host + ":" + s.cfg.Port,
		ReadTimeout:       s.cfg.ReadTimeout,
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      s.cfg.WriteTimeout,
		IdleTimeout:       s.cfg.IdleTimeout,
		MaxHeaderBytes:    1 << 20,
		Handler:           api,
	}
}

// Run starts the server and listens for incoming requests.
func (s *Server) Run(api http.Handler) error {
	s.http = s.httpServer(api)

	return s.http.ListenAndServe()
}

// Shutdown gracefully shuts down the server without interrupting any active connections.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}
