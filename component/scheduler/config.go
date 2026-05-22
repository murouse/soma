package scheduler

import (
	"fmt"
	"log/slog"

	"github.com/go-co-op/gocron/v2"
)

type Config struct {
	Jobs             []Job
	SchedulerOptions []gocron.SchedulerOption
	Logger           *slog.Logger
}

type Job struct {
	Definition gocron.JobDefinition
	Task       gocron.Task
	Options    []gocron.JobOption
}

func (c *Config) Apply(opts ...Option) error {
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return fmt.Errorf("apply option: %w", err)
		}
	}

	return nil
}

type Option func(*Config) error

func WithJobs(jobs ...Job) Option {
	return func(config *Config) error {
		config.Jobs = append(config.Jobs, jobs...)
		return nil
	}
}

func WithSchedulerOptions(opts ...gocron.SchedulerOption) Option {
	return func(config *Config) error {
		config.SchedulerOptions = append(config.SchedulerOptions, opts...)
		return nil
	}
}

func WithResetSchedulerOptions(opts ...gocron.SchedulerOption) Option {
	return func(config *Config) error {
		config.SchedulerOptions = opts
		return nil
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(config *Config) error {
		config.Logger = logger
		return nil
	}
}
