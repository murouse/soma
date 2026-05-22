package httpserver

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/murouse/logo"
)

func Default(port int, handler http.Handler) *Config {
	return &Config{
		Port:              port,
		Handler:           handler,
		TLSConfig:         nil,
		ReadTimeout:       0,
		ReadHeaderTimeout: 0,
		WriteTimeout:      0,
		IdleTimeout:       0,
		MaxHeaderBytes:    0,
		ShutdownTimeout:   0,
		Middlewares: []func(next http.Handler) http.Handler{
			middleware.Recoverer,
			middleware.Logger,
		},
		Logger: slog.Default().With(logo.Component("http-server")),
	}
}

func DefaultWith(port int, handler http.Handler, opts ...Option) (*Config, error) {
	cfg := Default(port, handler)
	if err := cfg.Apply(opts...); err != nil {
		return nil, fmt.Errorf("apply options: %w", err)
	}
	return cfg, nil
}
