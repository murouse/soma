package profiler

import (
	"fmt"
	"time"
)

func Default() *Config {
	return &Config{
		Host:            "127.0.0.1", // только локально; в k8s доступен через kubectl port-forward
		Port:            1491,
		ShutdownTimeout: time.Second,
	}
}

func DefaultWith(opts ...Option) (*Config, error) {
	cfg := Default()
	if err := cfg.Apply(opts...); err != nil {
		return nil, fmt.Errorf("apply options: %w", err)
	}
	return cfg, nil
}
