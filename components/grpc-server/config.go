package grpcserver

import (
	"google.golang.org/grpc"
)

type Config struct {
	Port              int
	ReflectionEnabled bool
	Impls             []ImplementationAdapter
	ServerOptions     []grpc.ServerOption
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
