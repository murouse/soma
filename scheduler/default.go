package scheduler

import "github.com/go-co-op/gocron/v2"

func Default() *Config {
	return &Config{
		Jobs:             []Job{},
		SchedulerOptions: []gocron.SchedulerOption{},
	}
}
