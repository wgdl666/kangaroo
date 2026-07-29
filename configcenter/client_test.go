package configcenter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/wgdl666/kangaroo/env"
)

func TestFetchAndConditionalGet(t *testing.T) {
	var hits atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/psms/wg.mirror.hub/envs/ppe_mirror_zby/bundle", func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Header.Get("If-None-Match") == `W/"abc"` {
			w.Header().Set("ETag", `W/"abc"`)
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", `W/"abc"`)
		_ = json.NewEncoder(w).Encode(Bundle{
			PSM: "wg.mirror.hub", Environment: "ppe_mirror_zby",
			ReleaseID: "rel_1", Generation: 1, ETag: `W/"abc"`,
			Settings: map[string]any{"xx_wg": map[string]any{"region": "CN"}},
			Secrets:  map[string]string{"TOKEN": "secret"},
		})
	})
	mux.HandleFunc("/api/v1/psms/wg.mirror.hub/envs/ppe_mirror_zby/sdk/heartbeat", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"ok":true}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := New(Options{
		BaseURL:          srv.URL,
		Vars:             env.Vars{PSM: "wg.mirror.hub", Env: "ppe_mirror_zby", Region: "CN"},
		InstanceID:       "test-1",
		DisableHeartbeat: false,
	})
	if err != nil {
		t.Fatal(err)
	}

	snap, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if snap.Bundle.Generation != 1 || snap.Fallback {
		t.Fatalf("snap=%+v", snap)
	}
	region, ok := snap.Setting("xx_wg.region")
	if !ok || region != "CN" {
		t.Fatalf("region=%v ok=%v", region, ok)
	}
	if sec, ok := snap.Secret("TOKEN"); !ok || sec != "secret" {
		t.Fatalf("secret=%q ok=%v", sec, ok)
	}

	snap2, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if snap2.ETag != `W/"abc"` {
		t.Fatalf("etag=%q", snap2.ETag)
	}
	if hits.Load() < 2 {
		t.Fatalf("hits=%d", hits.Load())
	}
}

func TestPPEFallbackToProd(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/psms/wg.mirror.hub/envs/ppe_mirror_zby/bundle", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	mux.HandleFunc("/api/v1/psms/wg.mirror.hub/envs/prod/bundle", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `W/"prod1"`)
		_ = json.NewEncoder(w).Encode(Bundle{
			PSM: "wg.mirror.hub", Environment: "prod",
			ReleaseID: "rel_prod", Generation: 9, ETag: `W/"prod1"`,
			Settings: map[string]any{"port": float64(50051)},
			Secrets:  map[string]string{},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := New(Options{
		BaseURL:          srv.URL,
		Vars:             env.Vars{PSM: "wg.mirror.hub", Env: "ppe_mirror_zby", Region: "CN"},
		DisableHeartbeat: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	snap, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !snap.Fallback || snap.EffectiveEnv != "prod" || snap.RequestedEnv != "ppe_mirror_zby" {
		t.Fatalf("snap=%+v", snap)
	}
	if snap.Bundle.Generation != 9 {
		t.Fatalf("generation=%d", snap.Bundle.Generation)
	}
}

func TestProdMissingFails(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/psms/wg.mirror.hub/envs/prod/bundle", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := New(Options{
		BaseURL:          srv.URL,
		Vars:             env.Vars{PSM: "wg.mirror.hub", Env: "prod", Region: "CN"},
		DisableHeartbeat: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Fetch(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAuthHeader(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/psms/wg.mirror.hub/envs/prod/bundle", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("ETag", `W/"1"`)
		_ = json.NewEncoder(w).Encode(Bundle{
			PSM: "wg.mirror.hub", Environment: "prod", ReleaseID: "r", Generation: 1, ETag: `W/"1"`,
			Settings: map[string]any{}, Secrets: map[string]string{},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := New(Options{
		BaseURL: srv.URL, Token: "tok",
		Vars:             env.Vars{PSM: "wg.mirror.hub", Env: "prod", Region: "CN"},
		DisableHeartbeat: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Fetch(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestNewRequiresBaseURL(t *testing.T) {
	t.Setenv(KeyBaseURL, "")
	_, err := New(Options{Vars: env.Vars{PSM: "a", Env: "prod", Region: "CN"}})
	if err == nil || !strings.Contains(err.Error(), KeyBaseURL) {
		t.Fatalf("err=%v", err)
	}
}
