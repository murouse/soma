package soma

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/murouse/logo"
	grpcgateway "github.com/murouse/soma/component/grpc-gateway"
	grpcserver "github.com/murouse/soma/component/grpc-server"
	httpserver "github.com/murouse/soma/component/http-server"
	"github.com/murouse/soma/component/profiler"
	"github.com/murouse/soma/component/scheduler"
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
		logger:          slog.Default().With(logo.Component("entrypoint")), // TODO заменить
		httpServer:      []httpserver.Config{},
	}
}

func DefaultWith(opts ...EntrypointOption) (*EntrypointConfig, error) {
	cfg := Default()
	if err := cfg.Apply(opts...); err != nil {
		return nil, fmt.Errorf("apply options: %w", err)
	}
	return cfg, nil
}
