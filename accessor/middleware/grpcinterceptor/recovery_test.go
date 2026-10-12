package grpcinterceptor

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type testServerStream struct {
	grpc.ServerStream
}

func (testServerStream) Context() context.Context {
	return context.Background()
}

func TestManager_RecoveryUnaryInterceptor(t *testing.T) {
	interceptor := NewManager(slog.New(slog.DiscardHandler)).RecoveryUnaryInterceptor()

	ctx := context.Background()

	type testCase struct {
		name    string
		handler grpc.UnaryHandler
		want    any
		wantErr error
	}

	tests := []testCase{
		{
			name:    "success",
			handler: func(context.Context, any) (any, error) { return "response", nil },
			want:    "response",
			wantErr: nil,
		},
		{
			name:    "handler error",
			handler: func(context.Context, any) (any, error) { return nil, assert.AnError },
			want:    nil,
			wantErr: assert.AnError,
		},
		{
			name:    "panic becomes internal error",
			handler: func(context.Context, any) (any, error) { panic(assert.AnError) },
			want:    nil,
			wantErr: status.Error(codes.Internal, "internal server error"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, tc.handler)

			if tc.wantErr != nil {
				assert.Nil(t, resp)
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.want, resp)
			}
		})
	}
}

func TestManager_RecoveryStreamInterceptor(t *testing.T) {
	interceptor := NewManager(slog.New(slog.DiscardHandler)).RecoveryStreamInterceptor()

	type testCase struct {
		name    string
		handler grpc.StreamHandler
		wantErr error
	}

	tests := []testCase{
		{
			name:    "success",
			handler: func(any, grpc.ServerStream) error { return nil },
			wantErr: nil,
		},
		{
			name:    "handler error",
			handler: func(any, grpc.ServerStream) error { return assert.AnError },
			wantErr: assert.AnError,
		},
		{
			name:    "panic becomes internal error",
			handler: func(any, grpc.ServerStream) error { panic(assert.AnError) },
			wantErr: status.Error(codes.Internal, "internal server error"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := interceptor(nil, testServerStream{}, &grpc.StreamServerInfo{}, tc.handler)

			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
