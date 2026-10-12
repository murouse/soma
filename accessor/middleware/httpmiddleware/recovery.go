package httpmiddleware

import (
	"errors"
	"net/http"

	"github.com/murouse/golgi/attr"
)

func (m *Manager) RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			p := recover()
			if p == nil {
				return
			}

			// http.ErrAbortHandler — штатный способ прервать ответ (так делает httputil.ReverseProxy):
			// net/http молча закрывает соединение, поэтому панику пробрасываем дальше.
			if err, ok := p.(error); ok && errors.Is(err, http.ErrAbortHandler) {
				panic(p)
			}

			m.logger.ErrorContext(r.Context(), "recovered from panic", attr.Panic(p), attr.Stack())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}()

		next.ServeHTTP(w, r)
	})
}
