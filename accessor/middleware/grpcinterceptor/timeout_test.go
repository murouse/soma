package grpcinterceptor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestUnaryTimeoutInterceptor(t *testing.T) {
	clientDeadline := time.Now().Add(time.Hour)
	clientCtx, cancel := context.WithDeadline(context.Background(), clientDeadline)
	defer cancel()

	handler := func(ctx context.Context, _ any) (any, error) {
		deadline, _ := ctx.Deadline()
		return deadline, nil
	}

	type testCase struct {
		name    string
		ctx     context.Context
		timeout time.Duration
		want    time.Time
	}

	tests := []testCase{
		{
			name:    "default timeout without client deadline",
			ctx:     context.Background(),
			timeout: time.Minute,
			want:    time.Now().Add(time.Minute),
		},
		{
			name:    "client deadline is kept",
			ctx:     clientCtx,
			timeout: time.Minute,
			want:    clientDeadline,
		},
		{
			name:    "zero timeout disables default",
			ctx:     context.Background(),
			timeout: 0,
			want:    time.Time{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := UnaryTimeoutInterceptor(tc.timeout)(tc.ctx, nil, &grpc.UnaryServerInfo{}, handler)

			assert.NoError(t, err)
			assert.WithinDuration(t, tc.want, resp.(time.Time), time.Second)
		})
	}
}
