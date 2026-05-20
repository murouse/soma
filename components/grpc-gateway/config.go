package grpcgateway

import (
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

type Config struct {
	Port            int
	ShutdownTimeout time.Duration
	ServeMuxOptions []runtime.ServeMuxOption
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

func WithServeMuxOptions(opts ...runtime.ServeMuxOption) Option {
	return func(config *Config) error {
		config.ServeMuxOptions = append(config.ServeMuxOptions, opts...)
		return nil
	}
}

func WithResetServeMuxOptions(opts ...runtime.ServeMuxOption) Option {
	return func(config *Config) error {
		config.ServeMuxOptions = opts
		return nil
	}
}
