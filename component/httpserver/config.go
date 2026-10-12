package httpserver

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

var (
	ErrNilHandler = errors.New("http handler is nil")
	// ErrDefaultServeMux: net/http/pprof (его импортирует профайлер) в init() регистрирует
	// /debug/pprof в http.DefaultServeMux, и такой сервер отдал бы профайлер наружу.
	ErrDefaultServeMux = errors.New("http.DefaultServeMux is not allowed: it exposes /debug/pprof")
)

type Config struct {
	Name    string // для логов и ошибок; у встроенных компонентов — их имя
	Host    string // пустой — слушать на всех интерфейсах
	Port    int
	Handler http.Handler

	TLSConfig         *tls.Config
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int

	ShutdownTimeout time.Duration                          // 0 — ограничение только контекстом Shutdown
	Middlewares     []func(next http.Handler) http.Handler // выполняются внутри встроенных (логирование, recovery)
	Logger          *slog.Logger                           // nil — slog.Default()
}

func (c *Config) Apply(opts ...Option) error {
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return fmt.Errorf("apply option: %w", err)
		}
	}

	return nil
}

func (c *Config) Validate() error {
	if c.Handler == nil {
		return ErrNilHandler
	}
	if c.Handler == http.DefaultServeMux {
		return ErrDefaultServeMux
	}
	return nil
}

type Option func(*Config) error

func WithName(name string) Option {
	return func(c *Config) error {
		c.Name = name
		return nil
	}
}

func WithHost(host string) Option {
	return func(c *Config) error {
		c.Host = host
		return nil
	}
}

// WithTLSConfig включает TLS. Сертификаты должны быть заданы в cfg.Certificates или cfg.GetCertificate.
func WithTLSConfig(cfg *tls.Config) Option {
	return func(c *Config) error {
		c.TLSConfig = cfg
		return nil
	}
}

func WithReadTimeout(timeout time.Duration) Option {
	return func(c *Config) error {
		c.ReadTimeout = timeout
		return nil
	}
}

func WithReadHeaderTimeout(timeout time.Duration) Option {
	return func(c *Config) error {
		c.ReadHeaderTimeout = timeout
		return nil
	}
}

func WithWriteTimeout(timeout time.Duration) Option {
	return func(c *Config) error {
		c.WriteTimeout = timeout
		return nil
	}
}

func WithIdleTimeout(timeout time.Duration) Option {
	return func(c *Config) error {
		c.IdleTimeout = timeout
		return nil
	}
}

func WithMaxHeaderBytes(maxHeaderBytes int) Option {
	return func(c *Config) error {
		c.MaxHeaderBytes = maxHeaderBytes
		return nil
	}
}

func WithShutdownTimeout(timeout time.Duration) Option {
	return func(c *Config) error {
		c.ShutdownTimeout = timeout
		return nil
	}
}

func WithMiddlewares(middlewares ...func(next http.Handler) http.Handler) Option {
	return func(c *Config) error {
		c.Middlewares = append(c.Middlewares, middlewares...)
		return nil
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(c *Config) error {
		c.Logger = logger
		return nil
	}
}
