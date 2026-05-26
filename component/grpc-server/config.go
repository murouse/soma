package grpcserver

import (
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/grpc"
)

type Config struct {
	Port                int
	ReflectionEnabled   bool
	HealthServerEnabled bool
	Impls               []ImplementationAdapter
	ServerOptions       []grpc.ServerOption
	UnaryTimeout        time.Duration
	Logger              *slog.Logger
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

func WithReflection(enabled bool) Option {
	return func(config *Config) error {
		config.ReflectionEnabled = enabled
		return nil
	}
}

func WithAdapters(impls ...ImplementationAdapter) Option {
	return func(config *Config) error {
		config.Impls = append(config.Impls, impls...)
		return nil
	}
}

func WithServerOptions(opts ...grpc.ServerOption) Option {
	return func(config *Config) error {
		config.ServerOptions = append(config.ServerOptions, opts...)
		return nil
	}
}

func WithResetServerOptions(opts ...grpc.ServerOption) Option {
	return func(config *Config) error {
		config.ServerOptions = opts
		return nil
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(config *Config) error {
		config.Logger = logger
		return nil
	}
}

func WithHealthServer(enabled bool) Option {
	return func(config *Config) error {
		config.HealthServerEnabled = enabled
		return nil
	}
}

func WithUnaryTimeout(timeout time.Duration) Option {
	return func(config *Config) error {
		config.UnaryTimeout = timeout
		return nil
	}
}
