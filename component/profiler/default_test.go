package profiler

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultWith(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)

	type testCase struct {
		name    string
		opts    []Option
		want    *Config
		wantErr error
	}

	tests := []testCase{
		{
			name: "local only by default",
			opts: nil,
			want: &Config{
				Host:            "127.0.0.1",
				Port:            1491,
				ShutdownTimeout: time.Second,
			},
			wantErr: nil,
		},
		{
			name: "custom options",
			opts: []Option{
				WithHost(""),
				WithPort(6060),
				WithShutdownTimeout(2 * time.Second),
				WithLogger(logger),
			},
			want: &Config{
				Host:            "",
				Port:            6060,
				ShutdownTimeout: 2 * time.Second,
				Logger:          logger,
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
