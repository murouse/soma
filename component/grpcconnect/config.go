package grpcconnect

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect/v2/connecthttp"
)

type Config struct {
	Port            int
	ShutdownTimeout time.Duration
	HandlerOptions  []connecthttp.Option // опции connecthttp.Mount: кодеки, сжатие, лимиты
	TLSConfig       *tls.Config          // TLS к gRPC-серверу; nil — h2c без шифрования
	Logger          *slog.Logger         // nil — slog.Default()
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

func WithPort(port int) Option {
	return func(config *Config) error {
		config.Port = port
		return nil
	}
}

func WithShutdownTimeout(shutdownTimeout time.Duration) Option {
	return func(config *Config) error {
		config.ShutdownTimeout = shutdownTimeout
		return nil
	}
}

func WithHandlerOptions(opts ...connecthttp.Option) Option {
	return func(config *Config) error {
		config.HandlerOptions = append(config.HandlerOptions, opts...)
		return nil
	}
}

func WithTLSConfig(tlsConfig *tls.Config) Option {
	return func(config *Config) error {
		config.TLSConfig = tlsConfig
		return nil
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(config *Config) error {
		config.Logger = logger
		return nil
	}
}
