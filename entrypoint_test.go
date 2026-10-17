package soma

import (
	"context"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/murouse/soma/component/grpcgateway"
	grpcgatewaymock "github.com/murouse/soma/component/grpcgateway/mock"
	"github.com/murouse/soma/component/grpcserver"
	grpcservermock "github.com/murouse/soma/component/grpcserver/mock"
	"github.com/murouse/soma/component/httpserver"
	"github.com/murouse/soma/mock"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type drainingProcess struct {
	*mock.MockProcess
	*mock.MockDrainer
}

type (
	serverServiceMock  = grpcservermock.MockService
	gatewayServiceMock = grpcgatewaymock.MockService
)

type gatewayService struct {
	*serverServiceMock
	*gatewayServiceMock
}

func TestNewEntrypoint(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProcess := mock.NewMockProcess(ctrl)

	type testCase struct {
		name      string
		opts      []EntrypointOption
		wantNames []string
		wantErr   error
	}

	tests := []testCase{
		{
			name:      "no components by default",
			opts:      nil,
			wantNames: []string{},
			wantErr:   nil,
		},
		{
			name:      "custom processes go first",
			opts:      []EntrypointOption{WithProfiler(), WithProcesses(mockProcess)},
			wantNames: []string{"*mock.MockProcess", "profiler"},
			wantErr:   nil,
		},
		{
			name: "all components",
			opts: []EntrypointOption{
				WithHTTPServer(http.NotFoundHandler(), 8080),
				WithProfiler(),
				WithGRPCConnect(),
				WithGRPCGateway(),
				WithGRPCServer(),
				WithScheduler(),
			},
			wantNames: []string{"scheduler", "grpc-server", "grpc-gateway", "grpc-connect", "profiler", "http-server"},
			wantErr:   nil,
		},
		{
			name:      "gateway without grpc server",
			opts:      []EntrypointOption{WithGRPCGateway()},
			wantNames: nil,
			wantErr:   ErrGatewayRequiresGRPCServer,
		},
		{
			name:      "connect without grpc server",
			opts:      []EntrypointOption{WithGRPCConnect()},
			wantNames: nil,
			wantErr:   ErrConnectRequiresGRPCServer,
		},
		{
			name:      "http server without handler",
			opts:      []EntrypointOption{WithHTTPServer(nil, 8080)},
			wantNames: nil,
			wantErr:   httpserver.ErrNilHandler,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NewEntrypoint(tc.opts...)

			if tc.wantErr != nil {
				assert.Nil(t, got)
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantNames, lo.Map(got.processes, func(p Process, _ int) string { return processName(p) }))
			}
		})
	}
}

func TestGatewayServices(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serverOnly := grpcservermock.NewMockService(ctrl)
	withGateway := gatewayService{grpcservermock.NewMockService(ctrl), grpcgatewaymock.NewMockService(ctrl)}

	type testCase struct {
		name     string
		services []grpcserver.Service
		want     []grpcgateway.Service
	}

	tests := []testCase{
		{
			name:     "no services",
			services: nil,
			want:     nil,
		},
		{
			name:     "only services with gateway handlers",
			services: []grpcserver.Service{serverOnly, withGateway},
			want:     []grpcgateway.Service{withGateway},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := gatewayServices(tc.services)

			assert.Equal(t, tc.want, got)
		})
	}
}

func TestEntrypoint_Run(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFirst := mock.NewMockProcess(ctrl)
	mockSecond := mock.NewMockProcess(ctrl)
	mockDrainer := mock.NewMockDrainer(ctrl)
	mockCloser := mock.NewMockCloser(ctrl)

	e := &Entrypoint{
		processes:       []Process{mockFirst, drainingProcess{mockSecond, mockDrainer}},
		prepareTimeout:  time.Second,
		shutdownTimeout: 100 * time.Millisecond,
		closeTimeout:    time.Second,
		closer:          mockCloser,
		logger:          slog.New(slog.DiscardHandler),
	}

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	stuck := make(chan struct{})
	defer close(stuck)

	untilDone := func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}
	untilStuck := func(context.Context) error {
		<-stuck
		return nil
	}
	aliveCtx := gomock.Cond(func(ctx context.Context) bool { return ctx.Err() == nil })

	type testCase struct {
		name       string
		ctx        context.Context
		setupMocks func()
		wantErr    error
	}

	tests := []testCase{
		{
			name: "all processes finished",
			ctx:  context.Background(),
			setupMocks: func() {
				gomock.InOrder(
					mockFirst.EXPECT().Prepare(gomock.Any()).Return(nil),
					mockSecond.EXPECT().Prepare(gomock.Any()).Return(nil),
				)
				mockFirst.EXPECT().Run(gomock.Any()).Return(nil)
				mockSecond.EXPECT().Run(gomock.Any()).Return(nil)
				gomock.InOrder(
					mockDrainer.EXPECT().Drain(),
					mockSecond.EXPECT().Shutdown(gomock.Any()).Return(nil),
					mockFirst.EXPECT().Shutdown(gomock.Any()).Return(nil),
					mockCloser.EXPECT().Close(gomock.Any()).Return(nil),
				)
			},
			wantErr: nil,
		},
		{
			name: "context canceled",
			ctx:  canceledCtx,
			setupMocks: func() {
				gomock.InOrder(
					mockFirst.EXPECT().Prepare(gomock.Any()).Return(nil),
					mockSecond.EXPECT().Prepare(gomock.Any()).Return(nil),
				)
				mockFirst.EXPECT().Run(gomock.Any()).DoAndReturn(untilDone)
				mockSecond.EXPECT().Run(gomock.Any()).DoAndReturn(untilDone)
				gomock.InOrder(
					mockDrainer.EXPECT().Drain(),
					mockSecond.EXPECT().Shutdown(gomock.Any()).Return(nil),
					mockFirst.EXPECT().Shutdown(gomock.Any()).Return(nil),
					mockCloser.EXPECT().Close(gomock.Any()).Return(nil),
				)
			},
			wantErr: nil,
		},
		{
			name: "run error stops other processes",
			ctx:  context.Background(),
			setupMocks: func() {
				gomock.InOrder(
					mockFirst.EXPECT().Prepare(gomock.Any()).Return(nil),
					mockSecond.EXPECT().Prepare(gomock.Any()).Return(nil),
				)
				mockFirst.EXPECT().Run(gomock.Any()).Return(assert.AnError)
				mockSecond.EXPECT().Run(gomock.Any()).DoAndReturn(untilDone)
				gomock.InOrder(
					mockDrainer.EXPECT().Drain(),
					mockSecond.EXPECT().Shutdown(gomock.Any()).Return(nil),
					mockFirst.EXPECT().Shutdown(gomock.Any()).Return(nil),
					mockCloser.EXPECT().Close(gomock.Any()).Return(nil),
				)
			},
			wantErr: assert.AnError,
		},
		{
			name: "prepare error rolls back prepared processes",
			ctx:  context.Background(),
			setupMocks: func() {
				gomock.InOrder(
					mockFirst.EXPECT().Prepare(gomock.Any()).Return(nil),
					mockSecond.EXPECT().Prepare(gomock.Any()).Return(assert.AnError),
					mockFirst.EXPECT().Shutdown(gomock.Any()).Return(nil),
					mockCloser.EXPECT().Close(gomock.Any()).Return(nil),
				)
			},
			wantErr: assert.AnError,
		},
		{
			name: "first prepare error closes resources",
			ctx:  context.Background(),
			setupMocks: func() {
				gomock.InOrder(
					mockFirst.EXPECT().Prepare(gomock.Any()).Return(assert.AnError),
					mockCloser.EXPECT().Close(gomock.Any()).Return(nil),
				)
			},
			wantErr: assert.AnError,
		},
		{
			name: "shutdown error does not skip other processes",
			ctx:  context.Background(),
			setupMocks: func() {
				gomock.InOrder(
					mockFirst.EXPECT().Prepare(gomock.Any()).Return(nil),
					mockSecond.EXPECT().Prepare(gomock.Any()).Return(nil),
				)
				mockFirst.EXPECT().Run(gomock.Any()).Return(nil)
				mockSecond.EXPECT().Run(gomock.Any()).Return(nil)
				gomock.InOrder(
					mockDrainer.EXPECT().Drain(),
					mockSecond.EXPECT().Shutdown(gomock.Any()).Return(assert.AnError),
					mockFirst.EXPECT().Shutdown(gomock.Any()).Return(nil),
					mockCloser.EXPECT().Close(gomock.Any()).Return(nil),
				)
			},
			wantErr: assert.AnError,
		},
		{
			name: "close error",
			ctx:  context.Background(),
			setupMocks: func() {
				gomock.InOrder(
					mockFirst.EXPECT().Prepare(gomock.Any()).Return(nil),
					mockSecond.EXPECT().Prepare(gomock.Any()).Return(nil),
				)
				mockFirst.EXPECT().Run(gomock.Any()).Return(nil)
				mockSecond.EXPECT().Run(gomock.Any()).Return(nil)
				gomock.InOrder(
					mockDrainer.EXPECT().Drain(),
					mockSecond.EXPECT().Shutdown(gomock.Any()).Return(nil),
					mockFirst.EXPECT().Shutdown(gomock.Any()).Return(nil),
					mockCloser.EXPECT().Close(gomock.Any()).Return(assert.AnError),
				)
			},
			wantErr: assert.AnError,
		},
		{
			name: "stuck process does not block closing resources",
			ctx:  canceledCtx,
			setupMocks: func() {
				gomock.InOrder(
					mockFirst.EXPECT().Prepare(gomock.Any()).Return(nil),
					mockSecond.EXPECT().Prepare(gomock.Any()).Return(nil),
				)
				mockFirst.EXPECT().Run(gomock.Any()).DoAndReturn(untilStuck)
				mockSecond.EXPECT().Run(gomock.Any()).DoAndReturn(untilDone)
				gomock.InOrder(
					mockDrainer.EXPECT().Drain(),
					mockSecond.EXPECT().Shutdown(gomock.Any()).Return(nil),
					mockFirst.EXPECT().Shutdown(gomock.Any()).Return(nil),
					mockCloser.EXPECT().Close(aliveCtx).Return(nil),
				)
			},
			wantErr: context.DeadlineExceeded,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupMocks()

			err := e.Run(tc.ctx)

			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
