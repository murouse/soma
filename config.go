package soma

import (
	"fmt"
	"time"

	grpcgateway "github.com/murouse/soma/components/grpc-gateway"
	grpcserver "github.com/murouse/soma/components/grpc-server"
	"github.com/murouse/soma/components/profiler"
	"github.com/murouse/soma/components/scheduler"
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
}

func (c *EntrypointConfig) Validate() error {
	if c.grpcServer == nil && c.grpcGateway != nil {
		return fmt.Errorf("must provide either grpcServer or grpcGateway")
	}

	return nil
}

type EntrypointOption func(*EntrypointConfig) error

func WithProcesses(processes ...Process) EntrypointOption {
	return func(cfg *EntrypointConfig) error {
		cfg.customProcesses = append(cfg.customProcesses, processes...)
		return nil
	}
}

func WithPrepareTimeout(prepareTimeout time.Duration) EntrypointOption {
	return func(cfg *EntrypointConfig) error {
		cfg.prepareTimeout = prepareTimeout
		return nil
	}
}

func WithShutdownTimeout(shutdownTimeout time.Duration) EntrypointOption {
	return func(cfg *EntrypointConfig) error {
		cfg.shutdownTimeout = shutdownTimeout
		return nil
	}
}

func WithClosers(closures ...func() error) EntrypointOption {
	return func(cfg *EntrypointConfig) error {
		cfg.closures = append(cfg.closures, closures...)
		return nil
	}
}

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
