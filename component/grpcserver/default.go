package grpcserver

import (
	"fmt"
	"time"
)

func Default() *Config {
	return &Config{
		Port:                1482,
		HealthServerEnabled: true,
		UnaryTimeout:        time.Minute,
	}
}

func DefaultWith(opts ...Option) (*Config, error) {
	cfg := Default()
	if err := cfg.Apply(opts...); err != nil {
		return nil, fmt.Errorf("apply options: %w", err)
	}
	return cfg, nil
}
