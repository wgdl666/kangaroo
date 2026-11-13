package callmeta

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type identity struct{ device, user string }
type identityContextKey struct{}

// WithIdentity 冻结调用发起者；空字段是有效快照，不回填目标用户，也不作为鉴权依据。
func WithIdentity(ctx context.Context, deviceID, userID string) context.Context {
	return context.WithValue(ctx, identityContextKey{}, identity{deviceID, userID})
}

func Identity(ctx context.Context) (deviceID, userID string) {
	value, _ := ctx.Value(identityContextKey{}).(identity)
	return value.device, value.user
}

// ValidIdentityValue 限制跨协议字段大小和控制字符，避免日志/HTTP 头注入。
func ValidIdentityValue(value string) bool {
	return len(value) <= 256 && !strings.ContainsFunc(value, func(r rune) bool { return r < 32 || r == 127 })
}

func identityValue(value string) string {
	if !ValidIdentityValue(value) {
		return ""
	}
	return value
}

func injectIdentity(ctx context.Context, carrier propagation.TextMapCarrier) {
	device, user := Identity(ctx)
	// 新载体省略空字段；复用载体必须清除上一请求的身份。
	for key, value := range map[string]string{DeviceID: device, UserID: user} {
		if value != "" || carrier.Get(key) != "" {
			carrier.Set(key, identityValue(value))
		}
	}
}

// AnnotateSpan 在已建立的请求/执行 Span 标注来源；不能在派生快照时修改共享父 Span。
func AnnotateSpan(ctx context.Context) {
	device, user := Identity(ctx)
	trace.SpanFromContext(ctx).SetAttributes(attribute.String("device_id", device), attribute.String("user_id", user))
}
