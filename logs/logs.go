// Package logs 提供与 OpenTelemetry TraceContext 对齐的轻量结构化日志入口。
package logs

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

// exitProcess 允许测试验证 Fatal 在日志写出后终止进程；线上始终使用 os.Exit。
// Fatal 不依赖 defer，因为进程退出后 defer 无法保证执行，日志必须先同步提交给 handler。
var exitProcess = os.Exit

// LevelFatal 使用 slog 预留在 Error 之上的级别表示进程级致命错误。
// 该值需由应用侧的 OTel handler 映射为 OpenTelemetry 的 FATAL，不能退化成普通 ERROR。
const LevelFatal slog.Level = slog.LevelError + 4

// Entry 直接复用 slog.Logger；业务字段沿用 slog 的 With，避免重复实现普通日志能力。
type Entry struct{ *slog.Logger }

// Default 返回服务默认日志对象；所有日志级别入口都从 Entry 提供，避免包级和实例级两套 API。
func Default() Entry { return Entry{Logger: slog.Default()} }

// With 创建带首个业务字段的派生日志对象；是 Default().With 的简写，便于连续补充业务字段。
func With(key string, value any) Entry {
	return Default().With(key, value)
}

// With 为当前日志对象追加字段，直接复用 slog.Logger.With 的派生 Logger 语义。
func (e Entry) With(key string, value any) Entry {
	return Entry{Logger: e.slog().With(key, value)}
}

// Fatal 写入进程级致命错误后退出；调用方只能在服务无法继续提供功能时使用。
func (e Entry) Fatal(message string) {
	emit(context.Background(), e.slog(), LevelFatal, message)
	exitProcess(1)
}

// CtxDebug 输出关联当前 Trace/Span 的调试日志，名称与包级 CtxDebug 保持一致。
func (e Entry) CtxDebug(ctx context.Context, message string) {
	emit(ctx, e.slog(), slog.LevelDebug, message)
}

// CtxInfo 输出关联当前 Trace/Span 的普通日志，名称与包级 CtxInfo 保持一致。
func (e Entry) CtxInfo(ctx context.Context, message string) {
	emit(ctx, e.slog(), slog.LevelInfo, message)
}

// CtxWarn 输出关联当前 Trace/Span 的告警日志，名称与包级 CtxWarn 保持一致。
func (e Entry) CtxWarn(ctx context.Context, message string) {
	emit(ctx, e.slog(), slog.LevelWarn, message)
}

// CtxError 输出关联当前 Trace/Span 的错误日志，名称与包级 CtxError 保持一致。
func (e Entry) CtxError(ctx context.Context, message string) {
	emit(ctx, e.slog(), slog.LevelError, message)
}

// CtxFatal 写入关联当前调用的致命错误后退出，保留故障对应的 Trace/Span 供线上排障。
func (e Entry) CtxFatal(ctx context.Context, message string) {
	emit(ctx, e.slog(), LevelFatal, message)
	exitProcess(1)
}

// emit 是所有 Entry 的唯一落盘路径，统一保证 ctx 的 Trace/Span 字段不丢失。
func emit(ctx context.Context, logger *slog.Logger, level slog.Level, message string, attrs ...slog.Attr) {
	if ctx == nil {
		ctx = context.Background()
	}
	attrs = append(traceAttrs(ctx), attrs...)
	logger.LogAttrs(ctx, level, message, attrs...)
}

func (e Entry) slog() *slog.Logger {
	if e.Logger == nil {
		return slog.Default()
	}
	return e.Logger
}

func traceAttrs(ctx context.Context) []slog.Attr {
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		return nil
	}
	// 显式打印关联 ID，便于控制台和非 Trace 查询场景直接定位同一次调用。
	return []slog.Attr{
		slog.String("trace_id", spanContext.TraceID().String()),
		slog.String("span_id", spanContext.SpanID().String()),
	}
}
