// Package callmeta 定义 WG 内部调用的统一传播契约，不承载认证凭据或业务正文。
package callmeta

import (
	"context"

	"go.opentelemetry.io/otel/propagation"
)

const (
	BusinessKey   = "business_key"
	BusinessID    = "business_id"
	PromptDebug   = "prompt_debug"
	MetadataKey   = "metadata"
	BusinessLine  = "x-wg-business-line"
	BusinessScene = "x-wg-business-scene"
	DeviceID      = "x-wg-device-id"
	UserID        = "x-wg-user-id"
)

type attribution struct{ line, scene string }
type attributionContextKey struct{}

// WithAttribution 标记用户触发的业务线与功能；与技术 caller 和调试业务关联分开。
func WithAttribution(ctx context.Context, line, scene string) context.Context {
	return context.WithValue(ctx, attributionContextKey{}, attribution{line, scene})
}
func Attribution(ctx context.Context) (line, scene string) {
	value, _ := ctx.Value(attributionContextKey{}).(attribution)
	return value.line, value.scene
}

type business struct{ key, id string }
type businessContextKey struct{}

// WithBusiness 冻结本次调用的业务关联；值类型避免并发请求共享可变 map。
func WithBusiness(ctx context.Context, key, id string) context.Context {
	return context.WithValue(ctx, businessContextKey{}, business{key, id})
}

func Business(ctx context.Context) (key, id string) {
	value, _ := ctx.Value(businessContextKey{}).(business)
	return value.key, value.id
}

// Propagator 只允许标准 TraceContext、来源身份与成对业务标识跨内部服务传播，不复制 baggage/认证头。
// TraceContext 必须按当前 Span 重新注入，不能把入口 traceparent 原样转发。
type Propagator struct{}

func (Propagator) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	propagation.TraceContext{}.Inject(ctx, carrier)
	injectIdentity(ctx, carrier)
	if line, scene := Attribution(ctx); line != "" {
		carrier.Set(BusinessLine, line)
		carrier.Set(BusinessScene, scene)
	}
	if key, id := Business(ctx); key != "" && id != "" {
		carrier.Set(BusinessKey, key)
		carrier.Set(BusinessID, id)
	}
}
func (Propagator) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	ctx = propagation.TraceContext{}.Extract(ctx, carrier)
	ctx = WithIdentity(ctx, identityValue(carrier.Get(DeviceID)), identityValue(carrier.Get(UserID)))
	key, id := carrier.Get(BusinessKey), carrier.Get(BusinessID)
	if key == "" || id == "" {
		key, id = "", ""
	}
	return WithAttribution(WithBusiness(ctx, key, id), carrier.Get(BusinessLine), carrier.Get(BusinessScene))
}
func (Propagator) Fields() []string {
	return []string{"traceparent", "tracestate", BusinessKey, BusinessID, BusinessLine, BusinessScene, DeviceID, UserID}
}

// Capture 用同一协议持久化上下文；不保存 deadline 或取消状态。
func Capture(ctx context.Context) map[string]string {
	carrier := propagation.MapCarrier{}
	Propagator{}.Inject(ctx, carrier)
	return map[string]string(carrier)
}

func Restore(ctx context.Context, values map[string]string) context.Context {
	return Propagator{}.Extract(ctx, propagation.MapCarrier(values))
}

// SaveTask 把基础设施 metadata 与业务游标分开；调用方更新游标时须保留该字段。
func SaveTask(ctx context.Context, task map[string]any) {
	values := Capture(ctx)
	if len(values) != 0 {
		task[MetadataKey] = values
	}
}

func RestoreTask(ctx context.Context, task map[string]any) context.Context {
	values := map[string]string{}
	switch raw := task[MetadataKey].(type) {
	case map[string]string:
		values = raw
	case map[string]any:
		for _, key := range (Propagator{}).Fields() {
			if value, ok := raw[key].(string); ok {
				values[key] = value
			}
		}
	}
	return Restore(ctx, values)
}

// CopyTask 在重建业务游标时复制 metadata，避免共享 map 被后续更新污染。
func CopyTask(dst, src map[string]any) {
	if _, ok := src[MetadataKey]; ok {
		dst[MetadataKey] = Capture(RestoreTask(context.Background(), src))
	}
}

// Headers 适配 AMQP 的类型化头；正文和其它消息属性不参与上下文传播。
func Headers(ctx context.Context) map[string]any {
	values := map[string]any{}
	for key, value := range Capture(ctx) {
		values[key] = value
	}
	return values
}
func FromHeaders(ctx context.Context, headers map[string]any) context.Context {
	values := map[string]string{}
	for _, key := range (Propagator{}).Fields() {
		switch value := headers[key].(type) {
		case string:
			values[key] = value
		case []byte:
			values[key] = string(value)
		}
	}
	return Restore(ctx, values)
}

// WithSubscene 只为当前业务步骤派生局部上下文，构造模型请求时转换为固定字段，不经 header 传递。
func WithSubscene(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, subsceneContextKey{}, value)
}

type subsceneContextKey struct{}

func Subscene(ctx context.Context) string {
	value, _ := ctx.Value(subsceneContextKey{}).(string)
	return value
}

// RequestBusinessPropagator 用于业务归属已进入请求体的 RPC；仍保留追踪与草稿调试关联。
// 传输层不再复制业务线/场景，避免请求体与 header 出现两份事实来源。
type RequestBusinessPropagator struct{}

func (RequestBusinessPropagator) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	propagation.TraceContext{}.Inject(ctx, carrier)
	injectIdentity(ctx, carrier)
	if key, id := Business(ctx); key != "" && id != "" {
		carrier.Set(BusinessKey, key)
		carrier.Set(BusinessID, id)
	}
}
func (RequestBusinessPropagator) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	ctx = propagation.TraceContext{}.Extract(ctx, carrier)
	ctx = WithIdentity(ctx, identityValue(carrier.Get(DeviceID)), identityValue(carrier.Get(UserID)))
	// ModelHub 边界沿用成对关联契约，半对标识不能绑定草稿或残留上一请求的关联。
	key, id := carrier.Get(BusinessKey), carrier.Get(BusinessID)
	if key == "" || id == "" {
		key, id = "", ""
	}
	return WithBusiness(ctx, key, id)
}
func (RequestBusinessPropagator) Fields() []string {
	return []string{"traceparent", "tracestate", BusinessKey, BusinessID, DeviceID, UserID}
}
