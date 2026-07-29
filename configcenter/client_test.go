package configcenter

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wgdl666/kangaroo/env"
)

type testConfig struct {
	Port int `json:"port"`
	XXWG struct {
		Region string `json:"region"`
	} `json:"xx_wg"`
	From string `json:"from"`
}

type testSecrets struct {
	Token string `json:"TOKEN"`
	K     string `json:"K"`
}

func withPlatform(t *testing.T, psm, environment, region string) {
	t.Helper()
	prev := [3]string{env.PSM, env.Env, env.Region}
	env.PSM, env.Env, env.Region = psm, environment, region
	t.Cleanup(func() {
		env.PSM, env.Env, env.Region = prev[0], prev[1], prev[2]
		resetForTest()
	})
}

func TestGetConfigAndSecrets(t *testing.T) {
	withPlatform(t, "wg.mirror.hub", "ppe_mirror_zby", "CN")
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/psms/wg.mirror.hub/envs/ppe_mirror_zby/bundle", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"psm": "wg.mirror.hub", "environment": "ppe_mirror_zby",
			"settings": map[string]any{"xx_wg": map[string]any{"region": "CN"}, "port": float64(50051)},
			"secrets":  map[string]string{"TOKEN": "secret"},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	MustInit(Options{BaseURL: srv.URL})

	cfg, err := GetConfig[testConfig](context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 50051 || cfg.XXWG.Region != "CN" {
		t.Fatalf("config=%+v", cfg)
	}

	sec, err := GetSecrets[testSecrets](context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if sec.Token != "secret" {
		t.Fatalf("secrets=%+v", sec)
	}

	raw, err := GetConfig[map[string]any](context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if raw["port"] != float64(50051) {
		t.Fatalf("map config=%v", raw)
	}
}

func TestPPEFallback(t *testing.T) {
	withPlatform(t, "wg.mirror.hub", "ppe_mirror_zby", "CN")
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/psms/wg.mirror.hub/envs/ppe_mirror_zby/bundle", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	mux.HandleFunc("/api/v1/psms/wg.mirror.hub/envs/prod/bundle", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"settings": map[string]any{"xx_wg": map[string]any{"region": "CN"}, "from": "prod"},
			"secrets":  map[string]string{"K": "v"},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	MustInit(Options{BaseURL: srv.URL})
	cfg, err := GetConfig[testConfig](context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.From != "prod" {
		t.Fatalf("cfg=%+v", cfg)
	}
}

func TestRegionMismatch(t *testing.T) {
	withPlatform(t, "wg.mirror.hub", "prod", "CN")
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/psms/wg.mirror.hub/envs/prod/bundle", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"settings": map[string]any{"xx_wg": map[string]any{"region": "US"}},
			"secrets":  map[string]string{},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	MustInit(Options{BaseURL: srv.URL})
	_, err := GetConfig[testConfig](context.Background())
	if err == nil {
		t.Fatal("expected region mismatch")
	}
}

func TestNotInit(t *testing.T) {
	resetForTest()
	_, err := GetConfig[testConfig](context.Background())
	if !errors.Is(err, ErrNotInit) {
		t.Fatalf("err=%v", err)
	}
}
