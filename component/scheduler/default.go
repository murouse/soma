package scheduler

import (
	"fmt"
	"log/slog"

	"github.com/go-co-op/gocron/v2"
	"github.com/murouse/logo/attr"
)

func Default() *Config {
	return &Config{
		Jobs:             []Job{},
		SchedulerOptions: []gocron.SchedulerOption{},
		Logger:           slog.Default().With(attr.Component("scheduler")),
	}
}

func DefaultWith(opts ...Option) (*Config, error) {
	cfg := Default()
	if err := cfg.Apply(opts...); err != nil {
		return nil, fmt.Errorf("apply options: %w", err)
	}
	return cfg, nil
}
