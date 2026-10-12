package grpcinterceptor

import (
	"context"
	"log/slog"
	"strings"
	"time"

	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware/v2"
	"github.com/murouse/golgi"
	"github.com/murouse/golgi/attr"
	"github.com/murouse/soma/accessor/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const (
	attrGRPCGroupKey  string = "grpc"
	attrGRPCMethodKey string = "method"
	attrGRPCPeerKey   string = "peer"
	attrGRPCCodeKey   string = "code"
	attrGRPCStatusKey string = "status"

	attrGRPCMetadataGroupKey string = "metadata"

	healthMethodPrefix string = "/grpc.health.v1.Health/"
)

func (m *Manager) LoggingAttrsUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		return handler(m.withLoggingAttrs(ctx, info.FullMethod), req)
	}
}

func (m *Manager) LoggingAttrsStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Контекст стрима неизменяем, поэтому подменяем его через обертку.
		wrapped := grpcmiddleware.WrapServerStream(ss)
		wrapped.WrappedContext = m.withLoggingAttrs(ss.Context(), info.FullMethod)
		return handler(srv, wrapped)
	}
}

func (m *Manager) LoggingUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if strings.HasPrefix(info.FullMethod, healthMethodPrefix) {
			return handler(ctx, req)
		}

		start := time.Now()
		res, err := handler(ctx, req)
		m.logResult(ctx, start, err)

		return res, err
	}
}

func (m *Manager) LoggingStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if strings.HasPrefix(info.FullMethod, healthMethodPrefix) {
			return handler(srv, ss)
		}

		start := time.Now()
		err := handler(srv, ss)
		m.logResult(ss.Context(), start, err)

		return err
	}
}

// withLoggingAttrs кладет в контекст атрибуты запроса, которые попадут во все логи хендлера.
func (m *Manager) withLoggingAttrs(ctx context.Context, fullMethod string) context.Context {
	attrs := []slog.Attr{
		slog.String(attrGRPCMethodKey, fullMethod),
	}

	if p, ok := peer.FromContext(ctx); ok {
		attrs = append(attrs, slog.String(attrGRPCPeerKey, p.Addr.String()))
	}

	// Передаем метод md.Get как хелпер-колбэк
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if metaAttrs := middleware.ExtractMetaAttrs(m.loggingMetadataKeys, md.Get); len(metaAttrs) > 0 {
			attrs = append(attrs, slog.GroupAttrs(attrGRPCMetadataGroupKey, metaAttrs...))
		}
	}

	return golgi.WithAttrs(ctx, slog.GroupAttrs(attrGRPCGroupKey, attrs...))
}

// logResult пишет финальный лог запроса с кодом ответа.
func (m *Manager) logResult(ctx context.Context, start time.Time, err error) {
	code := status.Code(err)
	extra := []any{
		slog.GroupAttrs(attrGRPCGroupKey,
			slog.Int(attrGRPCCodeKey, int(code)),
			slog.String(attrGRPCStatusKey, code.String()),
		),
	}

	if err != nil {
		extra = append(extra, attr.Error(err))
	}

	middleware.LogRequestFinal(ctx, m.logger, levelForCode(code), start, extra...)
}

// levelForCode: ошибки клиента — Warn, ошибки сервиса — Error.
func levelForCode(code codes.Code) slog.Level {
	switch code {
	case codes.OK:
		return slog.LevelInfo
	case codes.Canceled, codes.InvalidArgument, codes.NotFound, codes.AlreadyExists,
		codes.PermissionDenied, codes.Unauthenticated, codes.ResourceExhausted,
		codes.FailedPrecondition, codes.Aborted, codes.OutOfRange:
		return slog.LevelWarn
	default: // Unknown, DeadlineExceeded, Unimplemented, Internal, Unavailable, DataLoss
		return slog.LevelError
	}
}
