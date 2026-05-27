package httpmiddleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/murouse/logo"
	"github.com/murouse/soma/accessor/middleware"
)

const (
	attrHttpGroupKey  string = "http"
	attrHttpMethodKey string = "method"
	attrHttpPathKey   string = "path"
	attrHttpRemoteKey string = "remote"

	attrHeaderGroupKey string = "header"
)

// Вспомогательная структура для перехвата HTTP-статуса
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (m *Manager) LoggingAttrsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attrs := []slog.Attr{
			slog.String(attrHttpMethodKey, r.Method),
			slog.String(attrHttpPathKey, r.URL.Path),
			slog.String(attrHttpRemoteKey, r.RemoteAddr),
		}

		// Адаптируем http.Header под наш контракт через анонимную функцию
		getHttpHeader := func(key string) []string {
			if val := r.Header.Get(key); val != "" {
				return []string{val}
			}
			return nil
		}

		if metaAttrs := middleware.ExtractMetaAttrs(m.loggingHeaderKeys, getHttpHeader); len(metaAttrs) > 0 {
			attrs = append(attrs, slog.GroupAttrs(attrHeaderGroupKey, metaAttrs...))
		}

		ctx := logo.WithAttrs(r.Context(), slog.GroupAttrs(attrHttpGroupKey, attrs...))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *Manager) LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" { // todo
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)

		// Вызываем общий логгер финала
		isError := rw.statusCode >= http.StatusBadRequest
		middleware.LogRequestFinal(r.Context(), m.logger, start, isError, slog.Int("http.code", rw.statusCode))
	})
}
