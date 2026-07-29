package configcenter

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/wgdl666/kangaroo/env"
)

// Options configures the Config Center SDK client.
type Options struct {
	// BaseURL is the Config Center origin (no trailing path). Falls back to XX_WG_CONFIG_CENTER_URL.
	BaseURL string
	// Token is the SDK bearer token. Falls back to XX_WG_CONFIG_CENTER_TOKEN.
	Token string
	// Vars identifies psm/env/region. If zero, uses env package globals (env.MustInit).
	Vars env.Vars
	// InstanceID is reported in heartbeats. Defaults to XX_WG_INSTANCE_ID or hostname.
	InstanceID string
	// PollInterval defaults to DefaultPollInterval (5s).
	PollInterval time.Duration
	// HTTPClient overrides the default client (10s timeout).
	HTTPClient *http.Client
	// OnUpdate is called after a successful new snapshot is installed (not on 304).
	OnUpdate func(Snapshot)
	// OnError is called for poll/heartbeat failures after Start.
	OnError func(error)
	// DisableHeartbeat skips POST /sdk/heartbeat.
	DisableHeartbeat bool
}

// Client pulls published bundles and keeps an atomic local snapshot.
type Client struct {
	opts Options
	vars env.Vars
	http *http.Client

	mu      sync.RWMutex
	current *Snapshot

	stopOnce sync.Once
	stopCh   chan struct{}
	wg       sync.WaitGroup
	started  bool
}

// New builds a client. Call Start (or Fetch once) before Current.
func New(opts Options) (*Client, error) {
	opts.BaseURL = strings.TrimRight(strings.TrimSpace(firstNonEmpty(opts.BaseURL, os.Getenv(KeyBaseURL))), "/")
	opts.Token = strings.TrimSpace(firstNonEmpty(opts.Token, os.Getenv(KeyToken)))
	if opts.BaseURL == "" {
		return nil, fmt.Errorf("configcenter: %s (or Options.BaseURL) is required", KeyBaseURL)
	}

	vars := opts.Vars
	if vars.PSM == "" && vars.Env == "" && vars.Region == "" {
		vars = env.Vars{PSM: env.PSM, Env: env.Env, Region: env.Region}
	}
	if err := vars.Validate(); err != nil {
		return nil, err
	}

	if opts.PollInterval <= 0 {
		opts.PollInterval = DefaultPollInterval
	}
	if opts.InstanceID == "" {
		opts.InstanceID = defaultInstanceID()
	}
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	return &Client{
		opts:   opts,
		vars:   vars,
		http:   httpClient,
		stopCh: make(chan struct{}),
	}, nil
}

// Start does an initial Fetch, then polls every PollInterval until Stop.
func (c *Client) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return fmt.Errorf("configcenter: already started")
	}
	c.started = true
	c.mu.Unlock()

	if _, err := c.Fetch(ctx); err != nil {
		c.mu.Lock()
		c.started = false
		c.mu.Unlock()
		return err
	}

	c.wg.Add(1)
	go c.pollLoop()
	return nil
}

// Stop ends the background poller. Safe to call multiple times.
func (c *Client) Stop() {
	c.stopOnce.Do(func() { close(c.stopCh) })
	c.wg.Wait()
}

// Current returns the latest snapshot, or false if none loaded yet.
func (c *Client) Current() (Snapshot, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.current == nil {
		return Snapshot{}, false
	}
	return *c.current, true
}

// Vars returns the platform env vars used by this client.
func (c *Client) Vars() env.Vars { return c.vars }

// Fetch pulls the published bundle once (with PPE→prod fallback) and atomically
// installs it when content changed. Returns the installed (or unchanged) snapshot.
func (c *Client) Fetch(ctx context.Context) (Snapshot, error) {
	snap, unchanged, err := c.pull(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	if unchanged {
		c.mu.RLock()
		defer c.mu.RUnlock()
		if c.current == nil {
			return Snapshot{}, fmt.Errorf("configcenter: unexpected empty snapshot after 304")
		}
		return *c.current, nil
	}
	c.install(snap)
	if !c.opts.DisableHeartbeat {
		if hbErr := c.postHeartbeat(ctx, snap, true, ""); hbErr != nil && c.opts.OnError != nil {
			c.opts.OnError(hbErr)
		}
	}
	return snap, nil
}

func (c *Client) pull(ctx context.Context) (Snapshot, bool /*unchanged*/, error) {
	requested := c.vars.Env
	b, outETag, notModified, err := c.fetchBundle(ctx, requested, c.etagFor(requested))
	if err == nil {
		if notModified {
			return Snapshot{}, true, nil
		}
		return Snapshot{
			Bundle:       b,
			ETag:         outETag,
			RequestedEnv: requested,
			EffectiveEnv: requested,
			Fallback:     false,
		}, false, nil
	}
	if !errors.Is(err, ErrNotFound) || !c.vars.IsPPE() {
		return Snapshot{}, false, err
	}

	// PPE miss → fall back to prod (server does not do this).
	b, outETag, notModified, err = c.fetchBundle(ctx, env.EnvProd, c.etagFor(env.EnvProd))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Snapshot{}, false, fmt.Errorf("configcenter: no published config for %s/%s (and prod fallback missing): %w", c.vars.PSM, requested, err)
		}
		return Snapshot{}, false, err
	}
	if notModified {
		return Snapshot{}, true, nil
	}
	return Snapshot{
		Bundle:       b,
		ETag:         outETag,
		RequestedEnv: requested,
		EffectiveEnv: env.EnvProd,
		Fallback:     true,
	}, false, nil
}

func (c *Client) etagFor(environment string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.current == nil || c.current.EffectiveEnv != environment {
		return ""
	}
	return c.current.ETag
}

func (c *Client) install(snap Snapshot) {
	c.mu.Lock()
	c.current = &snap
	c.mu.Unlock()
	if c.opts.OnUpdate != nil {
		c.opts.OnUpdate(snap)
	}
}

func (c *Client) pollLoop() {
	defer c.wg.Done()
	ticker := time.NewTicker(c.opts.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), c.opts.PollInterval)
			_, err := c.Fetch(ctx)
			cancel()
			if err != nil && c.opts.OnError != nil {
				c.opts.OnError(err)
			}
		}
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
