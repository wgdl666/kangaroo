package callmeta

import (
	"context"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc/stats"
)

// GRPCClientHandler/GRPCServerHandler 同时建立 OTel Span 和传播业务关联，避免两套拦截器覆盖父 Span。
func GRPCClientHandler() stats.Handler {
	return otelgrpc.NewClientHandler(otelgrpc.WithPropagators(Propagator{}))
}
func GRPCServerHandler() stats.Handler {
	return &identityServerHandler{Handler: otelgrpc.NewServerHandler(otelgrpc.WithPropagators(Propagator{}))}
}

// 在 OTel 提取上下文并创建 Span 后标注身份，不另建拦截器或替换父 Span。
type identityServerHandler struct{ stats.Handler }

func (h *identityServerHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	ctx = h.Handler.TagRPC(ctx, info)
	AnnotateSpan(ctx)
	return ctx
}
