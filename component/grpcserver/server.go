package grpcserver

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"

	"buf.build/go/protovalidate"
	mwprotovalidate "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/protovalidate"
	"github.com/murouse/golgi/attr"
	"github.com/murouse/soma/accessor/middleware/grpcinterceptor"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// Service регистрирует реализацию gRPC-сервиса на сервере.
type Service interface {
	RegisterServer(server *grpc.Server)
}

// Server реализует gRPC-сервер с поддержкой сервисов.
type Server struct {
	cfg *Config

	logger   *slog.Logger
	server   *grpc.Server
	health   *health.Server
	listener net.Listener
}

func New(cfg *Config) *Server {
	return &Server{
		cfg:    cfg,
		logger: cmp.Or(cfg.Logger, slog.Default()).With(attr.Component("grpc-server")),
	}
}

func (s *Server) Name() string {
	return "grpc-server"
}

// Prepare занимает порт и собирает сервер: если порт занят, приложение упадет
// до старта остальных процессов.
func (s *Server) Prepare(ctx context.Context) error {
	options, err := s.serverOptions()
	if err != nil {
		return fmt.Errorf("server options: %w", err)
	}

	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", net.JoinHostPort("", strconv.Itoa(s.cfg.Port)))
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	s.listener = listener

	s.server = grpc.NewServer(options...)

	if s.cfg.HealthServerEnabled {
		s.health = health.NewServer()
		healthpb.RegisterHealthServer(s.server, s.health)
	}

	for _, service := range s.cfg.Services {
		service.RegisterServer(s.server)
	}

	if s.cfg.ReflectionEnabled {
		reflection.Register(s.server)
	}

	return nil
}

// serverOptions возвращает встроенные интерцепторы, за которыми идут пользовательские опции.
func (s *Server) serverOptions() ([]grpc.ServerOption, error) {
	validator, err := protovalidate.New()
	if err != nil {
		return nil, fmt.Errorf("new validator: %w", err)
	}

	interceptors := grpcinterceptor.NewManager(s.logger)

	return append([]grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			interceptors.LoggingAttrsUnaryInterceptor(),
			interceptors.LoggingUnaryInterceptor(),
			interceptors.RecoveryUnaryInterceptor(),
			mwprotovalidate.UnaryServerInterceptor(validator),
			grpcinterceptor.UnaryTimeoutInterceptor(s.cfg.UnaryTimeout),
		),
		grpc.ChainStreamInterceptor(
			interceptors.LoggingAttrsStreamInterceptor(),
			interceptors.LoggingStreamInterceptor(),
			interceptors.RecoveryStreamInterceptor(),
			mwprotovalidate.StreamServerInterceptor(validator),
		),
	}, s.cfg.ServerOptions...), nil
}

// Addr возвращает фактический адрес после Prepare (полезно при порте 0).
func (s *Server) Addr() net.Addr {
	return s.listener.Addr()
}

func (s *Server) Run(ctx context.Context) error {
	s.logger.InfoContext(ctx, "serving grpc", slog.String("addr", s.Addr().String()))

	if err := s.server.Serve(s.listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return fmt.Errorf("serve: %w", err)
	}

	return nil
}

// Drain переводит health-сервер в NOT_SERVING: балансировщики перестают слать новые
// запросы, а текущие продолжают обслуживаться до Shutdown.
func (s *Server) Drain() {
	if s.health != nil {
		s.health.Shutdown()
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	stopped := make(chan struct{})

	go func() {
		s.server.GracefulStop()
		close(stopped)
	}()

	var errs []error
	select {
	case <-stopped:
	case <-ctx.Done():
		s.logger.WarnContext(ctx, "graceful shutdown timeout exceeded, forcing stop")
		s.server.Stop() // Жесткий stop, закрывает все соединения
		errs = append(errs, fmt.Errorf("graceful stop: %w", ctx.Err()))
	}

	// Если Run не успел вызвать Serve, grpc.Server не знает о листенере — закрываем сами.
	if err := s.listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		errs = append(errs, fmt.Errorf("close listener: %w", err))
	}

	return errors.Join(errs...)
}
