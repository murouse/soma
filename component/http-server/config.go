package httpserver

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type Config struct {
	Port    int
	Handler http.Handler

	TLSConfig         *tls.Config
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int

	ShutdownTimeout time.Duration
	Middlewares     []func(next http.Handler) http.Handler
	Logger          *slog.Logger
}

func (c *Config) Apply(opts ...Option) error {
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return fmt.Errorf("apply option: %w", err)
		}
	}

	return nil
}

type Option func(*Config) error

func WithTLSConfig(cfg *tls.Config) Option {
	return func(c *Config) error {
		c.TLSConfig = cfg
		return nil
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(config *Config) error {
		config.Logger = logger
		return nil
	}
}

func WithMiddleware(middleware ...func(next http.Handler) http.Handler) Option {
	return func(c *Config) error {
		c.Middlewares = append(c.Middlewares, middleware...)
		return nil
	}
}

func WithResetMiddleware(middleware ...func(next http.Handler) http.Handler) Option {
	return func(c *Config) error {
		c.Middlewares = middleware
		return nil
	}
}
