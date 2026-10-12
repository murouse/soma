package scheduler

import (
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"
)

func Default() *Config {
	return &Config{
		SchedulerOptions:  []gocron.SchedulerOption{gocron.WithLocation(time.Local)},
		LoggingJobEnabled: true,
	}
}

func DefaultWith(opts ...Option) (*Config, error) {
	cfg := Default()
	if err := cfg.Apply(opts...); err != nil {
		return nil, fmt.Errorf("apply options: %w", err)
	}
	return cfg, nil
}
