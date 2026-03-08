// Package gateway provides LiteLLM gateway client functionality.
//
// Purpose:
//
//	Provide HTTP client operations for the LiteLLM gateway including
//	health checks, key generation, and API interactions.
//
// Responsibilities:
//   - Gateway health checks
//   - HTTP request building with proper headers
//   - Key generation API calls
//   - Response parsing
//
// Non-scope:
//   - Does not manage virtual key storage
//   - Does not handle authentication beyond master key
//
// Invariants/Assumptions:
//   - Gateway follows LiteLLM API conventions
//   - Master key is provided for authenticated endpoints
//
// Scope:
//   - File-local implementation and interfaces only.
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

// Client provides gateway HTTP operations
type Client struct {
	host           string
	port           int
	httpClient     *http.Client
	masterKey      string
	connectTimeout time.Duration
	maxTime        time.Duration
}

// Option configures the Client
type Option func(*Client)

// WithHost sets the gateway host
func WithHost(host string) Option {
	return func(c *Client) {
		c.host = host
	}
}

// WithPort sets the gateway port
func WithPort(port int) Option {
	return func(c *Client) {
		c.port = port
	}
}

// WithMasterKey sets the master key for authentication
func WithMasterKey(key string) Option {
	return func(c *Client) {
		c.masterKey = key
	}
}

// WithTimeout sets the request timeout
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// NewClient creates a new gateway client
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

// BaseURL returns the gateway base URL
func (c *Client) BaseURL() string {
	return fmt.Sprintf("http://%s:%d", c.host, c.port)
}

// HasMasterKey returns true when a non-empty master key is configured.
func (c *Client) HasMasterKey() bool {
	return strings.TrimSpace(c.masterKey) != ""
}

// Health checks the gateway health endpoint
// Returns true if the endpoint is authorized and healthy.
func (c *Client) Health(ctx context.Context) (bool, int, error) {
	url := fmt.Sprintf("%s/health", c.BaseURL())
	code, err := c.doStatusRequest(ctx, url)
	if err != nil {
		return false, code, err
	}
	return code == http.StatusOK, code, nil
}

// Models checks the gateway models endpoint
// Returns true if the endpoint is authorized and accessible.
func (c *Client) Models(ctx context.Context) (bool, int, error) {
	url := fmt.Sprintf("%s/v1/models", c.BaseURL())
	code, err := c.doStatusRequest(ctx, url)
	if err != nil {
		return false, code, err
	}
	return code == http.StatusOK, code, nil
}

// GenerateKey generates a new virtual key
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

// doStatusRequest performs a GET request and returns the status code
func (c *Client) doStatusRequest(ctx context.Context, url string) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, c.connectTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	if c.HasMasterKey() {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.masterKey))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// Consume body to ensure connection can be reused
	_, _ = io.Copy(io.Discard, resp.Body)

	return resp.StatusCode, nil
}

// GenerateKeyRequest represents a key generation request
type GenerateKeyRequest struct {
	KeyAlias            string   `json:"key_alias"`
	MaxBudget           float64  `json:"max_budget"`
	BudgetDuration      string   `json:"budget_duration"`
	Models              []string `json:"models,omitempty"`
	RPMLimit            int      `json:"rpm_limit,omitempty"`
	TPMLimit            int      `json:"tpm_limit,omitempty"`
	MaxParallelRequests int      `json:"max_parallel_requests,omitempty"`
}

// GenerateKeyResponse represents a key generation response
type GenerateKeyResponse struct {
	Key            string  `json:"key"`
	KeyAlias       string  `json:"key_alias"`
	MaxBudget      float64 `json:"max_budget"`
	BudgetDuration string  `json:"budget_duration"`
}

// ExtractKey extracts the key from a response
func (r *GenerateKeyResponse) ExtractKey() string {
	return r.Key
}
