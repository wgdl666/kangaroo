package configcenter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/wgdl666/kangaroo/env"
)

// Options configures the Config Center HTTP endpoint.
// PSM / Env / Region are taken from kangaroo/env on each fetch.
type Options struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

var (
	ErrNotFound     = errors.New("configcenter: bundle not found")
	ErrUnauthorized = errors.New("configcenter: unauthorized")
	ErrNotInit      = errors.New("configcenter: call MustInit before GetConfig/GetSecrets")

	mu     sync.RWMutex
	inited bool
	base   string
	token  string
	httpC  *http.Client
)

// MustInit sets the Config Center endpoint. Call once at process start.
// Panics if Options.BaseURL is empty.
func MustInit(opts Options) {
	u := strings.TrimRight(strings.TrimSpace(opts.BaseURL), "/")
	if u == "" {
		panic("configcenter: Options.BaseURL is required")
	}
	c := opts.HTTPClient
	if c == nil {
		c = &http.Client{Timeout: 10 * time.Second}
	}
	mu.Lock()
	base = u
	token = strings.TrimSpace(opts.Token)
	httpC = c
	inited = true
	mu.Unlock()
}

// GetConfig pulls 应用配置 (settings) and decodes into T.
// T is typically a struct matching settings JSON, or map[string]any.
func GetConfig[T any](ctx context.Context) (T, error) {
	var zero T
	b, err := fetchBundle(ctx)
	if err != nil {
		return zero, err
	}
	out, err := decodeAs[T](b.Settings)
	if err != nil {
		return zero, fmt.Errorf("configcenter: decode config: %w", err)
	}
	return out, nil
}

// GetSecrets pulls 密钥配置 (secrets) and decodes into T.
// T is typically a struct with json tags matching secret keys, or map[string]string.
func GetSecrets[T any](ctx context.Context) (T, error) {
	var zero T
	b, err := fetchBundle(ctx)
	if err != nil {
		return zero, err
	}
	out, err := decodeAs[T](b.Secrets)
	if err != nil {
		return zero, fmt.Errorf("configcenter: decode secrets: %w", err)
	}
	return out, nil
}

func decodeAs[T any](v any) (T, error) {
	var zero T
	if v == nil {
		return zero, nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return zero, err
	}
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		return zero, err
	}
	return out, nil
}

func clientState() (baseURL, tok string, c *http.Client, err error) {
	mu.RLock()
	defer mu.RUnlock()
	if !inited {
		return "", "", nil, ErrNotInit
	}
	return base, token, httpC, nil
}

func platform() (psm, environment, region string, err error) {
	if err := env.Validate(); err != nil {
		return "", "", "", fmt.Errorf("configcenter: platform env not ready (call env.MustInit): %w", err)
	}
	return env.PSM, env.Env, env.Region, nil
}

func resetForTest() {
	mu.Lock()
	inited = false
	base, token, httpC = "", "", nil
	mu.Unlock()
}
