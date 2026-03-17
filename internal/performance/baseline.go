// baseline.go - Lightweight performance baseline harness for the AI Control Plane.
//
// Purpose:
//
//	Provide a deterministic, local performance harness for measuring gateway
//	latency and throughput in the validated offline reference environment.
//
// Responsibilities:
//   - Execute repeated chat-completions requests against the gateway
//   - Measure latency distribution and aggregate throughput
//   - Return machine-readable and human-readable baseline metrics
//
// Scope:
//   - Covers local baseline measurements only
//   - Does not claim customer-grade capacity proof
//
// Usage:
//   - Called from `acpctl benchmark baseline`
//   - Used by `make performance-baseline`
//
// Invariants/Assumptions:
//   - Requests target an OpenAI-compatible `/v1/chat/completions` endpoint
//   - The benchmark should use low-cost or offline-safe models by default
package performance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/config"
)

// BaselineOptions configures a baseline run.
type BaselineOptions struct {
	GatewayURL     string
	MasterKey      string
	Model          string
	Profile        string
	Prompt         string
	WarmupRequests int
	Requests       int
	Concurrency    int
	MaxTokens      int
	HTTPTimeout    time.Duration
	RequestFunc    RequestFunc
	Now            func() time.Time
}

// RequestFunc issues one inference request and returns its duration.
type RequestFunc func(ctx context.Context, opts BaselineOptions) (time.Duration, error)

// Sample captures one request result.
type Sample struct {
	Index   int           `json:"index"`
	Latency time.Duration `json:"latency_ns"`
	Status  string        `json:"status"`
	Error   string        `json:"error,omitempty"`
}

// Summary captures aggregate baseline metrics.
type Summary struct {
	GatewayURL       string    `json:"gateway_url"`
	Model            string    `json:"model"`
	Profile          string    `json:"profile,omitempty"`
	WarmupRequests   int       `json:"warmup_requests"`
	Requests         int       `json:"requests"`
	Concurrency      int       `json:"concurrency"`
	Successes        int       `json:"successes"`
	Failures         int       `json:"failures"`
	StartedAtUTC     time.Time `json:"started_at_utc"`
	FinishedAtUTC    time.Time `json:"finished_at_utc"`
	Duration         string    `json:"duration"`
	RequestsPerSec   float64   `json:"requests_per_sec"`
	AverageLatencyMS float64   `json:"average_latency_ms"`
	P50LatencyMS     float64   `json:"p50_latency_ms"`
	P95LatencyMS     float64   `json:"p95_latency_ms"`
	MaxLatencyMS     float64   `json:"max_latency_ms"`
	MinLatencyMS     float64   `json:"min_latency_ms"`
	Samples          []Sample  `json:"samples"`
}

// RunBaseline executes the configured performance baseline.
func RunBaseline(ctx context.Context, opts BaselineOptions) (*Summary, error) {
	normalized, err := normalizeOptions(opts)
	if err != nil {
		return nil, err
	}
	requestFn := normalized.RequestFunc
	if requestFn == nil {
		requestFn = defaultRequest
	}
	nowFn := normalized.Now
	if nowFn == nil {
		nowFn = func() time.Time { return time.Now().UTC() }
	}
	for warmupIndex := 0; warmupIndex < normalized.WarmupRequests; warmupIndex++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		if _, warmupErr := requestFn(ctx, normalized); warmupErr != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, fmt.Errorf("warmup request %d failed: %w", warmupIndex+1, warmupErr)
		}
	}

	started := nowFn()

	jobs := make(chan int)
	results := make(chan Sample, normalized.Requests)
	var wg sync.WaitGroup
	for worker := 0; worker < normalized.Concurrency; worker++ {
		wg.Go(func() {
			for {
				select {
				case <-ctx.Done():
					return
				case index, ok := <-jobs:
					if !ok {
						return
					}
					latency, requestErr := requestFn(ctx, normalized)
					sample := Sample{Index: index, Latency: latency, Status: "ok"}
					if requestErr != nil {
						if ctx.Err() != nil {
							return
						}
						sample.Status = "error"
						sample.Error = requestErr.Error()
					}
					select {
					case results <- sample:
					case <-ctx.Done():
						return
					}
				}
			}
		})
	}
enqueue:
	for index := 0; index < normalized.Requests; index++ {
		select {
		case jobs <- index + 1:
		case <-ctx.Done():
			break enqueue
		}
	}
	close(jobs)
	wg.Wait()
	close(results)

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	samples := make([]Sample, 0, normalized.Requests)
	for sample := range results {
		samples = append(samples, sample)
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i].Index < samples[j].Index })
	finished := nowFn()

	summary := &Summary{
		GatewayURL:     normalized.GatewayURL,
		Model:          normalized.Model,
		Profile:        normalized.Profile,
		WarmupRequests: normalized.WarmupRequests,
		Requests:       normalized.Requests,
		Concurrency:    normalized.Concurrency,
		StartedAtUTC:   started,
		FinishedAtUTC:  finished,
		Duration:       finished.Sub(started).Round(time.Millisecond).String(),
		Samples:        samples,
	}
	populateMetrics(summary)
	return summary, nil
}

func normalizeOptions(opts BaselineOptions) (BaselineOptions, error) {
	if strings.TrimSpace(opts.GatewayURL) == "" {
		opts.GatewayURL = config.ResolveGatewaySettings(config.GatewayResolveInput{}).BaseURL
	}
	if strings.TrimSpace(opts.MasterKey) == "" {
		return BaselineOptions{}, fmt.Errorf("master key is required")
	}
	if strings.TrimSpace(opts.Model) == "" {
		opts.Model = "mock-gpt"
	}
	if strings.TrimSpace(opts.Prompt) == "" {
		opts.Prompt = "Provide a short response for performance baseline verification."
	}
	if opts.WarmupRequests < 0 {
		return BaselineOptions{}, fmt.Errorf("warmup requests must be >= 0")
	}
	if opts.Requests <= 0 {
		return BaselineOptions{}, fmt.Errorf("requests must be > 0")
	}
	if opts.Concurrency <= 0 {
		return BaselineOptions{}, fmt.Errorf("concurrency must be > 0")
	}
	if opts.MaxTokens <= 0 {
		opts.MaxTokens = 32
	}
	if opts.HTTPTimeout <= 0 {
		opts.HTTPTimeout = 30 * time.Second
	}
	return opts, nil
}

func defaultRequest(ctx context.Context, opts BaselineOptions) (time.Duration, error) {
	httpClient := &http.Client{Timeout: opts.HTTPTimeout}
	payload := map[string]any{
		"model":      opts.Model,
		"messages":   []map[string]string{{"role": "user", "content": opts.Prompt}},
		"max_tokens": opts.MaxTokens,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("marshal request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(opts.GatewayURL, "/")+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+opts.MasterKey)
	request.Header.Set("Content-Type", "application/json")

	started := time.Now()
	response, err := httpClient.Do(request)
	latency := time.Since(started)
	if err != nil {
		return latency, fmt.Errorf("request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return latency, fmt.Errorf("unexpected status %d", response.StatusCode)
	}
	return latency, nil
}

func populateMetrics(summary *Summary) {
	latencies := make([]float64, 0, len(summary.Samples))
	for _, sample := range summary.Samples {
		if sample.Status == "ok" {
			summary.Successes++
			latencies = append(latencies, float64(sample.Latency)/float64(time.Millisecond))
		} else {
			summary.Failures++
		}
	}
	if len(latencies) == 0 {
		return
	}
	sort.Float64s(latencies)
	summary.MinLatencyMS = latencies[0]
	summary.MaxLatencyMS = latencies[len(latencies)-1]
	summary.P50LatencyMS = percentile(latencies, 50)
	summary.P95LatencyMS = percentile(latencies, 95)
	var total float64
	for _, latency := range latencies {
		total += latency
	}
	summary.AverageLatencyMS = total / float64(len(latencies))
	elapsedSeconds := summary.FinishedAtUTC.Sub(summary.StartedAtUTC).Seconds()
	if elapsedSeconds > 0 {
		summary.RequestsPerSec = float64(summary.Successes) / elapsedSeconds
	}
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}
	rank := (p / 100) * float64(len(sorted)-1)
	lower := int(math.Floor(rank))
	upper := int(math.Ceil(rank))
	if lower == upper {
		return sorted[lower]
	}
	weight := rank - float64(lower)
	return sorted[lower] + (sorted[upper]-sorted[lower])*weight
}
