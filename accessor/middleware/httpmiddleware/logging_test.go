package httpmiddleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManager_LoggingMiddleware(t *testing.T) {
	var logs bytes.Buffer
	m := NewManager(slog.New(slog.NewJSONHandler(&logs, nil)))

	type testCase struct {
		name      string
		path      string
		status    int
		wantLevel string
	}

	tests := []testCase{
		{
			name:      "success is info",
			path:      "/",
			status:    http.StatusOK,
			wantLevel: "INFO",
		},
		{
			name:      "client error is warn",
			path:      "/",
			status:    http.StatusNotFound,
			wantLevel: "WARN",
		},
		{
			name:      "server error is error",
			path:      "/",
			status:    http.StatusInternalServerError,
			wantLevel: "ERROR",
		},
		{
			name:      "health check is not logged",
			path:      "/healthz",
			status:    http.StatusOK,
			wantLevel: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logs.Reset()

			var flushErr error
			handler := m.LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				flushErr = http.NewResponseController(w).Flush()
			}))

			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, tc.path, nil))

			assert.NoError(t, flushErr)
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
