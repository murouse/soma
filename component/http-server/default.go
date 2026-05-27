package httpserver

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/murouse/logo/attr"
	httpmiddleware "github.com/murouse/soma/accessor/middleware/http-middleware"
)

func Default(port int, handler http.Handler) *Config {
	interceptorManager := httpmiddleware.NewManager(slog.Default().With(attr.Component("middleware")))

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
			interceptorManager.LoggingAttrsMiddleware,
			interceptorManager.LoggingMiddleware,
			interceptorManager.RecoveryMiddleware,
		},
		Logger: slog.Default().With(attr.Component("http-server")),
	}
}

func DefaultWith(port int, handler http.Handler, opts ...Option) (*Config, error) {
	cfg := Default(port, handler)
	if err := cfg.Apply(opts...); err != nil {
		return nil, fmt.Errorf("apply options: %w", err)
	}
	return cfg, nil
}

func NewDefaultWith(port int, handler http.Handler, opts ...Option) (*Server, error) {
	cfg, err := DefaultWith(port, handler, opts...)
	if err != nil {
		return nil, fmt.Errorf("default with: %w", err)
	}
	return New(cfg), nil
}
