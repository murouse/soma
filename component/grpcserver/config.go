package grpcserver

import (
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/grpc"
)

type Config struct {
	Port                int
	ReflectionEnabled   bool
	HealthServerEnabled bool
	Services            []Service
	ServerOptions       []grpc.ServerOption // применяются после встроенных; их интерцепторы выполняются внутри встроенных
	UnaryTimeout        time.Duration       // дедлайн по умолчанию для запросов без клиентского дедлайна; 0 — без него
	Logger              *slog.Logger        // nil — slog.Default()
}

func (c *Config) Apply(opts ...Option) error {
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return fmt.Errorf("apply option: %w", err)
		}
	}

	return nil
}

type Option func(*Config) error

func WithPort(port int) Option {
	return func(config *Config) error {
		config.Port = port
		return nil
	}
}

func WithReflectionEnabled(enabled bool) Option {
	return func(config *Config) error {
		config.ReflectionEnabled = enabled
		return nil
	}
}

func WithHealthServerEnabled(enabled bool) Option {
	return func(config *Config) error {
		config.HealthServerEnabled = enabled
		return nil
	}
}

// WithServices регистрирует gRPC-сервисы. Если сервис также реализует grpcgateway.Service,
// gateway подхватит его автоматически.
func WithServices(services ...Service) Option {
	return func(config *Config) error {
		config.Services = append(config.Services, services...)
		return nil
	}
}

func WithServerOptions(opts ...grpc.ServerOption) Option {
	return func(config *Config) error {
		config.ServerOptions = append(config.ServerOptions, opts...)
		return nil
	}
}

func WithUnaryTimeout(timeout time.Duration) Option {
	return func(config *Config) error {
		config.UnaryTimeout = timeout
		return nil
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(config *Config) error {
		config.Logger = logger
		return nil
	}
}
