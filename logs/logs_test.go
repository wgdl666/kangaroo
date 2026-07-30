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

type capturedState struct {
	mu   sync.Mutex
	logs []capturedLog
}

// captureHandler 仅验证封装传给 slog 的结构化属性，不依赖真实 OTel exporter。
type captureHandler struct {
	state *capturedState
	attrs []slog.Attr
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *captureHandler) Handle(_ context.Context, record slog.Record) error {
	entry := capturedLog{message: record.Message, level: record.Level, attrs: map[string]string{}}
	for _, attr := range h.attrs {
		entry.attrs[attr.Key] = attr.Value.String()
	}
	record.Attrs(func(attr slog.Attr) bool {
		entry.attrs[attr.Key] = attr.Value.String()
		return true
	})
	h.state.mu.Lock()
	h.state.logs = append(h.state.logs, entry)
	h.state.mu.Unlock()
	return nil
}

func (h *captureHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// 派生 Handler 保存 slog.With 的字段，测试真实覆盖链式字段是否进入同一条记录。
	combined := append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &captureHandler{state: h.state, attrs: combined}
}
func (h *captureHandler) WithGroup(string) slog.Handler { return h }

func newCaptureHandler() *captureHandler { return &captureHandler{state: &capturedState{}} }

func TestInfoAddsTraceAndSpanIDs(t *testing.T) {
	handler := newCaptureHandler()
	previous := slog.Default()
	slog.SetDefault(slog.New(handler))
	t.Cleanup(func() { slog.SetDefault(previous) })

	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{1},
		SpanID:  trace.SpanID{2},
	})
	ctx := trace.ContextWithSpanContext(context.Background(), spanContext)

	Default().CtxInfo(ctx, "提交任务 %s", "task-1")

	if len(handler.state.logs) != 1 {
		t.Fatalf("log count = %d, want 1", len(handler.state.logs))
	}
	got := handler.state.logs[0]
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
	handler := newCaptureHandler()
	previous := slog.Default()
	slog.SetDefault(slog.New(handler))
	t.Cleanup(func() { slog.SetDefault(previous) })

	Default().Info("后台任务已启动")

	if len(handler.state.logs) != 1 {
		t.Fatalf("log count = %d, want 1", len(handler.state.logs))
	}
	if _, exists := handler.state.logs[0].attrs["trace_id"]; exists {
		t.Fatal("trace_id must be absent without a valid SpanContext")
	}
}

func TestEntryWithChainsFieldsAndKeepsTraceContext(t *testing.T) {
	handler := newCaptureHandler()
	previous := slog.Default()
	slog.SetDefault(slog.New(handler))
	t.Cleanup(func() { slog.SetDefault(previous) })

	spanContext := trace.NewSpanContext(trace.SpanContextConfig{TraceID: trace.TraceID{1}, SpanID: trace.SpanID{2}})
	ctx := trace.ContextWithSpanContext(context.Background(), spanContext)
	// 链式字段必须只写入本次日志，并与传入 ctx 的 Trace/Span 一起导出。
	With("task_id", "task-1").With("session_id", "session-1").CtxInfo(ctx, "ootd_workflow_task_created")

	got := handler.state.logs[0]
	if got.attrs["task_id"] != "task-1" || got.attrs["session_id"] != "session-1" {
		t.Fatalf("structured fields = %#v", got.attrs)
	}
	if got.attrs["trace_id"] != spanContext.TraceID().String() {
		t.Fatalf("trace_id = %q", got.attrs["trace_id"])
	}
}

func TestFatalLogsBeforeExiting(t *testing.T) {
	handler := newCaptureHandler()
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
	Default().Fatal("配置无法加载")

	if len(handler.state.logs) != 1 {
		t.Fatalf("log count = %d, want 1", len(handler.state.logs))
	}
	if handler.state.logs[0].message != "配置无法加载" {
		t.Fatalf("message = %q", handler.state.logs[0].message)
	}
	if handler.state.logs[0].level != LevelFatal {
		t.Fatalf("level = %d, want %d", handler.state.logs[0].level, LevelFatal)
	}
}
