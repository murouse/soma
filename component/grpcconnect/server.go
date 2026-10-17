// Package grpcconnect — HTTP-шлюз, принимающий Connect, gRPC-Web и gRPC
// (https://connectrpc.com) и проксирующий запросы в gRPC-сервер.
//
// В отличие от grpc-gateway, не требует сгенерированного кода: методы и типы сообщений
// берутся из дескрипторов зарегистрированных сервисов. Запросы приходят в gRPC-сервер
// обычным gRPC, поэтому его интерцепторы выполняются как для любого другого клиента.
package grpcconnect

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net"
	"net/http"
	"slices"
	"strings"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"connectrpc.com/vanguard"
	"github.com/murouse/golgi/attr"
	"github.com/murouse/soma/component/httpserver"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// internalServicePrefix — служебные сервисы grpc-go (health, reflection) наружу не проксируются.
const internalServicePrefix = "grpc."

var ErrServiceNotFound = errors.New("service descriptor not found")

// ServiceInfoProvider отдает зарегистрированные gRPC-сервисы. Его реализует *grpc.Server.
type ServiceInfoProvider interface {
	GetServiceInfo() map[string]grpc.ServiceInfo
}

// Server представляет собой Connect-шлюз, проксирующий запросы к gRPC-сервисам.
type Server struct {
	cfg        *Config
	services   ServiceInfoProvider
	grpcTarget string

	logger     *slog.Logger
	transport  *http.Transport
	httpServer *httpserver.Server
}

// New создает шлюз к gRPC-серверу по адресу grpcTarget (например, "localhost:1482").
// Список сервисов читается из services на фазе Prepare.
func New(cfg *Config, services ServiceInfoProvider, grpcTarget string) *Server {
	return &Server{
		cfg:        cfg,
		services:   services,
		grpcTarget: grpcTarget,
		logger:     cmp.Or(cfg.Logger, slog.Default()).With(attr.Component("grpc-connect")),
	}
}

func (s *Server) Name() string {
	return "grpc-connect"
}

func (s *Server) Prepare(ctx context.Context) error {
	proxy := connect.NewServer()
	upstream := connect.NewClient(s.upstreamTransport())

	for _, name := range slices.Sorted(maps.Keys(s.services.GetServiceInfo())) {
		if strings.HasPrefix(name, internalServicePrefix) {
			continue
		}
		service, err := findService(name)
		if err != nil {
			return err
		}
		proxy.Register(vanguard.ForwardService(upstream, service)...)
	}

	mux := http.NewServeMux()
	connecthttp.Mount(mux, proxy, s.cfg.HandlerOptions...)

	var err error
	s.httpServer, err = httpserver.NewDefaultWith(s.cfg.Port, mux,
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

// upstreamTransport — gRPC поверх HTTP/2 к gRPC-серверу: h2c или TLS из конфига.
func (s *Server) upstreamTransport() connect.Transport {
	var protocols http.Protocols
	scheme := "http://"
	if s.cfg.TLSConfig != nil {
		protocols.SetHTTP2(true)
		scheme = "https://"
	} else {
		protocols.SetUnencryptedHTTP2(true)
	}

	s.transport = &http.Transport{Protocols: &protocols, TLSClientConfig: s.cfg.TLSConfig}
	client := &http.Client{Transport: s.transport}

	return connecthttp.NewTransport(client, scheme+s.grpcTarget, connecthttp.WithGRPC())
}

func findService(name string) (protoreflect.ServiceDescriptor, error) {
	desc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(name))
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrServiceNotFound, name, err)
	}
	service, ok := desc.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, fmt.Errorf("%w: %s is %T", ErrServiceNotFound, name, desc)
	}
	return service, nil
}

func (s *Server) Addr() net.Addr {
	return s.httpServer.Addr()
}

func (s *Server) Run(ctx context.Context) error {
	return s.httpServer.Run(ctx)
}

func (s *Server) Shutdown(ctx context.Context) error {
	err := s.httpServer.Shutdown(ctx)
	s.transport.CloseIdleConnections()
	return err
}
