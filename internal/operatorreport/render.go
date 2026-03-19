// Package operatorreport renders canonical operator-facing runtime reports.
//
// Purpose:
//   - Provide one-command operator reporting on top of the shared runtime model.
//
// Responsibilities:
//   - Render typed runtime status reports as markdown-like text, JSON, or a
//     self-contained HTML dashboard.
//   - Archive generated reports using private local-only filesystem helpers.
//
// Scope:
//   - Operator-report rendering and archival only.
//
// Usage:
//   - Used by `acpctl ops report`, `make operator-report`, and
//   - `make operator-dashboard`.
//
// Invariants/Assumptions:
//   - Archived reports remain private local artifacts.
package operatorreport

import (
	"bytes"
	"fmt"
	"html/template"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/fsutil"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

// Format identifies the supported operator-report output formats.
type Format string

const (
	// FormatMarkdown renders the shared human-readable runtime report.
	FormatMarkdown Format = "markdown"
	// FormatJSON renders the shared JSON runtime report.
	FormatJSON Format = "json"
	// FormatHTML renders a self-contained operator dashboard snapshot.
	FormatHTML Format = "html"
)

// Request captures render-time report preferences.
type Request struct {
	Format Format
	Wide   bool
}

// Render formats the shared runtime report for operator-facing consumption.
func Render(report status.StatusReport, req Request) ([]byte, string, error) {
	switch req.Format {
	case FormatJSON:
		var buf bytes.Buffer
		if err := report.WriteJSON(&buf); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "json", nil
	case FormatHTML:
		payload, err := renderHTML(report, req.Wide)
		if err != nil {
			return nil, "", err
		}
		return payload, "html", nil
	default:
		var buf bytes.Buffer
		if err := report.WriteHuman(&buf, req.Wide); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "md", nil
	}
}

// Archive persists the rendered operator report under a private local archive root.
func Archive(repoRoot string, archiveDir string, stamp string, payload []byte, ext string) (string, error) {
	if archiveDir == "" {
		return "", nil
	}
	targetDir := filepath.Join(repoRoot, archiveDir, stamp)
	if err := fsutil.EnsurePrivateDir(targetDir); err != nil {
		return "", fmt.Errorf("create operator report directory: %w", err)
	}
	path := filepath.Join(targetDir, fmt.Sprintf("operator-report-%s.%s", stamp, ext))
	if err := fsutil.AtomicWritePrivateFile(path, payload); err != nil {
		return "", fmt.Errorf("write operator report: %w", err)
	}
	return path, nil
}

type dashboardView struct {
	OverallLabel string
	OverallClass string
	Timestamp    string
	Duration     string
	Components   []dashboardComponent
}

type dashboardComponent struct {
	Name        string
	Title       string
	LevelLabel  string
	LevelClass  string
	Message     string
	Suggestions []string
	Details     []string
}

func renderHTML(report status.StatusReport, wide bool) ([]byte, error) {
	view := dashboardView{
		OverallLabel: strings.ToUpper(string(report.Overall)),
		OverallClass: levelClass(report.Overall),
		Timestamp:    report.Timestamp,
		Duration:     report.Duration,
		Components:   orderedComponents(report, wide),
	}
	const dashboardTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>AI Control Plane Operator Dashboard</title>
  <style>
    :root {
      color-scheme: light dark;
      --bg: #0b1020;
      --panel: #121933;
      --panel-border: #27304d;
      --text: #e8edf7;
      --muted: #aab6d3;
      --ok: #1f9d55;
      --warn: #d29b19;
      --fail: #cf3f3f;
      --unknown: #6b7280;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      background: linear-gradient(180deg, #09101f 0%, #0f172a 100%);
      color: var(--text);
      padding: 32px;
    }
    h1, h2, h3, p { margin: 0; }
    .shell { max-width: 1280px; margin: 0 auto; }
    .hero {
      display: flex;
      justify-content: space-between;
      gap: 24px;
      align-items: end;
      margin-bottom: 24px;
      flex-wrap: wrap;
    }
    .meta { color: var(--muted); margin-top: 10px; display: flex; gap: 18px; flex-wrap: wrap; }
    .badge {
      border-radius: 999px;
      padding: 10px 16px;
      font-weight: 700;
      letter-spacing: 0.06em;
      border: 1px solid currentColor;
    }
    .ok { color: #6ee7b7; }
    .warn { color: #fde68a; }
    .fail { color: #fca5a5; }
    .unknown { color: #cbd5e1; }
    .grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
      gap: 16px;
    }
    .card {
      background: rgba(18, 25, 51, 0.94);
      border: 1px solid var(--panel-border);
      border-radius: 18px;
      padding: 18px;
      box-shadow: 0 20px 50px rgba(0, 0, 0, 0.22);
    }
    .card-head {
      display: flex;
      justify-content: space-between;
      gap: 12px;
      align-items: baseline;
      margin-bottom: 12px;
    }
    .title { font-size: 1.05rem; font-weight: 700; }
    .level { font-size: 0.78rem; font-weight: 700; letter-spacing: 0.05em; }
    .message { font-size: 0.96rem; margin-bottom: 12px; }
    .details, .suggestions { margin: 0; padding-left: 18px; color: var(--muted); }
    .details li, .suggestions li { margin: 6px 0; }
    .section-label {
      display: inline-block;
      margin-top: 12px;
      margin-bottom: 6px;
      color: var(--muted);
      font-size: 0.78rem;
      text-transform: uppercase;
      letter-spacing: 0.08em;
    }
  </style>
</head>
<body>
  <div class="shell">
    <div class="hero">
      <div>
        <h1>AI Control Plane Operator Dashboard</h1>
        <div class="meta">
          <span>Generated: {{ .Timestamp }}</span>
          <span>Collection duration: {{ .Duration }}</span>
          <span>Source: acpctl ops report --format html</span>
        </div>
      </div>
      <div class="badge {{ .OverallClass }}">Overall {{ .OverallLabel }}</div>
    </div>
    <div class="grid">
      {{- range .Components }}
      <section class="card">
        <div class="card-head">
          <div class="title">{{ .Title }}</div>
          <div class="level {{ .LevelClass }}">{{ .LevelLabel }}</div>
        </div>
        <p class="message">{{ .Message }}</p>
        {{- if .Details }}
        <div class="section-label">Details</div>
        <ul class="details">
          {{- range .Details }}
          <li>{{ . }}</li>
          {{- end }}
        </ul>
        {{- end }}
        {{- if .Suggestions }}
        <div class="section-label">Next actions</div>
        <ul class="suggestions">
          {{- range .Suggestions }}
          <li>{{ . }}</li>
          {{- end }}
        </ul>
        {{- end }}
      </section>
      {{- end }}
    </div>
  </div>
</body>
</html>
`
	tmpl, err := template.New("dashboard").Parse(dashboardTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse operator dashboard template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, view); err != nil {
		return nil, fmt.Errorf("render operator dashboard template: %w", err)
	}
	return buf.Bytes(), nil
}

func orderedComponents(report status.StatusReport, wide bool) []dashboardComponent {
	ordered := make([]dashboardComponent, 0, len(report.Components))
	seen := make(map[string]struct{}, len(report.Components))
	appendComponent := func(name string, component status.ComponentStatus) {
		item := dashboardComponent{
			Name:        name,
			Title:       strings.ToUpper(name[:1]) + name[1:],
			LevelLabel:  strings.ToUpper(string(component.Level)),
			LevelClass:  levelClass(component.Level),
			Message:     component.Message,
			Suggestions: append([]string(nil), component.Suggestions...),
		}
		if wide {
			item.Details = append(item.Details, component.Details.Lines()...)
		}
		ordered = append(ordered, item)
		seen[name] = struct{}{}
	}
	for _, name := range status.DefaultComponentOrder {
		component, ok := report.Components[name]
		if ok {
			appendComponent(name, component)
		}
	}
	var extras []string
	for name := range report.Components {
		if _, ok := seen[name]; ok {
			continue
		}
		extras = append(extras, name)
	}
	sort.Strings(extras)
	for _, name := range extras {
		appendComponent(name, report.Components[name])
	}
	return ordered
}

func levelClass(level status.HealthLevel) string {
	switch level {
	case status.HealthLevelHealthy:
		return "ok"
	case status.HealthLevelWarning:
		return "warn"
	case status.HealthLevelUnhealthy:
		return "fail"
	default:
		return "unknown"
	}
}
