package server

import (
	"context"
	"net/http"

	"tldw/config"
)

type Server struct {
	cfg  *config.Config
	http *http.Server
}

// New creates a new Server.
func New(cfg *config.Config) *Server {
	return &Server{cfg: cfg}
}

// Run starts the server and listens for incoming requests.
func (s *Server) Run(api http.Handler) error {
	s.http = &http.Server{
		Addr:           s.cfg.Http.Host + ":" + s.cfg.Http.Port,
		ReadTimeout:    s.cfg.Http.ReadTimeout,
		WriteTimeout:   s.cfg.Http.WriteTimeout,
		MaxHeaderBytes: 1 << 20,
		Handler:        api,
	}

	return s.http.ListenAndServe()
}

// Shutdown gracefully shuts down the server without interrupting any active connections.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}
