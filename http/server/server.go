package server

import (
	"context"
	"net/http"

	"tldw/config"
)

type Server struct {
	cfg  config.HTTPConfig
	http *http.Server
}

// New creates a new Server.
func New(cfg config.HTTPConfig) *Server {
	return &Server{cfg: cfg}
}

// Run starts the server and listens for incoming requests.
func (s *Server) Run(api http.Handler) error {
	s.http = &http.Server{
		Addr:           s.cfg.Host + ":" + s.cfg.Port,
		ReadTimeout:    s.cfg.ReadTimeout,
		WriteTimeout:   s.cfg.WriteTimeout,
		MaxHeaderBytes: 1 << 20,
		Handler:        api,
	}

	return s.http.ListenAndServe()
}

// Shutdown gracefully shuts down the server without interrupting any active connections.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}
