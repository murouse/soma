package grpcgateway

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/cors"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	httpserver "github.com/murouse/soma/component/http-server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type ImplementationAdapter interface {
	RegisterHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error
}

// Server представляет собой HTTP-шлюз, проксирующий запросы к gRPC-сервисам.
type Server struct {
	cfg      *Config
	impls    []ImplementationAdapter
	grpcPort int

	logger         *slog.Logger
	grpcClientConn *grpc.ClientConn

	*httpserver.Server
}

func New(cfg *Config, impls []ImplementationAdapter, grpcPort int) *Server {
	return &Server{cfg: cfg, impls: impls, grpcPort: grpcPort, logger: cfg.Logger}
}

func (s *Server) Prepare(ctx context.Context) error {
	// Инициализируем gRPC-клиент для проксирования
	var err error
	s.grpcClientConn, err = grpc.NewClient(fmt.Sprintf(":%d", s.grpcPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		//grpc.WithDefaultCallOptions(g.grpcCallOptions...),
	)
	if err != nil {
		return fmt.Errorf("new grpc client: %w", err)
	}

	serveMuxOptions := s.cfg.ServeMuxOptions
	if s.cfg.HealthEndpointPath != nil {
		serveMuxOptions = append(serveMuxOptions, runtime.WithHealthEndpointAt(
			healthpb.NewHealthClient(s.grpcClientConn), *s.cfg.HealthEndpointPath),
		)
	}

	runtimeServeMux := runtime.NewServeMux(serveMuxOptions...) // Внутренний роутер grpc-gateway

	mainMux := http.NewServeMux()

	// Регистрируем пользовательские кастомные хендлеры
	for _, httpHandler := range s.cfg.HttpHandlers {
		mainMux.Handle(httpHandler.pattern, httpHandler.handler)
	}

	// Регистрируем grpc-gateway как catch-all fallback хендлер для всего остального
	mainMux.Handle("/", runtimeServeMux)

	for _, impl := range s.impls {
		err = errors.Join(err, impl.RegisterHandler(ctx, runtimeServeMux, s.grpcClientConn))
	}
	if err != nil {
		return fmt.Errorf("register gateway: %w", err)
	}

	handler := cors.AllowAll().Handler(mainMux)

	s.Server, err = httpserver.NewDefaultWith(s.cfg.Port, handler,
		httpserver.WithLogger(s.logger),
	)
	if err != nil {
		return fmt.Errorf("new http server: %w", err)
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, s.cfg.ShutdownTimeout)
	defer cancel()

	if err := s.Server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	if err := s.grpcClientConn.Close(); err != nil {
		return fmt.Errorf("close grpc client conn: %w", err)
	}

	return nil
}
