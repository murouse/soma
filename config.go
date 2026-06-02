package soma

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	grpcgateway "github.com/murouse/soma/component/grpc-gateway"
	grpcserver "github.com/murouse/soma/component/grpc-server"
	httpserver "github.com/murouse/soma/component/http-server"
	"github.com/murouse/soma/component/profiler"
	"github.com/murouse/soma/component/scheduler"
)

type EntrypointConfig struct {
	scheduler       *scheduler.Config
	grpcServer      *grpcserver.Config
	grpcGateway     *grpcgateway.Config
	profiler        *profiler.Config
	customProcesses []Process // процессы, добавляемые пользователем через WithProcesses
	prepareTimeout  time.Duration
	shutdownTimeout time.Duration
	closures        []func() error
	logger          *slog.Logger
	httpServers     []httpserver.Config
}

func (c *EntrypointConfig) Validate() error {
	if c.grpcServer == nil && c.grpcGateway != nil {
		return fmt.Errorf("must provide either grpcServer or grpcGateway")
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

type EntrypointOption func(*EntrypointConfig) error

// WithLogger устанавливает кастомный экземпляр логгера slog.
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

// WithShutdownTimeout задает максимальное время на корректную остановку (Shutdown) и закрытие ресурсов.
func WithShutdownTimeout(shutdownTimeout time.Duration) EntrypointOption {
	return func(cfg *EntrypointConfig) error {
		cfg.shutdownTimeout = shutdownTimeout
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
			return err
		}
		c.scheduler = cfg
		return nil
	}
}

// WithGrpcServer активирует и настраивает основной gRPC-сервер приложения.
func WithGrpcServer(opts ...grpcserver.Option) EntrypointOption {
	return func(c *EntrypointConfig) error {
		cfg, err := grpcserver.DefaultWith(opts...)
		if err != nil {
			return err
		}
		c.grpcServer = cfg
		return nil
	}
}

// WithGrpcGateway активирует HTTP-шлюз для проксирования REST-запросов в gRPC-сервисы. Требует наличия gRPC-сервера.
func WithGrpcGateway(opts ...grpcgateway.Option) EntrypointOption {
	return func(c *EntrypointConfig) error {
		cfg, err := grpcgateway.DefaultWith(opts...)
		if err != nil {
			return err
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
			return err
		}
		c.profiler = cfg
		return nil
	}
}

// WithHttpServer добавляет экземпляр кастомного HTTP-сервера с заданным обработчиком и портом
func WithHttpServer(handler http.Handler, port int, opts ...httpserver.Option) EntrypointOption {
	return func(c *EntrypointConfig) error {
		cfg, err := httpserver.DefaultWith(port, handler, opts...)
		if err != nil {
			return err
		}
		c.httpServers = append(c.httpServers, *cfg)
		return nil
	}
}
