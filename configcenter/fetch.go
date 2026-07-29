package configcenter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func (c *Client) fetchBundle(ctx context.Context, environment, etag string) (Bundle, string, bool, error) {
	u := c.bundleURL(environment)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return Bundle{}, "", false, err
	}
	c.applyAuth(req)
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return Bundle{}, "", false, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotModified:
		return Bundle{}, resp.Header.Get("ETag"), true, nil
	case http.StatusNotFound:
		return Bundle{}, "", false, ErrNotFound
	case http.StatusUnauthorized, http.StatusForbidden:
		return Bundle{}, "", false, ErrUnauthorized
	case http.StatusOK:
		// continue
	default:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return Bundle{}, "", false, fmt.Errorf("configcenter: get bundle: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return Bundle{}, "", false, err
	}
	var b Bundle
	if err := json.Unmarshal(raw, &b); err != nil {
		return Bundle{}, "", false, fmt.Errorf("configcenter: decode bundle: %w", err)
	}
	outETag := resp.Header.Get("ETag")
	if outETag == "" {
		outETag = b.ETag
	}
	if b.ETag == "" {
		b.ETag = outETag
	}
	return b, outETag, false, nil
}

func (c *Client) postHeartbeat(ctx context.Context, snap Snapshot, loadOK bool, lastErr string) error {
	if c.opts.InstanceID == "" {
		return nil
	}
	payload := map[string]any{
		"instance_id": c.opts.InstanceID,
		"release_id":  snap.Bundle.ReleaseID,
		"generation":  snap.Bundle.Generation,
		"load_ok":     loadOK,
		"last_error":  truncate(lastErr, 200),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	// Heartbeat always reports against the requested XX_WG_ENV path.
	u := strings.TrimRight(c.opts.BaseURL, "/") + "/api/v1/psms/" +
		url.PathEscape(c.vars.PSM) + "/envs/" + url.PathEscape(c.vars.Env) + "/sdk/heartbeat"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	c.applyAuth(req)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<10))
		return fmt.Errorf("configcenter: heartbeat: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

func (c *Client) bundleURL(environment string) string {
	return strings.TrimRight(c.opts.BaseURL, "/") + "/api/v1/psms/" +
		url.PathEscape(c.vars.PSM) + "/envs/" + url.PathEscape(environment) + "/bundle"
}

func (c *Client) applyAuth(req *http.Request) {
	if c.opts.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.opts.Token)
		req.Header.Set("X-Config-Token", c.opts.Token)
	}
}

func defaultInstanceID() string {
	if v := strings.TrimSpace(os.Getenv(KeyInstanceID)); v != "" {
		return v
	}
	host, err := os.Hostname()
	if err != nil || host == "" {
		return "unknown"
	}
	return host
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// Setting reads a dotted path from Snapshot.Settings (e.g. "xx_wg.region").
func (s Snapshot) Setting(path string) (any, bool) {
	return getPath(s.Bundle.Settings, path)
}

// Secret returns a secret by key.
func (s Snapshot) Secret(key string) (string, bool) {
	if s.Bundle.Secrets == nil {
		return "", false
	}
	v, ok := s.Bundle.Secrets[key]
	return v, ok
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
