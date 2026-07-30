// Package logs 提供与 OpenTelemetry TraceContext 对齐的轻量结构化日志入口。
package logs

import (
	"context"
	"fmt"
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

// Info 以当前 ctx 输出 Info 日志，并把当前 Trace/Span ID 作为可见结构化属性。
// 传入 ctx 还会让已配置的 OTel slog handler 将日志关联到同一条 Trace。
func Info(ctx context.Context, format string, args ...any) {
	log(ctx, slog.LevelInfo, format, args...)
}

// Warn 以当前 ctx 输出 Warn 日志，适用于可继续处理但需要被关注的业务异常。
func Warn(ctx context.Context, format string, args ...any) {
	log(ctx, slog.LevelWarn, format, args...)
}

// Error 以当前 ctx 输出 Error 日志，适用于当前操作已经失败的场景。
func Error(ctx context.Context, format string, args ...any) {
	log(ctx, slog.LevelError, format, args...)
}

// Fatal 记录不可恢复的进程级错误后以退出码 1 结束进程。
// 调用方仅应在服务无法继续提供正确结果时使用；业务请求失败应使用 Error，避免误杀服务。
func Fatal(ctx context.Context, format string, args ...any) {
	log(ctx, LevelFatal, format, args...)
	exitProcess(1)
}

func log(ctx context.Context, level slog.Level, format string, args ...any) {
	if ctx == nil {
		ctx = context.Background()
	}
	attrs := traceAttrs(ctx)
	// 使用 LogAttrs(ctx, ...) 而非无 context 的 Info，确保 OTel handler 能读取当前 SpanContext。
	slog.LogAttrs(ctx, level, fmt.Sprintf(format, args...), attrs...)
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
