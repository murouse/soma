package grpcserver

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"

	"google.golang.org/grpc/reflection"
)

type ImplementationAdapter interface {
	RegisterServer(grpcServer *grpc.Server)
	RegisterHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error
}

type Server struct {
	cfg *Config

	server *grpc.Server
}

func New(cfg *Config) *Server {
	return &Server{
		cfg: cfg,
	}
}

func (s *Server) PreRun(_ context.Context) error {
	s.server = grpc.NewServer(s.cfg.ServerOptions...)

	for _, impl := range s.cfg.Impls {
		fmt.Printf("RegisterServer grpc\n")
		impl.RegisterServer(s.server)
	}

	if s.cfg.Params.ReflectionEnabled {
		fmt.Printf("ReflectionEnabled grpc\n")
		reflection.Register(s.server)
	}

	return nil
}

func (s *Server) Run(ctx context.Context) error {
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", fmt.Sprintf(":%d", s.cfg.Params.Port))
	if err != nil {
		return fmt.Errorf("listen: %v", err)
	}

	fmt.Printf("run serving grpc at %d\n", s.cfg.Params.Port)
	if err = s.server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return fmt.Errorf("serve: %w", err)
	}

	return nil
}

func (s *Server) Shutdown(_ context.Context) error {
	s.server.GracefulStop()
	return nil
}
