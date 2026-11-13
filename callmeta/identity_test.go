package callmeta

import (
	"context"
	"encoding/json"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"net/http"
	"testing"
)

func TestIdentityBoundariesAndFrozenTask(t *testing.T) {
	span := trace.NewSpanContext(trace.SpanContextConfig{TraceID: trace.TraceID{1}, SpanID: trace.SpanID{2}, TraceFlags: trace.FlagsSampled})
	origin := WithIdentity(trace.ContextWithSpanContext(context.Background(), span), "mirror", "a")
	for _, propagator := range []propagation.TextMapPropagator{Propagator{}, RequestBusinessPropagator{}} {
		headers := http.Header{}
		propagator.Inject(origin, HTTPHeaders(headers))
		restored := propagator.Extract(context.Background(), HTTPHeaders(headers))
		if device, user := Identity(restored); device != "mirror" || user != "a" {
			t.Fatal(headers)
		}
		if trace.SpanContextFromContext(restored).TraceID() != span.TraceID() {
			t.Fatal("lost trace")
		}
		propagator.Inject(context.Background(), HTTPHeaders(headers))
		if headers.Get(DeviceID) != "" || headers.Get(UserID) != "" {
			t.Fatal("reused carrier leaked identity")
		}
		empty := propagator.Extract(origin, propagation.MapCarrier{})
		if device, user := Identity(empty); device != "" || user != "" {
			t.Fatal("inherited prior request")
		}
	}
	task := map[string]any{}
	SaveTask(origin, task)
	raw, err := json.Marshal(task)
	if err != nil {
		t.Fatal(err)
	}
	var stored map[string]any
	if err = json.Unmarshal(raw, &stored); err != nil {
		t.Fatal(err)
	}
	switched := WithIdentity(origin, "mirror", "b")
	restored := RestoreTask(switched, stored)
	restored = FromHeaders(context.Background(), Headers(restored))
	if device, user := Identity(restored); device != "mirror" || user != "a" {
		t.Fatal("task adopted new session user")
	}
	if _, user := Identity(switched); user != "b" {
		t.Fatal("mutated caller")
	}
	if _, user := Identity(origin); user != "a" {
		t.Fatal("mutated parent")
	}
}

func TestIdentityRejectsHeaderInjection(t *testing.T) {
	for _, value := range []string{"user\nforged", "user\rforged", string(make([]byte, 257))} {
		if ValidIdentityValue(value) {
			t.Fatal("accepted invalid identity")
		}
		ctx := Propagator{}.Extract(context.Background(), propagation.MapCarrier{UserID: value})
		if _, got := Identity(ctx); got != "" {
			t.Fatal("restored invalid identity")
		}
	}
}
