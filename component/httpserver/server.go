package httpserver

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"slices"
	"strconv"

	"github.com/murouse/golgi/attr"
	"github.com/murouse/soma/accessor/middleware/httpmiddleware"
)

type Server struct {
	cfg      *Config
	logger   *slog.Logger
	server   *http.Server
	listener net.Listener
}

func New(cfg *Config) *Server {
	logger := cmp.Or(cfg.Logger, slog.Default()).With(attr.Component(cfg.Name))

	handler := cfg.Handler
	for _, middleware := range slices.Backward(cfg.Middlewares) {
		handler = middleware(handler)
	}

	// Встроенные middleware снаружи пользовательских: их паники и статусы тоже попадают в лог.
	mw := httpmiddleware.NewManager(logger)
	handler = mw.LoggingAttrsMiddleware(mw.LoggingMiddleware(mw.RecoveryMiddleware(handler)))

	return &Server{
		cfg:    cfg,
		logger: logger,
		server: &http.Server{
			Handler:           handler,
			TLSConfig:         cfg.TLSConfig,
			ReadTimeout:       cfg.ReadTimeout,
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       cfg.IdleTimeout,
			MaxHeaderBytes:    cfg.MaxHeaderBytes,
		},
	}
}

func (s *Server) Name() string {
	return s.cfg.Name
}

// Prepare занимает порт: если он занят, приложение упадет до старта остальных процессов.
func (s *Server) Prepare(ctx context.Context) error {
	address := net.JoinHostPort(s.cfg.Host, strconv.Itoa(s.cfg.Port))
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	s.listener = listener
	return nil
}

// Addr возвращает фактический адрес после Prepare (полезно при порте 0).
func (s *Server) Addr() net.Addr {
	return s.listener.Addr()
}

func (s *Server) Run(ctx context.Context) error {
	s.logger.InfoContext(ctx, "serving http", slog.String("addr", s.Addr().String()))

	var err error
	if s.server.TLSConfig != nil {
		// Пустые пути: сертификаты берутся из TLSConfig.
		err = s.server.ServeTLS(s.listener, "", "")
	} else {
		err = s.server.Serve(s.listener)
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}

	return nil
}

// Shutdown закрывает листенер и ждет завершения активных запросов. Ожидание ограничено
// переданным ctx и, если задан, ShutdownTimeout. Если запросы не успели завершиться,
// соединения закрываются принудительно.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.cfg.ShutdownTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.cfg.ShutdownTimeout)
		defer cancel()
	}

	var errs []error
	if err := s.server.Shutdown(ctx); err != nil {
		s.logger.WarnContext(ctx, "graceful shutdown failed, forcing close", attr.Error(err))
		errs = append(errs, fmt.Errorf("shutdown server: %w", err), s.server.Close())
	}

	// Если Run не успел вызвать Serve, http.Server не знает о листенере — закрываем сами.
	if err := s.listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		errs = append(errs, fmt.Errorf("close listener: %w", err))
	}

	return errors.Join(errs...)
}
