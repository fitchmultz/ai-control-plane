// Package chargeback defines the typed chargeback reporting domain.
//
// Purpose:
//   - Build and deliver chargeback notifications through a focused notifier
//     layer.
//
// Responsibilities:
//   - Construct generic and Slack payloads from typed domain inputs.
//   - Deliver JSON webhook notifications with bounded HTTP deadlines.
//
// Non-scope:
//   - Does not read environment variables.
//   - Does not write archive files.
//
// Invariants/Assumptions:
//   - Notification delivery is explicit and opt-in from workflow callers.
//   - Non-2xx responses are treated as failures.
//
// Scope:
//   - Chargeback notification payloads and delivery only.
//
// Usage:
//   - Used by the report workflow and notification-focused tests.
package chargeback

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/textutil"
)

type WebhookNotifier struct {
	Client *http.Client
}

func (n WebhookNotifier) Notify(ctx context.Context, config NotificationConfig, data ReportData) error {
	if textutil.IsBlank(config.GenericWebhookURL) && textutil.IsBlank(config.SlackWebhookURL) {
		return nil
	}
	client := n.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	if !textutil.IsBlank(config.GenericWebhookURL) {
		payload, err := BuildGenericWebhookPayload(GenericWebhookInput{
			Event:       defaultGenericNotificationEvent,
			ReportMonth: data.Input.ReportMonth,
			TotalSpend:  data.Input.TotalSpend,
			Variance:    data.Input.Variance,
			Anomalies:   data.Input.Anomalies,
			Timestamp:   data.Range.GeneratedAtUTC.Format(time.RFC3339),
		})
		if err != nil {
			return fmt.Errorf("build generic notification payload: %w", err)
		}
		if err := postJSON(ctx, client, config.GenericWebhookURL, payload); err != nil {
			return fmt.Errorf("send generic notification: %w", err)
		}
	}
	if !textutil.IsBlank(config.SlackWebhookURL) {
		color := defaultSlackColor
		if data.VarianceExceeded {
			color = "danger"
		}
		payload, err := BuildSlackWebhookPayload(SlackWebhookInput{
			ReportMonth: data.Input.ReportMonth,
			TotalSpend:  data.Input.TotalSpend,
			Variance:    data.Input.Variance,
			Color:       color,
			Epoch:       data.Range.GeneratedAtUTC.Unix(),
		})
		if err != nil {
			return fmt.Errorf("build slack notification payload: %w", err)
		}
		if err := postJSON(ctx, client, config.SlackWebhookURL, payload); err != nil {
			return fmt.Errorf("send slack notification: %w", err)
		}
	}
	return nil
}

func BuildGenericWebhookPayload(input GenericWebhookInput) ([]byte, error) {
	payload := struct {
		Event       string    `json:"event"`
		ReportMonth string    `json:"report_month"`
		TotalSpend  float64   `json:"total_spend"`
		Variance    string    `json:"variance_percent"`
		Anomalies   []Anomaly `json:"anomalies"`
		Timestamp   string    `json:"timestamp"`
	}{
		Event:       input.Event,
		ReportMonth: input.ReportMonth,
		TotalSpend:  input.TotalSpend,
		Variance:    input.Variance,
		Anomalies:   input.Anomalies,
		Timestamp:   input.Timestamp,
	}
	return json.Marshal(payload)
}

func BuildSlackWebhookPayload(input SlackWebhookInput) ([]byte, error) {
	payload := struct {
		Text        string `json:"text"`
		Attachments []struct {
			Color  string `json:"color"`
			Title  string `json:"title"`
			Fields []struct {
				Title string `json:"title"`
				Value string `json:"value"`
				Short bool   `json:"short"`
			} `json:"fields"`
			Footer string `json:"footer"`
			TS     int64  `json:"ts"`
		} `json:"attachments"`
	}{
		Text: "📊 Monthly Chargeback Report Generated",
	}

	attachment := struct {
		Color  string `json:"color"`
		Title  string `json:"title"`
		Fields []struct {
			Title string `json:"title"`
			Value string `json:"value"`
			Short bool   `json:"short"`
		} `json:"fields"`
		Footer string `json:"footer"`
		TS     int64  `json:"ts"`
	}{
		Color:  input.Color,
		Title:  "Report for " + input.ReportMonth,
		Footer: "AI Control Plane",
		TS:     input.Epoch,
	}
	attachment.Fields = []struct {
		Title string `json:"title"`
		Value string `json:"value"`
		Short bool   `json:"short"`
	}{
		{Title: "Total Spend", Value: fmt.Sprintf("$%.2f", input.TotalSpend), Short: true},
		{Title: "Variance", Value: input.Variance + "%", Short: true},
	}
	payload.Attachments = append(payload.Attachments, attachment)
	return json.Marshal(payload)
}

func postJSON(ctx context.Context, client *http.Client, targetURL string, body []byte) error {
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
