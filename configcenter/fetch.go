package configcenter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/wgdl666/kangaroo/env"
)

type bundle struct {
	PSM         string            `json:"psm"`
	Environment string            `json:"environment"`
	ReleaseID   string            `json:"release_id"`
	Generation  int64             `json:"generation"`
	ETag        string            `json:"etag"`
	Settings    map[string]any    `json:"settings"`
	Secrets     map[string]string `json:"secrets"`
}

func fetchBundle(ctx context.Context) (bundle, error) {
	baseURL, tok, client, err := clientState()
	if err != nil {
		return bundle{}, err
	}
	psm, environment, region, err := platform()
	if err != nil {
		return bundle{}, err
	}

	b, err := getBundle(ctx, client, baseURL, tok, psm, environment)
	if err == nil {
		return b, matchRegion(b, region)
	}
	if !errors.Is(err, ErrNotFound) || !env.IsPPE() {
		return bundle{}, err
	}

	b, err = getBundle(ctx, client, baseURL, tok, psm, env.EnvProd)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return bundle{}, fmt.Errorf("configcenter: no published config for %s/%s (and prod fallback missing): %w", psm, environment, err)
		}
		return bundle{}, err
	}
	return b, matchRegion(b, region)
}

func getBundle(ctx context.Context, client *http.Client, baseURL, tok, psm, environment string) (bundle, error) {
	u := strings.TrimRight(baseURL, "/") + "/api/v1/psms/" +
		url.PathEscape(psm) + "/envs/" + url.PathEscape(environment) + "/bundle"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return bundle{}, err
	}
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("X-Config-Token", tok)
	}
	resp, err := client.Do(req)
	if err != nil {
		return bundle{}, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotFound:
		return bundle{}, ErrNotFound
	case http.StatusUnauthorized, http.StatusForbidden:
		return bundle{}, ErrUnauthorized
	case http.StatusOK:
		// ok
	default:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return bundle{}, fmt.Errorf("configcenter: get bundle: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return bundle{}, err
	}
	var b bundle
	if err := json.Unmarshal(raw, &b); err != nil {
		return bundle{}, fmt.Errorf("configcenter: decode bundle: %w", err)
	}
	return b, nil
}

// matchRegion checks settings.xx_wg.region against env.Region when present.
func matchRegion(b bundle, region string) error {
	v, ok := getPath(b.Settings, "xx_wg.region")
	if !ok {
		return nil
	}
	got := strings.TrimSpace(fmt.Sprint(v))
	if got == "" || got == region {
		return nil
	}
	return fmt.Errorf("configcenter: region mismatch: env=%s bundle.xx_wg.region=%s", region, got)
}

func getPath(m map[string]any, path string) (any, bool) {
	parts := strings.Split(path, ".")
	var cur any = m
	for _, p := range parts {
		obj, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := obj[p]
		if !ok {
			return nil, false
		}
		cur = next
	}
	return cur, true
}
