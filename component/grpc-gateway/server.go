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
	runtimeServeMux := runtime.NewServeMux(s.cfg.ServeMuxOptions...) // Внутренний роутер grpc-gateway

	mainMux := http.NewServeMux()

	// Регистрируем пользовательские кастомные хендлеры
	for _, httpHandler := range s.cfg.HttpHandlers {
		mainMux.Handle(httpHandler.pattern, httpHandler.handler)
	}

	// Регистрируем grpc-gateway как catch-all fallback хендлер для всего остального
	mainMux.Handle("/", runtimeServeMux)

	// Инициализируем gRPC-клиент для проксирования
	var err error
	s.grpcClientConn, err = grpc.NewClient(fmt.Sprintf(":%d", s.grpcPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		//grpc.WithDefaultCallOptions(g.grpcCallOptions...),
	)
	if err != nil {
		return fmt.Errorf("new grpc client: %w", err)
	}

	for _, impl := range s.impls {
		err = errors.Join(err, impl.RegisterHandler(ctx, runtimeServeMux, s.grpcClientConn))
	}
	if err != nil {
		return fmt.Errorf("register gateway: %w", err)
	}

	handler := cors.AllowAll().Handler(mainMux)

	s.Server = httpserver.New(&httpserver.Config{
		Port:    s.cfg.Port,
		Handler: handler,
		Logger:  s.logger,
	})

	return nil
}
