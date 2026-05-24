package scheduler

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/murouse/logo/attr"
)

func Default() *Config {
	logger := slog.Default().With(attr.Component("scheduler"))

	return &Config{
		Jobs: []Job{},
		SchedulerOptions: []gocron.SchedulerOption{
			gocron.WithLocation(time.Local),
			gocron.WithLogger(logger), // внутрибиблиотечный
		},
		Logger:            logger,
		GlobalJobOptions:  []gocron.JobOption{},
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
