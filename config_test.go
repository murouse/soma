package soma

import (
	"io"
	"log/slog"
	"testing"

	"github.com/murouse/soma/component/grpcconnect"
	"github.com/murouse/soma/component/grpcgateway"
	"github.com/murouse/soma/component/grpcserver"
	"github.com/murouse/soma/component/httpserver"
	"github.com/murouse/soma/component/profiler"
	"github.com/murouse/soma/component/scheduler"
	"github.com/stretchr/testify/assert"
)

func TestEntrypointConfig_Validate(t *testing.T) {
	type testCase struct {
		name    string
		cfg     *EntrypointConfig
		wantErr error
	}

	tests := []testCase{
		{
			name:    "empty config",
			cfg:     &EntrypointConfig{},
			wantErr: nil,
		},
		{
			name:    "gateway with grpc server",
			cfg:     &EntrypointConfig{grpcServer: &grpcserver.Config{}, grpcGateway: &grpcgateway.Config{}},
			wantErr: nil,
		},
		{
			name:    "gateway without grpc server",
			cfg:     &EntrypointConfig{grpcGateway: &grpcgateway.Config{}},
			wantErr: ErrGatewayRequiresGRPCServer,
		},
		{
			name:    "connect without grpc server",
			cfg:     &EntrypointConfig{grpcConnect: &grpcconnect.Config{}},
			wantErr: ErrConnectRequiresGRPCServer,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()

			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEntrypointConfig_inheritLogger(t *testing.T) {
	entrypointLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	componentLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	config := func(entrypoint, component *slog.Logger) *EntrypointConfig {
		return &EntrypointConfig{
			logger:      entrypoint,
			scheduler:   &scheduler.Config{Logger: component},
			grpcServer:  &grpcserver.Config{Logger: component},
			grpcGateway: &grpcgateway.Config{Logger: component},
			profiler:    &profiler.Config{Logger: component},
			httpServers: []*httpserver.Config{{Logger: component}},
		}
	}

	type testCase struct {
		name string
		cfg  *EntrypointConfig
		want *EntrypointConfig
	}

	tests := []testCase{
		{
			name: "without entrypoint logger",
			cfg:  config(nil, nil),
			want: config(nil, nil),
		},
		{
			name: "components inherit entrypoint logger",
			cfg:  config(entrypointLogger, nil),
			want: config(entrypointLogger, entrypointLogger),
		},
		{
			name: "component logger is kept",
			cfg:  config(entrypointLogger, componentLogger),
			want: config(entrypointLogger, componentLogger),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.cfg.inheritLogger()

			assert.Equal(t, tc.want, tc.cfg)
		})
	}
}
