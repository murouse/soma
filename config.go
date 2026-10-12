package soma

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/murouse/soma/component/grpcgateway"
	"github.com/murouse/soma/component/grpcserver"
	"github.com/murouse/soma/component/httpserver"
	"github.com/murouse/soma/component/profiler"
	"github.com/murouse/soma/component/scheduler"
)

type EntrypointConfig struct {
	scheduler       *scheduler.Config
	grpcServer      *grpcserver.Config
	grpcGateway     *grpcgateway.Config
	profiler        *profiler.Config
	httpServers     []*httpserver.Config
	customProcesses []Process // процессы, добавляемые пользователем через WithProcesses
	prepareTimeout  time.Duration
	shutdownDelay   time.Duration
	shutdownTimeout time.Duration
	closeTimeout    time.Duration
	closures        []func() error
	logger          *slog.Logger // nil — slog.Default()
}

// ErrGatewayRequiresGRPCServer возвращается, если gRPC-Gateway включен без gRPC-сервера.
var ErrGatewayRequiresGRPCServer = errors.New("grpc gateway requires grpc server: add WithGRPCServer")

func (c *EntrypointConfig) Validate() error {
	if c.grpcServer == nil && c.grpcGateway != nil {
		return ErrGatewayRequiresGRPCServer
	}

	return nil
}

func (c *EntrypointConfig) Apply(opts ...EntrypointOption) error {
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return fmt.Errorf("apply option: %w", err)
		}
	}

	return nil
}

// inheritLogger передает логгер точки входа компонентам, у которых нет собственного.
// Без логгера точки входа компоненты сами берут slog.Default().
func (c *EntrypointConfig) inheritLogger() {
	if c.logger == nil {
		return
	}

	if c.scheduler != nil {
		c.scheduler.Logger = cmp.Or(c.scheduler.Logger, c.logger)
	}
	if c.grpcServer != nil {
		c.grpcServer.Logger = cmp.Or(c.grpcServer.Logger, c.logger)
	}
	if c.grpcGateway != nil {
		c.grpcGateway.Logger = cmp.Or(c.grpcGateway.Logger, c.logger)
	}
	if c.profiler != nil {
		c.profiler.Logger = cmp.Or(c.profiler.Logger, c.logger)
	}
	for _, httpServer := range c.httpServers {
		httpServer.Logger = cmp.Or(httpServer.Logger, c.logger)
	}
}

type EntrypointOption func(*EntrypointConfig) error

// WithLogger задает логгер точки входа. Компоненты без собственного логгера наследуют его.
func WithLogger(logger *slog.Logger) EntrypointOption {
	return func(e *EntrypointConfig) error {
		e.logger = logger
		return nil
	}
}

// WithProcesses регистрирует пользовательские фоновые процессы, реализующие интерфейс Process.
func WithProcesses(processes ...Process) EntrypointOption {
	return func(cfg *EntrypointConfig) error {
		cfg.customProcesses = append(cfg.customProcesses, processes...)
		return nil
	}
}

// WithPrepareTimeout задает максимальное время на выполнение фазы инициализации Prepare для всех компонентов.
func WithPrepareTimeout(prepareTimeout time.Duration) EntrypointOption {
	return func(cfg *EntrypointConfig) error {
		cfg.prepareTimeout = prepareTimeout
		return nil
	}
}

// WithShutdownDelay задает паузу между Drain (health переходит в NOT_SERVING) и Shutdown.
// За это время балансировщик (например, Kubernetes) перестает слать трафик на экземпляр,
// и остановка проходит без ошибок у клиентов.
func WithShutdownDelay(shutdownDelay time.Duration) EntrypointOption {
	return func(cfg *EntrypointConfig) error {
		cfg.shutdownDelay = shutdownDelay
		return nil
	}
}

// WithShutdownTimeout задает максимальное время на остановку процессов: Shutdown всех процессов
// и ожидание завершения их Run.
func WithShutdownTimeout(shutdownTimeout time.Duration) EntrypointOption {
	return func(cfg *EntrypointConfig) error {
		cfg.shutdownTimeout = shutdownTimeout
		return nil
	}
}

// WithCloseTimeout задает максимальное время на закрытие ресурсов из WithClosers.
// Отсчитывается отдельно от WithShutdownTimeout, после остановки процессов.
func WithCloseTimeout(closeTimeout time.Duration) EntrypointOption {
	return func(cfg *EntrypointConfig) error {
		cfg.closeTimeout = closeTimeout
		return nil
	}
}

// WithClosers регистрирует ресурсы (бд, брокеры, соединения), которые должны быть закрыты после остановки серверов.
func WithClosers(closers ...io.Closer) EntrypointOption {
	return func(cfg *EntrypointConfig) error {
		for _, closer := range closers {
			cfg.closures = append(cfg.closures, closer.Close)
		}
		return nil
	}
}

// WithScheduler активирует и настраивает планировщик задач gocron на основе переданных опций.
func WithScheduler(opts ...scheduler.Option) EntrypointOption {
	return func(c *EntrypointConfig) error {
		cfg, err := scheduler.DefaultWith(opts...)
		if err != nil {
			return fmt.Errorf("scheduler: %w", err)
		}
		c.scheduler = cfg
		return nil
	}
}

// WithGRPCServer активирует и настраивает основной gRPC-сервер приложения.
func WithGRPCServer(opts ...grpcserver.Option) EntrypointOption {
	return func(c *EntrypointConfig) error {
		cfg, err := grpcserver.DefaultWith(opts...)
		if err != nil {
			return fmt.Errorf("grpc server: %w", err)
		}
		c.grpcServer = cfg
		return nil
	}
}

// WithGRPCGateway активирует HTTP-шлюз для проксирования REST-запросов в gRPC-сервисы. Требует наличия
// gRPC-сервера и подхватывает те его сервисы, которые реализуют grpcgateway.Service.
func WithGRPCGateway(opts ...grpcgateway.Option) EntrypointOption {
	return func(c *EntrypointConfig) error {
		cfg, err := grpcgateway.DefaultWith(opts...)
		if err != nil {
			return fmt.Errorf("grpc gateway: %w", err)
		}
		c.grpcGateway = cfg
		return nil
	}
}

// WithProfiler активирует служебный HTTP-сервер со стандартными эндпоинтами pprof для профилирования.
func WithProfiler(opts ...profiler.Option) EntrypointOption {
	return func(c *EntrypointConfig) error {
		cfg, err := profiler.DefaultWith(opts...)
		if err != nil {
			return fmt.Errorf("profiler: %w", err)
		}
		c.profiler = cfg
		return nil
	}
}

// WithHTTPServer добавляет экземпляр кастомного HTTP-сервера с заданным обработчиком и портом.
func WithHTTPServer(handler http.Handler, port int, opts ...httpserver.Option) EntrypointOption {
	return func(c *EntrypointConfig) error {
		cfg, err := httpserver.DefaultWith(port, handler, opts...)
		if err != nil {
			return fmt.Errorf("http server: %w", err)
		}
		c.httpServers = append(c.httpServers, cfg)
		return nil
	}
}
