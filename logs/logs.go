// Package logs 提供与 OpenTelemetry TraceContext 对齐的轻量结构化日志入口。
package logs

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

// exitProcess 允许测试验证 Fatal 在日志写出后终止进程；线上始终使用 os.Exit。
// Fatal 不依赖 defer，因为进程退出后 defer 无法保证执行，日志必须先同步提交给 handler。
var exitProcess = os.Exit

// LevelFatal 使用 slog 预留在 Error 之上的级别表示进程级致命错误。
// 该值需由应用侧的 OTel handler 映射为 OpenTelemetry 的 FATAL，不能退化成普通 ERROR。
const LevelFatal slog.Level = slog.LevelError + 4

// Entry 包装 slog.With 返回的派生 Logger。每个 Entry 独立持有字段，不会污染全局 Logger。
type Entry struct{ logger *slog.Logger }

// Default 返回服务默认日志对象；所有日志级别入口都从 Entry 提供，避免包级和实例级两套 API。
func Default() Entry { return Entry{logger: slog.Default()} }

// With 创建带首个业务字段的派生日志对象；是 Default().With 的简写，便于连续补充业务字段。
func With(key string, value any) Entry {
	return Default().With(key, value)
}

// With 为当前日志对象追加字段，直接复用 slog.Logger.With 的派生 Logger 语义。
func (e Entry) With(key string, value any) Entry {
	return Entry{logger: e.slog().With(key, value)}
}

// Debug 输出无业务上下文的调试日志；旧调用的键值参数继续兼容，便于服务逐步迁移为 With 风格。
func (e Entry) Debug(message string, args ...any) {
	e.log(context.Background(), slog.LevelDebug, message, args...)
}

// Info 输出无业务上下文的普通日志。
func (e Entry) Info(message string, args ...any) {
	e.log(context.Background(), slog.LevelInfo, message, args...)
}

// Warn 输出无业务上下文的告警日志。
func (e Entry) Warn(message string, args ...any) {
	e.log(context.Background(), slog.LevelWarn, message, args...)
}

// Error 输出无业务上下文的错误日志。
func (e Entry) Error(message string, args ...any) {
	e.log(context.Background(), slog.LevelError, message, args...)
}

// Fatal 写入进程级致命错误后退出；调用方只能在服务无法继续提供功能时使用。
func (e Entry) Fatal(message string, args ...any) {
	e.log(context.Background(), LevelFatal, message, args...)
	exitProcess(1)
}

// CtxDebug 输出关联当前 Trace/Span 的调试日志，名称与包级 CtxDebug 保持一致。
func (e Entry) CtxDebug(ctx context.Context, message string, args ...any) {
	e.log(ctx, slog.LevelDebug, message, args...)
}

// CtxInfo 输出关联当前 Trace/Span 的普通日志，名称与包级 CtxInfo 保持一致。
func (e Entry) CtxInfo(ctx context.Context, message string, args ...any) {
	e.log(ctx, slog.LevelInfo, message, args...)
}

// CtxWarn 输出关联当前 Trace/Span 的告警日志，名称与包级 CtxWarn 保持一致。
func (e Entry) CtxWarn(ctx context.Context, message string, args ...any) {
	e.log(ctx, slog.LevelWarn, message, args...)
}

// CtxError 输出关联当前 Trace/Span 的错误日志，名称与包级 CtxError 保持一致。
func (e Entry) CtxError(ctx context.Context, message string, args ...any) {
	e.log(ctx, slog.LevelError, message, args...)
}

// CtxFatal 写入关联当前调用的致命错误后退出，保留故障对应的 Trace/Span 供线上排障。
func (e Entry) CtxFatal(ctx context.Context, message string, args ...any) {
	e.log(ctx, LevelFatal, message, args...)
	exitProcess(1)
}

// log 是 Entry 唯一的写入入口；兼容历史 printf 和键值参数，确保迁移期间日志语义不变。
func (e Entry) log(ctx context.Context, level slog.Level, message string, args ...any) {
	if strings.Contains(message, "%") {
		emit(ctx, e.slog(), level, fmt.Sprintf(message, args...))
		return
	}
	emit(ctx, e.slog(), level, message, keyValuesToAttrs(args)...)
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
	if e.logger == nil {
		return slog.Default()
	}
	return e.logger
}

func keyValuesToAttrs(values []any) []slog.Attr {
	attrs := make([]slog.Attr, 0, len(values)/2)
	for index := 0; index+1 < len(values); index += 2 {
		key, ok := values[index].(string)
		if !ok {
			continue
		}
		attrs = append(attrs, slog.Any(key, values[index+1]))
	}
	return attrs
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
