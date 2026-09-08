package session

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/projectdiscovery/goflags"
	"github.com/projectdiscovery/ratelimit"
	"github.com/serrapa/analitix/pkg/providerconfig"
	"github.com/serrapa/analitix/pkg/types"
)

type Config struct {
	Timeout                   time.Duration
	RateLimit                 int
	RateLimits                map[string]goflags.RateLimit
	DefaultProviderRateLimits map[string]types.RateLimitSpec
	ProviderConfig            *providerconfig.Config
	Proxy                     string
	Delay                     time.Duration
	Retries                   int
}

type Client struct {
	HTTPClient       *http.Client
	ProviderConfig   *providerconfig.Config
	globalLimiter    *ratelimit.Limiter
	providerLimiters map[string]*ratelimit.Limiter
	delay            time.Duration
	retries          int
	mu               sync.Mutex
}

func New(ctx context.Context, cfg Config) *Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.ProviderConfig == nil {
		cfg.ProviderConfig = &providerconfig.Config{}
	}
	if cfg.Retries <= 0 {
		cfg.Retries = 1
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = http.ProxyFromEnvironment
	if cfg.Proxy != "" {
		if proxyURL, err := url.Parse(cfg.Proxy); err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	client := &Client{
		HTTPClient: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
		},
		ProviderConfig:   cfg.ProviderConfig,
		providerLimiters: make(map[string]*ratelimit.Limiter),
		delay:            cfg.Delay,
		retries:          cfg.Retries,
	}

	if cfg.RateLimit > 0 {
		client.globalLimiter = ratelimit.New(ctx, uint(cfg.RateLimit), time.Second)
	} else {
		client.globalLimiter = ratelimit.NewUnlimited(ctx)
	}

	for name, spec := range cfg.DefaultProviderRateLimits {
		if spec.MaxCount == 0 || spec.Duration <= 0 {
			continue
		}
		client.providerLimiters[name] = ratelimit.New(ctx, spec.MaxCount, spec.Duration)
	}

	for name, rl := range cfg.RateLimits {
		if rl.MaxCount == 0 || rl.Duration <= 0 {
			continue
		}
		client.providerLimiters[name] = ratelimit.New(ctx, rl.MaxCount, rl.Duration)
	}

	return client
}

func (c *Client) Close() {
	if c.globalLimiter != nil {
		c.globalLimiter.Stop()
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	for _, limiter := range c.providerLimiters {
		limiter.Stop()
	}
}

func (c *Client) Get(ctx context.Context, providerName, target string) ([]byte, *http.Response, error) {
	var lastErr error
	for attempt := 0; attempt < c.retries; attempt++ {
		body, response, err := c.getOnce(ctx, providerName, target)
		if err == nil {
			return body, response, nil
		}
		lastErr = err
		if response == nil || response.StatusCode < http.StatusInternalServerError || attempt == c.retries-1 {
			return nil, response, err
		}
	}

	return nil, nil, lastErr
}

func (c *Client) getOnce(ctx context.Context, providerName, target string) ([]byte, *http.Response, error) {
	if c.globalLimiter != nil {
		c.globalLimiter.Take()
	}
	if limiter := c.providerLimiter(providerName); limiter != nil {
		limiter.Take()
	}
	if err := c.waitDelay(ctx); err != nil {
		return nil, nil, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, nil, err
	}

	request.Header.Set("User-Agent", "analitix/0.0.1-dev")
	request.Header.Set("Accept", "*/*")

	response, err := c.HTTPClient.Do(request)
	if err != nil {
		return nil, nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 10*1024*1024))
	if err != nil {
		return nil, response, fmt.Errorf("could not read response body: %w", err)
	}
	if response.StatusCode >= http.StatusInternalServerError {
		return nil, response, fmt.Errorf("temporary http status %d", response.StatusCode)
	}

	return body, response, nil
}

func (c *Client) waitDelay(ctx context.Context) error {
	if c.delay <= 0 {
		return nil
	}

	timer := time.NewTimer(c.delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (c *Client) providerLimiter(name string) *ratelimit.Limiter {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.providerLimiters[name]
}
