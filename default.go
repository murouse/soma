package soma

import (
	grpcgateway "github.com/murouse/soma/grpc-gateway"
	grpcserver "github.com/murouse/soma/grpc-server"
	"github.com/murouse/soma/profiler"
	"github.com/murouse/soma/scheduler"
)

func Default() *EntrypointConfig {
	return &EntrypointConfig{
		scheduler:       scheduler.Default(),
		grpcServer:      grpcserver.Default(),
		grpcGateway:     grpcgateway.Default(),
		profiler:        profiler.Default(),
		customProcesses: []Process{},
	}
}
