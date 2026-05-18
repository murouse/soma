package grpcgateway

import (
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/protobuf/encoding/protojson"
)

func Default() *Config {
	return &Config{
		Params: Params{
			Port:            1480,
			ShutdownTimeout: time.Second,
		},
		ServeMuxOptions: []runtime.ServeMuxOption{
			runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.HTTPBodyMarshaler{Marshaler: &runtime.JSONPb{
				MarshalOptions: protojson.MarshalOptions{UseProtoNames: false, EmitUnpopulated: true},
			}}),
		},
	}
}
