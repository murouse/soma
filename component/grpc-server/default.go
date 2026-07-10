package grpcserver

import (
	"fmt"
	"log/slog"
	"time"

	bufprotovalidate "buf.build/go/protovalidate"
	mwprotovalidate "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/protovalidate"
	"github.com/murouse/golgi/attr"
	"github.com/murouse/soma/accessor/middleware/grpc-interceptor"
	"google.golang.org/grpc"
)

func Default() (*Config, error) {
	validator, err := bufprotovalidate.New()
	if err != nil {
		return nil, fmt.Errorf("initialize validator: %w", err)
	}

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
				mwprotovalidate.UnaryServerInterceptor(validator),
			),
			grpc.ChainStreamInterceptor(), // todo
		},
		UnaryTimeout: time.Minute,
		Logger:       slog.Default().With(attr.Component("grpc-server")),
	}, nil
}

func DefaultWith(opts ...Option) (*Config, error) {
	cfg, err := Default()
	if err != nil {
		return nil, fmt.Errorf("default: %w", err)
	}

	if err = cfg.Apply(opts...); err != nil {
		return nil, fmt.Errorf("apply options: %w", err)
	}
	return cfg, nil
}
