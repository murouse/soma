package grpcgateway

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/murouse/soma/component/grpcgateway/mock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestServer_Prepare(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock.NewMockService(ctrl)

	ctx := context.Background()
	logger := slog.New(slog.DiscardHandler)

	type testCase struct {
		name       string
		setupMocks func()
		wantErr    error
	}

	tests := []testCase{
		{
			name: "registers handlers",
			setupMocks: func() {
				mockService.EXPECT().RegisterHandler(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			wantErr: nil,
		},
		{
			name: "register handler error",
			setupMocks: func() {
				mockService.EXPECT().RegisterHandler(gomock.Any(), gomock.Any(), gomock.Any()).Return(assert.AnError)
			},
			wantErr: assert.AnError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupMocks()
			cfg, err := DefaultWith(WithPort(0), WithLogger(logger))
			assert.NoError(t, err)
			s := New(cfg, []Service{mockService}, "localhost:0")

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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock.NewMockService(ctrl)

	ctx := context.Background()
	logger := slog.New(slog.DiscardHandler)

	swaggerDir := t.TempDir()
	assert.NoError(t, os.WriteFile(filepath.Join(swaggerDir, "index.html"), []byte("swagger"), 0o600))

	noContent := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	type testCase struct {
		name            string
		opts            []Option
		path            string
		setupMocks      func()
		wantStatus      int
		wantAllowOrigin string
	}

	tests := []testCase{
		{
			name: "swagger disabled by default",
			opts: []Option{WithSwaggerDir(swaggerDir)},
			path: "/swagger/",
			setupMocks: func() {
				mockService.EXPECT().RegisterHandler(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			wantStatus:      http.StatusNotFound,
			wantAllowOrigin: "",
		},
		{
			name: "swagger enabled",
			opts: []Option{WithSwaggerEnabled(true), WithSwaggerDir(swaggerDir)},
			path: "/swagger/",
			setupMocks: func() {
				mockService.EXPECT().RegisterHandler(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			wantStatus:      http.StatusOK,
			wantAllowOrigin: "",
		},
		{
			name: "cors enabled",
			opts: []Option{WithCORSEnabled(true)},
			path: "/swagger/",
			setupMocks: func() {
				mockService.EXPECT().RegisterHandler(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			wantStatus:      http.StatusNotFound,
			wantAllowOrigin: "*",
		},
		{
			name: "custom http handler",
			opts: []Option{WithHTTPHandler("/custom", noContent)},
			path: "/custom",
			setupMocks: func() {
				mockService.EXPECT().RegisterHandler(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			wantStatus:      http.StatusNoContent,
			wantAllowOrigin: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupMocks()
			cfg, err := DefaultWith(append([]Option{WithPort(0), WithLogger(logger)}, tc.opts...)...)
			assert.NoError(t, err)
			s := New(cfg, []Service{mockService}, "localhost:0")
			assert.NoError(t, s.Prepare(ctx))

			runErr := make(chan error, 1)
			go func() { runErr <- s.Run(ctx) }()

			status, allowOrigin, err := request(ctx, "http://"+s.Addr().String()+tc.path)

			assert.NoError(t, err)
			assert.Equal(t, tc.wantStatus, status)
			assert.Equal(t, tc.wantAllowOrigin, allowOrigin)
			assert.NoError(t, s.Shutdown(ctx))
			assert.NoError(t, <-runErr)
		})
	}
}

func request(ctx context.Context, url string) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Origin", "https://example.com")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, "", err
	}

	return resp.StatusCode, resp.Header.Get("Access-Control-Allow-Origin"), resp.Body.Close()
}
