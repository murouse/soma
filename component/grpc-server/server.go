package grpcserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"google.golang.org/grpc/reflection"
)

type ImplementationAdapter interface {
	RegisterServer(grpcServer *grpc.Server)
	RegisterHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error
}

// Server реализует gRPC-сервер с поддержкой адаптеров сервисов.
type Server struct {
	cfg *Config

	logger *slog.Logger
	server *grpc.Server
}

func New(cfg *Config) *Server {
	return &Server{
		cfg:    cfg,
		logger: cfg.Logger,
	}
}

func (s *Server) Prepare(ctx context.Context) error {
	s.server = grpc.NewServer(s.cfg.ServerOptions...)

	if s.cfg.HealthServerEnabled {
		healthpb.RegisterHealthServer(s.server, health.NewServer())
	}

	for _, impl := range s.cfg.Impls {
		impl.RegisterServer(s.server)
	}

	if s.cfg.ReflectionEnabled {
		reflection.Register(s.server)
	}

	return nil
}

func (s *Server) Run(ctx context.Context) error {
	s.logger.InfoContext(ctx, "serving grpc at", slog.Int("port", s.cfg.Port), slog.String("addr", "http://localhost:"+strconv.Itoa(s.cfg.Port)))
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", fmt.Sprintf(":%d", s.cfg.Port))
	if err != nil {
		return fmt.Errorf("listen: %v", err)
	}

	if err = s.server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return fmt.Errorf("serve: %w", err)
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.server.GracefulStop()
	return nil
}
