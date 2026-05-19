package soma

import (
	"fmt"

	grpcgateway "github.com/murouse/soma/grpc-gateway"
	grpcserver "github.com/murouse/soma/grpc-server"
	"github.com/murouse/soma/profiler"
	"github.com/murouse/soma/scheduler"
)

type EntrypointConfig struct {
	scheduler   *scheduler.Config
	grpcServer  *grpcserver.Config
	grpcGateway *grpcgateway.Config
	profiler    *profiler.Config

	customProcesses []Process // процессы, добавляемые пользователем через WithProcesses
}

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

func (c *EntrypointConfig) Validate() error {
	if c.grpcServer == nil && c.grpcGateway != nil {
		return fmt.Errorf("must provide either grpcServer or grpcGateway")
	}

	return nil
}
