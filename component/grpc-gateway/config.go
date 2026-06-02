package grpcgateway

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/cors"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

type Config struct {
	Port               int
	ShutdownTimeout    time.Duration
	ServeMuxOptions    []runtime.ServeMuxOption
	HttpHandlers       []httpHandlerConfig
	HealthEndpointPath *string
	Swagger            *swaggerConfig
	Cors               cors.Options

	Logger *slog.Logger
}

type swaggerConfig struct {
	dirOnDisk string // "./deploy/swagger"
	httpPath  string //  "/docs/"
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

func WithResetServeMuxOptions(opts ...runtime.ServeMuxOption) Option {
	return func(config *Config) error {
		config.ServeMuxOptions = opts
		return nil
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(config *Config) error {
		config.Logger = logger
		return nil
	}
}

func WithHttpHandler(pattern string, handler http.Handler) Option {
	return func(config *Config) error {
		config.HttpHandlers = append(config.HttpHandlers, httpHandlerConfig{pattern, handler})
		return nil
	}
}

func WithHealthEndpoint(path *string) Option {
	return func(config *Config) error {
		config.HealthEndpointPath = path
		return nil
	}
}

func WithSwagger(dirOnDisk string, httpPath string) Option {
	return func(config *Config) error {
		config.Swagger = &swaggerConfig{dirOnDisk, httpPath}
		return nil
	}
}

func WithCors(cors cors.Options) Option {
	return func(config *Config) error {
		config.Cors = cors
		return nil
	}
}
