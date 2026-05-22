package grpcserver

import (
	"fmt"
	"log/slog"

	"github.com/murouse/logo"
	"google.golang.org/grpc"
)

func Default() *Config {
	return &Config{
		Port:              1482,
		ReflectionEnabled: true,
		Impls:             []ImplementationAdapter{},
		ServerOptions:     []grpc.ServerOption{},
		Logger:            slog.Default().With(logo.Component("grpc-server")),
	}
}

func DefaultWith(opts ...Option) (*Config, error) {
	cfg := Default()
	if err := cfg.Apply(opts...); err != nil {
		return nil, fmt.Errorf("apply options: %w", err)
	}
	return cfg, nil
}
