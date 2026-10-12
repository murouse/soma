package httpmiddleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManager_RecoveryMiddleware(t *testing.T) {
	m := NewManager(slog.New(slog.DiscardHandler))

	type testCase struct {
		name       string
		panicValue any
		wantStatus int
		wantPanic  any
	}

	tests := []testCase{
		{
			name:       "no panic",
			panicValue: nil,
			wantStatus: http.StatusOK,
			wantPanic:  nil,
		},
		{
			name:       "panic returns internal server error",
			panicValue: assert.AnError,
			wantStatus: http.StatusInternalServerError,
			wantPanic:  nil,
		},
		{
			name:       "abort handler panic is propagated",
			panicValue: http.ErrAbortHandler,
			wantStatus: 0,
			wantPanic:  http.ErrAbortHandler,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := m.RecoveryMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				if tc.panicValue != nil {
					panic(tc.panicValue)
				}
			}))

			rec := httptest.NewRecorder()
			serve := func() { handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil)) }

			if tc.wantPanic != nil {
				assert.PanicsWithValue(t, tc.wantPanic, serve)
			} else {
				assert.NotPanics(t, serve)
				assert.Equal(t, tc.wantStatus, rec.Code)
			}
		})
	}
}
