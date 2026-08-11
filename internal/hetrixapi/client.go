// Modified by the hetrixtools-sync project from the Apache-2.0 upstream source.
package hetrixtools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultBaseURL is the default root URL for HetrixTools API endpoints.
	DefaultBaseURL = "https://api.hetrixtools.com"
	// DefaultV3BaseURL is the default base URL for HetrixTools REST endpoints.
	DefaultV3BaseURL = DefaultBaseURL + "/v3"
	// DefaultV2BaseURL is the default base URL for older HetrixTools token-path endpoints.
	DefaultV2BaseURL = DefaultBaseURL + "/v2"
)

// Client calls HetrixTools APIs through semantic resource methods.
type Client struct {
	v3BaseURL      string
	v2BaseURL      string
	token          string
	http           *http.Client
	v2Interval     time.Duration
	v3Interval     time.Duration
	v2Limiter      *rateLimiter
	v3UserLimiter  *rateLimiter
	limiterMu      sync.Mutex
	v3Limiters     map[string]*rateLimiter
	cacheMu        sync.Mutex
	uptimeMonitors []UptimeMonitor
	blMonitors     []BlacklistMonitor
}

type rateLimiter struct {
	mu           sync.Mutex
	lastRequest  time.Time
	blockedUntil time.Time
}

type requestLimiter struct {
	limiter  *rateLimiter
	interval time.Duration
	scope    string
}

// Option customizes a Client created by NewClient or NewClientWithBaseURL.
type Option func(*Client)

// WithHTTPClient configures the HTTP client used for API requests.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		if httpClient != nil {
			c.http = httpClient
		}
	}
}

// WithMinimumRequestInterval configures client-side pacing for both v2 and v3
// API calls. Passing zero disables pacing and is intended for tests, examples,
// or callers that provide their own limiter.
func WithMinimumRequestInterval(interval time.Duration) Option {
	return func(c *Client) {
		if interval >= 0 {
			c.v2Interval = interval
			c.v3Interval = interval
		}
	}
}

// WithV2RequestInterval configures pacing for legacy v2 token-path API calls,
// which share one client-wide limiter bucket.
func WithV2RequestInterval(interval time.Duration) Option {
	return func(c *Client) {
		if interval >= 0 {
			c.v2Interval = interval
		}
	}
}

// WithV3RequestInterval configures pacing for v3 REST API calls. The interval
// applies to both the user-level limiter and per-endpoint limiter.
func WithV3RequestInterval(interval time.Duration) Option {
	return func(c *Client) {
		if interval >= 0 {
			c.v3Interval = interval
		}
	}
}

// WithV2BaseURL overrides the base URL used for older token-path endpoints.
func WithV2BaseURL(baseURL string) Option {
	return func(c *Client) {
		if strings.TrimSpace(baseURL) != "" {
			c.v2BaseURL = strings.TrimRight(baseURL, "/")
		}
	}
}

// WithV3BaseURL overrides the base URL used for REST endpoints.
func WithV3BaseURL(baseURL string) Option {
	return func(c *Client) {
		if strings.TrimSpace(baseURL) != "" {
			c.v3BaseURL = strings.TrimRight(baseURL, "/")
		}
	}
}

// NewClient returns a client configured with the default HetrixTools base URLs.
// The returned client uses a 30-second HTTP timeout and default request pacing
// suitable for HetrixTools rate limits.
func NewClient(token string, options ...Option) *Client {
	return NewClientWithBaseURL(DefaultBaseURL, token, options...)
}

// NewClientWithBaseURL returns a client configured with a custom API root URL.
// For compatibility, baseURL may also be a versioned /v2 or /v3 URL.
// The returned client uses a 30-second HTTP timeout and default request pacing
// suitable for HetrixTools rate limits.
func NewClientWithBaseURL(baseURL string, token string, options ...Option) *Client {
	v2BaseURL, v3BaseURL := versionedBaseURLs(baseURL)

	c := &Client{
		v3BaseURL: v3BaseURL,
		v2BaseURL: v2BaseURL,
		token:     token,
		http: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) > 0 && req.URL.Host != via[0].URL.Host {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
		// HetrixTools v1/v2 allows 120 requests/minute across all v1/v2
		// endpoints. v3 is limited per user and per API call, so the client uses
		// a separate limiter per v3 method/path while still honoring 429 retries.
		v2Interval:    500 * time.Millisecond,
		v3Interval:    500 * time.Millisecond,
		v2Limiter:     &rateLimiter{},
		v3UserLimiter: &rateLimiter{},
		v3Limiters:    map[string]*rateLimiter{},
	}
	for _, option := range options {
		option(c)
	}
	return c
}

func (c *Client) getJSON(ctx context.Context, path string, query map[string]string, out any) error {
	body, err := c.doV3(ctx, http.MethodGet, path, query, nil)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(body, out)
}

func (c *Client) postJSON(ctx context.Context, path string, body any, out any) error {
	responseBody, err := c.doV3(ctx, http.MethodPost, path, nil, body)
	if err != nil {
		return err
	}
	if out == nil || len(responseBody) == 0 {
		return nil
	}
	return json.Unmarshal(responseBody, out)
}

func (c *Client) putJSON(ctx context.Context, path string, body any, out any) error {
	responseBody, err := c.doV3(ctx, http.MethodPut, path, nil, body)
	if err != nil {
		return err
	}
	if out == nil || len(responseBody) == 0 {
		return nil
	}
	return json.Unmarshal(responseBody, out)
}

func (c *Client) deleteJSON(ctx context.Context, path string, body any) error {
	_, err := c.doV3(ctx, http.MethodDelete, path, nil, body)
	return err
}

func (c *Client) getEndpoint(ctx context.Context, path string, query map[string]string) ([]byte, error) {
	return c.doV3(ctx, http.MethodGet, path, query, nil)
}

func (c *Client) doV3(ctx context.Context, method string, path string, query map[string]string, body any) ([]byte, error) {
	return c.do(ctx, c.v3BaseURL, method, path, query, body, true, []requestLimiter{
		{limiter: c.v3UserLimiter, interval: c.v3Interval, scope: "user"},
		{limiter: c.v3EndpointLimiter(method, path), interval: c.v3Interval, scope: "endpoint"},
	})
}

func (c *Client) doV2JSON(ctx context.Context, method string, path string, body any) ([]byte, error) {
	return c.do(ctx, c.v2BaseURL, method, c.v2Path(path), nil, body, false, []requestLimiter{{limiter: c.v2Limiter, interval: c.v2Interval}})
}

func (c *Client) doV2Form(ctx context.Context, path string, values url.Values) ([]byte, error) {
	return c.doForm(ctx, c.v2BaseURL, c.v2Path(path), values, []requestLimiter{{limiter: c.v2Limiter, interval: c.v2Interval}})
}

func decodeActionResponse(body []byte) (*ActionResponse, error) {
	var envelope struct {
		Status       string `json:"status"`
		ErrorMessage string `json:"error_message"`
		MonitorID    string `json:"monitor_id"`
		ServerID     string `json:"server_id"`
		Action       string `json:"action"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	if envelope.Status != "SUCCESS" || envelope.ErrorMessage != "" {
		message := envelope.ErrorMessage
		if message == "" {
			message = fmt.Sprintf("unexpected action status %q", envelope.Status)
		}
		return nil, Error{Response: &APIErrorResponse{Status: envelope.Status, ErrorMessage: message}}
	}
	return &ActionResponse{Status: envelope.Status, MonitorID: envelope.MonitorID, ServerID: envelope.ServerID, Action: envelope.Action}, nil
}

func (c *Client) do(ctx context.Context, baseURL string, method string, path string, query map[string]string, body any, bearerAuth bool, limiters []requestLimiter) ([]byte, error) {
	requestURL, err := requestURL(baseURL, path, query)
	if err != nil {
		return nil, err
	}

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearerAuth && c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	return c.doRequest(req, limiters)
}

func (c *Client) doForm(ctx context.Context, baseURL string, path string, values url.Values, limiters []requestLimiter) ([]byte, error) {
	requestURL, err := requestURL(baseURL, path, nil)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return c.doRequest(req, limiters)
}

func (c *Client) doRequest(req *http.Request, limiters []requestLimiter) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < 6; attempt++ {
		attemptReq := req.Clone(req.Context())
		if req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			attemptReq.Body = body
		}

		for _, limiter := range limiters {
			if err := waitForRequestSlot(attemptReq.Context(), limiter); err != nil {
				return nil, err
			}
		}

		resp, err := c.http.Do(attemptReq)
		if err != nil {
			redacted := fmt.Errorf("request failed: %s", c.redact(err.Error()))
			lastErr = redacted
			// A transport failure after a non-idempotent request is ambiguous: the
			// server may have applied it before the response was lost. Never retry
			// such a request automatically, as that can create duplicate monitors.
			if req.Method != http.MethodGet && req.Method != http.MethodHead {
				return nil, redacted
			}
			if attempt == 5 {
				return nil, redacted
			}
			if err := sleepContext(attemptReq.Context(), retryDelay(nil, attempt)); err != nil {
				return nil, err
			}
			continue
		}

		responseBody, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		updateRateLimits(resp.Header, limiters)

		if resp.StatusCode == http.StatusTooManyRequests && attempt < 5 && (req.Method == http.MethodGet || req.Method == http.MethodHead) {
			lastErr = Error{StatusCode: resp.StatusCode, Body: c.redact(string(responseBody))}
			blockRateLimiters(limiters, time.Now().Add(retryDelay(resp, attempt)))
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, Error{StatusCode: resp.StatusCode, Body: c.redact(string(responseBody))}
		}

		return responseBody, nil
	}
	return nil, lastErr
}

func waitForRequestSlot(ctx context.Context, bucket requestLimiter) error {
	if bucket.limiter == nil {
		return nil
	}
	bucket.limiter.mu.Lock()
	defer bucket.limiter.mu.Unlock()

	wait := bucket.interval - time.Since(bucket.limiter.lastRequest)
	if blockedWait := time.Until(bucket.limiter.blockedUntil); blockedWait > wait {
		wait = blockedWait
	}
	if wait > 0 {
		if err := sleepContext(ctx, wait); err != nil {
			return err
		}
	}
	bucket.limiter.lastRequest = time.Now()
	return nil
}

func updateRateLimits(headers http.Header, limiters []requestLimiter) {
	for _, bucket := range limiters {
		if bucket.scope == "" {
			continue
		}
		remaining, ok := intHeader(headers, "ratelimit-remaining-"+bucket.scope)
		if !ok || remaining > 0 {
			continue
		}
		reset, ok := resetHeader(headers, "ratelimit-reset-"+bucket.scope)
		if ok {
			blockRateLimiter(bucket.limiter, reset)
		}
	}
}

func blockRateLimiters(limiters []requestLimiter, until time.Time) {
	for _, bucket := range limiters {
		blockRateLimiter(bucket.limiter, until)
	}
}

func blockRateLimiter(limiter *rateLimiter, until time.Time) {
	if limiter == nil || until.IsZero() {
		return
	}
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	if until.After(limiter.blockedUntil) {
		limiter.blockedUntil = until
	}
}

func intHeader(headers http.Header, name string) (int, bool) {
	value := strings.TrimSpace(headers.Get(name))
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func resetHeader(headers http.Header, name string) (time.Time, bool) {
	value := strings.TrimSpace(headers.Get(name))
	if value == "" {
		return time.Time{}, false
	}
	reset, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	return time.Unix(reset, 0), true
}

func (c *Client) v3EndpointLimiter(method string, path string) *rateLimiter {
	key := method + " " + normalizedEndpointPath(path)
	c.limiterMu.Lock()
	defer c.limiterMu.Unlock()
	limiter, ok := c.v3Limiters[key]
	if !ok {
		limiter = &rateLimiter{}
		c.v3Limiters[key] = limiter
	}
	return limiter
}

func normalizedEndpointPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, part := range parts {
		if isLikelyIdentifier(part) {
			parts[i] = ":id"
		}
	}
	return "/" + strings.Join(parts, "/")
}

func isLikelyIdentifier(value string) bool {
	if len(value) < 16 {
		return false
	}
	for _, r := range value {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

func retryDelay(resp *http.Response, attempt int) time.Duration {
	if resp != nil {
		if retryAfter := strings.TrimSpace(resp.Header.Get("Retry-After")); retryAfter != "" {
			if seconds, err := strconv.Atoi(retryAfter); err == nil && seconds >= 0 {
				return time.Duration(seconds) * time.Second
			}
			if retryAt, err := http.ParseTime(retryAfter); err == nil {
				if wait := time.Until(retryAt); wait > 0 {
					return wait
				}
			}
		}
	}

	delay := time.Duration(1<<attempt) * time.Second
	if delay > 30*time.Second {
		return 30 * time.Second
	}
	return delay
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (c *Client) redact(value string) string {
	if c.token == "" {
		return value
	}
	value = strings.ReplaceAll(value, c.token, "[REDACTED]")
	value = strings.ReplaceAll(value, url.PathEscape(c.token), "[REDACTED]")
	return value
}

func (c *Client) clearMonitorCaches() {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	c.uptimeMonitors = nil
	c.blMonitors = nil
}

func (c *Client) v2Path(path string) string {
	return "/" + url.PathEscape(c.token) + "/" + strings.TrimLeft(path, "/")
}

func requestURL(baseURL string, path string, query map[string]string) (string, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return "", fmt.Errorf("endpoint path must be relative, got %q", path)
	}

	u, err := url.Parse(strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(path, "/"))
	if err != nil {
		return "", err
	}

	values := u.Query()
	for key, value := range query {
		if value != "" {
			values.Set(key, value)
		}
	}
	u.RawQuery = values.Encode()

	return u.String(), nil
}

func versionedBaseURLs(baseURL string) (string, string) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if strings.HasSuffix(baseURL, "/v2") || strings.HasSuffix(baseURL, "/v3") {
		baseURL = baseURL[:len(baseURL)-3]
	}
	return baseURL + "/v2", baseURL + "/v3"
}

// Error is the common error type returned by the client for HetrixTools API failures.
type Error struct {
	StatusCode int
	Body       string
	Response   *APIErrorResponse
}

// Error returns a human-readable HetrixTools API error string.
func (e Error) Error() string {
	if e.Response != nil {
		if e.Response.ErrorMessage != "" {
			return e.Response.ErrorMessage
		}
		return "hetrixtools API action failed"
	}
	if e.Body == "" {
		return fmt.Sprintf("hetrixtools API returned HTTP %d", e.StatusCode)
	}
	return fmt.Sprintf("hetrixtools API returned HTTP %d: %s", e.StatusCode, e.Body)
}

// IsNotFound reports whether err is a HetrixTools 404 response.
func IsNotFound(err error) bool {
	var apiErr Error
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}
