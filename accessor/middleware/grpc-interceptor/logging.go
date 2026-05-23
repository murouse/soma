package grpcinterceptor

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/murouse/logo"
	"github.com/murouse/logo/attr"
	"github.com/murouse/soma/accessor/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

const (
	attrGrpcGroupKey  string = "grpc"
	attrGrpcMethodKey string = "method"
	attrGrpcPeerKey   string = "peer"

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
		ctx = logo.WithAttrs(ctx, slog.GroupAttrs(attrGrpcGroupKey, attrs...))

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

		var extra []any
		if err != nil {
			extra = []any{attr.Error(err)}
		}
		middleware.LogRequestFinal(ctx, m.logger, start, err != nil, extra...)

		return res, err
	}
}
