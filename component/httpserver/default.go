package httpserver

import (
	"fmt"
	"net/http"
	"time"
)

func Default(port int, handler http.Handler) *Config {
	return &Config{
		Name:    "http-server",
		Port:    port,
		Handler: handler,
		// Без ReadHeaderTimeout клиент может бесконечно держать соединения,
		// присылая заголовки по байту (slowloris).
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}
}

func DefaultWith(port int, handler http.Handler, opts ...Option) (*Config, error) {
	cfg := Default(port, handler)
	if err := cfg.Apply(opts...); err != nil {
		return nil, fmt.Errorf("apply options: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
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
