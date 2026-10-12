package soma

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os/signal"
	"slices"
	"strconv"
	"syscall"
	"time"

	"github.com/murouse/golgi/attr"
	"github.com/murouse/soma/accessor/closer"
	"github.com/murouse/soma/component/grpcgateway"
	"github.com/murouse/soma/component/grpcserver"
	"github.com/murouse/soma/component/httpserver"
	"github.com/murouse/soma/component/profiler"
	"github.com/murouse/soma/component/scheduler"
	"golang.org/x/sync/errgroup"
)

// Entrypoint управляет жизненным циклом всех зарегистрированных процессов.
//
// Жизненный цикл:
//  1. Prepare — последовательно, в порядке регистрации, с общим таймаутом prepareTimeout.
//     Если Prepare одного из процессов падает, уже подготовленные процессы останавливаются
//     (Shutdown в обратном порядке) и ресурсы закрываются.
//  2. Run — конкурентно. Фаза длится, пока не придет SIGINT/SIGTERM, не отменится родительский
//     контекст, один из процессов не вернет ошибку или все процессы не завершатся сами.
//  3. Drain — процессы, реализующие Drainer, перестают сообщать о готовности, затем пауза
//     shutdownDelay, чтобы балансировщик успел снять экземпляр с трафика.
//  4. Shutdown — последовательно, в обратном порядке (LIFO), затем ожидание завершения всех Run.
//     На всё вместе отводится shutdownTimeout.
//  5. Close — закрытие технических ресурсов (WithClosers) с отдельным бюджетом closeTimeout,
//     чтобы зависший процесс не лишал ресурсы шанса закрыться.
//
// Максимальное время остановки: shutdownDelay + shutdownTimeout + closeTimeout.
type Entrypoint struct {
	processes       []Process
	prepareTimeout  time.Duration
	shutdownDelay   time.Duration
	shutdownTimeout time.Duration
	closeTimeout    time.Duration
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
// Здесь же компоненты собираются вместе: сами пакеты компонентов друг о друге не знают.
func NewEntrypoint(opts ...EntrypointOption) (*Entrypoint, error) {
	cfg, err := DefaultWith(opts...)
	if err != nil {
		return nil, fmt.Errorf("default config: %w", err)
	}

	cfg.inheritLogger()
	logger := cmp.Or(cfg.logger, slog.Default())

	processes := slices.Clone(cfg.customProcesses)

	if cfg.scheduler != nil {
		sch, err := scheduler.New(cfg.scheduler)
		if err != nil {
			return nil, fmt.Errorf("scheduler new: %w", err)
		}
		processes = append(processes, sch)
	}

	if cfg.grpcServer != nil {
		processes = append(processes, grpcserver.New(cfg.grpcServer))
	}

	// Наличие grpcServer гарантирует Validate
	if cfg.grpcGateway != nil {
		processes = append(processes, grpcgateway.New(
			cfg.grpcGateway,
			gatewayServices(cfg.grpcServer.Services),
			net.JoinHostPort("localhost", strconv.Itoa(cfg.grpcServer.Port)),
		))
	}

	if cfg.profiler != nil {
		processes = append(processes, profiler.New(cfg.profiler))
	}

	for _, httpServer := range cfg.httpServers {
		processes = append(processes, httpserver.New(httpServer))
	}

	return &Entrypoint{
		processes:       processes,
		prepareTimeout:  cfg.prepareTimeout,
		shutdownDelay:   cfg.shutdownDelay,
		shutdownTimeout: cfg.shutdownTimeout,
		closeTimeout:    cfg.closeTimeout,
		closer:          closer.New(logger.With(attr.Component("closer")), cfg.closures...),
		logger:          logger.With(attr.Component("entrypoint")),
	}, nil
}

// gatewayServices отбирает gRPC-сервисы, которые умеют регистрировать хендлеры grpc-gateway.
func gatewayServices(services []grpcserver.Service) []grpcgateway.Service {
	var gatewayServices []grpcgateway.Service
	for _, service := range services {
		if gatewayService, ok := service.(grpcgateway.Service); ok {
			gatewayServices = append(gatewayServices, gatewayService)
		}
	}
	return gatewayServices
}

// Run запускает все зарегистрированные процессы конкурентно и ожидает их завершения.
// Возвращает объединение ошибок всех фаз: Prepare, Run, Shutdown и Close.
func (e *Entrypoint) Run(ctx context.Context) error {
	// Подписываемся на сигналы до Prepare: SIGTERM во время долгой инициализации тоже
	// должен приводить к корректной остановке, а не к убийству процесса.
	ctx, stopSignals := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	// ctx к моменту остановки уже отменен, но его значения (атрибуты логов, трейсы) нужны и там.
	stopCtx := context.WithoutCancel(ctx)

	// Фаза Prepare (последовательная)
	e.logger.DebugContext(ctx, "prepare phase")
	prepared, err := e.prepare(ctx)
	if err != nil {
		e.logger.ErrorContext(ctx, "prepare failed, rolling back", attr.Error(err))
		// Run ни у кого не запускался: снимать с балансировки и ждать нечего (runDone == nil).
		return errors.Join(err, e.shutdown(stopCtx, prepared, nil))
	}

	// Фаза Run
	e.logger.DebugContext(ctx, "run phase")
	// runCtx отменяется при сигнале (через ctx), при первой ошибке любого Run
	// и когда все Run завершились (errgroup отменяет контекст при возврате Wait).
	runEG, runCtx := errgroup.WithContext(ctx)
	for _, process := range e.processes {
		runEG.Go(func() error {
			if err := process.Run(runCtx); err != nil && !errors.Is(err, context.Canceled) {
				return fmt.Errorf("run %s: %w", processName(process), err)
			}
			return nil
		})
	}

	// Буфер 1: если Shutdown не дождется Run, горутина не зависнет на отправке.
	runDone := make(chan error, 1)
	go func() { runDone <- runEG.Wait() }()

	<-runCtx.Done()
	// Отписываемся от сигналов сразу: повторный Ctrl+C во время зависшей остановки
	// завершит процесс стандартным обработчиком.
	stopSignals()
	e.logger.InfoContext(ctx, "initiating shutdown", slog.Any("cause", context.Cause(runCtx)))

	e.drain(stopCtx)
	return e.shutdown(stopCtx, e.processes, runDone)
}

// prepare последовательно готовит процессы и возвращает те, чей Prepare завершился успешно.
func (e *Entrypoint) prepare(ctx context.Context) ([]Process, error) {
	prepareCtx, cancel := context.WithTimeout(ctx, e.prepareTimeout)
	defer cancel()

	for i, process := range e.processes {
		if err := process.Prepare(prepareCtx); err != nil {
			return e.processes[:i], fmt.Errorf("prepare %s: %w", processName(process), err)
		}
	}

	return e.processes, nil
}

// drain снимает процессы с балансировки и ждет shutdownDelay, чтобы балансировщик успел
// это заметить. Запросы в это время продолжают обслуживаться.
func (e *Entrypoint) drain(ctx context.Context) {
	e.logger.DebugContext(ctx, "drain phase")

	for _, process := range slices.Backward(e.processes) {
		if drainer, ok := process.(Drainer); ok {
			drainer.Drain()
		}
	}

	if e.shutdownDelay > 0 {
		e.logger.InfoContext(ctx, "waiting before shutdown", slog.Duration("delay", e.shutdownDelay))
		time.Sleep(e.shutdownDelay)
	}
}

// shutdown останавливает процессы в обратном порядке (LIFO), дожидается завершения их Run
// (если runDone != nil) и закрывает ресурсы. Ошибка любого шага не прерывает остальные.
func (e *Entrypoint) shutdown(ctx context.Context, processes []Process, runDone <-chan error) error {
	e.logger.DebugContext(ctx, "shutdown phase")

	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, e.shutdownTimeout)
	defer shutdownCancel()

	var errs []error
	for _, process := range slices.Backward(processes) {
		if err := process.Shutdown(shutdownCtx); err != nil {
			errs = append(errs, fmt.Errorf("shutdown %s: %w", processName(process), err))
		}
	}

	if runDone != nil {
		// Процесс, нарушающий контракт (игнорирует и ctx, и Shutdown), не должен
		// подвешивать остановку навсегда.
		select {
		case err := <-runDone:
			errs = append(errs, err)
		case <-shutdownCtx.Done():
			errs = append(errs, fmt.Errorf("wait processes: %w", shutdownCtx.Err()))
		}
	}

	// Отдельный бюджет: даже если процессы съели весь shutdownTimeout,
	// ресурсы (БД, брокеры) все равно получают шанс закрыться.
	closeCtx, closeCancel := context.WithTimeout(ctx, e.closeTimeout)
	defer closeCancel()

	e.logger.DebugContext(ctx, "close phase")
	if err := e.closer.Close(closeCtx); err != nil {
		errs = append(errs, fmt.Errorf("close resources: %w", err))
	}

	return errors.Join(errs...)
}

// processName возвращает имя процесса для логов и ошибок: Name(), если процесс
// реализует Named, иначе имя типа.
func processName(process Process) string {
	if named, ok := process.(Named); ok {
		return named.Name()
	}
	return fmt.Sprintf("%T", process)
}
