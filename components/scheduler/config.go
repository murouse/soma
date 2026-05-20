package scheduler

import (
	"github.com/go-co-op/gocron/v2"
)

type Config struct {
	Jobs             []Job
	SchedulerOptions []gocron.SchedulerOption
}

type Job struct {
	Definition gocron.JobDefinition
	Task       gocron.Task
	Options    []gocron.JobOption
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
