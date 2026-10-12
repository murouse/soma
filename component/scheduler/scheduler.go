package scheduler

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/murouse/golgi/attr"
)

// Scheduler является оберткой над планировщиком задач.
type Scheduler struct {
	scheduler gocron.Scheduler
	logger    *slog.Logger
}

func New(cfg *Config) (*Scheduler, error) {
	logger := cmp.Or(cfg.Logger, slog.Default()).With(attr.Component("scheduler"))

	globalJobOptions := slices.Clone(cfg.GlobalJobOptions)
	if cfg.LoggingJobEnabled {
		globalJobOptions = append(globalJobOptions, loggingEventListeners(logger))
	}

	// Встроенный логгер — первым, чтобы пользовательский gocron.WithLogger мог его заменить.
	options := append([]gocron.SchedulerOption{gocron.WithLogger(logger)}, cfg.SchedulerOptions...)
	options = append(options, gocron.WithGlobalJobOptions(globalJobOptions...))

	s, err := gocron.NewScheduler(options...)
	if err != nil {
		return nil, fmt.Errorf("new scheduler: %w", err)
	}

	for _, job := range cfg.Jobs {
		j, err := s.NewJob(job.Definition, job.Task, job.Options...)
		if err != nil {
			return nil, fmt.Errorf("new job: %w", err)
		}

		logger.Debug("job created", attr.JobName(j.Name()), attr.JobID(j.ID()))
	}

	return &Scheduler{
		scheduler: s,
		logger:    logger,
	}, nil
}

func loggingEventListeners(logger *slog.Logger) gocron.JobOption {
	return gocron.WithEventListeners(
		gocron.BeforeJobRuns(func(jobID uuid.UUID, jobName string) {
			logger.Debug("job started", attr.JobName(jobName), attr.JobID(jobID))
		}),
		gocron.AfterJobRuns(func(jobID uuid.UUID, jobName string) {
			logger.Debug("job completed", attr.JobName(jobName), attr.JobID(jobID))
		}),
		gocron.AfterJobRunsWithError(func(jobID uuid.UUID, jobName string, err error) {
			logger.Error("job failed", attr.JobName(jobName), attr.JobID(jobID), attr.Error(err))
		}),
	)
}

func (s *Scheduler) Name() string {
	return "scheduler"
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
