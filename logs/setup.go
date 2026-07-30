package logs

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
	"go.opentelemetry.io/otel/trace"
)

// Config 统一描述服务日志的导出方式；业务服务只提供身份和部署配置，不自行拼装 slog handler。
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	Endpoint       string
	Headers        map[string]string
	MinLevel       string
	Console        bool
}

// Logger 是服务侧唯一需要持有的日志器，slog 仅保留为 Kangaroo 的内部实现细节。
type Logger struct {
	inner         *slog.Logger
	logProvider   *sdklog.LoggerProvider
	traceProvider *sdktrace.TracerProvider
	tracer        trace.Tracer
}

var defaultLogger = &Logger{inner: slog.Default()}

// Get 返回当前服务初始化后的统一日志器，供不持有依赖的基础设施代码使用。
func Get() *Logger { return defaultLogger }

// Setup 在服务启动时安装全局日志器，并按 Config 将日志导出到 OTLP 平台。
func Setup(ctx context.Context, cfg Config) (*Logger, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(cfg.ServiceName) == "" {
		return nil, fmt.Errorf("logs: service name is required")
	}

	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(cfg.ServiceName),
		semconv.ServiceVersion(cfg.ServiceVersion),
		attribute.String("deployment.environment.name", cfg.Environment),
	)
	provider := sdklog.NewLoggerProvider(sdklog.WithResource(res))
	traceProvider := sdktrace.NewTracerProvider(sdktrace.WithResource(res))
	if strings.TrimSpace(cfg.Endpoint) != "" {
		exporter, err := otlploghttp.New(ctx,
			otlploghttp.WithEndpoint(cfg.Endpoint),
			otlploghttp.WithHeaders(cfg.Headers),
		)
		if err != nil {
			_ = provider.Shutdown(ctx)
			return nil, fmt.Errorf("logs: create OTLP exporter: %w", err)
		}
		traceExporter, err := otlptracehttp.New(ctx,
			otlptracehttp.WithEndpoint(cfg.Endpoint),
			otlptracehttp.WithHeaders(cfg.Headers),
		)
		if err != nil {
			_ = provider.Shutdown(ctx)
			return nil, fmt.Errorf("logs: create OTLP trace exporter: %w", err)
		}
		provider = sdklog.NewLoggerProvider(
			sdklog.WithResource(res),
			sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		)
		traceProvider = sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
			sdktrace.WithBatcher(traceExporter),
		)
	}
	global.SetLoggerProvider(provider)
	otel.SetTracerProvider(traceProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))

	handlers := []slog.Handler{NewOtelHandler(provider.Logger(cfg.ServiceName), parseLevel(cfg.MinLevel))}
	if cfg.Console {
		handlers = append([]slog.Handler{slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})}, handlers...)
	}
	logger := &Logger{
		inner:         slog.New(NewMultiHandler(handlers...)),
		logProvider:   provider,
		traceProvider: traceProvider,
		tracer:        traceProvider.Tracer(cfg.ServiceName),
	}
	slog.SetDefault(logger.inner)
	defaultLogger = logger
	return logger, nil
}

// NewConsole 为命令行工具和测试提供不导出的统一日志器，避免服务代码直接构造 slog。
func NewConsole(minLevel string) *Logger {
	logger := &Logger{inner: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: parseLevel(minLevel)}))}
	// 命令行工具没有 OTLP 配置，也要安装为默认日志器，避免无依赖代码落回旧的全局 slog。
	slog.SetDefault(logger.inner)
	defaultLogger = logger
	return logger
}

// NewDiscard 为单元测试提供静默日志器，避免测试代码直接依赖 slog 的 Handler 构造细节。
func NewDiscard() *Logger {
	return &Logger{inner: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

// Info 输出普通运行信息。
func (l *Logger) Info(msg string, args ...any) { l.inner.Info(msg, args...) }

// Debug 输出调试信息。
func (l *Logger) Debug(msg string, args ...any) { l.inner.Debug(msg, args...) }

// Warn 输出可继续处理的异常信息。
func (l *Logger) Warn(msg string, args ...any) { l.inner.Warn(msg, args...) }

// Error 输出当前操作失败的信息。
func (l *Logger) Error(msg string, args ...any) { l.inner.Error(msg, args...) }

// WarnContext 输出与当前 Trace/Span 关联的告警日志。
func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.inner.WarnContext(ctx, msg, args...)
}

// Shutdown 在服务退出前冲刷已批量缓存的日志。
func (l *Logger) Shutdown(ctx context.Context) error {
	if l == nil {
		return nil
	}
	var errs []error
	if l.logProvider != nil {
		errs = append(errs, l.logProvider.Shutdown(ctx))
	}
	if l.traceProvider != nil {
		errs = append(errs, l.traceProvider.Shutdown(ctx))
	}
	return errors.Join(errs...)
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

// OtelHandler 将标准日志记录转换为 OTel Log Record，并保留传入 ctx 的 Trace/Span 关联。
type OtelHandler struct {
	logger   otellog.Logger
	minLevel slog.Level
	attrs    []slog.Attr
	groups   []string
}

func NewOtelHandler(logger otellog.Logger, minLevel slog.Level) *OtelHandler {
	return &OtelHandler{logger: logger, minLevel: minLevel}
}

func (h *OtelHandler) Enabled(_ context.Context, level slog.Level) bool { return level >= h.minLevel }

func (h *OtelHandler) Handle(ctx context.Context, record slog.Record) error {
	var rec otellog.Record
	rec.SetTimestamp(record.Time)
	rec.SetSeverity(slogLevelToOtel(record.Level))
	rec.SetSeverityText(record.Level.String())
	rec.SetBody(otellog.StringValue(record.Message))
	attrs := make([]otellog.KeyValue, 0, len(h.groups)+len(h.attrs)+record.NumAttrs())
	for _, group := range h.groups {
		attrs = append(attrs, otellog.String("group", group))
	}
	for _, attr := range h.attrs {
		attrs = append(attrs, slogAttrToOtel(attr))
	}
	record.Attrs(func(attr slog.Attr) bool { attrs = append(attrs, slogAttrToOtel(attr)); return true })
	rec.AddAttributes(attrs...)
	h.logger.Emit(ctx, rec)
	return nil
}

func (h *OtelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	combined := append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &OtelHandler{logger: h.logger, minLevel: h.minLevel, attrs: combined, groups: h.groups}
}

func (h *OtelHandler) WithGroup(name string) slog.Handler {
	groups := append(append([]string{}, h.groups...), name)
	return &OtelHandler{logger: h.logger, minLevel: h.minLevel, attrs: h.attrs, groups: groups}
}

func slogLevelToOtel(level slog.Level) otellog.Severity {
	switch {
	case level >= LevelFatal:
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

func slogAttrToOtel(attr slog.Attr) otellog.KeyValue {
	switch attr.Value.Kind() {
	case slog.KindString:
		return otellog.String(attr.Key, attr.Value.String())
	case slog.KindInt64:
		return otellog.Int64(attr.Key, attr.Value.Int64())
	case slog.KindFloat64:
		return otellog.Float64(attr.Key, attr.Value.Float64())
	case slog.KindBool:
		return otellog.Bool(attr.Key, attr.Value.Bool())
	case slog.KindTime:
		return otellog.String(attr.Key, attr.Value.Time().String())
	case slog.KindDuration:
		return otellog.String(attr.Key, attr.Value.Duration().String())
	default:
		return otellog.String(attr.Key, attr.Value.String())
	}
}

// MultiHandler 让统一 Logger 同时向控制台和 OTLP 导出，具体目的地由 Config 决定。
type MultiHandler struct{ handlers []slog.Handler }

func NewMultiHandler(handlers ...slog.Handler) *MultiHandler {
	return &MultiHandler{handlers: handlers}
}
func (m *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range m.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}
func (m *MultiHandler) Handle(ctx context.Context, record slog.Record) error {
	for _, handler := range m.handlers {
		if handler.Enabled(ctx, record.Level) {
			if err := handler.Handle(ctx, record); err != nil {
				return err
			}
		}
	}
	return nil
}
func (m *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, handler := range m.handlers {
		handlers[i] = handler.WithAttrs(attrs)
	}
	return &MultiHandler{handlers: handlers}
}
func (m *MultiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, handler := range m.handlers {
		handlers[i] = handler.WithGroup(name)
	}
	return &MultiHandler{handlers: handlers}
}
