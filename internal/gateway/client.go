// Package gateway provides LiteLLM gateway client functionality.
//
// Purpose:
//
//	Provide typed gateway probing and API operations for the ACP runtime.
//
// Responsibilities:
//   - Resolve effective gateway endpoint and authentication settings.
//   - Execute typed authorized probes for /health and /v1/models.
//   - Provide shared HTTP request construction and error shaping.
//   - Execute key generation requests against the gateway API.
//
// Non-scope:
//   - Does not manage virtual key storage.
//   - Does not own CLI-specific output formatting.
//
// Invariants/Assumptions:
//   - Gateway defaults stay aligned with internal/config defaults.
//   - Authorized operator probes use the LiteLLM master key when configured.
//
// Scope:
//   - File-local gateway probing and client implementation only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/config"
)

const (
	defaultPort           = config.DefaultLiteLLMPort
	defaultScheme         = "http"
	defaultConnectTimeout = config.DefaultHTTPTimeout
	defaultMaxTime        = 30 * time.Second
)

// Probe captures the outcome of a single gateway endpoint probe.
type Probe struct {
	Path       string        `json:"path"`
	URL        string        `json:"url"`
	HTTPStatus int           `json:"http_status,omitempty"`
	Reachable  bool          `json:"reachable"`
	Authorized bool          `json:"authorized"`
	Healthy    bool          `json:"healthy"`
	Latency    time.Duration `json:"latency,omitempty"`
	Error      string        `json:"error,omitempty"`
}

// Status captures typed gateway runtime health.
type Status struct {
	Scheme              string `json:"scheme"`
	BaseURL             string `json:"base_url"`
	TLSEnabled          bool   `json:"tls_enabled"`
	MasterKeyConfigured bool   `json:"master_key_configured"`
	Health              Probe  `json:"health"`
	Models              Probe  `json:"models"`
}

// Client provides gateway HTTP operations.
type Client struct {
	scheme         string
	host           string
	port           int
	baseURL        string
	httpClient     *http.Client
	masterKey      string
	connectTimeout time.Duration
	maxTime        time.Duration
}

// StatusReader narrows gateway access to typed runtime probes.
type StatusReader interface {
	Status(ctx context.Context) Status
}

// Option configures the Client.
type Option func(*Client)

// WithHost sets the gateway host.
func WithHost(host string) Option {
	return func(c *Client) {
		c.host = host
		c.baseURL = ""
	}
}

// WithPort sets the gateway port.
func WithPort(port int) Option {
	return func(c *Client) {
		c.port = port
		c.baseURL = ""
	}
}

// WithScheme sets the gateway URL scheme.
func WithScheme(scheme string) Option {
	return func(c *Client) {
		c.scheme = normalizeScheme(scheme)
		c.baseURL = ""
	}
}

// WithBaseURL sets the gateway base URL directly.
func WithBaseURL(raw string) Option {
	return func(c *Client) {
		if parsed, ok := parseBaseURL(raw); ok {
			c.scheme = parsed.Scheme
			c.host = parsed.Host
			c.port = parsed.Port
			c.baseURL = parsed.BaseURL
		}
	}
}

// WithMasterKey sets the master key for authentication.
func WithMasterKey(key string) Option {
	return func(c *Client) {
		c.masterKey = key
	}
}

// WithTimeout sets the request timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// NewClient creates a new gateway client.
func NewClient(opts ...Option) *Client {
	runtime := config.NewLoader().Gateway(false)

	c := &Client{
		scheme:         normalizeScheme(runtime.Scheme),
		host:           runtime.Host,
		port:           runtime.PortInt,
		baseURL:        strings.TrimSpace(runtime.BaseURL),
		httpClient:     &http.Client{Timeout: defaultMaxTime},
		masterKey:      runtime.MasterKey,
		connectTimeout: defaultConnectTimeout,
		maxTime:        defaultMaxTime,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// BaseURL returns the gateway base URL.
func (c *Client) BaseURL() string {
	if strings.TrimSpace(c.baseURL) != "" {
		return c.baseURL
	}
	return fmt.Sprintf("%s://%s:%d", normalizeScheme(c.scheme), c.host, c.port)
}

// HasMasterKey returns true when a non-empty master key is configured.
func (c *Client) HasMasterKey() bool {
	return strings.TrimSpace(c.masterKey) != ""
}

func normalizeScheme(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "https":
		return "https"
	default:
		return defaultScheme
	}
}

type parsedBaseURL struct {
	Scheme  string
	Host    string
	Port    int
	BaseURL string
}

func parseBaseURL(raw string) (parsedBaseURL, bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Hostname() == "" {
		return parsedBaseURL{}, false
	}
	port := defaultPortForScheme(parsed.Scheme)
	if parsed.Port() != "" {
		parsedPort, convErr := strconv.Atoi(parsed.Port())
		if convErr != nil || parsedPort <= 0 {
			return parsedBaseURL{}, false
		}
		port = parsedPort
	}
	return parsedBaseURL{
		Scheme:  parsed.Scheme,
		Host:    parsed.Hostname(),
		Port:    port,
		BaseURL: strings.TrimSuffix((&url.URL{Scheme: parsed.Scheme, Host: parsed.Host}).String(), "/"),
	}, true
}

func defaultPortForScheme(scheme string) int {
	if strings.EqualFold(strings.TrimSpace(scheme), "https") {
		return 443
	}
	return defaultPort
}

// Status executes the canonical ACP gateway probes.
func (c *Client) Status(ctx context.Context) Status {
	status := Status{
		Scheme:              normalizeScheme(c.scheme),
		BaseURL:             c.BaseURL(),
		TLSEnabled:          normalizeScheme(c.scheme) == "https",
		MasterKeyConfigured: c.HasMasterKey(),
	}

	status.Health = c.probe(ctx, "/health")
	status.Models = c.probe(ctx, "/v1/models")

	return status
}

// Health checks the gateway health endpoint.
// Returns true if the endpoint is authorized and healthy.
func (c *Client) Health(ctx context.Context) (bool, int, error) {
	probe := c.probe(ctx, "/health")
	if probe.Error != "" {
		return false, probe.HTTPStatus, fmt.Errorf("gateway probe %s failed: %s", probe.Path, probe.Error)
	}
	return probe.Healthy, probe.HTTPStatus, nil
}

// Models checks the gateway models endpoint.
// Returns true if the endpoint is authorized and accessible.
func (c *Client) Models(ctx context.Context) (bool, int, error) {
	probe := c.probe(ctx, "/v1/models")
	if probe.Error != "" {
		return false, probe.HTTPStatus, fmt.Errorf("gateway probe %s failed: %s", probe.Path, probe.Error)
	}
	return probe.Healthy, probe.HTTPStatus, nil
}

// GenerateKey generates a new virtual key.
func (c *Client) GenerateKey(ctx context.Context, req *GenerateKeyRequest) (*GenerateKeyResponse, error) {
	if c.masterKey == "" {
		return nil, fmt.Errorf("master key is required for key generation")
	}

	url := fmt.Sprintf("%s/key/generate", c.BaseURL())

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.masterKey))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("key generation failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	var result GenerateKeyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

func (c *Client) probe(ctx context.Context, path string) Probe {
	url := c.BaseURL() + path
	probe := Probe{
		Path: path,
		URL:  url,
	}

	start := time.Now()
	reqCtx, cancel := context.WithTimeout(ctx, c.connectTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		probe.Error = fmt.Sprintf("request creation failed: %v", err)
		return probe
	}
	if c.HasMasterKey() {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.masterKey))
	}

	resp, err := c.httpClient.Do(req)
	probe.Latency = time.Since(start).Round(time.Millisecond)
	if err != nil {
		probe.Error = err.Error()
		return probe
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	probe.Reachable = true
	probe.HTTPStatus = resp.StatusCode
	probe.Authorized = resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusForbidden
	probe.Healthy = resp.StatusCode == http.StatusOK

	return probe
}

// GenerateKeyRequest represents a key generation request.
type GenerateKeyRequest struct {
	KeyAlias            string   `json:"key_alias"`
	MaxBudget           float64  `json:"max_budget"`
	BudgetDuration      string   `json:"budget_duration"`
	Models              []string `json:"models,omitempty"`
	RPMLimit            int      `json:"rpm_limit,omitempty"`
	TPMLimit            int      `json:"tpm_limit,omitempty"`
	MaxParallelRequests int      `json:"max_parallel_requests,omitempty"`
}

// GenerateKeyResponse represents a key generation response.
type GenerateKeyResponse struct {
	Key            string  `json:"key"`
	KeyAlias       string  `json:"key_alias"`
	MaxBudget      float64 `json:"max_budget"`
	BudgetDuration string  `json:"budget_duration"`
}

// DeleteKeyRequest represents a key deletion request.
type DeleteKeyRequest struct {
	KeyAlias string `json:"key_alias"`
}

// ExtractKey extracts the key from a response.
func (r *GenerateKeyResponse) ExtractKey() string {
	return r.Key
}

// DeleteKey deletes a virtual key by alias.
func (c *Client) DeleteKey(ctx context.Context, alias string) error {
	if c.masterKey == "" {
		return fmt.Errorf("master key is required for key deletion")
	}
	if strings.TrimSpace(alias) == "" {
		return fmt.Errorf("key alias is required for key deletion")
	}

	url := fmt.Sprintf("%s/key/delete", c.BaseURL())
	payload, err := json.Marshal(DeleteKeyRequest{KeyAlias: strings.TrimSpace(alias)})
	if err != nil {
		return fmt.Errorf("failed to marshal delete request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.masterKey))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("delete request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read delete response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("key deletion failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	return nil
}
