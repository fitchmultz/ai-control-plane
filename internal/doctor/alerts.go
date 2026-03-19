// Package doctor provides environment preflight diagnostics.
//
// Purpose:
//   - Deliver actionable doctor findings through pluggable alert adapters.
//
// Responsibilities:
//   - Build stable alert payloads from doctor reports.
//   - Deliver generic and Slack-compatible webhook notifications.
//
// Scope:
//   - Doctor alert serialization and delivery only.
//
// Usage:
//   - Used by `acpctl doctor --notify`.
//
// Invariants/Assumptions:
//   - Only actionable budget and detection findings are emitted.
package doctor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	sharedhealth "github.com/mitchfultz/ai-control-plane/internal/health"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

// AlertFinding captures one actionable doctor finding in a webhook payload.
type AlertFinding struct {
	CheckID  string                  `json:"check_id"`
	Name     string                  `json:"name"`
	Level    sharedhealth.Level      `json:"level"`
	Message  string                  `json:"message"`
	Severity Severity                `json:"severity"`
	Details  status.ComponentDetails `json:"details,omitempty"`
}

// AlertPayload is the canonical generic webhook payload for doctor findings.
type AlertPayload struct {
	Event     string             `json:"event"`
	Source    string             `json:"source"`
	Overall   sharedhealth.Level `json:"overall"`
	Timestamp string             `json:"timestamp"`
	Findings  []AlertFinding     `json:"findings"`
}

// AlertAdapter sends doctor findings to a specific notification destination.
type AlertAdapter interface {
	Name() string
	Send(ctx context.Context, payload AlertPayload) error
}

// GenericWebhookAdapter sends the canonical JSON payload to a generic webhook.
type GenericWebhookAdapter struct {
	URL    string
	Client *http.Client
}

// SlackWebhookAdapter sends a Slack-compatible attachment payload.
type SlackWebhookAdapter struct {
	URL    string
	Client *http.Client
}

func (a GenericWebhookAdapter) Name() string { return "generic" }

func (a GenericWebhookAdapter) Send(ctx context.Context, payload AlertPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return postDoctorJSON(ctx, a.client(), a.URL, body)
}

func (a SlackWebhookAdapter) Name() string { return "slack" }

func (a SlackWebhookAdapter) Send(ctx context.Context, payload AlertPayload) error {
	type field struct {
		Title string `json:"title"`
		Value string `json:"value"`
		Short bool   `json:"short"`
	}
	type attachment struct {
		Color  string  `json:"color"`
		Title  string  `json:"title"`
		Fields []field `json:"fields"`
		Footer string  `json:"footer"`
		TS     int64   `json:"ts"`
	}
	msg := struct {
		Text        string       `json:"text"`
		Attachments []attachment `json:"attachments"`
	}{
		Text: "AI Control Plane doctor findings",
	}

	color := "warning"
	if payload.Overall == sharedhealth.LevelUnhealthy {
		color = "danger"
	}
	fields := make([]field, 0, len(payload.Findings))
	for _, finding := range payload.Findings {
		fields = append(fields, field{
			Title: finding.Name,
			Value: finding.Message,
			Short: false,
		})
	}
	msg.Attachments = []attachment{{
		Color:  color,
		Title:  fmt.Sprintf("%d actionable finding(s)", len(payload.Findings)),
		Fields: fields,
		Footer: "AI Control Plane",
		TS:     time.Now().Unix(),
	}}

	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return postDoctorJSON(ctx, a.client(), a.URL, body)
}

func (a GenericWebhookAdapter) client() *http.Client {
	if a.Client != nil {
		return a.Client
	}
	return &http.Client{Timeout: 10 * time.Second}
}

func (a SlackWebhookAdapter) client() *http.Client {
	if a.Client != nil {
		return a.Client
	}
	return &http.Client{Timeout: 10 * time.Second}
}

// NotifyActionableFindings emits actionable doctor findings to configured adapters.
func NotifyActionableFindings(ctx context.Context, cfg config.AlertSettings, report Report) error {
	payload := BuildAlertPayload(report)
	if len(payload.Findings) == 0 {
		return nil
	}

	var adapters []AlertAdapter
	if cfg.GenericWebhookURL != "" {
		adapters = append(adapters, GenericWebhookAdapter{URL: cfg.GenericWebhookURL})
	}
	if cfg.SlackWebhookURL != "" {
		adapters = append(adapters, SlackWebhookAdapter{URL: cfg.SlackWebhookURL})
	}

	for _, adapter := range adapters {
		if err := adapter.Send(ctx, payload); err != nil {
			return fmt.Errorf("%s alert failed: %w", adapter.Name(), err)
		}
	}
	return nil
}

// BuildAlertPayload extracts actionable doctor findings into the canonical payload.
func BuildAlertPayload(report Report) AlertPayload {
	findings := make([]AlertFinding, 0)
	for _, result := range report.Results {
		if result.Level == sharedhealth.LevelHealthy {
			continue
		}
		if result.ID != "budget_findings" && result.ID != "detections_findings" {
			continue
		}
		findings = append(findings, AlertFinding{
			CheckID:  result.ID,
			Name:     result.Name,
			Level:    result.Level,
			Message:  result.Message,
			Severity: result.Severity,
			Details:  result.Details,
		})
	}

	return AlertPayload{
		Event:     "doctor_findings",
		Source:    "acpctl doctor",
		Overall:   report.Overall,
		Timestamp: report.Timestamp,
		Findings:  findings,
	}
}

func postDoctorJSON(ctx context.Context, client *http.Client, targetURL string, body []byte) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("unexpected HTTP %d", response.StatusCode)
	}
	return nil
}
