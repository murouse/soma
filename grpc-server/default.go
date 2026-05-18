package grpcserver

import "google.golang.org/grpc"

func Default() *Config {
	return &Config{
		Params: Params{
			Port:              1482,
			ReflectionEnabled: true,
		},
		Impls:         []ImplementationAdapter{},
		ServerOptions: []grpc.ServerOption{},
	}
}
