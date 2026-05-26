package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type Server struct {
	port            int
	shutdownTimeout time.Duration
	logger          *slog.Logger

	*http.Server
}

func New(cfg *Config) *Server {
	handler := cfg.Handler
	for i := len(cfg.Middlewares) - 1; i >= 0; i-- {
		handler = cfg.Middlewares[i](handler)
	}

	return &Server{
		port:            cfg.Port,
		shutdownTimeout: cfg.ShutdownTimeout,
		logger:          cfg.Logger,
		Server: &http.Server{
			Addr:              fmt.Sprintf(":%d", cfg.Port),
			Handler:           handler,
			TLSConfig:         cfg.TLSConfig,
			ReadTimeout:       cfg.ReadTimeout,
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       cfg.IdleTimeout,
			MaxHeaderBytes:    cfg.MaxHeaderBytes,
		},
	}
}

func (s *Server) Prepare(_ context.Context) error {
	return nil
}

func (s *Server) Run(ctx context.Context) error {
	s.logger.InfoContext(ctx, "serving http at", slog.Int("port", s.port), slog.String("addr", "http://localhost:"+strconv.Itoa(s.port)))

	if err := s.Server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen and serve: %w", err)
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, s.shutdownTimeout)
	defer cancel()

	if err := s.Server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	return nil
}
