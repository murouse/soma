package grpcinterceptor

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/murouse/golgi"
	"github.com/murouse/golgi/attr"
	"github.com/murouse/soma/accessor/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const (
	attrGrpcGroupKey  string = "grpc"
	attrGrpcMethodKey string = "method"
	attrGrpcPeerKey   string = "peer"
	attrGrpcCodeKey   string = "code"
	attrGrpcStatusKey string = "status"

	attrGrpcMetadataGroupKey string = "metadata"
)

func (m *Manager) LoggingAttrsUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		attrs := []slog.Attr{
			slog.String(attrGrpcMethodKey, info.FullMethod),
		}

		if p, ok := peer.FromContext(ctx); ok {
			attrs = append(attrs, slog.String(attrGrpcPeerKey, p.Addr.String()))
		}

		// Передаем метод md.Get как хелсер-колбэк
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if metaAttrs := middleware.ExtractMetaAttrs(m.loggingMetadataKeys, md.Get); len(metaAttrs) > 0 {
				attrs = append(attrs, slog.GroupAttrs(attrGrpcMetadataGroupKey, metaAttrs...))
			}
		}
		ctx = golgi.WithAttrs(ctx, slog.GroupAttrs(attrGrpcGroupKey, attrs...))

		return handler(ctx, req)
	}
}

func (m *Manager) LoggingUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if strings.HasPrefix(info.FullMethod, "/grpc.health.v1.Health/") {
			return handler(ctx, req)
		}

		start := time.Now()
		res, err := handler(ctx, req)

		st, _ := status.FromError(err)
		extra := []any{
			slog.GroupAttrs(attrGrpcGroupKey,
				slog.Int(attrGrpcCodeKey, int(st.Code())),
				slog.String(attrGrpcStatusKey, st.Code().String()),
			),
		}

		if err != nil {
			extra = append(extra, attr.Error(err))
		}

		// Вызываем общий логгер финала
		isError := err != nil
		middleware.LogRequestFinal(ctx, m.logger, start, isError, extra...)

		return res, err
	}
}
