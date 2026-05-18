package grpcserver

import (
	"google.golang.org/grpc"
)

type Config struct {
	Params        Params
	Impls         []ImplementationAdapter
	ServerOptions []grpc.ServerOption
}

type Params struct {
	Port              int
	ReflectionEnabled bool
}

type Option func(*Config) error

func WithPort(port int) Option {
	return func(config *Config) error {
		config.Params.Port = port
		return nil
	}
}

func WithReflection(enabled bool) Option {
	return func(config *Config) error {
		config.Params.ReflectionEnabled = enabled
		return nil
	}
}

func WithAdapters(impls []ImplementationAdapter) Option {
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
