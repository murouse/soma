package soma

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/murouse/soma/accessor/closer"
	grpcgateway "github.com/murouse/soma/components/grpc-gateway"
	grpcserver "github.com/murouse/soma/components/grpc-server"

	"github.com/murouse/soma/components/profiler"
	"github.com/murouse/soma/components/scheduler"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"
)

// Entrypoint управляет жизненным циклом всех зарегистрированных процессов.
type Entrypoint struct {
	processes       []Process
	shutdownTimeout time.Duration
	prepareTimeout  time.Duration
	closer          Closer // Process.Shutdown для жизненного цикла серверов, а Closer для технических ресурсов
}

type Closer interface {
	Close(ctx context.Context) error
}

// NewEntrypointRun инициализирует и запускает точку входа в рамках текущего контекста.
func NewEntrypointRun(ctx context.Context, opts ...EntrypointOption) error {
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
		grpcServer := grpcserver.New(cfg.grpcServer)
		processes = append(processes, grpcServer)
	}

	// GRPC-Gateway Server
	if lo.IsNotNil(cfg.grpcGateway) {
		grpcGateway := grpcgateway.New(
			cfg.grpcGateway,
			lo.Map(cfg.grpcServer.Impls, func(impl grpcserver.ImplementationAdapter, _ int) grpcgateway.ImplementationAdapter {
				return impl
			}),
			cfg.grpcServer.Port,
		)
		processes = append(processes, grpcGateway)
	}

	// Profiler
	if lo.IsNotNil(cfg.profiler) {
		prf := profiler.New(cfg.profiler)
		processes = append(processes, prf)
	}

	return &Entrypoint{
		processes:       processes,
		prepareTimeout:  cfg.prepareTimeout,
		shutdownTimeout: cfg.shutdownTimeout,
		closer:          closer.New(cfg.closures...),
	}, nil
}

// Run запускает все зарегистрированные процессы конкурентно и ожидает их завершения.
func (e *Entrypoint) Run(ctx context.Context) error {
	// Фаза Prepare (последовательная)
	prepareCtx, prepareCancel := context.WithTimeout(ctx, e.prepareTimeout)
	defer prepareCancel()
	for _, process := range e.processes {
		if err := process.Prepare(prepareCtx); err != nil {
			return fmt.Errorf("process prepare: %w", err)
		}
	}

	// Фаза Run
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
		fmt.Printf("received signal: %v, initiating shutdown\n", sig)
		runCancel()
	case <-runCtx.Done():
		fmt.Println("context canceled, initiating shutdown")
	}

	// Фаза Shutdown (последовательная, LIFO)
	// Завершаем процессы
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), e.shutdownTimeout)
	defer shutdownCancel()
	for i := len(e.processes) - 1; i >= 0; i-- {
		if err := e.processes[i].Shutdown(shutdownCtx); err != nil {
			fmt.Printf("process shutdown error: %v\n", err)
		}
	}

	// Ждем завершения всех процессов
	err := runEG.Wait()

	// Закрываем ресурсы
	if closeErr := e.closer.Close(shutdownCtx); err != nil {
		err = errors.Join(err, closeErr)
	}

	return err
}
