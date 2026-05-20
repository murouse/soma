package soma

import (
	"fmt"
	"time"

	grpcgateway "github.com/murouse/soma/components/grpc-gateway"
	grpcserver "github.com/murouse/soma/components/grpc-server"
	"github.com/murouse/soma/components/profiler"
	"github.com/murouse/soma/components/scheduler"
)

func Default() *EntrypointConfig {
	return &EntrypointConfig{
		scheduler:       scheduler.Default(),
		grpcServer:      grpcserver.Default(),
		grpcGateway:     grpcgateway.Default(),
		profiler:        profiler.Default(),
		customProcesses: []Process{},
		prepareTimeout:  time.Second * 5,
		shutdownTimeout: time.Second * 5,
		closures:        []func() error{},
	}
}

func (c *EntrypointConfig) Apply(opts ...EntrypointOption) error {
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return fmt.Errorf("apply option: %w", err)
		}
	}

	return nil
}

func DefaultWith(opts ...EntrypointOption) (*EntrypointConfig, error) {
	cfg := Default()
	if err := cfg.Apply(opts...); err != nil {
		return nil, fmt.Errorf("apply options: %w", err)
	}
	return cfg, nil
}
