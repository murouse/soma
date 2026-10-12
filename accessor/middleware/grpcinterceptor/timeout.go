package grpcinterceptor

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

func UnaryTimeoutInterceptor(timeout time.Duration) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// Если таймаут не задан, просто пропускаем
		if timeout <= 0 {
			return handler(ctx, req)
		}

		// Проверяем, передал ли клиент свой дедлайн
		if _, ok := ctx.Deadline(); ok {
			// У клиента уже есть свой таймаут, серверный дефолт не накладываем
			return handler(ctx, req)
		}

		// Создаем новый контекст с нашим дефолтным таймаутом
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		return handler(ctx, req)
	}
}
