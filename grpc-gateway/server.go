package grpcgateway

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ImplementationAdapter interface {
	RegisterHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error
}

type Server struct {
	cfg      *Config
	impls    []ImplementationAdapter
	grpcPort int

	grpcClientConn *grpc.ClientConn
	server         *http.Server
}

func New(cfg *Config, impls []ImplementationAdapter, grpcPort int) *Server {
	return &Server{cfg: cfg, impls: impls, grpcPort: grpcPort}
}

func (s *Server) PreRun(ctx context.Context) error {
	runtimeServeMux := runtime.NewServeMux(s.cfg.ServeMuxOptions...)

	router := chi.NewRouter()
	router.Use(middleware.Recoverer)
	//router.Mount("/", runtimeServeMux)
	router.Handle("/", runtimeServeMux)

	var err error
	s.grpcClientConn, err = grpc.NewClient(fmt.Sprintf(":%d", s.grpcPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		//grpc.WithDefaultCallOptions(g.grpcCallOptions...),
	)
	if err != nil {
		return fmt.Errorf("new grpc client: %w", err)
	}

	for _, impl := range s.impls {
		fmt.Printf("Registering grpc gateway implementation\n")
		err = errors.Join(err, impl.RegisterHandler(ctx, runtimeServeMux, s.grpcClientConn))
	}
	if err != nil {
		return fmt.Errorf("register gateway: %w", err)
	}

	handler := cors.AllowAll().Handler(router)

	s.server = &http.Server{
		Handler: handler,
		Addr:    fmt.Sprintf(":%d", s.cfg.Params.Port),
	}

	return nil
}

func (s *Server) Run(ctx context.Context) error {
	fmt.Printf("run serving grpc-gateway at %d\n", s.cfg.Params.Port)
	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen and serve: %w", err)
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, s.cfg.Params.ShutdownTimeout)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	if err := s.grpcClientConn.Close(); err != nil {
		return fmt.Errorf("close grpc client conn: %w", err)
	}

	return nil
}
