package grpcserver

import (
	"log/slog"
	"testing"
	"time"

	"github.com/murouse/soma/component/grpcserver/mock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestDefaultWith(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock.NewMockService(ctrl)
	logger := slog.New(slog.DiscardHandler)

	type testCase struct {
		name    string
		opts    []Option
		want    *Config
		wantErr error
	}

	tests := []testCase{
		{
			name: "reflection disabled by default",
			opts: nil,
			want: &Config{
				Port:                1482,
				HealthServerEnabled: true,
				UnaryTimeout:        time.Minute,
			},
			wantErr: nil,
		},
		{
			name: "custom options",
			opts: []Option{
				WithPort(1483),
				WithReflectionEnabled(true),
				WithHealthServerEnabled(false),
				WithServices(mockService),
				WithUnaryTimeout(time.Second),
				WithLogger(logger),
			},
			want: &Config{
				Port:              1483,
				ReflectionEnabled: true,
				Services:          []Service{mockService},
				UnaryTimeout:      time.Second,
				Logger:            logger,
			},
			wantErr: nil,
		},
		{
			name:    "option error",
			opts:    []Option{func(*Config) error { return assert.AnError }},
			want:    nil,
			wantErr: assert.AnError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := DefaultWith(tc.opts...)

			if tc.wantErr != nil {
				assert.Nil(t, got)
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.want, got)
			}
		})
	}
}
