package grpcserver

import (
	"context"
	"log/slog"
	"net"
	"syscall"
	"testing"

	"github.com/murouse/soma/component/grpcserver/mock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func TestServer_Prepare(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock.NewMockService(ctrl)

	ctx := context.Background()
	logger := slog.New(slog.DiscardHandler)

	busy, err := (&net.ListenConfig{}).Listen(ctx, "tcp", ":0")
	assert.NoError(t, err)
	defer func() { assert.NoError(t, busy.Close()) }()

	type testCase struct {
		name       string
		port       int
		setupMocks func()
		wantErr    error
	}

	tests := []testCase{
		{
			name: "registers services",
			port: 0,
			setupMocks: func() {
				mockService.EXPECT().RegisterServer(gomock.Any())
			},
			wantErr: nil,
		},
		{
			name:       "port in use",
			port:       busy.Addr().(*net.TCPAddr).Port,
			setupMocks: func() {},
			wantErr:    syscall.EADDRINUSE,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupMocks()
			s := New(&Config{Port: tc.port, Services: []Service{mockService}, Logger: logger})

			err := s.Prepare(ctx)

			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
				assert.NoError(t, s.Shutdown(ctx))
			}
		})
	}
}

func TestServer_Drain(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.DiscardHandler)

	type testCase struct {
		name  string
		drain bool
		want  healthpb.HealthCheckResponse_ServingStatus
	}

	tests := []testCase{
		{
			name:  "serving before drain",
			drain: false,
			want:  healthpb.HealthCheckResponse_SERVING,
		},
		{
			name:  "not serving after drain",
			drain: true,
			want:  healthpb.HealthCheckResponse_NOT_SERVING,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := New(&Config{HealthServerEnabled: true, Logger: logger})
			assert.NoError(t, s.Prepare(ctx))
			go func() { _ = s.Run(ctx) }()

			conn, err := grpc.NewClient(s.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			assert.NoError(t, err)

			if tc.drain {
				s.Drain()
			}

			resp, err := healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{})

			assert.NoError(t, err)
			assert.Equal(t, tc.want, resp.GetStatus())
			assert.NoError(t, conn.Close())
			assert.NoError(t, s.Shutdown(ctx))
		})
	}
}

func TestServer_Shutdown(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.DiscardHandler)

	type testCase struct {
		name string
		run  bool
	}

	tests := []testCase{
		{
			name: "without run releases port",
			run:  false,
		},
		{
			name: "after run releases port",
			run:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := New(&Config{Logger: logger})
			assert.NoError(t, s.Prepare(ctx))

			if tc.run {
				go func() { _ = s.Run(ctx) }()
			}

			err := s.Shutdown(ctx)

			assert.NoError(t, err)
			assert.True(t, canListen(ctx, s.Addr().String()))
		})
	}
}

func canListen(ctx context.Context, addr string) bool {
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", addr)
	if err != nil {
		return false
	}
	return listener.Close() == nil
}
