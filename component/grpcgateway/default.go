package grpcgateway

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/cors"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/protobuf/encoding/protojson"
)

// Default возвращает конфигурацию, в которой swagger и CORS выключены:
// каждое включается явно опциями WithSwaggerEnabled и WithCORSEnabled.
func Default() *Config {
	return &Config{
		Port:            1480,
		ShutdownTimeout: time.Second,
		ServeMuxOptions: []runtime.ServeMuxOption{
			runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.HTTPBodyMarshaler{Marshaler: &runtime.JSONPb{
				MarshalOptions: protojson.MarshalOptions{UseProtoNames: false, EmitUnpopulated: true},
			}}),
		},
		HealthEndpointEnabled: true,
		HealthEndpointPath:    "/healthz",
		SwaggerDir:            "./docs/swagger",
		SwaggerPath:           "/swagger/",
		CORSOptions: cors.Options{
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
	}
}

func DefaultWith(opts ...Option) (*Config, error) {
	cfg := Default()
	if err := cfg.Apply(opts...); err != nil {
		return nil, fmt.Errorf("apply options: %w", err)
	}
	return cfg, nil
}
