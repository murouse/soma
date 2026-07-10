package grpcinterceptor

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/murouse/golgi/attr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (m *Manager) RecoveryUnaryInterceptor() grpc.UnaryServerInterceptor {
	grpcPanicRecoveryHandler := func(ctx context.Context, p any) (err error) {
		if p == nil {
			return status.Error(codes.Internal, "nil panic")
		}
		m.logger.ErrorContext(ctx, "recovered from panic", attr.Panic(p), attr.Stack())
		return status.Error(codes.Internal, "internal server error")
	}

	return recovery.UnaryServerInterceptor(recovery.WithRecoveryHandlerContext(grpcPanicRecoveryHandler))
}
