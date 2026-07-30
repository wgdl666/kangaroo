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

// Field 是一条日志的不可变结构化字段；它遵循 slog.Attr，避免散装 key/value 在调用处错位。
type Field = slog.Attr

// SetKV 创建可查询的单个日志字段。字段只属于当前日志调用，不保存到全局状态，避免并发请求串值。
func SetKV(key string, value any) Field { return slog.Any(key, value) }

// Debug 用于服务生命周期等无业务上下文的低优先级日志。
func Debug(message string, args ...any) { log(context.Background(), slog.LevelDebug, message, args...) }

// Info 用于服务生命周期等无业务上下文的普通日志。
func Info(message string, args ...any) { log(context.Background(), slog.LevelInfo, message, args...) }

// Warn 用于服务生命周期等无业务上下文的告警日志。
func Warn(message string, args ...any) { log(context.Background(), slog.LevelWarn, message, args...) }

// Error 用于服务生命周期等无业务上下文的错误日志。
func Error(message string, args ...any) { log(context.Background(), slog.LevelError, message, args...) }

// Fatal 记录无业务上下文的致命错误后结束进程。
func Fatal(message string, args ...any) {
	log(context.Background(), LevelFatal, message, args...)
	exitProcess(1)
}

// CtxDebug 记录并关联当前业务 ctx 的调试日志。
func CtxDebug(ctx context.Context, message string, args ...any) {
	log(ctx, slog.LevelDebug, message, args...)
}

// CtxInfo 记录并关联当前业务 ctx 的普通日志。
func CtxInfo(ctx context.Context, message string, args ...any) {
	log(ctx, slog.LevelInfo, message, args...)
}

// CtxWarn 记录并关联当前业务 ctx 的告警日志。
func CtxWarn(ctx context.Context, message string, args ...any) {
	log(ctx, slog.LevelWarn, message, args...)
}

// CtxError 记录并关联当前业务 ctx 的错误日志。
func CtxError(ctx context.Context, message string, args ...any) {
	log(ctx, slog.LevelError, message, args...)
}

// CtxFatal 记录并关联当前业务 ctx 的致命错误后结束进程。
func CtxFatal(ctx context.Context, message string, args ...any) {
	log(ctx, LevelFatal, message, args...)
	exitProcess(1)
}

func log(ctx context.Context, level slog.Level, message string, args ...any) {
	if ctx == nil {
		ctx = context.Background()
	}
	attrs := traceAttrs(ctx)
	// 兼容已有 printf 与散装 key/value 调用；新代码应传 SetKV，字段会保持独立且可查询。
	if strings.Contains(message, "%") {
		message = fmt.Sprintf(message, args...)
	} else {
		attrs = append(attrs, keyValuesToAttrs(args)...)
	}
	// 使用 LogAttrs(ctx, ...) 而非无 context 的 Info，确保 OTel handler 能读取当前 SpanContext。
	slog.LogAttrs(ctx, level, message, attrs...)
}

func keyValuesToAttrs(values []any) []slog.Attr {
	attrs := make([]slog.Attr, 0, len(values))
	for index := 0; index < len(values); {
		if field, ok := values[index].(slog.Attr); ok {
			attrs = append(attrs, field)
			index++
			continue
		}
		if index+1 >= len(values) {
			break
		}
		key, ok := values[index].(string)
		if !ok {
			index += 2
			continue
		}
		attrs = append(attrs, slog.Any(key, values[index+1]))
		index += 2
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
