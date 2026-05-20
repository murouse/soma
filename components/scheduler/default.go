package scheduler

import (
	"fmt"

	"github.com/go-co-op/gocron/v2"
)

func Default() *Config {
	return &Config{
		Jobs:             []Job{},
		SchedulerOptions: []gocron.SchedulerOption{},
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
