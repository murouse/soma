package grpcgateway

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"slices"

	"github.com/go-chi/cors"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/murouse/golgi/attr"
	"github.com/murouse/soma/component/httpserver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// Service регистрирует HTTP-хендлеры grpc-gateway для gRPC-сервиса.
type Service interface {
	RegisterHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error
}

// Server представляет собой HTTP-шлюз, проксирующий запросы к gRPC-сервисам.
type Server struct {
	cfg        *Config
	services   []Service
	grpcTarget string

	logger     *slog.Logger
	conn       *grpc.ClientConn
	httpServer *httpserver.Server
}

// New создает шлюз к gRPC-серверу по адресу grpcTarget (например, "localhost:1482").
func New(cfg *Config, services []Service, grpcTarget string) *Server {
	return &Server{
		cfg:        cfg,
		services:   services,
		grpcTarget: grpcTarget,
		logger:     cmp.Or(cfg.Logger, slog.Default()).With(attr.Component("grpc-gateway")),
	}
}

func (s *Server) Name() string {
	return "grpc-gateway"
}

func (s *Server) Prepare(ctx context.Context) error {
	if err := s.prepare(ctx); err != nil {
		// Shutdown для процесса с упавшим Prepare не вызывается: соединение закрываем сами.
		if s.conn != nil {
			err = errors.Join(err, s.conn.Close())
		}
		return err
	}
	return nil
}

func (s *Server) prepare(ctx context.Context) error {
	dialOptions := append([]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}, s.cfg.DialOptions...)

	var err error
	s.conn, err = grpc.NewClient(s.grpcTarget, dialOptions...)
	if err != nil {
		return fmt.Errorf("new grpc client: %w", err)
	}

	gatewayMux := runtime.NewServeMux(s.serveMuxOptions()...)
	for _, service := range s.services {
		if err = service.RegisterHandler(ctx, gatewayMux, s.conn); err != nil {
			return fmt.Errorf("register handler: %w", err)
		}
	}

	if s.cfg.SwaggerEnabled {
		s.logger.InfoContext(ctx, "serving swagger", slog.String("path", s.cfg.SwaggerPath))
	}

	s.httpServer, err = httpserver.NewDefaultWith(s.cfg.Port, s.handler(gatewayMux),
		httpserver.WithName(s.Name()),
		httpserver.WithLogger(s.cfg.Logger),
		httpserver.WithShutdownTimeout(s.cfg.ShutdownTimeout),
	)
	if err != nil {
		return fmt.Errorf("new http server: %w", err)
	}

	if err = s.httpServer.Prepare(ctx); err != nil {
		return fmt.Errorf("prepare http server: %w", err)
	}

	return nil
}

func (s *Server) serveMuxOptions() []runtime.ServeMuxOption {
	options := slices.Clone(s.cfg.ServeMuxOptions)
	if s.cfg.HealthEndpointEnabled {
		// Проксирует gRPC health: после Drain эндпоинт отвечает 503.
		options = append(options, runtime.WithHealthEndpointAt(healthpb.NewHealthClient(s.conn), s.cfg.HealthEndpointPath))
	}
	return options
}

// handler собирает итоговый HTTP-хендлер: пользовательские хендлеры и swagger,
// все остальное уходит в grpc-gateway.
func (s *Server) handler(gatewayMux http.Handler) http.Handler {
	mux := http.NewServeMux()

	for _, httpHandler := range s.cfg.HTTPHandlers {
		mux.Handle(httpHandler.pattern, httpHandler.handler)
	}

	if s.cfg.SwaggerEnabled {
		fileServer := http.FileServer(http.Dir(s.cfg.SwaggerDir))
		mux.Handle(s.cfg.SwaggerPath, http.StripPrefix(s.cfg.SwaggerPath, fileServer))
	}

	mux.Handle("/", gatewayMux)

	if !s.cfg.CORSEnabled {
		return mux
	}
	return cors.New(s.cfg.CORSOptions).Handler(mux)
}

// Addr возвращает фактический адрес после Prepare (полезно при порте 0).
func (s *Server) Addr() net.Addr {
	return s.httpServer.Addr()
}

func (s *Server) Run(ctx context.Context) error {
	return s.httpServer.Run(ctx)
}

func (s *Server) Shutdown(ctx context.Context) error {
	// Соединение с gRPC закрываем, даже если HTTP-сервер не остановился штатно.
	return errors.Join(s.httpServer.Shutdown(ctx), s.closeConn())
}

func (s *Server) closeConn() error {
	if err := s.conn.Close(); err != nil {
		return fmt.Errorf("close grpc client conn: %w", err)
	}
	return nil
}
