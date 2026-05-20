package grpcserver

import (
	"fmt"

	"google.golang.org/grpc"
)

func Default() *Config {
	return &Config{
		Port:              1482,
		ReflectionEnabled: true,
		Impls:             []ImplementationAdapter{},
		ServerOptions:     []grpc.ServerOption{},
	}
}

func (c *Config) Apply(opts ...Option) error {
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return fmt.Errorf("apply option: %w", err)
		}
	}

	return nil
}

func DefaultWith(opts ...Option) (*Config, error) {
	cfg := Default()
	if err := cfg.Apply(opts...); err != nil {
		return nil, fmt.Errorf("apply options: %w", err)
	}
	return cfg, nil
}
