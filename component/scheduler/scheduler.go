package scheduler

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/murouse/logo/attr"
)

// Scheduler является оберткой над планировщиком задач.
type Scheduler struct {
	scheduler gocron.Scheduler
	logger    *slog.Logger
}

func New(cfg *Config) (*Scheduler, error) {
	if cfg.LoggingJobEnabled {
		cfg.GlobalJobOptions = append(cfg.GlobalJobOptions, gocron.WithEventListeners( // todo в default
			gocron.BeforeJobRuns(func(jobID uuid.UUID, jobName string) { // лог перед стартом задачи
				slog.Debug("job started", attr.JobName(jobName), attr.JobID(jobID.String()))
			}),

			gocron.AfterJobRuns(func(jobID uuid.UUID, jobName string) { // лог после успешного выполнения задачи
				slog.Debug("job completed", attr.JobName(jobName), attr.JobID(jobID.String()))
			}),

			gocron.AfterJobRunsWithError(func(jobID uuid.UUID, jobName string, err error) { // лог если задача упала с ошибкой
				slog.Error("job failed", attr.JobName(jobName), attr.JobID(jobID.String()), attr.Error(err))
			}),
		))
	}

	cfg.SchedulerOptions = append(cfg.SchedulerOptions, gocron.WithGlobalJobOptions(cfg.GlobalJobOptions...)) // переопределяем

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

		cfg.Logger.Debug("job created", attr.JobName(j.Name()), attr.JobID(j.ID().String()))
	}

	return &Scheduler{
		scheduler: s,
		logger:    cfg.Logger,
	}, nil
}

func (s *Scheduler) Prepare(ctx context.Context) error {
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
