package httpserver

import (
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultWith(t *testing.T) {
	mux := http.NewServeMux()
	logger := slog.New(slog.DiscardHandler)

	type testCase struct {
		name    string
		handler http.Handler
		opts    []Option
		want    *Config
		wantErr error
	}

	tests := []testCase{
		{
			name:    "defaults",
			handler: mux,
			opts:    nil,
			want: &Config{
				Name:              "http-server",
				Port:              8080,
				Handler:           mux,
				ReadHeaderTimeout: 10 * time.Second,
				IdleTimeout:       2 * time.Minute,
			},
			wantErr: nil,
		},
		{
			name:    "custom options",
			handler: mux,
			opts: []Option{
				WithName("admin"),
				WithHost("127.0.0.1"),
				WithReadTimeout(time.Second),
				WithReadHeaderTimeout(2 * time.Second),
				WithWriteTimeout(3 * time.Second),
				WithIdleTimeout(4 * time.Second),
				WithMaxHeaderBytes(5),
				WithShutdownTimeout(6 * time.Second),
				WithLogger(logger),
			},
			want: &Config{
				Name:              "admin",
				Host:              "127.0.0.1",
				Port:              8080,
				Handler:           mux,
				ReadTimeout:       time.Second,
				ReadHeaderTimeout: 2 * time.Second,
				WriteTimeout:      3 * time.Second,
				IdleTimeout:       4 * time.Second,
				MaxHeaderBytes:    5,
				ShutdownTimeout:   6 * time.Second,
				Logger:            logger,
			},
			wantErr: nil,
		},
		{
			name:    "nil handler",
			handler: nil,
			opts:    nil,
			want:    nil,
			wantErr: ErrNilHandler,
		},
		{
			name:    "default serve mux",
			handler: http.DefaultServeMux,
			opts:    nil,
			want:    nil,
			wantErr: ErrDefaultServeMux,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := DefaultWith(8080, tc.handler, tc.opts...)

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
