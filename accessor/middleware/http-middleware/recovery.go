package httpmiddleware

import (
	"net/http"

	"github.com/murouse/logo/attr"
)

func (m *Manager) RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if p := recover(); p != nil {
				m.logger.ErrorContext(r.Context(), "recovered from panic", attr.Panic(p), attr.Stack())
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
