package callmeta

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/propagation"
)

// 两种 RPC 边界都必须拒绝半对调试关联，避免请求恢复时错用父上下文里的草稿。
func TestBusinessPairExtractionAcrossBoundaries(t *testing.T) {
	for name, propagator := range map[string]propagation.TextMapPropagator{"internal": Propagator{}, "modelhub": RequestBusinessPropagator{}} {
		for _, pair := range [][2]string{{PromptDebug, "snapshot"}, {PromptDebug, ""}, {"", "snapshot"}, {"", ""}} {
			t.Run(name+"/"+pair[0]+"/"+pair[1], func(t *testing.T) {
				parent := WithBusiness(context.Background(), PromptDebug, "previous")
				ctx := propagator.Extract(parent, propagation.MapCarrier{BusinessKey: pair[0], BusinessID: pair[1]})
				want := pair
				if want[0] == "" || want[1] == "" {
					want = [2]string{}
				}
				key, id := Business(ctx)
				if key != want[0] || id != want[1] {
					t.Fatalf("business pair = (%q,%q), want %v", key, id, want)
				}
			})
		}
	}
}
