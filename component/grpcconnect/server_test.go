package grpcconnect

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/murouse/soma/component/grpcserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const echoService = "soma.test.v1.EchoService"

// echo — сервис без сгенерированного кода: дескриптор регистрируется вручную в init.
type echo struct{}

func (echo) RegisterServer(server *grpc.Server) {
	server.RegisterService(&grpc.ServiceDesc{
		ServiceName: echoService,
		HandlerType: (*any)(nil),
		Methods: []grpc.MethodDesc{{
			MethodName: "Echo",
			Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
				in := new(wrapperspb.StringValue)
				if err := dec(in); err != nil {
					return nil, err
				}
				handler := func(ctx context.Context, req any) (any, error) {
					md, _ := metadata.FromIncomingContext(ctx)
					if err := grpc.SetHeader(ctx, metadata.Pairs("x-echo", strings.Join(md.Get("x-test"), ","))); err != nil {
						return nil, err
					}
					return wrapperspb.String(req.(*wrapperspb.StringValue).GetValue()), nil
				}
				info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/" + echoService + "/Echo"}
				return interceptor(ctx, in, info, handler)
			},
		}},
	}, echo{})
}

func init() {
	file, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Name:       new("soma/test/v1/echo.proto"),
		Package:    new("soma.test.v1"),
		Dependency: []string{"google/protobuf/wrappers.proto"},
		Syntax:     new("proto3"),
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name: new("EchoService"),
			Method: []*descriptorpb.MethodDescriptorProto{{
				Name:       new("Echo"),
				InputType:  new(".google.protobuf.StringValue"),
				OutputType: new(".google.protobuf.StringValue"),
			}},
		}},
	}, protoregistry.GlobalFiles)
	if err != nil {
		panic(err)
	}
	if err = protoregistry.GlobalFiles.RegisterFile(file); err != nil {
		panic(err)
	}
}

type fakeServices []string

func (f fakeServices) GetServiceInfo() map[string]grpc.ServiceInfo {
	info := make(map[string]grpc.ServiceInfo, len(f))
	for _, name := range f {
		info[name] = grpc.ServiceInfo{}
	}
	return info
}

func TestServer_Prepare(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.DiscardHandler)

	type testCase struct {
		name     string
		services fakeServices
		wantErr  error
	}

	tests := []testCase{
		{
			name:     "registers services",
			services: fakeServices{echoService},
			wantErr:  nil,
		},
		{
			name:     "skips internal services",
			services: fakeServices{"grpc.health.v1.Health", "grpc.unknown.v1.Service"},
			wantErr:  nil,
		},
		{
			name:     "unknown service descriptor",
			services: fakeServices{"unknown.v1.Service"},
			wantErr:  ErrServiceNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := DefaultWith(WithPort(0), WithLogger(logger))
			require.NoError(t, err)
			s := New(cfg, tc.services, "localhost:0")

			err = s.Prepare(ctx)

			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
				assert.NoError(t, s.Shutdown(ctx))
			}
		})
	}
}

func TestServer_Run(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.DiscardHandler)

	// Интерцептор gRPC-сервера должен выполняться для запросов через шлюз.
	intercepted := func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if err := grpc.SetHeader(ctx, metadata.Pairs("x-intercepted", "true")); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}

	grpcCfg, err := grpcserver.DefaultWith(
		grpcserver.WithPort(0),
		grpcserver.WithServices(echo{}),
		grpcserver.WithServerOptions(grpc.ChainUnaryInterceptor(intercepted)),
		grpcserver.WithLogger(logger),
	)
	require.NoError(t, err)
	grpcSrv := grpcserver.New(grpcCfg)
	require.NoError(t, grpcSrv.Prepare(ctx))
	go func() { _ = grpcSrv.Run(ctx) }()
	defer func() { assert.NoError(t, grpcSrv.Shutdown(ctx)) }()

	cfg, err := DefaultWith(WithPort(0), WithLogger(logger))
	require.NoError(t, err)
	grpcTarget := net.JoinHostPort("localhost", strconv.Itoa(grpcSrv.Addr().(*net.TCPAddr).Port))
	s := New(cfg, grpcSrv, grpcTarget)
	require.NoError(t, s.Prepare(ctx))
	runErr := make(chan error, 1)
	go func() { runErr <- s.Run(ctx) }()

	type testCase struct {
		name            string
		path            string
		wantStatus      int
		wantBody        string
		wantEcho        string
		wantIntercepted string
	}

	tests := []testCase{
		{
			name:            "proxies unary call with headers",
			path:            "/" + echoService + "/Echo",
			wantStatus:      http.StatusOK,
			wantBody:        `"hello"`,
			wantEcho:        "abc",
			wantIntercepted: "true",
		},
		{
			name:       "health service is not exposed",
			path:       "/grpc.health.v1.Health/Check",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+s.Addr().String()+tc.path, strings.NewReader(`"hello"`))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Test", "abc")

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())

			assert.Equal(t, tc.wantStatus, resp.StatusCode)
			if tc.wantBody != "" {
				assert.JSONEq(t, tc.wantBody, string(body))
			}
			assert.Equal(t, tc.wantEcho, resp.Header.Get("X-Echo"))
			assert.Equal(t, tc.wantIntercepted, resp.Header.Get("X-Intercepted"))
		})
	}

	assert.NoError(t, s.Shutdown(ctx))
	assert.NoError(t, <-runErr)
}
