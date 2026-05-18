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

func (c *EntrypointConfig) Validate() error {
	if c.grpcServer == nil && c.grpcGateway != nil {
		return fmt.Errorf("must provide either grpcServer or grpcGateway")
	}

	return nil
}
