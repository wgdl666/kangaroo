package callmeta

import (
	"context"
	"sync"
	"testing"

	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// 追踪只走标准 traceparent。subscene 属于当前模型步骤，不进 header、AMQP 或任务 metadata。
// turn_id、task_id、第二份 trace_id 即使被写进载体，恢复时也要丢掉。
func TestTransportWhitelistParentAndRequestBoundary(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })

	parentCtx, parent := provider.Tracer("callmeta").Start(context.Background(), "parent")
	parentCtx = WithSubscene(WithAttribution(WithBusiness(parentCtx, PromptDebug, "draft"), "mirror", "photo"), "person_detection")

	headers := Headers(parentCtx)
	allowed := map[string]struct{}{}
	for _, key := range (Propagator{}).Fields() {
		allowed[key] = struct{}{}
	}
	for key := range headers {
		if _, ok := allowed[key]; !ok {
			t.Fatalf("header outside whitelist: %s", key)
		}
	}
	for _, forbidden := range []string{"trace_id", "turn_id", "task_id", "business_subscene", "authorization"} {
		if _, ok := headers[forbidden]; ok {
			t.Fatalf("propagated %s", forbidden)
		}
	}
	if headers[BusinessLine] != "mirror" || headers[BusinessScene] != "photo" || headers[BusinessID] != "draft" {
		t.Fatalf("missing attribution headers: %v", headers)
	}
	if _, ok := headers["traceparent"]; !ok {
		t.Fatal("missing traceparent")
	}

	poisoned := map[string]any{}
	for key, value := range headers {
		poisoned[key] = value
	}
	poisoned[BusinessID] = []byte("draft")
	poisoned["trace_id"] = "second-trace"
	poisoned["turn_id"] = "turn-1"
	poisoned["task_id"] = "task-1"
	poisoned["authorization"] = "must-not-restore"
	restored := FromHeaders(context.Background(), poisoned)
	if _, id := Business(restored); id != "draft" {
		t.Fatalf("lost byte business id")
	}
	if line, scene := Attribution(restored); line != "mirror" || scene != "photo" {
		t.Fatalf("lost attribution")
	}
	if Subscene(restored) != "" {
		t.Fatal("amqp restored subscene")
	}
	if trace.SpanContextFromContext(restored).TraceID() != parent.SpanContext().TraceID() {
		t.Fatal("lost trace id")
	}

	_, child := provider.Tracer("callmeta").Start(restored, "child")
	child.End()
	parent.End()
	var childParent trace.SpanID
	for _, span := range recorder.Ended() {
		if span.Name() == "child" {
			childParent = span.Parent().SpanID()
		}
	}
	if childParent != parent.SpanContext().SpanID() {
		t.Fatalf("child parent = %s, origin = %s", childParent, parent.SpanContext().SpanID())
	}

	task := map[string]any{}
	SaveTask(parentCtx, task)
	metadata, _ := task[MetadataKey].(map[string]string)
	metadata["trace_id"] = "second-trace"
	metadata["turn_id"] = "turn-1"
	task["turn_id"] = "business-column"
	task[MetadataKey] = metadata
	persisted := RestoreTask(context.Background(), task)
	if Subscene(persisted) != "" {
		t.Fatal("task metadata restored subscene")
	}
	again := Capture(persisted)
	if again["trace_id"] != "" || again["turn_id"] != "" {
		t.Fatalf("non-whitelist survived: %v", again)
	}

	requestCarrier := propagation.MapCarrier{}
	RequestBusinessPropagator{}.Inject(parentCtx, requestCarrier)
	if requestCarrier.Get(BusinessLine) != "" || requestCarrier.Get(BusinessScene) != "" {
		t.Fatalf("request propagator copied attribution: %v", requestCarrier)
	}
	if requestCarrier.Get("traceparent") == "" || requestCarrier.Get(BusinessKey) != PromptDebug {
		t.Fatalf("request propagator dropped trace or debug link: %v", requestCarrier)
	}
	requestCtx := RequestBusinessPropagator{}.Extract(context.Background(), requestCarrier)
	if line, scene := Attribution(requestCtx); line != "" || scene != "" || Subscene(requestCtx) != "" {
		t.Fatal("request extract invented attribution")
	}
	if _, id := Business(requestCtx); id != "draft" {
		t.Fatal(requestCarrier)
	}
	if Subscene(parentCtx) != "person_detection" {
		t.Fatal("parent subscene mutated")
	}

	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := string(rune('a' + i%26))
			step := WithSubscene(WithBusiness(parentCtx, PromptDebug, id), "step")
			if Subscene(step) != "step" {
				t.Errorf("step lost")
			}
			if _, got := Business(Restore(context.Background(), Capture(step))); got != id {
				t.Errorf("crossed %s", got)
			}
		}(i)
	}
	wg.Wait()
	if _, id := Business(parentCtx); id != "draft" {
		t.Fatal("parent business mutated")
	}
}
