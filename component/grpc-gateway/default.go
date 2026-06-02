package grpcgateway

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/cors"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/murouse/logo/attr"
	"github.com/samber/lo"
	"google.golang.org/protobuf/encoding/protojson"
)

func Default() *Config {
	return &Config{
		Port:            1480,
		ShutdownTimeout: time.Second,
		ServeMuxOptions: []runtime.ServeMuxOption{
			runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.HTTPBodyMarshaler{Marshaler: &runtime.JSONPb{
				MarshalOptions: protojson.MarshalOptions{UseProtoNames: false, EmitUnpopulated: true},
			}}),
		},
		HttpHandlers:       []httpHandlerConfig{},
		HealthEndpointPath: lo.ToPtr("/healthz"),
		Swagger: &swaggerConfig{
			dirOnDisk: "./docs/swagger",
			httpPath:  "/swagger/",
		},
		Cors: cors.Options{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{
				http.MethodHead,
				http.MethodGet,
				http.MethodPost,
				http.MethodPut,
				http.MethodPatch,
				http.MethodDelete,
			},
			AllowedHeaders:   []string{"*"},
			AllowCredentials: false,
		},
		Logger: slog.Default().With(attr.Component("grpc-gateway")),
	}
}

func DefaultWith(opts ...Option) (*Config, error) {
	cfg := Default()
	if err := cfg.Apply(opts...); err != nil {
		return nil, fmt.Errorf("apply options: %w", err)
	}
	return cfg, nil
}
