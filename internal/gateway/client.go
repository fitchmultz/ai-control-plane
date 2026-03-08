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
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/config"
)

const (
	defaultHost           = config.DefaultGatewayHost
	defaultPort           = config.DefaultLiteLLMPort
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
	BaseURL             string `json:"base_url"`
	MasterKeyConfigured bool   `json:"master_key_configured"`
	Health              Probe  `json:"health"`
	Models              Probe  `json:"models"`
}

// Client provides gateway HTTP operations.
type Client struct {
	host           string
	port           int
	httpClient     *http.Client
	masterKey      string
	connectTimeout time.Duration
	maxTime        time.Duration
}

// Option configures the Client.
type Option func(*Client)

// WithHost sets the gateway host.
func WithHost(host string) Option {
	return func(c *Client) {
		c.host = host
	}
}

// WithPort sets the gateway port.
func WithPort(port int) Option {
	return func(c *Client) {
		c.port = port
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
	host := runtime.Host
	port := runtime.PortInt

	c := &Client{
		host:           host,
		port:           port,
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
	return fmt.Sprintf("http://%s:%d", c.host, c.port)
}

// HasMasterKey returns true when a non-empty master key is configured.
func (c *Client) HasMasterKey() bool {
	return strings.TrimSpace(c.masterKey) != ""
}

// Status executes the canonical ACP gateway probes.
func (c *Client) Status(ctx context.Context) Status {
	status := Status{
		BaseURL:             c.BaseURL(),
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

// ExtractKey extracts the key from a response.
func (r *GenerateKeyResponse) ExtractKey() string {
	return r.Key
}
