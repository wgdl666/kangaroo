// Package telemetry 负责服务启动时的 OpenTelemetry 导出管道和生命周期。
// 业务日志与 Span 操作分别留在 logs、tracing，避免业务代码接触 Provider 细节。
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Config 描述服务身份和 OTLP 连接参数；业务层不参与 Handler 或 Provider 装配。
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	Endpoint       string
	Headers        map[string]string
	MinLevel       string
	Console        bool
}

// Runtime 仅持有 Provider 生命周期，服务退出时用它冲刷已缓存的遥测数据。
type Runtime struct {
	logProvider   *sdklog.LoggerProvider
	traceProvider *sdktrace.TracerProvider
}

// Setup 安装全局 slog、Trace Provider 和 W3C 传播器，使 logs 与 tracing 自动写入同一条链路。
func Setup(ctx context.Context, cfg Config) (*Runtime, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(cfg.ServiceName) == "" {
		return nil, fmt.Errorf("telemetry: service name is required")
	}
	res := resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceName(cfg.ServiceName), semconv.ServiceVersion(cfg.ServiceVersion), attribute.String("deployment.environment.name", cfg.Environment))
	logProvider := sdklog.NewLoggerProvider(sdklog.WithResource(res))
	traceProvider := sdktrace.NewTracerProvider(sdktrace.WithResource(res))
	if strings.TrimSpace(cfg.Endpoint) != "" {
		logExporter, err := otlploghttp.New(ctx, otlploghttp.WithEndpoint(cfg.Endpoint), otlploghttp.WithHeaders(cfg.Headers))
		if err != nil {
			_ = logProvider.Shutdown(ctx)
			return nil, fmt.Errorf("telemetry: create OTLP log exporter: %w", err)
		}
		traceExporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpoint(cfg.Endpoint), otlptracehttp.WithHeaders(cfg.Headers))
		if err != nil {
			_ = logProvider.Shutdown(ctx)
			return nil, fmt.Errorf("telemetry: create OTLP trace exporter: %w", err)
		}
		logProvider = sdklog.NewLoggerProvider(sdklog.WithResource(res), sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)))
		traceProvider = sdktrace.NewTracerProvider(sdktrace.WithResource(res), sdktrace.WithBatcher(traceExporter))
	}
	global.SetLoggerProvider(logProvider)
	otel.SetTracerProvider(traceProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	handlers := []slog.Handler{newOTelHandler(logProvider.Logger(cfg.ServiceName), parseLevel(cfg.MinLevel))}
	if cfg.Console {
		handlers = append([]slog.Handler{slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})}, handlers...)
	}
	slog.SetDefault(slog.New(newMultiHandler(handlers...)))
	return &Runtime{logProvider: logProvider, traceProvider: traceProvider}, nil
}

// NewConsole 为 CLI 及测试安装控制台 Handler；业务日志仍只能经 logs 包写入。
func NewConsole(minLevel string) {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: parseLevel(minLevel)})))
}

// NewDiscard 为测试安装静默 Handler，避免测试依赖 slog 的构造细节。
func NewDiscard() { slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil))) }

// Shutdown 在服务退出时依次冲刷日志和 Trace，避免批量导出中的数据丢失。
func (r *Runtime) Shutdown(ctx context.Context) error {
	if r == nil {
		return nil
	}
	return errors.Join(shutdownLog(ctx, r.logProvider), shutdownTrace(ctx, r.traceProvider))
}
func shutdownLog(ctx context.Context, p *sdklog.LoggerProvider) error {
	if p == nil {
		return nil
	}
	return p.Shutdown(ctx)
}
func shutdownTrace(ctx context.Context, p *sdktrace.TracerProvider) error {
	if p == nil {
		return nil
	}
	return p.Shutdown(ctx)
}

func parseLevel(value string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelDebug
	}
}

// OTel Handler 将 slog 记录变成 OTel Log Record，并以传入 ctx 自动关联 Trace/Span。
type otelHandler struct {
	logger   otellog.Logger
	minLevel slog.Level
	attrs    []slog.Attr
	groups   []string
}

func newOTelHandler(logger otellog.Logger, minLevel slog.Level) *otelHandler {
	return &otelHandler{logger: logger, minLevel: minLevel}
}
func (h *otelHandler) Enabled(_ context.Context, level slog.Level) bool { return level >= h.minLevel }
func (h *otelHandler) Handle(ctx context.Context, record slog.Record) error {
	var rec otellog.Record
	rec.SetTimestamp(record.Time)
	rec.SetSeverity(toOTelLevel(record.Level))
	rec.SetSeverityText(record.Level.String())
	rec.SetBody(otellog.StringValue(record.Message))
	attrs := make([]otellog.KeyValue, 0, len(h.groups)+len(h.attrs)+record.NumAttrs())
	for _, group := range h.groups {
		attrs = append(attrs, otellog.String("group", group))
	}
	for _, attr := range h.attrs {
		attrs = append(attrs, slogAttr(attr))
	}
	record.Attrs(func(attr slog.Attr) bool { attrs = append(attrs, slogAttr(attr)); return true })
	rec.AddAttributes(attrs...)
	h.logger.Emit(ctx, rec)
	return nil
}
func (h *otelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &otelHandler{logger: h.logger, minLevel: h.minLevel, attrs: append(append([]slog.Attr{}, h.attrs...), attrs...), groups: h.groups}
}
func (h *otelHandler) WithGroup(name string) slog.Handler {
	return &otelHandler{logger: h.logger, minLevel: h.minLevel, attrs: h.attrs, groups: append(append([]string{}, h.groups...), name)}
}
func toOTelLevel(level slog.Level) otellog.Severity {
	switch {
	case level >= slog.LevelError+4:
		return otellog.SeverityFatal
	case level >= slog.LevelError:
		return otellog.SeverityError
	case level >= slog.LevelWarn:
		return otellog.SeverityWarn
	case level >= slog.LevelInfo:
		return otellog.SeverityInfo
	default:
		return otellog.SeverityDebug
	}
}
func slogAttr(attr slog.Attr) otellog.KeyValue {
	switch attr.Value.Kind() {
	case slog.KindString:
		return otellog.String(attr.Key, attr.Value.String())
	case slog.KindInt64:
		return otellog.Int64(attr.Key, attr.Value.Int64())
	case slog.KindFloat64:
		return otellog.Float64(attr.Key, attr.Value.Float64())
	case slog.KindBool:
		return otellog.Bool(attr.Key, attr.Value.Bool())
	default:
		return otellog.String(attr.Key, attr.Value.String())
	}
}

type multiHandler struct{ handlers []slog.Handler }

func newMultiHandler(handlers ...slog.Handler) *multiHandler {
	return &multiHandler{handlers: handlers}
}
func (m *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}
func (m *multiHandler) Handle(ctx context.Context, record slog.Record) error {
	for _, h := range m.handlers {
		if h.Enabled(ctx, record.Level) {
			if err := h.Handle(ctx, record); err != nil {
				return err
			}
		}
	}
	return nil
}
func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}
	return &multiHandler{handlers: handlers}
}
func (m *multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithGroup(name)
	}
	return &multiHandler{handlers: handlers}
}
