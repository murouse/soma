package soma

import (
	"log/slog"
	"testing"
	"time"

	"github.com/murouse/soma/mock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestDefaultWith(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProcess := mock.NewMockProcess(ctrl)
	logger := slog.New(slog.DiscardHandler)

	type testCase struct {
		name    string
		opts    []EntrypointOption
		want    *EntrypointConfig
		wantErr error
	}

	tests := []testCase{
		{
			name: "components disabled by default",
			opts: nil,
			want: &EntrypointConfig{
				prepareTimeout:  5 * time.Second,
				shutdownTimeout: 5 * time.Second,
				closeTimeout:    5 * time.Second,
			},
			wantErr: nil,
		},
		{
			name: "custom options",
			opts: []EntrypointOption{
				WithLogger(logger),
				WithPrepareTimeout(time.Second),
				WithShutdownDelay(2 * time.Second),
				WithShutdownTimeout(3 * time.Second),
				WithCloseTimeout(4 * time.Second),
				WithProcesses(mockProcess),
			},
			want: &EntrypointConfig{
				customProcesses: []Process{mockProcess},
				prepareTimeout:  time.Second,
				shutdownDelay:   2 * time.Second,
				shutdownTimeout: 3 * time.Second,
				closeTimeout:    4 * time.Second,
				logger:          logger,
			},
			wantErr: nil,
		},
		{
			name:    "option error",
			opts:    []EntrypointOption{func(*EntrypointConfig) error { return assert.AnError }},
			want:    nil,
			wantErr: assert.AnError,
		},
		{
			name:    "invalid config",
			opts:    []EntrypointOption{WithGRPCGateway()},
			want:    nil,
			wantErr: ErrGatewayRequiresGRPCServer,
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
