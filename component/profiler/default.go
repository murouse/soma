package profiler

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/murouse/logo/attr"
)

func Default() *Config {
	return &Config{
		Port:              1491,
		ShutdownTimeout:   time.Second,
		ReadHeaderTimeout: time.Minute,
		Logger:            slog.Default().With(attr.Component("profiler")),
	}
}

func DefaultWith(opts ...Option) (*Config, error) {
	cfg := Default()
	if err := cfg.Apply(opts...); err != nil {
		return nil, fmt.Errorf("apply options: %w", err)
	}
	return cfg, nil
}
