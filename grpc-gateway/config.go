package grpcgateway

import (
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

type Config struct {
	Params          Params
	ServeMuxOptions []runtime.ServeMuxOption
}

type Params struct {
	Port            int
	ShutdownTimeout time.Duration
}

type Option func(*Config) error

func WithParams(params Params) Option {
	return func(config *Config) error {
		config.Params = params
		return nil
	}
}

func WithServeMuxOptions(opts ...runtime.ServeMuxOption) Option {
	return func(config *Config) error {
		config.ServeMuxOptions = append(config.ServeMuxOptions, opts...)
		return nil
	}
}
