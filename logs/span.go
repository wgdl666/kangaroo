package logs

import (
	"context"
	"encoding/json"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Span 是统一观测入口暴露的 Span 包装，服务业务无需直接依赖 OTel 实现。
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

// TraceID 返回当前 Span 所属的 Trace，用于业务任务持久化和跨服务恢复。
func (s *Span) TraceID() string {
	if s == nil || s.inner == nil || !s.inner.SpanContext().TraceID().IsValid() {
		return ""
	}
	return s.inner.SpanContext().TraceID().String()
}

// AddEvent 为当前业务节点补充时间线事件；仅接受常用标量，避免上报不可序列化数据。
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

func (l *Logger) StartSpan(ctx context.Context, name string) (context.Context, *Span) {
	if l == nil || l.tracer == nil {
		return ctx, nil
	}
	childCtx, span := l.tracer.Start(ctx, name)
	return childCtx, &Span{inner: span}
}

// StartRootSpan 为一次独立业务请求创建根 Span，避免长连接把多个请求错误合并。
func (l *Logger) StartRootSpan(ctx context.Context, name string) (context.Context, *Span) {
	if l == nil || l.tracer == nil {
		return ctx, nil
	}
	rootCtx, span := l.tracer.Start(ctx, name, trace.WithNewRoot())
	return rootCtx, &Span{inner: span}
}

func (l *Logger) StartSpanWithAttrs(ctx context.Context, name string, attrs map[string]string) (context.Context, *Span) {
	if l == nil || l.tracer == nil {
		return ctx, nil
	}
	kvs := make([]attribute.KeyValue, 0, len(attrs))
	for key, value := range attrs {
		kvs = append(kvs, attribute.String(key, value))
	}
	childCtx, span := l.tracer.Start(ctx, name, trace.WithAttributes(kvs...), trace.WithSpanKind(trace.SpanKindServer))
	return childCtx, &Span{inner: span}
}

// SpanFromContext 从当前业务上下文恢复 Span，用于在同一链路上记录事件和属性。
func SpanFromContext(ctx context.Context) *Span {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return nil
	}
	return &Span{inner: span}
}

func StartSpanFromContext(ctx context.Context, name string) (context.Context, *Span) {
	childCtx, span := otel.Tracer("").Start(ctx, name)
	return childCtx, &Span{inner: span}
}
