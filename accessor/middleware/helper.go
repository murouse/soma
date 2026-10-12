package middleware

import (
	"context"
	"log/slog"
	"time"
)

const (
	attrDurationKey string = "duration"

	requestHandledMessage string = "request handled"
)

// ExtractMetaAttrs собирает только разрешенные заголовки из абстрактной мапы.
// Подходит и для gRPC (metadata.MD), и для HTTP (http.Header).
func ExtractMetaAttrs(allowedKeys []string, getFunc func(string) []string) []slog.Attr {
	metaAttrs := make([]slog.Attr, 0, len(allowedKeys))
	for _, key := range allowedKeys {
		if values := getFunc(key); len(values) > 0 && values[0] != "" {
			metaAttrs = append(metaAttrs, slog.String(key, values[0]))
		}
	}
	return metaAttrs
}

// LogRequestFinal пишет финальный лог запроса. Уровень выбирает вызывающий по коду ответа:
// успех — Info, ошибка клиента — Warn, ошибка сервиса — Error.
func LogRequestFinal(ctx context.Context, logger *slog.Logger, level slog.Level, start time.Time, extraAttrs ...any) {
	attrs := append([]any{slog.Duration(attrDurationKey, time.Since(start))}, extraAttrs...)
	logger.Log(ctx, level, requestHandledMessage, attrs...)
}
