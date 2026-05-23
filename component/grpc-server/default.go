package grpcserver

import (
	"fmt"
	"log/slog"

	"github.com/murouse/logo/attr"
	"github.com/murouse/soma/accessor/middleware/grpc-interceptor"
	"google.golang.org/grpc"
)

func Default() *Config {
	interceptorManager := grpcinterceptor.NewManager(slog.Default().With(attr.Component("interceptor")))

	return &Config{
		Port:                1482,
		ReflectionEnabled:   true,
		HealthServerEnabled: true,
		Impls:               []ImplementationAdapter{},
		ServerOptions: []grpc.ServerOption{
			grpc.ChainUnaryInterceptor(
				interceptorManager.LoggingAttrsUnaryInterceptor(),
				interceptorManager.LoggingUnaryInterceptor(),
				interceptorManager.RecoveryUnaryInterceptor(),
			),
			grpc.ChainStreamInterceptor(), // todo
		},
		Logger: slog.Default().With(attr.Component("grpc-server")),
	}
}

func DefaultWith(opts ...Option) (*Config, error) {
	cfg := Default()
	if err := cfg.Apply(opts...); err != nil {
		return nil, fmt.Errorf("apply options: %w", err)
	}
	return cfg, nil
}
