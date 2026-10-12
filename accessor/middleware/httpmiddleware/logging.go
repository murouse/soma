package httpmiddleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/murouse/golgi"
	"github.com/murouse/soma/accessor/middleware"
)

const (
	attrHTTPGroupKey  string = "http"
	attrHTTPMethodKey string = "method"
	attrHTTPPathKey   string = "path"
	attrHTTPRemoteKey string = "remote"
	attrHTTPCodeKey   string = "code"
	attrHTTPStatusKey string = "status"

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

// Unwrap дает http.ResponseController доступ к исходному ResponseWriter. Без него
// не работают Flush (стриминг grpc-gateway, SSE) и Hijack (WebSocket).
func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

func (m *Manager) LoggingAttrsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attrs := []slog.Attr{
			slog.String(attrHTTPMethodKey, r.Method),
			slog.String(attrHTTPPathKey, r.URL.Path),
			slog.String(attrHTTPRemoteKey, r.RemoteAddr),
		}

		// Адаптируем http.Header под наш контракт через анонимную функцию
		getHTTPHeader := func(key string) []string {
			if val := r.Header.Get(key); val != "" {
				return []string{val}
			}
			return nil
		}

		if metaAttrs := middleware.ExtractMetaAttrs(m.loggingHeaderKeys, getHTTPHeader); len(metaAttrs) > 0 {
			attrs = append(attrs, slog.GroupAttrs(attrHeaderGroupKey, metaAttrs...))
		}

		ctx := golgi.WithAttrs(r.Context(), slog.GroupAttrs(attrHTTPGroupKey, attrs...))
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

		middleware.LogRequestFinal(r.Context(), m.logger, levelForStatus(rw.statusCode), start, slog.GroupAttrs(attrHTTPGroupKey,
			slog.Int(attrHTTPCodeKey, rw.statusCode),
			slog.String(attrHTTPStatusKey, http.StatusText(rw.statusCode)),
		))
	})
}

// levelForStatus: 5xx — ошибка сервиса, 4xx — ошибка клиента, остальное — успех.
func levelForStatus(status int) slog.Level {
	switch {
	case status >= http.StatusInternalServerError:
		return slog.LevelError
	case status >= http.StatusBadRequest:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}
