package middleware

import (
	"context"
	"log/slog"
	"time"
)

const (
	attrDurationKey string = "duration"

	successMessage string = "successfully handled request"
	failureMessage string = "error handling request"
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

// LogRequestFinal инкапсулирует общую логику записи финального лога.
func LogRequestFinal(ctx context.Context, logger *slog.Logger, start time.Time, isError bool, extraAttrs ...any) {
	duration := time.Since(start)

	level := slog.LevelInfo
	msg := successMessage

	if isError {
		level = slog.LevelError
		msg = failureMessage
	}

	attrs := make([]any, 0, 1+len(extraAttrs))
	attrs = append(attrs, slog.Duration(attrDurationKey, duration))
	attrs = append(attrs, extraAttrs...)

	logger.Log(ctx, level, msg, attrs...)
}
