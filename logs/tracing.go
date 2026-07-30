package logs

import (
	"context"
	"encoding/json"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Span 包装 OTel Span，隐藏 SDK 细节并维持业务事件的 map 写法。
type Span struct{ inner trace.Span }

func (s *Span) SetName(name string) {
	if s != nil && s.inner != nil {
		s.inner.SetName(name)
	}
}
func (s *Span) End() {
	if s != nil && s.inner != nil {
		s.inner.End()
	}
}
func (s *Span) RecordError(err error) {
	if s != nil && s.inner != nil {
		s.inner.RecordError(err)
	}
}
func (s *Span) SetInt64(key string, value int64) {
	if s != nil && s.inner != nil {
		s.inner.SetAttributes(attribute.Int64(key, value))
	}
}
func (s *Span) SetInt(key string, value int) {
	if s != nil && s.inner != nil {
		s.inner.SetAttributes(attribute.Int(key, value))
	}
}
func (s *Span) SetString(key, value string) {
	if s != nil && s.inner != nil {
		s.inner.SetAttributes(attribute.String(key, value))
	}
}
func (s *Span) SetFloat64(key string, value float64) {
	if s != nil && s.inner != nil {
		s.inner.SetAttributes(attribute.Float64(key, value))
	}
}
func (s *Span) TraceID() string {
	if s == nil || s.inner == nil || !s.inner.SpanContext().TraceID().IsValid() {
		return ""
	}
	return s.inner.SpanContext().TraceID().String()
}

// AddEvent 用简洁 map 写业务阶段；只转换实际支持的属性类型，避免业务方耦合 OTel attribute。
func (s *Span) AddEvent(name string, attrs map[string]any) {
	if s == nil || s.inner == nil {
		return
	}
	kvs := make([]attribute.KeyValue, 0, len(attrs))
	for key, value := range attrs {
		switch typed := value.(type) {
		case string:
			kvs = append(kvs, attribute.String(key, typed))
		case int:
			kvs = append(kvs, attribute.Int(key, typed))
		case int64:
			kvs = append(kvs, attribute.Int64(key, typed))
		case uint64:
			kvs = append(kvs, attribute.Int64(key, int64(typed)))
		case float64:
			kvs = append(kvs, attribute.Float64(key, typed))
		case bool:
			kvs = append(kvs, attribute.Bool(key, typed))
		case json.RawMessage:
			kvs = append(kvs, attribute.String(key, string(typed)))
		}
	}
	s.inner.AddEvent(name, trace.WithAttributes(kvs...))
}
func (s *Span) RecordErrorWithAttr(err error, key, value string) {
	if s != nil && s.inner != nil {
		s.inner.RecordError(err, trace.WithAttributes(attribute.String(key, value)))
	}
}

// StartSpan 创建当前 ctx 的子 Span；Setup 后它会使用已安装的全局 Provider。
func StartSpan(ctx context.Context, name string) (context.Context, *Span) {
	child, span := otel.Tracer("").Start(ctx, name)
	return child, &Span{inner: span}
}

// StartRootSpan 为一轮独立业务请求创建根 Span，避免长连接把多轮请求误合并。
func StartRootSpan(ctx context.Context, name string) (context.Context, *Span) {
	root, span := otel.Tracer("").Start(ctx, name, trace.WithNewRoot())
	return root, &Span{inner: span}
}
func StartSpanWithAttrs(ctx context.Context, name string, attrs map[string]string) (context.Context, *Span) {
	kvs := make([]attribute.KeyValue, 0, len(attrs))
	for key, value := range attrs {
		kvs = append(kvs, attribute.String(key, value))
	}
	child, span := otel.Tracer("").Start(ctx, name, trace.WithAttributes(kvs...), trace.WithSpanKind(trace.SpanKindServer))
	return child, &Span{inner: span}
}
func SpanFromContext(ctx context.Context) *Span {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return nil
	}
	return &Span{inner: span}
}
func StartSpanFromContext(ctx context.Context, name string) (context.Context, *Span) {
	return StartSpan(ctx, name)
}
