package grpcinterceptor

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestManager_LoggingUnaryInterceptor(t *testing.T) {
	var logs bytes.Buffer
	interceptor := NewManager(slog.New(slog.NewJSONHandler(&logs, nil))).LoggingUnaryInterceptor()

	ctx := context.Background()

	type testCase struct {
		name       string
		method     string
		handlerErr error
		wantLevel  string
	}

	tests := []testCase{
		{
			name:       "ok is info",
			method:     "/test.Service/Method",
			handlerErr: nil,
			wantLevel:  "INFO",
		},
		{
			name:       "client error is warn",
			method:     "/test.Service/Method",
			handlerErr: status.Error(codes.NotFound, "not found"),
			wantLevel:  "WARN",
		},
		{
			name:       "server error is error",
			method:     "/test.Service/Method",
			handlerErr: status.Error(codes.Internal, "internal"),
			wantLevel:  "ERROR",
		},
		{
			name:       "unknown error is error",
			method:     "/test.Service/Method",
			handlerErr: assert.AnError,
			wantLevel:  "ERROR",
		},
		{
			name:       "health check is not logged",
			method:     "/grpc.health.v1.Health/Check",
			handlerErr: nil,
			wantLevel:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logs.Reset()
			handler := func(context.Context, any) (any, error) { return nil, tc.handlerErr }

			_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: tc.method}, handler)

			if tc.handlerErr != nil {
				assert.ErrorIs(t, err, tc.handlerErr)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.wantLevel, logLevel(logs.Bytes()))
		})
	}
}

func logLevel(data []byte) string {
	var entry struct {
		Level string `json:"level"`
	}
	_ = json.Unmarshal(data, &entry)
	return entry.Level
}
