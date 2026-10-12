package grpcgateway

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/cors"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

type Config struct {
	Port            int
	ShutdownTimeout time.Duration
	ServeMuxOptions []runtime.ServeMuxOption
	DialOptions     []grpc.DialOption // применяются после insecure по умолчанию: TLS к gRPC-серверу задается здесь
	HTTPHandlers    []httpHandlerConfig

	HealthEndpointEnabled bool
	HealthEndpointPath    string

	SwaggerEnabled bool
	SwaggerDir     string // каталог со спецификацией на диске
	SwaggerPath    string // HTTP-путь, по которому она отдается

	CORSEnabled bool
	CORSOptions cors.Options

	Logger *slog.Logger // nil — slog.Default()
}

type httpHandlerConfig struct {
	pattern string
	handler http.Handler
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

func WithShutdownTimeout(shutdownTimeout time.Duration) Option {
	return func(config *Config) error {
		config.ShutdownTimeout = shutdownTimeout
		return nil
	}
}

func WithServeMuxOptions(opts ...runtime.ServeMuxOption) Option {
	return func(config *Config) error {
		config.ServeMuxOptions = append(config.ServeMuxOptions, opts...)
		return nil
	}
}

func WithDialOptions(opts ...grpc.DialOption) Option {
	return func(config *Config) error {
		config.DialOptions = append(config.DialOptions, opts...)
		return nil
	}
}

func WithHTTPHandler(pattern string, handler http.Handler) Option {
	return func(config *Config) error {
		config.HTTPHandlers = append(config.HTTPHandlers, httpHandlerConfig{pattern, handler})
		return nil
	}
}

func WithHealthEndpointEnabled(enabled bool) Option {
	return func(config *Config) error {
		config.HealthEndpointEnabled = enabled
		return nil
	}
}

func WithHealthEndpointPath(path string) Option {
	return func(config *Config) error {
		config.HealthEndpointPath = path
		return nil
	}
}

func WithSwaggerEnabled(enabled bool) Option {
	return func(config *Config) error {
		config.SwaggerEnabled = enabled
		return nil
	}
}

func WithSwaggerDir(dir string) Option {
	return func(config *Config) error {
		config.SwaggerDir = dir
		return nil
	}
}

func WithSwaggerPath(path string) Option {
	return func(config *Config) error {
		config.SwaggerPath = path
		return nil
	}
}

func WithCORSEnabled(enabled bool) Option {
	return func(config *Config) error {
		config.CORSEnabled = enabled
		return nil
	}
}

func WithCORSOptions(options cors.Options) Option {
	return func(config *Config) error {
		config.CORSOptions = options
		return nil
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(config *Config) error {
		config.Logger = logger
		return nil
	}
}
