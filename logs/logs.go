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

// Info 以当前 ctx 输出 Info 日志；无格式化占位符时，args 按 key/value 结构化字段处理。
func Info(ctx context.Context, message string, args ...any) {
	log(ctx, slog.LevelInfo, message, args...)
}

// Warn 以当前 ctx 输出 Warn 日志，适用于可继续处理但需要被关注的业务异常。
func Warn(ctx context.Context, message string, args ...any) {
	log(ctx, slog.LevelWarn, message, args...)
}

// Error 以当前 ctx 输出 Error 日志，适用于当前操作已经失败的场景。
func Error(ctx context.Context, message string, args ...any) {
	log(ctx, slog.LevelError, message, args...)
}

// Fatal 记录不可恢复的进程级错误后以退出码 1 结束进程。
// 调用方仅应在服务无法继续提供正确结果时使用；业务请求失败应使用 Error，避免误杀服务。
func Fatal(ctx context.Context, message string, args ...any) {
	log(ctx, LevelFatal, message, args...)
	exitProcess(1)
}

func log(ctx context.Context, level slog.Level, message string, args ...any) {
	if ctx == nil {
		ctx = context.Background()
	}
	attrs := traceAttrs(ctx)
	// 兼容原有 printf 风格日志；新代码未使用占位符时，把偶数位置参数保留为可查询字段。
	if strings.Contains(message, "%") {
		message = fmt.Sprintf(message, args...)
	} else {
		attrs = append(attrs, keyValuesToAttrs(args)...)
	}
	// 使用 LogAttrs(ctx, ...) 而非无 context 的 Info，确保 OTel handler 能读取当前 SpanContext。
	slog.LogAttrs(ctx, level, message, attrs...)
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
