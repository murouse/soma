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
	return recovery.UnaryServerInterceptor(recovery.WithRecoveryHandlerContext(m.recoveryHandler))
}

// RecoveryStreamInterceptor обязателен для стриминговых RPC: grpc-go не перехватывает паники
// в хендлерах, и паника в стриме без него роняет весь процесс.
func (m *Manager) RecoveryStreamInterceptor() grpc.StreamServerInterceptor {
	return recovery.StreamServerInterceptor(recovery.WithRecoveryHandlerContext(m.recoveryHandler))
}

func (m *Manager) recoveryHandler(ctx context.Context, p any) error {
	if p == nil {
		return status.Error(codes.Internal, "nil panic")
	}
	m.logger.ErrorContext(ctx, "recovered from panic", attr.Panic(p), attr.Stack())
	return status.Error(codes.Internal, "internal server error")
}
