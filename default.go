package soma

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/murouse/logo/attr"
	grpcgateway "github.com/murouse/soma/component/grpc-gateway"
	grpcserver "github.com/murouse/soma/component/grpc-server"
	httpserver "github.com/murouse/soma/component/http-server"
	"github.com/murouse/soma/component/profiler"
	"github.com/murouse/soma/component/scheduler"
)

func Default() (*EntrypointConfig, error) {
	grpcServer, err := grpcserver.Default()
	if err != nil {
		return nil, fmt.Errorf("grpc server default: %w", err)
	}

	return &EntrypointConfig{
		scheduler:       scheduler.Default(),
		grpcServer:      grpcServer,
		grpcGateway:     grpcgateway.Default(),
		profiler:        profiler.Default(),
		customProcesses: []Process{},
		prepareTimeout:  time.Second * 5,
		shutdownTimeout: time.Second * 5,
		closures:        []func() error{},
		logger:          slog.Default().With(attr.Component("entrypoint")),
		httpServers:     []httpserver.Config{},
	}, nil
}

func DefaultWith(opts ...EntrypointOption) (*EntrypointConfig, error) {
	cfg, err := Default()
	if err != nil {
		return nil, fmt.Errorf("default: %w", err)
	}

	if err = cfg.Apply(opts...); err != nil {
		return nil, fmt.Errorf("apply options: %w", err)
	}
	return cfg, nil
}
