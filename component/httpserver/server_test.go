package httpserver

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestServer_Prepare(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.DiscardHandler)

	busy, err := (&net.ListenConfig{}).Listen(ctx, "tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer func() { assert.NoError(t, busy.Close()) }()

	type testCase struct {
		name    string
		port    int
		wantErr error
	}

	tests := []testCase{
		{
			name:    "free port",
			port:    0,
			wantErr: nil,
		},
		{
			name:    "port in use",
			port:    busy.Addr().(*net.TCPAddr).Port,
			wantErr: syscall.EADDRINUSE,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, err := NewDefaultWith(tc.port, http.NotFoundHandler(), WithHost("127.0.0.1"), WithLogger(logger))
			assert.NoError(t, err)

			err = s.Prepare(ctx)

			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
				assert.NoError(t, s.Shutdown(ctx))
			}
		})
	}
}

func TestServer_Run(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.DiscardHandler)

	tlsServer := httptest.NewTLSServer(http.NotFoundHandler())
	tlsServer.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	type testCase struct {
		name   string
		opts   []Option
		scheme string
		client *http.Client
		want   int
	}

	tests := []testCase{
		{
			name:   "plain http",
			opts:   nil,
			scheme: "http",
			client: http.DefaultClient,
			want:   http.StatusNoContent,
		},
		{
			name:   "tls",
			opts:   []Option{WithTLSConfig(&tls.Config{Certificates: tlsServer.TLS.Certificates})},
			scheme: "https",
			client: tlsServer.Client(),
			want:   http.StatusNoContent,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, err := NewDefaultWith(0, handler, append([]Option{WithHost("127.0.0.1"), WithLogger(logger)}, tc.opts...)...)
			assert.NoError(t, err)
			assert.NoError(t, s.Prepare(ctx))

			runErr := make(chan error, 1)
			go func() { runErr <- s.Run(ctx) }()

			got, err := statusCode(ctx, tc.client, tc.scheme+"://"+s.Addr().String())

			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
			assert.NoError(t, s.Shutdown(ctx))
			assert.NoError(t, <-runErr)
		})
	}
}

func TestServer_Shutdown(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.DiscardHandler)

	type testCase struct {
		name            string
		run             bool
		shutdownTimeout time.Duration
		ctxTimeout      time.Duration
		requestDuration time.Duration
		wantErr         error
	}

	tests := []testCase{
		{
			name:            "without run releases port",
			run:             false,
			shutdownTimeout: 0,
			ctxTimeout:      time.Second,
			requestDuration: 0,
			wantErr:         nil,
		},
		{
			name:            "waits for in-flight request without server timeout",
			run:             true,
			shutdownTimeout: 0,
			ctxTimeout:      time.Second,
			requestDuration: 50 * time.Millisecond,
			wantErr:         nil,
		},
		{
			name:            "server timeout exceeded",
			run:             true,
			shutdownTimeout: 20 * time.Millisecond,
			ctxTimeout:      time.Second,
			requestDuration: time.Second,
			wantErr:         context.DeadlineExceeded,
		},
		{
			name:            "context timeout exceeded",
			run:             true,
			shutdownTimeout: time.Second,
			ctxTimeout:      20 * time.Millisecond,
			requestDuration: time.Second,
			wantErr:         context.DeadlineExceeded,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			started := make(chan struct{})
			handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				close(started)
				time.Sleep(tc.requestDuration)
			})

			s, err := NewDefaultWith(0, handler, WithHost("127.0.0.1"), WithLogger(logger), WithShutdownTimeout(tc.shutdownTimeout))
			assert.NoError(t, err)
			assert.NoError(t, s.Prepare(ctx))

			if tc.run {
				go func() { _ = s.Run(ctx) }()
				go func() { _, _ = statusCode(ctx, http.DefaultClient, "http://"+s.Addr().String()) }()
				<-started
			}

			shutdownCtx, cancel := context.WithTimeout(ctx, tc.ctxTimeout)
			defer cancel()

			err = s.Shutdown(shutdownCtx)

			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
			assert.True(t, canListen(ctx, s.Addr().String()))
		})
	}
}

func statusCode(ctx context.Context, client *http.Client, url string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}

	return resp.StatusCode, resp.Body.Close()
}

func canListen(ctx context.Context, addr string) bool {
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", addr)
	if err != nil {
		return false
	}
	return listener.Close() == nil
}
