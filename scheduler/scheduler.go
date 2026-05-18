package scheduler

import (
	"context"
	"fmt"

	"github.com/go-co-op/gocron/v2"
)

type Scheduler struct {
	scheduler gocron.Scheduler
}

func New(cfg *Config) (*Scheduler, error) {
	// create a scheduler
	s, err := gocron.NewScheduler(cfg.SchedulerOptions...)
	if err != nil {
		return nil, fmt.Errorf("new scheduler: %w", err)
	}

	// add a jobs to the scheduler
	for _, job := range cfg.Jobs {
		j, err := s.NewJob(
			job.Definition,
			job.Task,
			job.Options...,
		)
		if err != nil {
			return nil, fmt.Errorf("new job: %w", err)
		}

		fmt.Printf("job id %s created\n", j.ID())
	}

	return &Scheduler{
		scheduler: s,
	}, nil
}

func (s *Scheduler) PreRun(_ context.Context) error {
	return nil
}

func (s *Scheduler) Run(ctx context.Context) error {
	s.scheduler.Start()
	<-ctx.Done()
	return ctx.Err()
}

func (s *Scheduler) Shutdown(ctx context.Context) error {
	if err := s.scheduler.StopJobsWithContext(ctx); err != nil {
		return fmt.Errorf("stop jobs: %w", err)
	}

	if err := s.scheduler.ShutdownWithContext(ctx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	return nil
}
