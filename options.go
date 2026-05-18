package soma

import (
	"fmt"

	grpcgateway "github.com/murouse/soma/grpc-gateway"
	grpcserver "github.com/murouse/soma/grpc-server"

	"github.com/murouse/soma/profiler"
	"github.com/murouse/soma/scheduler"
)

// ===========
// Entrypoint
// ===========

type EntrypointOption func(*EntrypointConfig) error

func buildEntrypointConfig(opts ...EntrypointOption) (*EntrypointConfig, error) {
	// сначала заполняем дефолтом
	cfg := Default()

	// потом переопределяем
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, fmt.Errorf("apply option: %w", err)
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	return cfg, nil
}

// ===========
// Processes
// ===========

func WithProcesses(processes ...Process) EntrypointOption {
	return func(cfg *EntrypointConfig) error {
		cfg.customProcesses = append(cfg.customProcesses, processes...)
		return nil
	}
}

// ===========
// Scheduler
// ===========

func WithScheduler(opts ...scheduler.Option) EntrypointOption {
	return func(c *EntrypointConfig) error {
		cfg := scheduler.Default()
		for _, opt := range opts {
			if err := opt(cfg); err != nil {
				return fmt.Errorf("apply option: %w", err)
			}
		}

		c.scheduler = cfg
		return nil
	}
}

// ===========
// GrpcServer
// ===========

func WithGrpcServer(opts ...grpcserver.Option) EntrypointOption {
	return func(c *EntrypointConfig) error {
		cfg := grpcserver.Default()
		for _, opt := range opts {
			if err := opt(cfg); err != nil {
				return fmt.Errorf("apply option: %w", err)
			}
		}

		c.grpcServer = cfg
		return nil
	}
}

// ===========
// GrpcGateway
// ===========

func WithGrpcGateway(opts ...grpcgateway.Option) EntrypointOption {
	return func(c *EntrypointConfig) error {
		cfg := grpcgateway.Default()
		for _, opt := range opts {
			if err := opt(cfg); err != nil {
				return fmt.Errorf("apply option: %w", err)
			}
		}

		c.grpcGateway = cfg
		return nil
	}
}

// ===========
// Profiler
// ===========

func WithProfiler(opts ...profiler.Option) EntrypointOption {
	return func(c *EntrypointConfig) error {
		cfg := profiler.Default()
		for _, opt := range opts {
			if err := opt(cfg); err != nil {
				return fmt.Errorf("apply option: %w", err)
			}
		}

		c.profiler = cfg
		return nil
	}
}
