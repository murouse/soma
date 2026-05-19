package soma

import (
	"context"
	"fmt"
	"time"

	grpcgateway "github.com/murouse/soma/grpc-gateway"
	grpcserver "github.com/murouse/soma/grpc-server"

	"github.com/murouse/soma/profiler"
	"github.com/murouse/soma/scheduler"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"
)

// Entrypoint - точка входа
type Entrypoint struct {
	processes []Process
}

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

// NewEntrypoint создает точку входа
func NewEntrypoint(opts ...EntrypointOption) (*Entrypoint, error) {
	cfg, err := buildEntrypointConfig(opts...)
	if err != nil {
		return nil, fmt.Errorf("build config: %w", err)
	}

	processes := cfg.customProcesses

	// Scheduler
	if lo.IsNotNil(cfg.scheduler) {
		schdl, err := scheduler.New(cfg.scheduler)
		if err != nil {
			return nil, fmt.Errorf("scheduler new: %w", err)
		}
		processes = append(processes, schdl)
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
			cfg.grpcServer.Params.Port,
		)
		processes = append(processes, grpcGateway)
	}

	// Profiler
	if lo.IsNotNil(cfg.profiler) {
		prof := profiler.New(cfg.profiler)
		processes = append(processes, prof)
	}

	return &Entrypoint{
		processes: processes,
	}, nil
}

// Run запускает фреймворк из точки входа
func (e *Entrypoint) Run(ctx context.Context) error {
	// Фаза PreRun: выполняем строго последовательно до запуска горутин
	for _, process := range e.processes {
		if err := process.PreRun(ctx); err != nil {
			return fmt.Errorf("process pre run: %w", err)
		}
	}

	// Создаем errgroup с контекстом, который отменится, если один из процессов упадет
	eg, groupCtx := errgroup.WithContext(ctx)

	// Фаза Run: запускаем все процессы конкурентно
	for _, process := range e.processes {
		eg.Go(func() error {
			if err := process.Run(groupCtx); err != nil {
				return fmt.Errorf("process run: %w", err)
			}
			return nil
		})
	}

	// Фаза Shutdown: перехватываем отмену контекста оркестратора
	go func() {
		<-groupCtx.Done() // Ждем, пока главный ctx отменится или любой процесс вернет ошибку

		// Для шатдауна нужен СВОЙ контекст с таймаутом,
		// так как groupCtx на этом этапе уже гарантированно закрыт (Done)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Гасим процессы в обратном порядке (LIFO)
		for i := len(e.processes) - 1; i >= 0; i-- {
			if err := e.processes[i].Shutdown(shutdownCtx); err != nil {
				fmt.Printf("failed to shutdown process at index %d: %v\n", i, err)
			}
		}
	}()

	return eg.Wait()
}
