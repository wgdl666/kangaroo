package logs

import (
	"context"
	"log/slog"
	"sync"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

type capturedLog struct {
	message string
	level   slog.Level
	attrs   map[string]string
}

// captureHandler 仅验证封装传给 slog 的结构化属性，不依赖真实 OTel exporter。
type captureHandler struct {
	mu   sync.Mutex
	logs []capturedLog
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *captureHandler) Handle(_ context.Context, record slog.Record) error {
	entry := capturedLog{message: record.Message, level: record.Level, attrs: map[string]string{}}
	record.Attrs(func(attr slog.Attr) bool {
		entry.attrs[attr.Key] = attr.Value.String()
		return true
	})
	h.mu.Lock()
	h.logs = append(h.logs, entry)
	h.mu.Unlock()
	return nil
}

func (h *captureHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *captureHandler) WithGroup(string) slog.Handler      { return h }

func TestInfoAddsTraceAndSpanIDs(t *testing.T) {
	handler := &captureHandler{}
	previous := slog.Default()
	slog.SetDefault(slog.New(handler))
	t.Cleanup(func() { slog.SetDefault(previous) })

	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{1},
		SpanID:  trace.SpanID{2},
	})
	ctx := trace.ContextWithSpanContext(context.Background(), spanContext)

	CtxInfo(ctx, "提交任务 %s", "task-1")

	if len(handler.logs) != 1 {
		t.Fatalf("log count = %d, want 1", len(handler.logs))
	}
	got := handler.logs[0]
	if got.message != "提交任务 task-1" {
		t.Fatalf("message = %q", got.message)
	}
	if got.attrs["trace_id"] != spanContext.TraceID().String() {
		t.Fatalf("trace_id = %q", got.attrs["trace_id"])
	}
	if got.attrs["span_id"] != spanContext.SpanID().String() {
		t.Fatalf("span_id = %q", got.attrs["span_id"])
	}
}

func TestInfoWithoutSpanStillLogs(t *testing.T) {
	handler := &captureHandler{}
	previous := slog.Default()
	slog.SetDefault(slog.New(handler))
	t.Cleanup(func() { slog.SetDefault(previous) })

	Info("后台任务已启动")

	if len(handler.logs) != 1 {
		t.Fatalf("log count = %d, want 1", len(handler.logs))
	}
	if _, exists := handler.logs[0].attrs["trace_id"]; exists {
		t.Fatal("trace_id must be absent without a valid SpanContext")
	}
}

func TestCtxInfoKeepsSetKVAsStructuredAttributes(t *testing.T) {
	handler := &captureHandler{}
	previous := slog.Default()
	slog.SetDefault(slog.New(handler))
	t.Cleanup(func() { slog.SetDefault(previous) })

	// 业务查询依赖 task_id、session_id 为独立字段，不能再把它们格式化进 message。
	CtxInfo(context.Background(), "ootd_workflow_task_created",
		SetKV("task_id", "task-1"),
		SetKV("session_id", "session-1"),
	)

	got := handler.logs[0]
	if got.attrs["task_id"] != "task-1" || got.attrs["session_id"] != "session-1" {
		t.Fatalf("structured fields = %#v", got.attrs)
	}
}

func TestFatalLogsBeforeExiting(t *testing.T) {
	handler := &captureHandler{}
	previousLogger := slog.Default()
	previousExit := exitProcess
	slog.SetDefault(slog.New(handler))
	exitProcess = func(code int) {
		if code != 1 {
			t.Errorf("exit code = %d, want 1", code)
		}
	}
	t.Cleanup(func() {
		slog.SetDefault(previousLogger)
		exitProcess = previousExit
	})

	// 致命错误必须先落入日志 handler，再请求结束进程，避免只退出而没有排查证据。
	Fatal("配置无法加载")

	if len(handler.logs) != 1 {
		t.Fatalf("log count = %d, want 1", len(handler.logs))
	}
	if handler.logs[0].message != "配置无法加载" {
		t.Fatalf("message = %q", handler.logs[0].message)
	}
	if handler.logs[0].level != LevelFatal {
		t.Fatalf("level = %d, want %d", handler.logs[0].level, LevelFatal)
	}
}
