package soma

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/murouse/golgi/attr"
	"github.com/murouse/soma/accessor/closer"
	grpcgateway "github.com/murouse/soma/component/grpc-gateway"
	grpcserver "github.com/murouse/soma/component/grpc-server"
	httpserver "github.com/murouse/soma/component/http-server"

	"github.com/murouse/soma/component/profiler"
	"github.com/murouse/soma/component/scheduler"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"
)

// Entrypoint управляет жизненным циклом всех зарегистрированных процессов.
type Entrypoint struct {
	processes       []Process
	shutdownTimeout time.Duration
	prepareTimeout  time.Duration
	closer          Closer // Process.Shutdown для жизненного цикла серверов, а Closer для технических ресурсов
	logger          *slog.Logger
}

type Closer interface {
	Close(ctx context.Context) error
}

// Run инициализирует и запускает точку входа в рамках текущего контекста.
func Run(ctx context.Context, opts ...EntrypointOption) error {
	entrypoint, err := NewEntrypoint(opts...)
	if err != nil {
		return fmt.Errorf("new entrypoint: %w", err)
	}

	if err = entrypoint.Run(ctx); err != nil {
		return fmt.Errorf("entrypoint run: %w", err)
	}

	return nil
}

// NewEntrypoint создает новую точку входа на основе предоставленных опций.
func NewEntrypoint(opts ...EntrypointOption) (*Entrypoint, error) {
	cfg, err := DefaultWith(opts...)
	if err != nil {
		return nil, fmt.Errorf("default config: %w", err)
	}

	if err = cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	processes := cfg.customProcesses

	// Scheduler
	if lo.IsNotNil(cfg.scheduler) {
		sch, err := scheduler.New(cfg.scheduler)
		if err != nil {
			return nil, fmt.Errorf("scheduler new: %w", err)
		}
		processes = append(processes, sch)
	}

	// GRPC Server
	if lo.IsNotNil(cfg.grpcServer) {
		processes = append(processes, grpcserver.New(cfg.grpcServer))
	}

	// GRPC-Gateway Server
	if lo.IsNotNil(cfg.grpcGateway) {
		processes = append(processes, grpcgateway.New(
			cfg.grpcGateway,
			lo.Map(cfg.grpcServer.Impls, func(impl grpcserver.ImplementationAdapter, _ int) grpcgateway.ImplementationAdapter {
				return impl
			}),
			cfg.grpcServer.Port,
		))
	}

	// Profiler
	if lo.IsNotNil(cfg.profiler) {
		processes = append(processes, profiler.New(cfg.profiler))
	}

	// HTTP Servers
	for _, httpServer := range cfg.httpServers {
		processes = append(processes, httpserver.New(&httpServer))
	}

	return &Entrypoint{
		processes:       processes,
		prepareTimeout:  cfg.prepareTimeout,
		shutdownTimeout: cfg.shutdownTimeout,
		closer:          closer.New(cfg.closures...),
		logger:          cfg.logger,
	}, nil
}

// Run запускает все зарегистрированные процессы конкурентно и ожидает их завершения.
func (e *Entrypoint) Run(ctx context.Context) error {
	// Фаза Prepare (последовательная)
	e.logger.DebugContext(ctx, "prepare phase")
	prepareCtx, prepareCancel := context.WithTimeout(ctx, e.prepareTimeout)
	defer prepareCancel()
	for _, process := range e.processes {
		if err := process.Prepare(prepareCtx); err != nil {
			return fmt.Errorf("process prepare: %w", err)
		}
	}

	// Фаза Run
	e.logger.DebugContext(ctx, "run phase")
	runCtx, runCancel := context.WithCancel(ctx) // для того, чтобы отменить в случае получения сигнала
	defer runCancel()
	runEG, runCtx := errgroup.WithContext(runCtx) // ддя того, чтобы отменился в случае ошибки в одном из Run
	for _, process := range e.processes {
		runEG.Go(func() error {
			if err := process.Run(runCtx); err != nil && !errors.Is(err, context.Canceled) {
				return fmt.Errorf("process run: %w", err)
			}
			return nil
		})
	}

	// Ожидание либо ошибки, либо сигнала
	exitChan := make(chan os.Signal, 1)
	signal.Notify(exitChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-exitChan:
		e.logger.InfoContext(ctx, "received signal, initiating shutdown", slog.String("signal", sig.String()))
		runCancel()
	case <-runCtx.Done():
		e.logger.InfoContext(ctx, "context canceled, initiating shutdown")
	}

	// Фаза Shutdown (последовательная, LIFO)
	// Завершаем процессы
	e.logger.DebugContext(ctx, "shutdown phase")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), e.shutdownTimeout)
	defer shutdownCancel()
	for i := len(e.processes) - 1; i >= 0; i-- {
		if err := e.processes[i].Shutdown(shutdownCtx); err != nil {
			e.logger.ErrorContext(ctx, "process shutdown", attr.Error(err))
		}
	}

	// Ждем завершения всех процессов и закрываем ресурсы
	return errors.Join(runEG.Wait(), e.closer.Close(shutdownCtx))
}
